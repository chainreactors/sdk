package neutron

import (
	"fmt"
	"time"
)

// ========================================
// Config 配置
// ========================================

// Config Neutron SDK 配置
type Config struct {
	// Cyberhub 配置（可选）
	CyberhubURL string // Cyberhub API 地址，为空则仅使用本地POC
	APIKey      string // API Key 认证

	// 本地配置
	LocalPath string // 本地 POC 文件/目录路径

	// 过滤配置
	Source string // POC 来源过滤（如 "github", "local" 等）

	// 请求配置
	Timeout    time.Duration // HTTP 请求超时时间
	MaxRetries int           // 最大重试次数
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		CyberhubURL: "",
		APIKey:      "",
		LocalPath:   "", // 空表示当前目录
		Source:      "", // 不过滤来源
		Timeout:     10 * time.Second,
		MaxRetries:  3,
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

// SetCyberhubURL 设置 Cyberhub URL
func (c *Config) SetCyberhubURL(url string) *Config {
	c.CyberhubURL = url
	return c
}

// SetAPIKey 设置 API Key
func (c *Config) SetAPIKey(key string) *Config {
	c.APIKey = key
	return c
}

// SetLocalPath 设置本地路径
func (c *Config) SetLocalPath(path string) *Config {
	c.LocalPath = path
	return c
}

// SetSource 设置 POC 来源过滤
func (c *Config) SetSource(source string) *Config {
	c.Source = source
	return c
}

// SetTimeout 设置请求超时时间
func (c *Config) SetTimeout(timeout time.Duration) *Config {
	c.Timeout = timeout
	return c
}

// SetMaxRetries 设置最大重试次数
func (c *Config) SetMaxRetries(maxRetries int) *Config {
	c.MaxRetries = maxRetries
	return c
}
