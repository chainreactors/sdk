package spray

import (
	"github.com/chainreactors/spray/core"
)

// NewDefaultOption 创建并返回一个带有默认值且已初始化的 Runner 配置
// 这个函数统一处理所有的默认配置和基础初始化，对外部 SDK 隐藏内部细节
// 返回的 Option 已经完成了 pkg.Load() 等基础初始化

type Option struct {
	*core.Option
}

func NewDefaultOption() *Option {
	opt := &Option{Option: &core.Option{}}

	// Request 配置
	opt.Method = "GET"
	opt.PortRange = "80,443"
	opt.MaxBodyLength = 100
	opt.RandomUserAgent = false

	// Status 配置
	opt.BlackStatus = "400,410"
	opt.WhiteStatus = "200"
	opt.FuzzyStatus = "500,501,502,503,301,302,404"
	opt.UniqueStatus = "403,200,404"

	// 检查配置
	opt.CheckPeriod = 200
	opt.ErrPeriod = 10
	opt.BreakThreshold = 20

	// 递归配置
	opt.Recursive = "current.IsDir()"
	opt.Depth = 0
	opt.Index = "/"
	opt.Random = ""

	// 重试配置
	opt.RetryCount = 0
	opt.SimhashDistance = 8

	// 运行模式配置
	opt.Mod = "path"
	opt.Client = "auto"
	opt.Timeout = 5
	opt.Threads = 20
	opt.PoolSize = 5
	opt.Deadline = 999999

	// 输出配置 (SDK 模式下默认静默)
	opt.Quiet = true
	opt.NoBar = true
	opt.NoStat = true
	opt.NoColor = false
	opt.Json = false
	opt.FileOutput = "json"

	// 插件配置
	opt.Advance = false
	opt.Finger = false
	opt.CrawlPlugin = false
	opt.BakPlugin = false
	opt.FuzzuliPlugin = false
	opt.CommonPlugin = false
	opt.ActivePlugin = false
	opt.ReconPlugin = false
	opt.CrawlDepth = 3
	opt.AppendDepth = 2

	// 指纹引擎配置
	opt.FingerEngines = "all"

	return opt
}

func cloneOption(opt *Option) *Option {
	if opt == nil || opt.Option == nil {
		return NewDefaultOption()
	}
	coreOpt := *opt.Option
	clone := &Option{Option: &coreOpt}
	clone.URL = cloneStrings(opt.URL)
	clone.CIDRs = cloneStrings(opt.CIDRs)
	clone.Dictionaries = cloneStrings(opt.Dictionaries)
	clone.Rules = cloneStrings(opt.Rules)
	clone.AppendRule = cloneStrings(opt.AppendRule)
	clone.AppendFile = cloneStrings(opt.AppendFile)
	clone.Prefixes = cloneStrings(opt.Prefixes)
	clone.Suffixes = cloneStrings(opt.Suffixes)
	clone.Replaces = cloneStringMap(opt.Replaces)
	clone.Skips = cloneStrings(opt.Skips)
	clone.Headers = cloneStrings(opt.Headers)
	clone.Cookie = cloneStrings(opt.Cookie)
	clone.Extracts = cloneStrings(opt.Extracts)
	clone.Scope = cloneStrings(opt.Scope)
	clone.Verbose = append([]bool(nil), opt.Verbose...)
	clone.Proxies = cloneStrings(opt.Proxies)
	clone.FingerFiles = cloneStrings(opt.FingerFiles)
	return clone
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

// ========================================
// 链式配置方法 (With***)
// ========================================

// WithThreads 设置并发线程数
func (opt *Option) WithThreads(n int) *Option {
	opt.Threads = n
	return opt
}

// WithTimeout 设置请求超时时间（秒）
func (opt *Option) WithTimeout(n int) *Option {
	opt.Timeout = n
	return opt
}

// WithMethod 设置 HTTP 请求方法
func (opt *Option) WithMethod(method string) *Option {
	opt.Method = method
	return opt
}

// WithHeaders 设置自定义请求头
func (opt *Option) WithHeaders(headers []string) *Option {
	opt.Headers = headers
	return opt
}

// WithProxy 设置代理
func (opt *Option) WithProxy(proxy string) *Option {
	opt.Proxies = []string{proxy}
	return opt
}

// WithFinger 启用/禁用指纹识别
func (opt *Option) WithFinger(enable bool) *Option {
	opt.Finger = enable
	return opt
}

// WithCrawl 启用/禁用爬虫
func (opt *Option) WithCrawl(enable bool) *Option {
	opt.CrawlPlugin = enable
	return opt
}

// WithDepth 设置递归深度
func (opt *Option) WithDepth(depth int) *Option {
	opt.Depth = depth
	return opt
}

// WithMod 设置运行模式 (path/host/param)
func (opt *Option) WithMod(mod string) *Option {
	opt.Mod = mod
	return opt
}
