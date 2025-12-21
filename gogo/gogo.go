package gogo

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	fingersEngine "github.com/chainreactors/fingers/fingers"
	"github.com/chainreactors/gogo/v2/core"
	"github.com/chainreactors/gogo/v2/engine"
	"github.com/chainreactors/gogo/v2/pkg"
	"github.com/chainreactors/logs"
	neutronTemplates "github.com/chainreactors/neutron/templates"
	"github.com/chainreactors/parsers"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/panjf2000/ants/v2"
)

// ========================================
// Engine 实现
// ========================================

// GogoEngine GoGo 引擎实现
type GogoEngine struct {
	opt             *pkg.RunnerOption
	threads         int
	inited          bool
	fingersEngine   *fingersEngine.FingersEngine          // 可选的自定义 fingers 引擎
	neutronTemplates map[string][]*neutronTemplates.Template // 可选的 neutron templates map
}

// NewGogoEngine 创建 GoGo 引擎
func NewGogoEngine(opt *pkg.RunnerOption) *GogoEngine {
	if opt == nil {
		opt = pkg.DefaultRunnerOption
	}
	return &GogoEngine{
		opt:     opt,
		threads: 1000,
		inited:  false,
	}
}

// NewGogoEngineWithFingers 创建 GoGo 引擎并设置自定义 fingers 引擎
func NewGogoEngineWithFingers(opt *pkg.RunnerOption, fingersEngine *fingersEngine.FingersEngine) *GogoEngine {
	engine := NewGogoEngine(opt)
	engine.fingersEngine = fingersEngine
	return engine
}

// NewGogoEngineWithNeutron 创建 GoGo 引擎并设置 neutron templates
func NewGogoEngineWithNeutron(opt *pkg.RunnerOption, templates []*neutronTemplates.Template) *GogoEngine {
	engine := NewGogoEngine(opt)
	engine.neutronTemplates = buildTemplateMap(templates)
	return engine
}

// NewGogoEngineWithFingersAndNeutron 创建 GoGo 引擎并同时设置 fingers 和 neutron
func NewGogoEngineWithFingersAndNeutron(opt *pkg.RunnerOption, fingersEngine *fingersEngine.FingersEngine, templates []*neutronTemplates.Template) *GogoEngine {
	engine := NewGogoEngine(opt)
	engine.fingersEngine = fingersEngine
	engine.neutronTemplates = buildTemplateMap(templates)
	return engine
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

// NewEngine 创建 GoGo 引擎 (兼容旧 API)
func NewEngine(opt *pkg.RunnerOption) *GogoEngine {
	return NewGogoEngine(opt)
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
		pkg.FingerEngine = e.fingersEngine
		logs.Log.Infof("using custom fingers engine: %d http fingers, %d socket fingers",
			len(e.fingersEngine.HTTPFingers), len(e.fingersEngine.SocketFingers))
	} else {
		// 否则使用默认加载方式
		if err := pkg.LoadFinger(nil); err != nil {
			return fmt.Errorf("load finger config failed: %v", err)
		}
	}

	// 如果提供了自定义 neutron templates，直接使用
	if e.neutronTemplates != nil {
		pkg.TemplateMap = e.neutronTemplates
		templateCount := 0
		for _, templates := range e.neutronTemplates {
			templateCount += len(templates)
		}
		logs.Log.Infof("using custom neutron templates: %d templates in %d categories",
			templateCount, len(e.neutronTemplates))
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
	ctx    context.Context
	config *Config
}

// NewContext 创建 GoGo 上下文
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

// Config GoGo 配置
type Config struct {
	Threads int
	Opt     *pkg.RunnerOption
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		Threads: 1000,
		Opt:     pkg.DefaultRunnerOption,
	}
}

func (c *Config) Validate() error {
	if c.Threads <= 0 {
		return fmt.Errorf("threads must be positive")
	}
	return nil
}

// SetThreads 设置线程数
func (c *Config) SetThreads(threads int) *Config {
	c.Threads = threads
	return c
}

// SetOption 设置运行选项
func (c *Config) SetOption(opt *pkg.RunnerOption) *Config {
	c.Opt = opt
	return c
}

// SetVersionLevel 设置指纹识别级别
func (c *Config) SetVersionLevel(level int) *Config {
	c.Opt.VersionLevel = level
	return c
}

// SetExploit 设置漏洞检测模式
func (c *Config) SetExploit(exploit string) *Config {
	c.Opt.Exploit = exploit
	return c
}

// SetDelay 设置超时时间（秒）
func (c *Config) SetDelay(delay int) *Config {
	c.Opt.Delay = delay
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
	config := ctx.Config().(*Config)

	workflow := &pkg.Workflow{
		IP:    task.IP,
		Ports: task.Ports,
	}

	return e.workflowStream(ctx.Context(), workflow, config)
}

func (e *GogoEngine) executeWorkflow(ctx sdk.Context, task *WorkflowTask) (<-chan sdk.Result, error) {
	config := ctx.Config().(*Config)
	return e.workflowStream(ctx.Context(), task.Workflow, config)
}

func (e *GogoEngine) workflowStream(ctx context.Context, workflow *pkg.Workflow, config *Config) (<-chan sdk.Result, error) {
	// 创建基础配置
	baseConfig := pkg.NewDefaultConfig(config.Opt)
	preparedConfig := workflow.PrepareConfig(baseConfig)

	// 初始化配置
	initConfig, err := core.InitConfig(preparedConfig)
	if err != nil {
		return nil, fmt.Errorf("init config failed: %v", err)
	}

	// 设置线程数
	if config.Threads > 0 {
		initConfig.Threads = config.Threads
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
