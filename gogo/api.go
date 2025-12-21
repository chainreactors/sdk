package gogo

import (
	"context"
	"time"

	"github.com/chainreactors/gogo/v2/engine"
	"github.com/chainreactors/gogo/v2/pkg"
	"github.com/chainreactors/parsers"
	sdk "github.com/chainreactors/sdk/pkg"
)

// ========================================
// 便捷 API（保持原有使用习惯）
// ========================================

// ScanOne 单目标扫描
func (e *GogoEngine) ScanOne(ctx context.Context, ip, port string) *parsers.GOGOResult {
	result := pkg.NewResult(ip, port)

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return result.GOGOResult
	default:
	}

	if !e.inited {
		e.Init()
	}

	engine.Dispatch(e.opt, result)
	return result.GOGOResult
}

// Scan 批量端口扫描（同步）
func (e *GogoEngine) Scan(ctx context.Context, ip, ports string) ([]*parsers.GOGOResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	sdkCtx := NewContext().WithConfig(NewConfig().SetThreads(e.threads).SetOption(e.opt))
	wrappedCtx := &contextWrapper{ctx: ctx, sdkCtx: sdkCtx}

	task := NewScanTask(ip, ports)
	results, err := sdk.ExecuteSync(e, wrappedCtx, task)
	if err != nil {
		return nil, err
	}

	gogoResults := make([]*parsers.GOGOResult, 0, len(results))
	for _, r := range results {
		if r.Success() {
			gogoResults = append(gogoResults, r.(*Result).GOGOResult())
		}
	}

	return gogoResults, nil
}

// ScanStream 批量端口扫描（流式）
func (e *GogoEngine) ScanStream(ctx context.Context, ip, ports string) (<-chan *parsers.GOGOResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	sdkCtx := NewContext().WithConfig(NewConfig().SetThreads(e.threads).SetOption(e.opt))
	wrappedCtx := &contextWrapper{ctx: ctx, sdkCtx: sdkCtx}

	task := NewScanTask(ip, ports)
	resultCh, err := e.Execute(wrappedCtx, task)
	if err != nil {
		return nil, err
	}

	// 转换为 GOGOResult channel
	gogoResultCh := make(chan *parsers.GOGOResult, 100)
	go func() {
		defer close(gogoResultCh)
		for result := range resultCh {
			if result.Success() {
				gogoResultCh <- result.(*Result).GOGOResult()
			}
		}
	}()

	return gogoResultCh, nil
}

// Workflow 工作流扫描（同步）
func (e *GogoEngine) Workflow(ctx context.Context, workflow *pkg.Workflow) ([]*parsers.GOGOResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	sdkCtx := NewContext().WithConfig(NewConfig().SetThreads(e.threads).SetOption(e.opt))
	wrappedCtx := &contextWrapper{ctx: ctx, sdkCtx: sdkCtx}

	task := NewWorkflowTask(workflow)
	results, err := sdk.ExecuteSync(e, wrappedCtx, task)
	if err != nil {
		return nil, err
	}

	gogoResults := make([]*parsers.GOGOResult, 0, len(results))
	for _, r := range results {
		if r.Success() {
			gogoResults = append(gogoResults, r.(*Result).GOGOResult())
		}
	}

	return gogoResults, nil
}

// WorkflowStream 工作流扫描（流式）
func (e *GogoEngine) WorkflowStream(ctx context.Context, workflow *pkg.Workflow) (<-chan *parsers.GOGOResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	sdkCtx := NewContext().WithConfig(NewConfig().SetThreads(e.threads).SetOption(e.opt))
	wrappedCtx := &contextWrapper{ctx: ctx, sdkCtx: sdkCtx}

	task := NewWorkflowTask(workflow)
	resultCh, err := e.Execute(wrappedCtx, task)
	if err != nil {
		return nil, err
	}

	// 转换为 GOGOResult channel
	gogoResultCh := make(chan *parsers.GOGOResult, 100)
	go func() {
		defer close(gogoResultCh)
		for result := range resultCh {
			if result.Success() {
				gogoResultCh <- result.(*Result).GOGOResult()
			}
		}
	}()

	return gogoResultCh, nil
}

// SetThreads 设置线程数
func (e *GogoEngine) SetThreads(threads int) {
	e.threads = threads
}

// ========================================
// 辅助类型
// ========================================

// contextWrapper 包装标准 context 为 sdk.Context
type contextWrapper struct {
	ctx    context.Context
	sdkCtx sdk.Context
}

func (w *contextWrapper) Context() context.Context {
	return w.ctx
}

func (w *contextWrapper) Config() sdk.Config {
	return w.sdkCtx.Config()
}

func (w *contextWrapper) WithConfig(config sdk.Config) sdk.Context {
	return &contextWrapper{
		ctx:    w.ctx,
		sdkCtx: w.sdkCtx.WithConfig(config),
	}
}

func (w *contextWrapper) WithTimeout(timeout time.Duration) sdk.Context {
	ctx, _ := context.WithTimeout(w.ctx, timeout)
	return &contextWrapper{
		ctx:    ctx,
		sdkCtx: w.sdkCtx,
	}
}

func (w *contextWrapper) WithCancel() (sdk.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(w.ctx)
	return &contextWrapper{
		ctx:    ctx,
		sdkCtx: w.sdkCtx,
	}, cancel
}
