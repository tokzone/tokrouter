# TokRouter 用户手册

## 快速开始

### 安装
```bash
# 从源码编译
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build -o tokrouter ./cmd/tokrouter
```

### 基本配置
创建 `config.yaml`:
```yaml
keys:
  - name: openai-main
    base_url: "https://api.openai.com/v1"
    secret: "${OPENAI_API_KEY}"
    format: openai
    enabled: true
    models:
      - name: gpt-4
      - name: gpt-3.5-turbo

  - name: anthropic-main
    base_url: "https://api.anthropic.com/v1"
    secret: "${ANTHROPIC_API_KEY}"
    format: anthropic
    enabled: true
    models:
      - name: claude-3-opus
```

**说明**：以上配置已足够运行。其他配置项（server、router、stats、log、trace）均有默认值，按需覆盖即可。

### 启动服务
```bash
./tokrouter serve --config config.yaml
```

## 配置详解

### 必需配置

**keys** - API 密钥配置（必需，至少一个）

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| name | string | ✓ | Key 标识 |
| base_url | string | ✓ | API 基础 URL |
| secret | string | ✓ | API 密钥（支持环境变量 `${VAR}`） |
| format | string | ✓ | 协议格式: openai/anthropic/gemini/cohere |
| enabled | bool | - | 是否启用（默认 true） |
| models | []ModelConfig | ✓ | 支持的模型列表（至少一个） |

**Model 配置**

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| name | string | ✓ | 模型名称 |
| alias | string | - | 模型别名（将请求模型名映射到实际模型名） |
| priority | int64 | - | 端点优先级（越低越优先，默认 0） |

### 可选配置（均有默认值）

**Server 配置**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| host | "127.0.0.1" | 监听地址 |
| port | 8765 | 监听端口 |
| tls_cert | "" | TLS 证书路径 |
| tls_key | "" | TLS 密钥路径 |

**Router 配置**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| retry.max_retries | 2 | 最大重试次数 |
| retry.timeout | "30s" | 请求超时 |
| retry.backoff | "exponential" | 退避策略 |

**Stats 配置**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| enabled | true | 是否启用统计 |
| db_path | "./data/usage.db" | 数据库路径 |

**Log 配置**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| level | "info" | 日志级别 (debug/info/warn/error) |
| format | "json" | 日志格式 (json/text) |
| output | "stdout" | 输出位置 |

**Trace 配置**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| enabled | true | 是否启用追踪 |
| header | "x-request-id" | 追踪请求头 |

## CLI 命令

### tokrouter init
交互式配置向导，包含输入验证和配置确认步骤：
```bash
./tokrouter init
```

新功能：
- 端口范围验证（1-65535）
- URL 格式验证
- 配置摘要显示
- 保存前确认步骤
- 按 Ctrl+C 可随时取消

### tokrouter serve
启动服务器
```bash
./tokrouter serve                   # 默认配置 (127.0.0.1:8765)
./tokrouter serve --host 0.0.0.0    # 监听所有接口
./tokrouter serve --port 9000       # 自定义端口
```

### tokrouter status
查看 Key 状态
```bash
./tokrouter status              # 显示当前状态
./tokrouter status --watch      # 实时刷新（每 2 秒）
```

### tokrouter models
列出所有可用模型（新命令）：
```bash
./tokrouter models
```

输出示例：
```
Available Models
┌─────────────────┬────────────┬──────────┬──────────┬─────────┐
│ MODEL           │ PROVIDER   │ FORMAT   │ PRIORITY │ STATUS  │
├─────────────────┼────────────┼──────────┼──────────┼─────────┤
│ gpt-4           │ openai     │ openai   │ 0        │ enabled │
│ claude-3-opus   │ anthropic  │ anthropic│ 50       │ enabled │
└─────────────────┴────────────┴──────────┴──────────┴─────────┘
```

### tokrouter keys
管理 API 密钥
```bash
./tokrouter keys                 # 列出所有密钥
./tokrouter keys list            # 同上
./tokrouter keys add             # 添加密钥（交互式模式）
./tokrouter keys add --name my-key --format openai --secret $KEY --base-url https://api.example.com/v1  # 非交互式
./tokrouter keys remove <name>   # 删除密钥
./tokrouter keys enable <name>   # 启用密钥
./tokrouter keys disable <name>  # 禁用密钥
./tokrouter keys ping <name>     # 测试密钥连通性（显示延迟和测试汇总）
```

`keys ping` 输出改进：
```
Testing key 'openai-main'...
  Format:  openai
  BaseURL: https://api.openai.com/v1

  Testing model 'gpt-4'... ✓ OK (245ms)
  Testing model 'gpt-3.5-turbo'... ✓ OK (189ms)

Summary: 2 passed, 0 failed
```

### tokrouter summary
查看使用统计（含平均延迟和成功率）
```bash
./tokrouter summary             # 本月统计
./tokrouter summary --today     # 今日统计
./tokrouter summary --week      # 本周统计
./tokrouter summary --chart     # 显示 ASCII 图表
./tokrouter summary --export csv > stats.csv
./tokrouter summary --export json > stats.json
```

输出示例：
```
Usage Summary (month)
┌────────────────┬───────┬────────┬──────────┬────────────┬─────────┐
│ KEY            │ INPUT │ OUTPUT │ REQUESTS │ AVG LATENCY│ SUCCESS │
├────────────────┼───────┼────────┼──────────┼────────────┼─────────┤
│ openai-main    │ 15234 │ 45678  │ 1234     │ 245ms      │ 98.5%   │
│ anthropic-main │ 23456 │ 12345  │ 567      │ 189ms      │ 99.2%   │
└────────────────┴───────┴────────┴──────────┴────────────┴─────────┘
```

### tokrouter config
显示当前配置
```bash
./tokrouter config
```

## 使用场景

### 负载均衡
配置多个相同模型的端点，自动轮询:
```yaml
keys:
  - name: "openai-1"
    base_url: "https://api.openai.com/v1"
    secret: "sk-xxx1"
    format: "openai"
    enabled: true
    models:
      - name: "gpt-4"
        priority: 100
  - name: "openai-2"
    base_url: "https://api.openai.com/v1"
    secret: "sk-xxx2"
    format: "openai"
    enabled: true
    models:
      - name: "gpt-4"
        priority: 200  # Higher priority = less preferred
```

### 故障转移
当某个端点失败时，自动切换到其他端点。熔断器会在连续失败 3 次后暂时屏蔽端点，60 秒后自动恢复。

### 成本监控
启用统计后，使用 summary 命令查看各提供者的使用量和成本:
```bash
./tokrouter summary --chart
```

### 协议转换
使用 Anthropic SDK 调用 OpenAI 后端:
```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:8765
claude  # Claude Code 会自动使用 tokrouter
```

使用 OpenAI SDK 调用 Anthropic 后端:
```bash
export OPENAI_API_BASE=http://127.0.0.1:8765/v1
aider --model gpt-4
```

### 模型级路由
请求只路由到匹配模型的端点。配置多个模型时，请求会自动路由到对应模型的端点。

### 模型别名
将请求模型名映射到实际模型名:
```yaml
keys:
  - name: "openai"
    models:
      - name: "gpt-4-turbo"
        alias: "gpt-4-1106-preview"  # 请求 gpt-4-turbo → 实际用 gpt-4-1106-preview
        priority: 50
```

### 热重载
无需重启即可重载配置:
```bash
kill -SIGHUP $(pidof tokrouter)
```

### 延迟感知路由
端点选择策略：
1. 优先级优先（低优先级优先）
2. 优先级相同时，按 EWMA 延迟选择（近期延迟权重更高）

自动避开响应慢的端点。

## 故障排查

### 日志级别
```yaml
log:
  level: "debug"  # debug/info/warn/error
  format: "json"  # json/text
  output: "stdout"
```

### 常见错误

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| connection refused | 服务未启动 | 检查 serve 命令输出 |
| invalid API key | 密钥无效 | 检查 key.secret |
| timeout | 上游响应慢 | 增加 retry.timeout |
| no healthy endpoints | 所有端点不可用 | 检查网络和密钥 |
| config invalid: no keys | 未配置密钥 | 添加 keys 配置 |

### 健康检查
```bash
curl http://localhost:8765/health
curl http://localhost:8765/status
```

### 调试模式
```bash
# 设置日志级别
log:
  level: "debug"
```