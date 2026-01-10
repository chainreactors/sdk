# Fingers SDK

基于 Cyberhub 的 Fingers 指纹管理 SDK，提供对 fingers 库的统一加载和匹配入口。

## 🌟 亮点

- **统一入口**: `NewEngine` 负责加载，`Match` 负责匹配（也可 `Get()` 取底层引擎）
- **本地/远程**: 支持从本地 YAML/目录或 Cyberhub 加载
- **零冗余**: Cyberhub 响应使用 `json:",inline"` 直接嵌入 `fingers.Finger`
- **无侵入集成**: gogo/spray 等通过注入引擎完成集成

## 📦 安装

```bash
go get github.com/chainreactors/sdk/fingers
```

## 🚀 快速开始

### API 设计

Fingers SDK 在初始化阶段完成加载，随后你可以直接匹配：

```go
config := fingers.NewConfig().WithCyberhub("http://127.0.0.1:8080", "your-api-key")
engine, _ := fingers.NewEngine(config)

frameworks, _ := engine.Match(httpResponseBytes)
```

如需使用原生 fingers 引擎：

```go
libEngine := engine.Get()
frameworks, _ := libEngine.DetectContent(httpResponseBytes)
```

### 示例 1：从 Cyberhub 加载

```go
config := fingers.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")

engine, _ := fingers.NewEngine(config)
frameworks, _ := engine.Match(httpResponseBytes)
```

### 示例 2：从本地文件/目录加载

```go
config := fingers.NewConfig()
config.WithLocalFile("./fingers.yaml") // 文件或目录

engine, _ := fingers.NewEngine(config)
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
    "fmt"
    "net/http"

    rootsdk "github.com/chainreactors/sdk"
    "github.com/chainreactors/sdk/fingers"
)

// 通过全局工厂创建
config := fingers.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")
engine, _ := rootsdk.NewEngine("fingers", config)
defer engine.Close()

// 使用 SDK 接口
resp, _ := http.Get("http://example.com")
defer resp.Body.Close()

ctx := fingers.NewContext()
task := fingers.NewMatchTaskFromResponse(resp)

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

## 🔧 配置选项

```go
config := fingers.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")
config.SetSources("github")
config.SetTimeout(30 * time.Second)

// 选择本地加载时使用 WithLocalFile，会覆盖 Cyberhub 配置
// config.WithLocalFile("./fingers.yaml")

engine, _ := fingers.NewEngine(config)
```

也可以直接注入内存数据：

```go
config := fingers.NewConfig()
config.WithFingers(fingersData)
config.WithAliases(aliases)

engine, _ := fingers.NewEngine(config)
```

## 🏗️ 架构设计

### 核心结构

```go
// pkg/cyberhub/types.go
type FingerprintResponse struct {
    *fingers.Finger `json:",inline" yaml:",inline"`
    Alias           *alias.Alias `json:"alias,omitempty" yaml:"alias,omitempty"`
}
```

### 目录结构

```
sdk/
├── fingers/           # Fingers SDK
│   ├── config.go     # 配置管理
│   ├── engine.go     # Engine 封装
│   ├── types.go      # Context/Task/Result
│   ├── additions.go  # 动态扩展 (AddFingers/AddFingersFile)
│   └── init.go       # 全局注册
├── pkg/cyberhub/     # Cyberhub 客户端
│   ├── client.go
│   ├── config.go
│   └── types.go
├── gogo/             # gogo 集成
└── spray/            # spray 集成
```

## 🎯 特性

- [x] Cyberhub Export API 集成
- [x] 本地 YAML/目录加载
- [x] Alias 管理
- [x] SDK Engine 接口（可选）
- [x] 支持 `[]byte` 和 `http.Response` 匹配
- [x] 动态扩展（AddFingers / AddFingersFile）

## 📖 文档

- [SDK 主文档](../README.md)
- [CLI 示例](../examples/fingers/main.go)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 License

MIT License
