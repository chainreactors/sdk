package fingers

import (
	"fmt"
	"os"
	"path/filepath"

	fingersEngine "github.com/chainreactors/fingers/fingers"
	"gopkg.in/yaml.v3"
)

// AddFingers adds fingerprints to the current engine and rebuilds it.
func (e *Engine) AddFingers(fingers fingersEngine.Fingers) error {
	if len(fingers) == 0 {
		return fmt.Errorf("fingers cannot be empty")
	}
	if e.config == nil {
		e.config = NewConfig()
	}

	if e.engine == nil {
		engine, err := buildEngineFromFingers(e.config.FullFingers.Fingers(), e.config.FullFingers.Aliases())
		if err != nil {
			return err
		}
		e.aliases = e.config.FullFingers.Aliases()
		e.engine = engine
	}

	fingersEngineImpl, err := e.GetFingersEngine()
	if err != nil {
		return err
	}
	if fingersEngineImpl == nil {
		return fmt.Errorf("fingers engine is not initialized")
	}

	if err := fingersEngineImpl.Append(fingers); err != nil {
		return err
	}
	e.config.FullFingers = e.config.FullFingers.Merge(fingers, nil)
	return nil
}

// AddFingersFile loads fingerprints from a yaml file or directory and adds them.
func (e *Engine) AddFingersFile(path string) error {
	fingers, err := loadFingersFromPath(path)
	if err != nil {
		return err
	}
	return e.AddFingers(fingers)
}

func loadFingersFromPath(path string) (fingersEngine.Fingers, error) {
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

	var loaded fingersEngine.Fingers
	for _, yamlFile := range yamlFiles {
		file, err := os.Open(yamlFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", yamlFile, err)
		}

		var raw []*fingersEngine.Finger
		if err := yaml.NewDecoder(file).Decode(&raw); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to decode fingerprints: %w", err)
		}
		file.Close()

		loaded = append(loaded, raw...)
	}

	if len(loaded) == 0 {
		return nil, fmt.Errorf("no fingers loaded from %s", path)
	}

	return loaded, nil
}
