import { get } from './request'
import type { OpsUserDetail, OpsUserListResult } from './types'

export function fetchOpsUsers(params?: {
  q?: string
  limit?: number
  offset?: number
}): Promise<OpsUserListResult> {
  return get<OpsUserListResult>('/api/v1/ops/users', { params })
}

export function fetchOpsUserDetail(id: string): Promise<OpsUserDetail> {
  return get<OpsUserDetail>(`/api/v1/ops/users/${encodeURIComponent(id)}`)
}
