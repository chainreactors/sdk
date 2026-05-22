package cyberhub

import (
	"fmt"
	"os"

	"github.com/chainreactors/fingers/fingers"
	"github.com/chainreactors/neutron/templates"
	"gopkg.in/yaml.v3"
)

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
