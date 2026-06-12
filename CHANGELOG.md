# Changelog

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
