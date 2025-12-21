# Chainreactors SDK Examples

这个目录包含了四个基于 Chainreactors SDK 构建的命令行工具。

## 目录结构

```
examples/
├── fingers/    # 指纹识别工具
├── neutron/    # POC 扫描工具
├── gogo/       # 端口扫描和指纹识别工具
└── spray/      # HTTP 批量探测工具
```

## 快速开始

### 编译所有工具

```bash
cd examples
go build -o fingers/fingers.exe ./fingers/main.go
go build -o neutron/neutron.exe ./neutron/main.go
go build -o gogo/gogo.exe ./gogo/main.go
go build -o spray/spray.exe ./spray/main.go
```

---

## 1. Fingers - 指纹识别工具

识别 Web 应用的技术栈和框架。

### 基本用法

```bash
# 从本地加载指纹
./fingers/fingers.exe -target http://127.0.0.1:3000

# 从 Cyberhub 加载指纹
./fingers/fingers.exe -url http://127.0.0.1:8080 -key your_api_key -target http://127.0.0.1:3000

# 按 source 过滤
./fingers/fingers.exe -url http://127.0.0.1:8080 -key your_key -source github -target http://example.com

# JSON 输出
./fingers/fingers.exe -target http://127.0.0.1:3000 -json

# 显示详细信息
./fingers/fingers.exe -target http://127.0.0.1:3000 -info
```

### 参数说明

- `-url`: Cyberhub URL (可选)
- `-key`: Cyberhub API Key (与 -url 一起使用)
- `-source`: 按来源过滤指纹 (可选)
- `-engines`: 启用特定引擎 (可选)
- `-target`: 目标 URL **(必需)**
- `-json`: JSON 格式输出
- `-info`: 显示详细信息（版本、CPE 等）

---

## 2. Neutron - POC 扫描工具

使用 POC 对目标进行漏洞扫描。

### 基本用法

```bash
# 从 Cyberhub 加载 POC 并扫描
./neutron/neutron.exe -url http://127.0.0.1:8080 -key your_key -target http://127.0.0.1:3000

# 从本地目录加载 POC
./neutron/neutron.exe -path ./pocs -target http://127.0.0.1:3000

# 列出所有 POC
./neutron/neutron.exe -url http://127.0.0.1:8080 -key your_key -list

# 按严重程度过滤
./neutron/neutron.exe -url ... -key ... -target ... -severity critical,high

# 按标签过滤
./neutron/neutron.exe -url ... -key ... -target ... -tags cve,rce

# 按 source 过滤
./neutron/neutron.exe -url ... -key ... -source github -target ...

# 执行特定 POC
./neutron/neutron.exe -url ... -key ... -target ... -poc CVE-2021-12345

# 限制执行数量
./neutron/neutron.exe -url ... -key ... -target ... -max 10

# JSON 输出
./neutron/neutron.exe -url ... -key ... -target ... -json
```

### 参数说明

- `-url`: Cyberhub URL (可选)
- `-key`: Cyberhub API Key (与 -url 一起使用)
- `-source`: 按来源过滤 POC (可选)
- `-path`: 本地 POC 目录或文件 (可选)
- `-target`: 目标 URL **(扫描时必需)**
- `-poc`: 指定 POC ID (可选)
- `-list`: 列出所有 POC
- `-severity`: 按严重程度过滤 (info/low/medium/high/critical)
- `-tags`: 按标签过滤
- `-timeout`: 请求超时时间 (默认: 10秒)
- `-max`: 最大执行 POC 数量 (0 = 全部)
- `-json`: JSON 格式输出

---

## 3. GoGo - 端口扫描工具

扫描端口并识别服务指纹。

### 基本用法

```bash
# 基本扫描
./gogo/gogo.exe -target 127.0.0.1 -ports 80,443,8080

# CIDR 扫描
./gogo/gogo.exe -target 192.168.1.0/24 -ports 80,443

# 使用 Cyberhub 指纹
./gogo/gogo.exe -url http://127.0.0.1:8080 -key your_key -target 127.0.0.1

# 同时加载 Fingers 和 Neutron POC
./gogo/gogo.exe -url ... -key ... -fingers -neutron -target 127.0.0.1

# 按 source 过滤
./gogo/gogo.exe -url ... -key ... -source github -target 127.0.0.1

# 高级选项
./gogo/gogo.exe -target 127.0.0.1 -ports 1-65535 -threads 2000 -version 2

# JSON 输出
./gogo/gogo.exe -target 127.0.0.1 -json
```

### 参数说明

- `-url`: Cyberhub URL (可选)
- `-key`: Cyberhub API Key (与 -url 一起使用)
- `-source`: 按来源过滤 (可选)
- `-fingers`: 从 Cyberhub 加载指纹 (默认: true)
- `-neutron`: 从 Cyberhub 加载 POC (默认: false)
- `-target`: 目标 IP 或 CIDR **(必需)**
- `-ports`: 端口列表 (默认: 80,443,8080,8443)
- `-threads`: 线程数 (默认: 1000)
- `-version`: 版本识别级别 0-3 (默认: 0)
- `-exploit`: 漏洞检测模式 (none/all/known)
- `-timeout`: 超时时间 (默认: 5秒)
- `-json`: JSON 格式输出
- `-v`: 详细输出

---

## 4. Spray - HTTP 批量探测工具

批量检测 HTTP 服务状态。

### 基本用法

```bash
# 单个 URL
./spray/spray.exe -u http://127.0.0.1:3000

# 从文件读取 URL
./spray/spray.exe -f urls.txt

# 自定义线程和超时
./spray/spray.exe -f urls.txt -threads 100 -timeout 5

# 匹配特定状态码
./spray/spray.exe -f urls.txt -mc 200,301,302

# 过滤特定状态码
./spray/spray.exe -f urls.txt -fc 404,403

# 使用代理
./spray/spray.exe -u http://example.com -proxy http://127.0.0.1:8080

# 静默模式（只显示匹配的 URL）
./spray/spray.exe -f urls.txt -q

# JSON 输出
./spray/spray.exe -f urls.txt -json

# 保存结果到文件
./spray/spray.exe -f urls.txt -o results.txt
```

### 参数说明

- `-u`: 单个目标 URL
- `-f`: 包含 URL 的文件 (每行一个)
- `-threads`: 线程数 (默认: 50)
- `-timeout`: 超时时间 (默认: 10秒)
- `-retries`: 最大重试次数 (默认: 0)
- `-method`: HTTP 方法 (默认: GET)
- `-headers`: 自定义请求头 (格式: 'Key1:Value1,Key2:Value2')
- `-ua`: 自定义 User-Agent
- `-proxy`: 代理 URL
- `-follow`: 跟随重定向 (默认: true)
- `-fc`: 过滤状态码 (逗号分隔)
- `-mc`: 匹配状态码 (逗号分隔)
- `-fs`: 过滤大小
- `-ms`: 匹配大小
- `-json`: JSON 格式输出
- `-v`: 详细输出
- `-q`: 静默模式
- `-o`: 输出文件

---

## 测试示例

### Fingers

```bash
# 测试本地服务
./fingers/fingers.exe -target http://127.0.0.1:3000

# 测试远程服务
./fingers/fingers.exe -target http://127.0.0.1:8080
```

### Neutron

```bash
# 扫描本地服务
./neutron/neutron.exe -url http://127.0.0.1:8080 -key your_key -target http://127.0.0.1:3000 -max 5

# 列出 critical 级别的 POC
./neutron/neutron.exe -url http://127.0.0.1:8080 -key your_key -list -severity critical
```

### GoGo

```bash
# 扫描本地端口
./gogo/gogo.exe -target 127.0.0.1 -ports 3000,8080

# 使用 Cyberhub 数据
./gogo/gogo.exe -url http://127.0.0.1:8080 -key your_key -target 127.0.0.1 -ports 80,443,3000,8080
```

### Spray

```bash
# 创建测试文件
echo "http://127.0.0.1:3000" > test_urls.txt
echo "http://127.0.0.1:8080" >> test_urls.txt

# 批量探测
./spray/spray.exe -f test_urls.txt

# 只显示 200 状态的 URL
./spray/spray.exe -f test_urls.txt -mc 200 -q
```

---

## 常见问题

### Q: 如何获取 Cyberhub API Key?

A: 在 Cyberhub 管理界面的设置页面生成 API Key。

### Q: 支持哪些数据源?

A:
- **本地**: 使用内置数据或本地文件
- **Cyberhub**: 从 Cyberhub 服务加载数据
- **Source 过滤**: 可以按 source 字段过滤（如 github, local 等）

### Q: JSON 输出格式是什么?

A: 所有工具都支持 `-json` 参数，输出标准 JSON 格式，方便集成到自动化工作流。

### Q: 如何提高扫描速度?

A:
- 增加 `-threads` 参数值
- 减少 `-timeout` 时间
- 使用 `-max` 限制扫描数量

---

## 开发说明

这些工具都是基于 Chainreactors SDK 开发的简单封装。如果你需要更复杂的功能，可以：

1. 直接使用 SDK 编写自定义代码
2. 修改这些示例工具的源码
3. 参考 SDK 文档创建新工具

完整 SDK 文档: [Chainreactors SDK](https://github.com/chainreactors/sdk)

---

## License

MIT
