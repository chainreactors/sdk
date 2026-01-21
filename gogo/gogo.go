package gogo

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

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
		fingerImpl, err := e.fingersEngine.GetFingersEngine()
		if fingerImpl != nil && err == nil {
			pkg.FingerEngine = fingerImpl
		}
	} else {
		// 否则使用默认加载方式，但允许失败
		if err := pkg.LoadFinger(nil); err != nil {
			return err
		}
	}

	// 如果提供了自定义 neutron 引擎，直接使用
	if e.neutronEngine != nil {
		templates := e.neutronEngine.Get()
		if len(templates) == 0 {
			return fmt.Errorf("neutron templates are empty")
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

	var runCtx *Context
	if ctx == nil {
		runCtx = NewContext()
	} else {
		var ok bool
		runCtx, ok = ctx.(*Context)
		if !ok {
			return nil, fmt.Errorf("unsupported context type: %T", ctx)
		}
	}

	switch t := task.(type) {
	case *ScanTask:
		return e.executeScan(runCtx, t)
	case *WorkflowTask:
		return e.executeWorkflow(runCtx, t)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", task.Type())
	}
}

func (e *GogoEngine) Close() error {
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

func (e *GogoEngine) executeScan(ctx *Context, task *ScanTask) (<-chan sdk.Result, error) {
	if ctx == nil {
		ctx = NewContext()
	}
	runCtx := ctx

	workflow := &pkg.Workflow{
		IP:    task.IP,
		Ports: task.Ports,
	}

	return e.workflowStream(runCtx.Context(), workflow, runCtx)
}

func (e *GogoEngine) executeWorkflow(ctx *Context, task *WorkflowTask) (<-chan sdk.Result, error) {
	if ctx == nil {
		ctx = NewContext()
	}
	runCtx := ctx
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
