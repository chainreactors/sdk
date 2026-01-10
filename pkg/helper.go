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
	ctx context.Context
}

// NewBaseContext 创建基础 Context
func NewBaseContext() *BaseContext {
	return &BaseContext{
		ctx: context.Background(),
	}
}

func (c *BaseContext) Context() context.Context {
	return c.ctx
}

func (c *BaseContext) WithTimeout(timeout time.Duration) Context {
	ctx, _ := context.WithTimeout(c.ctx, timeout)
	return &BaseContext{
		ctx: ctx,
	}
}

func (c *BaseContext) WithCancel() (Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	return &BaseContext{
		ctx: ctx,
	}, cancel
}
