import { get, patch, post } from './request'
import type {
  OpsRelease,
  OpsReleaseCreate,
  OpsReleaseListResult,
  OpsReleasePatch,
} from './types'

export function fetchOpsReleases(params?: {
  product?: string
  channel?: string
  platform?: string
  arch?: string
  status?: string
  version?: string
  limit?: number
  offset?: number
}): Promise<OpsReleaseListResult> {
  return get<OpsReleaseListResult>('/api/v1/ops/releases', { params })
}

export function fetchOpsRelease(id: string): Promise<OpsRelease> {
  return get<OpsRelease>(`/api/v1/ops/releases/${encodeURIComponent(id)}`)
}

export function createOpsRelease(body: OpsReleaseCreate): Promise<OpsRelease> {
  return post<OpsRelease>('/api/v1/ops/releases', body)
}

export function patchOpsRelease(id: string, body: OpsReleasePatch): Promise<OpsRelease> {
  return patch<OpsRelease>(`/api/v1/ops/releases/${encodeURIComponent(id)}`, body)
}

export function publishOpsRelease(id: string): Promise<OpsRelease> {
  return post<OpsRelease>(`/api/v1/ops/releases/${encodeURIComponent(id)}/publish`, {})
}

export function yankOpsRelease(id: string): Promise<OpsRelease> {
  return post<OpsRelease>(`/api/v1/ops/releases/${encodeURIComponent(id)}/yank`, {})
}
