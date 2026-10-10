import request from '@/utils/request'
import type { ApiResponse } from '@/types'

export interface MyPlanFeature {
  key: string
  name: string
  kind: 'module' | 'capability' | 'other'
  enabled: boolean
  always_on?: boolean
}

export interface MyPlanLimit {
  key: string
  name: string
  limit: number
  used: number
  remaining: number
  unlimited: boolean
}

export interface MyPlanPayload {
  enabled: boolean
  version?: number
  plan?: { code: string; name: string; description?: string } | null
  subscription?: {
    status: string
    billing_cycle?: string
    starts_at?: string | null
    ends_at?: string | null
    trial_ends_at?: string | null
  } | null
  features?: MyPlanFeature[]
  limits?: MyPlanLimit[]
}

/** Read-only plan summary for the current tenant (GET /entitlements/me). */
export function getMyPlan() {
  return request({
    url: '/entitlements/me',
    method: 'get',
  }) as Promise<ApiResponse<MyPlanPayload>>
}
