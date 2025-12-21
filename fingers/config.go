package fingers

import (
	"fmt"
	"time"
)

// ========================================
// Config 配置
// ========================================

// Config Fingers SDK 配置
type Config struct {
	// Cyberhub 配置（可选）
	CyberhubURL string // Cyberhub API 地址，为空则仅使用本地指纹
	APIKey      string // API Key 认证

	// 引擎配置
	EnableEngines []string // 启用的引擎列表，nil 表示使用默认引擎

	// 过滤配置
	Source string // 指纹来源过滤（如 "github", "local" 等）

	// 请求配置
	Timeout    time.Duration // HTTP 请求超时时间
	MaxRetries int           // 最大重试次数
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		CyberhubURL:   "",
		APIKey:        "",
		EnableEngines: nil, // 使用默认引擎
		Source:        "",  // 不过滤来源
		Timeout:       10 * time.Second,
		MaxRetries:    3,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 如果配置了 Cyberhub URL，必须提供 API Key
	if c.CyberhubURL != "" && c.APIKey == "" {
		return fmt.Errorf("api_key is required when cyberhub_url is set")
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

// SetEnableEngines 设置启用的引擎列表
func (c *Config) SetEnableEngines(engines []string) *Config {
	c.EnableEngines = engines
	return c
}

// SetSource 设置指纹来源过滤
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
