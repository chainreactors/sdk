package neutron

import (
	"context"

	"github.com/chainreactors/neutron/templates"
)

// Load 加载并返回 templates 列表
// config 为 nil 时使用默认本地配置
func Load(config *Config) ([]*templates.Template, error) {
	if config == nil {
		config = NewConfig()
	}

	engine, err := NewEngine(config)
	if err != nil {
		return nil, err
	}

	return engine.Load(context.Background())
}

// LoadRemote 从 Cyberhub 加载 templates 列表
func LoadRemote(url, apiKey string) ([]*templates.Template, error) {
	config := NewConfig().
		SetCyberhubURL(url).
		SetAPIKey(apiKey)
	return Load(config)
}

// LoadLocal 从本地文件/目录加载 templates 列表
// path 为空则从当前目录加载
func LoadLocal(path string) ([]*templates.Template, error) {
	config := NewConfig().SetLocalPath(path)
	return Load(config)
}
