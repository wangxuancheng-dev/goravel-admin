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
    timeout: 120000,
  })
}
