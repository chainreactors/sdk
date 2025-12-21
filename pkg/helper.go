package sdk

import (
	"context"
	"time"
)

// ========================================
// 基础 Context 实现
// ========================================

// BaseContext 提供 Context 接口的基础实现
// 各引擎可以组合使用或自定义实现
type BaseContext struct {
	ctx    context.Context
	config Config
}

// NewBaseContext 创建基础 Context
func NewBaseContext(config Config) *BaseContext {
	return &BaseContext{
		ctx:    context.Background(),
		config: config,
	}
}

func (c *BaseContext) Context() context.Context {
	return c.ctx
}

func (c *BaseContext) Config() Config {
	return c.config
}

func (c *BaseContext) WithConfig(config Config) Context {
	return &BaseContext{
		ctx:    c.ctx,
		config: config,
	}
}

func (c *BaseContext) WithTimeout(timeout time.Duration) Context {
	ctx, _ := context.WithTimeout(c.ctx, timeout)
	return &BaseContext{
		ctx:    ctx,
		config: c.config,
	}
}

func (c *BaseContext) WithCancel() (Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	return &BaseContext{
		ctx:    ctx,
		config: c.config,
	}, cancel
}

// ========================================
// 辅助函数
// ========================================

// ExecuteSync 同步执行任务，返回所有结果
// 这是一个通用的辅助函数，可用于所有引擎
func ExecuteSync(engine Engine, ctx Context, task Task) ([]Result, error) {
	resultCh, err := engine.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	var results []Result
	for result := range resultCh {
		results = append(results, result)
	}

	return results, nil
}
