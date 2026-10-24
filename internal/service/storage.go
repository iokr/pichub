package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/crypto"
	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
	"github.com/pichub/pichub/internal/storage"
	"github.com/pichub/pichub/internal/storage/manager"
)

// StorageService 处理存储策略管理。
type StorageService struct {
	cfg      *config.Config
	log      *logger.Logger
	repos    Repos
	storages StorageManager
	cipher   *crypto.Cipher
}

// NewStorageService 创建存储策略服务。
func NewStorageService(d Deps) *StorageService {
	return &StorageService{cfg: d.Config, log: d.Log, repos: d.Repos, storages: d.Storages, cipher: d.Cipher}
}

// List 返回可管理的存储策略。
func (s *StorageService) List(ctx context.Context, user *model.User, admin bool) ([]*dto.StorageResp, error) {
	var (
		list []*model.StorageConfig
		err  error
	)
	if admin {
		list, err = s.repos.Storage.ListSystem(ctx)
	} else {
		if !s.cfg.Storage.AllowUserStrategy {
			return []*dto.StorageResp{}, nil
		}
		list, err = s.repos.Storage.ListByUser(ctx, user.ID)
	}
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return dto.NewStorageRespList(list), nil
}

// Create 创建存储策略。
func (s *StorageService) Create(ctx context.Context, user *model.User, req *dto.StorageCreateReq, admin bool) (*dto.StorageResp, error) {
	if !admin && !s.cfg.Storage.AllowUserStrategy {
		return nil, errs.ErrForbidden.WithMessage("自定义存储未开放")
	}
	driver := strings.ToLower(strings.TrimSpace(req.Driver))
	if !validDriver(driver) {
		return nil, errs.ErrInvalidParam.WithDetail("不支持的存储驱动")
	}
	if err := validateStorageFields(driver, req.Endpoint, req.Bucket, req.URL, req.AccessKey, req.SecretKey, req.Username, req.Password); err != nil {
		return nil, err
	}
	conf := manager.StrategyConfig{
		RootPath:     req.RootPath,
		PathTemplate: req.PathTemplate,
		Endpoint:     req.Endpoint,
		Region:       req.Region,
		Bucket:       req.Bucket,
		PathStyle:    req.PathStyle,
		UseSSL:       req.UseSSL,
		PublicURL:    req.PublicURL,
		URL:          req.URL,
	}
	cfgJSON, err := json.Marshal(conf)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	sec := manager.StrategySecret{AccessKey: req.AccessKey, SecretKey: req.SecretKey, Password: req.Password}
	if req.Username != "" {
		sec.AccessKey = req.Username
	}
	var secretEnc []byte
	if sec.AccessKey != "" || sec.SecretKey != "" || sec.Password != "" {
		plain, err := json.Marshal(sec)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		secretEnc, err = s.cipher.Encrypt(plain)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
	}
	status := int8(model.StatusEnabled)
	if req.Status != nil {
		status = *req.Status
	}
	sc := &model.StorageConfig{
		Name:       strings.TrimSpace(req.Name),
		Driver:     driver,
		Config:     cfgJSON,
		Secret:     secretEnc,
		Domain:     strings.TrimSpace(req.Domain),
		PathPrefix: strings.TrimSpace(req.PathPrefix),
		IsDefault:  req.IsDefault,
		Status:     status,
	}
	if !admin {
		sc.UserID = &user.ID
	}
	if sc.Name == "" {
		return nil, errs.ErrInvalidParam.WithDetail("策略名称不能为空")
	}
	if sc.IsDefault && admin {
		if err := s.repos.Storage.ClearDefault(ctx); err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
	}
	if err := s.repos.Storage.Create(ctx, sc); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if !admin && req.IsDefault {
		if err := s.setUserDefault(ctx, user.ID, sc.ID); err != nil {
			return nil, err
		}
	}
	s.storages.Invalidate(sc.ID)
	return dto.NewStorageResp(sc), nil
}

// Update 更新存储策略。
func (s *StorageService) Update(ctx context.Context, user *model.User, id uint64, req *dto.StorageUpdateReq, admin bool) (*dto.StorageResp, error) {
	sc, err := s.accessible(ctx, user, id, admin)
	if err != nil {
		return nil, err
	}
	conf := manager.StrategyConfig{}
	if len(sc.Config) > 0 {
		_ = json.Unmarshal(sc.Config, &conf)
	}
	if req.RootPath != nil {
		conf.RootPath = *req.RootPath
	}
	if req.PathTemplate != nil {
		conf.PathTemplate = *req.PathTemplate
	}
	if req.Endpoint != nil {
		conf.Endpoint = *req.Endpoint
	}
	if req.Region != nil {
		conf.Region = *req.Region
	}
	if req.Bucket != nil {
		conf.Bucket = *req.Bucket
	}
	if req.PathStyle != nil {
		conf.PathStyle = *req.PathStyle
	}
	if req.UseSSL != nil {
		conf.UseSSL = req.UseSSL
	}
	if req.PublicURL != nil {
		conf.PublicURL = *req.PublicURL
	}
	if req.URL != nil {
		conf.URL = *req.URL
	}

	secretChanged := false
	sec := manager.StrategySecret{}
	if len(sc.Secret) > 0 {
		if plain, derr := s.cipher.Decrypt(sc.Secret); derr == nil {
			_ = json.Unmarshal(plain, &sec)
		}
	}
	if req.AccessKey != nil {
		sec.AccessKey = *req.AccessKey
		secretChanged = true
	}
	if req.SecretKey != nil {
		sec.SecretKey = *req.SecretKey
		secretChanged = true
	}
	if req.Username != nil {
		sec.AccessKey = *req.Username
		secretChanged = true
	}
	if req.Password != nil {
		sec.Password = *req.Password
		secretChanged = true
	}

	fields := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errs.ErrInvalidParam.WithDetail("策略名称不能为空")
		}
		fields["name"] = name
	}
	if req.Domain != nil {
		fields["domain"] = strings.TrimSpace(*req.Domain)
	}
	if req.PathPrefix != nil {
		fields["path_prefix"] = strings.TrimSpace(*req.PathPrefix)
	}
	if req.Status != nil {
		if *req.Status != model.StatusEnabled && *req.Status != model.StatusDisabled {
			return nil, errs.ErrInvalidParam.WithDetail("状态值无效")
		}
		fields["status"] = *req.Status
	}
	if req.IsDefault != nil {
		if *req.IsDefault && sc.UserID == nil {
			if err := s.repos.Storage.ClearDefault(ctx); err != nil {
				return nil, errs.ErrInternal.Wrap(err)
			}
		}
		fields["is_default"] = *req.IsDefault
	}
	cfgJSON, err := json.Marshal(conf)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	fields["config"] = cfgJSON
	if secretChanged {
		plain, err := json.Marshal(sec)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		enc, err := s.cipher.Encrypt(plain)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		fields["secret"] = enc
	}
	if err := s.repos.Storage.UpdateFields(ctx, sc.ID, fields); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	s.storages.Invalidate(sc.ID)
	if req.IsDefault != nil && *req.IsDefault && sc.UserID != nil {
		if err := s.setUserDefault(ctx, *sc.UserID, sc.ID); err != nil {
			return nil, err
		}
	}
	return s.getResp(ctx, sc.ID)
}

// Delete 删除存储策略。
func (s *StorageService) Delete(ctx context.Context, user *model.User, id uint64, admin bool) error {
	sc, err := s.accessible(ctx, user, id, admin)
	if err != nil {
		return err
	}
	if sc.IsDefault {
		return errs.ErrInvalidParam.WithDetail("默认策略不能删除，请先切换默认策略")
	}
	if err := s.repos.Storage.Delete(ctx, sc.ID); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	s.storages.Invalidate(sc.ID)
	return nil
}

// Test 测试存储策略连通性。
func (s *StorageService) Test(ctx context.Context, user *model.User, id uint64, admin bool) (*dto.StorageTestResp, error) {
	sc, err := s.accessible(ctx, user, id, admin)
	if err != nil {
		return nil, err
	}
	ok := true
	message := "连接成功"
	if terr := s.storages.Test(ctx, sc); terr != nil {
		ok = false
		message = terr.Error()
	}
	if err := s.repos.Storage.UpdateFields(ctx, sc.ID, map[string]any{
		"last_test_at": model.Now(),
		"last_test_ok": ok,
	}); err != nil && s.log != nil {
		s.log.Warn("update storage test result failed", logger.Err(err))
	}
	return &dto.StorageTestResp{OK: ok, Message: message}, nil
}

func (s *StorageService) accessible(ctx context.Context, user *model.User, id uint64, admin bool) (*model.StorageConfig, error) {
	sc, err := s.repos.Storage.GetByID(ctx, id)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if sc == nil {
		return nil, errs.ErrNotFound.WithMessage("存储策略不存在")
	}
	if sc.UserID == nil {
		if !admin {
			return nil, errs.ErrForbidden.WithMessage("无权操作系统存储策略")
		}
		return sc, nil
	}
	if user == nil || *sc.UserID != user.ID {
		return nil, errs.ErrNotFound.WithMessage("存储策略不存在")
	}
	return sc, nil
}

func (s *StorageService) setUserDefault(ctx context.Context, userID, id uint64) error {
	if err := s.repos.User.UpdateFields(ctx, userID, map[string]any{"storage_config_id": id}); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	list, err := s.repos.Storage.ListByUser(ctx, userID)
	if err == nil {
		for _, item := range list {
			if item.ID == id || !item.IsDefault {
				continue
			}
			_ = s.repos.Storage.UpdateFields(ctx, item.ID, map[string]any{"is_default": false})
		}
	}
	return nil
}

func (s *StorageService) getResp(ctx context.Context, id uint64) (*dto.StorageResp, error) {
	sc, err := s.repos.Storage.GetByID(ctx, id)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if sc == nil {
		return nil, errs.ErrNotFound
	}
	return dto.NewStorageResp(sc), nil
}

func validDriver(driver string) bool {
	switch driver {
	case storage.DriverLocal, storage.DriverS3, storage.DriverOSS, storage.DriverCOS, storage.DriverWebDAV:
		return true
	}
	return false
}

func validateStorageFields(driver, endpoint, bucket, url, accessKey, secretKey, username, password string) error {
	switch driver {
	case storage.DriverLocal:
		return nil
	case storage.DriverWebDAV:
		if strings.TrimSpace(url) == "" {
			return errs.ErrInvalidParam.WithDetail("WebDAV 地址不能为空")
		}
		if username == "" && accessKey == "" {
			return errs.ErrInvalidParam.WithDetail("WebDAV 用户名不能为空")
		}
		return nil
	default:
		if strings.TrimSpace(endpoint) == "" {
			return errs.ErrInvalidParam.WithDetail("Endpoint 不能为空")
		}
		if strings.TrimSpace(bucket) == "" {
			return errs.ErrInvalidParam.WithDetail("Bucket 不能为空")
		}
		if accessKey == "" || secretKey == "" {
			return errs.ErrInvalidParam.WithDetail("AccessKey 与 SecretKey 不能为空")
		}
		return nil
	}
}
