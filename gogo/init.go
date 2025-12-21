package gogo

import (
	"fmt"

	"github.com/chainreactors/gogo/v2/pkg"
	sdk "github.com/chainreactors/sdk/pkg"
)

func init() {
	// 注册 gogo 引擎到全局注册表
	sdk.Register("gogo", func(config interface{}) (sdk.Engine, error) {
		var opt *pkg.RunnerOption

		if config == nil {
			opt = nil // NewGogoEngine 会使用默认配置
		} else {
			var ok bool
			opt, ok = config.(*pkg.RunnerOption)
			if !ok {
				return nil, fmt.Errorf("invalid config type for gogo engine, expected *pkg.RunnerOption, got %T", config)
			}
		}

		engine := NewGogoEngine(opt)
		if err := engine.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize gogo engine: %w", err)
		}

		return engine, nil
	})
}
