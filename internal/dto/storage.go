package dto

import (
	"encoding/json"
	"time"

	"github.com/pichub/pichub/internal/model"
)

// storageConfigPayload 是存储策略非敏感配置的 JSON 结构。
type storageConfigPayload struct {
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

// StorageResp 是存储策略响应。
type StorageResp struct {
	ID           uint64     `json:"id"`
	UserID       *uint64    `json:"user_id"`
	Name         string     `json:"name"`
	Driver       string     `json:"driver"`
	Domain       string     `json:"domain"`
	PathPrefix   string     `json:"path_prefix"`
	IsDefault    bool       `json:"is_default"`
	Status       int8       `json:"status"`
	RootPath     string     `json:"root_path,omitempty"`
	PathTemplate string     `json:"path_template,omitempty"`
	Endpoint     string     `json:"endpoint,omitempty"`
	Region       string     `json:"region,omitempty"`
	Bucket       string     `json:"bucket,omitempty"`
	PathStyle    bool       `json:"path_style"`
	UseSSL       *bool      `json:"use_ssl"`
	PublicURL    string     `json:"public_url,omitempty"`
	URL          string     `json:"url,omitempty"`
	HasSecret    bool       `json:"has_secret"`
	LastTestAt   *time.Time `json:"last_test_at"`
	LastTestOK   *bool      `json:"last_test_ok"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// NewStorageResp 由模型构造存储策略响应。
func NewStorageResp(sc *model.StorageConfig) *StorageResp {
	if sc == nil {
		return nil
	}
	resp := &StorageResp{
		ID:         sc.ID,
		UserID:     sc.UserID,
		Name:       sc.Name,
		Driver:     sc.Driver,
		Domain:     sc.Domain,
		PathPrefix: sc.PathPrefix,
		IsDefault:  sc.IsDefault,
		Status:     sc.Status,
		HasSecret:  len(sc.Secret) > 0,
		LastTestAt: sc.LastTestAt,
		LastTestOK: sc.LastTestOK,
		CreatedAt:  sc.CreatedAt,
		UpdatedAt:  sc.UpdatedAt,
	}
	if len(sc.Config) > 0 {
		var conf storageConfigPayload
		if err := json.Unmarshal(sc.Config, &conf); err == nil {
			resp.RootPath = conf.RootPath
			resp.PathTemplate = conf.PathTemplate
			resp.Endpoint = conf.Endpoint
			resp.Region = conf.Region
			resp.Bucket = conf.Bucket
			resp.PathStyle = conf.PathStyle
			resp.UseSSL = conf.UseSSL
			resp.PublicURL = conf.PublicURL
			resp.URL = conf.URL
		}
	}
	return resp
}

// NewStorageRespList 批量构造存储策略响应。
func NewStorageRespList(list []*model.StorageConfig) []*StorageResp {
	out := make([]*StorageResp, 0, len(list))
	for _, sc := range list {
		out = append(out, NewStorageResp(sc))
	}
	return out
}

// StorageCreateReq 是创建存储策略请求。
type StorageCreateReq struct {
	Name         string `json:"name" binding:"required"`
	Driver       string `json:"driver" binding:"required"`
	Domain       string `json:"domain"`
	PathPrefix   string `json:"path_prefix"`
	IsDefault    bool   `json:"is_default"`
	Status       *int8  `json:"status"`
	RootPath     string `json:"root_path"`
	PathTemplate string `json:"path_template"`
	Endpoint     string `json:"endpoint"`
	Region       string `json:"region"`
	Bucket       string `json:"bucket"`
	PathStyle    bool   `json:"path_style"`
	UseSSL       *bool  `json:"use_ssl"`
	PublicURL    string `json:"public_url"`
	URL          string `json:"url"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

// StorageUpdateReq 是更新存储策略请求。
type StorageUpdateReq struct {
	Name         *string `json:"name"`
	Domain       *string `json:"domain"`
	PathPrefix   *string `json:"path_prefix"`
	IsDefault    *bool   `json:"is_default"`
	Status       *int8   `json:"status"`
	RootPath     *string `json:"root_path"`
	PathTemplate *string `json:"path_template"`
	Endpoint     *string `json:"endpoint"`
	Region       *string `json:"region"`
	Bucket       *string `json:"bucket"`
	PathStyle    *bool   `json:"path_style"`
	UseSSL       *bool   `json:"use_ssl"`
	PublicURL    *string `json:"public_url"`
	URL          *string `json:"url"`
	AccessKey    *string `json:"access_key"`
	SecretKey    *string `json:"secret_key"`
	Username     *string `json:"username"`
	Password     *string `json:"password"`
}

// StorageTestResp 是存储策略连通性测试响应。
type StorageTestResp struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}
