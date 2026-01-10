package neutron

import (
	"context"
	"fmt"

	"github.com/chainreactors/neutron/protocols"
	"github.com/chainreactors/neutron/templates"
)

// ========================================
// Engine - Neutron 加载引擎
// ========================================

// Engine Neutron 加载引擎，支持本地和远程数据源
type Engine struct {
	templates []*templates.Template
	config    *Config
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
	if e.templates != nil {
		return e.templates, nil
	}

	if e.config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if err := e.config.Load(ctx); err != nil {
		return nil, err
	}
	if e.config == nil || len(e.config.Templates) == 0 {
		return nil, fmt.Errorf("templates data is empty")
	}

	// 编译所有加载的 templates
	compiledTemplates := e.compileTemplates(e.config.Templates)

	e.templates = compiledTemplates

	return compiledTemplates, nil
}

// Get 获取已加载的 templates
func (e *Engine) Get() []*templates.Template {
	return e.templates
}

// Count 获取已加载的 template 数量
func (e *Engine) Count() int {
	return len(e.templates)
}

// Reload 重新加载 templates
func (e *Engine) Reload(ctx context.Context) error {
	e.templates = nil
	_, err := e.Load(ctx)
	return err
}

// Close 关闭引擎
func (e *Engine) Close() error {
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
