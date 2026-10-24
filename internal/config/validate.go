package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// Validate 校验关键配置项，避免带着错误配置启动。
func (c *Config) Validate() error {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	if c.Site.Name == "" {
		add("site.name is required")
	}
	if c.Server.Addr == "" {
		add("server.addr is required")
	} else if _, _, err := net.SplitHostPort(c.Server.Addr); err != nil {
		add("server.addr is invalid: %v", err)
	}
	switch c.Server.Mode {
	case "debug", "release", "test":
	default:
		add("server.mode must be debug/release/test, got %q", c.Server.Mode)
	}
	if c.Database.Host == "" || c.Database.Database == "" || c.Database.Username == "" {
		add("database host/username/database are required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		add("database.port is invalid")
	}
	if c.Redis.Addr == "" {
		add("redis.addr is required")
	}
	if len(c.Security.JWTSecret) < 32 {
		add("security.jwt_secret must be at least 32 chars")
	}
	if c.Security.SecretKey == "" {
		add("security.secret_key is required")
	}
	if c.Security.BcryptCost < 4 || c.Security.BcryptCost > 31 {
		add("security.bcrypt_cost must be between 4 and 31")
	}
	if c.Security.PasswordMinLength < 6 {
		add("security.password_min_length must be >= 6")
	}
	switch c.Register.Mode {
	case "open", "invite":
	default:
		add("register.mode must be open or invite, got %q", c.Register.Mode)
	}
	switch c.Register.RequireEmailVerify {
	case "off", "soft", "force":
	default:
		add("register.require_email_verify must be off/soft/force, got %q", c.Register.RequireEmailVerify)
	}
	if c.Register.UsernamePattern != "" {
		if _, err := regexp.Compile(c.Register.UsernamePattern); err != nil {
			add("register.username_pattern is invalid: %v", err)
		}
	}
	switch c.Upload.DedupScope {
	case "user", "global":
	default:
		add("upload.dedup_scope must be user or global, got %q", c.Upload.DedupScope)
	}
	switch c.Storage.DefaultDriver {
	case "local", "s3", "oss", "cos", "webdav":
	default:
		add("storage.default_driver is unsupported: %q", c.Storage.DefaultDriver)
	}
	if c.Storage.DefaultDriver == "local" && c.Storage.Local.RootPath == "" {
		add("storage.local.root_path is required when default_driver=local")
	}
	if c.Image.Thumbnail.Enabled && len(c.Image.Thumbnail.Sizes) == 0 {
		add("image.thumbnail.sizes must not be empty when thumbnail enabled")
	}
	if c.Site.URL != "" {
		u, err := url.Parse(c.Site.URL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			add("site.url is invalid: %q", c.Site.URL)
		}
	}
	switch c.Log.Format {
	case "json", "console":
	default:
		add("log.format must be json or console, got %q", c.Log.Format)
	}
	switch c.Log.Output {
	case "stdout", "file", "both":
	default:
		add("log.output must be stdout/file/both, got %q", c.Log.Output)
	}
	if c.Mail.Enabled {
		if c.Mail.Host == "" || c.Mail.FromEmail == "" {
			add("mail.host and mail.from_email are required when mail enabled")
		}
		switch c.Mail.Encryption {
		case "ssl", "starttls", "none":
		default:
			add("mail.encryption must be ssl/starttls/none, got %q", c.Mail.Encryption)
		}
	}
	if c.Queue.WorkerCount <= 0 {
		add("queue.worker_count must be > 0")
	}
	if len(problems) > 0 {
		return errors.New("invalid config:\n  - " + strings.Join(problems, "\n  - "))
	}
	return nil
}
