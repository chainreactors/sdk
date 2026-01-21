package spray

import (
	"context"
	"fmt"

	sdkfingers "github.com/chainreactors/sdk/fingers"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/spray/core"
)

// ========================================
// Context 实现
// ========================================

// Context Spray 上下文
type Context struct {
	ctx context.Context
	opt *Option
}

var _ sdk.Context = (*Context)(nil)

// NewContext 创建 Spray 上下文
func NewContext() *Context {
	return &Context{
		ctx: context.Background(),
		opt: NewDefaultOption(),
	}
}

// WithContext 基于给定的 context.Context 复制 Context
func (c *Context) WithContext(ctx context.Context) *Context {
	return &Context{
		ctx: ctx,
		opt: c.opt,
	}
}

func (c *Context) Context() context.Context {
	return c.ctx
}

// SetThreads 设置线程数
func (c *Context) SetThreads(threads int) *Context {
	c.opt.Threads = threads
	return c
}

// SetTimeout 设置超时时间（秒）
func (c *Context) SetTimeout(timeout int) *Context {
	c.opt.Timeout = timeout
	return c
}

// SetMethod 设置 HTTP 方法
func (c *Context) SetMethod(method string) *Context {
	c.opt.Method = method
	return c
}

// SetHeaders 设置自定义请求头
func (c *Context) SetHeaders(headers []string) *Context {
	c.opt.Headers = headers
	return c
}

// SetFilter 设置过滤规则
func (c *Context) SetFilter(filter string) *Context {
	c.opt.Filter = filter
	return c
}

// SetMatch 设置匹配规则
func (c *Context) SetMatch(match string) *Context {
	c.opt.Match = match
	return c
}

// SetOption 设置完整选项
func (c *Context) SetOption(opt *core.Option) *Context {
	c.opt = &Option{opt}
	return c
}

// ========================================
// Plugin 配置方法
// ========================================

// SetAdvance 启用所有插件
func (c *Context) SetAdvance(enable bool) *Context {
	c.opt.Advance = enable
	return c
}

// SetActivePlugin 启用主动指纹路径插件
func (c *Context) SetActivePlugin(enable bool) *Context {
	c.opt.ActivePlugin = enable
	return c
}

// SetReconPlugin 启用信息提取插件
func (c *Context) SetReconPlugin(enable bool) *Context {
	c.opt.ReconPlugin = enable
	return c
}

// SetBakPlugin 启用备份文件发现插件
func (c *Context) SetBakPlugin(enable bool) *Context {
	c.opt.BakPlugin = enable
	return c
}

// SetFuzzuliPlugin 启用 Fuzzuli 插件
func (c *Context) SetFuzzuliPlugin(enable bool) *Context {
	c.opt.FuzzuliPlugin = enable
	return c
}

// SetCommonPlugin 启用常见文件发现插件
func (c *Context) SetCommonPlugin(enable bool) *Context {
	c.opt.CommonPlugin = enable
	return c
}

// SetCrawlPlugin 启用爬虫插件
func (c *Context) SetCrawlPlugin(enable bool) *Context {
	c.opt.CrawlPlugin = enable
	return c
}

// SetCrawlDepth 设置爬虫深度
func (c *Context) SetCrawlDepth(depth int) *Context {
	c.opt.CrawlDepth = depth
	return c
}

// SetFinger 启用主动指纹检测
func (c *Context) SetFinger(enable bool) *Context {
	c.opt.Finger = enable
	return c
}

// SetExtracts 设置信息提取规则
func (c *Context) SetExtracts(extracts []string) *Context {
	c.opt.Extracts = extracts
	return c
}

// SetRecursiveDepth 设置递归深度
func (c *Context) SetRecursiveDepth(depth int) *Context {
	c.opt.Depth = depth
	return c
}

// ========================================
// Config 实现
// ========================================

// Config Spray 配置
type Config struct {
	FingersEngine *sdkfingers.Engine
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

// ========================================
// Task 实现
// ========================================

// CheckTask URL 检测任务
type CheckTask struct {
	URLs []string
}

// NewCheckTask 创建 URL 检测任务
func NewCheckTask(urls []string) *CheckTask {
	return &CheckTask{URLs: urls}
}

func (t *CheckTask) Type() string {
	return "check"
}

func (t *CheckTask) Validate() error {
	if len(t.URLs) == 0 {
		return fmt.Errorf("URLs cannot be empty")
	}
	return nil
}

// BruteTask 暴力破解任务
type BruteTask struct {
	BaseURL  string
	Wordlist []string
}

// NewBruteTask 创建暴力破解任务
func NewBruteTask(baseURL string, wordlist []string) *BruteTask {
	return &BruteTask{
		BaseURL:  baseURL,
		Wordlist: wordlist,
	}
}

func (t *BruteTask) Type() string {
	return "brute"
}

func (t *BruteTask) Validate() error {
	if t.BaseURL == "" {
		return fmt.Errorf("BaseURL cannot be empty")
	}
	if len(t.Wordlist) == 0 {
		return fmt.Errorf("Wordlist cannot be empty")
	}
	return nil
}
