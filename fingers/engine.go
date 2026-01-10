package fingers

import (
	"context"
	"fmt"
	"net/http"
	"os"
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
	"gopkg.in/yaml.v3"
)

// ========================================
// Engine - 统一的指纹引擎
// ========================================

// Engine 是对 fingers 库的封装，支持多种数据源加载
type Engine struct {
	engine       *fingersLib.Engine
	config       *Config
	client       *cyberhub.Client // 仅在远程模式下使用
	mu           sync.RWMutex
	rawFingers   fingersEngine.Fingers   // 原始指纹数据（用于筛选）
	aliases      []*alias.Alias          // 原始别名数据
	pocIndex     map[string][]string     // fingerprintName → pocNames
	productIndex map[string][]string     // vendor:product → pocNames
	aliasIndex   map[string]*alias.Alias // fingerprintName → alias
}

type engineState struct {
	rawFingers   fingersEngine.Fingers
	aliases      []*alias.Alias
	pocIndex     map[string][]string
	productIndex map[string][]string
	aliasIndex   map[string]*alias.Alias
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
		)
	}

	return e, nil
}

// ========================================
// 统一 API - 只提供一种加载方式
// ========================================

// Load 加载并返回 fingers 库的 Engine
// config 为 nil 时使用默认本地配置
func Load(config *Config) (*fingersLib.Engine, error) {
	if config == nil {
		config = NewConfig()
	}

	engine, err := NewEngine(config)
	if err != nil {
		return nil, err
	}

	return engine.Load(context.Background())
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
	if e.config.Filename != "" {
		engine, err = e.loadFromFile(e.config.Filename)
	} else if e.config.IsRemoteEnabled() {
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

	// 从 engine 中提取原始指纹数据，用于后续筛选
	e.extractFingersFromEngine(engine)

	return engine, nil
}

// extractFingersFromEngine 从 fingersLib.Engine 中提取原始指纹数据
func (e *Engine) extractFingersFromEngine(engine *fingersLib.Engine) {
	impl := engine.GetEngine("fingers")
	if impl == nil {
		return
	}

	fEngine, ok := impl.(*fingersEngine.FingersEngine)
	if !ok || fEngine == nil {
		return
	}

	// 合并 HTTP 和 Socket 指纹
	e.rawFingers = append(fEngine.HTTPFingers, fEngine.SocketFingers...)

	// 提取 aliases
	if engine.Aliases != nil {
		e.aliasIndex = make(map[string]*alias.Alias)
		for name, a := range engine.Aliases.Aliases {
			e.aliasIndex[name] = a
			e.aliases = append(e.aliases, a)
		}

		// 构建 POC 索引
		e.pocIndex = make(map[string][]string)
		e.productIndex = make(map[string][]string)
		for name, a := range engine.Aliases.Aliases {
			if len(a.Pocs) > 0 {
				e.pocIndex[name] = a.Pocs
				if a.Vendor != "" && a.Product != "" {
					key := a.Vendor + ":" + a.Product
					e.productIndex[key] = append(e.productIndex[key], a.Pocs...)
				}
			}
		}
	}
}

// loadFromRemote 从 Cyberhub 加载指纹
func (e *Engine) loadFromRemote(ctx context.Context) (*fingersLib.Engine, error) {
	fingerprints, err := e.loadFingerprints(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fingerprints from cyberhub: %w", err)
	}

	return e.convertToEngine(fingerprints)
}

func (e *Engine) loadFromFile(path string) (*fingersLib.Engine, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var raw []*fingersEngine.Finger
	if err := yaml.NewDecoder(file).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode fingerprints: %w", err)
	}

	engine, err := buildEngineFromFingers(fingersEngine.Fingers(raw), nil)
	if err != nil {
		return nil, err
	}

	e.rawFingers = fingersEngine.Fingers(raw)
	e.aliases = nil
	e.pocIndex = nil
	e.productIndex = nil
	e.aliasIndex = nil
	return engine, nil
}

func (e *Engine) loadFingerprints(ctx context.Context) ([]cyberhub.FingerprintResponse, error) {
	var filter *cyberhub.ExportFilter
	if e.config != nil {
		filter = e.config.ExportFilter
	}

	responses, err := e.client.ExportFingerprints(ctx, true, "", filter)
	if err != nil {
		return nil, err
	}

	return responses, nil
}

// convertToEngine 将 Cyberhub 响应转换为 fingers.Engine
func (e *Engine) convertToEngine(responses []cyberhub.FingerprintResponse) (*fingersLib.Engine, error) {
	engine, state, err := buildEngineFromResponses(responses)
	if err != nil {
		return nil, err
	}

	e.rawFingers = state.rawFingers
	e.aliases = state.aliases
	e.pocIndex = state.pocIndex
	e.productIndex = state.productIndex
	e.aliasIndex = state.aliasIndex
	return engine, nil
}

func buildEngineFromResponses(responses []cyberhub.FingerprintResponse) (*fingersLib.Engine, *engineState, error) {
	engine := &fingersLib.Engine{
		EnginesImpl:  make(map[string]fingersLib.EngineImpl),
		Enabled:      make(map[string]bool),
		Capabilities: make(map[string]common.EngineCapability),
	}

	var httpFingers, socketFingers fingersEngine.Fingers
	var aliases []*alias.Alias

	pocIndex := make(map[string][]string)
	productIndex := make(map[string][]string)
	aliasIndex := make(map[string]*alias.Alias)

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

			if len(aliasData.Pocs) > 0 {
				pocIndex[finger.Name] = aliasData.Pocs

				if aliasData.Vendor != "" && aliasData.Product != "" {
					key := aliasData.Vendor + ":" + aliasData.Product
					productIndex[key] = append(productIndex[key], aliasData.Pocs...)
				}
			}

			aliasIndex[finger.Name] = aliasData
		}
	}

	rawFingers := append(httpFingers, socketFingers...)
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
	return engine, &engineState{
		rawFingers:   rawFingers,
		aliases:      aliases,
		pocIndex:     pocIndex,
		productIndex: productIndex,
		aliasIndex:   aliasIndex,
	}, nil
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

func (e *Engine) buildPOCHasIndex() map[string]bool {
	if e.pocIndex == nil {
		return nil
	}
	index := make(map[string]bool, len(e.pocIndex))
	for name, pocs := range e.pocIndex {
		index[name] = len(pocs) > 0
	}
	return index
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
	e.rawFingers = nil
	e.aliases = nil
	_, err := e.Load(ctx)
	return err
}

// ========================================
// 筛选功能
// ========================================

// GetRawFingers 获取原始指纹列表（用于外部筛选）
func (e *Engine) GetRawFingers() fingersEngine.Fingers {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rawFingers
}

// buildEngineFromFingers 从指纹列表构建引擎
func buildEngineFromFingers(fingers fingersEngine.Fingers, aliases []*alias.Alias) (*fingersLib.Engine, error) {
	engine := &fingersLib.Engine{
		EnginesImpl:  make(map[string]fingersLib.EngineImpl),
		Enabled:      make(map[string]bool),
		Capabilities: make(map[string]common.EngineCapability),
	}

	var httpFingers, socketFingers fingersEngine.Fingers
	for _, finger := range fingers {
		if finger.Protocol == "http" {
			httpFingers = append(httpFingers, finger)
		} else if finger.Protocol == "tcp" {
			socketFingers = append(socketFingers, finger)
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

func filterAliasesForFingers(aliases []*alias.Alias, fingers fingersEngine.Fingers) []*alias.Alias {
	if len(aliases) == 0 || len(fingers) == 0 {
		return nil
	}
	nameIndex := make(map[string]struct{}, len(fingers))
	for _, finger := range fingers {
		nameIndex[finger.Name] = struct{}{}
	}
	var result []*alias.Alias
	for _, aliasData := range aliases {
		if _, ok := nameIndex[aliasData.Name]; ok {
			result = append(result, aliasData)
		}
	}
	return result
}

// Count 获取指纹总数
func (e *Engine) Count() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.rawFingers)
}

// Close 关闭引擎
func (e *Engine) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

// ========================================
// POC 关联查询 API
// ========================================

// GetPOCNames 根据指纹名称获取关联的 POC 名称列表
func (e *Engine) GetPOCNames(fingerprintName string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.pocIndex == nil {
		return nil
	}
	return e.pocIndex[fingerprintName]
}

// GetPOCNamesByProduct 根据 vendor/product 获取关联的 POC 名称列表
func (e *Engine) GetPOCNamesByProduct(vendor, product string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.productIndex == nil {
		return nil
	}
	key := vendor + ":" + product
	return e.productIndex[key]
}

// GetAlias 根据指纹名称获取 Alias
func (e *Engine) GetAlias(fingerprintName string) *alias.Alias {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.aliasIndex == nil {
		return nil
	}
	return e.aliasIndex[fingerprintName]
}

// GetAllPOCNames 获取所有关联的 POC 名称（去重）
func (e *Engine) GetAllPOCNames() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.pocIndex == nil {
		return nil
	}

	seen := make(map[string]struct{})
	var result []string
	for _, names := range e.pocIndex {
		for _, name := range names {
			if _, exists := seen[name]; !exists {
				seen[name] = struct{}{}
				result = append(result, name)
			}
		}
	}
	return result
}

// GetPOCNamesFromFrameworks 从匹配结果中获取所有关联的 POC 名称
func (e *Engine) GetPOCNamesFromFrameworks(frameworks common.Frameworks) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.pocIndex == nil {
		return nil
	}

	seen := make(map[string]struct{})
	var result []string
	for _, fw := range frameworks {
		if names, ok := e.pocIndex[fw.Name]; ok {
			for _, name := range names {
				if _, exists := seen[name]; !exists {
					seen[name] = struct{}{}
					result = append(result, name)
				}
			}
		}
	}
	return result
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
