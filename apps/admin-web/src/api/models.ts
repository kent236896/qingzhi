import { http } from '@/api/client'

export type ModelType = 'chat' | 'embedding' | 'rerank' | 'vllm' | 'asr'
export type BindingRole = 'embedding' | 'summary' | 'wiki_synthesis' | 'vlm' | 'asr' | 'rerank'

export interface Credentials {
  api_key: { configured: boolean }
  app_secret: { configured: boolean }
}

export interface AiModel {
  id: string
  name: string
  display_name: string
  type: ModelType
  source: string
  description: string
  parameters: Record<string, unknown>
  is_builtin: boolean
  status: string
  credentials: Credentials
  capabilities?: Record<string, unknown>
  bound_roles: string[]
  created_at: string
  updated_at: string
}

export interface ExtraOption {
  label: string
  value: string
}

export interface ExtraField {
  key: string
  label: string
  type: string
  options?: ExtraOption[]
  required: boolean
  placeholder?: string
  model_types: string[]
  secret: boolean
}

export interface CatalogModel {
  id: string
  name: string
  type: ModelType
  api?: string
  reasoning: boolean
  context_window?: number
  max_output_tokens?: number
  dimension?: number
  thinking_levels?: string[]
}

export interface Provider {
  value: string
  label: string
  description: string
  icon?: string
  api?: string
  auth?: string
  requires_auth: boolean
  default_urls: Record<string, string>
  model_types: string[]
  extra_fields: ExtraField[]
  credential_labels: Record<string, unknown>[]
  models: CatalogModel[]
  thinking?: Record<string, unknown>
  order: number
}

export interface UsageResource {
  id: string
  name: string
  bindings: string[]
}

export interface ModelUsage {
  knowledge_bases: UsageResource[]
  agents: UsageResource[]
  long_term_memory: { bindings: string[] }
  knowledge_base_total: number
  agent_total: number
  local_roles?: string[]
}

export interface Binding {
  role: BindingRole
  model_id: string
  model_name: string
  display_name: string
  updated_by: string
  updated_at: string
  warning?: string
}

export interface TestResult {
  available: boolean
  message: string
  dimension?: number
  elapsed_ms: number
}

export interface OllamaStatus {
  available: boolean
  version: string
  base_url: string
  error?: string
}

export interface OllamaModel {
  name: string
  size: number
  modified_at: string
}

export interface OllamaTask {
  id: string
  model_name: string
  status: string
  progress: number
  message: string
}

const providerCache = new Map<string, Provider[]>()

export function listProviders(type: ModelType, icons: boolean) {
  const key = `${type}:${icons ? 'icon' : 'plain'}`
  const cached = providerCache.get(key)
  if (cached) {
    return Promise.resolve(cached)
  }
  return http
    .get<Provider[]>('/api/v1/weknora/model-providers', { params: { type, icons: icons ? '1' : '0' }, skipErrorToast: true })
    .then((response) => {
      const items = response.data ?? []
      providerCache.set(key, items)
      return items
    })
}

export function resolveProvider(params: { provider: string; model: string; base_url: string; type: ModelType }) {
  return http.get<Record<string, unknown>>('/api/v1/weknora/model-providers/resolve', { params, skipErrorToast: true, timeout: 30000 })
}

export function listModels(params: { type?: string; keyword?: string } = {}) {
  return http.get<AiModel[]>('/api/v1/weknora/models', { params })
}

export function createModel(body: Record<string, unknown>) {
  return http.post<AiModel>('/api/v1/weknora/models', body, { skipErrorToast: true, timeout: 60000 })
}

export function updateModel(id: string, body: Record<string, unknown>) {
  return http.put<AiModel>(`/api/v1/weknora/models/${id}`, body, { skipErrorToast: true, timeout: 60000 })
}

export function deleteModel(id: string) {
  return http.delete<null>(`/api/v1/weknora/models/${id}`, { skipErrorToast: true })
}

export function copyModel(id: string, displayName: string) {
  return http.post<AiModel>(`/api/v1/weknora/models/${id}/copy`, { display_name: displayName }, { skipErrorToast: true })
}

export function clearCredential(id: string, field: 'api_key' | 'app_secret') {
  return http.delete<null>(`/api/v1/weknora/models/${id}/credentials/${field}`, { skipErrorToast: true })
}

export function testModel(body: Record<string, unknown>) {
  return http.post<TestResult>('/api/v1/weknora/models/test', body, { skipErrorToast: true, timeout: 120000 })
}

export function debugModel(id: string, form: FormData) {
  return http.post<Record<string, unknown>>(`/api/v1/weknora/models/${id}/debug`, form, { skipErrorToast: true, timeout: 120000 })
}

export function listBindings() {
  return http.get<Binding[]>('/api/v1/weknora/model-bindings')
}

export function saveBinding(role: BindingRole, modelId: string) {
  return http.put<Binding>(`/api/v1/weknora/model-bindings/${role}`, { model_id: modelId }, { skipErrorToast: true })
}

export function ollamaStatus() {
  return http.get<OllamaStatus>('/api/v1/weknora/ollama/status', { skipErrorToast: true })
}

export function ollamaModels() {
  return http.get<OllamaModel[]>('/api/v1/weknora/ollama/models', { skipErrorToast: true })
}

export function downloadOllama(name: string) {
  return http.post<OllamaTask>('/api/v1/weknora/ollama/models/download', { name }, { skipErrorToast: true })
}

export function ollamaProgress(taskId: string) {
  return http.get<OllamaTask>(`/api/v1/weknora/ollama/download/progress/${taskId}`, { skipErrorToast: true })
}
