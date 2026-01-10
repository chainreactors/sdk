package neutron

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chainreactors/neutron/templates"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"gopkg.in/yaml.v3"
)

// NewConfig 创建默认配置
func NewConfig() *Config {
	base := cyberhub.NewConfig()
	return &Config{
		Config:    *base,
		LocalPath: "",
		Templates: nil,
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
	c.Templates = tpls
	return c
}

// WithCyberhub 设置远程加载配置（不立即拉取）
func (c *Config) WithCyberhub(url, apiKey string) *Config {
	c.CyberhubURL = url
	c.APIKey = apiKey
	c.LocalPath = ""
	c.Templates = nil
	c.Filename = ""
	return c
}

// WithLocalFile 设置本地加载配置（不立即读取）
func (c *Config) WithLocalFile(path string) *Config {
	c.LocalPath = path
	c.CyberhubURL = ""
	c.APIKey = ""
	c.Templates = nil
	c.Filename = ""
	return c
}

// Load 执行数据加载
func (c *Config) Load(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if len(c.Templates) > 0 {
		return nil
	}
	if c.LocalPath != "" {
		info, err := os.Stat(c.LocalPath)
		if err != nil {
			return fmt.Errorf("failed to access path %s: %w", c.LocalPath, err)
		}

		var yamlFiles []string
		if info.IsDir() {
			err = filepath.Walk(c.LocalPath, func(filePath string, fileInfo os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				ext := filepath.Ext(filePath)
				if ext == ".yaml" || ext == ".yml" {
					yamlFiles = append(yamlFiles, filePath)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to walk directory %s: %w", c.LocalPath, err)
			}
		} else {
			yamlFiles = []string{c.LocalPath}
		}

		var loaded []*templates.Template
		for _, yamlFile := range yamlFiles {
			content, readErr := os.ReadFile(yamlFile)
			if readErr != nil {
				return fmt.Errorf("read %s: %w", yamlFile, readErr)
			}

			var list []*templates.Template
			if err := yaml.Unmarshal(content, &list); err == nil && len(list) > 0 {
				loaded = append(loaded, list...)
				continue
			}

			tpl := &templates.Template{}
			if err := yaml.Unmarshal(content, tpl); err != nil {
				return fmt.Errorf("parse %s: %w", yamlFile, err)
			}
			loaded = append(loaded, tpl)
		}

		if len(loaded) == 0 {
			return fmt.Errorf("no templates loaded from %s", c.LocalPath)
		}

		c.Templates = loaded
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
		c.Templates = loaded
		return nil
	}
	return fmt.Errorf("no data source configured")
}
