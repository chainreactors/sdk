# Chainreactors SDK Examples

## 目录结构

```
examples/
├── engines/                        # 每个引擎一个完整示例
│   ├── fingers/     指纹识别
│   ├── gogo/        端口扫描 + 指纹识别
│   ├── spray/       HTTP 批量探测
│   ├── neutron/     POC 漏洞扫描
│   ├── zombie/      弱口令爆破
│   └── proton/      敏感数据扫描
│
├── sniper/                         # 核心攻击链示例
│   └── main.go      指纹识别 → 关联 POC → 精准攻击
│
└── cases/                          # 小粒度使用案例
    ├── match_detail/         获取 matcher 详情
    ├── spray_crawl_finger/   爬虫 + 深度指纹
    ├── request_response/     POC 请求/响应捕获
    ├── association/          关联查询
    ├── cyberhub_provider/    Provider 数据加载
    ├── filter/               过滤 API
    ├── proxy/                多级代理配置
    └── host_collision/       Host 碰撞攻击
```

---

## Engine Examples

每个引擎示例都支持本地数据和 Cyberhub 两种模式。

### Fingers - 指纹识别

```bash
# 本地数据
go run ./examples/engines/fingers -target http://example.com

# Cyberhub
go run ./examples/engines/fingers -url http://127.0.0.1:8080 -key your_key -target http://example.com
```

### Gogo - 端口扫描

```bash
# 本地数据
go run ./examples/engines/gogo -target 192.168.1.0/24 -ports 80,443 -threads 2000

# Cyberhub（自动加载 fingers + neutron）
go run ./examples/engines/gogo -url http://127.0.0.1:8080 -key your_key -target 192.168.1.1 -ports 80,443
```

### Spray - HTTP 批量探测

```bash
go run ./examples/engines/spray -u http://example.com
go run ./examples/engines/spray -f urls.txt -threads 100 -mc 200,301,302 -json
```

### Neutron - POC 漏洞扫描

```bash
# Cyberhub
go run ./examples/engines/neutron -url http://127.0.0.1:8080 -key your_key -target http://example.com

# 本地 POC
go run ./examples/engines/neutron -path ./pocs -target http://example.com

# 列出 / 过滤
go run ./examples/engines/neutron -url ... -list
go run ./examples/engines/neutron -url ... -target ... -severity critical -tags rce
```

### Zombie - 弱口令爆破

```bash
# Brute（笛卡尔积）
go run ./examples/engines/zombie -target 192.168.1.1 -service ssh -users root,admin -passwords 123456,admin

# Pitchfork（配对凭据）
go run ./examples/engines/zombie -target 192.168.1.1 -service mysql -mode pitchfork -auths root::123456,admin::admin

# Sniper（单次尝试）
go run ./examples/engines/zombie -target 192.168.1.1 -service redis -mode sniper -users root -passwords 123456
```

### Proton - 敏感数据扫描

```bash
go run ./examples/engines/proton -input config.yaml
go run ./examples/engines/proton -input - < secrets.env
```

---

## Sniper - 核心攻击链

完整的 **指纹识别 → 关联查询 → 精准攻击** 工作流：

```bash
go run ./examples/sniper -url http://127.0.0.1:8080 -key your_key -target http://192.168.1.1:8080
```

工作流程：
1. 对目标进行指纹识别，识别 Web 框架和技术栈
2. 通过关联索引查找匹配的 POC 模板
3. 仅执行关联的 POC，实现精准漏洞检测

---

## Cases

| 案例 | 说明 |
|------|------|
| `match_detail` | 获取 fingers matcher 的详细匹配信息 |
| `spray_crawl_finger` | 单 URL 爬虫 + 深度指纹探测 |
| `request_response` | 捕获 POC 执行的完整请求/响应 |
| `association` | 关联索引查询（finger↔alias↔template↔CVE） |
| `cyberhub_provider` | Cyberhub Provider 数据加载和过滤 |
| `filter` | ExportFilter 和本地过滤 API |
| `proxy` | 三级代理配置（Client > Config > Context） |
| `host_collision` | Host 碰撞（虚拟主机枚举） |
