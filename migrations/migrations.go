// Package migrations 内嵌数据库迁移脚本，保证单二进制即可完成建表。
package migrations

import (
	"embed"
	"sort"
	"strings"
)

//go:embed *.sql
var files embed.FS

// UpFiles 返回全部 .up.sql 文件名（升序）。
func UpFiles() ([]string, error) {
	entries, err := files.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Read 读取迁移脚本内容。
func Read(name string) (string, error) {
	b, err := files.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
