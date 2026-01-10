package neutron

import (
	"context"
	"fmt"

	"github.com/chainreactors/neutron/protocols"
	"github.com/chainreactors/neutron/templates"
	sdk "github.com/chainreactors/sdk/pkg"
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
	if err := config.Load(context.Background()); err != nil {
		return nil, err
	}
	templates := config.Templates.Templates()
	if len(templates) == 0 {
		return nil, fmt.Errorf("templates data is empty")
	}

	e := &Engine{
		config: config,
	}

	e.templates = e.compileTemplates(templates)

	return e, nil
}

// NewEngineWithTemplates creates an Engine using Templates directly.
func NewEngineWithTemplates(templates Templates) (*Engine, error) {
	if templates.Len() == 0 {
		return nil, fmt.Errorf("templates data is empty")
	}

	config := NewConfig()
	config.Templates = templates

	e := &Engine{
		config: config,
	}

	e.templates = e.compileTemplates(templates.Templates())
	return e, nil
}

// Name 返回引擎名称（实现 sdk.Engine 接口）
func (e *Engine) Name() string {
	return "neutron"
}

// Execute 执行任务（实现 sdk.Engine 接口）
func (e *Engine) Execute(ctx sdk.Context, task sdk.Task) (<-chan sdk.Result, error) {
	if e == nil {
		return nil, fmt.Errorf("neutron engine is nil")
	}
	if err := task.Validate(); err != nil {
		return nil, err
	}

	execTask, ok := task.(*ExecuteTask)
	if !ok {
		return nil, fmt.Errorf("unsupported task type: %s", task.Type())
	}

	templates := execTask.Templates
	if templates == nil {
		templates = e.templates
	}
	if len(templates) == 0 {
		return nil, fmt.Errorf("templates are empty")
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

	return e.executeTemplates(runCtx, templates, execTask.Target, execTask.Payload)
}

func (e *Engine) executeTemplates(ctx *Context, templates []*templates.Template, target string, payload map[string]interface{}) (<-chan sdk.Result, error) {
	resultCh := make(chan sdk.Result)

	go func() {
		defer close(resultCh)

		for _, t := range templates {
			result, err := t.Execute(target, payload)
			if err != nil {
				if err == protocols.OpsecError {
					continue
				}
				select {
				case resultCh <- &ExecuteResult{
					success:  false,
					err:      err,
					template: t,
					result:   result,
				}:
				case <-ctx.Context().Done():
					return
				}
				continue
			}

			select {
			case resultCh <- &ExecuteResult{
				success:  true,
				err:      nil,
				template: t,
				result:   result,
			}:
			case <-ctx.Context().Done():
				return
			}
		}
	}()

	return resultCh, nil
}

// Get 获取已加载的 templates
func (e *Engine) Get() []*templates.Template {
	return e.templates
}

// Count 获取已加载的 template 数量
func (e *Engine) Count() int {
	return len(e.templates)
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
