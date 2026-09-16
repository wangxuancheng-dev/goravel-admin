import request from '../utils/request'

// 登录
export function login(data) {
  return request({
    url: '/login',
    method: 'post',
    data
  })
}

// 获取当前用户信息
export function getInfo() {
  return request({
    url: '/info',
    method: 'get'
  })
}

// 退出登录
export function logout() {
  return request({
    url: '/logout',
    method: 'post'
  })
}

// 获取登录验证码
export function getLoginCaptcha(params) {
  const query = {}
  if (params?.check) {
    query.check = 1
  }
  const username = typeof params?.username === 'string' ? params.username.trim() : ''
  if (username) {
    query.username = username
  }
  return request({
    url: '/login/captcha',
    method: 'get',
    params: Object.keys(query).length ? query : undefined
  })
}

// 登录页白标（站点名 / Logo / 主题色）
export function getLoginBranding() {
  return request({
    url: '/login/branding',
    method: 'get',
    skipErrorMessage: true
  })
}

// 获取谷歌验证码绑定状态
export function getGoogleAuthenticatorStatus() {
  return request({
    url: '/google-authenticator/status',
    method: 'get'
  })
}

// 获取谷歌验证码二维码
export function getGoogleAuthenticatorQRCode() {
  return request({
    url: '/google-authenticator/qrcode',
    method: 'get'
  })
}

// 绑定谷歌验证码
export function bindGoogleAuthenticator(data) {
  return request({
    url: '/google-authenticator/bind',
    method: 'post',
    data
  })
}

// 解绑谷歌验证码
export function unbindGoogleAuthenticator(data) {
  return request({
    url: '/google-authenticator/unbind',
    method: 'post',
    data
  })
}

