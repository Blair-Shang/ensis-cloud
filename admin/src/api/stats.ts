import { get } from './request'
import type { OpsStats } from './types'

export function fetchOpsStats(): Promise<OpsStats> {
  return get<OpsStats>('/api/v1/ops/stats')
}
