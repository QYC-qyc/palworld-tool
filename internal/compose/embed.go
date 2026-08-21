// Package compose 内嵌 docker-compose 模板，供面板自动更新本地 compose 文件。
package compose

import (
	_ "embed"
)

//go:embed docker-compose.prod.yml
var ComposeTemplate []byte

// Template 返回内置的 docker-compose 模板内容。
func Template() []byte {
	return ComposeTemplate
}
