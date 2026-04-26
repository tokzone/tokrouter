package config

import (
	"fmt"
	"sort"
)

// Provider name constants for type-safe preset references.
const (
	// International providers
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGoogle    = "google"
	ProviderMistral   = "mistral"
	ProviderCohere    = "cohere"
	ProviderGroq      = "groq"
	ProviderDeepSeek  = "deepseek"

	// Chinese providers
	ProviderZhipu       = "zhipu"
	ProviderQwen        = "qwen"
	ProviderTencent     = "tencent"
	ProviderBaidu       = "baidu"
	ProviderQianfan     = "qianfan"
	ProviderHuawei      = "huawei"
	ProviderMoonshot    = "moonshot"
	ProviderMinimax     = "minimax"
	ProviderSiliconflow = "siliconflow"
	ProviderYi          = "yi"
	ProviderStepfun     = "stepfun"
	ProviderBaichuan    = "baichuan"
	ProviderXunfei      = "xunfei"
	ProviderDoubao      = "doubao"
	ProviderParallel    = "parallel"

	// Aggregation platforms
	ProviderTogether    = "together"
	ProviderReplicate   = "replicate"
	ProviderOpenRouter  = "openrouter"
)

// ProviderPreset defines a preset configuration for a LLM provider.
type ProviderPreset struct {
	Name          string       // Preset identifier (e.g., "openai", "deepseek", "qwen")
	DisplayName   string       // Display name (e.g., "OpenAI", "DeepSeek", "阿里通义千问")
	BaseURL       string       // API base URL
	Format        string       // Protocol format: openai, anthropic, gemini, cohere
	DefaultModels []ModelInfo  // Default model list
	DocURL        string       // Documentation / API Key URL
	Region        string       // Service region: "global" or "china"
}

// ModelInfo defines a model with optional alias and context length.
type ModelInfo struct {
	Name    string // Model API name
	Alias   string // Optional alias (e.g., "4o" -> "gpt-4o")
	Context int    // Optional context length (e.g., 128000)
}

// BuiltinPresets contains all built-in provider presets.
var BuiltinPresets = map[string]ProviderPreset{
	// ===== International Providers =====
	ProviderOpenAI: {
		Name:        ProviderOpenAI,
		DisplayName: "OpenAI",
		BaseURL:     "https://api.openai.com/v1",
		Format:      FormatOpenAI,
		Region:      "global",
		DocURL:      "https://platform.openai.com/api-keys",
		DefaultModels: []ModelInfo{
			{Name: "gpt-4o", Alias: "4o", Context: 128000},
			{Name: "gpt-4o-mini", Alias: "4o-mini", Context: 128000},
			{Name: "gpt-4-turbo", Alias: "4t", Context: 128000},
			{Name: "gpt-4", Context: 8192},
			{Name: "gpt-3.5-turbo", Alias: "35t", Context: 16385},
			{Name: "o1", Context: 200000},
			{Name: "o1-mini", Context: 200000},
			{Name: "o3-mini", Context: 200000},
		},
	},
	ProviderAnthropic: {
		Name:        ProviderAnthropic,
		DisplayName: "Anthropic",
		BaseURL:     "https://api.anthropic.com",
		Format:      FormatAnthropic,
		Region:      "global",
		DocURL:      "https://console.anthropic.com/settings/keys",
		DefaultModels: []ModelInfo{
			{Name: "claude-3-5-sonnet-20241022", Alias: "claude-3.5-sonnet", Context: 200000},
			{Name: "claude-3-5-haiku-20241022", Alias: "claude-3.5-haiku", Context: 200000},
			{Name: "claude-3-opus-20240229", Alias: "claude-3-opus", Context: 200000},
			{Name: "claude-3-sonnet-20240229", Alias: "claude-3-sonnet", Context: 200000},
			{Name: "claude-3-haiku-20240307", Alias: "claude-3-haiku", Context: 200000},
		},
	},
	ProviderGoogle: {
		Name:        ProviderGoogle,
		DisplayName: "Google Gemini",
		BaseURL:     "https://generativelanguage.googleapis.com",
		Format:      FormatGemini,
		Region:      "global",
		DocURL:      "https://aistudio.google.com/apikey",
		DefaultModels: []ModelInfo{
			{Name: "gemini-2.0-flash", Alias: "gemini-2-flash", Context: 1048576},
			{Name: "gemini-2.0-flash-lite", Alias: "gemini-2-flash-lite", Context: 1048576},
			{Name: "gemini-1.5-pro", Alias: "gemini-1.5-pro", Context: 2097152},
			{Name: "gemini-1.5-flash", Alias: "gemini-1.5-flash", Context: 1048576},
			{Name: "gemini-1.5-flash-8b", Alias: "gemini-1.5-flash-8b", Context: 1048576},
		},
	},
	ProviderMistral: {
		Name:        ProviderMistral,
		DisplayName: "Mistral AI",
		BaseURL:     "https://api.mistral.ai/v1",
		Format:      FormatOpenAI,
		Region:      "global",
		DocURL:      "https://console.mistral.ai/api-keys/",
		DefaultModels: []ModelInfo{
			{Name: "mistral-large-latest", Alias: "mistral-large", Context: 128000},
			{Name: "mistral-medium-latest", Alias: "mistral-medium", Context: 128000},
			{Name: "mistral-small-latest", Alias: "mistral-small", Context: 128000},
			{Name: "codestral-latest", Alias: "codestral", Context: 64000},
			{Name: "ministral-8b-latest", Alias: "ministral-8b", Context: 128000},
		},
	},
	ProviderCohere: {
		Name:        ProviderCohere,
		DisplayName: "Cohere",
		BaseURL:     "https://api.cohere.ai/v1",
		Format:      FormatCohere,
		Region:      "global",
		DocURL:      "https://dashboard.cohere.com/api-keys",
		DefaultModels: []ModelInfo{
			{Name: "command-r-plus", Alias: "r-plus", Context: 128000},
			{Name: "command-r", Alias: "r", Context: 128000},
			{Name: "command-light", Alias: "light", Context: 4096},
			{Name: "command", Context: 4096},
		},
	},
	ProviderGroq: {
		Name:        ProviderGroq,
		DisplayName: "Groq",
		BaseURL:     "https://api.groq.com/openai/v1",
		Format:      FormatOpenAI,
		Region:      "global",
		DocURL:      "https://console.groq.com/keys",
		DefaultModels: []ModelInfo{
			{Name: "llama-3.3-70b-versatile", Alias: "llama-70b", Context: 128000},
			{Name: "llama-3.1-8b-instant", Alias: "llama-8b", Context: 128000},
			{Name: "mixtral-8x7b-32768", Alias: "mixtral", Context: 32768},
			{Name: "gemma2-9b-it", Alias: "gemma2", Context: 8192},
		},
	},
	ProviderDeepSeek: {
		Name:        ProviderDeepSeek,
		DisplayName: "DeepSeek",
		BaseURL:     "https://api.deepseek.com",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://platform.deepseek.com/api-keys",
		DefaultModels: []ModelInfo{
			{Name: "deepseek-chat", Alias: "chat", Context: 64000},
			{Name: "deepseek-coder", Alias: "coder", Context: 64000},
			{Name: "deepseek-reasoner", Alias: "r1", Context: 64000},
		},
	},

	// ===== Chinese Providers =====
	ProviderZhipu: {
		Name:        ProviderZhipu,
		DisplayName: "智谱 AI",
		BaseURL:     "https://open.bigmodel.cn/api/paas/v4",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://open.bigmodel.cn/api-keys",
		DefaultModels: []ModelInfo{
			{Name: "glm-4-plus", Alias: "glm-plus", Context: 128000},
			{Name: "glm-4-0520", Alias: "glm-4", Context: 128000},
			{Name: "glm-4-air", Alias: "glm-air", Context: 128000},
			{Name: "glm-4-airx", Alias: "glm-airx", Context: 128000},
			{Name: "glm-4-flash", Alias: "glm-flash", Context: 128000},
			{Name: "glm-4-long", Alias: "glm-long", Context: 1048576},
			{Name: "glm-3-turbo", Alias: "glm-3", Context: 4096},
		},
	},
	ProviderQwen: {
		Name:        ProviderQwen,
		DisplayName: "阿里通义千问",
		BaseURL:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://help.aliyun.com/document_detail/2712195.html",
		DefaultModels: []ModelInfo{
			{Name: "qwen-max", Alias: "max", Context: 32768},
			{Name: "qwen-max-latest", Alias: "max-latest", Context: 32768},
			{Name: "qwen-plus", Alias: "plus", Context: 128000},
			{Name: "qwen-plus-latest", Alias: "plus-latest", Context: 128000},
			{Name: "qwen-turbo", Alias: "turbo", Context: 128000},
			{Name: "qwen-turbo-latest", Alias: "turbo-latest", Context: 128000},
			{Name: "qwen-long", Alias: "long", Context: 10000000},
			{Name: "qwen-coder-plus", Alias: "coder-plus", Context: 128000},
			{Name: "qwen-coder-turbo", Alias: "coder-turbo", Context: 128000},
			{Name: "qwen-vl-max", Alias: "vl-max", Context: 32768},
			{Name: "qwen-vl-plus", Alias: "vl-plus", Context: 8192},
			{Name: "qwen-audio-turbo", Alias: "audio", Context: 8192},
		},
	},
	ProviderTencent: {
		Name:        ProviderTencent,
		DisplayName: "腾讯混元",
		BaseURL:     "https://api.hunyuan.cloud.tencent.com/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://cloud.tencent.com/document/product/1729",
		DefaultModels: []ModelInfo{
			{Name: "hunyuan-lite", Alias: "lite", Context: 256000},
			{Name: "hunyuan-standard", Alias: "standard", Context: 32000},
			{Name: "hunyuan-standard-256k", Alias: "standard-256k", Context: 256000},
			{Name: "hunyuan-pro", Alias: "pro", Context: 32000},
			{Name: "hunyuan-turbo", Alias: "turbo", Context: 32000},
			{Name: "hunyuan-turbo-latest", Alias: "turbo-latest", Context: 32000},
			{Name: "hunyuan-vision", Alias: "vision", Context: 8192},
			{Name: "hunyuan-coding", Alias: "coding", Context: 32000},
		},
	},
	ProviderBaidu: {
		Name:        ProviderBaidu,
		DisplayName: "百度文心",
		BaseURL:     "https://aigp.baidu.com/rpc/1.0/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://cloud.baidu.com/doc/WENXINWORKSHOP/index.html",
		DefaultModels: []ModelInfo{
			{Name: "ernie-4.0-8k", Alias: "ernie-4", Context: 8192},
			{Name: "ernie-4.0-turbo-8k", Alias: "ernie-4-turbo", Context: 8192},
			{Name: "ernie-3.5-8k", Alias: "ernie-3.5", Context: 8192},
			{Name: "ernie-3.5-turbo-8k", Alias: "ernie-3.5-turbo", Context: 8192},
			{Name: "ernie-speed-8k", Alias: "speed", Context: 8192},
			{Name: "ernie-lite-8k", Alias: "lite", Context: 8192},
			{Name: "ernie-tiny-8k", Alias: "tiny", Context: 8192},
		},
	},
	ProviderQianfan: {
		Name:        ProviderQianfan,
		DisplayName: "百度千帆",
		BaseURL:     "https://qianfan.baidubce.com/v2",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://cloud.baidu.com/doc/WENXINWORKSHOP/s/fl3o9l4bc",
		DefaultModels: []ModelInfo{
			{Name: "yi-34b-chat", Alias: "yi", Context: 4096},
			{Name: "llama-2-70b-chat", Alias: "llama-70b", Context: 4096},
			{Name: "qwen-72b-chat", Alias: "qwen-72b", Context: 4096},
			{Name: "chatglm3-6b", Alias: "glm-6b", Context: 4096},
			{Name: "baichuan2-13b-chat", Alias: "baichuan", Context: 4096},
		},
	},
	ProviderHuawei: {
		Name:        ProviderHuawei,
		DisplayName: "华为盘古",
		BaseURL:     "https://pangu-api.huawei.com/api/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://www.huawei.com/cn/products/cloud/pangu-models",
		DefaultModels: []ModelInfo{
			{Name: "pangu-nlp-5b", Alias: "pangu-5b", Context: 4096},
			{Name: "pangu-nlp-10b", Alias: "pangu-10b", Context: 4096},
			{Name: "pangu-sigma", Alias: "sigma", Context: 4096},
		},
	},
	ProviderMoonshot: {
		Name:        ProviderMoonshot,
		DisplayName: "月之暗面 Kimi",
		BaseURL:     "https://api.moonshot.cn/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://platform.moonshot.cn/console/api-keys",
		DefaultModels: []ModelInfo{
			{Name: "moonshot-v1-8k", Alias: "kimi-8k", Context: 8192},
			{Name: "moonshot-v1-32k", Alias: "kimi-32k", Context: 32768},
			{Name: "moonshot-v1-128k", Alias: "kimi-128k", Context: 131072},
		},
	},
	ProviderMinimax: {
		Name:        ProviderMinimax,
		DisplayName: "稀宇 MiniMax",
		BaseURL:     "https://api.minimax.chat/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://www.minimaxi.com/document/api-keys",
		DefaultModels: []ModelInfo{
			{Name: "abab6.5s-chat", Alias: "6.5s", Context: 24576},
			{Name: "abab6.5g-chat", Alias: "6.5g", Context: 24576},
			{Name: "abab6.5-chat", Alias: "6.5", Context: 24576},
			{Name: "abab5.5-chat", Alias: "5.5", Context: 16384},
			{Name: "abab5.5s-chat", Alias: "5.5s", Context: 16384},
		},
	},
	ProviderSiliconflow: {
		Name:        ProviderSiliconflow,
		DisplayName: "硅基流动",
		BaseURL:     "https://api.siliconflow.cn/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://siliconflow.cn/api-keys",
		DefaultModels: []ModelInfo{
			{Name: "Qwen/Qwen2.5-72B-Instruct", Alias: "qwen-72b", Context: 32768},
			{Name: "Qwen/Qwen2.5-32B-Instruct", Alias: "qwen-32b", Context: 32768},
			{Name: "deepseek-ai/DeepSeek-V3", Alias: "deepseek-v3", Context: 64000},
			{Name: "deepseek-ai/DeepSeek-R1", Alias: "deepseek-r1", Context: 64000},
			{Name: "THUDM/glm-4-9b-chat", Alias: "glm-4-9b", Context: 8192},
			{Name: "meta-llama/Llama-3.3-70B-Instruct", Alias: "llama-70b", Context: 128000},
		},
	},
	ProviderYi: {
		Name:        ProviderYi,
		DisplayName: "零一万物",
		BaseURL:     "https://api.lingyiwanwu.com/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://platform.lingyiwanwu.com/apikeys",
		DefaultModels: []ModelInfo{
			{Name: "yi-large", Alias: "large", Context: 32768},
			{Name: "yi-large-turbo", Alias: "large-turbo", Context: 32768},
			{Name: "yi-medium", Alias: "medium", Context: 16384},
			{Name: "yi-medium-200k", Alias: "medium-200k", Context: 200000},
			{Name: "yi-spark", Alias: "spark", Context: 16384},
			{Name: "yi-vision", Alias: "vision", Context: 16384},
		},
	},
	ProviderStepfun: {
		Name:        ProviderStepfun,
		DisplayName: "阶跃星辰",
		BaseURL:     "https://api.stepfun.com/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://platform.stepfun.com/docs/api-keys/create-api-key",
		DefaultModels: []ModelInfo{
			{Name: "step-1-8k", Alias: "step-8k", Context: 8192},
			{Name: "step-1-32k", Alias: "step-32k", Context: 32768},
			{Name: "step-1-128k", Alias: "step-128k", Context: 131072},
			{Name: "step-1-256k", Alias: "step-256k", Context: 262144},
			{Name: "step-2-8k", Alias: "step-2", Context: 8192},
			{Name: "step-vl-1-8k", Alias: "step-vision", Context: 8192},
		},
	},
	ProviderBaichuan: {
		Name:        ProviderBaichuan,
		DisplayName: "百川智能",
		BaseURL:     "https://api.baichuan-ai.com/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://platform.baichuan-ai.com/console/api-key",
		DefaultModels: []ModelInfo{
			{Name: "Baichuan4", Alias: "bc4", Context: 128000},
			{Name: "Baichuan3-Turbo", Alias: "bc3-turbo", Context: 128000},
			{Name: "Baichuan3-Turbo-128k", Alias: "bc3-128k", Context: 128000},
			{Name: "Baichuan2-Turbo", Alias: "bc2-turbo", Context: 4096},
		},
	},
	ProviderXunfei: {
		Name:        ProviderXunfei,
		DisplayName: "讯飞星火",
		BaseURL:     "https://spark-api-open.xf-yun.com/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://xinghuo.xfyun.cn/sparkapi",
		DefaultModels: []ModelInfo{
			{Name: "generalv3.5", Alias: "spark-3.5", Context: 8192},
			{Name: "generalv3", Alias: "spark-3", Context: 8192},
			{Name: "generalv2", Alias: "spark-2", Context: 4096},
			{Name: "generalv1.5", Alias: "spark-1.5", Context: 4096},
			{Name: "generalv1", Alias: "spark-1", Context: 4096},
		},
	},
	ProviderDoubao: {
		Name:        ProviderDoubao,
		DisplayName: "字节豆包",
		BaseURL:     "https://ark.cn-beijing.volces.com/api/v3",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://www.volcengine.com/docs/82379/1299815",
		DefaultModels: []ModelInfo{
			{Name: "doubao-pro-32k", Alias: "pro-32k", Context: 32768},
			{Name: "doubao-pro-128k", Alias: "pro-128k", Context: 131072},
			{Name: "doubao-lite-32k", Alias: "lite-32k", Context: 32768},
			{Name: "doubao-lite-128k", Alias: "lite-128k", Context: 131072},
			{Name: "doubao-1.5-pro-32k", Alias: "1.5-pro", Context: 32768},
			{Name: "doubao-1.5-pro-128k", Alias: "1.5-pro-128k", Context: 131072},
		},
	},
	ProviderParallel: {
		Name:        ProviderParallel,
		DisplayName: "并行科技",
		BaseURL:     "https://api.paralos.com/v1",
		Format:      FormatOpenAI,
		Region:      "china",
		DocURL:      "https://www.parallelchain.cn",
		DefaultModels: []ModelInfo{
			{Name: "paralos-1", Alias: "p1", Context: 8192},
			{Name: "paralos-2", Alias: "p2", Context: 8192},
			{Name: "paralos-pro", Alias: "pro", Context: 32768},
		},
	},

	// ===== Aggregation Platforms =====
	ProviderTogether: {
		Name:        ProviderTogether,
		DisplayName: "Together AI",
		BaseURL:     "https://api.together.xyz/v1",
		Format:      FormatOpenAI,
		Region:      "global",
		DocURL:      "https://api.together.xyz/settings/api-keys",
		DefaultModels: []ModelInfo{
			{Name: "meta-llama/Llama-3.3-70B-Instruct-Turbo", Alias: "llama-70b", Context: 128000},
			{Name: "meta-llama/Llama-3.2-11B-Vision-Instruct", Alias: "llama-11b-vision", Context: 128000},
			{Name: "Qwen/Qwen2.5-72B-Instruct", Alias: "qwen-72b", Context: 32768},
			{Name: "mistralai/Mixtral-8x7B-Instruct-v0.1", Alias: "mixtral", Context: 32768},
		},
	},
	ProviderReplicate: {
		Name:        ProviderReplicate,
		DisplayName: "Replicate",
		BaseURL:     "https://api.replicate.com/v1",
		Format:      FormatOpenAI,
		Region:      "global",
		DocURL:      "https://replicate.com/account/api-tokens",
		DefaultModels: []ModelInfo{
			{Name: "meta/llama-2-70b-chat", Alias: "llama-2-70b", Context: 4096},
			{Name: "mistralai/mixtral-8x7b-instruct-v0.1", Alias: "mixtral", Context: 32768},
		},
	},
	ProviderOpenRouter: {
		Name:        ProviderOpenRouter,
		DisplayName: "OpenRouter",
		BaseURL:     "https://openrouter.ai/api/v1",
		Format:      FormatOpenAI,
		Region:      "global",
		DocURL:      "https://openrouter.ai/keys",
		DefaultModels: []ModelInfo{
			{Name: "openai/gpt-4o", Alias: "gpt-4o", Context: 128000},
			{Name: "anthropic/claude-3.5-sonnet", Alias: "claude-3.5-sonnet", Context: 200000},
			{Name: "google/gemini-pro-1.5", Alias: "gemini-1.5-pro", Context: 1048576},
			{Name: "meta-llama/llama-3.1-70b-instruct", Alias: "llama-70b", Context: 128000},
		},
	},
}

// GetPreset returns a preset by name, or an error if not found.
func GetPreset(name string) (ProviderPreset, error) {
	preset, ok := BuiltinPresets[name]
	if !ok {
		return ProviderPreset{}, fmt.Errorf("unknown provider preset: %q", name)
	}
	return preset, nil
}

// ListPresets returns all built-in presets sorted by region then name.
func ListPresets() []ProviderPreset {
	presets := make([]ProviderPreset, 0, len(BuiltinPresets))
	for _, p := range BuiltinPresets {
		presets = append(presets, p)
	}
	// Sort by region (global first) then by name
	sort.Slice(presets, func(i, j int) bool {
		if presets[i].Region != presets[j].Region {
			return presets[i].Region < presets[j].Region // "global" < "china"
		}
		return presets[i].Name < presets[j].Name
	})
	return presets
}

// PresetExists checks if a preset name exists.
func PresetExists(name string) bool {
	return BuiltinPresets[name].Name != ""
}