import request from '../utils/request'

export function getScheduleList() {
  return request({
    url: '/schedules',
    method: 'get',
  })
}

export function runSchedule(command) {
  return request({
    url: '/schedules/run',
    method: 'post',
    data: { command },
    // Align with long ops (backup); server run routes disable RequestTimeout.
    timeout: 600000,
  })
}

export function getFlexibleScheduleList() {
  return request({
    url: '/flexible-schedules',
    method: 'get',
  })
}

export function getFlexibleScheduleHandlers() {
  return request({
    url: '/flexible-schedules/handlers',
    method: 'get',
  })
}

export function createFlexibleSchedule(data) {
  return request({
    url: '/flexible-schedules',
    method: 'post',
    data,
  })
}

export function updateFlexibleSchedule(id, data) {
  return request({
    url: `/flexible-schedules/${id}`,
    method: 'put',
    data,
  })
}

export function deleteFlexibleSchedule(id) {
  return request({
    url: `/flexible-schedules/${id}`,
    method: 'delete',
  })
}

export function runFlexibleSchedule(id, force = false) {
  return request({
    url: `/flexible-schedules/${id}/run${force ? '?force=1' : ''}`,
    method: 'post',
    timeout: 600000,
  })
}

export function previewFlexibleSchedule(data) {
  return request({
    url: '/flexible-schedules/preview',
    method: 'post',
    data,
  })
}
