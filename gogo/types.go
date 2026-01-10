package gogo

import (
	"context"
	"fmt"
	"time"

	"github.com/chainreactors/gogo/v2/pkg"
	sdkfingers "github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/neutron"
	sdk "github.com/chainreactors/sdk/pkg"
)

// ========================================
// Context 实现
// ========================================

// Context GoGo 上下文
type Context struct {
	ctx     context.Context
	threads int
	opt     *pkg.RunnerOption
}

// NewContext 创建 GoGo 上下文
func NewContext() *Context {
	return &Context{
		ctx:     context.Background(),
		threads: 1000,
		opt:     pkg.DefaultRunnerOption,
	}
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) WithTimeout(timeout time.Duration) sdk.Context {
	ctx, _ := context.WithTimeout(c.ctx, timeout)
	return &Context{
		ctx:     ctx,
		threads: c.threads,
		opt:     c.opt,
	}
}

func (c *Context) WithCancel() (sdk.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	return &Context{
		ctx:     ctx,
		threads: c.threads,
		opt:     c.opt,
	}, cancel
}

// SetThreads 设置线程数
func (c *Context) SetThreads(threads int) *Context {
	if threads > 0 {
		c.threads = threads
	}
	return c
}

// SetOption 设置运行选项
func (c *Context) SetOption(opt *pkg.RunnerOption) *Context {
	c.opt = opt
	return c
}

// SetVersionLevel 设置指纹识别级别
func (c *Context) SetVersionLevel(level int) *Context {
	if c.opt == nil {
		c.opt = pkg.DefaultRunnerOption
	}
	c.opt.VersionLevel = level
	return c
}

// SetExploit 设置漏洞检测模式
func (c *Context) SetExploit(exploit string) *Context {
	if c.opt == nil {
		c.opt = pkg.DefaultRunnerOption
	}
	c.opt.Exploit = exploit
	return c
}

// SetDelay 设置超时时间（秒）
func (c *Context) SetDelay(delay int) *Context {
	if c.opt == nil {
		c.opt = pkg.DefaultRunnerOption
	}
	c.opt.Delay = delay
	return c
}

// ========================================
// Config 实现
// ========================================

// Config GoGo 配置
type Config struct {
	FingersEngine *sdkfingers.Engine
	NeutronEngine *neutron.Engine
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Validate() error {
	return nil
}

// WithFingersEngine 设置自定义 fingers 引擎
func (c *Config) WithFingersEngine(engine *sdkfingers.Engine) *Config {
	c.FingersEngine = engine
	return c
}

// WithNeutronEngine 设置自定义 neutron 引擎
func (c *Config) WithNeutronEngine(engine *neutron.Engine) *Config {
	c.NeutronEngine = engine
	return c
}

// ========================================
// Task 实现
// ========================================

// ScanTask 扫描任务
type ScanTask struct {
	IP    string
	Ports string
}

// NewScanTask 创建扫描任务
func NewScanTask(ip, ports string) *ScanTask {
	return &ScanTask{IP: ip, Ports: ports}
}

func (t *ScanTask) Type() string {
	return "scan"
}

func (t *ScanTask) Validate() error {
	if t.IP == "" {
		return fmt.Errorf("IP cannot be empty")
	}
	if t.Ports == "" {
		return fmt.Errorf("Ports cannot be empty")
	}
	return nil
}

// WorkflowTask 工作流任务
type WorkflowTask struct {
	Workflow *pkg.Workflow
}

// NewWorkflowTask 创建工作流任务
func NewWorkflowTask(workflow *pkg.Workflow) *WorkflowTask {
	return &WorkflowTask{Workflow: workflow}
}

func (t *WorkflowTask) Type() string {
	return "workflow"
}

func (t *WorkflowTask) Validate() error {
	if t.Workflow == nil {
		return fmt.Errorf("Workflow cannot be nil")
	}
	return nil
}
