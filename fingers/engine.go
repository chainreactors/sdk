package fingers

import (
	"context"
	"fmt"

	fingersLib "github.com/chainreactors/fingers"
	"github.com/chainreactors/fingers/alias"
	"github.com/chainreactors/fingers/common"
	fingersEngine "github.com/chainreactors/fingers/fingers"
	"github.com/chainreactors/fingers/resources"
	sdk "github.com/chainreactors/sdk/pkg"
)

// ========================================
// Engine - 统一的指纹引擎
// ========================================

// Engine 是对 fingers 库的封装，支持多种数据源加载
type Engine struct {
	engine  *fingersLib.Engine
	config  *Config
	aliases []*alias.Alias // 原始别名数据
}

// NewEngine 创建一个新的 Engine 实例
// 根据 config 自动选择加载方式（本地/远程）
func NewEngine(config *Config) (*Engine, error) {
	if config == nil {
		config = NewConfig()
	}

	if err := config.Load(context.Background()); err != nil {
		return nil, err
	}

	fingers := config.FullFingers.Fingers()
	if len(fingers) == 0 {
		return nil, fmt.Errorf("fingers data is empty")
	}

	e := &Engine{
		config: config,
	}

	engine, err := buildEngineFromFingers(fingers, config.FullFingers.Aliases())
	if err != nil {
		return nil, err
	}

	e.aliases = config.FullFingers.Aliases()
	e.engine = engine

	return e, nil
}

// NewEngineWithFingers creates an Engine using FullFingers directly.
func NewEngineWithFingers(fingers FullFingers) (*Engine, error) {
	if fingers.Len() == 0 {
		return nil, fmt.Errorf("fingers data is empty")
	}

	config := NewConfig()
	config.FullFingers = fingers

	engine, err := buildEngineFromFingers(fingers.Fingers(), fingers.Aliases())
	if err != nil {
		return nil, err
	}

	return &Engine{
		engine:  engine,
		config:  config,
		aliases: fingers.Aliases(),
	}, nil
}

// ========================================
// 统一 API - 只提供一种加载方式
// ========================================

// Get 获取底层的 fingers.Engine
func (e *Engine) Get() *fingersLib.Engine {
	return e.engine
}

// GetFingersEngine 获取 FingersEngine（用于 gogo 集成）
func (e *Engine) GetFingersEngine() (*fingersEngine.FingersEngine, error) {
	if e.engine == nil {
		return nil, fmt.Errorf("fingers engine is not initialized")
	}

	impl := e.engine.GetEngine("fingers")
	if impl == nil {
		return nil, nil
	}

	return impl.(*fingersEngine.FingersEngine), nil
}

// Reload 重新加载指纹
func (e *Engine) Reload(ctx context.Context) error {
	if e.config == nil {
		return fmt.Errorf("config is nil")
	}
	if err := e.config.Load(ctx); err != nil {
		return err
	}

	engine, err := buildEngineFromFingers(e.config.FullFingers.Fingers(), e.config.FullFingers.Aliases())
	if err != nil {
		return err
	}

	e.aliases = e.config.FullFingers.Aliases()
	e.engine = engine
	return nil
}

// buildEngineFromFingers 从指纹列表构建引擎
func buildEngineFromFingers(fingers fingersEngine.Fingers, aliases []*alias.Alias) (*fingersLib.Engine, error) {
	engine, err := fingersLib.NewEngine(
		fingersLib.FaviconEngine,
		fingersLib.EHoleEngine,
		fingersLib.FingerPrintEngine,
		fingersLib.GobyEngine,
		fingersLib.WappalyzerEngine,
		fingersLib.NmapEngine,
	)
	if err != nil {
		return nil, err
	}

	var httpFingers, socketFingers fingersEngine.Fingers
	for _, finger := range fingers {
		if finger.Protocol == "http" {
			httpFingers = append(httpFingers, finger)
		} else if finger.Protocol == "tcp" {
			socketFingers = append(socketFingers, finger)
		}
	}
	_, err = resources.LoadPorts()
	if err != nil {
		return nil, err
	}
	fEngine, err := fingersEngine.NewEngine(httpFingers, socketFingers)
	if err != nil {
		return nil, err
	}

	engine.Register(fEngine)

	if len(aliases) > 0 {
		aliasEngine, err := alias.NewAliases(aliases...)
		if err == nil {
			engine.Aliases = aliasEngine
		}
	}

	engine.Compile()
	return engine, nil
}

// Count 获取指纹总数
func (e *Engine) Count() int {
	if e.config == nil {
		return 0
	}
	return len(e.config.FullFingers.Fingers())
}

// Close 关闭引擎
func (e *Engine) Close() error {
	return nil
}

// ========================================
// 核心匹配 API - 原子化设计
// ========================================

// Match 匹配单个 HTTP 响应原始数据（唯一的核心 API）
func (e *Engine) Match(data []byte) (common.Frameworks, error) {
	if e.engine == nil {
		return nil, fmt.Errorf("fingers engine is not initialized")
	}
	return e.engine.DetectContent(data)
}

// ========================================
// SDK Engine 接口实现（可选）
// ========================================

// Name 返回引擎名称（实现 sdk.Engine 接口）
func (e *Engine) Name() string {
	return "fingers"
}

// Execute 执行任务（实现 sdk.Engine 接口）
func (e *Engine) Execute(ctx sdk.Context, task sdk.Task) (<-chan sdk.Result, error) {
	// 确保引擎已初始化
	if e.engine == nil {
		return nil, fmt.Errorf("fingers engine is not initialized")
	}

	// 验证任务
	if err := task.Validate(); err != nil {
		return nil, err
	}

	// 只支持 MatchTask
	matchTask, ok := task.(*MatchTask)
	if !ok {
		return nil, fmt.Errorf("unsupported task type: %s", task.Type())
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

	return e.executeMatch(runCtx, matchTask)
}

// executeMatch 执行单个指纹匹配任务
func (e *Engine) executeMatch(ctx *Context, task *MatchTask) (<-chan sdk.Result, error) {
	resultCh := make(chan sdk.Result, 1)

	go func() {
		defer close(resultCh)

		frameworks, err := e.Match(task.Data)

		// 发送结果
		select {
		case resultCh <- &MatchResult{
			success:    err == nil,
			err:        err,
			frameworks: frameworks,
		}:
		case <-ctx.Context().Done():
		}
	}()

	return resultCh, nil
}
