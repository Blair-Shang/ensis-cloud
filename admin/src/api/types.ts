/**
 * Ops（运营后台）API 共用类型定义。
 * 与 `server` 的 `/niuma/cloud/api/v1/ops/*` 响应对齐；视图层请从 `@/api` 引用。
 */

/** 员工账号（与终端用户 users 分离，仅能登录 /ops） */
export type OpsStaff = {
  /** 员工 ID，如 staff_xxx */
  id: string
  /** 登录邮箱 */
  email: string
  /** 显示名 */
  displayName: string
  /** admin 超管 / operator 日常运营 */
  role?: 'admin' | 'operator' | string
}

/** POST /api/v1/ops/auth/login 成功响应 */
export type OpsLoginResult = {
  /** 员工 JWT access token，后续 Ops 请求放 Authorization */
  accessToken: string
  /** access 过期时间（RFC3339） */
  expiresAt: string
  /** 当前员工信息 */
  staff: OpsStaff
}

/** GET /api/v1/ops/stats 概览统计 */
export type OpsStats = {
  /** 注册终端用户总数 */
  users: number
  /** 反馈工单总数 */
  feedback: number
  /** 近 7 日新增反馈数 */
  feedbackLast7d: number
  /** 更新安装包拉起/下载累计次数 */
  downloadHits?: number
  /** 当日（UTC）调用过系统模型的用户数 */
  aiUsersToday?: number
  /** 当日（UTC）系统模型对话次数 */
  aiRequestsToday?: number
}

/**
 * 终端用户（桌面/官网注册的产品账户）。
 * 列表接口会附带 feedbackCount / sessionCount。
 */
export type OpsUser = {
  id: string
  email: string
  displayName: string
  /** 账户状态：active / disabled 等 */
  status: string
  /** 注册时间（RFC3339 字符串） */
  createdAt: string
  /** 该用户提交的反馈条数（列表接口） */
  feedbackCount?: number
  /** 未吊销的 refresh 会话数（列表接口） */
  sessionCount?: number
  /** 邮箱验证完成时间；未验证则为 null/缺省 */
  emailVerifiedAt?: string | null
}

/** GET /api/v1/ops/users 分页列表 */
export type OpsUserListResult = {
  items: OpsUser[]
  total: number
}

/**
 * 反馈工单。
 * 对外 JSON 字段固定；库内映射为专业列名（subject/content_text/ticket_status…）。
 * status：open（新建）/ triaged（已分拣）/ closed（关闭）
 */
export type OpsFeedbackAttachment = {
  id: string
  name: string
  contentType: string
  size: number
  file?: string
}

export type OpsFeedback = {
  id: string
  /** 关联用户 ID；官网匿名提交时可为空 */
  userId?: string
  title: string
  body: string
  /** 联系邮箱（回复通知目标） */
  contact?: string
  /** 分类：bug / feature / other 等 */
  category: string
  status: string
  /** 内部备注（仅员工可见） */
  adminNote?: string
  /** 对用户可见的回复 */
  staffReply?: string
  /** 回复时间 */
  staffReplyAt?: string | null
  /** 客户端版本号 */
  clientVersion?: string
  /** 产品标识，如 niuma */
  product?: string
  /** 用户截图 */
  attachments?: OpsFeedbackAttachment[]
  createdAt: string
  updatedAt?: string
}

/** GET /api/v1/ops/feedback 分页列表 */
export type OpsFeedbackListResult = {
  items: OpsFeedback[]
  total: number
}

/** GET /api/v1/ops/users/:id 用户详情（含反馈摘要） */
export type OpsUserDetail = {
  user: OpsUser
  sessionCount: number
  feedback: OpsFeedback[]
}

/** PATCH /api/v1/ops/feedback/:id 请求体 */
export type OpsFeedbackPatch = {
  /** 目标状态：open | triaged | closed */
  status?: string
  /** 内部备注；省略则服务端保留原值 */
  adminNote?: string
  /** 对用户可见回复；变更后邮件通知 contact，并出现在 /feedback/mine */
  staffReply?: string
}

/** 员工账号列表项（含状态） */
export type OpsStaffAccount = OpsStaff & {
  status: string
  createdAt: string
  updatedAt?: string
}

export type OpsStaffListResult = {
  items: OpsStaffAccount[]
  total: number
}

/** 审计日志 */
export type OpsAuditLog = {
  id: string
  actorKind: string
  actorID: string
  actionCode: string
  resourceKind: string
  resourceID: string
  detail: Record<string, unknown> | unknown
  sourceIPHash: string
  createdAt: string
}

export type OpsAuditListResult = {
  items: OpsAuditLog[]
  total: number
}

/** 应用发布记录（Ops） */
export type OpsRelease = {
  id: string
  product: string
  version: string
  channel: string
  platform: string
  arch: string
  minSupportedVersion?: string
  forceUpdate: boolean
  title: string
  notesMd: string
  downloadUrl: string
  sha256: string
  fileSize: number
  status: string
  publishedAt?: string | null
  createdBy?: string
  createdAt: string
  updatedAt: string
}

export type OpsReleaseListResult = {
  items: OpsRelease[]
  total: number
}

export type OpsReleaseCreate = {
  product?: string
  version: string
  channel?: string
  platform: string
  arch: string
  minSupportedVersion?: string
  forceUpdate?: boolean
  title: string
  notesMd?: string
  downloadUrl: string
  sha256: string
  fileSize?: number
}

/** 系统模型上游接入点（列表不含密钥明文）。 */
export type OpsAIEndpoint = {
  id: string
  providerName: string
  baseUrl: string
  hasApiSecret: boolean
  enabled: boolean
  defaultModel: string
  dailyRequestLimit: number
  dailyTokenLimit: number
  sortSeq: number
  createdByStaffId: string
  updatedByStaffId: string
  createdAt: string
  updatedAt: string
}

export type OpsAIModel = {
  id: string
  endpointId: string
  code: string
  label: string
  recordStatus: string
  sortSeq: number
  createdByStaffId: string
  updatedByStaffId: string
  createdAt: string
  updatedAt: string
}

export type OpsAIEndpointWrite = {
  providerName?: string
  baseUrl?: string
  apiSecret?: string
  enabled?: boolean
  defaultModel?: string
  dailyRequestLimit?: number
  dailyTokenLimit?: number
  sortSeq?: number
}

export type OpsAIEndpointTest = {
  endpointId?: string
  baseUrl?: string
  apiSecret?: string
  model?: string
}

export type OpsAIEndpointTestResult = {
  ok: boolean
  status: number
  latencyMs: number
  message: string
  model: string
}

/** GET /api/v1/ops/ai/usage 按日用量一行 */
export type OpsAIUsageRow = {
  userId: string
  email: string
  displayName: string
  status: string
  usageDay: string
  requestCount: number
  tokenCount: number
  updatedAt: string
}

export type OpsAIUsageSummary = {
  usageDay: string
  userCount: number
  requestCount: number
  tokenCount: number
}

export type OpsAIUsageResult = {
  day: string
  summary: OpsAIUsageSummary
  dailyRequestLimit: number
  dailyTokenLimit: number
  /** 含选定日在内的近 7 日汇总（缺日补零） */
  trend?: OpsAIUsageSummary[]
  /** 当日请求次数前 8，供排行图 */
  top?: OpsAIUsageRow[]
  items: OpsAIUsageRow[]
  total: number
}

export type OpsReleasePatch = {
  minSupportedVersion?: string
  forceUpdate?: boolean
  title?: string
  notesMd?: string
  downloadUrl?: string
  sha256?: string
  fileSize?: number
}
