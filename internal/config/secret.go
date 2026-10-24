package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// randomHex 生成 n 字节随机数的十六进制字符串。
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// randomBase64 生成 n 字节随机数的标准 Base64 字符串。
func randomBase64(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

// patchSecurityKeys 在保留注释的前提下，回写 security 段内的密钥配置。
func patchSecurityKeys(path string, values map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")

	sectionStart, sectionEnd := -1, len(lines)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if line == trimmed && trimmed == "security:" {
			sectionStart = i
			for j := i + 1; j < len(lines); j++ {
				l := lines[j]
				t := strings.TrimSpace(l)
				if t == "" || strings.HasPrefix(t, "#") {
					continue
				}
				// 回到顶层的下一个配置段
				if l == t {
					sectionEnd = j
					break
				}
			}
			break
		}
	}

	if sectionStart < 0 {
		// 没有 security 段时追加一段。
		lines = append(lines, "", "security:")
		sectionStart = len(lines) - 1
		sectionEnd = len(lines)
	}

	replaced := map[string]bool{}
	for i := sectionStart + 1; i < sectionEnd && i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		for key, val := range values {
			if strings.HasPrefix(trimmed, key+":") {
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				lines[i] = fmt.Sprintf("%s%s: %q", indent, key, val)
				replaced[key] = true
			}
		}
	}

	var missing []string
	for key, val := range values {
		if !replaced[key] {
			missing = append(missing, fmt.Sprintf("  %s: %q", key, val))
		}
	}
	if len(missing) > 0 {
		tail := append([]string{}, lines[sectionEnd:]...)
		lines = append(lines[:sectionEnd], missing...)
		lines = append(lines, tail...)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600)
}
