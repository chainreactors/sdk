package fingers

import (
	"fmt"

	"github.com/chainreactors/sdk/pkg/cyberhub"
)

// ========================================
// Config 配置
// ========================================

// Config Fingers SDK 配置
type Config struct {
	cyberhub.Config

	// 引擎配置
	EnableEngines []string // 启用的引擎列表，nil 表示使用默认引擎
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	base := cyberhub.NewConfig()
	return &Config{
		Config:        *base,
		EnableEngines: nil, // 使用默认引擎
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 如果配置了 Cyberhub URL，必须提供 API Key
	if c.CyberhubURL != "" && c.APIKey == "" {
		return fmt.Errorf("api_key is required when cyberhub_url is set")
	}
	if err := c.Config.Validate(); err != nil {
		return err
	}
	return nil
}

// IsRemoteEnabled 是否启用远程加载
func (c *Config) IsRemoteEnabled() bool {
	return c.CyberhubURL != "" && c.APIKey != ""
}

// IsLocalEnabled 是否启用本地加载
func (c *Config) IsLocalEnabled() bool {
	// 本地加载始终可用
	return true
}

// SetEnableEngines 设置启用的引擎列表
func (c *Config) SetEnableEngines(engines []string) *Config {
	c.EnableEngines = engines
	return c
}
