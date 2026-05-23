package neutron

import (
	"context"
	"fmt"

	"github.com/chainreactors/neutron/protocols"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/sdk/pkg/types"
)

// ========================================
// Engine - Neutron 加载引擎
// ========================================

// Engine Neutron 加载引擎，支持本地和远程数据源
type Engine struct {
	templates []*types.Template
	config    *Config
	capacity  *sdk.Capacity
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

	// 尝试加载配置，如果失败则创建空引擎
	if err := config.Load(context.Background()); err != nil {
		// 返回空引擎，允许后续配置
		return &Engine{
			config:    config,
			templates: nil,
		}, nil
	}

	tpls := config.Templates.Templates()
	if len(tpls) == 0 {
		// 返回空引擎，允许后续配置
		return &Engine{
			config:    config,
			templates: nil,
		}, nil
	}

	e := &Engine{
		config: config,
	}
	if config.Capacity > 0 {
		e.capacity = sdk.NewCapacity(config.Capacity)
	}

	e.templates = e.compileTemplates(tpls)

	return e, nil
}

// NewEngineWithTemplates creates an Engine using Templates directly.
func NewEngineWithTemplates(tpls Templates) (*Engine, error) {
	if tpls.Len() == 0 {
		return nil, fmt.Errorf("templates data is empty")
	}

	config := NewConfig()
	config.Templates = tpls

	e := &Engine{
		config: config,
	}

	e.templates = e.compileTemplates(tpls.Templates())
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
		// 返回空 channel，允许引擎在未配置时也能使用
		ch := make(chan sdk.Result)
		close(ch)
		return ch, nil
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

func (e *Engine) executeTemplates(ctx *Context, templates []*types.Template, target string, payload map[string]interface{}) (<-chan sdk.Result, error) {
	if e.capacity != nil {
		if err := e.capacity.Acquire(ctx.Context(), 1); err != nil {
			return nil, err
		}
	}

	resultCh := make(chan sdk.Result)

	go func() {
		defer close(resultCh)
		if e.capacity != nil {
			defer e.capacity.Release(1)
		}

		for _, t := range templates {
			result, events, err := t.ExecuteWithEvents(target, payload)
			if err != nil {
				if err == types.OpsecError {
					continue
				}
				select {
				case resultCh <- &ExecuteResult{
					success:  false,
					err:      err,
					template: t,
					data:     &NeutronResult{Result: result, Events: events},
				}:
				case <-ctx.Context().Done():
					return
				}
				continue
			}

			select {
			case resultCh <- &ExecuteResult{
				success:  true,
				template: t,
				data:     &NeutronResult{Result: result, Events: events},
			}:
			case <-ctx.Context().Done():
				return
			}
		}
	}()

	return resultCh, nil
}

// Get 获取已加载的 templates
func (e *Engine) Get() []*types.Template {
	return e.templates
}

// Count 获取已加载的 template 数量
func (e *Engine) Count() int {
	return len(e.templates)
}

// SetCapacity configures a capacity limit on an already-created engine.
// Subsequent Execute calls will acquire/release from this shared bucket.
func (e *Engine) SetCapacity(total int) {
	if total > 0 {
		e.capacity = sdk.NewCapacity(total)
	}
}

// Capacity returns the engine's capacity bucket, or nil if unconfigured.
func (e *Engine) Capacity() *sdk.Capacity {
	return e.capacity
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

func (e *Engine) compileTemplates(allTemplates []*types.Template) []*types.Template {
	compiledTemplates := make([]*types.Template, 0, len(allTemplates))
	options := e.compileOptions()

	for _, t := range allTemplates {
		if err := t.Compile(options); err != nil {
			continue
		}
		compiledTemplates = append(compiledTemplates, t)
	}
	return compiledTemplates
}
