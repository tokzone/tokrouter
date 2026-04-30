# tokrouter

一个二进制文件管理所有 LLM API。命令行完成全部配置，无需手写文件。

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Build](https://img.shields.io/github/actions/workflow/status/tokzone/tokrouter/release.yml?style=flat-square)](https://github.com/tokzone/tokrouter/actions)
[![Version](https://img.shields.io/badge/Version-v0.7.4-blue?style=flat-square)]()

[English Documentation](README.md)

---

## 安装

### 下载预编译二进制（推荐）

**Linux**
```bash
curl -sL https://github.com/tokzone/tokrouter/releases/latest/download/tokrouter-linux-amd64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**macOS (Apple Silicon)**
```bash
curl -sL https://github.com/tokzone/tokrouter/releases/latest/download/tokrouter-darwin-arm64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**Windows (PowerShell)**
```powershell
Invoke-WebRequest -Uri "https://github.com/tokzone/tokrouter/releases/latest/download/tokrouter-windows-amd64.exe" -OutFile "tokrouter.exe"
```

验证安装：
```bash
tokrouter version
```

### 从源码编译

```bash
git clone https://github.com/tokzone/tokrouter.git && cd tokrouter
go build -o tokrouter ./cmd/tokrouter
```

### Docker 部署

```bash
docker run -d \
  --name tokrouter \
  -p 8765:8765 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v tokrouter_data:/app/data \
  -e OPENAI_API_KEY=sk-xxx \
  ghcr.io/tokzone/tokrouter:latest
```

---

## 推荐流程

安装完成后，两步上线：

### 第 1 步：一键配置

```bash
./tokrouter assistant auto --url http://127.0.0.1:8765
```

自动检测本机已安装的 AI 编程工具（Claude Code、Cursor、Aider、Codex CLI 等），列出检测结果。接着引导你从 26 个预设中选择默认模型——选了未配置的服务商会提示填入 API Key，自动完成添加。确认后一键写入所有工具的配置。

### 第 2 步：启动服务

```bash
./tokrouter start
# 网关运行在 http://127.0.0.1:8765
```

验证：
```bash
curl http://127.0.0.1:8765/health
```

至此你的 AI 工具已经全部指向 tokrouter，打开即可使用。后续需要添加更多 Key 时，用 `./tokrouter add` 交互式添加。

---

## assistant auto 详解

`assistant auto` 是 tokrouter 的核心功能：自动检测本机已安装的 AI 编程工具，引导选择默认模型（未配置的服务商会提示填入 API Key 自动添加），一键将所有工具指向 tokrouter。

### 支持的 AI 工具

| 工具 | 协议 | 配置方式 | 写入位置 |
|------|------|---------|---------|
| **Claude Code** | Anthropic Messages | 环境变量 | Shell profile (`~/.zshrc` 等) |
| **Codex CLI** | Responses API | TOML | `~/.codex/config.toml` |
| **Cursor** | OpenAI Chat | JSON | `~/.cursor/config.json` |
| **Aider** | OpenAI Chat | 环境变量 | Shell profile (`~/.zshrc` 等) |
| **Windsurf** | OpenAI Chat | JSON | `~/.windsurf/config.json` |
| **Cline** | OpenAI Chat | JSON | VS Code `settings.json` |
| **Continue.dev** | OpenAI Chat | JSON | `~/.continue/config.json` |

Codex CLI 特别说明：新版使用 OpenAI Responses API，生成的 TOML 自动包含 `wire_api = "responses"`。

### 单独配置某个工具

```bash
./tokrouter assistant claude-code --url http://127.0.0.1:8765
./tokrouter assistant codex --url http://127.0.0.1:8765
./tokrouter assistant cursor --url http://127.0.0.1:8765
```

---

## 预设服务商

支持 26 个内置预设。预设自动填充 API 地址、协议格式、默认模型，无需手动指定：

```
openai         anthropic      google         mistral
cohere         groq           deepseek       zhipu
qwen           tencent        baidu          qianfan
huawei         moonshot       minimax        siliconflow
yi             stepfun        baichuan       xunfei
doubao         parallel       modelscope     together
replicate      openrouter
```

```bash
./tokrouter show preset deepseek    # 显示 base_url、format、默认模型
./tokrouter list presets            # 列出全部预设
```

---

## 命令速查

### 配置服务

```bash
./tokrouter add                              # 交互式添加
./tokrouter add <preset> --secret sk-xxx     # 直接指定预设
./tokrouter remove <name>                    # 删除服务
./tokrouter config <name> --enable           # 启用
./tokrouter config <name> --disable          # 禁用
./tokrouter config <name> --secret sk-new    # 更新 Key
./tokrouter config <name> --add-model gpt-4  # 添加模型
./tokrouter config <name> --remove-model old # 移除模型
```

### 运行网关

```bash
./tokrouter start                     # 默认 127.0.0.1:8765
./tokrouter start --port 8080         # 指定端口
./tokrouter start --host 0.0.0.0      # 监听所有网卡
./tokrouter stop                      # 停止服务
kill -SIGHUP $(pidof tokrouter)       # 热加载配置，不中断服务
```

### 查看信息

```bash
./tokrouter list services             # 当前配置的服务
./tokrouter list models               # 全部可用模型
./tokrouter list presets              # 所有预设
./tokrouter list assistants           # 支持的 AI 工具
./tokrouter show health               # 端点健康状态
./tokrouter show health --watch       # 实时刷新
./tokrouter show usage --month        # 月度用量统计
./tokrouter show usage --chart        # Token 分布图
./tokrouter show usage --export csv   # 导出 CSV
```

### Shell 补全

```bash
./tokrouter completion bash           # Bash
./tokrouter completion zsh            # Zsh
./tokrouter completion fish           # Fish
```

---

## 配置文件

大部分操作可通过命令完成，直接编辑 `config.yaml` 适合批量修改。

```yaml
server:
  host: "127.0.0.1"       # 监听地址
  port: 8765               # 端口

router:
  retry:
    max_retries: 2         # 失败重试次数
    timeout: 30s
    backoff: exponential

keys:
  # 方式一：预设（推荐，自动填充 base_url 和格式）
  - name: my-deepseek     # 唯一名称，不填则用 provider 名
    provider: deepseek
    secret: "${DEEPSEEK_API_KEY}"
    models:               # 可选，不写则用预设全部模型
      - name: deepseek-v4-pro
        priority: 100
      - name: deepseek-chat
        priority: 200

  # 方式二：自定义（手动指定所有字段）
  - name: self-hosted
    base_url: https://llm.example.com/v1
    format: openai        # openai / anthropic / gemini / cohere
    secret: sk-xxx
    models:
      - name: local-model
```

### 优先级和熔断

同一模型配多个 Key 时按 `priority`（越小越优先）选择。某个 Key 故障时自动切换到下一个。

### 模型别名

```yaml
models:
  - name: gpt-4-turbo
    alias: gpt-4-1106-preview   # 对工具暴露 gpt-4-turbo，实际转发到 gpt-4-1106-preview
```

---

## 架构

```
┌─────────────────────────────────────────────┐
│                 tokrouter                    │
├─────────────────────────────────────────────┤
│  usage/             成本追踪 (SQLite)       │
│  router/            应用服务                │
│  server/            HTTP 服务器 + 处理器    │
│  config/            YAML 加载               │
│  cli/               命令行入口              │
├─────────────────────────────────────────────┤
│              fluxcore (领域层)              │
│  Router             领域服务                │
│  ServiceEndpoint    网络熔断聚合            │
│  Route              模型熔断聚合            │
│  RouteTable         预计算快照              │
│  RouteRepository    熔断状态持久化          │
│  message/           请求/响应中间表示       │
│  errors/            错误分类                │
│  translate/         协议转换                │
└─────────────────────────────────────────────┘

1 个二进制。1 个配置文件。0 依赖。
```

---

## 协议转换

```bash
# 用 Anthropic SDK 调用 OpenAI 后端
export ANTHROPIC_BASE_URL=http://127.0.0.1:8765

# 用 OpenAI SDK 调用 Anthropic 后端
export OPENAI_API_BASE=http://127.0.0.1:8765/v1

# 格式自动转换，你的 SDK 无感知
```

---

## 熔断器

双层熔断保护：

- **ServiceEndpoint 层（网络）**：DNS / 连接拒绝 / 超时 → 立即熔断（阈值=1），恢复：120s
- **Route 层（模型）**：429 限流 / 500 服务错误 → 累计熔断（阈值=3），恢复：60s

tokrouter 自动切换到下一个健康路由。熔断状态在配置重载（SIGHUP）后保留。

---

## 热重载

无需重启即可重载配置，熔断状态通过 `RouteRepository` 跨重载保留：

```bash
kill -SIGHUP $(pidof tokrouter)
```

---

## 成本追踪

```bash
./tokrouter show usage --month        # 月度统计
./tokrouter show usage --chart        # Token 分布图
./tokrouter show usage --export csv   # 导出 CSV
```

---

## 常见问题

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

**Q: priority 如何理解？**
`priority` 越低越优先。默认 0（最高优先）。用于多端点时的初始选择，运行时由延迟动态调整。

**Q: 支持流式响应吗？**
支持，OpenAI 和 Anthropic 格式都完全支持流式。

**Q: 自动降级如何工作？**
双层保护：网络错误立即触发服务端点熔断（阈值=1，恢复=120s）。模型错误（429/5xx）累计 3 次触发路由熔断（恢复=60s），路由器自动切换到下一个健康路由。

---

## API 端点

| 端点 | 协议 | 流式 |
|------|------|:----:|
| `POST /v1/chat/completions` | OpenAI Chat Completions | ✓ |
| `POST /v1/messages` | Anthropic Messages | ✓ |
| `POST /v1/responses` | OpenAI Responses（Codex CLI） | ✓ |
| `GET /v1/models` | 模型列表 | - |
| `GET /health` | 健康检查 | - |
| `GET /status` | 服务状态 | - |
| `GET /openapi.yaml` | OpenAPI 规范 | - |
| `GET /docs` | Swagger UI | - |

---

## 许可证

MIT。永久免费。
