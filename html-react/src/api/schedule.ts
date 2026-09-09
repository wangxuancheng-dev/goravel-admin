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

export function getScheduleList() {
  return request.get<{ list: ScheduleTask[]; total: number }>('/schedules')
}

export function runSchedule(command: string) {
  return request.post<{ result: ScheduleRunResult }>(
    '/schedules/run',
    { command },
    { timeout: 120000 },
  )
}
