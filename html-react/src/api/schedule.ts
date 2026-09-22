import request from '@/utils/request'

export interface ScheduleTask {
  id: string
  command: string
  cron: string
  name?: string
  description?: string
  on_one_server?: boolean
  skip_if_still_running?: boolean
  delay_if_still_running?: boolean
  last_run_at?: string
  last_status?: string
  last_error?: string
  last_output?: string
  last_duration_ms?: number
  last_triggered_by?: string
}

export interface ScheduleRunResult {
  command: string
  status: string
  error?: string
  output?: string
  duration_ms?: number
  run_at?: string
  triggered_by?: string
}

export interface FlexibleScheduleRow {
  id: number
  name: string
  handler: string
  handler_name?: string
  cron_expr: string
  timezone: string
  tenant_id: number
  tenant_code?: string
  enabled: boolean
  last_run_at?: string
  last_status?: string
  last_error?: string
  last_output?: string
  last_duration_ms?: number
  last_slot?: string
  next_runs?: string[]
}

export interface FlexibleHandlerMeta {
  key: string
  name: string
  description: string
  tenant_aware: boolean
}

export function getScheduleList() {
  return request.get<{ list: ScheduleTask[]; total: number }>('/schedules')
}

export function runSchedule(command: string) {
  return request.post<{ result: ScheduleRunResult }>(
    '/schedules/run',
    { command },
    // Align with long ops (backup); server run routes disable RequestTimeout.
    { timeout: 600000 },
  )
}

export function getFlexibleScheduleList() {
  return request.get<{ list: FlexibleScheduleRow[]; total: number }>('/flexible-schedules')
}

export function getFlexibleScheduleHandlers() {
  return request.get<{ list: FlexibleHandlerMeta[] }>('/flexible-schedules/handlers')
}

export function createFlexibleSchedule(data: Record<string, unknown>) {
  return request.post<{ flexible_schedule: FlexibleScheduleRow }>('/flexible-schedules', data)
}

export function updateFlexibleSchedule(id: number, data: Record<string, unknown>) {
  return request.put<{ flexible_schedule: FlexibleScheduleRow }>(`/flexible-schedules/${id}`, data)
}

export function deleteFlexibleSchedule(id: number) {
  return request.delete(`/flexible-schedules/${id}`)
}

export function runFlexibleSchedule(id: number, force = false) {
  const q = force ? '?force=1' : ''
  return request.post<{ flexible_schedule: FlexibleScheduleRow }>(
    `/flexible-schedules/${id}/run${q}`,
    {},
    { timeout: 600000 },
  )
}

export function previewFlexibleSchedule(data: { cron_expr: string; timezone?: string; count?: number }) {
  return request.post<{ next_runs: string[] }>('/flexible-schedules/preview', data)
}
