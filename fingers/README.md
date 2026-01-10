# Fingers SDK

基于 Cyberhub 的 Fingers 指纹管理 SDK，提供对 fingers 库的二次封装。

## 🌟 亮点

- **极简设计**: 总共仅 **563 行**代码（fingers）+ **196 行**（cyberhub）= **759 行**
- **零冗余**: 使用 `json:",inline"` 直接嵌入 `fingers.Finger`，types.go 仅 **51 行**
- **完美匹配**: 客户端结构 = 后端 `ExportFinger` 结构
- **功能完整**: 2876 个指纹 + 2823 个 Aliases 全量管理
- **统一抽象**: 无 Loader 抽象，local/remote 统一在 Engine 内部处理
- **极简 API**: 仅 1 个函数 - `Load`
- **无侵入集成**: gogo/spray 自己组装，SDK 只负责加载

## 📦 安装

```bash
go get github.com/chainreactors/sdk/fingers
```

## 🚀 快速开始

### API 设计

Fingers SDK 本质上是对 fingers 库的二次封装，只提供**加载功能**：

```go
// 统一入口，仅此而已
fingers.Load(config)
```

返回的是 `*fingersLib.Engine`，用户自己决定如何使用。

### 示例 1：基础使用

```go
import (
    "github.com/chainreactors/sdk/fingers"
)

// 从 Cyberhub 加载
config := fingers.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")

engine, _ := fingers.Load(config)

// 或从本地加载（指定引擎列表）
localConfig := fingers.NewConfig()
localConfig.SetEnableEngines([]string{"fingers"})
engine, _ := fingers.Load(localConfig)

// 使用 fingers 库的原生 API
frameworks, _ := engine.DetectResponse(resp)
```

### 示例 2：自定义配置

```go
config := fingers.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")
config.SetTimeout(30 * time.Second)

engine, _ := fingers.Load(config)
```

### 示例 3：集成到 gogo（自己组装）

```go
import (
    "github.com/chainreactors/sdk/fingers"
    "github.com/chainreactors/sdk/gogo"
)

// 1. 加载完整引擎
config := fingers.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")

fingersEngine, _ := fingers.NewEngine(config)

// 2. 注入到 gogo
gogoConfig := gogo.NewConfig().WithFingersEngine(fingersEngine)
gogoEngine := gogo.NewEngine(gogoConfig)
gogoEngine.Init()
```

### 示例 4：集成到 spray（自己组装）

```go
import (
    "github.com/chainreactors/sdk/fingers"
    "github.com/chainreactors/sdk/spray"
)

// 1. 加载完整引擎
config := fingers.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")

fingersEngine, _ := fingers.NewEngine(config)

// 2. 直接注入到 spray（spray 需要完整 Engine）
sprayConfig := spray.NewConfig().WithFingersEngine(fingersEngine)
sprayEngine := spray.NewEngine(sprayConfig)
sprayEngine.Init()
```

### 示例 5：SDK Engine（可选）

如果需要统一的 SDK 接口：

```go
import (
    "github.com/chainreactors/sdk"
    "github.com/chainreactors/sdk/fingers"
)

// 通过全局工厂创建
config := fingers.NewConfig()
engine, _ := sdk.NewEngine("fingers", config)
defer engine.Close()

// 使用 SDK 接口
ctx := fingers.NewContext()
task := fingers.NewMatchTask(httpResponse)

resultCh, _ := engine.Execute(ctx, task)
for result := range resultCh {
    if result.Success() {
        matchResult := result.(*fingers.MatchResult)
        for _, fw := range matchResult.Frameworks() {
            fmt.Printf("指纹: %s\n", fw.Name)
        }
    }
}
```

## 📊 数据统计

- **指纹数量**: 2876（Cyberhub） / 4373（本地）
- **Aliases**: 2823 个
- **响应大小**: ~3.6MB (全量导出)
- **初始化时间**: < 1 秒

## 🏗️ 架构设计

### 核心结构

```go
// pkg/cyberhub/types.go - 极简设计
type FingerprintResponse struct {
    *fingers.Finger `json:",inline"` // 零冗余！
    Alias           *alias.Alias `json:"alias,omitempty"`
}
```

**优势**:
- ✅ 完美匹配后端 `ExportFinger`
- ✅ 零字段映射，零手动转换
- ✅ Finger 结构变更自动同步

### 目录结构

```
sdk/
├── fingers/           # 核心包（41+97+388+37 = 563 行）
│   ├── config.go     # 配置管理（97 行）
│   ├── engine.go     # 统一 Engine + SDK 接口（含极简 API）
│   └── init.go       # 全局注册（37 行）
├── pkg/cyberhub/     # Cyberhub 客户端（145+51 = 196 行）
│   ├── client.go     # HTTP 客户端（145 行）
│   └── types.go      # 数据类型（51 行）
├── gogo/             # gogo 集成（自己组装）
└── spray/            # spray 集成（自己组装）
```

### 设计原则

1. **SDK 即加载器**: SDK 只负责加载 fingers 库的 Engine，不提供额外封装
2. **用户自己组装**: gogo/spray 等集成由用户自己从 Engine 提取需要的部分
3. **极简 API**:
   - `Load(config)` - 通用加载
4. **无侵入**: 不强制用户使用 SDK Engine，可以直接用 fingers 库

## 🎯 API 演进历史

| 版本 | API 数量 | 问题 | 改进 |
|------|---------|------|------|
| v1.0 | 6+ 个 `New*` 函数 | 命名混淆，不知道用哪个 | ❌ |
| v2.0 | 三层 API（New*/Load*/LoadForGogo*） | 层次清晰但过度设计 | 🤔 |
| v3.0 | **1 个函数**（Load） | 极简，用户自己组装 | ✅ |

## ✅ 测试结果

```bash
# 基础测试
$ go run test/test_fingers.go
✅ 远程引擎: fingers:2876
✅ 本地引擎: fingers:4373 (+ 其他引擎)
✅ gogo 集成（自己组装）: 2851 HTTP 指纹, 25 Socket 指纹

# 集成测试
$ go run test/test_integration.go
✅ gogo 集成: 2851 HTTP 指纹
✅ spray 集成: fingers:2876
✅ Aliases: 2823 个

# SDK Engine 测试
$ go run test/test_sdk_engine.go
✅ 引擎创建成功: fingers
✅ 匹配成功，找到 5 个指纹
```

## 🔧 配置选项

```go
config := fingers.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")
config.WithLocalFile("fingers.yaml") // 可选：从导出的 YAML 加载
config.SetTimeout(30 * time.Second)
config.SetEnableEngines([]string{"fingers", "wappalyzer"})

engine, _ := fingers.Load(config)
```

## 🎯 特性

- [x] Cyberhub Export API 集成
- [x] 一次性加载全量指纹（2876 条）
- [x] Alias 管理（2823 个）
- [x] 本地/远程自动切换
- [x] 极简 API（仅 1 个函数）
- [x] 用户自己组装集成
- [x] SDK Engine 接口（可选）
- [x] 支持 `[]byte` 和 `http.Response` 匹配
- [x] 无 Loader 抽象
- [x] 超时控制（默认 30s）

## 📖 文档

- [完整实现文档](../IMPLEMENTATION.md) - 详细的技术实现
- [测试代码](../test/) - 完整的测试示例
- [示例代码](../examples/fingers_example.go) - 使用示例

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 License

MIT License

---

**开发完成**: 2025-12-21
**版本**: v3.0.0 Final
**代码量**: 759 行（极简）
**状态**: ✅ 生产就绪
**设计理念**: SDK 即加载器，用户自己组装
