# 让 Codex CLI 用上 DeepSeek V4 的一个土办法

Codex CLI 出来后一直在用，确实比之前 copilot 那种补全方式顺手。但有个问题很烦——它只认 OpenAI 的模型。

我 DeepSeek 账户里还有大几百的额度，V4 日常写代码完全够用，但 Codex 就是不支持。不只是 DeepSeek，试了下智谱、豆包、通义，全部不行。Claude Code 也一个德行，只认 Anthropic，我想让它走 OpenAI 的额度，没门。

搜了一圈，这类问题其实很普遍。每个 AI 编程工具都绑定了特定的协议和厂商：

- Codex CLI 只对接 OpenAI Responses API
- Claude Code 只对接 Anthropic Messages API
- Cursor/Aider/Cline 只对接 OpenAI Chat API

你有再多 Key，协议不对就白搭。

后来发现一个叫 tokrouter 的小工具，思路很简单——在你本机起一个网关，把协议翻译掉：

```
Codex (Responses API)  →  tokrouter  →  DeepSeek (OpenAI Chat)
Claude Code (Messages) →  tokrouter  →  智谱 (OpenAI Chat)
```

Codex 以为自己在跟 OpenAI 通信，实际上 tokrouter 在背后把请求转成了 DeepSeek 能认的格式，响应再转回来。Response API 到 Chat API 的转换也帮你做了。

装起来就两步：

```bash
# 安装
curl -sL https://github.com/tokzone/tokrouter/releases/latest/download/tokrouter-linux-amd64 -o tokrouter
chmod +x tokrouter && sudo mv tokrouter /usr/local/bin/

# 一键配置
tokrouter assistant auto --url http://127.0.0.1:8765
# 自动检测 Codex、Claude Code、Cursor 等 → 选 DeepSeek V4 → 填 Key → 完成
```

这个 `assistant auto` 确实省事，它自己检测你装了哪些 AI 工具，列出支持的模型让你选，没配过的厂商当场让你填 Key，然后把所有工具的配置文件一把写完。不用自己去翻 `.zshrc`、`config.json`、`config.toml` 挨个改。

然后 `tokrouter start`，搞定。

现在 Codex 和 Claude Code 都在走 DeepSeek，Cursor 走智谱，所有工具的请求都过这一个网关。顺便还有个意外的好处——用量能统一看了：

```bash
tokrouter show usage --month
```

各个工具花了多少 Token 一清二楚，之前 Key 散落在各个工具里根本算不明白。

GitHub: [github.com/tokzone/tokrouter](https://github.com/tokzone/tokrouter)，MIT 协议，一个二进制没依赖。
