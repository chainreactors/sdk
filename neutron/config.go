package neutron

import (
	"context"
	"fmt"
	"time"

	"github.com/chainreactors/sdk/pkg/cyberhub"
	"github.com/chainreactors/sdk/pkg/types"
)

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		Timeout: 10 * time.Second,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	return nil
}

// WithProvider 设置远程数据源
func (c *Config) WithProvider(p *cyberhub.Provider) *Config {
	c.Provider = p
	return c
}

// WithTemplates 设置已加载的模板
func (c *Config) WithTemplates(tpls []*types.Template) *Config {
	c.Templates = (Templates{}).Merge(tpls)
	return c
}

// WithLocalFile 设置本地加载配置
func (c *Config) WithLocalFile(path string) *Config {
	c.LocalPath = path
	c.Provider = nil
	c.Templates = Templates{}
	return c
}

// WithFilter filters current Templates using predicate.
func (c *Config) WithFilter(predicate func(*types.Template) bool) *Config {
	if c == nil {
		return c
	}
	c.Templates = c.Templates.Filter(predicate)
	return c
}

// WithCapacity sets the total capacity for concurrent Execute calls.
func (c *Config) WithCapacity(total int) *Config {
	c.Capacity = total
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
	if c.Provider != nil {
		tpls, err := c.Provider.POCs(ctx)
		if err != nil {
			return err
		}
		c.Templates = (Templates{}).Merge(tpls)
		return nil
	}

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

	return fmt.Errorf("no data source configured: please use WithLocalFile(), Provider, or WithTemplates() to configure template data")
}
