package cyberhub

import (
	"fmt"
	"os"
	"time"

	"github.com/chainreactors/fingers/fingers"
	"github.com/chainreactors/neutron/templates"
	"gopkg.in/yaml.v3"
)

type Config struct {
	CyberhubURL  string
	APIKey       string
	Timeout      time.Duration
	ExportFilter *ExportFilter
	Filename     string
}

func NewConfig() *Config {
	return &Config{
		CyberhubURL:  "",
		APIKey:       "",
		Timeout:      10 * time.Second,
		ExportFilter: NewExportFilter(),
		Filename:     "",
	}
}

func (c *Config) Validate() error {
	if c.ExportFilter == nil {
		c.ExportFilter = NewExportFilter()
	}
	return nil
}

func (c *Config) IsRemoteEnabled() bool {
	return c.CyberhubURL != "" && c.APIKey != ""
}

func (c *Config) SetCyberhubURL(url string) *Config {
	c.CyberhubURL = url
	return c
}

func (c *Config) SetAPIKey(key string) *Config {
	c.APIKey = key
	return c
}

func (c *Config) SetTimeout(timeout time.Duration) *Config {
	c.Timeout = timeout
	return c
}

func (c *Config) SetExportFilter(filter *ExportFilter) *Config {
	if filter == nil {
		filter = NewExportFilter()
	}
	c.ExportFilter = filter
	return c
}

func (c *Config) SetTags(tags ...string) *Config {
	if c.ExportFilter == nil {
		c.ExportFilter = NewExportFilter()
	}
	c.ExportFilter.Tags = tags
	return c
}

func (c *Config) SetSources(sources ...string) *Config {
	if c.ExportFilter == nil {
		c.ExportFilter = NewExportFilter()
	}
	c.ExportFilter.Sources = sources
	return c
}

func (c *Config) SetLimit(limit int) *Config {
	if c.ExportFilter == nil {
		c.ExportFilter = NewExportFilter()
	}
	c.ExportFilter.Limit = limit
	return c
}

func (c *Config) SetCreatedAfter(t time.Time) *Config {
	if c.ExportFilter == nil {
		c.ExportFilter = NewExportFilter()
	}
	c.ExportFilter.CreatedAfter = &t
	return c
}

func (c *Config) SetCreatedBefore(t time.Time) *Config {
	if c.ExportFilter == nil {
		c.ExportFilter = NewExportFilter()
	}
	c.ExportFilter.CreatedBefore = &t
	return c
}

func (c *Config) SetUpdatedAfter(t time.Time) *Config {
	if c.ExportFilter == nil {
		c.ExportFilter = NewExportFilter()
	}
	c.ExportFilter.UpdatedAfter = &t
	return c
}

func (c *Config) SetUpdatedBefore(t time.Time) *Config {
	if c.ExportFilter == nil {
		c.ExportFilter = NewExportFilter()
	}
	c.ExportFilter.UpdatedBefore = &t
	return c
}

func (c *Config) WithFilename(path string) *Config {
	c.Filename = path
	return c
}

func SaveFingersToFile(filename string, data []*fingers.Finger) error {
	return saveYAMLToFile(filename, data)
}

func SaveTemplatesToFile(filename string, data []*templates.Template) error {
	return saveYAMLToFile(filename, data)
}

func saveYAMLToFile(filename string, payload interface{}) error {
	tmpPath := filename + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(payload); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to encode yaml: %w", err)
	}
	if err := encoder.Close(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close yaml encoder: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filename); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
