/**
 * 云端系统模型接入预设（OpenAI 兼容 /chat/completions）。
 * 选择后填充展示名、Base URL、默认模型与推荐目录；密钥仍由运营填写。
 */

export type AIProviderPreset = {
  id: string
  label: string
  defaultName: string
  baseUrl: string
  defaultModel: string
  models: readonly string[]
  hint?: string
}

export const AI_PROVIDER_PRESET_CUSTOM = 'custom'

export const AI_PROVIDER_PRESETS: readonly AIProviderPreset[] = [
  {
    id: 'openai',
    label: 'OpenAI',
    defaultName: 'OpenAI',
    baseUrl: 'https://api.openai.com/v1',
    defaultModel: 'gpt-4o',
    models: ['gpt-4o', 'gpt-4o-mini', 'gpt-4.1', 'gpt-4.1-mini', 'o3-mini', 'o4-mini'],
  },
  {
    id: 'deepseek',
    label: 'DeepSeek',
    defaultName: 'DeepSeek',
    baseUrl: 'https://api.deepseek.com/v1',
    defaultModel: 'deepseek-chat',
    models: ['deepseek-chat', 'deepseek-reasoner'],
  },
  {
    id: 'moonshot',
    label: 'Moonshot / Kimi',
    defaultName: 'Moonshot',
    baseUrl: 'https://api.moonshot.cn/v1',
    defaultModel: 'kimi-k2.5',
    models: ['kimi-k2.5', 'kimi-k2-turbo-preview', 'moonshot-v1-128k', 'moonshot-v1-32k'],
  },
  {
    id: 'qwen',
    label: '通义千问 (DashScope)',
    defaultName: '通义千问',
    baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    defaultModel: 'qwen-plus',
    models: ['qwen-plus', 'qwen-turbo', 'qwen-max', 'qwen-long', 'qwq-plus'],
  },
  {
    id: 'zhipu',
    label: '智谱 GLM',
    defaultName: '智谱 GLM',
    baseUrl: 'https://open.bigmodel.cn/api/paas/v4/chat/completions',
    defaultModel: 'glm-4-plus',
    models: ['glm-4-plus', 'glm-4-flash', 'glm-4-air', 'glm-z1-flash'],
  },
  {
    id: 'volcengine',
    label: '火山方舟 / 豆包',
    defaultName: '火山方舟',
    baseUrl: 'https://ark.cn-beijing.volces.com/api/v3/chat/completions',
    defaultModel: '',
    models: [],
    hint: '模型编码填方舟推理接入点 ID，不是展示名。',
  },
  {
    id: 'siliconflow',
    label: 'SiliconFlow 硅基流动',
    defaultName: 'SiliconFlow',
    baseUrl: 'https://api.siliconflow.cn/v1',
    defaultModel: 'deepseek-ai/DeepSeek-V3',
    models: [
      'deepseek-ai/DeepSeek-V3',
      'deepseek-ai/DeepSeek-R1',
      'Qwen/Qwen2.5-72B-Instruct',
      'THUDM/GLM-4-9B-0414',
    ],
  },
  {
    id: 'openrouter',
    label: 'OpenRouter',
    defaultName: 'OpenRouter',
    baseUrl: 'https://openrouter.ai/api/v1',
    defaultModel: 'openai/gpt-4o',
    models: [
      'openai/gpt-4o',
      'openai/gpt-4o-mini',
      'anthropic/claude-sonnet-4',
      'google/gemini-2.5-pro',
      'deepseek/deepseek-chat',
    ],
  },
  {
    id: 'groq',
    label: 'Groq',
    defaultName: 'Groq',
    baseUrl: 'https://api.groq.com/openai/v1',
    defaultModel: 'llama-3.3-70b-versatile',
    models: ['llama-3.3-70b-versatile', 'llama-3.1-8b-instant', 'qwen/qwen3-32b'],
  },
  {
    id: 'gemini',
    label: 'Google Gemini (OpenAI 兼容)',
    defaultName: 'Google Gemini',
    baseUrl: 'https://generativelanguage.googleapis.com/v1beta/openai/chat/completions',
    defaultModel: 'gemini-2.5-flash',
    models: ['gemini-2.5-flash', 'gemini-2.5-pro', 'gemini-2.0-flash'],
  },
  {
    id: 'minimax',
    label: 'MiniMax',
    defaultName: 'MiniMax',
    baseUrl: 'https://api.minimaxi.com/v1',
    defaultModel: 'MiniMax-Text-01',
    models: ['MiniMax-Text-01', 'abab6.5s-chat'],
  },
  {
    id: 'stepfun',
    label: '阶跃星辰 StepFun',
    defaultName: '阶跃星辰',
    baseUrl: 'https://api.stepfun.com/v1',
    defaultModel: 'step-2-mini',
    models: ['step-2-mini', 'step-1-8k', 'step-1-32k', 'step-1-128k'],
  },
  {
    id: 'yi',
    label: '零一万物 Yi',
    defaultName: '零一万物',
    baseUrl: 'https://api.lingyiwanwu.com/v1',
    defaultModel: 'yi-lightning',
    models: ['yi-lightning', 'yi-large', 'yi-medium', 'yi-spark'],
  },
  {
    id: 'baichuan',
    label: '百川 Baichuan',
    defaultName: '百川',
    baseUrl: 'https://api.baichuan-ai.com/v1',
    defaultModel: 'Baichuan4-Turbo',
    models: ['Baichuan4-Turbo', 'Baichuan4', 'Baichuan3-Turbo'],
  },
  {
    id: AI_PROVIDER_PRESET_CUSTOM,
    label: '自定义 / OpenAI 兼容',
    defaultName: 'NiuMa',
    baseUrl: '',
    defaultModel: '',
    models: [],
    hint: '填写上游 /v1 或完整 /chat/completions 地址。',
  },
]

export function findAIProviderPreset(id: string): AIProviderPreset | undefined {
  return AI_PROVIDER_PRESETS.find((p) => p.id === id)
}

/** 按已保存的 Base URL / 展示名回显厂商。 */
export function matchAIProviderPreset(input: { baseUrl?: string; providerName?: string }): string {
  const url = (input.baseUrl ?? '').trim().replace(/\/+$/, '')
  const name = (input.providerName ?? '').trim().toLowerCase()

  if (url) {
    const normalize = (u: string) => {
      let out = u
      while (out.endsWith('/')) {
        out = out.slice(0, -1)
      }
      const suffix = '/chat/completions'
      if (out.toLowerCase().endsWith(suffix)) {
        out = out.slice(0, -suffix.length)
      }
      return out
    }
    const normalized = normalize(url)
    const byUrl = AI_PROVIDER_PRESETS.find((p) => {
      if (!p.baseUrl || p.id === AI_PROVIDER_PRESET_CUSTOM) return false
      const presetUrl = normalize(p.baseUrl)
      return (
        normalized === presetUrl ||
        normalized.startsWith(presetUrl) ||
        presetUrl.startsWith(normalized)
      )
    })
    if (byUrl) return byUrl.id
  }

  if (name) {
    const byName = AI_PROVIDER_PRESETS.find(
      (p) =>
        p.id !== AI_PROVIDER_PRESET_CUSTOM &&
        (name.includes(p.id) || p.defaultName.toLowerCase() === name),
    )
    if (byName) return byName.id
  }

  return AI_PROVIDER_PRESET_CUSTOM
}
