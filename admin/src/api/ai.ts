import { del, get, patch, post } from './request'
import type {
  OpsAIEndpoint,
  OpsAIEndpointTest,
  OpsAIEndpointTestResult,
  OpsAIEndpointWrite,
  OpsAIModel,
  OpsAIPullModelsResult,
  OpsAIUsageResult,
} from './types'

export function fetchOpsAIEndpoints(): Promise<{ items: OpsAIEndpoint[] }> {
  return get<{ items: OpsAIEndpoint[] }>('/api/v1/ops/ai/endpoints')
}

export function createOpsAIEndpoint(body: OpsAIEndpointWrite): Promise<OpsAIEndpoint> {
  return post<OpsAIEndpoint>('/api/v1/ops/ai/endpoints', body)
}

export function fetchOpsAIEndpoint(id: string): Promise<OpsAIEndpoint> {
  return get<OpsAIEndpoint>(`/api/v1/ops/ai/endpoints/${encodeURIComponent(id)}`)
}

export function fetchOpsAIEndpointSecret(id: string): Promise<{ found: boolean }> {
  return get<{ found: boolean }>(`/api/v1/ops/ai/endpoints/${encodeURIComponent(id)}/secret`)
}

export function patchOpsAIEndpoint(id: string, body: OpsAIEndpointWrite): Promise<OpsAIEndpoint> {
  return patch<OpsAIEndpoint>(`/api/v1/ops/ai/endpoints/${encodeURIComponent(id)}`, body)
}

export function deleteOpsAIEndpoint(id: string): Promise<{ deleted: boolean }> {
  return del<{ deleted: boolean }>(`/api/v1/ops/ai/endpoints/${encodeURIComponent(id)}`)
}

export function testOpsAIEndpoint(body: OpsAIEndpointTest): Promise<OpsAIEndpointTestResult> {
  return post<OpsAIEndpointTestResult>('/api/v1/ops/ai/endpoints/test', body)
}

export function pullOpsAIModels(body: OpsAIEndpointTest): Promise<OpsAIPullModelsResult> {
  return post<OpsAIPullModelsResult>('/api/v1/ops/ai/endpoints/pull-models', body)
}

export function fetchOpsAIModels(endpointId: string): Promise<{ items: OpsAIModel[] }> {
  return get<{ items: OpsAIModel[] }>(
    `/api/v1/ops/ai/endpoints/${encodeURIComponent(endpointId)}/models`,
  )
}

export function createOpsAIModel(
  endpointId: string,
  body: { code: string; label?: string; recordStatus?: string; sortSeq?: number },
): Promise<OpsAIModel> {
  return post<OpsAIModel>(`/api/v1/ops/ai/endpoints/${encodeURIComponent(endpointId)}/models`, body)
}

export function patchOpsAIModel(
  modelId: string,
  body: { code?: string; label?: string; recordStatus?: string; sortSeq?: number },
): Promise<OpsAIModel> {
  return patch<OpsAIModel>(`/api/v1/ops/ai/models/${encodeURIComponent(modelId)}`, body)
}

export function deleteOpsAIModel(modelId: string): Promise<{ deleted: boolean }> {
  return del<{ deleted: boolean }>(`/api/v1/ops/ai/models/${encodeURIComponent(modelId)}`)
}

export function fetchOpsAIUsage(params?: {
  day?: string
  email?: string
  userId?: string
  limit?: number
  offset?: number
}): Promise<OpsAIUsageResult> {
  return get<OpsAIUsageResult>('/api/v1/ops/ai/usage', { params })
}
