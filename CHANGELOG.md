# Changelog

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
