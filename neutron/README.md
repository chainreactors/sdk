# Neutron SDK

Neutron SDK 为 [chainreactors/neutron](https://github.com/chainreactors/neutron) POC 引擎提供了简洁的加载接口，支持从本地文件和 Cyberhub 远程加载 POC。

## 设计理念

**SDK = Loader，用户 = Composer**

- 提供 **3 个原子化 API**，用户自行组装复杂功能
- 不过度封装，返回原生 `*templates.Template`
- 支持本地和远程双数据源

## 快速开始

### 1. 从 Cyberhub 加载 POC

```go
import (
    "github.com/chainreactors/sdk/neutron"
)

// 最简单的方式
config := neutron.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")
engine, err := neutron.NewEngine(config)
if err != nil {
    log.Fatal(err)
}
templates := engine.Get()

fmt.Printf("加载了 %d 个 POC\n", len(templates))
```

### 2. 从本地目录加载 POC

```go
// 加载指定目录的所有 YAML 文件
config := neutron.NewConfig()
config.WithLocalFile("./my_pocs")
engine, err := neutron.NewEngine(config)
if err != nil {
    log.Fatal(err)
}
templates := engine.Get()
```

### 3. 高级配置

```go
config := neutron.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")
config.SetTags("cve", "rce")               // 按标签过滤
config.WithLocalFile("pocs.yaml")          // 可选：从导出的 YAML 加载
config.SetTimeout(30 * time.Second)

engine, err := neutron.NewEngine(config)
if err != nil {
    log.Fatal(err)
}
templates := engine.Get()
```

## API 参考

### `neutron.NewEngine(config *Config)`

初始化引擎时完成加载与编译：

```go
config := neutron.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")

engine, err := neutron.NewEngine(config)
if err != nil {
    log.Fatal(err)
}
templates := engine.Get()
```

## 配置选项

```go
type Config struct {
    // Cyberhub 配置
    CyberhubURL string // Cyberhub API 地址
    APIKey      string // API Key 认证

    // 本地配置
    LocalPath string // 本地 POC 文件/目录路径
    Templates neutron.Templates // 已加载的 POC

    // 过滤配置
    Tags []string // 标签过滤

    // 请求配置
    Timeout time.Duration // HTTP 请求超时时间
}
```

## 使用示例

### 示例 1: 从 Cyberhub 加载并执行

```go
package main

import (
    "fmt"
    "github.com/chainreactors/sdk/neutron"
)

func main() {
    // 1. 加载 POC
    config := neutron.NewConfig()
    config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")
    engine, err := neutron.NewEngine(config)
    if err != nil {
        panic(err)
    }
    templates := engine.Get()
    fmt.Printf("✅ 加载了 %d 个 POC\n", len(templates))

    // 3. 执行 POC
    targetURL := "http://example.com"
    for _, t := range templates {
        result, err := t.Execute(targetURL, nil)
        if err != nil {
            continue
        }
        if result != nil && result.Matched {
            fmt.Printf("🎯 匹配: %s - %s\n", t.Id, t.Info.Name)
        }
    }
}
```

### 示例 2: 流式批量扫描（用户组装）

```go
package main

import (
    "fmt"
    "sync"
    neutronTemplates "github.com/chainreactors/neutron/templates"
    "github.com/chainreactors/sdk/neutron"
)

func main() {
    // 1. 加载并编译 POC
    config := neutron.NewConfig()
    config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")
    engine, _ := neutron.NewEngine(config)
    compiledPOCs := engine.Get()

    // 2. 用户自己组装流式扫描
    type ScanTask struct {
        Target string
        POC    *neutronTemplates.Template
    }

    targets := []string{"http://example.com", "http://test.com"}

    inputCh := make(chan ScanTask, 100)
    outputCh := make(chan bool, 100)

    // 生产者
    go func() {
        defer close(inputCh)
        for _, target := range targets {
            for _, poc := range compiledPOCs {
                inputCh <- ScanTask{Target: target, POC: poc}
            }
        }
    }()

    // 处理器（20 并发）
    go func() {
        defer close(outputCh)

        var wg sync.WaitGroup
        semaphore := make(chan struct{}, 20)

        for task := range inputCh {
            wg.Add(1)
            semaphore <- struct{}{}

            go func(t ScanTask) {
                defer wg.Done()
                defer func() { <-semaphore }()

                result, _ := t.POC.Execute(t.Target, nil)
                matched := result != nil && result.Matched
                outputCh <- matched
            }(task)
        }

        wg.Wait()
    }()

    // 消费者
    matchCount := 0
    for matched := range outputCh {
        if matched {
            matchCount++
        }
    }

    fmt.Printf("✅ 共匹配 %d 个 POC\n", matchCount)
}
```

### 示例 3: 混合本地和远程数据源

```go
config := neutron.NewConfig()
config.WithCyberhub("http://127.0.0.1:8080", "your-api-key")
config.WithLocalFile("./my_custom_pocs") // 同时加载本地 POC

engine, err := neutron.NewEngine(config)
templates := engine.Get()
// templates 包含来自 Cyberhub 和本地目录的所有 POC
```

## 完整示例

SDK 提供了 3 个完整示例：

1. **`examples/neutron_local_example.go`** - 从本地加载并执行
2. **`examples/neutron_cyberhub_example.go`** - 从 Cyberhub 加载并执行
3. **`examples/neutron_stream_example.go`** - 流式批量扫描（用户组装模式）

运行示例：

```bash
# 从 Cyberhub 加载示例
go run examples/neutron_cyberhub_example.go

# 流式扫描示例
go run examples/neutron_stream_example.go
```

## 测试结果

```bash
✅ 成功加载 9444 个 POC
✅ 成功编译 9444 个 POC
⏱️  加载速度: ~1s
```

## 与 Fingers SDK 的一致性

Neutron SDK 和 Fingers SDK 遵循相同的设计理念：

| 特性 | Fingers SDK | Neutron SDK |
|------|-------------|-------------|
| **加载函数** | `Load` | `NewEngine` |
| **返回类型** | `*fingersLib.Engine` | `[]*templates.Template` |
| **数据源** | 本地 + Cyberhub | 本地 + Cyberhub |
| **API 数量** | 3 个 | 3 个 |
| **设计理念** | SDK = Loader | SDK = Loader |

## 架构设计

```
neutron/
├── config.go       # 配置结构
└── engine.go       # 引擎实现（初始化时加载）

pkg/cyberhub/
├── client.go       # ExportPOCs() API
└── types.go        # POCResponse (inline templates.Template)
```

## 依赖项

- `github.com/chainreactors/neutron` - Neutron POC 引擎
- `github.com/chainreactors/sdk/pkg/cyberhub` - Cyberhub API 客户端
- `gopkg.in/yaml.v3` - YAML 解析

## 注意事项

1. **Cyberhub 必须运行** - 使用远程配置前确保 Cyberhub 服务可访问
2. **编译 POC** - 初始化引擎时自动完成
3. **变量支持** - 某些 POC 需要 wordlist、BaseDNS 等变量，通过 `Execute(target, payload)` 的 payload 参数传递
4. **错误处理** - POC 执行可能返回 `protocols.OpsecError`，表示 opsec 模式跳过

## License

MIT





