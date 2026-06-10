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
// Task - ScanTask
// ========================================

type ScanTask struct {
	Target string
}

func NewScanTask(target string) *ScanTask {
	return &ScanTask{Target: target}
}

func (t *ScanTask) Type() string { return "scan" }

func (t *ScanTask) Validate() error {
	if t.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}
	return nil
}

// ========================================
// Task - ScanDataTask
// ========================================

type ScanDataTask struct {
	Data     []byte
	FilePath string
}

func NewScanDataTask(data []byte, filePath string) *ScanDataTask {
	return &ScanDataTask{Data: data, FilePath: filePath}
}

func (t *ScanDataTask) Type() string { return "scan-data" }

func (t *ScanDataTask) Validate() error {
	if len(t.Data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}
	if t.FilePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	return nil
}
