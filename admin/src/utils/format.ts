/** 运营后台展示用格式化工具 */

import { ApiError } from '@/api/request'

/** API 错误文案；鉴权跳转中返回空串，调用方勿写入页面 */
export function formatApiError(e: unknown, fallback = '操作失败'): string {
  if (e instanceof ApiError) {
    if (e.authRedirect || e.code === 'unauthorized' || e.status === 401) return ''
    switch (e.code) {
      case 'email_taken':
        return '邮箱已被占用'
      case 'invalid_email':
        return '邮箱格式不正确'
      case 'invalid_password':
        return '密码至少 10 位，且须含字母和数字'
      case 'forbidden':
        return '没有权限执行该操作'
      case 'last_admin':
        return '不能停用或降级最后一位管理员'
      case 'invalid_credentials':
        return '当前密码不正确'
      case 'cannot_disable_self':
        return '不能停用自己的账号'
      case 'invalid_status':
        return '状态无效'
      case 'account_disabled':
        return '账号已停用'
      case 'not_found':
        return '资源不存在'
      case 'version_taken':
        return '该平台/架构下版本号已存在'
      case 'invalid_version':
        return '版本号须为 X.Y.Z'
      case 'invalid_sha256':
        return 'SHA-256 须为 64 位小写十六进制'
      case 'download_url_https_required':
        return '下载地址必须是 HTTPS'
      case 'download_url_host_forbidden':
        return '下载地址主机不允许（禁内网/本机/IP）'
      case 'download_url_host_not_allowed':
        return '下载地址不在允许的域名列表中'
      case 'download_url_credentials_forbidden':
        return '下载地址不能包含账号密码'
      case 'invalid_download_url':
        return '下载地址无效'
      case 'release_yanked':
        return '已撤回的发布不可再编辑/发布'
      case 'invalid_title':
        return '请填写标题'
      case 'notes_too_large':
        return '更新说明过长'
      case 'rate_limited':
        return '请求过于频繁，请稍后再试'
      default:
        return e.code || fallback
    }
  }
  return e instanceof Error ? e.message : fallback
}

export function formatDateTime(value?: string | null): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

export function formatDate(value?: string | null): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleDateString('zh-CN')
}

export function accountStatusLabel(status?: string): string {
  switch (status) {
    case 'active':
      return '正常'
    case 'disabled':
      return '停用'
    default:
      return status || '—'
  }
}

export function accountStatusVariant(status?: string): 'success' | 'danger' | 'default' {
  switch (status) {
    case 'active':
      return 'success'
    case 'disabled':
      return 'danger'
    default:
      return 'default'
  }
}

export function feedbackStatusLabel(status?: string): string {
  switch (status) {
    case 'open':
      return '待处理'
    case 'triaged':
      return '处理中'
    case 'closed':
      return '已关闭'
    default:
      return status || '—'
  }
}

export function feedbackStatusVariant(status?: string): 'info' | 'warning' | 'default' | 'success' {
  switch (status) {
    case 'open':
      return 'info'
    case 'triaged':
      return 'warning'
    case 'closed':
      return 'success'
    default:
      return 'default'
  }
}

export function feedbackCategoryLabel(category?: string): string {
  switch (category) {
    case 'bug':
      return '缺陷'
    case 'feature':
      return '建议'
    case 'other':
      return '其他'
    default:
      return category || '—'
  }
}

export function formatFileSize(bytes?: number | null): string {
  if (bytes == null || !Number.isFinite(bytes) || bytes < 0) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function actorKindLabel(kind?: string): string {
  switch (kind) {
    case 'staff':
      return '员工'
    case 'user':
      return '用户'
    case 'system':
      return '系统'
    case 'guest':
      return '访客'
    default:
      return kind || '—'
  }
}

export function initialOf(name?: string): string {
  const s = (name || '').trim()
  return s ? s.slice(0, 1).toUpperCase() : '?'
}

export function detailText(d: unknown): string {
  if (d == null) return '—'
  if (typeof d === 'string') return d || '—'
  try {
    const s = JSON.stringify(d)
    return s === '{}' ? '—' : s
  } catch {
    return String(d)
  }
}
