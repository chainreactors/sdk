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

// WithRemote 设置远程加载配置并拉取数据
func (c *Config) WithRemote(url, apiKey string) (*Config, error) {
	c.CyberhubURL = url
	c.APIKey = apiKey
	c.LocalPath = ""
	c.Templates = nil
	c.Filename = ""

	if !c.IsRemoteEnabled() {
		return nil, fmt.Errorf("remote config is incomplete")
	}

	client := cyberhub.NewClient(c.CyberhubURL, c.APIKey, c.Timeout)
	responses, err := client.ExportPOCs(context.Background(), nil, nil, "", "", c.ExportFilter)
	if err != nil {
		return nil, err
	}

	loaded := make([]*templates.Template, 0, len(responses))
	for _, resp := range responses {
		loaded = append(loaded, resp.GetTemplate())
	}

	c.Templates = loaded
	return c, nil
}

// WithLocal 设置本地加载并读取数据
func (c *Config) WithLocal(path string) (*Config, error) {
	c.LocalPath = path
	c.CyberhubURL = ""
	c.APIKey = ""
	c.Templates = nil
	c.Filename = ""

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access path %s: %w", path, err)
	}

	var yamlFiles []string
	if info.IsDir() {
		err = filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, walkErr error) error {
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
			return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
		}
	} else {
		yamlFiles = []string{path}
	}

	var loaded []*templates.Template
	for _, yamlFile := range yamlFiles {
		content, readErr := os.ReadFile(yamlFile)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", yamlFile, readErr)
		}

		var list []*templates.Template
		if err := yaml.Unmarshal(content, &list); err == nil && len(list) > 0 {
			loaded = append(loaded, list...)
			continue
		}

		tpl := &templates.Template{}
		if err := yaml.Unmarshal(content, tpl); err != nil {
			return nil, fmt.Errorf("parse %s: %w", yamlFile, err)
		}
		loaded = append(loaded, tpl)
	}

	if len(loaded) == 0 {
		return nil, fmt.Errorf("no templates loaded from %s", path)
	}

	c.Templates = loaded
	return c, nil
}
