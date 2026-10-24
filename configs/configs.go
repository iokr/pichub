// Package configs 提供内置的配置模板文件。
package configs

import _ "embed"

// ExampleYAML 是完整的 config.yaml 模板（带注释）。
//
//go:embed config.yaml
var ExampleYAML []byte
