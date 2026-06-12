# Changelog

## v0.3.0 (2026-06-11)

### Breaking Changes

- **gogo/spray/zombie**: `GogoEngine`、`SprayEngine` 统一重命名为 `Engine`；`NewEngine` 签名统一返回 `(*Engine, error)`
- **proton**: `ProtonFinding` 重命名为 `ProtonResult`
- **types**: 移除 `pkg/types/registry.go`，各引擎 init 注册逻辑移除

### Features

- **proton**: 集成 proton 文件扫描引擎到 SDK，支持本地文件敏感信息扫描（密码、密钥、凭证等）
- **neutron**: 新增 JSON/XPath extractor 和 matcher 支持（通过 `neutron/operators/full` 引入 gojq/antchfx）
- **neutron**: 新增 SSL/TLS 探测协议、RootURL 挂载、DSL cert_* 函数、动态协议注册
- **types**: 新增 `ReviewStatus` 常量替代审核状态 magic string

### Refactor

- 移除未使用代码和冗余抽象（-284 行）
- 各引擎 init.go 注册逻辑移除，简化初始化流程
- proton 类型统一收归 `pkg/types`，与其他引擎风格一致

### Bug Fixes

- **spray**: 修复 context 取消后 `emitStats` panic 的问题

### Tests

- 新增 neutron SSL/RootURL/DSL/registrar 测试
- 新增 JSON/XPath YAML 解析测试
- proton engine 集成测试（734 行）
- 新增 spray context cancel panic 复现测试

### Dependencies

- fingers `v1.2.1` → `v1.2.2-0.2026..f5c144e`（remote 依赖更新）
- neutron `c816917` → `e112381`（SSL 协议 + JSON/XPath operators/full + Go 1.11 兼容修复）
- proton `03df34b` → `89c10c8`（FFI export + pure matching SDK + Template.Execute bugfix）
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
