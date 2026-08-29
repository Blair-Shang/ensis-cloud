import { get, patch, post, setToken } from './request'
import { setStaff } from './auth'
import type { OpsAuditListResult, OpsLoginResult, OpsStaffAccount, OpsStaffListResult, OpsUser } from './types'

export function fetchOpsAuditLogs(params?: {
  action?: string
  actorKind?: string
  actorId?: string
  resourceKind?: string
  resourceId?: string
  limit?: number
  offset?: number
}): Promise<OpsAuditListResult> {
  return get<OpsAuditListResult>('/api/v1/ops/audit-logs', { params })
}

export function fetchOpsStaff(params?: {
  id?: string
  email?: string
  displayName?: string
  status?: string
  role?: string
  limit?: number
  offset?: number
}): Promise<OpsStaffListResult> {
  return get<OpsStaffListResult>('/api/v1/ops/staff', { params })
}

export function createOpsStaff(body: {
  email: string
  password: string
  displayName?: string
  role?: string
}): Promise<OpsStaffAccount> {
  return post<OpsStaffAccount>('/api/v1/ops/staff', body)
}

export function patchOpsStaff(
  id: string,
  body: { displayName?: string; status?: string },
): Promise<OpsStaffAccount> {
  return patch<OpsStaffAccount>(`/api/v1/ops/staff/${encodeURIComponent(id)}`, body)
}

export function resetOpsStaffPassword(id: string, password: string): Promise<{ ok: boolean }> {
  return post<{ ok: boolean }>(`/api/v1/ops/staff/${encodeURIComponent(id)}/password`, { password })
}

export async function changeOpsPassword(body: {
  currentPassword: string
  newPassword: string
}): Promise<OpsLoginResult> {
  const res = await post<OpsLoginResult>('/api/v1/ops/auth/password', body)
  setToken(res.accessToken)
  if (res.staff) setStaff(res.staff)
  return res
}

export function createOpsUser(body: {
  email: string
  password: string
  displayName?: string
}): Promise<OpsUser> {
  return post<OpsUser>('/api/v1/ops/users', body)
}

export function patchOpsUser(id: string, body: { status?: string }): Promise<OpsUser> {
  return patch<OpsUser>(`/api/v1/ops/users/${encodeURIComponent(id)}`, body)
}

export function resetOpsUserPassword(id: string, password: string): Promise<{ ok: boolean }> {
  return post<{ ok: boolean }>(`/api/v1/ops/users/${encodeURIComponent(id)}/password`, { password })
}
