# Changelog

## v0.3.3 (2026-06-16)

新增 association 模糊搜索机制，支持对索引内所有实体的子串模糊匹配；修复 CyberHub POC 导出默认状态过滤导致零结果的问题。

### New Features

**association — 模糊搜索**

- 新增 `WithSearch` 查询构建器，支持跨指纹/别名/模板的子串模糊搜索
- `Lookup` 空查询（无 terms 且无 search）返回索引内全部实体
- 新增 `FingersWithTemplates` 便捷方法，返回有关联模板的指纹及模板数量
- 模糊搜索可与精确 terms 组合使用，结果自动去重并展开关联实体

```go
// 模糊搜索
result := idx.Lookup(association.NewQuery().WithSearch("nginx"))
// 模糊 + 精确组合
result = idx.Lookup(association.NewQuery().WithSearch("splunk").WithTags("webserver"))
// 查询有关联模板的指纹
fwt := result.FingersWithTemplates(idx)
```

### Bug Fixes

- **cyberhub**: `applyDefaultPOCStatus` 不再默认注入 `status=active`，修复 CyberHub 实例 POC 无 status 字段时导出零结果的问题

### Tests

- 新增 8 个 search 机制单元测试（子串匹配、去重、大小写无关、组合查询等）
- 新增 7 个 CyberHub 集成测试（真实数据验证，通过 `CYBERHUB_URL`/`CYBERHUB_KEY` 环境变量控制）

## v0.3.2 (2026-06-15)

新增 CPE 自动关联机制，重构 httpx 为通用 client generator，升级 neutron/fingers 大幅提升主动指纹匹配准确率。

### Breaking Changes

- **全引擎**: `Execute(nil, task)` 不再静默 fallback 到 `NewContext()`，改为返回 error。调用方必须显式传入 Context

### New Features

**association — CPE 自动关联（新模块）**

- 新增 `pkg/association` 模块，基于 CPE vendor:product 自动建立指纹 ↔ POC 关联索引
- `BuildFromProvider` 一行代码从 CyberHub 加载指纹与 POC 数据并构建索引
- 支持按指纹名、CPE、POC ID、tag、CVE 等多维度查询，结果包含关联的指纹、别名和 POC 模板
- `LookupFrameworks` 直接从扫描结果的 Frameworks 查询关联 POC
- `QueryFromResult` 从 gogo/spray/zombie 的 `types.Result` 自动提取查询条件

```go
// 构建关联索引
idx, _ := association.BuildFromProvider(ctx, cyberhub.NewProvider(url, key))

// 指纹名查询关联 POC
result := idx.Lookup(association.NewQuery().WithFingers("tomcat"))
for _, tpl := range result.Templates {
    fmt.Println(tpl.ID) // "CVE-2017-12615"
}

// 直接从 Frameworks 查询
related := idx.LookupFrameworks(scanResult.Frameworks)

// 从扫描结果自动提取查询条件
related = idx.Lookup(association.QueryFromResult(engineResult))
```

**httpx — 通用 client generator**

- 重构 `pkg/httpx` 为带预设的 HTTP client 工厂
- 新增 `BrowserConfig()` 预设（自动注入浏览器 UA/Accept/Accept-Language）和 `DefaultConfig()` 裸配置
- 新增 `WithTimeout`/`WithProxy`/`WithRedirects`/`WithHeaders` builder 方法
- `NewClient` 内聚 proxy fallback：代理解析失败自动回退无代理 client

```go
client, _ := httpx.NewClient(httpx.BrowserConfig().WithProxy("socks5://127.0.0.1:1080"))
```

**neutron — redirect 策略对齐 + per-context CookieJar**

- redirect 策略三态化（DontFollow / FollowAll / FollowSameHost），对齐 nuclei RedirectFlow 语义
- per-context CookieJar：每次模板执行自动创建独立 CookieJar，redirect 链内 Set-Cookie 正确传递
- xray 转换器修正 `follow_redirects` 默认行为，自动检测依赖跳转响应的规则并保留原始响应

**fingers — 主动探测优化**

- 主动探测新增跨 finger 的 `cachingSender` 请求缓存，相同 send_data 路径只发一次 HTTP 请求
- hub/xray 引擎改用 `ExecuteWithTransport` 整模板执行，修复多请求模板 `body_N` 塌陷为 `body_1` 的漏匹配

### Bug Fixes

- **全引擎**: `emitStats` 在 context 取消后跳过回调，避免 send on closed channel panic
- **neutron**: favicon 探测增强——最终跳转 URL 做 base、增加 `favicon.ico` 探测路径、独立 context 避免超时误判
- **neutron**: 修正 RootURL 挂载路径拼接

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
