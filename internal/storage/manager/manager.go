package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/crypto"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
	"github.com/pichub/pichub/internal/storage"
)

// StrategyRepo 是策略管理器依赖的仓储接口（定义在消费方）。
type StrategyRepo interface {
	GetByID(ctx context.Context, id uint64) (*model.StorageConfig, error)
	GetDefault(ctx context.Context) (*model.StorageConfig, error)
	ListSystem(ctx context.Context) ([]*model.StorageConfig, error)
	Create(ctx context.Context, s *model.StorageConfig) error
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
}

// StrategyConfig 是存储策略的非敏感配置。
type StrategyConfig struct {
	RootPath     string `json:"root_path,omitempty"`
	PathTemplate string `json:"path_template,omitempty"`
	Endpoint     string `json:"endpoint,omitempty"`
	Region       string `json:"region,omitempty"`
	Bucket       string `json:"bucket,omitempty"`
	PathStyle    bool   `json:"path_style,omitempty"`
	UseSSL       *bool  `json:"use_ssl,omitempty"`
	PublicURL    string `json:"public_url,omitempty"`
	URL          string `json:"url,omitempty"`
}

// StrategySecret 是存储策略的敏感配置（加密存储）。
type StrategySecret struct {
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
	Password  string `json:"password,omitempty"`
}

// Manager 负责存储策略解析与驱动实例缓存。
type Manager struct {
	cfg    config.StorageConfig
	repo   StrategyRepo
	cipher *crypto.Cipher
	log    *logger.Logger

	mu       sync.RWMutex
	cache    map[uint64]storage.Storage
	fallback storage.Storage
}

// NewManager 创建策略管理器并初始化 YAML 兜底驱动。
func NewManager(cfg config.StorageConfig, repo StrategyRepo, cipher *crypto.Cipher, log *logger.Logger) (*Manager, error) {
	m := &Manager{
		cfg:    cfg,
		repo:   repo,
		cipher: cipher,
		log:    log,
		cache:  make(map[uint64]storage.Storage),
	}
	fallback, err := NewDriver(storage.DriverConfig{
		Driver: cfg.DefaultDriver,
		Local:  cfg.Local,
		S3:     cfg.S3,
		OSS:    cfg.OSS,
		COS:    cfg.COS,
		WebDAV: cfg.WebDAV,
	})
	if err != nil {
		return nil, fmt.Errorf("init fallback storage: %w", err)
	}
	m.fallback = fallback
	return m, nil
}

// Fallback 返回 YAML 兜底驱动。
func (m *Manager) Fallback() storage.Storage { return m.fallback }

// FallbackDriver 返回兜底驱动名。
func (m *Manager) FallbackDriver() string { return m.cfg.DefaultDriver }

// AllowUserStrategy 返回是否允许用户自定义存储策略。
func (m *Manager) AllowUserStrategy() bool { return m.cfg.AllowUserStrategy }

// EnsureDefault 确保存在系统默认策略（本地存储）。
func (m *Manager) EnsureDefault(ctx context.Context) error {
	list, err := m.repo.ListSystem(ctx)
	if err != nil {
		return err
	}
	hasDefault := false
	hasLocal := false
	for _, s := range list {
		if s.IsDefault && s.Status == model.StatusEnabled {
			hasDefault = true
		}
		if s.Driver == storage.DriverLocal {
			hasLocal = true
		}
	}
	if hasLocal && hasDefault {
		return m.cleanupLocalSecret(ctx, list)
	}
	if !hasLocal {
		cfgJSON, _ := json.Marshal(StrategyConfig{
			RootPath:     m.cfg.Local.RootPath,
			PathTemplate: m.cfg.Local.PathTemplate,
		})
		name := m.cfg.DefaultStrategyName
		if name == "" {
			name = "本地存储"
		}
		sc := &model.StorageConfig{
			Name:       name,
			Driver:     storage.DriverLocal,
			Config:     cfgJSON,
			Secret:     []byte{},
			PathPrefix: m.cfg.Local.PathPrefix,
			IsDefault:  true,
			Status:     model.StatusEnabled,
		}
		if err := m.repo.Create(ctx, sc); err != nil {
			return fmt.Errorf("create default local storage: %w", err)
		}
		if m.log != nil {
			m.log.Info("created default local storage strategy", logger.Uint64("id", sc.ID))
		}
		return nil
	}
	// 有本地策略但没有默认标记：把第一个本地策略设为默认。
	for _, s := range list {
		if s.Driver == storage.DriverLocal {
			if err := m.repo.UpdateFields(ctx, s.ID, map[string]any{"is_default": true}); err != nil {
				return err
			}
			break
		}
	}
	return m.cleanupLocalSecret(ctx, list)
}

// cleanupLocalSecret 清理历史版本写入本地策略的明文占位密钥。
// 本地驱动不读取敏感配置，这些数据只会让管理接口误报「已配置密钥」。
func (m *Manager) cleanupLocalSecret(ctx context.Context, list []*model.StorageConfig) error {
	if m.cipher == nil {
		return nil
	}
	for _, s := range list {
		if s.Driver != storage.DriverLocal || len(s.Secret) == 0 {
			continue
		}
		if _, err := m.cipher.Decrypt(s.Secret); err == nil {
			continue
		}
		if err := m.repo.UpdateFields(ctx, s.ID, map[string]any{"secret": []byte{}}); err != nil {
			return err
		}
		if m.log != nil {
			m.log.Warn("cleared invalid local storage secret", logger.Uint64("id", s.ID))
		}
	}
	return nil
}

// Resolve 按优先级解析用户应使用的存储：请求指定 > 用户默认 > 系统默认 > YAML 兜底。
func (m *Manager) Resolve(ctx context.Context, user *model.User, requestedID uint64) (storage.Storage, *model.StorageConfig, error) {
	var sc *model.StorageConfig
	var err error
	switch {
	case requestedID > 0:
		sc, err = m.repo.GetByID(ctx, requestedID)
	case user != nil && user.StorageConfigID != nil:
		sc, err = m.repo.GetByID(ctx, *user.StorageConfigID)
	default:
		sc, err = m.repo.GetDefault(ctx)
	}
	if err != nil {
		return nil, nil, err
	}
	if sc != nil && sc.Status == model.StatusEnabled {
		st, berr := m.Build(ctx, sc)
		if berr != nil {
			return nil, nil, berr
		}
		return st, sc, nil
	}
	// 兜底策略。
	if user != nil && user.StorageConfigID != nil && requestedID == 0 {
		if sc2, e := m.repo.GetByID(ctx, *user.StorageConfigID); e == nil && sc2 != nil && sc2.Status == model.StatusEnabled {
			if st, berr := m.Build(ctx, sc2); berr == nil {
				return st, sc2, nil
			}
		}
	}
	return m.fallback, nil, nil
}

// Build 根据策略配置构造（并缓存）驱动。
func (m *Manager) Build(ctx context.Context, sc *model.StorageConfig) (storage.Storage, error) {
	m.mu.RLock()
	if st, ok := m.cache[sc.ID]; ok {
		m.mu.RUnlock()
		return st, nil
	}
	m.mu.RUnlock()

	st, err := m.build(sc)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.cache[sc.ID] = st
	m.mu.Unlock()
	return st, nil
}

func (m *Manager) build(sc *model.StorageConfig) (storage.Storage, error) {
	var sconf StrategyConfig
	if len(sc.Config) > 0 {
		if err := json.Unmarshal(sc.Config, &sconf); err != nil {
			return nil, fmt.Errorf("parse storage config: %w", err)
		}
	}
	var sec StrategySecret
	// 本地驱动不读取敏感配置；历史版本曾写入明文占位，解密必然失败，这里直接跳过。
	if sc.Driver != storage.DriverLocal && len(sc.Secret) > 0 && m.cipher != nil {
		plain, err := m.cipher.Decrypt(sc.Secret)
		if err != nil {
			return nil, fmt.Errorf("decrypt storage secret: %w", err)
		}
		if err := json.Unmarshal(plain, &sec); err != nil {
			return nil, fmt.Errorf("parse storage secret: %w", err)
		}
	}

	useSSL := true
	if sconf.UseSSL != nil {
		useSSL = *sconf.UseSSL
	}

	dc := storage.DriverConfig{
		Driver: sc.Driver,
		Domain: sc.Domain,
		Local:  m.cfg.Local,
		S3:     m.cfg.S3,
		OSS:    m.cfg.OSS,
		COS:    m.cfg.COS,
		WebDAV: m.cfg.WebDAV,
	}
	apply := func(dst *config.ObjectStorageConfig) {
		if sconf.Endpoint != "" {
			dst.Endpoint = sconf.Endpoint
		}
		if sconf.Region != "" {
			dst.Region = sconf.Region
		}
		if sconf.Bucket != "" {
			dst.Bucket = sconf.Bucket
		}
		dst.AccessKey = sec.AccessKey
		dst.SecretKey = sec.SecretKey
		if sconf.PathStyle {
			dst.PathStyle = true
		}
		dst.UseSSL = useSSL
		if sconf.PublicURL != "" {
			dst.PublicURL = sconf.PublicURL
		}
	}
	if sconf.RootPath != "" {
		dc.Local.RootPath = sconf.RootPath
	}
	if sconf.PathTemplate != "" {
		dc.Local.PathTemplate = sconf.PathTemplate
	}
	apply(&dc.S3)
	apply(&dc.OSS)
	apply(&dc.COS)
	if sconf.URL != "" {
		dc.WebDAV.URL = sconf.URL
	}
	if sconf.PublicURL != "" {
		dc.WebDAV.PublicURL = sconf.PublicURL
	}
	dc.WebDAV.Username = sec.AccessKey
	dc.WebDAV.Password = sec.SecretKey
	if sec.Password != "" {
		dc.WebDAV.Password = sec.Password
	}
	return NewDriver(dc)
}

// Invalidate 清除缓存（策略更新后调用）。
func (m *Manager) Invalidate(id uint64) {
	m.mu.Lock()
	delete(m.cache, id)
	m.mu.Unlock()
}

// Test 测试策略连通性。
func (m *Manager) Test(ctx context.Context, sc *model.StorageConfig) error {
	st, err := m.build(sc)
	if err != nil {
		return err
	}
	return st.Test(ctx)
}

// PublicURL 计算策略下对象的公开地址。
func (m *Manager) PublicURL(sc *model.StorageConfig, st storage.Storage, key string) string {
	if sc != nil && sc.Domain != "" {
		return storage.JoinURL(sc.Domain, key)
	}
	return st.PublicURL(key)
}
