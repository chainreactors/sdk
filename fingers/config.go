package fingers

import (
	"context"
	"fmt"
	"os"

	"github.com/chainreactors/fingers/alias"
	fingersEngine "github.com/chainreactors/fingers/fingers"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"gopkg.in/yaml.v3"
)

// NewConfig 创建默认配置
func NewConfig() *Config {
	base := cyberhub.NewConfig()
	return &Config{
		Config:        *base,
		EnableEngines: nil,
		Fingers:       nil,
		Aliases:       nil,
	}
}

// ========================================
// Config 配置
// ========================================

// Config Fingers SDK 配置
type Config struct {
	cyberhub.Config

	// 引擎配置
	EnableEngines []string
	Fingers       fingersEngine.Fingers
	Aliases       []*alias.Alias
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

// WithRemote 设置远程加载配置并拉取数据
func (c *Config) WithRemote(url, apiKey string) (*Config, error) {
	c.CyberhubURL = url
	c.APIKey = apiKey
	c.Filename = ""
	c.Fingers = nil
	c.Aliases = nil

	if !c.IsRemoteEnabled() {
		return nil, fmt.Errorf("remote config is incomplete")
	}

	client := cyberhub.NewClient(c.CyberhubURL, c.APIKey, c.Timeout)
	fingersData, aliases, err := client.ExportFingers(context.Background(), "", c.ExportFilter)
	if err != nil {
		return nil, err
	}

	c.Fingers = fingersData
	c.Aliases = aliases
	return c, nil
}

// WithLocal 设置本地文件加载并读取数据
func (c *Config) WithLocal(filename string) (*Config, error) {
	c.Filename = filename
	c.CyberhubURL = ""
	c.APIKey = ""
	c.Fingers = nil
	c.Aliases = nil

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var raw []*fingersEngine.Finger
	if err := yaml.NewDecoder(file).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode fingerprints: %w", err)
	}

	c.Fingers = fingersEngine.Fingers(raw)
	c.Aliases = nil
	return c, nil
}

// WithFingers 设置指纹数据
func (c *Config) WithFingers(fingers fingersEngine.Fingers) *Config {
	c.Fingers = fingers
	return c
}

// WithAliases 设置别名数据
func (c *Config) WithAliases(aliases []*alias.Alias) *Config {
	c.Aliases = aliases
	return c
}
