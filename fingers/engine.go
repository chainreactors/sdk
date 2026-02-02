package fingers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	fingersLib "github.com/chainreactors/fingers"
	"github.com/chainreactors/fingers/alias"
	"github.com/chainreactors/fingers/common"
	"github.com/chainreactors/fingers/favicon"
	fingersEngine "github.com/chainreactors/fingers/fingers"
	"github.com/chainreactors/fingers/resources"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/utils/httputils"
)

// ========================================
// Engine - 统一的指纹引擎
// ========================================

// Engine 是对 fingers 库的封装，支持多种数据源加载
type Engine struct {
	engine  *fingersLib.Engine
	config  *Config
	aliases []*alias.Alias // 原始别名数据
}

// NewEngine 创建一个新的 Engine 实例
// 根据 config 自动选择加载方式（本地/远程）
func NewEngine(config *Config) (*Engine, error) {
	if config == nil {
		config = NewConfig()
	}

	// 尝试加载配置，如果失败则创建空引擎
	if err := config.Load(context.Background()); err != nil {
		// 返回空引擎，允许后续配置
		return &Engine{
			config: config,
			engine: nil,
		}, nil
	}

	fingers := config.FullFingers.Fingers()
	if len(fingers) == 0 {
		// 返回空引擎，允许后续配置
		return &Engine{
			config: config,
			engine: nil,
		}, nil
	}

	e := &Engine{
		config: config,
	}

	engine, err := buildEngineFromFingers(fingers, config.FullFingers.Aliases())
	if err != nil {
		return nil, err
	}

	e.aliases = config.FullFingers.Aliases()
	e.engine = engine

	return e, nil
}

// NewEngineWithFingers creates an Engine using FullFingers directly.
func NewEngineWithFingers(fingers FullFingers) (*Engine, error) {
	if fingers.Len() == 0 {
		return nil, fmt.Errorf("fingers data is empty")
	}

	config := NewConfig()
	config.FullFingers = fingers

	engine, err := buildEngineFromFingers(fingers.Fingers(), fingers.Aliases())
	if err != nil {
		return nil, err
	}

	return &Engine{
		engine:  engine,
		config:  config,
		aliases: fingers.Aliases(),
	}, nil
}

// ========================================
// 统一 API - 只提供一种加载方式
// ========================================

// Get 获取底层的 fingers.Engine
func (e *Engine) Get() *fingersLib.Engine {
	return e.engine
}

// GetFingersEngine 获取 FingersEngine（用于 gogo 集成）
func (e *Engine) GetFingersEngine() (*fingersEngine.FingersEngine, error) {
	if e.engine == nil {
		// 返回 nil，允许引擎在未配置时也能使用
		return nil, nil
	}

	impl := e.engine.GetEngine("fingers")
	if impl == nil {
		return nil, nil
	}

	return impl.(*fingersEngine.FingersEngine), nil
}

// Reload 重新加载指纹
func (e *Engine) Reload(ctx context.Context) error {
	if e.config == nil {
		return fmt.Errorf("config is nil")
	}
	if err := e.config.Load(ctx); err != nil {
		return err
	}

	engine, err := buildEngineFromFingers(e.config.FullFingers.Fingers(), e.config.FullFingers.Aliases())
	if err != nil {
		return err
	}

	e.aliases = e.config.FullFingers.Aliases()
	e.engine = engine
	return nil
}

// buildEngineFromFingers 从指纹列表构建引擎
func buildEngineFromFingers(fingers fingersEngine.Fingers, aliases []*alias.Alias) (*fingersLib.Engine, error) {
	engine := &fingersLib.Engine{
		EnginesImpl:  make(map[string]fingersLib.EngineImpl),
		Enabled:      make(map[string]bool),
		Capabilities: make(map[string]common.EngineCapability),
	}

	// 初始化 Favicon 引擎（Compile 需要）
	faviconEngine := favicon.NewFavicons()
	engine.EnginesImpl["favicon"] = faviconEngine
	engine.Capabilities["favicon"] = faviconEngine.Capability()

	var httpFingers, socketFingers fingersEngine.Fingers
	for _, finger := range fingers {
		if finger.Protocol == "http" {
			httpFingers = append(httpFingers, finger)
		} else if finger.Protocol == "tcp" {
			socketFingers = append(socketFingers, finger)
		}
	}
	_, err := resources.LoadPorts()
	if err != nil {
		return nil, err
	}
	fEngine, err := fingersEngine.NewEngine(httpFingers, socketFingers)
	if err != nil {
		return nil, err
	}

	engine.Register(fEngine)

	if len(aliases) > 0 {
		aliasEngine, err := alias.NewAliases(aliases...)
		if err == nil {
			engine.Aliases = aliasEngine
		}
	}

	engine.Compile()
	return engine, nil
}

// Count 获取指纹总数
func (e *Engine) Count() int {
	if e.config == nil {
		return 0
	}
	return len(e.config.FullFingers.Fingers())
}

// Close 关闭引擎
func (e *Engine) Close() error {
	return nil
}

// ========================================
// 核心匹配 API - 原子化设计
// ========================================

// Match 匹配单个 HTTP 响应原始数据（被动指纹识别 - Level 0）
func (e *Engine) Match(data []byte) (common.Frameworks, error) {
	if e.engine == nil {
		// 返回空结果，允许引擎在未配置时也能使用
		return nil, nil
	}
	return e.engine.DetectContent(data)
}

// MatchFavicon 匹配 Favicon 指纹（被动识别）
// 参数:
//   - data: favicon 图标的原始字节数据
// 返回:
//   - common.Frameworks: 匹配到的指纹列表
//   - error: 错误信息
func (e *Engine) MatchFavicon(data []byte) (common.Frameworks, error) {
	if e.engine == nil {
		// 返回空结果，允许引擎在未配置时也能使用
		return nil, nil
	}
	return e.engine.MatchFavicon(data), nil
}

// MatchHTTP 匹配 HTTP 响应指纹（被动识别）
// 参数:
//   - resp: HTTP 响应对象
// 返回:
//   - common.Frameworks: 匹配到的指纹列表
//   - error: 错误信息
func (e *Engine) MatchHTTP(resp *http.Response) (common.Frameworks, error) {
	if e.engine == nil {
		// 返回空结果，允许引擎在未配置时也能使用
		return nil, nil
	}

	// 读取响应原始数据
	data := httputils.ReadRaw(resp)

	// 调用底层引擎进行匹配
	return e.engine.DetectContent(data)
}

// HTTPMatch HTTP/HTTPS 主动探测指纹识别（批量同步版本）
// 参数:
//   - ctx: 上下文（包含 timeout、level 等配置）
//   - urls: 目标URL列表（如 []string{"http://example.com", "https://example.com:8080"}）
// 返回:
//   - []*TargetResult: 每个目标的探测结果
//   - error: 错误信息（仅在引擎初始化失败等严重错误时返回）
func (e *Engine) HTTPMatch(ctx *Context, urls []string) ([]*TargetResult, error) {
	// 调用流式版本
	resultCh, err := e.HTTPMatchStream(ctx, urls)
	if err != nil {
		return nil, err
	}

	// 收集所有结果
	var results []*TargetResult
	for result := range resultCh {
		results = append(results, result)
	}

	return results, nil
}

// scanHTTPTarget 扫描单个 HTTP 目标（内部方法）
func (e *Engine) scanHTTPTarget(ctx *Context, url string, level int, client *http.Client, fEngine *fingersEngine.FingersEngine) *TargetResult {
	result := &TargetResult{
		Target: url,
	}

	// 解析 URL 获取 base URL (scheme://host:port)
	parsedURL, err := parseURL(url)
	if err != nil {
		result.Err = fmt.Errorf("invalid url: %w", err)
		return result
	}
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host
	if parsedURL.Port != "" && parsedURL.Port != "80" && parsedURL.Port != "443" {
		baseURL += ":" + parsedURL.Port
	}

	// 创建 sender 函数
	sender := fingersEngine.Sender(func(data []byte) ([]byte, bool) {
		sendPath := string(data)
		// 使用 pathJoin 连接基础路径和发送路径
		fullPath := pathJoin(parsedURL.Path, sendPath)
		fullURL := baseURL + fullPath

		req, err := http.NewRequest(http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, false
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		resp, err := client.Do(req)
		if err != nil {
			return nil, false
		}
		defer resp.Body.Close()

		return httputils.ReadRaw(resp), true
	})

	// 执行主动探测
	for _, finger := range fEngine.HTTPFingers {
		frame, vuln, ok := finger.ActiveMatch(level, sender)
		if ok && frame != nil {
			result.Results = append(result.Results, &common.ServiceResult{
				Framework: frame,
				Vuln:      vuln,
			})
		}
	}

	return result
}

// scanServiceTarget 扫描单个 Service 目标（内部方法）
func (e *Engine) scanServiceTarget(ctx *Context, target string, level int) *TargetResult {
	result := &TargetResult{
		Target: target,
	}

	// 解析target获取host和port
	host, port, err := parseTarget(target)
	if err != nil {
		result.Err = fmt.Errorf("invalid target: %w", err)
		return result
	}

	// 从 Context 获取 timeout
	timeout := ctx.GetTimeout()
	if timeout <= 0 {
		timeout = 10 // 默认10秒
	}

	// 创建默认的 ServiceSender
	sender := common.NewServiceSender(time.Duration(timeout) * time.Second)

	// 执行主动探测
	serviceResults, err := e.engine.DetectService(host, port, level, sender, nil)
	if err != nil {
		result.Err = err
		return result
	}

	result.Results = serviceResults
	return result
}

// ServiceMatch 通用服务主动探测指纹识别（批量同步版本）
// 参数:
//   - ctx: 上下文（包含 timeout、level 等配置）
//   - targets: 目标地址列表（格式：ip:port 或 host:port，如 []string{"192.168.1.1:80", "example.com:443"}）
// 返回:
//   - []*TargetResult: 每个目标的探测结果
//   - error: 错误信息（仅在引擎初始化失败等严重错误时返回）
func (e *Engine) ServiceMatch(ctx *Context, targets []string) ([]*TargetResult, error) {
	// 调用流式版本
	resultCh, err := e.ServiceMatchStream(ctx, targets)
	if err != nil {
		return nil, err
	}

	// 收集所有结果
	var results []*TargetResult
	for result := range resultCh {
		results = append(results, result)
	}

	return results, nil
}

// HTTPMatchStream HTTP/HTTPS 主动探测指纹识别（批量流式版本）
// 参数:
//   - ctx: 上下文（包含 timeout、level 等配置）
//   - urls: 目标URL列表（如 []string{"http://example.com", "https://example.com:8080"}）
// 返回:
//   - <-chan *TargetResult: 结果 channel（每个目标扫描完成后立即发送）
//   - error: 错误信息（仅在引擎初始化失败等严重错误时返回）
func (e *Engine) HTTPMatchStream(ctx *Context, urls []string) (<-chan *TargetResult, error) {
	if e.engine == nil {
		// 返回空 channel，允许引擎在未配置时也能使用
		ch := make(chan *TargetResult)
		close(ch)
		return ch, nil
	}

	// 从 Context 获取 level (HTTP: 0-3)
	level := ctx.GetLevel()
	if level < 0 || level > 3 {
		return nil, fmt.Errorf("invalid level: %d, must be 0-3", level)
	}

	// 获取 FingersEngine
	fEngine, err := e.GetFingersEngine()
	if err != nil {
		return nil, err
	}
	if fEngine == nil || fEngine.HTTPFingers == nil {
		return nil, fmt.Errorf("http fingers not initialized")
	}

	// 从 Context 获取 HTTP 客户端
	client := ctx.GetClient()

	// 创建结果 channel
	resultCh := make(chan *TargetResult)

	// 在 goroutine 中执行批量探测
	go func() {
		defer close(resultCh)

		for _, url := range urls {
			// 扫描单个目标
			targetResult := e.scanHTTPTarget(ctx, url, level, client, fEngine)

			// 发送结果到 channel
			select {
			case resultCh <- targetResult:
			case <-ctx.Context().Done():
				return
			}
		}
	}()

	return resultCh, nil
}

// ServiceMatchStream 通用服务主动探测指纹识别（流式版本）
// 参数:
//   - ctx: 上下文（包含 timeout、level 等配置）
//   - target: 目标地址（格式：ip:port 或 host:port，如 "192.168.1.1:80", "example.com:443"）
// 返回:
//   - <-chan *common.ServiceResult: 结果 channel
//   - error: 错误信息
// ServiceMatchStream 通用服务主动探测指纹识别（批量流式版本）
// 参数:
//   - ctx: 上下文（包含 timeout、level 等配置）
//   - targets: 目标地址列表（格式：ip:port 或 host:port，如 []string{"192.168.1.1:80", "example.com:443"}）
// 返回:
//   - <-chan *TargetResult: 结果 channel（每个目标扫描完成后立即发送）
//   - error: 错误信息（仅在引擎初始化失败等严重错误时返回）
func (e *Engine) ServiceMatchStream(ctx *Context, targets []string) (<-chan *TargetResult, error) {
	if e.engine == nil {
		// 返回空 channel，允许引擎在未配置时也能使用
		ch := make(chan *TargetResult)
		close(ch)
		return ch, nil
	}

	// 从 Context 获取 level (Service: 0-9)
	level := ctx.GetLevel()
	if level < 0 || level > 9 {
		return nil, fmt.Errorf("invalid level: %d, must be 0-9", level)
	}

	// 创建结果 channel
	resultCh := make(chan *TargetResult)

	// 在 goroutine 中执行批量探测
	go func() {
		defer close(resultCh)

		for _, target := range targets {
			// 扫描单个目标
			targetResult := e.scanServiceTarget(ctx, target, level)

			// 发送结果到 channel
			select {
			case resultCh <- targetResult:
			case <-ctx.Context().Done():
				return
			}
		}
	}()

	return resultCh, nil
}

// ========================================
// SDK Engine 接口实现（可选）
// ========================================

// Name 返回引擎名称（实现 sdk.Engine 接口）
func (e *Engine) Name() string {
	return "fingers"
}

// Execute 执行任务（实现 sdk.Engine 接口）
func (e *Engine) Execute(ctx sdk.Context, task sdk.Task) (<-chan sdk.Result, error) {
	// 确保引擎已初始化
	if e.engine == nil {
		// 返回空 channel，允许引擎在未配置时也能使用
		ch := make(chan sdk.Result)
		close(ch)
		return ch, nil
	}

	// 验证任务
	if err := task.Validate(); err != nil {
		return nil, err
	}

	// 只支持 MatchTask
	matchTask, ok := task.(*MatchTask)
	if !ok {
		return nil, fmt.Errorf("unsupported task type: %s", task.Type())
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

	return e.executeMatch(runCtx, matchTask)
}

// executeMatch 执行单个指纹匹配任务
func (e *Engine) executeMatch(ctx *Context, task *MatchTask) (<-chan sdk.Result, error) {
	resultCh := make(chan sdk.Result, 1)

	go func() {
		defer close(resultCh)

		frameworks, err := e.Match(task.Data)

		// 发送结果
		select {
		case resultCh <- &MatchResult{
			success:    err == nil,
			err:        err,
			frameworks: frameworks,
		}:
		case <-ctx.Context().Done():
		}
	}()

	return resultCh, nil
}

// ========================================
// 辅助函数
// ========================================

// parseURL 解析URL，提取scheme、host、port和path
type parsedURL struct {
	Scheme string
	Host   string
	Port   string
	Path   string
}

func parseURL(rawURL string) (*parsedURL, error) {
	// 确保URL有协议前缀
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	host := u.Hostname()
	port := u.Port()

	// 如果没有指定端口，根据协议使用默认端口
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	return &parsedURL{
		Scheme: u.Scheme,
		Host:   host,
		Port:   port,
		Path:   u.Path,
	}, nil
}

// parseTarget 解析target（ip:port或host:port格式），提取host和port
func parseTarget(target string) (host, port string, err error) {
	// 使用net.SplitHostPort解析
	host, port, err = net.SplitHostPort(target)
	if err != nil {
		return "", "", fmt.Errorf("invalid target format, expected 'host:port': %w", err)
	}

	if host == "" {
		return "", "", fmt.Errorf("host cannot be empty")
	}

	if port == "" {
		return "", "", fmt.Errorf("port cannot be empty")
	}

	return host, port, nil
}

// pathJoin 连接两个URL路径
// 参数:
//   - base: 基础路径（如 "", "/", "/aaa", "/aaa/"）
//   - append: 要追加的路径（如 "/nacos/"）
// 返回:
//   - 连接后的路径
// 示例:
//   - pathJoin("", "/nacos/") → "/nacos/"
//   - pathJoin("/", "/nacos/") → "/nacos/"
//   - pathJoin("/aaa", "/nacos/") → "/aaa/nacos/"
//   - pathJoin("/aaa/", "/nacos/") → "/aaa/nacos/"
func pathJoin(base, append string) string {
	// 去除 base 的尾部斜杠
	base = strings.TrimSuffix(base, "/")

	// 如果 base 为空，直接返回 append
	if base == "" {
		return append
	}

	// 确保 append 以 / 开头
	if !strings.HasPrefix(append, "/") {
		append = "/" + append
	}

	return base + append
}
