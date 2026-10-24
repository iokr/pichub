package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"github.com/pichub/pichub/internal/config"
)

// HashPassword 生成 bcrypt 密码哈希。
func HashPassword(password string, cost int) (string, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword 校验密码。
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ValidatePasswordStrength 按配置校验密码强度。
func ValidatePasswordStrength(cfg *config.SecurityConfig, password string) error {
	if len(password) < cfg.PasswordMinLength {
		return fmt.Errorf("密码长度不能少于 %d 位", cfg.PasswordMinLength)
	}
	if len(password) > 72 {
		return errors.New("密码长度不能超过 72 位")
	}
	var hasLetter, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	if cfg.PasswordRequireLetter && !hasLetter {
		return errors.New("密码必须包含字母")
	}
	if cfg.PasswordRequireDigit && !hasDigit {
		return errors.New("密码必须包含数字")
	}
	if cfg.PasswordRequireSpecial && !hasSpecial {
		return errors.New("密码必须包含特殊字符")
	}
	return nil
}

// ValidateUsername 校验用户名格式。
func ValidateUsername(cfg *config.RegisterConfig, username string) error {
	if len(username) < cfg.UsernameMinLength || len(username) > cfg.UsernameMaxLength {
		return fmt.Errorf("用户名长度需在 %d-%d 之间", cfg.UsernameMinLength, cfg.UsernameMaxLength)
	}
	if cfg.UsernamePattern != "" {
		re, err := regexp.Compile(cfg.UsernamePattern)
		if err != nil {
			return err
		}
		if !re.MatchString(username) {
			return errors.New("用户名只能包含字母、数字、下划线和短横线")
		}
	}
	for _, r := range cfg.ReservedUsernames {
		if strings.EqualFold(r, username) {
			return errors.New("该用户名为系统保留")
		}
	}
	return nil
}

// NewAPIToken 生成 API Token，返回明文、哈希与前缀。
func NewAPIToken() (plain, hash, prefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", err
	}
	plain = "ph_" + hex.EncodeToString(b)
	hash = HashToken(plain)
	prefix = plain[:10]
	return plain, hash, prefix, nil
}

// HashToken 计算令牌的 sha256 十六进制。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// NewAccessKey 生成 S3 Access Key（20 位，PK 前缀）。
func NewAccessKey() (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("PK")
	for _, v := range b {
		sb.WriteByte(alphabet[int(v)%len(alphabet)])
	}
	return sb.String(), nil
}

// NewSecretKey 生成 40 位随机 Secret Key。
func NewSecretKey() (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 40)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, v := range b {
		sb.WriteByte(alphabet[int(v)%len(alphabet)])
	}
	return sb.String(), nil
}

// NewOpaqueToken 生成 n 字节随机十六进制令牌。
func NewOpaqueToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SignURL 生成图片访问签名。
func SignURL(secret, hash string, exp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(hash))
	mac.Write([]byte(":"))
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyURLSign 校验图片访问签名。
func VerifyURLSign(secret, hash string, exp int64, sign string) bool {
	if exp <= 0 || sign == "" {
		return false
	}
	if time.Now().Unix() > exp {
		return false
	}
	expected := SignURL(secret, hash, exp)
	return hmac.Equal([]byte(expected), []byte(sign))
}
