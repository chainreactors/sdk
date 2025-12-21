package fingers

import (
	"context"

	fingersLib "github.com/chainreactors/fingers"
)

// ========================================
// 统一 API - 只提供一种加载方式
// ========================================

// Load 加载并返回 fingers 库的 Engine
// config 为 nil 时使用默认本地配置
func Load(config *Config) (*fingersLib.Engine, error) {
	if config == nil {
		config = NewConfig()
	}

	engine, err := NewEngine(config)
	if err != nil {
		return nil, err
	}

	return engine.Load(context.Background())
}

// LoadRemote 从 Cyberhub 加载 fingers 库的 Engine
func LoadRemote(url, apiKey string) (*fingersLib.Engine, error) {
	config := NewConfig().
		SetCyberhubURL(url).
		SetAPIKey(apiKey)
	return Load(config)
}

// LoadLocal 从本地加载 fingers 库的 Engine
// engines 参数指定要加载的引擎列表，为空则加载所有默认引擎
func LoadLocal(engines ...string) (*fingersLib.Engine, error) {
	config := NewConfig().SetEnableEngines(engines)
	return Load(config)
}
