package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/pichub/pichub/configs"
)

// defaultSearchPaths 是未显式指定 -c 时的配置查找顺序。
var defaultSearchPaths = []string{
	"./config.yaml",
	"/app/config.yaml",
	"/etc/pichub/config.yaml",
	"../../configs/config.yaml",
}

// Load 按优先级加载配置：
// -c 参数 > ./config.yaml > /app/config.yaml > /etc/pichub/config.yaml > 内置默认值。
//
// 加载成功后还会完成密钥自动生成、密钥回写、权限检查与字段校验。
func Load(explicitPath string) (*Config, error) {
	path := resolvePath(explicitPath)
	cfg := Default()

	if path == "" {
		// 没有任何配置文件：使用内置默认值，并落盘一份模板供后续修改。
		cfg.SetLoadedPath("")
		cfg.warnf("no config file found, using built-in defaults")
	} else {
		v := viper.New()
		v.SetConfigFile(path)
		v.SetConfigType("yaml")
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
		decoderOpts := func(dc *mapstructure.DecoderConfig) {
			dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.StringToSliceHookFunc(","),
			)
		}
		if err := v.UnmarshalExact(cfg, decoderOpts); err != nil {
			return nil, fmt.Errorf("parse config %s (unknown or invalid field): %w", path, err)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			abs = path
		}
		cfg.SetLoadedPath(abs)
		checkPermission(cfg, path)
	}

	if err := prepareSecrets(cfg, path); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func resolvePath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	for _, p := range defaultSearchPaths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func checkPermission(cfg *Config, path string) {
	if runtime.GOOS == "windows" {
		// Windows 的 NTFS 不映射 Unix 权限位，os.Stat 恒返回 0666，跳过以免误报。
		return
	}
	st, err := os.Stat(path)
	if err != nil {
		return
	}
	if st.Mode().Perm()&0o077 != 0 {
		cfg.warnf("config file %s permission is %#o, recommend 0600", path, st.Mode().Perm())
	}
}

// prepareSecrets 在 auto_generate_secrets 开启时生成缺失的密钥并回写 YAML。
func prepareSecrets(cfg *Config, path string) error {
	if !cfg.Security.AutoGenerateSecrets {
		return nil
	}
	generated := map[string]string{}
	if weakSecret(cfg.Security.JWTSecret) {
		cfg.Security.JWTSecret = randomHex(32)
		generated["jwt_secret"] = cfg.Security.JWTSecret
	}
	if weakSecret(cfg.Security.SecretKey) {
		cfg.Security.SecretKey = randomBase64(32)
		generated["secret_key"] = cfg.Security.SecretKey
	}
	if weakSecret(cfg.Security.URLSignSecret) {
		cfg.Security.URLSignSecret = randomHex(32)
		generated["url_sign_secret"] = cfg.Security.URLSignSecret
	}

	if path == "" {
		// 首次启动：把带新密钥的模板写入 ./config.yaml。
		if err := writeTemplate("./config.yaml", cfg); err != nil {
			cfg.warnf("write default config ./config.yaml failed: %v", err)
			return nil
		}
		abs, _ := filepath.Abs("./config.yaml")
		cfg.SetLoadedPath(abs)
		return nil
	}
	if len(generated) == 0 {
		return nil
	}
	if err := patchSecurityKeys(path, generated); err != nil {
		return fmt.Errorf("write back generated secrets to %s: %w", path, err)
	}
	cfg.warnf("auto generated secrets written back to %s", path)
	return nil
}

// writeTemplate 用内置模板生成配置文件，并把生成的密钥填入。
func writeTemplate(path string, cfg *Config) error {
	if _, err := os.Stat(path); err == nil {
		return patchSecurityKeys(path, map[string]string{
			"jwt_secret":      cfg.Security.JWTSecret,
			"secret_key":      cfg.Security.SecretKey,
			"url_sign_secret": cfg.Security.URLSignSecret,
		})
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	if err := os.WriteFile(path, configs.ExampleYAML, 0o600); err != nil {
		return err
	}
	return patchSecurityKeys(path, map[string]string{
		"jwt_secret":      cfg.Security.JWTSecret,
		"secret_key":      cfg.Security.SecretKey,
		"url_sign_secret": cfg.Security.URLSignSecret,
	})
}

func weakSecret(s string) bool {
	return len(s) < 16 || strings.HasPrefix(s, "CHANGE_ME")
}
