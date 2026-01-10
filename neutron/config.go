package neutron

import (
	"fmt"

	"github.com/chainreactors/sdk/pkg/cyberhub"
)

// ========================================
// Config 配置
// ========================================

// Config Neutron SDK 配置
type Config struct {
	cyberhub.Config

	// 本地配置
	LocalPath string // 本地 POC 文件/目录路径
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	base := cyberhub.NewConfig()
	return &Config{
		Config:    *base,
		LocalPath: "", // 空表示当前目录
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 如果配置了 Cyberhub URL，必须提供 API Key
	if c.CyberhubURL != "" && c.APIKey == "" {
		return fmt.Errorf("api_key is required when cyberhub_url is set")
	}

	// 至少需要配置一种数据源
	if c.CyberhubURL == "" && c.LocalPath == "" {
		// 允许两者都为空，使用默认本地路径
		c.LocalPath = "."
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
	return c.LocalPath != ""
}

// SetLocalPath 设置本地路径
func (c *Config) SetLocalPath(path string) *Config {
	c.LocalPath = path
	return c
}
