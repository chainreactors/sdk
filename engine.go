// Package sdk 提供统一的引擎工厂和注册功能
package sdk

import (
	"fmt"

	sdk "github.com/chainreactors/sdk/pkg"
)

// ========================================
// 引擎注册和工厂
// ========================================

var (
	// registry 全局引擎注册表
	registry = make(map[string]sdk.EngineFactory)
)

func init() {
	// 设置全局注册函数，让子包可以注册自己
	sdk.SetRegisterFunc(func(name string, factory sdk.EngineFactory) {
		if _, exists := registry[name]; exists {
			panic(fmt.Sprintf("engine %s already registered", name))
		}
		registry[name] = factory
	})
}

// NewEngine 创建引擎实例
// name: 引擎名称
// config: 引擎配置（可选，传 nil 使用默认配置）
func NewEngine(name string, config interface{}) (sdk.Engine, error) {
	factory, exists := registry[name]

	if !exists {
		return nil, fmt.Errorf("unknown engine: %s (available: %v)", name, ListEngines())
	}

	return factory(config)
}

// ListEngines 列出所有已注册的引擎名称
func ListEngines() []string {
	engines := make([]string, 0, len(registry))
	for name := range registry {
		engines = append(engines, name)
	}
	return engines
}

// Unregister 注销引擎（主要用于测试）
func Unregister(name string) {
	delete(registry, name)
}
