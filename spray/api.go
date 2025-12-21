package spray

import (
	"context"
	"time"

	"github.com/chainreactors/parsers"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/spray/core"
)

// ========================================
// 便捷 API（保持原有使用习惯）
// ========================================

// Check URL 批量检测（同步）
func (e *SprayEngine) Check(ctx context.Context, urls []string) ([]*parsers.SprayResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	sdkCtx := NewContext().WithConfig(NewConfig().SetThreads(e.opt.Threads))
	wrappedCtx := &contextWrapper{ctx: ctx, sdkCtx: sdkCtx}

	task := NewCheckTask(urls)
	results, err := sdk.ExecuteSync(e, wrappedCtx, task)
	if err != nil {
		return nil, err
	}

	sprayResults := make([]*parsers.SprayResult, 0, len(results))
	for _, r := range results {
		if r.Success() {
			sprayResults = append(sprayResults, r.(*Result).SprayResult())
		}
	}

	return sprayResults, nil
}

// CheckStream URL 批量检测（流式）
func (e *SprayEngine) CheckStream(ctx context.Context, urls []string) (<-chan *parsers.SprayResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	sdkCtx := NewContext().WithConfig(NewConfig().SetThreads(e.opt.Threads))
	wrappedCtx := &contextWrapper{ctx: ctx, sdkCtx: sdkCtx}

	task := NewCheckTask(urls)
	resultCh, err := e.Execute(wrappedCtx, task)
	if err != nil {
		return nil, err
	}

	// 转换为 SprayResult channel
	sprayResultCh := make(chan *parsers.SprayResult, 100)
	go func() {
		defer close(sprayResultCh)
		for result := range resultCh {
			if result.Success() {
				sprayResultCh <- result.(*Result).SprayResult()
			}
		}
	}()

	return sprayResultCh, nil
}

// Brute 暴力破解（同步）
func (e *SprayEngine) Brute(ctx context.Context, baseURL string, wordlist []string) ([]*parsers.SprayResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	sdkCtx := NewContext().WithConfig(NewConfig().SetThreads(e.opt.Threads))
	wrappedCtx := &contextWrapper{ctx: ctx, sdkCtx: sdkCtx}

	task := NewBruteTask(baseURL, wordlist)
	results, err := sdk.ExecuteSync(e, wrappedCtx, task)
	if err != nil {
		return nil, err
	}

	sprayResults := make([]*parsers.SprayResult, 0, len(results))
	for _, r := range results {
		if r.Success() {
			sprayResults = append(sprayResults, r.(*Result).SprayResult())
		}
	}

	return sprayResults, nil
}

// BruteStream 暴力破解（流式）
func (e *SprayEngine) BruteStream(ctx context.Context, baseURL string, wordlist []string) (<-chan *parsers.SprayResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	sdkCtx := NewContext().WithConfig(NewConfig().SetThreads(e.opt.Threads))
	wrappedCtx := &contextWrapper{ctx: ctx, sdkCtx: sdkCtx}

	task := NewBruteTask(baseURL, wordlist)
	resultCh, err := e.Execute(wrappedCtx, task)
	if err != nil {
		return nil, err
	}

	// 转换为 SprayResult channel
	sprayResultCh := make(chan *parsers.SprayResult, 100)
	go func() {
		defer close(sprayResultCh)
		for result := range resultCh {
			if result.Success() {
				sprayResultCh <- result.(*Result).SprayResult()
			}
		}
	}()

	return sprayResultCh, nil
}

// SetThreads 设置线程数
func (e *SprayEngine) SetThreads(threads int) {
	e.opt.Threads = threads
}

// SetTimeout 设置超时时间（秒）
func (e *SprayEngine) SetTimeout(timeout int) {
	e.opt.Timeout = timeout
}

// SetOption 设置选项
func (e *SprayEngine) SetOption(opt *core.Option) {
	e.opt = opt
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
