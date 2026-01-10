package neutron

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chainreactors/neutron/templates"
	"gopkg.in/yaml.v3"
)

// AddPocs adds templates to the current engine and compiles them.
func (e *Engine) AddPocs(pocs []*templates.Template) error {
	if len(pocs) == 0 {
		return fmt.Errorf("pocs cannot be empty")
	}
	if e.config == nil {
		e.config = NewConfig()
	}

	if e.templates == nil && len(e.config.Templates) > 0 {
		e.templates = e.compileTemplates(e.config.Templates)
	}

	compiled := e.compileTemplates(pocs)
	if len(compiled) == 0 {
		return fmt.Errorf("no templates compiled")
	}

	e.templates = append(e.templates, compiled...)
	e.config.Templates = append(e.config.Templates, pocs...)
	return nil
}

// AddPocsFile loads templates from a yaml file or directory and adds them.
func (e *Engine) AddPocsFile(path string) error {
	pocs, err := loadTemplatesFromPath(path)
	if err != nil {
		return err
	}
	return e.AddPocs(pocs)
}

func loadTemplatesFromPath(path string) ([]*templates.Template, error) {
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

	return loaded, nil
}
