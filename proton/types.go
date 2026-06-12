package proton

import (
	"context"
	"fmt"

	"github.com/chainreactors/sdk/pkg/types"
)

// ========================================
// Type aliases — 统一引用 types 包
// ========================================

type (
	Finding    = types.ProtonResult
	MatchEvent = types.ProtonMatchEvent
	ScanStats  = types.ProtonScanStats
	Rule       = types.ProtonRule
)

// ========================================
// Context
// ========================================

type Context struct {
	ctx          context.Context
	statsHandler func(types.Stats)
}

var _ types.Context = (*Context)(nil)

func NewContext() *Context {
	return &Context{
		ctx: context.Background(),
	}
}

func (c *Context) WithContext(ctx context.Context) *Context {
	return &Context{
		ctx:          ctx,
		statsHandler: c.statsHandler,
	}
}

func (c *Context) SetStatsHandler(handler func(types.Stats)) *Context {
	c.statsHandler = handler
	return c
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) emitStats(stats types.Stats) {
	if c != nil && c.statsHandler != nil {
		c.statsHandler(stats)
	}
}

// ========================================
// Task - ScanDataTask
// ========================================

type ScanDataTask struct {
	Data  []byte
	Label string
}

func NewScanDataTask(data []byte, label string) *ScanDataTask {
	return &ScanDataTask{Data: data, Label: label}
}

func (t *ScanDataTask) Type() string { return "scan-data" }

func (t *ScanDataTask) Validate() error {
	if len(t.Data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}
	if t.Label == "" {
		return fmt.Errorf("label cannot be empty")
	}
	return nil
}
