package fingers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/chainreactors/fingers/common"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/utils/httputils"
)

// ========================================
// Context 实现
// ========================================

// Context Fingers 上下文
type Context struct {
	ctx           context.Context
	httpSender    HTTPSender
	client        *http.Client // 用户自定义的 HTTP 客户端
	defaultClient *http.Client // 默认 HTTP 客户端（根据 timeout/proxy 自动构建）
	timeout       int          // 超时时间（秒）
	proxy         string       // 代理地址（如 "socks5://127.0.0.1:1080"）
	level         int          // 探测级别（0=被动, 1=基础, 2=深度）
}

var _ sdk.Context = (*Context)(nil)

// NewContext 创建 Fingers 上下文
func NewContext() *Context {
	return &Context{
		ctx:     context.Background(),
		timeout: 10, // 默认10秒超时
		level:   1,  // 默认 level 1（基础探测）
	}
}

// WithContext 基于给定的 context.Context 复制 Context
func (c *Context) WithContext(ctx context.Context) *Context {
	return &Context{
		ctx:           ctx,
		httpSender:    c.httpSender,
		client:        c.client,
		defaultClient: c.defaultClient,
		timeout:       c.timeout,
		proxy:         c.proxy,
		level:         c.level,
	}
}

func (c *Context) Context() context.Context {
	return c.ctx
}

// WithTimeout 设置超时时间（秒）
func (c *Context) WithTimeout(timeout int) *Context {
	c.timeout = timeout
	c.buildDefaultClient() // 重建默认客户端以应用新配置
	return c
}

// WithProxy 设置代理地址
// 支持的格式：
//   - socks5://127.0.0.1:1080
//   - socks4://127.0.0.1:1080
//   - http://127.0.0.1:8080
//   - https://127.0.0.1:8080
func (c *Context) WithProxy(proxy string) *Context {
	c.proxy = proxy
	c.buildDefaultClient() // 重建默认客户端以应用新配置
	return c
}

// WithHTTPSender 设置自定义HTTPSender
func (c *Context) WithHTTPSender(sender HTTPSender) *Context {
	c.httpSender = sender
	return c
}

// WithClient 设置自定义HTTP客户端
func (c *Context) WithClient(client *http.Client) *Context {
	c.client = client
	return c
}

// GetHTTPSender 获取HTTPSender，如果未设置则返回默认实现
func (c *Context) GetHTTPSender() HTTPSender {
	if c.httpSender == nil {
		return NewDefaultHTTPSender(time.Duration(c.timeout)*time.Second, c.proxy)
	}
	return c.httpSender
}

// GetClient 获取HTTP客户端，如果未设置则返回默认客户端
func (c *Context) GetClient() *http.Client {
	// 优先返回用户自定义的客户端
	if c.client != nil {
		return c.client
	}

	// 如果默认客户端未初始化，则构建它
	if c.defaultClient == nil {
		c.buildDefaultClient()
	}

	return c.defaultClient
}

// buildDefaultClient 构建默认HTTP客户端
// 根据当前的 timeout 和 proxy 配置创建客户端
func (c *Context) buildDefaultClient() {
	// 设置超时时间
	timeout := time.Duration(c.timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	// 创建基础 Transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// TODO: 如果需要支持 proxy，可以在这里配置 transport.Proxy
	// 目前 proxy 主要用于 HTTPSender，HTTP Client 暂不处理 proxy
	// 如果用户需要 proxy，建议使用 WithClient() 提供自定义客户端

	// 创建并存储默认客户端
	c.defaultClient = &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// GetTimeout 获取超时时间（秒）
func (c *Context) GetTimeout() int {
	return c.timeout
}

// GetProxy 获取代理地址
func (c *Context) GetProxy() string {
	return c.proxy
}

// WithLevel 设置探测级别
// 参数:
//   - level: 探测级别（0=被动, 1=基础, 2=深度）
func (c *Context) WithLevel(level int) *Context {
	c.level = level
	return c
}

// GetLevel 获取探测级别
func (c *Context) GetLevel() int {
	return c.level
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

// TargetResult 目标扫描结果（用于批量扫描）
type TargetResult struct {
	Target  string                  // 扫描的目标 URL 或 target
	Results []*common.ServiceResult // 指纹识别结果
	Error   error                   // 错误信息（如果有）
}

// Success 是否成功（无错误）
func (r *TargetResult) Success() bool {
	return r.Error == nil
}

// HasResults 是否有匹配结果
func (r *TargetResult) HasResults() bool {
	return len(r.Results) > 0
}

// ========================================
// Task 实现
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
