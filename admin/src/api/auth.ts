import { clearSession, post, setToken } from './request'
import type { OpsLoginResult, OpsStaff } from './types'

const STAFF_KEY = 'niuma.ops.staff'

export async function loginOps(email: string, password: string): Promise<OpsLoginResult> {
  const res = await post<OpsLoginResult>(
    '/api/v1/ops/auth/login',
    { email, password },
    { auth: false },
  )
  setToken(res.accessToken)
  setStaff(res.staff)
  return res
}

export async function logoutOps(): Promise<void> {
  try {
    await post<{ ok: boolean }>('/api/v1/ops/auth/logout', {}, { auth: false })
  } catch {
    // 本地会话仍须清掉
  }
  clearSession()
}

export function setStaff(staff: OpsStaff | null): void {
  if (staff) sessionStorage.setItem(STAFF_KEY, JSON.stringify(staff))
  else sessionStorage.removeItem(STAFF_KEY)
}

export function clearStaff(): void {
  sessionStorage.removeItem(STAFF_KEY)
  localStorage.removeItem(STAFF_KEY)
}

export function getStaff(): OpsStaff | null {
  try {
    const raw = sessionStorage.getItem(STAFF_KEY) || localStorage.getItem(STAFF_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as OpsStaff
    if (!parsed || typeof parsed.email !== 'string') return null
    return parsed
  } catch {
    return null
  }
}

export function isOpsAdmin(staff = getStaff()): boolean {
  return staff?.role === 'admin'
}
