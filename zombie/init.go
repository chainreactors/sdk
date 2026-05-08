package zombie

import (
	"fmt"

	sdk "github.com/chainreactors/sdk/pkg"
)

func init() {
	sdk.Register("zombie", func(config interface{}) (sdk.Engine, error) {
		var cfg *Config
		if config == nil {
			cfg = NewConfig()
		} else {
			var ok bool
			cfg, ok = config.(*Config)
			if !ok {
				return nil, fmt.Errorf("invalid config type for zombie engine, expected *zombie.Config, got %T", config)
			}
		}

		engine := NewEngine(cfg)
		if err := engine.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize zombie engine: %w", err)
		}
		return engine, nil
	})
}
