package proton

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/neutron/protocols"
	"github.com/chainreactors/proton/template"
	"gopkg.in/yaml.v3"
)

type Config struct {
	TemplatePaths    []string           // paths to YAML template files or directories
	TemplateData     [][]byte           // raw YAML data (each entry is a YAML array of templates)
	Rules            []Rule        // pre-compiled rules (skip template parsing)
	ResourceProvider func(string) []byte // loader for embedded template data
	Categories       []string           // categories to load via ResourceProvider (e.g. "keys", "spray")

	TextOnly bool // skip binary files (default: true)
	Capacity int

	Tags        []string
	ExcludeTags []string
	IDs         []string
	ExcludeIDs  []string
}

func NewConfig() *Config {
	return &Config{
		TextOnly: true,
	}
}

func (c *Config) Validate() error {
	return nil
}

func (c *Config) WithTemplatePaths(paths ...string) *Config {
	c.TemplatePaths = append(c.TemplatePaths, paths...)
	return c
}

func (c *Config) WithTemplateData(data ...[]byte) *Config {
	c.TemplateData = append(c.TemplateData, data...)
	return c
}

func (c *Config) WithRules(rules ...Rule) *Config {
	c.Rules = append(c.Rules, rules...)
	return c
}

func (c *Config) WithResourceProvider(provider func(string) []byte) *Config {
	c.ResourceProvider = provider
	return c
}

func (c *Config) WithCategories(categories ...string) *Config {
	c.Categories = append(c.Categories, categories...)
	return c
}

func (c *Config) WithTextOnly(textOnly bool) *Config {
	c.TextOnly = textOnly
	return c
}

func (c *Config) WithCapacity(total int) *Config {
	c.Capacity = total
	return c
}

func (c *Config) WithTags(tags ...string) *Config {
	c.Tags = append(c.Tags, tags...)
	return c
}

func (c *Config) WithExcludeTags(tags ...string) *Config {
	c.ExcludeTags = append(c.ExcludeTags, tags...)
	return c
}

func (c *Config) WithIDs(ids ...string) *Config {
	c.IDs = append(c.IDs, ids...)
	return c
}

func (c *Config) WithExcludeIDs(ids ...string) *Config {
	c.ExcludeIDs = append(c.ExcludeIDs, ids...)
	return c
}

func (c *Config) Load() ([]Rule, error) {
	execOpts := &protocols.ExecuterOptions{
		Options: &protocols.Options{TextOnly: c.TextOnly},
	}

	var tmpls []*template.Template

	if c.ResourceProvider != nil && len(c.Categories) > 0 {
		for _, cat := range c.Categories {
			name := "found_" + strings.ReplaceAll(cat, "/", "_")
			data := c.ResourceProvider(name)
			if len(data) == 0 {
				continue
			}
			loaded, err := parseTemplateArray(name, data, execOpts)
			if err != nil {
				continue
			}
			tmpls = append(tmpls, loaded...)
		}
	}

	for i, data := range c.TemplateData {
		loaded, err := parseTemplateArray(fmt.Sprintf("data-%d", i), data, execOpts)
		if err != nil {
			continue
		}
		tmpls = append(tmpls, loaded...)
	}

	for _, path := range c.TemplatePaths {
		loaded, err := loadFromPath(path, execOpts)
		if err != nil {
			return nil, fmt.Errorf("loading template %s: %w", path, err)
		}
		tmpls = append(tmpls, loaded...)
	}

	tmpls = filterTemplates(tmpls, c)

	var rules []Rule
	for _, tmpl := range tmpls {
		rules = append(rules, Rule{
			ID:       tmpl.Id,
			Name:     tmpl.Info.Name,
			Severity: tmpl.Info.Severity,
			Requests: tmpl.RequestsFile,
		})
	}

	rules = append(rules, c.Rules...)
	return rules, nil
}

func parseTemplateArray(name string, data []byte, execOpts *protocols.ExecuterOptions) ([]*template.Template, error) {
	var pocs []interface{}
	if err := yaml.Unmarshal(data, &pocs); err != nil {
		return nil, err
	}

	var tmpls []*template.Template
	for _, poc := range pocs {
		bs, err := yaml.Marshal(poc)
		if err != nil {
			continue
		}
		tmpl, err := parseTemplate("embedded:"+name, bs, execOpts)
		if err != nil {
			continue
		}
		tmpls = append(tmpls, tmpl)
	}
	return tmpls, nil
}

func parseTemplate(name string, data []byte, execOpts *protocols.ExecuterOptions) (*template.Template, error) {
	var tmpl template.Template
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parse %s: %v", name, err)
	}
	if len(tmpl.RequestsFile) == 0 {
		return nil, fmt.Errorf("no file requests in %s", name)
	}
	if err := tmpl.Compile(execOpts); err != nil {
		return nil, fmt.Errorf("compile %s: %v", name, err)
	}
	return &tmpl, nil
}

func loadFromPath(path string, execOpts *protocols.ExecuterOptions) ([]*template.Template, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		tmpl, err := parseTemplate(path, data, execOpts)
		if err != nil {
			return nil, err
		}
		return []*template.Template{tmpl}, nil
	}

	var tmpls []*template.Template
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".yaml") && !strings.HasSuffix(p, ".yml") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		tmpl, err := parseTemplate(p, data, execOpts)
		if err != nil {
			return nil
		}
		tmpls = append(tmpls, tmpl)
		return nil
	})
	return tmpls, nil
}

func filterTemplates(tmpls []*template.Template, cfg *Config) []*template.Template {
	includeTags := toSet(cfg.Tags)
	excludeTags := toSet(cfg.ExcludeTags)
	includeIDs := toSet(cfg.IDs)
	excludeIDs := toSet(cfg.ExcludeIDs)

	if len(includeTags) == 0 && len(excludeTags) == 0 &&
		len(includeIDs) == 0 && len(excludeIDs) == 0 {
		return tmpls
	}

	var filtered []*template.Template
	for _, tmpl := range tmpls {
		if len(excludeIDs) > 0 && excludeIDs[tmpl.Id] {
			continue
		}
		if len(includeIDs) > 0 && !includeIDs[tmpl.Id] {
			continue
		}
		tags := tmpl.GetTags()
		if len(excludeTags) > 0 && matchAnyTag(tags, excludeTags) {
			continue
		}
		if len(includeTags) > 0 && !matchAnyTag(tags, includeTags) {
			continue
		}
		filtered = append(filtered, tmpl)
	}
	return filtered
}

func toSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[strings.TrimSpace(strings.ToLower(item))] = true
	}
	return s
}

func matchAnyTag(tags []string, set map[string]bool) bool {
	for _, tag := range tags {
		if set[strings.TrimSpace(strings.ToLower(tag))] {
			return true
		}
	}
	return false
}
