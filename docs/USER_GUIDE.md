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
server:
  host: "127.0.0.1"
  port: 8765

keys:
  - name: "openai"
    base_url: "https://api.openai.com/v1"
    secret: "${OPENAI_API_KEY}"
    format: "openai"
    enabled: true
    models:
      - name: "gpt-4"
        pricing:
          input: 0.03
          output: 0.06

  - name: "anthropic"
    base_url: "https://api.anthropic.com/v1"
    secret: "${ANTHROPIC_API_KEY}"
    format: "anthropic"
    enabled: true
    models:
      - name: "claude-3-opus"
        pricing:
          input: 0.015
          output: 0.075
```

### 启动服务
```bash
./tokrouter serve --config config.yaml
```

## 配置详解

### Server 配置
| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| host | string | "127.0.0.1" | 监听地址 |
| port | int | 8765 | 监听端口 |
| tls_cert | string | "" | TLS 证书路径 |
| tls_key | string | "" | TLS 密钥路径 |

### Key 配置
| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | Key 标识 |
| base_url | string | API 基础 URL |
| secret | string | API 密钥（支持环境变量 `${VAR}`） |
| format | string | 协议格式: openai/anthropic/gemini/cohere |
| enabled | bool | 是否启用 |
| models | []ModelConfig | 支持的模型列表 |

### Model 配置
| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 模型名称 |
| alias | string | 模型别名（可选，将请求模型名映射到实际模型名） |
| pricing.input | float | 输入价格（$/1K tokens） |
| pricing.output | float | 输出价格（$/1K tokens） |

### Router 配置
| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| retry.max_retries | int | 2 | 最大重试次数 |
| retry.timeout | string | "30s" | 请求超时 |
| retry.backoff | string | "exponential" | 退避策略 |

### Stats 配置
| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| enabled | bool | true | 是否启用统计 |
| db_path | string | "./data/usage.db" | 数据库路径 |

## CLI 命令

### tokrouter init
交互式配置向导
```bash
./tokrouter init
```

### tokrouter serve
启动服务器
```bash
./tokrouter serve                   # 默认配置
./tokrouter serve --port 9000       # 自定义端口
```

### tokrouter status
查看 Key 状态
```bash
./tokrouter status              # 显示当前状态
./tokrouter status --watch      # 实时刷新（每 2 秒）
```

### tokrouter keys
管理 API 密钥
```bash
./tokrouter keys                 # 列出所有密钥
./tokrouter keys list            # 同上
./tokrouter keys add             # 添加密钥（交互式）
./tokrouter keys remove <name>   # 删除密钥
./tokrouter keys enable <name>   # 启用密钥
./tokrouter keys disable <name>  # 禁用密钥
./tokrouter keys ping <name>     # 测试密钥连通性
```

### tokrouter summary
查看使用统计
```bash
./tokrouter summary             # 本月统计
./tokrouter summary --today     # 今日统计
./tokrouter summary --week      # 本周统计
./tokrouter summary --chart     # 显示 ASCII 图表
./tokrouter summary --export csv > stats.csv
./tokrouter summary --export json > stats.json
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
        pricing: {input: 0.03, output: 0.06}
  - name: "openai-2"
    base_url: "https://api.openai.com/v1"
    secret: "sk-xxx2"
    format: "openai"
    enabled: true
    models:
      - name: "gpt-4"
        pricing: {input: 0.03, output: 0.06}
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
        pricing: {input: 0.01, output: 0.03}
```

### 热重载
无需重启即可重载配置:
```bash
kill -SIGHUP $(pidof tokrouter)
```

### 延迟感知路由
端点选择策略：
1. 价格优先（低价优先）
2. 价格相同时，按 EWMA 延迟选择（近期延迟权重更高）

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