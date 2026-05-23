# GoGo - 端口扫描

GoGo 引擎提供端口扫描功能，并可集成 Fingers 和 Neutron 实现扫描 → 指纹识别 → 漏洞检测的一体化流程。

## 创建引擎

### 最简方式：一个 Provider 搞定

```go
provider := cyberhub.NewProvider("http://hub:8080", "api-key")
config := gogo.NewConfig().WithProvider(provider)

engine := gogo.NewEngine(config)
engine.Init()  // 自动创建 fingers + neutron 引擎
```

`WithProvider` 会在 `Init()` 时自动从 Cyberhub 加载指纹和 POC 数据，创建内部的 Fingers 和 Neutron 引擎。

### 自定义组合

如果需要更细粒度的控制，可以手动注入引擎：

```go
// 分别创建 fingers 和 neutron
fingersEng, _ := fingers.NewEngine(fingersConfig)
neutronEng, _ := neutron.NewEngine(neutronConfig)

config := gogo.NewConfig().
    WithFingersEngine(fingersEng).
    WithNeutronEngine(neutronEng).
    WithCapacity(5000)

engine := gogo.NewEngine(config)
engine.Init()
```

### 不加载数据

GoGo 可以不加载任何外部数据，使用内置指纹进行基础扫描：

```go
engine := gogo.NewEngine(nil)
engine.Init()
```

> 源码：[`gogo/gogo.go`](../gogo/gogo.go)、[`gogo/config.go`](../gogo/types.go)

## 端口扫描

### 同步扫描

```go
ctx := gogo.NewContext().
    SetThreads(2000).
    SetVersionLevel(2).
    SetExploit("none").
    SetDelay(5)

results, err := engine.Scan(ctx, "192.168.1.0/24", "80,443,8080-8090")
for _, r := range results {
    fmt.Printf("%s:%s [%s]\n", r.Ip, r.Port, r.Status)
    for _, fw := range r.Frameworks {
        fmt.Printf("  finger: %s\n", fw.Name)
    }
}
```

### 流式扫描

```go
resultCh, err := engine.ScanStream(ctx, "10.0.0.0/16", "top100")
for result := range resultCh {
    // 每发现一个开放端口就立即收到
    fmt.Printf("%s:%s open\n", result.Ip, result.Port)
}
```

### 单目标扫描

```go
result := engine.ScanOne(ctx, "192.168.1.1", "80")
fmt.Println(result.Status, result.Frameworks)
```

> 示例：[`examples/gogo/main.go`](../examples/gogo/main.go)

## Context 配置

```go
ctx := gogo.NewContext().
    SetThreads(2000).          // 并发线程数（默认 1000）
    SetVersionLevel(2).        // 指纹识别级别 0-3
    SetExploit("all").         // POC 模式：none / all / known
    SetDelay(5)                // 超时时间（秒）
```

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `SetThreads` | 并发线程数 | 1000 |
| `SetVersionLevel` | 指纹识别深度，0=关闭，1=基础，2=深度，3=全量 | 0 |
| `SetExploit` | 漏洞检测模式 | `"none"` |
| `SetDelay` | 单次请求超时（秒） | 5 |

> 源码：[`gogo/types.go`](../gogo/types.go)

## 工作流扫描

Workflow 允许更精细地控制扫描行为：

```go
workflow := &types.Workflow{
    IP:    "192.168.1.0/24",
    Ports: "80,443,8080",
}

results, err := engine.Workflow(ctx, workflow)

// 流式版本
resultCh, err := engine.WorkflowStream(ctx, workflow)
```

## 扫描结果

`*types.GOGOResult` 包含丰富的扫描信息：

```go
r.Ip          // IP 地址
r.Port        // 端口号
r.Status      // 状态（open 等）
r.Protocol    // 协议（http/https/tcp 等）
r.Title       // HTTP 标题
r.Frameworks  // 识别到的指纹 map[string]*Framework
r.Vulns       // 检测到的漏洞 map[string]*Vuln
```

## 通过统一接口使用

```go
task := gogo.NewScanTask("192.168.1.0/24", "80,443")
resultCh, err := engine.Execute(ctx, task)

for result := range resultCh {
    if data, ok := types.ResultData[*types.GOGOResult](result); ok {
        fmt.Println(data.Ip, data.Port)
    }
}
```

## 统计回调

```go
ctx := gogo.NewContext().
    SetStatsHandler(func(s types.Stats) {
        fmt.Printf("targets=%d requests=%d results=%d errors=%d duration=%v\n",
            s.Targets, s.Requests, s.Results, s.Errors, s.Duration)
    })
```

## 关联索引

GoGo 在 Init 时如果同时加载了 Fingers 和 Neutron，会自动构建关联索引：

```go
engine.Init()
idx := engine.Index()  // *association.Index

// 查询指纹关联的 POC
result := idx.Lookup(association.NewQuery().WithFingers("tomcat"))
for _, t := range result.Templates {
    fmt.Printf("poc: %s\n", t.Id)
}
```

更多关联查询用法见 [Association 关联查询](association.md)。

## 端口格式

支持以下格式：

- 单端口：`80`
- 逗号分隔：`80,443,8080`
- 范围：`8080-8090`
- 混合：`22,80,443,8000-9000`
- 预设：`top100`

## IP 格式

- 单 IP：`192.168.1.1`
- CIDR：`192.168.1.0/24`
- 域名：`example.com`
- IPv6：`[::1]`
