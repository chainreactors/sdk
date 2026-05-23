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
	sdkfingers "github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/neutron"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/sdk/pkg/association"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"github.com/chainreactors/sdk/pkg/types"
	"github.com/panjf2000/ants/v2"
)

// ========================================
// Engine 实现
// ========================================

// GogoEngine GoGo 引擎实现
type GogoEngine struct {
	mu               sync.Mutex
	inited           bool
	provider         *cyberhub.Provider
	fingersEngine    *sdkfingers.Engine // 可选的自定义 fingers 引擎
	neutronEngine    *neutron.Engine    // 可选的 neutron 引擎
	index            *association.Index
	resourceProvider func(string) []byte
	capacity         *sdk.Capacity
}

// NewGogoEngine 创建 GoGo 引擎
func NewGogoEngine(config *Config) *GogoEngine {
	if config == nil {
		config = NewConfig()
	}

	e := &GogoEngine{
		inited:           false,
		provider:         config.Provider,
		fingersEngine:    config.FingersEngine,
		neutronEngine:    config.NeutronEngine,
		index:            config.Index,
		resourceProvider: config.ResourceProvider,
	}
	if config.Capacity > 0 {
		e.capacity = sdk.NewCapacity(config.Capacity)
	}
	return e
}

// buildTemplateMap 构建 template map（按 finger、id、tag 分类）
func buildTemplateMap(templates []*types.Template) map[string][]*types.Template {
	templateMap := make(map[string][]*types.Template)

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
	e.mu.Lock()
	defer e.mu.Unlock()

	e.installResourceProvider()
	if e.inited {
		return nil
	}

	// 自动从 Provider 创建 fingers/neutron 引擎
	if e.provider != nil {
		if e.fingersEngine == nil {
			fc := sdkfingers.NewConfig().WithProvider(e.provider)
			if eng, err := sdkfingers.NewEngine(fc); err == nil {
				e.fingersEngine = eng
			}
		}
		if e.neutronEngine == nil {
			nc := neutron.NewConfig().WithProvider(e.provider)
			if eng, err := neutron.NewEngine(nc); err == nil {
				e.neutronEngine = eng
			}
		}
	}

	// 尝试加载端口配置，失败时使用默认配置
	if err := pkg.LoadPortConfig(""); err != nil {
		// 使用默认端口配置，不阻止初始化
		logs.Log.Debugf("load port config failed, using default: %v", err)
	}

	if ok := e.applyInjectedFingers(); !ok {
		// 尝试创建默认的 fingers 引擎
		defaultFingers, err := sdkfingers.NewEngine(nil)
		if err == nil && defaultFingers != nil {
			e.fingersEngine = defaultFingers
			e.applyInjectedFingers()
		}
		if e.fingersEngine == nil || pkg.FingerEngine == nil {
			// 如果创建失败，尝试使用内置指纹
			if err := pkg.LoadFinger(nil); err != nil {
				logs.Log.Debugf("load finger failed, continuing without fingerprints: %v", err)
			}
		}
	}

	if ok := e.applyInjectedNeutron(); !ok {
		// 否则使用默认加载方式，允许失败
		if err := pkg.LoadNeutron(""); err != nil {
			logs.Log.Debugf("load neutron failed, using built-in: %v", err)
		}
	}

	// 自动构建关联索引
	if e.index == nil && e.fingersEngine != nil && e.neutronEngine != nil {
		idx := association.NewIndex()
		idx.BuildWithFingers(e.fingersEngine.Fingers(), e.fingersEngine.Aliases(), e.neutronEngine.Get())
		e.index = idx
	}

	e.inited = true
	return nil
}

func (e *GogoEngine) installResourceProvider() {
	if e.resourceProvider == nil {
		return
	}
	pkg.SetResourceProvider(e.resourceProvider)
}

// InstallResourceProvider installs the configured resource provider without
// initializing scanner globals. CLI wrappers call this before core parsing so
// direct commands load aiscan-managed templates during their own Init path.
func (e *GogoEngine) InstallResourceProvider() {
	if e == nil {
		return
	}
	e.installResourceProvider()
}

func (e *GogoEngine) applyInjectedEngines() error {
	e.applyInjectedFingers()
	e.applyInjectedNeutron()
	return nil
}

func (e *GogoEngine) applyInjectedFingers() bool {
	if e.fingersEngine == nil {
		return false
	}
	fingerImpl, err := e.fingersEngine.GetFingersEngine()
	if fingerImpl == nil || err != nil {
		logs.Log.Debugf("custom fingers engine is empty, falling back to built-in fingers")
		return false
	}
	pkg.FingerEngine = fingerImpl
	return true
}

func (e *GogoEngine) applyInjectedNeutron() bool {
	if e.neutronEngine == nil {
		return false
	}
	templates := e.neutronEngine.Get()
	if len(templates) == 0 {
		logs.Log.Debugf("custom neutron engine has no templates, skipping neutron integration")
		return false
	}
	templateMap := buildTemplateMap(templates)
	pkg.TemplateMap = templateMap
	templateCount := 0
	for _, values := range templateMap {
		templateCount += len(values)
	}
	logs.Log.Infof("resources type=neutron source=custom templates=%d categories=%d",
		templateCount, len(templateMap))
	return true
}

func (e *GogoEngine) Name() string {
	return "gogo"
}

// SetCapacity configures a capacity limit on an already-created engine.
func (e *GogoEngine) SetCapacity(total int) {
	if total > 0 {
		e.capacity = sdk.NewCapacity(total)
	}
}

// Capacity returns the engine's capacity bucket, or nil if unconfigured.
func (e *GogoEngine) Capacity() *sdk.Capacity {
	return e.capacity
}

// Index returns the association index, or nil if not built.
func (e *GogoEngine) Index() *association.Index {
	return e.index
}

func (e *GogoEngine) Execute(ctx sdk.Context, task sdk.Task) (<-chan sdk.Result, error) {
	if err := e.Init(); err != nil {
		return nil, err
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

func newResult(success bool, err error, data *types.GOGOResult) sdk.Result {
	return types.NewResult(success, err, data)
}

// ========================================
// 内部实现
// ========================================

func (e *GogoEngine) executeScan(ctx *Context, task *ScanTask) (<-chan sdk.Result, error) {
	if ctx == nil {
		ctx = NewContext()
	}
	runCtx := ctx

	workflow := &types.Workflow{
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

func (e *GogoEngine) workflowStream(ctx context.Context, workflow *types.Workflow, runCtx *Context) (<-chan sdk.Result, error) {
	// 创建基础配置
	if runCtx.opt == nil {
		runCtx.opt = types.NewDefaultGogoOption()
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

	// Acquire capacity before starting the scan goroutine
	threads := initConfig.Threads
	if e.capacity != nil {
		if err := e.capacity.Acquire(ctx, threads); err != nil {
			preparedConfig.Close()
			return nil, err
		}
	}

	// 创建结果 channel
	resultCh := make(chan sdk.Result, 100)

	// 启动扫描 goroutine
	go func() {
		defer close(resultCh)
		defer preparedConfig.Close()
		if e.capacity != nil {
			defer e.capacity.Release(threads)
		}

		var wg sync.WaitGroup
		var aliveCount int32
		var requests int64
		var errors int64
		var targets int64
		var tasks int64
		started := time.Now()
		defer func() {
			runCtx.emitStats(sdk.Stats{
				Engine:   e.Name(),
				Task:     "scan",
				Targets:  targets,
				Tasks:    tasks,
				Requests: atomic.LoadInt64(&requests),
				Results:  int64(atomic.LoadInt32(&aliveCount)),
				Errors:   atomic.LoadInt64(&errors),
				Duration: time.Since(started),
			})
		}()

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
			atomic.AddInt64(&requests, 1)
			engine.Dispatch(initConfig.RunnerOpt, result)

			if result.Open {
				atomic.AddInt32(&aliveCount, 1)
				// 发送结果到 channel
				select {
				case resultCh <- newResult(true, nil, result.GOGOResult):
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
				targets++

				for _, port := range initConfig.PortList {
					wg.Add(1)
					tasks++
					if err := scanPool.Invoke([]string{ipStr, port}); err != nil {
						atomic.AddInt64(&errors, 1)
						wg.Done()
					}
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
func (e *GogoEngine) ScanOne(ctx *Context, ip, port string) *types.GOGOResult {
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

	if err := e.Init(); err != nil {
		return result.GOGOResult
	}

	engine.Dispatch(runCtx.opt, result)
	return result.GOGOResult
}

// Scan 批量端口扫描（同步）
func (e *GogoEngine) Scan(ctx *Context, ip, ports string) ([]*types.GOGOResult, error) {
	if err := e.Init(); err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = NewContext()
	}

	task := NewScanTask(ip, ports)
	resultCh, err := e.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	var gogoResults []*types.GOGOResult
	for r := range resultCh {
		if result, ok := types.ResultData[*types.GOGOResult](r); r.Success() && ok && result != nil {
			gogoResults = append(gogoResults, result)
		}
	}

	return gogoResults, nil
}

// ScanStream 批量端口扫描（流式）
func (e *GogoEngine) ScanStream(ctx *Context, ip, ports string) (<-chan *types.GOGOResult, error) {
	if err := e.Init(); err != nil {
		return nil, err
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
	gogoResultCh := make(chan *types.GOGOResult, 100)
	go func() {
		defer close(gogoResultCh)
		for result := range resultCh {
			if data, ok := types.ResultData[*types.GOGOResult](result); result.Success() && ok && data != nil {
				gogoResultCh <- data
			}
		}
	}()

	return gogoResultCh, nil
}

// Workflow 工作流扫描（同步）
func (e *GogoEngine) Workflow(ctx *Context, workflow *types.Workflow) ([]*types.GOGOResult, error) {
	if err := e.Init(); err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = NewContext()
	}

	task := NewWorkflowTask(workflow)
	resultCh, err := e.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	var gogoResults []*types.GOGOResult
	for r := range resultCh {
		if result, ok := types.ResultData[*types.GOGOResult](r); r.Success() && ok && result != nil {
			gogoResults = append(gogoResults, result)
		}
	}

	return gogoResults, nil
}

// WorkflowStream 工作流扫描（流式）
func (e *GogoEngine) WorkflowStream(ctx *Context, workflow *types.Workflow) (<-chan *types.GOGOResult, error) {
	if err := e.Init(); err != nil {
		return nil, err
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
	gogoResultCh := make(chan *types.GOGOResult, 100)
	go func() {
		defer close(gogoResultCh)
		for result := range resultCh {
			if data, ok := types.ResultData[*types.GOGOResult](result); result.Success() && ok && data != nil {
				gogoResultCh <- data
			}
		}
	}()

	return gogoResultCh, nil
}
