# Chainreactors SDK

Chainreactors SDK 提供统一的 Go 接口，用于各种安全工具的集成和使用。

[![Go Version](https://img.shields.io/badge/Go-1.20+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## 概述

Chainreactors SDK 是一个统一的 SDK 框架，为多个安全工具提供一致的 API 接口。目前支持：

- **GoGo**: 端口扫描和服务识别
- **Spray**: HTTP URL 检测和路径暴力破解

## 特性

- ✅ **统一接口**: 所有工具实现相同的核心接口
- ✅ **简洁 API**: 只有 4 个核心概念（Engine, Context, Task, Result）
- ✅ **灵活配置**: 支持默认配置和自定义配置
- ✅ **流式处理**: 内置 Stream API，适合大规模数据处理
- ✅ **Context 支持**: 完整支持 context 取消和超时
- ✅ **工厂模式**: 支持通过工厂创建引擎
- ✅ **多态使用**: 不同引擎可以通过统一接口操作

## 安装

```bash
go get github.com/chainreactors/sdk
```

## 快速开始

### 使用 GoGo SDK (端口扫描)

```go
import "github.com/chainreactors/sdk/gogo"

// 创建引擎
engine := gogo.NewGogoEngine(nil)
engine.Init()

ctx := context.Background()

// 单目标扫描
result := engine.ScanOne(ctx, "192.168.1.1", "80")
fmt.Printf("%s:%s - %s\n", result.Ip, result.Port, result.Status)

// 批量扫描
results, _ := engine.Scan(ctx, "192.168.1.0/24", "top100")
for _, r := range results {
    fmt.Printf("%s:%s - %s\n", r.Ip, r.Port, r.Title)
}
```

### 使用 Spray SDK (Web 扫描)

```go
import "github.com/chainreactors/sdk/spray"

// 创建引擎
engine := spray.NewSprayEngine(nil)
engine.Init()

ctx := context.Background()

// URL 检测
urls := []string{"http://example.com", "http://httpbin.org"}
results, _ := engine.Check(ctx, urls)
for _, r := range results {
    fmt.Printf("%s [%d]\n", r.UrlString, r.Status)
}

// 路径暴力破解
wordlist := []string{"admin", "api", ".git"}
results, _ = engine.Brute(ctx, "http://example.com", wordlist)
```

### 使用工厂模式

```go
import rootsdk "github.com/chainreactors/sdk"

// 列出所有已注册的引擎
engines := rootsdk.ListEngines()
fmt.Printf("Available engines: %v\n", engines)

// 通过工厂创建引擎
gogoEngine, _ := rootsdk.NewEngine("gogo", nil)
sprayEngine, _ := rootsdk.NewEngine("spray", nil)
```

### 使用统一接口

所有引擎都实现了统一的 SDK 接口，可以多态使用：

```go
import (
    rootsdk "github.com/chainreactors/sdk"
    sdk "github.com/chainreactors/sdk/sdk"
    "github.com/chainreactors/sdk/gogo"
    "github.com/chainreactors/sdk/spray"
)

// 多态使用引擎
var engines []sdk.Engine = []sdk.Engine{
    gogo.NewGogoEngine(nil),
    spray.NewSprayEngine(nil),
}

for _, engine := range engines {
    fmt.Printf("Engine: %s\n", engine.Name())
}

// 使用统一的 Execute 方法
engine, _ := rootsdk.NewEngine("gogo", nil)
ctx := gogo.NewContext()
task := gogo.NewScanTask("192.168.1.1", "80,443")
resultCh, _ := engine.Execute(ctx, task)

for result := range resultCh {
    if result.Success() {
        fmt.Printf("Result: %v\n", result.Data())
    }
}
```

## 核心概念

SDK 基于 4 个核心接口设计：

### 1. Engine (引擎)

引擎负责执行具体功能，所有引擎实现统一接口：

```go
type Engine interface {
    Name() string
    Execute(ctx Context, task Task) (<-chan Result, error)
    Close() error
}
```

### 2. Context (上下文)

上下文包含配置和控制信息，支持链式调用：

```go
type Context interface {
    Context() context.Context
    Config() Config
    WithConfig(config Config) Context
    WithTimeout(timeout time.Duration) Context
    WithCancel() (Context, context.CancelFunc)
}
```

### 3. Task (任务)

任务定义要执行的操作：

```go
type Task interface {
    Type() string
    Validate() error
}
```

### 4. Result (结果)

结果返回执行结果和状态：

```go
type Result interface {
    Success() bool
    Error() error
    Data() interface{}
}
```

## 使用方式

### 方式 1: 便捷 API（推荐）

每个 SDK 都提供了便捷的 API，简化常用操作：

```go
// GoGo
engine := gogo.NewGogoEngine(nil)
engine.Init()
results, _ := engine.Scan(ctx, "192.168.1.0/24", "80,443")

// Spray
engine := spray.NewSprayEngine(nil)
engine.Init()
results, _ := engine.Check(ctx, urls)
```

### 方式 2: 统一接口

使用统一的 Execute 方法，支持多态：

```go
engine := gogo.NewGogoEngine(nil)
ctx := gogo.NewContext()
task := gogo.NewScanTask("192.168.1.0/24", "80,443")
resultCh, _ := engine.Execute(ctx, task)
```

### 方式 3: 工厂模式

通过工厂创建引擎，支持动态选择：

```go
engine, _ := rootsdk.NewEngine("gogo", nil)
```

## 项目结构

```
D:\Programing\go\chainreactors\sdk\
├── sdk/                  # 核心接口定义
│   ├── interface.go     # 4 个核心接口
│   └── helper.go        # 辅助函数
│
├── engine.go            # 工厂和注册
├── sdk_test.go          # 核心接口测试
│
├── gogo/                # GoGo 端口扫描 SDK
│   ├── gogo.go         # 引擎实现
│   ├── api.go          # 便捷 API
│   ├── init.go         # 注册
│   ├── gogo_test.go    # 测试
│   └── README.md       # 文档
│
├── spray/               # Spray Web 扫描 SDK
│   ├── spray.go        # 引擎实现
│   ├── api.go          # 便捷 API
│   ├── init.go         # 注册
│   ├── spray_test.go   # 测试
│   └── README.md       # 文档
│
└── examples/            # 使用示例
    └── main.go         # 完整示例代码
```

## 各 SDK 文档

- [GoGo SDK](gogo/README.md) - 端口扫描和服务识别
- [Spray SDK](spray/README.md) - HTTP URL 检测和路径暴力破解

## 配置

### GoGo 配置

```go
import "github.com/chainreactors/gogo/v2/pkg"

opt := pkg.DefaultRunnerOption
opt.VersionLevel = 2      // 深度指纹识别
opt.Exploit = "auto"      // 启用漏洞检测
opt.Delay = 5             // 超时时间

engine := gogo.NewGogoEngine(opt)
engine.SetThreads(500)    // 设置线程数
```

### Spray 配置

```go
opt := spray.DefaultConfig()
opt.Threads = 200
opt.Timeout = 10
opt.Method = "POST"
opt.Headers = []string{"Authorization: Bearer token"}
opt.Filter = "current.Status == 404"

engine := spray.NewSprayEngine(opt)
engine.SetThreads(150)
engine.SetTimeout(15)
```

## 示例程序

运行完整示例：

```bash
cd examples
go run main.go
```

示例包含：
- GoGo 便捷 API 使用
- Spray 便捷 API 使用
- 使用统一接口
- Context 链式调用
- 多态使用
- 工厂模式

## 测试

```bash
# 运行所有测试
go test ./...

# 运行特定 SDK 的测试
go test ./gogo -v
go test ./spray -v

# 运行集成测试（需要网络）
go test ./gogo -v -run TestScanIntegration
```

## 常见问题

### Q: 如何选择使用哪个 API 方式？
A:
- 便捷 API：适合大多数场景，简单直接
- 统一接口：需要多态或动态选择引擎时使用
- 工厂模式：需要根据配置动态创建引擎时使用

### Q: Stream 和 Sync API 有什么区别？
A:
- Stream: 返回 channel，实时处理结果，适合大规模数据，内存占用小
- Sync: 返回切片，等待所有结果完成，适合小规模数据，方便处理

### Q: Context 超时后会发生什么？
A: 引擎会停止扫描并返回已经获得的结果，不会阻塞

### Q: 如何添加新的 SDK？
A: 实现 4 个核心接口，在 init.go 中注册即可

## 设计优势

- **统一性**: 所有 SDK 实现相同的核心接口
- **简洁性**: 只有 4 个核心接口，方法最小化
- **灵活性**: 保留各 SDK 的便捷 API，支持链式调用
- **扩展性**: 新增 SDK 只需实现 4 个接口

## 贡献

欢迎提交 Issue 和 Pull Request！

## License

MIT License

## 相关项目

- [GoGo](https://github.com/chainreactors/gogo) - 端口扫描工具
- [Spray](https://github.com/chainreactors/spray) - Web 扫描工具
- [Parsers](https://github.com/chainreactors/parsers) - 结果解析库
