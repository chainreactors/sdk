// Package client 提供类型安全的 SDK 客户端
package client

import (
	"fmt"
	"sync"

	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/gogo"
	"github.com/chainreactors/sdk/neutron"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/sdk/spray"
)

// ========================================
// 引擎注册和工厂
// ========================================

var (
	// registry 全局引擎注册表
	registry = make(map[string]sdk.EngineFactory)
)

func init() {
	// 设置全局引擎注册函数，让子包可以注册自己
	sdk.SetRegisterFunc(func(name string, factory sdk.EngineFactory) {
		if _, exists := registry[name]; exists {
			panic(fmt.Sprintf("engine %s already registered", name))
		}
		registry[name] = factory
	})
}

// newEngine 创建引擎实例（内部使用）
func newEngine(name string, config interface{}) (sdk.Engine, error) {
	factory, exists := registry[name]

	if !exists {
		return nil, fmt.Errorf("unknown engine: %s (available: %v)", name, listEngines())
	}

	return factory(config)
}

// listEngines 列出所有已注册的引擎名称（内部使用）
func listEngines() []string {
	engines := make([]string, 0, len(registry))
	for name := range registry {
		engines = append(engines, name)
	}
	return engines
}

// ========================================
// Client 结构
// ========================================

// Client SDK 客户端，提供类型安全的引擎访问
type Client struct {
	engines map[string]sdk.Engine
	mu      sync.RWMutex
}

// New 创建 SDK 客户端
func New() *Client {
	return &Client{
		engines: make(map[string]sdk.Engine),
	}
}

// getOrCreateEngine 获取或创建引擎（线程安全，懒加载）
func (c *Client) getOrCreateEngine(name string) (sdk.Engine, error) {
	// 先尝试读取
	c.mu.RLock()
	if eng, ok := c.engines[name]; ok {
		c.mu.RUnlock()
		return eng, nil
	}
	c.mu.RUnlock()

	// 需要创建
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查
	if eng, ok := c.engines[name]; ok {
		return eng, nil
	}

	// 使用全局工厂创建引擎
	eng, err := newEngine(name, nil)
	if err != nil {
		return nil, err
	}

	c.engines[name] = eng
	return eng, nil
}

// ========================================
// 类型安全的访问方法
// ========================================

// Fingers 获取 Fingers 引擎（类型安全）
func (c *Client) Fingers() (*fingers.Engine, error) {
	eng, err := c.getOrCreateEngine("fingers")
	if err != nil {
		return nil, err
	}
	return eng.(*fingers.Engine), nil
}

// Gogo 获取 Gogo 引擎（类型安全）
func (c *Client) Gogo() (*gogo.GogoEngine, error) {
	eng, err := c.getOrCreateEngine("gogo")
	if err != nil {
		return nil, err
	}
	return eng.(*gogo.GogoEngine), nil
}

// Spray 获取 Spray 引擎（类型安全）
func (c *Client) Spray() (*spray.SprayEngine, error) {
	eng, err := c.getOrCreateEngine("spray")
	if err != nil {
		return nil, err
	}
	return eng.(*spray.SprayEngine), nil
}

// Neutron 获取 Neutron 引擎（类型安全）
func (c *Client) Neutron() (*neutron.Engine, error) {
	eng, err := c.getOrCreateEngine("neutron")
	if err != nil {
		return nil, err
	}
	return eng.(*neutron.Engine), nil
}

// Close 关闭所有引擎
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, eng := range c.engines {
		if err := eng.Close(); err != nil {
			return err
		}
	}
	return nil
}
