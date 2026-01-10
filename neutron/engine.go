package neutron

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	templates []*templates.Template
	config    *Config
	client    *cyberhub.Client // 仅在远程模式下使用
	mu        sync.RWMutex
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
		)
	}

	return e, nil
}

// Load 加载并返回 templates 列表
// config 为 nil 时使用默认本地配置
func Load(config *Config) ([]*templates.Template, error) {
	if config == nil {
		config = NewConfig()
	}

	engine, err := NewEngine(config)
	if err != nil {
		return nil, err
	}

	return engine.Load(context.Background())
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
	if e.config.Filename != "" {
		allTemplates, err = e.loadFromFile(e.config.Filename)
		if err != nil {
			return nil, err
		}
	}

	if allTemplates == nil && e.config.IsRemoteEnabled() {
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

func (e *Engine) loadFromFile(path string) ([]*templates.Template, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var rawTemplates []*templates.Template
	if err := yaml.NewDecoder(file).Decode(&rawTemplates); err != nil {
		return nil, fmt.Errorf("failed to decode templates: %w", err)
	}

	if len(rawTemplates) == 0 {
		return nil, fmt.Errorf("no templates loaded from %s", path)
	}

	return rawTemplates, nil
}

// loadFromRemote 从 Cyberhub 加载 POC
func (e *Engine) loadFromRemote(ctx context.Context) ([]*templates.Template, error) {
	return e.loadRemoteTemplates(ctx)
}

func (e *Engine) loadRemoteTemplates(ctx context.Context) ([]*templates.Template, error) {
	var filter *cyberhub.ExportFilter
	if e.config != nil {
		filter = e.config.ExportFilter
	}

	responses, err := e.client.ExportPOCs(ctx, nil, nil, "", "", filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch POCs from cyberhub: %w", err)
	}

	var loadedTemplates []*templates.Template
	for _, resp := range responses {
		loadedTemplates = append(loadedTemplates, resp.GetTemplate())
	}

	return loadedTemplates, nil
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

// Close 关闭引擎
func (e *Engine) Close() error {
	if e.client != nil {
		return e.client.Close()
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
