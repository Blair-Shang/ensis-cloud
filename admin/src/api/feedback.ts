import { get, getToken, patch } from './request'
import type { OpsFeedback, OpsFeedbackListResult, OpsFeedbackPatch } from './types'

const API_BASE = (import.meta.env.VITE_CLOUD_API_BASE || '/cloud').replace(/\/$/, '')

export function fetchOpsFeedback(params?: {
  id?: string
  title?: string
  contact?: string
  category?: string
  status?: string
  product?: string
  userId?: string
  clientVersion?: string
  limit?: number
  offset?: number
}): Promise<OpsFeedbackListResult> {
  return get<OpsFeedbackListResult>('/api/v1/ops/feedback', { params })
}

export function updateOpsFeedback(id: string, body: OpsFeedbackPatch): Promise<OpsFeedback> {
  return patch<OpsFeedback>(`/api/v1/ops/feedback/${encodeURIComponent(id)}`, body)
}

/** 员工鉴权拉取反馈截图，返回 blob URL（调用方 revoke）。 */
export async function fetchOpsFeedbackAttachmentURL(
  feedbackId: string,
  attachmentId: string,
): Promise<string> {
  const token = getToken()
  const res = await fetch(
    `${API_BASE}/api/v1/feedback/${encodeURIComponent(feedbackId)}/attachments/${encodeURIComponent(attachmentId)}`,
    {
      headers: {
        Accept: 'image/*,*/*',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    },
  )
  if (!res.ok) throw new Error(`http_${res.status}`)
  return URL.createObjectURL(await res.blob())
}
