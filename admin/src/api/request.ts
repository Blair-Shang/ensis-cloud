/**
 * 全局 HTTP 客户端（axios）。业务请走 api/*.ts，勿直接在视图里拼请求。
 * access 只放内存；Ops refresh 走 HttpOnly Cookie（withCredentials）。
 */

import axios, { AxiosError, type AxiosRequestConfig } from 'axios'

const API_BASE = (import.meta.env.VITE_CLOUD_API_BASE || '/niuma/cloud').replace(/\/$/, '')
const STAFF_KEY = 'niuma.ops.staff'

export class ApiError extends Error {
  constructor(
    public code: string,
    public status: number,
    /** 已触发登出跳转，页面勿再展示该错误 */
    public authRedirect = false,
  ) {
    super(code)
    this.name = 'ApiError'
  }
}

let accessToken: string | null = null

export function getToken(): string | null {
  return accessToken
}

export function setToken(token: string | null): void {
  accessToken = token
}

export function clearToken(): void {
  accessToken = null
}

/** 清除内存 access + 员工缓存（退出 / 401 共用） */
export function clearSession(): void {
  clearToken()
  localStorage.removeItem(STAFF_KEY)
  sessionStorage.removeItem(STAFF_KEY)
}

let redirectingToLogin = false

/** access / refresh 均失效：清会话并用整页跳转，避免 SPA 软跳失败留在原页 */
function redirectToLogin(): void {
  if (redirectingToLogin) return
  redirectingToLogin = true
  clearSession()
  const base = import.meta.env.BASE_URL || '/niuma/ops/'
  const path = window.location.pathname + window.location.search
  const opsBase = base.endsWith('/') ? base.slice(0, -1) : base
  const loginPath = `${opsBase}/login`
  if (path === loginPath || path.startsWith(`${loginPath}?`)) return
  const redirect =
    path && path !== loginPath && !path.endsWith('/login')
      ? path.replace(new RegExp(`^${opsBase}`), '') || '/'
      : undefined
  const qs = redirect && redirect !== '/' ? `?redirect=${encodeURIComponent(redirect)}` : ''
  window.location.assign(`${loginPath}${qs}`)
}

export type RequestOptions = Omit<AxiosRequestConfig, 'data' | 'params' | 'url' | 'method' | 'auth'> & {
  /** 默认 true；登录 / refresh / logout 传 false */
  auth?: boolean
  body?: unknown
  params?: Record<string, string | number | undefined | null>
  method?: string
  /** 内部：refresh 后重试标记，避免死循环 */
  _retried?: boolean
}

const client = axios.create({
  baseURL: API_BASE,
  timeout: 60_000,
  withCredentials: true,
  headers: { Accept: 'application/json' },
})

function isUnauthorized(status: number, code: string): boolean {
  return status === 401 || code === 'unauthorized' || code === 'invalid_refresh'
}

let refreshInFlight: Promise<boolean> | null = null

/** 用 HttpOnly refresh cookie 换新 access；失败返回 false */
export async function tryRestoreSession(): Promise<boolean> {
  return tryRefreshAccess()
}

async function tryRefreshAccess(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      try {
        const res = await client.post<{
          accessToken?: string
          staff?: { id: string; email: string; displayName: string; role?: string }
        }>('/api/v1/ops/auth/refresh', {})
        if (!res.data?.accessToken) return false
        setToken(res.data.accessToken)
        if (res.data.staff) {
          sessionStorage.setItem(STAFF_KEY, JSON.stringify(res.data.staff))
        }
        return true
      } catch {
        return false
      } finally {
        refreshInFlight = null
      }
    })()
  }
  return refreshInFlight
}

function errorFromAxios(err: unknown): ApiError {
  if (err instanceof ApiError) return err
  if (err instanceof AxiosError) {
    const status = err.response?.status ?? 0
    const data = err.response?.data as { error?: string } | undefined
    const code = typeof data?.error === 'string' ? data.error : err.code || `http_${status || 0}`
    return new ApiError(code, status)
  }
  return new ApiError(err instanceof Error ? err.message : 'network_error', 0)
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { auth = true, body, params, headers: initHeaders, _retried, method, ...rest } = options
  const headers: Record<string, string> = {}
  if (initHeaders && typeof initHeaders === 'object' && !Array.isArray(initHeaders)) {
    for (const [k, v] of Object.entries(initHeaders)) {
      if (v == null || v === false) continue
      headers[k] = String(v)
    }
  }

  let data: unknown = body
  if (body !== undefined && body !== null) {
    if (!(typeof body === 'string' || body instanceof FormData || body instanceof Blob)) {
      if (!headers['Content-Type']) headers['Content-Type'] = 'application/json'
    }
  }

  if (auth) {
    const tok = getToken()
    if (tok) headers.Authorization = `Bearer ${tok}`
  }

  const cleanParams: Record<string, string | number> = {}
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      if (v === undefined || v === null || v === '') continue
      cleanParams[k] = v
    }
  }

  try {
    const res = await client.request<T>({
      ...rest,
      url: path,
      method: (method || 'GET') as AxiosRequestConfig['method'],
      data,
      params: Object.keys(cleanParams).length ? cleanParams : undefined,
      headers,
    })
    return res.data
  } catch (err) {
    const apiErr = errorFromAxios(err)
    if (auth && isUnauthorized(apiErr.status, apiErr.code)) {
      if (!_retried) {
        const refreshed = await tryRefreshAccess()
        if (refreshed) {
          return request<T>(path, { ...options, _retried: true })
        }
      }
      redirectToLogin()
      throw new ApiError(apiErr.code, apiErr.status, true)
    }
    throw apiErr
  }
}

export function get<T>(path: string, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
  return request<T>(path, { ...options, method: 'GET' })
}

export function post<T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
  return request<T>(path, { ...options, method: 'POST', body })
}

export function patch<T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
  return request<T>(path, { ...options, method: 'PATCH', body })
}

export function del<T>(path: string, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
  return request<T>(path, { ...options, method: 'DELETE' })
}
