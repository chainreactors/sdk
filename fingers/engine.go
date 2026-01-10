package fingers

import (
	"context"
	"encoding/json"
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
	rawFingers   fingersEngine.Fingers          // 原始指纹数据（用于筛选）
	aliases      []*alias.Alias                 // 原始别名数据
	fingerprints []cyberhub.FingerprintResponse // 缓存原始响应
	pocIndex     map[string][]string            // fingerprintName → pocNames
	productIndex map[string][]string            // vendor:product → pocNames
	aliasIndex   map[string]*alias.Alias        // fingerprintName → alias
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
	fingerprints, err := e.loadFingerprints(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fingerprints from cyberhub: %w", err)
	}

	return e.convertToEngine(fingerprints)
}

func (e *Engine) loadFingerprints(ctx context.Context, filter *FingerprintFilter) ([]cyberhub.FingerprintResponse, error) {
	query := e.buildFingerprintQuery(filter)

	responses, err := e.client.ExportFingerprintsWithQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	if filter == nil || filter.isLocalEmpty() {
		return responses, nil
	}

	return filterFingerprintResponses(responses, filter), nil
}

func (e *Engine) buildFingerprintQuery(filter *FingerprintFilter) *cyberhub.Query {
	query := cyberhub.NewQuery().WithFingerprint(true)

	if filter != nil && len(filter.Sources) > 0 {
		query.Filter("sources", filter.Sources...)
	} else if e.config.Source != "" {
		query.Filter("sources", e.config.Source)
	}

	if filter == nil {
		return query
	}

	if filter.Keyword != "" {
		query.Keyword(filter.Keyword)
	}
	if filter.Protocol != "" {
		query.Set("protocol", filter.Protocol)
	}
	if len(filter.Tags) > 0 {
		query.Tags(filter.Tags...)
	}
	if len(filter.Categories) > 0 {
		query.Filter("categories", filter.Categories...)
	}
	if filter.Vendor != "" {
		query.Set("vendor", filter.Vendor)
	}
	if filter.Product != "" {
		query.Set("product", filter.Product)
	}
	if len(filter.Authors) > 0 {
		for _, author := range filter.Authors {
			if author != "" {
				query.Set("author", author)
				break
			}
		}
	}
	if len(filter.Statuses) > 0 {
		query.Filter("statuses", filter.Statuses...)
	} else if filter.Status != "" {
		query.Set("status", filter.Status)
	}

	return query
}

func filterFingerprintResponses(responses []cyberhub.FingerprintResponse, filter *FingerprintFilter) []cyberhub.FingerprintResponse {
	if filter == nil || filter.isLocalEmpty() {
		return responses
	}

	aliasIndex := make(map[string]*alias.Alias)
	var pocIndex map[string]bool
	if filter.HasAssociatedPOC != nil {
		pocIndex = make(map[string]bool)
	}

	for _, resp := range responses {
		if resp.Finger == nil {
			continue
		}
		if aliasData := resp.GetAlias(); aliasData != nil {
			aliasIndex[resp.Finger.Name] = aliasData
			if pocIndex != nil {
				pocIndex[resp.Finger.Name] = len(aliasData.Pocs) > 0
			}
		}
	}

	filter.SetAliasIndex(aliasIndex)
	if pocIndex != nil {
		filter.SetPOCAssociationIndex(pocIndex)
	}

	var result []cyberhub.FingerprintResponse
	for _, resp := range responses {
		if resp.Finger == nil {
			continue
		}
		if filter.match(resp.Finger) {
			result = append(result, resp)
		}
	}
	return result
}

// convertToEngine 将 Cyberhub 响应转换为 fingers.Engine
func (e *Engine) convertToEngine(responses []cyberhub.FingerprintResponse) (*fingersLib.Engine, error) {
	engine, state, err := buildEngineFromResponses(responses)
	if err != nil {
		return nil, err
	}

	e.fingerprints = responses
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

// Filter 使用筛选器筛选指纹
func (e *Engine) Filter(filter *FingerprintFilter) fingersEngine.Fingers {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if filter == nil {
		return e.rawFingers
	}
	e.bindFilterIndexes(filter)
	return filter.Apply(e.rawFingers)
}

func (e *Engine) bindFilterIndexes(filter *FingerprintFilter) {
	if filter == nil {
		return
	}
	if filter.aliasIndex == nil && e.aliasIndex != nil {
		filter.SetAliasIndex(e.aliasIndex)
	}
	if filter.HasAssociatedPOC != nil && filter.pocFingerIndex == nil {
		filter.SetPOCAssociationIndex(e.buildPOCHasIndex())
	}
}

// LoadWithFilter 加载并筛选指纹
func (e *Engine) LoadWithFilter(ctx context.Context, filter *FingerprintFilter) (*fingersLib.Engine, error) {
	if filter == nil {
		return e.Load(ctx)
	}

	e.mu.RLock()
	loaded := e.engine != nil
	e.mu.RUnlock()

	if loaded {
		e.mu.RLock()
		defer e.mu.RUnlock()
		e.bindFilterIndexes(filter)
		filteredFingers := filter.Apply(e.rawFingers)
		aliases := filterAliasesForFingers(e.aliases, filteredFingers)
		return buildEngineFromFingers(filteredFingers, aliases)
	}

	if e.config.IsRemoteEnabled() {
		responses, err := e.loadFingerprints(ctx, filter)
		if err != nil {
			return nil, err
		}
		engine, _, err := buildEngineFromResponses(responses)
		if err != nil {
			return nil, err
		}
		return engine, nil
	}

	if _, err := e.Load(ctx); err != nil {
		return nil, err
	}

	e.mu.RLock()
	defer e.mu.RUnlock()
	e.bindFilterIndexes(filter)
	filteredFingers := filter.Apply(e.rawFingers)
	aliases := filterAliasesForFingers(e.aliases, filteredFingers)
	return buildEngineFromFingers(filteredFingers, aliases)
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
// 文件持久化 API
// ========================================

// SaveToFile 将已加载的指纹数据保存到文件（原子写入）
func (e *Engine) SaveToFile(path string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.fingerprints) == 0 {
		return fmt.Errorf("no fingerprints loaded to save")
	}

	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(e.fingerprints); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to encode fingerprints: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// LoadFromFile 从文件加载指纹数据并构建引擎
func (e *Engine) LoadFromFile(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var responses []cyberhub.FingerprintResponse
	if err := json.NewDecoder(file).Decode(&responses); err != nil {
		return fmt.Errorf("failed to decode fingerprints: %w", err)
	}

	engine, state, err := buildEngineFromResponses(responses)
	if err != nil {
		return fmt.Errorf("failed to build engine: %w", err)
	}

	e.engine = engine
	e.fingerprints = responses
	e.rawFingers = state.rawFingers
	e.aliases = state.aliases
	e.pocIndex = state.pocIndex
	e.productIndex = state.productIndex
	e.aliasIndex = state.aliasIndex

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
