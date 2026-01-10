package neutron

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/chainreactors/neutron/protocols"
	"github.com/chainreactors/neutron/templates"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"gopkg.in/yaml.v3"
)

// ========================================
// Engine - Neutron 加载引擎
// ========================================

// Engine Neutron 加载引擎，支持本地和远程数据源
type Engine struct {
	templates     []*templates.Template
	templateIndex map[string]*templates.Template // ID → template 索引
	config        *Config
	client        *cyberhub.Client // 仅在远程模式下使用
	mu            sync.RWMutex
}

// NewEngine 创建一个新的 Engine 实例
// 根据 config 自动选择加载方式（本地/远程）
func NewEngine(config *Config) (*Engine, error) {
	if config == nil {
		config = NewConfig()
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, err
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

// Load 加载 POC templates 并进行编译
func (e *Engine) Load(ctx context.Context) ([]*templates.Template, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.templates != nil {
		return e.templates, nil
	}

	var allTemplates []*templates.Template
	var err error

	// 根据配置选择加载方式
	if e.config.IsRemoteEnabled() {
		// 从远程加载
		allTemplates, err = e.loadFromRemote(ctx)
		if err != nil {
			return nil, err
		}
	}

	if e.config.IsLocalEnabled() {
		// 从本地加载
		localTemplates, err := e.loadFromLocal()
		if err != nil {
			// 如果已经有远程数据，本地加载失败仅记录警告
			if len(allTemplates) > 0 {
				fmt.Printf("Warning: failed to load local templates: %v\n", err)
			} else {
				return nil, err
			}
		} else {
			allTemplates = append(allTemplates, localTemplates...)
		}
	}

	// 编译所有加载的 templates
	compiledTemplates := e.compileTemplates(allTemplates)

	e.templates = compiledTemplates

	// 构建索引
	e.templateIndex = make(map[string]*templates.Template)
	for _, t := range compiledTemplates {
		if t.Id != "" {
			e.templateIndex[t.Id] = t
		}
	}

	return compiledTemplates, nil
}

// loadFromLocal 从本地文件/目录加载 POC
func (e *Engine) loadFromLocal() ([]*templates.Template, error) {
	path := e.config.LocalPath
	if path == "" {
		path = "."
	}

	// 检查路径是否存在
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access path %s: %w", path, err)
	}

	var yamlFiles []string

	if info.IsDir() {
		// 遍历目录查找 YAML 文件
		err = filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			ext := filepath.Ext(filePath)
			if ext == ".yaml" || ext == ".yml" {
				yamlFiles = append(yamlFiles, filePath)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
		}
	} else {
		// 单个文件
		yamlFiles = []string{path}
	}

	// 加载所有 YAML 文件
	var loadedTemplates []*templates.Template
	var loadErrors []string

	for _, yamlFile := range yamlFiles {
		content, err := os.ReadFile(yamlFile)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("read %s: %v", yamlFile, err))
			continue
		}

		t := &templates.Template{}
		if err := yaml.Unmarshal(content, t); err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("parse %s: %v", yamlFile, err))
			continue
		}

		loadedTemplates = append(loadedTemplates, t)
	}

	if len(loadErrors) > 0 {
		fmt.Printf("Warning: %d files failed to load: %v\n", len(loadErrors), loadErrors)
	}

	if len(loadedTemplates) == 0 {
		return nil, fmt.Errorf("no valid templates loaded from %s", path)
	}

	return loadedTemplates, nil
}

// loadFromRemote 从 Cyberhub 加载 POC
func (e *Engine) loadFromRemote(ctx context.Context) ([]*templates.Template, error) {
	return e.loadRemoteTemplates(ctx, nil)
}

func (e *Engine) loadRemoteTemplates(ctx context.Context, filter *POCFilter) ([]*templates.Template, error) {
	query := e.buildPOCQuery(filter)

	responses, err := e.client.ExportPOCsWithQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch POCs from cyberhub: %w", err)
	}

	var loadedTemplates []*templates.Template
	for _, resp := range responses {
		loadedTemplates = append(loadedTemplates, resp.GetTemplate())
	}

	if filter == nil || filter.isLocalEmpty() {
		return loadedTemplates, nil
	}

	return filter.Apply(loadedTemplates), nil
}

func (e *Engine) buildPOCQuery(filter *POCFilter) *cyberhub.Query {
	query := cyberhub.NewQuery()

	if filter != nil && len(filter.Sources) > 0 {
		query.Filter("source_names", filter.Sources...)
	} else if e.config.Source != "" {
		query.Set("source", e.config.Source)
	}

	if filter == nil {
		return query
	}

	if filter.Keyword != "" {
		query.Keyword(filter.Keyword)
	}
	if len(filter.Tags) > 0 {
		query.Tags(filter.Tags...)
	}
	if len(filter.Severities) > 0 {
		query.Severities(filter.Severities...)
	}
	if filter.Type != "" {
		query.Type(filter.Type)
	}
	if len(filter.SourceIDs) > 0 {
		for _, id := range filter.SourceIDs {
			query.Filter("source_ids", strconv.FormatUint(uint64(id), 10))
		}
	}
	if len(filter.Statuses) > 0 {
		query.Filter("statuses", filter.Statuses...)
	} else if filter.Status != "" {
		query.Status(filter.Status)
	}
	if len(filter.AdvancedFilters) > 0 {
		if data, err := json.Marshal(filter.AdvancedFilters); err == nil {
			query.Set("advanced_filters", string(data))
		}
	}

	return query
}

// Get 获取已加载的 templates
func (e *Engine) Get() []*templates.Template {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.templates
}

// Count 获取已加载的 template 数量
func (e *Engine) Count() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.templates)
}

// Reload 重新加载 templates
func (e *Engine) Reload(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.templates = nil
	_, err := e.Load(ctx)
	return err
}

// ========================================
// 筛选功能
// ========================================

// Filter 使用筛选器筛选POC
func (e *Engine) Filter(filter *POCFilter) []*templates.Template {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if filter == nil {
		return e.templates
	}
	return filter.Apply(e.templates)
}

// LoadWithFilter 加载并筛选POC
func (e *Engine) LoadWithFilter(ctx context.Context, filter *POCFilter) ([]*templates.Template, error) {
	if filter == nil {
		return e.Load(ctx)
	}

	e.mu.RLock()
	loaded := e.templates != nil
	e.mu.RUnlock()

	if loaded {
		e.mu.RLock()
		defer e.mu.RUnlock()
		return filter.Apply(e.templates), nil
	}

	var allTemplates []*templates.Template
	if e.config.IsRemoteEnabled() {
		remoteTemplates, err := e.loadRemoteTemplates(ctx, filter)
		if err != nil {
			return nil, err
		}
		allTemplates = append(allTemplates, remoteTemplates...)
	}

	if e.config.IsLocalEnabled() {
		localTemplates, err := e.loadFromLocal()
		if err != nil {
			if len(allTemplates) > 0 {
				fmt.Printf("Warning: failed to load local templates: %v\n", err)
			} else {
				return nil, err
			}
		} else {
			allTemplates = append(allTemplates, filter.Apply(localTemplates)...)
		}
	}

	compiledTemplates := e.compileTemplates(allTemplates)
	return compiledTemplates, nil
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

// SaveToFile 将已加载的 templates 保存到文件（原子写入）
func (e *Engine) SaveToFile(path string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.templates) == 0 {
		return fmt.Errorf("no templates loaded to save")
	}

	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(e.templates); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to encode templates: %w", err)
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

// LoadFromFile 从文件加载 templates 并编译
func (e *Engine) LoadFromFile(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var rawTemplates []*templates.Template
	if err := json.NewDecoder(file).Decode(&rawTemplates); err != nil {
		return fmt.Errorf("failed to decode templates: %w", err)
	}

	// 编译加载的 templates
	compiledTemplates := e.compileTemplates(rawTemplates)

	e.templates = compiledTemplates

	// 构建索引
	e.templateIndex = make(map[string]*templates.Template)
	for _, t := range compiledTemplates {
		if t.Id != "" {
			e.templateIndex[t.Id] = t
		}
	}

	return nil
}

// ========================================
// 按需加载 API
// ========================================

// compileOptions 返回编译选项
func (e *Engine) compileOptions() *protocols.ExecuterOptions {
	return &protocols.ExecuterOptions{
		Options: &protocols.Options{
			Timeout: int(e.config.Timeout.Seconds()),
		},
	}
}

func (e *Engine) compileTemplates(allTemplates []*templates.Template) []*templates.Template {
	compiledTemplates := make([]*templates.Template, 0, len(allTemplates))
	options := e.compileOptions()

	for _, t := range allTemplates {
		if err := t.Compile(options); err != nil {
			continue
		}
		compiledTemplates = append(compiledTemplates, t)
	}
	return compiledTemplates
}

// LoadByNames 按名称加载并编译 POC
// 会先检查缓存，缓存未命中则从远程加载
func (e *Engine) LoadByNames(ctx context.Context, names []string) ([]*templates.Template, error) {
	if len(names) == 0 {
		return nil, nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// 初始化索引（如果为空）
	if e.templateIndex == nil {
		e.templateIndex = make(map[string]*templates.Template)
	}

	var result []*templates.Template
	var missing []string

	// 检查缓存
	for _, name := range names {
		if t, ok := e.templateIndex[name]; ok {
			result = append(result, t)
		} else {
			missing = append(missing, name)
		}
	}

	// 如果全部命中缓存，直接返回
	if len(missing) == 0 {
		return result, nil
	}

	// 从远程加载缺失的 POC
	if e.client == nil {
		return result, fmt.Errorf("remote client not configured, cannot load POCs: %v", missing)
	}

	responses, err := e.client.ExportPOCsByNames(ctx, missing)
	if err != nil {
		return result, fmt.Errorf("failed to fetch POCs by names: %w", err)
	}

	// 编译并添加到缓存
	options := e.compileOptions()
	for _, resp := range responses {
		t := resp.GetTemplate()
		if err := t.Compile(options); err != nil {
			continue
		}
		result = append(result, t)

		// 添加到缓存
		e.templates = append(e.templates, t)
		if t.Id != "" {
			e.templateIndex[t.Id] = t
		}
	}

	return result, nil
}

// GetByName 从缓存获取或远程加载单个 POC
func (e *Engine) GetByName(ctx context.Context, name string) (*templates.Template, error) {
	e.mu.RLock()
	if t, ok := e.templateIndex[name]; ok {
		e.mu.RUnlock()
		return t, nil
	}
	e.mu.RUnlock()

	// 缓存未命中，远程加载
	loaded, err := e.LoadByNames(ctx, []string{name})
	if err != nil {
		return nil, err
	}
	if len(loaded) == 0 {
		return nil, fmt.Errorf("POC not found: %s", name)
	}

	return loaded[0], nil
}

// GetFromCache 从缓存获取 POC（不触发远程加载）
func (e *Engine) GetFromCache(name string) *templates.Template {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.templateIndex == nil {
		return nil
	}
	return e.templateIndex[name]
}

// HasInCache 检查 POC 是否在缓存中
func (e *Engine) HasInCache(name string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.templateIndex == nil {
		return false
	}
	_, ok := e.templateIndex[name]
	return ok
}
