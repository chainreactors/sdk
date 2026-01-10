package gogo

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainreactors/gogo/v2/core"
	"github.com/chainreactors/gogo/v2/engine"
	"github.com/chainreactors/gogo/v2/pkg"
	"github.com/chainreactors/logs"
	neutronTemplates "github.com/chainreactors/neutron/templates"
	"github.com/chainreactors/parsers"
	sdkfingers "github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/neutron"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/panjf2000/ants/v2"
)

// ========================================
// Engine 实现
// ========================================

// GogoEngine GoGo 引擎实现
type GogoEngine struct {
	inited        bool
	fingersEngine *sdkfingers.Engine // 可选的自定义 fingers 引擎
	neutronEngine *neutron.Engine    // 可选的 neutron 引擎
}

// NewGogoEngine 创建 GoGo 引擎
func NewGogoEngine(config *Config) *GogoEngine {
	if config == nil {
		config = NewConfig()
	}

	return &GogoEngine{
		inited:        false,
		fingersEngine: config.FingersEngine,
		neutronEngine: config.NeutronEngine,
	}
}

// buildTemplateMap 构建 template map（按 finger、id、tag 分类）
func buildTemplateMap(templates []*neutronTemplates.Template) map[string][]*neutronTemplates.Template {
	templateMap := make(map[string][]*neutronTemplates.Template)

	for _, template := range templates {
		// 按 fingers 归类
		for _, finger := range template.Fingers {
			key := toLowerKey(finger)
			templateMap[key] = append(templateMap[key], template)
		}

		// 按 id 归类
		if template.Id != "" {
			key := toLowerKey(template.Id)
			templateMap[key] = append(templateMap[key], template)
		}

		// 按 tags 归类
		for _, tag := range template.GetTags() {
			key := toLowerKey(tag)
			templateMap[key] = append(templateMap[key], template)
		}
	}

	return templateMap
}

func toLowerKey(s string) string {
	// 简单的 toLowerCase 实现
	return s // gogo 内部会处理大小写
}

// NewEngine 创建 GoGo 引擎
func NewEngine(config *Config) *GogoEngine {
	return NewGogoEngine(config)
}

// Init 初始化引擎（加载指纹库等）
func (e *GogoEngine) Init() error {
	if e.inited {
		return nil
	}

	if err := pkg.LoadPortConfig(""); err != nil {
		return fmt.Errorf("load port config failed: %v", err)
	}

	// 如果提供了自定义 fingers 引擎，直接使用
	if e.fingersEngine != nil {
		if _, err := e.fingersEngine.Load(context.Background()); err != nil {
			return fmt.Errorf("load fingers engine failed: %v", err)
		}
		fingerImpl, err := e.fingersEngine.GetFingersEngine()
		if err != nil {
			return fmt.Errorf("get fingers engine failed: %v", err)
		}
		if fingerImpl == nil {
			return fmt.Errorf("fingers engine is nil")
		}
		pkg.FingerEngine = fingerImpl
		logs.Log.Infof("using custom fingers engine: %d http fingers, %d socket fingers",
			len(fingerImpl.HTTPFingers), len(fingerImpl.SocketFingers))
	} else {
		// 否则使用默认加载方式
		if err := pkg.LoadFinger(nil); err != nil {
			return fmt.Errorf("load finger config failed: %v", err)
		}
	}

	// 如果提供了自定义 neutron 引擎，直接使用
	if e.neutronEngine != nil {
		templates := e.neutronEngine.Get()
		if templates == nil {
			loadedTemplates, err := e.neutronEngine.Load(context.Background())
			if err != nil {
				return fmt.Errorf("load neutron templates failed: %v", err)
			}
			templates = loadedTemplates
		}
		templateMap := buildTemplateMap(templates)
		pkg.TemplateMap = templateMap
		templateCount := 0
		for _, values := range templateMap {
			templateCount += len(values)
		}
		logs.Log.Infof("using custom neutron templates: %d templates in %d categories",
			templateCount, len(templateMap))
	} else {
		// 否则使用默认加载方式
		pkg.LoadNeutron("")
	}

	e.inited = true
	return nil
}

func (e *GogoEngine) Name() string {
	return "gogo"
}

func (e *GogoEngine) Execute(ctx sdk.Context, task sdk.Task) (<-chan sdk.Result, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	if err := task.Validate(); err != nil {
		return nil, err
	}

	switch t := task.(type) {
	case *ScanTask:
		return e.executeScan(ctx, t)
	case *WorkflowTask:
		return e.executeWorkflow(ctx, t)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", task.Type())
	}
}

func (e *GogoEngine) Close() error {
	return nil
}

// ========================================
// Context 实现
// ========================================

// Context GoGo 上下文
type Context struct {
	ctx     context.Context
	threads int
	opt     *pkg.RunnerOption
}

// NewContext 创建 GoGo 上下文
func NewContext() *Context {
	return &Context{
		ctx:     context.Background(),
		threads: 1000,
		opt:     pkg.DefaultRunnerOption,
	}
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) WithTimeout(timeout time.Duration) sdk.Context {
	ctx, _ := context.WithTimeout(c.ctx, timeout)
	return &Context{
		ctx:     ctx,
		threads: c.threads,
		opt:     c.opt,
	}
}

func (c *Context) WithCancel() (sdk.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	return &Context{
		ctx:     ctx,
		threads: c.threads,
		opt:     c.opt,
	}, cancel
}

// SetThreads 设置线程数
func (c *Context) SetThreads(threads int) *Context {
	if threads > 0 {
		c.threads = threads
	}
	return c
}

// SetOption 设置运行选项
func (c *Context) SetOption(opt *pkg.RunnerOption) *Context {
	c.opt = opt
	return c
}

// SetVersionLevel 设置指纹识别级别
func (c *Context) SetVersionLevel(level int) *Context {
	if c.opt == nil {
		c.opt = pkg.DefaultRunnerOption
	}
	c.opt.VersionLevel = level
	return c
}

// SetExploit 设置漏洞检测模式
func (c *Context) SetExploit(exploit string) *Context {
	if c.opt == nil {
		c.opt = pkg.DefaultRunnerOption
	}
	c.opt.Exploit = exploit
	return c
}

// SetDelay 设置超时时间（秒）
func (c *Context) SetDelay(delay int) *Context {
	if c.opt == nil {
		c.opt = pkg.DefaultRunnerOption
	}
	c.opt.Delay = delay
	return c
}

// ========================================
// Config 实现
// ========================================

// Config GoGo 配置
type Config struct {
	FingersEngine *sdkfingers.Engine
	NeutronEngine *neutron.Engine
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Validate() error {
	return nil
}

// WithFingersEngine 设置自定义 fingers 引擎
func (c *Config) WithFingersEngine(engine *sdkfingers.Engine) *Config {
	c.FingersEngine = engine
	return c
}

// WithNeutronEngine 设置自定义 neutron 引擎
func (c *Config) WithNeutronEngine(engine *neutron.Engine) *Config {
	c.NeutronEngine = engine
	return c
}

// ========================================
// Task 实现
// ========================================

// ScanTask 扫描任务
type ScanTask struct {
	IP    string
	Ports string
}

// NewScanTask 创建扫描任务
func NewScanTask(ip, ports string) *ScanTask {
	return &ScanTask{IP: ip, Ports: ports}
}

func (t *ScanTask) Type() string {
	return "scan"
}

func (t *ScanTask) Validate() error {
	if t.IP == "" {
		return fmt.Errorf("IP cannot be empty")
	}
	if t.Ports == "" {
		return fmt.Errorf("Ports cannot be empty")
	}
	return nil
}

// WorkflowTask 工作流任务
type WorkflowTask struct {
	Workflow *pkg.Workflow
}

// NewWorkflowTask 创建工作流任务
func NewWorkflowTask(workflow *pkg.Workflow) *WorkflowTask {
	return &WorkflowTask{Workflow: workflow}
}

func (t *WorkflowTask) Type() string {
	return "workflow"
}

func (t *WorkflowTask) Validate() error {
	if t.Workflow == nil {
		return fmt.Errorf("Workflow cannot be nil")
	}
	return nil
}

// ========================================
// Result 实现
// ========================================

// Result GoGo 扫描结果
type Result struct {
	success bool
	err     error
	data    *parsers.GOGOResult
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

// GOGOResult 获取原始结果（便捷方法）
func (r *Result) GOGOResult() *parsers.GOGOResult {
	return r.data
}

// ========================================
// 内部实现
// ========================================

func (e *GogoEngine) executeScan(ctx sdk.Context, task *ScanTask) (<-chan sdk.Result, error) {
	if ctx == nil {
		ctx = NewContext()
	}
	runCtx := ctx.(*Context)

	workflow := &pkg.Workflow{
		IP:    task.IP,
		Ports: task.Ports,
	}

	return e.workflowStream(runCtx.Context(), workflow, runCtx)
}

func (e *GogoEngine) executeWorkflow(ctx sdk.Context, task *WorkflowTask) (<-chan sdk.Result, error) {
	if ctx == nil {
		ctx = NewContext()
	}
	runCtx := ctx.(*Context)
	return e.workflowStream(runCtx.Context(), task.Workflow, runCtx)
}

func (e *GogoEngine) workflowStream(ctx context.Context, workflow *pkg.Workflow, runCtx *Context) (<-chan sdk.Result, error) {
	// 创建基础配置
	if runCtx.opt == nil {
		runCtx.opt = pkg.DefaultRunnerOption
	}
	baseConfig := pkg.NewDefaultConfig(runCtx.opt)
	preparedConfig := workflow.PrepareConfig(baseConfig)

	// 初始化配置
	initConfig, err := core.InitConfig(preparedConfig)
	if err != nil {
		return nil, fmt.Errorf("init config failed: %v", err)
	}

	// 设置线程数
	if runCtx.threads > 0 {
		initConfig.Threads = runCtx.threads
	}

	// 创建结果 channel
	resultCh := make(chan sdk.Result, 100)

	// 启动扫描 goroutine
	go func() {
		defer close(resultCh)
		defer preparedConfig.Close()

		var wg sync.WaitGroup
		var aliveCount int32

		// 创建扫描池
		scanPool, _ := ants.NewPoolWithFunc(initConfig.Threads, func(i interface{}) {
			defer wg.Done()

			// 检查 context 是否已取消
			select {
			case <-ctx.Done():
				return
			default:
			}

			ipPort := i.([]string)
			result := pkg.NewResult(ipPort[0], ipPort[1])

			// 调用扫描引擎
			engine.Dispatch(initConfig.RunnerOpt, result)

			if result.Open {
				atomic.AddInt32(&aliveCount, 1)
				// 发送结果到 channel
				select {
				case resultCh <- &Result{
					success: true,
					data:    result.GOGOResult,
				}:
				case <-ctx.Done():
					return
				default:
					logs.Log.Debugf("result channel full, dropping result for %s", result.GetTarget())
				}
			}
		})
		defer scanPool.Release()

		// 扫描目标
		for _, cidr := range initConfig.CIDRs {
			for ip := range cidr.Range() {
				// 检查 context 是否已取消
				select {
				case <-ctx.Done():
					logs.Log.Debug("workflow cancelled by context")
					wg.Wait()
					return
				default:
				}

				ipStr := ip.String()
				if ip.Ver == 6 {
					ipStr = "[" + ipStr + "]"
				}

				for _, port := range initConfig.PortList {
					wg.Add(1)
					_ = scanPool.Invoke([]string{ipStr, port})
				}
			}
		}

		wg.Wait()
		logs.Log.Debugf("workflow completed, found %d alive hosts", aliveCount)
	}()

	return resultCh, nil
}

// ========================================
// 便捷 API（保持原有使用习惯）
// ========================================

// ScanOne 单目标扫描
func (e *GogoEngine) ScanOne(ctx *Context, ip, port string) *parsers.GOGOResult {
	result := pkg.NewResult(ip, port)
	if ctx == nil {
		ctx = NewContext()
	}
	runCtx := ctx

	// 检查 context 是否已取消
	select {
	case <-runCtx.Context().Done():
		return result.GOGOResult
	default:
	}

	if !e.inited {
		e.Init()
	}

	engine.Dispatch(runCtx.opt, result)
	return result.GOGOResult
}

// Scan 批量端口扫描（同步）
func (e *GogoEngine) Scan(ctx *Context, ip, ports string) ([]*parsers.GOGOResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	if ctx == nil {
		ctx = NewContext()
	}

	task := NewScanTask(ip, ports)
	resultCh, err := e.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	var gogoResults []*parsers.GOGOResult
	for r := range resultCh {
		if r.Success() {
			gogoResults = append(gogoResults, r.(*Result).GOGOResult())
		}
	}

	return gogoResults, nil
}

// ScanStream 批量端口扫描（流式）
func (e *GogoEngine) ScanStream(ctx *Context, ip, ports string) (<-chan *parsers.GOGOResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	if ctx == nil {
		ctx = NewContext()
	}

	task := NewScanTask(ip, ports)
	resultCh, err := e.Execute(ctx, task)
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
func (e *GogoEngine) Workflow(ctx *Context, workflow *pkg.Workflow) ([]*parsers.GOGOResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	if ctx == nil {
		ctx = NewContext()
	}

	task := NewWorkflowTask(workflow)
	resultCh, err := e.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	var gogoResults []*parsers.GOGOResult
	for r := range resultCh {
		if r.Success() {
			gogoResults = append(gogoResults, r.(*Result).GOGOResult())
		}
	}

	return gogoResults, nil
}

// WorkflowStream 工作流扫描（流式）
func (e *GogoEngine) WorkflowStream(ctx *Context, workflow *pkg.Workflow) (<-chan *parsers.GOGOResult, error) {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil, err
		}
	}

	if ctx == nil {
		ctx = NewContext()
	}

	task := NewWorkflowTask(workflow)
	resultCh, err := e.Execute(ctx, task)
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
