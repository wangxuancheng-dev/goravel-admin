import request from '@/utils/request'
import type { ApiResponse } from '@/types'

export function getTraceAggregate(params?: Record<string, unknown>) {
  return request({ url: '/observability/trace', method: 'get', params }) as Promise<ApiResponse>
}

export function getSlowSqlTop(params?: Record<string, unknown>) {
  return request({ url: '/observability/slow-sql/top', method: 'get', params }) as Promise<ApiResponse>
}

export function getApiPerformanceOverview(params?: Record<string, unknown>) {
  return request({
    url: '/observability/api-performance/overview',
    method: 'get',
    params,
    skipErrorMessage: true,
  }) as Promise<ApiResponse>
}

export function getApiPerformanceTraces(params?: Record<string, unknown>) {
  return request({
    url: '/observability/api-performance/traces',
    method: 'get',
    params,
  }) as Promise<ApiResponse>
}

export function getAuditTimeline(params?: Record<string, unknown>) {
  return request({
    url: '/observability/audit-timeline',
    method: 'get',
    params,
  }) as Promise<ApiResponse>
}

export function getQueueDashboard() {
  return request({ url: '/observability/queue-dashboard', method: 'get' }) as Promise<ApiResponse>
}

export function getQueueAlertStatus() {
  return request({ url: '/observability/queue-alert', method: 'get' }) as Promise<ApiResponse>
}

export function testQueueAlert() {
  return request({ url: '/observability/queue-alert/test', method: 'post' }) as Promise<ApiResponse>
}

export function getPprofStatus() {
  return request({
    url: '/observability/pprof/status',
    method: 'get',
    skipErrorMessage: true,
  }) as Promise<ApiResponse>
}

export function verifyPprofToken(data: { token: string }) {
  return request({
    url: '/observability/pprof/verify',
    method: 'post',
    data,
    skipErrorMessage: true,
  }) as Promise<ApiResponse>
}

export function getPprofCpuHotspots(data: Record<string, unknown>) {
  return request({
    url: '/observability/pprof/cpu-hotspots',
    method: 'post',
    data,
    skipErrorMessage: true,
  }) as Promise<ApiResponse>
}

export function getPprofMemoryHotspots(data: Record<string, unknown>) {
  return request({
    url: '/observability/pprof/memory-hotspots',
    method: 'post',
    data,
    skipErrorMessage: true,
  }) as Promise<ApiResponse>
}
