# Chainreactors SDK

统一的安全扫描工具 Go SDK，提供一致的接口设计。

## 概述

Chainreactors SDK 为多个安全扫描工具提供统一接口：

- **Fingers**: Web 指纹识别（HTTP/Socket）
- **Neutron**: POC/漏洞扫描
- **GoGo**: 集成指纹识别和 POC 检测的端口扫描
- **Spray**: HTTP 批量检测和路径爆破

## 安装

```bash
go get github.com/chainreactors/sdk
```

## 架构设计

### 核心架构

SDK 采用简单的四组件架构：

1. **Engine（引擎）**: 实现具体的扫描逻辑
2. **Context（上下文）**: 携带配置和控制信息
3. **Task（任务）**: 定义扫描目标
4. **Result（结果）**: 返回扫描结果

每个引擎可以独立使用，也可以与其他引擎集成（如 GoGo 同时集成 Fingers 和 Neutron）。

### 数据源

所有引擎支持双重加载模式：

- **本地模式**: 从嵌入数据或文件系统加载
- **远程模式**: 从 Cyberhub API 加载，支持 sources 过滤

## 快速开始

### Fingers - 指纹识别

```go
import (
    "github.com/chainreactors/sdk/fingers"
)

// 创建并加载引擎
config := fingers.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your_key")

engine, _ := fingers.NewEngine(config)
libEngine := engine.Get()

// 检测指纹
frameworks, _ := libEngine.DetectContent(httpResponse)
```

### Neutron - POC 扫描

```go
import (
    "github.com/chainreactors/sdk/neutron"
)

// 创建并加载引擎
config := neutron.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your_key")

engine, _ := neutron.NewEngine(config)
templates := engine.Get()  // 自动编译

// 执行 POC
for _, t := range templates {
    result, _ := t.Execute("http://target.com", nil)
    if result.Matched {
        // 处理漏洞
    }
}
```

### GoGo - 集成扫描

```go
import (
    "github.com/chainreactors/sdk/gogo"
    "github.com/chainreactors/sdk/fingers"
    "github.com/chainreactors/sdk/neutron"
)

// 加载指纹库
fingersEngine, _ := fingers.NewEngine(fingersConfig)

// 加载 POC
neutronEngine, _ := neutron.NewEngine(neutronConfig)

// 创建集成扫描器
gogoConfig := gogo.NewConfig().
    WithFingersEngine(fingersEngine).
    WithNeutronEngine(neutronEngine)
gogoEngine := gogo.NewEngine(gogoConfig)
gogoEngine.Init()

// 执行扫描
gogoCtx := gogo.NewContext().
    SetThreads(1000).
    SetVersionLevel(2).
    SetExploit("all").
    SetDelay(5)
task := gogo.NewScanTask("192.168.1.0/24", "80,443,8080")
resultCh, _ := gogoEngine.Execute(gogoCtx, task)

for result := range resultCh {
    // 处理结果
}
```

### Spray - HTTP 检测

```go
import (
    "github.com/chainreactors/sdk/spray"
)

engine := spray.NewEngine(nil)
engine.Init()

urls := []string{"http://example.com", "http://target.com"}
sprayCtx := spray.NewContext().
    SetThreads(100).
    SetTimeout(10)
task := spray.NewCheckTask(urls)
resultCh, _ := engine.Execute(sprayCtx, task)

for result := range resultCh {
    sprayResult := result.(*spray.Result).SprayResult()
    // 处理结果
}
```

## 配置

### Fingers 配置

```go
config := fingers.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your_key")
config.SetSources("github")       // 可选：按来源过滤
config.WithLocalFile("fingers.yaml") // 可选：从导出的 YAML 加载
config.SetTimeout(10 * time.Second)
```

### Neutron 配置

```go
config := neutron.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your_key")
config.SetSources("github")       // 可选：按来源过滤
config.WithLocalFile("./pocs") // 可选：本地 POC 目录
config.SetTimeout(10 * time.Second)
```

### GoGo 配置

```go
config := gogo.NewConfig().
    WithFingersEngine(fingersEngine).
    WithNeutronEngine(neutronEngine)
```

### GoGo 运行时上下文

```go
ctx := gogo.NewContext().
    SetThreads(1000).
    SetVersionLevel(2).         // 0-3，数值越高检测越深
    SetExploit("all").          // none/all/known
    SetDelay(5)                 // 请求超时时间
```

### Spray 运行时上下文

```go
ctx := spray.NewContext().
    SetThreads(100).
    SetTimeout(10)
```

## 命令行工具

`examples/` 目录提供了预构建的命令行工具：

```bash
# 构建所有工具
cd examples
go build -o fingers/fingers.exe ./fingers/main.go
go build -o neutron/neutron.exe ./neutron/main.go
go build -o gogo/gogo.exe ./gogo/main.go
go build -o spray/spray.exe ./spray/main.go
```

详细使用方法参见 [examples/README.md](examples/README.md)。

## 项目结构

```
sdk/
├── fingers/              # 指纹识别引擎
│   ├── engine.go        # 核心引擎实现
│   ├── config.go        # 配置
│   └── init.go         # 注册入口
│
├── neutron/             # POC 扫描引擎
│   ├── engine.go       # 核心引擎（自动编译）
│   ├── config.go       # 配置
│   └── init.go        # 注册入口
│
├── gogo/               # 端口扫描（集成）
│   ├── gogo.go        # 支持 Fingers/Neutron 的引擎
│   ├── config.go      # 配置
│   └── init.go       # 注册入口
│
├── spray/              # HTTP 检测引擎
│   ├── spray.go       # 核心引擎实现
│   ├── config.go      # 配置
│   └── init.go       # 注册入口
│
├── pkg/
│   ├── cyberhub/      # 统一 API 客户端
│   │   ├── client.go  # HTTP 客户端（支持 gzip）
│   │   └── types.go   # API 类型
│   ├── interface.go   # 核心 SDK 接口
│   └── helper.go      # 工具函数
│
└── examples/           # CLI 工具实现
    ├── fingers/
    ├── neutron/
    ├── gogo/
    ├── spray/
    └── README.md
```

## 核心特性

### Cyberhub 集成

所有引擎都支持从 Cyberhub 加载数据：
- Gzip 压缩处理
- 基于 sources 的过滤
- API Key 认证

### POC 自动编译

Neutron 引擎在加载时自动编译 POC：
- 无需手动编译
- 编译失败的 POC 自动跳过
- ExecuterOptions 从引擎配置生成

### GoGo 集成

GoGo 可以同时集成 Fingers 和 Neutron：
- 模板按指纹、ID、标签建立索引
- 9,444 个 POC 生成 61,267 条索引（多重索引）
- 根据检测到的指纹自动匹配模板

## 开发

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./fingers -v
go test ./neutron -v
go test ./gogo -v
go test ./spray -v
```

### 添加新引擎

1. 实现 `pkg/interface.go` 中的核心接口
2. 创建引擎包，包含 `engine.go`、`config.go`、`init.go`
3. 在 `engine.go` 的 init 函数中注册
4. 在 `examples/` 中添加 CLI 工具

## License

MIT License

## 相关项目

- [Fingers](https://github.com/chainreactors/fingers) - 指纹识别库
- [Neutron](https://github.com/chainreactors/neutron) - POC 框架
- [GoGo](https://github.com/chainreactors/gogo) - 端口扫描器
- [Spray](https://github.com/chainreactors/spray) - HTTP 扫描器
