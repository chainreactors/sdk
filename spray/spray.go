package spray

import (
	"context"
	"fmt"
	"time"

	fingersLib "github.com/chainreactors/fingers"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/parsers"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/spray/core"
	"github.com/chainreactors/spray/pkg"
)

// ========================================
// Engine 实现
// ========================================

// SprayEngine Spray 引擎实现
type SprayEngine struct {
	opt           *core.Option
	inited        bool
	fingersEngine *fingersLib.Engine // 可选的自定义 fingers 引擎
}

// NewSprayEngine 创建 Spray 引擎
func NewSprayEngine(opt *core.Option) *SprayEngine {
	if opt == nil {
		opt = DefaultConfig()
	}
	return &SprayEngine{
		opt:    opt,
		inited: false,
	}
}

// NewSprayEngineWithFingers 创建 Spray 引擎并设置自定义 fingers 引擎
func NewSprayEngineWithFingers(opt *core.Option, fingersEngine *fingersLib.Engine) *SprayEngine {
	engine := NewSprayEngine(opt)
	engine.fingersEngine = fingersEngine
	return engine
}

// NewEngine 创建 Spray 引擎 (兼容旧 API)
func NewEngine(opt *core.Option) *SprayEngine {
	return NewSprayEngine(opt)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *core.Option {
	opt := &core.Option{}
	opt.Method = "GET"
	opt.MaxBodyLength = 100
	opt.RandomUserAgent = false
	opt.BlackStatus = "400,410"
	opt.WhiteStatus = "200"
	opt.FuzzyStatus = "500,501,502,503,301,302,404"
	opt.UniqueStatus = "403,200,404"
	opt.CheckPeriod = 200
	opt.ErrPeriod = 10
	opt.BreakThreshold = 20
	opt.Recursive = "current.IsDir()"
	opt.Depth = 0
	opt.Index = "/"
	opt.Random = ""
	opt.RetryCount = 0
	opt.SimhashDistance = 8
	opt.Mod = "path"
	opt.Client = "auto"
	opt.Timeout = 5
	opt.Threads = 20
	opt.PoolSize = 1
	opt.Deadline = 999999
	opt.Quiet = true
	opt.NoBar = true
	opt.NoStat = true
	opt.NoColor = false
	opt.Json = false
	opt.FileOutput = "json"
	opt.Advance = false
	opt.Finger = false
	opt.CrawlPlugin = false
	opt.BakPlugin = false
	opt.FuzzuliPlugin = false
	opt.CommonPlugin = false
	opt.ActivePlugin = false
	opt.ReconPlugin = false
	opt.CrawlDepth = 3
	opt.AppendDepth = 2
	opt.FingerEngines = "all"
	return opt
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
		pkg.FingerEngine = e.fingersEngine
		logs.Log.Infof("using custom fingers engine: %s", e.fingersEngine.String())

		// 提取 ActivePath (spray 需要)
		for _, f := range e.fingersEngine.Fingers().HTTPFingers {
			for _, rule := range f.Rules {
				if rule.SendDataStr != "" {
					pkg.ActivePath = append(pkg.ActivePath, rule.SendDataStr)
				}
			}
		}

		// FingerPrintHub 可能为 nil
		if hub := e.fingersEngine.FingerPrintHub(); hub != nil {
			for _, f := range hub.FingerPrints {
				if f.Path != "/" {
					pkg.ActivePath = append(pkg.ActivePath, f.Path)
				}
			}
		}
	} else {
		// 否则使用默认加载方式
		if err := pkg.LoadFingers(); err != nil {
			return fmt.Errorf("load fingers failed: %v", err)
		}
	}

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
// Context 实现
// ========================================

// Context Spray 上下文
type Context struct {
	ctx    context.Context
	config *Config
}

// NewContext 创建 Spray 上下文
func NewContext() *Context {
	return &Context{
		ctx:    context.Background(),
		config: NewConfig(),
	}
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) Config() sdk.Config {
	return c.config
}

func (c *Context) WithConfig(config sdk.Config) sdk.Context {
	return &Context{
		ctx:    c.ctx,
		config: config.(*Config),
	}
}

func (c *Context) WithTimeout(timeout time.Duration) sdk.Context {
	ctx, _ := context.WithTimeout(c.ctx, timeout)
	return &Context{
		ctx:    ctx,
		config: c.config,
	}
}

func (c *Context) WithCancel() (sdk.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	return &Context{
		ctx:    ctx,
		config: c.config,
	}, cancel
}

// ========================================
// Config 实现
// ========================================

// Config Spray 配置
type Config struct {
	Opt *core.Option
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		Opt: DefaultConfig(),
	}
}

func (c *Config) Validate() error {
	if c.Opt.Threads <= 0 {
		return fmt.Errorf("threads must be positive")
	}
	return nil
}

// SetThreads 设置线程数
func (c *Config) SetThreads(threads int) *Config {
	c.Opt.Threads = threads
	return c
}

// SetTimeout 设置超时时间（秒）
func (c *Config) SetTimeout(timeout int) *Config {
	c.Opt.Timeout = timeout
	return c
}

// SetMethod 设置 HTTP 方法
func (c *Config) SetMethod(method string) *Config {
	c.Opt.Method = method
	return c
}

// SetHeaders 设置自定义请求头
func (c *Config) SetHeaders(headers []string) *Config {
	c.Opt.Headers = headers
	return c
}

// SetFilter 设置过滤规则
func (c *Config) SetFilter(filter string) *Config {
	c.Opt.Filter = filter
	return c
}

// SetMatch 设置匹配规则
func (c *Config) SetMatch(match string) *Config {
	c.Opt.Match = match
	return c
}

// ========================================
// Task 实现
// ========================================

// CheckTask URL 检测任务
type CheckTask struct {
	URLs []string
}

// NewCheckTask 创建 URL 检测任务
func NewCheckTask(urls []string) *CheckTask {
	return &CheckTask{URLs: urls}
}

func (t *CheckTask) Type() string {
	return "check"
}

func (t *CheckTask) Validate() error {
	if len(t.URLs) == 0 {
		return fmt.Errorf("URLs cannot be empty")
	}
	return nil
}

// BruteTask 暴力破解任务
type BruteTask struct {
	BaseURL  string
	Wordlist []string
}

// NewBruteTask 创建暴力破解任务
func NewBruteTask(baseURL string, wordlist []string) *BruteTask {
	return &BruteTask{
		BaseURL:  baseURL,
		Wordlist: wordlist,
	}
}

func (t *BruteTask) Type() string {
	return "brute"
}

func (t *BruteTask) Validate() error {
	if t.BaseURL == "" {
		return fmt.Errorf("BaseURL cannot be empty")
	}
	if len(t.Wordlist) == 0 {
		return fmt.Errorf("Wordlist cannot be empty")
	}
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
	config := ctx.Config().(*Config)

	// 克隆配置
	opt := *config.Opt
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
	config := ctx.Config().(*Config)

	// 克隆配置
	opt := *config.Opt
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
