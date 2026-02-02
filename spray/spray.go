package spray

import (
	"context"
	"fmt"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/parsers"
	sdkfingers "github.com/chainreactors/sdk/fingers"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/spray/core"
	"github.com/chainreactors/spray/pkg"
)

// ========================================
// Engine 实现
// ========================================

// SprayEngine Spray 引擎实现
type SprayEngine struct {
	inited        bool
	fingersEngine *sdkfingers.Engine // 可选的自定义 fingers 引擎
}

// NewSprayEngine 创建 Spray 引擎
func NewSprayEngine(config *Config) *SprayEngine {
	if config == nil {
		config = NewConfig()
	}

	return &SprayEngine{
		inited:        false,
		fingersEngine: config.FingersEngine,
	}
}

// NewEngine 创建 Spray 引擎
func NewEngine(config *Config) *SprayEngine {
	return NewSprayEngine(config)
}

// Init 初始化引擎（加载指纹库等）
func (e *SprayEngine) Init() error {
	if e.inited {
		return nil
	}

	if err := pkg.Load(); err != nil {
		return fmt.Errorf("load config failed: %v", err)
	}
	// 如果提供了自定义 fingers 引擎，直接使用
	if e.fingersEngine != nil {
		libEngine := e.fingersEngine.Get()
		if libEngine == nil {
			return fmt.Errorf("fingers engine is nil")
		}
		pkg.FingerEngine = libEngine
		logs.Log.Infof("using custom fingers engine: %s", libEngine.String())
	} else {
		// 尝试创建默认的 fingers 引擎
		defaultFingers, err := sdkfingers.NewEngine(nil)
		if err == nil && defaultFingers != nil {
			e.fingersEngine = defaultFingers
			libEngine := defaultFingers.Get()
			if libEngine != nil {
				pkg.FingerEngine = libEngine
				logs.Log.Debugf("using default fingers engine")
			}
		} else {
			// 如果创建失败，尝试使用内置指纹
			if err := pkg.LoadFingers(); err != nil {
				logs.Log.Debugf("load fingers failed, using built-in: %v", err)
			}
		}
	}
	// 提取 ActivePath (spray 需要)
	if pkg.FingerEngine != nil {
		if fingers := pkg.FingerEngine.Fingers(); fingers != nil {
			for _, f := range fingers.HTTPFingers {
				for _, rule := range f.Rules {
					if rule.SendDataStr != "" {
						pkg.ActivePath = append(pkg.ActivePath, rule.SendDataStr)
					}
				}
			}
		}
	}
	e.inited = true
	return nil
}

func (e *SprayEngine) Name() string {
	return "spray"
}

func (e *SprayEngine) Execute(ctx sdk.Context, task sdk.Task) (<-chan sdk.Result, error) {
	if err := task.Validate(); err != nil {
		return nil, err
	}

	switch t := task.(type) {
	case *CheckTask:
		return e.executeCheck(ctx, t)
	case *BruteTask:
		return e.executeBrute(ctx, t)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", task.Type())
	}
}

func (e *SprayEngine) Close() error {
	return nil
}

// ========================================
// Result 实现
// ========================================

// Result Spray 检测结果
type Result struct {
	success bool
	err     error
	data    *parsers.SprayResult
}

func (r *Result) Success() bool {
	return r.success
}

func (r *Result) Error() error {
	return r.err
}

func (r *Result) Data() interface{} {
	return r.data
}

// SprayResult 获取原始结果（便捷方法）
func (r *Result) SprayResult() *parsers.SprayResult {
	return r.data
}

func (e *SprayEngine) handler(ctx context.Context, runner *core.Runner, ch chan sdk.Result) {
	// 启动结果处理 goroutine - 处理 OutputCh
	go func() {
		for bl := range runner.OutputCh {
			select {
			case ch <- &Result{
				success: bl.IsValid,
				data:    bl.SprayResult,
			}:
			case <-ctx.Done():
				return
			}
			runner.OutWg.Done()
		}
	}()

	// 启动结果处理 goroutine - 处理 FuzzyCh
	go func() {
		for bl := range runner.FuzzyCh {
			select {
			case ch <- &Result{
				success: bl.IsValid,
				data:    bl.SprayResult,
			}:
			case <-ctx.Done():
				return
			}
			runner.OutWg.Done()
		}
	}()
}

func (e *SprayEngine) executeCheck(ctx sdk.Context, task *CheckTask) (<-chan sdk.Result, error) {
	// 克隆配置
	opt := *ctx.(*Context).opt
	opt.URL = task.URLs

	// 创建 Runner
	runner, err := opt.NewRunner()
	if err != nil {
		return nil, fmt.Errorf("create runner failed: %v", err)
	}
	ch := make(chan sdk.Result)
	// 启动检测 goroutine
	go func() {
		defer e.closeRunner(runner)
		defer close(ch)
		e.handler(ctx.Context(), runner, ch)

		runner.RunWithCheck(ctx.Context())
	}()

	return ch, nil
}

func (e *SprayEngine) executeBrute(ctx sdk.Context, task *BruteTask) (<-chan sdk.Result, error) {
	// 克隆配置
	opt := *ctx.(*Context).opt
	opt.URL = []string{task.BaseURL}

	// 创建 Runner
	runner, err := opt.NewRunner()
	if err != nil {
		return nil, fmt.Errorf("create runner failed: %v", err)
	}

	runner.Wordlist = task.Wordlist
	runner.Total = len(task.Wordlist)
	runner.IsCheck = false

	resultCh := make(chan sdk.Result)
	// 启动检测 goroutine
	go func() {
		defer e.closeRunner(runner)
		defer close(resultCh)

		e.handler(ctx.Context(), runner, resultCh)

		runner.RunWithBrute(ctx.Context())
	}()

	return resultCh, nil
}

// ========================================
// 便捷 API（保持原有使用习惯）
// ========================================

// Check URL 批量检测（同步）
func (e *SprayEngine) Check(ctx *Context, urls []string) ([]*parsers.SprayResult, error) {
	task := NewCheckTask(urls)
	resultCh, err := e.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	var sprayResults []*parsers.SprayResult
	for r := range resultCh {
		// 返回所有结果，无论是否成功/有效
		// URL存活检测需要看到所有URL的状态，而不仅仅是有效的
		if result := r.(*Result).SprayResult(); result != nil {
			sprayResults = append(sprayResults, result)
		}
	}

	return sprayResults, nil
}

// CheckStream URL 批量检测（流式）
func (e *SprayEngine) CheckStream(ctx *Context, urls []string) (<-chan *parsers.SprayResult, error) {
	task := NewCheckTask(urls)
	resultCh, err := e.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	// 转换为 SprayResult channel
	sprayResultCh := make(chan *parsers.SprayResult, 1)
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
func (e *SprayEngine) Brute(ctx *Context, baseURL string, wordlist []string) ([]*parsers.SprayResult, error) {
	task := NewBruteTask(baseURL, wordlist)
	resultCh, err := e.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	var sprayResults []*parsers.SprayResult
	for r := range resultCh {
		// 返回所有结果，无论是否成功/有效
		if result := r.(*Result).SprayResult(); result != nil {
			sprayResults = append(sprayResults, result)
		}
	}

	return sprayResults, nil
}

// BruteStream 暴力破解（流式）
func (e *SprayEngine) BruteStream(ctx *Context, baseURL string, wordlist []string) (<-chan *parsers.SprayResult, error) {
	task := NewBruteTask(baseURL, wordlist)
	resultCh, err := e.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	// 转换为 SprayResult channel
	sprayResultCh := make(chan *parsers.SprayResult)
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

func (e *SprayEngine) closeRunner(runner *core.Runner) {
	if runner.OutputFile != nil {
		runner.OutputFile.Close()
	}
	if runner.DumpFile != nil {
		runner.DumpFile.Close()
	}
	if runner.StatFile != nil {
		runner.StatFile.Close()
	}
	if runner.Progress != nil {
		runner.Progress.Wait()
	}
}
