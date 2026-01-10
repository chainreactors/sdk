package neutron

import (
	"context"
	"fmt"
	"time"

	"github.com/chainreactors/neutron/operators"
	"github.com/chainreactors/neutron/templates"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/sdk/pkg/cyberhub"
)

// ========================================
// Config 配置
// ========================================

// Config Neutron SDK 配置
type Config struct {
	cyberhub.Config

	// 加载配置
	LocalPath string
	Templates []*templates.Template
}

// ========================================
// Context 实现
// ========================================

// Context Neutron 上下文
type Context struct {
	ctx context.Context
}

// NewContext 创建 Neutron 上下文
func NewContext() *Context {
	return &Context{
		ctx: context.Background(),
	}
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) WithTimeout(timeout time.Duration) sdk.Context {
	ctx, _ := context.WithTimeout(c.ctx, timeout)
	return &Context{
		ctx: ctx,
	}
}

func (c *Context) WithCancel() (sdk.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	return &Context{
		ctx: ctx,
	}, cancel
}

// ========================================
// Result 实现
// ========================================

// ExecuteResult POC 执行结果
type ExecuteResult struct {
	success  bool
	err      error
	template *templates.Template
	result   *operators.Result
}

func (r *ExecuteResult) Success() bool {
	return r.success
}

func (r *ExecuteResult) Error() error {
	return r.err
}

func (r *ExecuteResult) Data() interface{} {
	return r.result
}

// Template 返回执行的模板
func (r *ExecuteResult) Template() *templates.Template {
	return r.template
}

// Result 返回执行结果
func (r *ExecuteResult) Result() *operators.Result {
	return r.result
}

// Matched 是否命中
func (r *ExecuteResult) Matched() bool {
	return r.result != nil && r.result.Matched
}

// ========================================
// Task 实现
// ========================================

// ExecuteTask 执行任务
type ExecuteTask struct {
	Target    string
	Templates []*templates.Template
	Payload   map[string]interface{}
}

// NewExecuteTask 创建执行任务
func NewExecuteTask(target string) *ExecuteTask {
	return &ExecuteTask{Target: target}
}

func (t *ExecuteTask) Type() string {
	return "execute"
}

func (t *ExecuteTask) Validate() error {
	if t.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}
	if t.Templates != nil && len(t.Templates) == 0 {
		return fmt.Errorf("templates cannot be empty")
	}
	return nil
}
