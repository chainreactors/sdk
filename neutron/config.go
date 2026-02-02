package neutron

import (
	"context"
	"fmt"

	"github.com/chainreactors/neutron/templates"
	"github.com/chainreactors/sdk/pkg/cyberhub"
)

// NewConfig 创建默认配置
func NewConfig() *Config {
	base := cyberhub.NewConfig()
	return &Config{
		Config:    *base,
		LocalPath: "",
		Templates: Templates{},
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

// WithTemplates 设置已加载的模板
func (c *Config) WithTemplates(tpls []*templates.Template) *Config {
	c.Templates = (Templates{}).Merge(tpls)
	return c
}

// WithCyberhub 设置远程加载配置（不立即拉取）
func (c *Config) WithCyberhub(url, apiKey string) *Config {
	c.CyberhubURL = url
	c.APIKey = apiKey
	c.LocalPath = ""
	c.Templates = Templates{}
	c.Filename = ""
	return c
}

// WithLocalFile 设置本地加载配置（不立即读取）
func (c *Config) WithLocalFile(path string) *Config {
	c.LocalPath = path
	c.CyberhubURL = ""
	c.APIKey = ""
	c.Templates = Templates{}
	c.Filename = ""
	return c
}

// WithFilter filters current Templates using predicate.
func (c *Config) WithFilter(predicate func(*templates.Template) bool) *Config {
	if c == nil {
		return c
	}
	c.Templates = c.Templates.Filter(predicate)
	return c
}

// Load 执行数据加载
func (c *Config) Load(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.Templates.Len() > 0 {
		return nil
	}
	if c.LocalPath != "" {
		loaded, err := loadTemplatesFromPath(c.LocalPath)
		if err != nil {
			return err
		}
		c.Templates = (Templates{}).Merge(loaded)
		return nil
	}
	if c.IsRemoteEnabled() {
		client := cyberhub.NewClient(c.CyberhubURL, c.APIKey, c.Timeout)
		responses, err := client.ExportPOCs(ctx, nil, nil, "", "", c.ExportFilter)
		if err != nil {
			return err
		}

		loaded := make([]*templates.Template, 0, len(responses))
		for _, resp := range responses {
			loaded = append(loaded, resp.GetTemplate())
		}
		c.Templates = (Templates{}).Merge(loaded)
		return nil
	}

	// 尝试从默认路径加载模板
	defaultPaths := []string{
		"templates",
		"pocs",
		"./templates",
		"./pocs",
	}

	for _, path := range defaultPaths {
		loaded, err := loadTemplatesFromPath(path)
		if err == nil {
			c.Templates = (Templates{}).Merge(loaded)
			return nil
		}
	}

	// 如果所有默认路径都失败，返回友好的错误信息
	return fmt.Errorf("no data source configured: please use WithLocalFile(), WithCyberhub(), or WithTemplates() to configure template data")
}
