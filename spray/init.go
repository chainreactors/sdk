package spray

import (
	"fmt"

	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/spray/core"
)

func init() {
	// 注册 spray 引擎到全局注册表
	sdk.Register("spray", func(config interface{}) (sdk.Engine, error) {
		var opt *core.Option

		if config == nil {
			opt = nil // NewSprayEngine 会使用默认配置
		} else {
			var ok bool
			opt, ok = config.(*core.Option)
			if !ok {
				return nil, fmt.Errorf("invalid config type for spray engine, expected *core.Option, got %T", config)
			}
		}

		engine := NewSprayEngine(opt)
		if err := engine.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize spray engine: %w", err)
		}

		return engine, nil
	})
}
