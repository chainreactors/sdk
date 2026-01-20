package spray

import (
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

// DefaultConfig 返回默认配置
// 注意: 这个函数已弃用，请使用 NewDefaultRunnerConfig
func DefaultConfig() *core.Option {
	opt, _ := core.NewDefaultRunnerConfig()
	return opt
}

// NewDefaultRunnerConfig 创建并返回一个带有默认值且已初始化的 Runner 配置
// 这个函数直接调用 spray/core 中的 NewDefaultRunnerConfig
func NewDefaultRunnerConfig() (*core.Option, error) {
	return core.NewDefaultRunnerConfig()
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

		// 提取 ActivePath (spray 需要)
		if fingers := libEngine.Fingers(); fingers != nil {
			for _, f := range fingers.HTTPFingers {
				for _, rule := range f.Rules {
					if rule.SendDataStr != "" {
						pkg.ActivePath = append(pkg.ActivePath, rule.SendDataStr)
					}
				}
			}
		}
		// 注意: FingerPrintHub 的 FingerPrints 字段已移除，跳过相关处理
	}
	// 注意: 不调用 pkg.LoadFingers() 因为它使用了已弃用的 API

	e.inited = true
	return nil
}

func (e *SprayEngine) Name() string {
	return "spray"
}

func (e *SprayEngine) Execute(ctx sdk.Context, task sdk.Task) (<-chan sdk.Result, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

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

// ========================================
// 内部实现
// ========================================

func (e *SprayEngine) executeCheck(ctx sdk.Context, task *CheckTask) (<-chan sdk.Result, error) {
	if ctx == nil {
		ctx = NewContext()
	}
	runCtx := ctx.(*Context)
	if runCtx.opt == nil {
		runCtx.opt = DefaultConfig()
	}

	// 克隆配置
	opt := *runCtx.opt
	opt.URL = task.URLs

	// 准备配置
	if err := opt.Prepare(); err != nil {
		return nil, fmt.Errorf("prepare config failed: %v", err)
	}

	// 创建 Runner
	runner, err := opt.NewRunner()
	if err != nil {
		return nil, fmt.Errorf("create runner failed: %v", err)
	}

	runner.IsCheck = true
	// 禁用内置的 OutputHandler，让 SDK 完全控制输出处理
	runner.DisableOutputHandler = true

	// 创建结果 channel
	resultCh := make(chan sdk.Result, 100)

	// 启动检测 goroutine
	go func() {
		defer close(resultCh)
		defer e.closeRunner(runner)

		// 启动结果处理 goroutine
		go func() {
			for bl := range runner.OutputCh {
				select {
				case resultCh <- &Result{
					success: bl.IsValid,
					data:    bl.SprayResult,
				}:
				case <-ctx.Context().Done():
					return
				}
				runner.OutWg.Done()
			}
		}()

		// 运行检测
		if err := runner.Prepare(ctx.Context()); err != nil {
			logs.Log.Errorf("runner prepare failed: %v", err)
		}
	}()

	return resultCh, nil
}

func (e *SprayEngine) executeBrute(ctx sdk.Context, task *BruteTask) (<-chan sdk.Result, error) {
	if ctx == nil {
		ctx = NewContext()
	}
	runCtx := ctx.(*Context)
	if runCtx.opt == nil {
		runCtx.opt = DefaultConfig()
	}

	// 克隆配置
	opt := *runCtx.opt
	opt.URL = []string{task.BaseURL}

	// 准备配置
	if err := opt.Prepare(); err != nil {
		return nil, fmt.Errorf("prepare config failed: %v", err)
	}

	// 创建 Runner
	runner, err := opt.NewRunner()
	if err != nil {
		return nil, fmt.Errorf("create runner failed: %v", err)
	}

	runner.Wordlist = task.Wordlist
	runner.Total = len(task.Wordlist)
	runner.IsCheck = false
	// 禁用内置的 OutputHandler，让 SDK 完全控制输出处理
	runner.DisableOutputHandler = true

	// 创建结果 channel
	resultCh := make(chan sdk.Result, 100)

	// 启动暴力破解 goroutine
	go func() {
		defer close(resultCh)
		defer e.closeRunner(runner)

		// 启动结果处理 goroutine
		go func() {
			for bl := range runner.OutputCh {
				select {
				case resultCh <- &Result{
					success: bl.IsValid,
					data:    bl.SprayResult,
				}:
				case <-ctx.Context().Done():
					return
				}
				runner.OutWg.Done()
			}
		}()

		// 运行暴力破解
		if err := runner.Prepare(ctx.Context()); err != nil {
			logs.Log.Errorf("runner prepare failed: %v", err)
		}
	}()

	return resultCh, nil
}

// ========================================
// 便捷 API（保持原有使用习惯）
// ========================================

// Check URL 批量检测（同步）
func (e *SprayEngine) Check(ctx *Context, urls []string) ([]*parsers.SprayResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	if ctx == nil {
		ctx = NewContext()
	}

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
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	if ctx == nil {
		ctx = NewContext()
	}

	task := NewCheckTask(urls)
	resultCh, err := e.Execute(ctx, task)
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
func (e *SprayEngine) Brute(ctx *Context, baseURL string, wordlist []string) ([]*parsers.SprayResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	if ctx == nil {
		ctx = NewContext()
	}

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
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	if ctx == nil {
		ctx = NewContext()
	}

	task := NewBruteTask(baseURL, wordlist)
	resultCh, err := e.Execute(ctx, task)
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
