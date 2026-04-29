# tokrouter

**你的 LLM 聚合器**

一个配置文件，聚合所有 LLM API。

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Build](https://img.shields.io/github/actions/workflow/status/tokflux/tokrouter/release.yml?style=flat-square)](https://github.com/tokflux/tokrouter/actions)
[![Version](https://img.shields.io/badge/Version-v0.7.3-blue?style=flat-square)]()

[English Documentation](README.md)

---

## 4 行配置上线

```yaml
# config.yaml - 简化配置，使用预设
keys:
  - provider: openai
    secret: "${OPENAI_API_KEY}"

# 启动
tr start
# 网关就绪：http://127.0.0.1:8765
```

**提供商预设** — 只需指定 `provider` 和 `secret`，其他自动填充：
- 国际：openai, anthropic, google, mistral, cohere, groq, deepseek
- 国内：zhipu, qwen, tencent, baidu, moonshot, minimax, siliconflow, yi...
- 平台：together, replicate, openrouter

---

## 问题背景

多 LLM 提供商管理面临的挑战：

| 挑战 | 影响 |
|------|------|
| **密钥分散** | 密钥散落在各工具，无统一管理 |
| **成本不可见** | 不知道每个提供商花了多少钱 |
| **手动切换** | 配额用完需手动换提供商 |
| **格式锁定** | OpenAI SDK 无法调用 Anthropic，反之亦然 |
| **无降级** | 一个提供商失败，整个流程中断 |

---

## 解决方案

tokrouter 给你统一掌控：

| tokrouter | 手动管理 |
|-----------|----------|
| 单一 YAML 配置 | 密钥散落各处 |
| 自动成本追踪 | 看不到实际花费 |
| 自动故障转移 | 手动切换提供商 |
| Anthropic ↔ OpenAI 透明转换 | SDK 锁定 |
| 熔断器 + 重试 | 单点故障 |

---

## 安装

### 下载预编译二进制（推荐）

从 [GitHub Releases](https://github.com/tokflux/tokrouter/releases) 下载：

**Linux**
```bash
curl -sL https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-linux-amd64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**macOS (M1/M2)**
```bash
curl -sL https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-darwin-arm64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**Windows**
```powershell
Invoke-WebRequest -Uri "https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-windows-amd64.exe" -OutFile "tokrouter.exe"
```

验证安装：
```bash
tokrouter version
```

### 从源码构建

```bash
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter
```

---

## Docker 部署

### 快速开始

```bash
mkdir tokrouter-deploy && cd tokrouter-deploy

curl -O https://raw.githubusercontent.com/tokflux/tokrouter/main/docker-compose.yaml
curl -O https://raw.githubusercontent.com/tokflux/tokrouter/main/config.example.yaml
mv config.example.yaml config.yaml

vim config.yaml  # 设置 API Keys

export OPENAI_API_KEY=sk-xxx
export ANTHROPIC_API_KEY=sk-xxx
docker compose up -d

curl http://localhost:8765/health
```

### 手动 Docker 运行

```bash
docker run -d \
  --name tokrouter \
  -p 8765:8765 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v tokrouter_data:/app/data \
  -e OPENAI_API_KEY=sk-xxx \
  ghcr.io/tokflux/tokrouter:latest
```

---

## 快速开始

```bash
# 安装
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter

# 简化配置（使用预设）
cat > config.yaml << 'EOF'
keys:
  - provider: openai
    secret: "${OPENAI_API_KEY}"
  - provider: anthropic
    secret: "${ANTHROPIC_API_KEY}"
  - provider: deepseek
    secret: "${DEEPSEEK_API_KEY}"
EOF

tr start
```

**或使用交互式初始化：**

```bash
tr add       # 交互式：添加服务（预设或自定义）
tr start
```

---

## 适用人群

| 用户 | 用途 |
|------|------|
| **独立开发者** | 所有 AI 工具共用一套配置（Claude Code、aider、Cursor） |
| **AI 研究者** | 跨提供商实验，无需改代码 |
| **重度用户** | 最大化配额、最小化成本、自动降级 |
| **自托管者** | 无云依赖，数据完全掌控 |

---

## 架构

```
┌─────────────────────────────────────────────┐
│                 tokrouter                    │
│           "你的 LLM 聚合器"                  │
├─────────────────────────────────────────────┤
│                                             │
│  usage/             成本追踪 (SQLite)       │
│  router/            应用服务                │
│  server/            HTTP 服务器 + 处理器    │
│  config/            YAML 加载               │
│  cli/               命令行入口              │
│                                             │
├─────────────────────────────────────────────┤
│                                             │
│              fluxcore (领域层)              │
│                                             │
│  Router             领域服务                │
│  ServiceEndpoint    网络熔断聚合            │
│  Route              模型熔断聚合            │
│  RouteTable         预计算快照              │
│  RouteRepository    熔断状态持久化          │
│  message/           请求/响应中间表示       │
│  errors/            错误分类                │
│  translate/         协议转换                │
│                                             │
└─────────────────────────────────────────────┘

1 个二进制。1 个配置文件。0 依赖。
```

---

## 工作原理

```go
// router/router.go - 聚合服务
type routerCtx struct {
    oaTables   map[fluxcore.Model]*fluxcore.RouteTable
    anthTables map[fluxcore.Model]*fluxcore.RouteTable
    retryMax   int
}

type router struct {
    mu         sync.RWMutex
    ctx        *routerCtx                     // atomic.Pointer 重载时交换
    oaRouter   *fluxcore.Router
    anthRouter *fluxcore.Router
    svcEPs     map[string]*fluxcore.ServiceEndpoint
    routeRepo  *fluxcore.RouteRepository      // 熔断状态跨重载保留
    usageSvc   *usage.Service
}

func (r *router) ForwardOpenAI(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error) {
    r.mu.RLock()
    table := r.ctx.oaTables[fluxcore.Model(model)]
    retryMax := r.ctx.retryMax
    r.mu.RUnlock()

    if table == nil {
        return nil, nil, errors.New("no route for model")
    }

    route, resp, usage, err := r.oaRouter.Execute(ctx, table, body, retryMax)
    if err != nil {
        return nil, nil, err
    }
    r.usageSvc.RecordWithModelAndProvider(usage, model, route.SvcEP().Service().Name, false)
    return resp, usage, nil
}
```

所有路由逻辑（路由选择、重试、降级、健康反馈）由 fluxcore 的 `Router.Execute` 处理。

---

## 协议转换

```bash
# 用 Anthropic SDK 调用 OpenAI 后端
export ANTHROPIC_BASE_URL=http://127.0.0.1:8765

# 用 OpenAI SDK 调用 Anthropic 后端
export OPENAI_API_BASE=http://127.0.0.1:8765/v1

# 格式自动转换
# 你的 SDK 无感知
```

---

## 熔断器

fluxcore 双层熔断：

```
ServiceEndpoint 层（网络）：
  DNS / 连接拒绝 / 超时 → 立即熔断（阈值=1）
  恢复：120s

Route 层（模型）：
  429 限流 → 熔断（累计阈值=3）
  500 服务错误 → 熔断（累计阈值=3）
  恢复：60s

tokrouter 自动切换到下一个健康路由。
熔断状态通过 RouteRepository 在配置重载（SIGHUP）后保留。
```

---

## 模型级路由

请求只路由到匹配模型的端点：

```yaml
keys:
  - name: openai-main
    models:
      - name: gpt-4
        priority: 100
      - name: gpt-3.5-turbo
        priority: 10
```

请求 `gpt-4` → 只路由到 gpt-4 端点（不会路由到 gpt-3.5-turbo）。

### 模型别名

将请求模型名映射到实际模型名：

```yaml
keys:
  - name: openai-main
    models:
      - name: gpt-4-turbo
        alias: gpt-4-1106-preview  # 请求 "gpt-4-turbo" → 实际用 "gpt-4-1106-preview"
        priority: 50
```

---

## 热重载

无需重启即可重载配置。熔断状态通过 `RouteRepository.FindOrCreate()` 跨重载保留。

```bash
kill -SIGHUP $(pidof tokrouter)
```

---

## 延迟感知路由

端点选择策略：
1. **优先级**（低优先）
2. **EWMA 延迟**（近期延迟权重更高）

自动避开慢端点。

---

## 成本追踪

```bash
tokrouter status

Key            Format    Healthy    Models
openai-main    openai    OK         2/2
anthropic-main anthropic OK         1/1

tokrouter summary --month
Key            Input      Output     Requests   Avg Latency  Success
openai-main    152340     45678     1234       245ms        98.5%
anthropic-main 23456      12345     567        189ms        99.2%
```

---

## CLI 命令

```bash
# 快速开始
tr add openai --secret sk-xxx        # 使用预设添加服务
tr start                             # 启动服务器 127.0.0.1:8765
tr start --port 8080                 # 自定义端口

# 服务管理
tr add                               # 交互式：选择预设或自定义
tr add deepseek --secret sk-xxx      # 预设模式，自动填充配置
tr add --name my --base-url ... --format openai --secret sk-xxx --model gpt-4
tr remove <name>                     # 删除服务
tr config <name> --enable            # 启用/禁用服务
tr config <name> --secret sk-new     # 更新 API 密钥
tr config <name> --add-model gpt-4   # 添加模型到服务
tr config <name> --remove-model old  # 从服务移除模型

# 查看信息
tr list services                     # 列出所有服务（默认）
tr list models                       # 列出所有可用模型
tr list presets                      # 列出提供者预设（26 个内置）
tr list assistants                   # 列出支持的 AI 工具
tr show service <name>               # 服务详情
tr show preset <name>                # 预设详情
tr show config                       # 当前配置
tr show health                       # 端点健康状态
tr show health --watch               # 实时刷新（2 秒）
tr show usage --month                # 月度用量统计
tr show usage --chart                # Token 分布图
tr show usage --export csv           # 导出 CSV

# AI 助手集成
tr assistant list                    # 列出支持的 AI 工具
tr assistant auto                    # 自动检测并配置所有
tr assistant cursor                  # 配置指定工具

# 服务器生命周期
tr start [--host HOST] [--port PORT] # 启动服务器
tr stop                              # 优雅关闭服务器

# Shell 补全
tr completion bash|zsh|fish          # 生成补全脚本
```

---

## API 端点

| 端点 | 格式 | 说明 |
|------|------|------|
| `POST /v1/chat/completions` | OpenAI | OpenAI 兼容，支持流式 |
| `POST /v1/messages` | Anthropic | Anthropic 兼容，支持流式 |
| `GET /status` | JSON | 密钥状态 |
| `GET /health` | JSON | 健康检查（含依赖状态） |
| `GET /openapi.yaml` | YAML | OpenAPI 3.0 规范 |
| `GET /docs` | HTML | Swagger UI 文档 |

---

## AI 工具集成

**Claude Code**（Anthropic 格式）：
```json
// ~/.claude/settings.json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8765"
  }
}
```

**aider**（OpenAI 格式）：
```yaml
# ~/.aider.conf.yml
openai-api-base: http://127.0.0.1:8765/v1
model: gpt-4
```

**Cursor / VS Code Copilot**：
```json
// 设置
"openai.apiBase": "http://127.0.0.1:8765/v1"
```

---

## 协议支持

| 格式 | 端点 | 转换 |
|------|------|------|
| **OpenAI** | `/v1/chat/completions` | 原生 |
| **Anthropic** | `/v1/messages` | 双向转换 |
| **Gemini** | 提供商配置 | fluxcore 自动处理 |
| **Cohere** | 提供商配置 | fluxcore 自动处理 |

**OpenAI 兼容提供商**：

| 提供商 | Base URL |
|------|----------|
| 智谱 GLM-4 | `https://open.bigmodel.cn/api/paas/v4` |
| 阿里通义 | DashScope API |
| DeepSeek | `https://api.deepseek.com` |
| Mistral | `https://api.mistral.ai` |
| Groq | `https://api.groq.com` |

---

## 目录结构

```
tokrouter/
├── usage/            # 成本追踪 (SQLite)
│   ├── record.go     # 使用记录实体
│   ├── query.go      # QueryFilter, StatRow
│   ├── service.go    # 使用量服务
│   ├── sqlite.go     # SQLite 存储
│   ├── errors.go     # 使用量错误
│   ├── doc.go        # 包文档
│   └── *_test.go     # 测试 (sqlite, query, service)
│
├── router/           # 聚合服务
│   ├── router.go     # 路由服务
│   ├── doc.go        # 包文档
│   └── *_test.go     # 测试 (单元 + 并发)
│
├── server/           # HTTP 服务器
│   ├── handler.go    # API 处理器
│   ├── server.go     # HTTP 服务器
│   ├── log.go        # 请求日志 (含脱敏)
│   ├── errors.go     # HTTP 错误响应
│   ├── doc.go        # 包文档
│   └── *_test.go     # 测试 (handler, log, errors)
│
├── config/           # 配置
│   ├── config.go     # YAML 配置加载器
│   ├── doc.go        # 包文档
│   └── *_test.go     # 测试
│
├── cli/              # 命令行
│   ├── root.go       # CLI 入口
│   ├── cmd_init.go   # init 命令
│   ├── cmd_serve.go  # serve 命令
│   ├── cmd_status.go # status 命令
│   ├── cmd_keys.go   # keys 命令
│   ├── cmd_summary.go # summary 命令
│   ├── cmd_config.go # config 命令
│   ├── cmd_models.go # models 命令
│   ├── doc.go        # 包文档
│
├── cmd/tokrouter/    # 入口点
│   └── main.go
│
└── config.example.yaml
```

---

## 常见问题

**Q: 如何添加新的 API 密钥？**
```bash
tokrouter keys add --name my-key --format openai --secret $MY_KEY --base-url https://api.example.com/v1
```

**Q: 如何测试密钥是否可用？**
```bash
tokrouter keys ping openai-main
```

**Q: priority 如何理解？**

`priority: 100` 表示端点优先级，**越低越优先**。默认为 0（最高优先）。用于多端点时的初始选择，运行时由延迟动态调整。

**Q: 如何用 Claude Code 连接 tokrouter？**
```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:8765
claude
```

**Q: 如何用 aider 连接 tokrouter？**
```bash
export OPENAI_API_BASE=http://127.0.0.1:8765/v1
aider --model gpt-4
```

**Q: 为什么端口是 8765？**

端口 7890 与 Clash 代理冲突。8765 不常用，更安全。

**Q: 支持流式响应吗？**

支持，OpenAI 和 Anthropic 格式都完全支持流式。

**Q: 自动降级如何工作？**

双层保护：网络错误立即触发服务端点熔断（阈值=1，恢复=120s）。模型错误（429/5xx）累计 3 次触发路由熔断（恢复=60s），路由器自动切换到下一个健康路由。

---

## 快速上手

```bash
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter

tr add
tr start
```

**下一步：**
1. Star 本仓库
2. 运行 `tr add` 配置
3. 将 AI 工具指向 `http://127.0.0.1:8765`

---

## 相关项目

| 项目 | 说明 |
|------|------|
| **fluxcore** | 基于 DDD 的 LLM 路由引擎（ServiceEndpoint, Route, RouteTable, Router） |

---

## 许可证

MIT。永久免费。

---

**tokrouter - 你的 LLM 聚合器。一个配置，一个二进制，完全掌控。**