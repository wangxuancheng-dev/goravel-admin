import { useEffect, useState } from 'react'
import { App, Typography } from 'antd'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useUserStore } from '@/stores/user'
import { buildAdminLoginPath, setTenantCode } from '@/utils/tenant'

export default function OidcCallbackPage() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const setToken = useUserStore((s) => s.setToken)
  const fetchUserInfo = useUserStore((s) => s.fetchUserInfo)
  const [status, setStatus] = useState(t('login.oidc_processing'))

  useEffect(() => {
    const run = async () => {
      const err = String(params.get('error') || '').trim()
      const token = String(params.get('token') || '').trim()
      const tenantCode = String(params.get('tenant_code') || '').trim().toLowerCase()
      if (tenantCode) {
        setTenantCode(tenantCode)
      }
      if (err) {
        setStatus(t('login.oidc_failed'))
        message.error(t(`messages.${err}`, { defaultValue: err }))
        navigate(buildAdminLoginPath(tenantCode || undefined), { replace: true })
        return
      }
      if (!token) {
        setStatus(t('login.oidc_failed'))
        message.error(t('login.oidc_failed'))
        navigate(buildAdminLoginPath(tenantCode || undefined), { replace: true })
        return
      }
      try {
        setToken(token)
        await fetchUserInfo(true)
        message.success(t('login.login_success'))
        setStatus(t('login.login_success'))
        const mustChange = !!useUserStore.getState().adminInfo?.must_change_password
        navigate(mustChange ? '/profile/password' : '/', { replace: true })
      } catch {
        setStatus(t('login.oidc_failed'))
        message.error(t('login.oidc_failed'))
        navigate(buildAdminLoginPath(tenantCode || undefined), { replace: true })
      }
    }
    void run()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount once
  }, [])

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Typography.Text>{status}</Typography.Text>
    </div>
  )
}
