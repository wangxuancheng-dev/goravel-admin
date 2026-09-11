/**
 * Auth endpoints skip global 401→logout (Login page handles errors).
 * Must NOT match /login-logs (substring of /login).
 */
export function isAuthEndpointUrl(url = ''): boolean {
  const path = url.split('?')[0].replace(/\/+$/, '')
  return (
    path === 'login' ||
    path === 'login/captcha' ||
    path === 'logout' ||
    path.endsWith('/login') ||
    path.endsWith('/login/captcha') ||
    path.endsWith('/logout')
  )
}
