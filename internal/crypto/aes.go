// Package crypto 提供 AES-GCM 加解密，用于存储敏感配置。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// Cipher 封装 AES-256-GCM。
type Cipher struct {
	aead cipher.AEAD
}

// New 从配置中的 secret_key 创建加密器。
// secret_key 建议为 32 字节的 Base64；非法 Base64 时退化为 sha256 派生密钥。
func New(secretKey string) (*Cipher, error) {
	if secretKey == "" {
		return nil, errors.New("secret_key is empty")
	}
	var key []byte
	if raw, err := base64.StdEncoding.DecodeString(secretKey); err == nil && len(raw) == 32 {
		key = raw
	} else {
		sum := sha256.Sum256([]byte(secretKey))
		key = sum[:]
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt 加密明文，返回 nonce+ciphertext。
func (c *Cipher) Encrypt(plain []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, plain, nil), nil
}

// Decrypt 解密 nonce+ciphertext。
func (c *Cipher) Decrypt(enc []byte) ([]byte, error) {
	ns := c.aead.NonceSize()
	if len(enc) < ns {
		return nil, errors.New("ciphertext too short")
	}
	return c.aead.Open(nil, enc[:ns], enc[ns:], nil)
}

// EncryptString 加密字符串。
func (c *Cipher) EncryptString(plain string) ([]byte, error) {
	return c.Encrypt([]byte(plain))
}

// DecryptString 解密为字符串。
func (c *Cipher) DecryptString(enc []byte) (string, error) {
	b, err := c.Decrypt(enc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// EncryptJSON 加密 JSON 字节。
func (c *Cipher) EncryptJSON(plain []byte) ([]byte, error) { return c.Encrypt(plain) }
