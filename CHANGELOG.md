# Changelog

## v0.3.2 (2026-06-15)

重构 httpx 为通用 client generator，升级 neutron/fingers 大幅提升主动指纹匹配的准确率。

### Breaking Changes

- **全引擎**: `Execute(nil, task)` 不再静默 fallback 到 `NewContext()`，改为返回 error。调用方必须显式传入 Context

### New Features

- **httpx**: 重构为通用 client generator，新增 `BrowserConfig()` 预设与 `WithTimeout`/`WithProxy`/`WithRedirects` builder 方法；`NewClient` 内聚 proxy fallback
- **fingers**: 主动探测新增跨 finger 的 `cachingSender` 请求缓存，相同探测路径只发一次 HTTP 请求
- **neutron**: per-context CookieJar——每次模板执行自动创建独立 CookieJar，redirect 链内 Set-Cookie 正确传递，不同执行间隔离（对齐 nuclei contextargs 模式）
- **neutron**: redirect 策略三态化（DontFollow / FollowAll / FollowSameHost），对齐 nuclei RedirectFlow 语义，修复一批 `redirects: false` 模板因默认跟随跳转而丢失 Location 断言的问题
- **neutron**: xray 转换器修正 `follow_redirects` 默认行为——xray 默认跟随（省略=true），neutron 默认不跟；同时自动检测依赖跳转响应（Location header / 3xx status code）的规则并保留 3xx 原始响应
- **fingers(lib)**: hub/xray 主动探测改用 `ExecuteWithTransport` 整模板执行，修复之前逐请求执行丢失 `__request_index_offset` 导致多请求模板 `body_N` 塌陷为 `body_1` 的漏匹配问题

### Bug Fixes

- **全引擎**: `emitStats` 在 context 取消后跳过回调，避免 send on closed channel panic
- **neutron**: favicon 探测增强——使用最终跳转后的 URL 做 base、增加无前缀 `favicon.ico` 探测路径、favicon fetch 使用独立 context 避免页面请求超时导致误判
- **neutron**: 修正 RootURL 挂载路径拼接，xray 被动匹配支持 `{{RootURL}}/` 路径

### Dependencies

- fingers `f5c144e` → `7e07a99`
- neutron `1a0a5a8` → `a9bbe4f`

## v0.3.1 (2026-06-12)

修复 xray 指纹引擎的模板路由和误报问题，新增统一主动探测 API。

### Bug Fixes

- **fingers**: CyberHub 平台使用 `xray` tag 标记模板，但 SDK 仅识别 `source1` tag，导致 5400+ xray 模板全部错误路由到 fingerprinthub 引擎。现在同时支持 `source1` 和 `xray` 两个路由 tag
- **neutron**: 转换后的 xray 多步模板在中间 request 上带有 `internal-matchers: true` 标志，但 neutron 的 `Request` 结构体无此字段，YAML 反序列化时被静默丢弃。中间 request 的通用 word matcher（如 `location.href`）被当作正常 matcher 执行，导致大量误报。新增 `InternalMatchers bool` 字段，当为 true 时 matcher 仍执行（保证 extractor 和 req-condition 数据正常），但不上报 `Matched=true`

### New Features

- **fingers**: 新增 `Engine.ActiveMatch(baseURL, level, transport)` 统一主动探测 API。调用方只需提供 URL、探测级别和自定义 `http.RoundTripper`，内部自动调度 native fingers、fingerprinthub、xray 三个引擎的 `HTTPActiveMatch`，无需感知底层引擎差异。内部 `scanHTTPTarget` 已重构为委托调用此 API

### Dependencies

- neutron `e112381` → `1a0a5a8`（internal-matchers 支持）

## v0.3.0 (2026-06-11)

本版本包含 **Breaking Changes**。核心变更：集成 proton 敏感信息扫描引擎，neutron 新增 SSL/TLS 协议和 JSON/XPath 匹配能力，全引擎 API 统一重命名。

### Breaking Changes

- **gogo/spray/zombie**: `GogoEngine`、`SprayEngine` 统一重命名为 `Engine`；`NewEngine` 签名统一返回 `(*Engine, error)`
- **proton**: `ProtonFinding` 重命名为 `ProtonResult`
- **types**: 移除 `pkg/types/registry.go`，各引擎 init 注册逻辑移除

### New Features

**proton 敏感信息扫描引擎（新引擎）**

- 集成 proton 文件扫描引擎到 SDK，支持本地文件/内存数据的敏感信息匹配（密码、密钥、凭证、连接字符串等）
- 提供 `ScanData(data, label)` 和 `ScanBlock(data, label)` 两种 API，分别面向文本数据和二进制数据块（进程内存、网络流）
- proton 类型统一收归 `pkg/types`，与 gogo/spray/zombie/neutron 风格一致

**neutron — SSL/TLS 协议 + JSON/XPath matcher**

- 新增纯 TLS 探测协议（标准库实现），支持 ssl/tls 协议槽位、cipher_suites 注入
- 新增 `common/tlsx` 共享包：标准库 TLS/证书提取、`IsUntrusted`/`IsRevoked` 判断、cfssl 撤销检查子模块可选启用
- cert DSL 字段扩展到 8 项 + `raw_cert`，`XrayCertFields` 单源注册表统一 http/ssl 双命名空间
- `ScanContext.PathPrefix` → `RootURL` 挂载路径前缀
- 新增 `operators/full` 子模块：JSON/XPath extractor 和 matcher 支持（gojq + antchfx），替代原 mini-jq 实现
- 新增 DSL `dir()` 函数
- Go 1.11 全面兼容修复（`io.NopCloser`/`strings.ReplaceAll`/`Duration.Milliseconds` 替换、build tags、testify 降级）

**fingers — xray 引擎 + AC 快速匹配**

- 新增 xray 指纹引擎，基于 xray POC 转换实现
- 新增 AC 关键词预过滤 + RE2 默认正则引擎，1MB 基准测试验证
- 新增 `templates.Load` 接口，xray POC 支持 opt-in 加载
- 引擎级 `CaseInsensitive` 开关统一大小写匹配策略
- TinyGo 兼容：模板 sanitizer + sender 适配

**其他**

- 新增 `ReviewStatus` 常量替代审核状态 magic string

### Bug Fixes

- **spray**: 修复 context 取消后 `emitStats` panic 的问题
- **proton**: 修复 `Template.Execute` 未遍历 Request path + 多行 opResult 合并错误
- **neutron**: 隔离模板编译变量，避免并发执行时变量污染
- **fingers**: 修复 AC fast path 丢失匹配详情、multi-request 模板 extractor 结果未串联、nil-map panic

### Refactor

- 移除未使用代码和冗余抽象（-284 行）
- 各引擎 `init.go` 注册逻辑移除，简化初始化流程
- **proton**: 架构重构为 pure matching engine SDK — 文件遍历移出 scanner，Runner 公开扫描阶段 API，FFI export 恢复
- **neutron**: `common/tlsx` 抽离、`operators/full` 独立子模块、cert 字段映射收口

### Dependencies

- fingers `v1.2.1` → `v1.2.2-0.2026..f5c144e`（xray 引擎 + AC 预过滤 + CaseInsensitive + TinyGo 兼容）
- neutron `c816917` → `e112381`（SSL/TLS 协议 + JSON/XPath operators/full + cert DSL 扩展 + Go 1.11 兼容）
- proton `03df34b` → `89c10c8`（pure matching SDK 重构 + FFI export + Template.Execute bugfix）
- zombie `705f548` → `bdd2cdf`（ProxyDial 注入 + 依赖更新）

## v0.2.4 (2026-06-08)

### Features

- **neutron**: 新增 `VulnResult` 类型，提供可序列化的漏洞扫描结果结构（`types.VulnResult` + `ExecuteResult.VulnResult()`）
- **neutron**: 集成 pluggable POC 格式转换器注册表，支持 `templates.Load` 自动加载 xray 格式 POC

### Bug Fixes

- **fingers**: 修复 source1 tag 覆盖引擎实例而非复制副本的问题
- **fingers**: 主动扫描保留原始路径并正确解析重定向
- **neutron**: 隔离模板编译变量，避免并发执行时变量污染
- **cyberhub**: 优先使用 POC 原始 YAML，提升模板加载准确性

### Refactor

- **fingers**: 移除 SDK 层路径拼接和重定向包装，下沉至 neutron 层处理

### Dependencies

- neutron `ea95825` → `c816917`（converter registry + 变量隔离 + 运行期补齐）
- fingers `9b9b6fe` → `385e7d5`（xray template load + active-match 修复）
- parsers `da1ebd0` → `3d2c51b`
- spray `546e8ab` → `66dafe7`（proton import 路径适配）
- zombie `21a4ec2` → `705f548`
- proton `e7e7b12` → `03df34b`（protocols/file → proton/file 重构）
- proxyclient `74a84a4` → `2a80e08`
