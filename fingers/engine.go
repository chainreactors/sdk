package fingers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	fingersLib "github.com/chainreactors/fingers"
	"github.com/chainreactors/fingers/alias"
	"github.com/chainreactors/fingers/common"
	"github.com/chainreactors/fingers/favicon"
	fingersEngine "github.com/chainreactors/fingers/fingers"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"github.com/chainreactors/utils/httputils"
)

// ========================================
// Engine - 统一的指纹引擎
// ========================================

// Engine 是对 fingers 库的封装，支持多种数据源加载
type Engine struct {
	engine *fingersLib.Engine
	config *Config
	client *cyberhub.Client // 仅在远程模式下使用
	mu     sync.RWMutex
}

// NewEngine 创建一个新的 Engine 实例
// 根据 config 自动选择加载方式（本地/远程）
func NewEngine(config *Config) (*Engine, error) {
	if config == nil {
		config = NewConfig()
	}

	e := &Engine{
		config: config,
	}

	// 如果配置了远程，创建 client
	if config.IsRemoteEnabled() {
		e.client = cyberhub.NewClient(
			config.CyberhubURL,
			config.APIKey,
			config.Timeout,
			config.MaxRetries,
		)
	}

	return e, nil
}

// Load 加载指纹引擎
func (e *Engine) Load(ctx context.Context) (*fingersLib.Engine, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.engine != nil {
		return e.engine, nil
	}

	var engine *fingersLib.Engine
	var err error

	// 根据配置选择加载方式
	if e.config.IsRemoteEnabled() {
		engine, err = e.loadFromRemote(ctx)
	} else {
		engine, err = e.loadFromLocal()
	}

	if err != nil {
		return nil, err
	}

	e.engine = engine
	return engine, nil
}

// loadFromLocal 从本地加载指纹
func (e *Engine) loadFromLocal() (*fingersLib.Engine, error) {
	engines := e.config.EnableEngines
	// 如果为空，传 nil 给 NewEngine 让其使用默认引擎
	if len(engines) == 0 {
		engines = nil
	}

	engine, err := fingersLib.NewEngine(engines...)
	if err != nil {
		return nil, fmt.Errorf("failed to load local engines: %w", err)
	}

	return engine, nil
}

// loadFromRemote 从 Cyberhub 加载指纹
func (e *Engine) loadFromRemote(ctx context.Context) (*fingersLib.Engine, error) {
	fingerprints, err := e.client.ExportFingerprints(ctx, true, e.config.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fingerprints from cyberhub: %w", err)
	}

	return e.convertToEngine(fingerprints)
}

// convertToEngine 将 Cyberhub 响应转换为 fingers.Engine
func (e *Engine) convertToEngine(responses []cyberhub.FingerprintResponse) (*fingersLib.Engine, error) {
	engine := &fingersLib.Engine{
		EnginesImpl:  make(map[string]fingersLib.EngineImpl),
		Enabled:      make(map[string]bool),
		Capabilities: make(map[string]common.EngineCapability),
	}

	var httpFingers, socketFingers fingersEngine.Fingers
	var aliases []*alias.Alias

	for _, resp := range responses {
		if !resp.IsActive() {
			continue
		}

		finger := resp.GetFinger()
		if finger.Protocol == "http" {
			httpFingers = append(httpFingers, finger)
		} else if finger.Protocol == "tcp" {
			socketFingers = append(socketFingers, finger)
		}

		if aliasData := resp.GetAlias(); aliasData != nil {
			aliases = append(aliases, aliasData)
		}
	}

	fEngine := &fingersEngine.FingersEngine{
		HTTPFingers:              httpFingers,
		SocketFingers:            socketFingers,
		HTTPFingersActiveFingers: filterActiveFingers(httpFingers),
		Favicons:                 favicon.NewFavicons(),
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

func filterActiveFingers(fingers fingersEngine.Fingers) fingersEngine.Fingers {
	var active fingersEngine.Fingers
	for _, f := range fingers {
		if f.Focus {
			active = append(active, f)
		}
	}
	return active
}

// Get 获取底层的 fingers.Engine
func (e *Engine) Get() *fingersLib.Engine {
	return e.engine
}

// GetFingersEngine 获取 FingersEngine（用于 gogo 集成）
func (e *Engine) GetFingersEngine() (*fingersEngine.FingersEngine, error) {
	if e.engine == nil {
		if _, err := e.Load(context.Background()); err != nil {
			return nil, err
		}
	}

	impl := e.engine.GetEngine("fingers")
	if impl == nil {
		return nil, nil
	}

	return impl.(*fingersEngine.FingersEngine), nil
}

// Reload 重新加载指纹
func (e *Engine) Reload(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.engine = nil
	_, err := e.Load(ctx)
	return err
}

// Close 关闭引擎
func (e *Engine) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

// ========================================
// 核心匹配 API - 原子化设计
// ========================================

// Match 匹配单个 HTTP 响应原始数据（唯一的核心 API）
func (e *Engine) Match(data []byte) (common.Frameworks, error) {
	if e.engine == nil {
		if _, err := e.Load(context.Background()); err != nil {
			return nil, err
		}
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
		if _, err := e.Load(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to load fingerprints: %w", err)
		}
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

	return e.executeMatch(ctx, matchTask)
}

// executeMatch 执行单个指纹匹配任务
func (e *Engine) executeMatch(ctx sdk.Context, task *MatchTask) (<-chan sdk.Result, error) {
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

// ========================================
// Context 实现
// ========================================

// Context Fingers 上下文
type Context struct {
	ctx    context.Context
	config *Config
}

// NewContext 创建 Fingers 上下文
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
// Task 实现（SDK Engine 可选）
// ========================================

// MatchTask 指纹匹配任务
type MatchTask struct {
	Data []byte // HTTP 响应原始数据
}

// NewMatchTask 创建匹配任务
func NewMatchTask(data []byte) *MatchTask {
	return &MatchTask{Data: data}
}

// NewMatchTaskFromResponse 从 HTTP Response 创建任务
func NewMatchTaskFromResponse(resp *http.Response) *MatchTask {
	data := httputils.ReadRaw(resp)
	return &MatchTask{Data: data}
}

func (t *MatchTask) Type() string {
	return "match"
}

func (t *MatchTask) Validate() error {
	if len(t.Data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

// ========================================
// Result 实现
// ========================================

// MatchResult 指纹匹配结果
type MatchResult struct {
	success    bool
	err        error
	frameworks common.Frameworks
}

func (r *MatchResult) Success() bool {
	return r.success
}

func (r *MatchResult) Error() error {
	return r.err
}

func (r *MatchResult) Data() interface{} {
	return r.frameworks
}

// Frameworks 获取匹配到的指纹
func (r *MatchResult) Frameworks() common.Frameworks {
	return r.frameworks
}

// HasMatch 是否匹配到指纹
func (r *MatchResult) HasMatch() bool {
	return len(r.frameworks) > 0
}

// Count 匹配到的指纹数量
func (r *MatchResult) Count() int {
	return len(r.frameworks)
}
