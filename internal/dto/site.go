package dto

// SiteConfigResp 是站点公开配置响应，供前端初始化使用。
type SiteConfigResp struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	LogoURL           string   `json:"logo_url"`
	ICP               string   `json:"icp"`
	SiteURL           string   `json:"site_url"`
	RegisterEnabled   bool     `json:"register_enabled"`
	RegisterMode      string   `json:"register_mode"`
	EmailVerifyLevel  string   `json:"email_verify_level"`
	CaptchaEnabled    bool     `json:"captcha_enabled"`
	DefaultVisibility string   `json:"default_visibility"`
	MaxFileSize       int64    `json:"max_file_size"`
	AllowedExtensions []string `json:"allowed_extensions"`
	ThumbnailEnabled  bool     `json:"thumbnail_enabled"`
	ThumbnailSizes    []string `json:"thumbnail_sizes"`
	S3GatewayEnabled  bool     `json:"s3_gateway_enabled"`
	S3GatewayPrefix   string   `json:"s3_gateway_prefix"`
}
