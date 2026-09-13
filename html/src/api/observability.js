import request from '../utils/request'

export function getTraceAggregate(params) {
  return request({
    url: '/observability/trace',
    method: 'get',
    params
  })
}

export function getSlowSqlTop(params) {
  return request({
    url: '/observability/slow-sql/top',
    method: 'get',
    params
  })
}

export function getApiPerformanceOverview(params) {
  return request({
    url: '/observability/api-performance/overview',
    method: 'get',
    params,
    skipErrorMessage: true
  })
}

export function getApiPerformanceTraces(params) {
  return request({
    url: '/observability/api-performance/traces',
    method: 'get',
    params
  })
}

export function getAuditTimeline(params) {
  return request({
    url: '/observability/audit-timeline',
    method: 'get',
    params
  })
}

export function getQueueDashboard() {
  return request({
    url: '/observability/queue-dashboard',
    method: 'get'
  })
}

export function getQueueAlertStatus() {
  return request({
    url: '/observability/queue-alert',
    method: 'get'
  })
}

export function testQueueAlert() {
  return request({
    url: '/observability/queue-alert/test',
    method: 'post'
  })
}

export function getPprofStatus() {
  return request({
    url: '/observability/pprof/status',
    method: 'get',
    skipErrorMessage: true
  })
}

export function verifyPprofToken(data) {
  return request({
    url: '/observability/pprof/verify',
    method: 'post',
    data,
    skipErrorMessage: true
  })
}

export function getPprofCpuHotspots(data) {
  return request({
    url: '/observability/pprof/cpu-hotspots',
    method: 'post',
    data,
    skipErrorMessage: true
  })
}

export function getPprofMemoryHotspots(data) {
  return request({
    url: '/observability/pprof/memory-hotspots',
    method: 'post',
    data,
    skipErrorMessage: true
  })
}
