import { Alert, Button, Space } from 'antd'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useUserStore } from '@/stores/user'
import { useTabsStore } from '@/stores/tabs'
import { buildAdminLoginPath } from '@/utils/tenant'

function formatRemaining(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000))
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

/** Always-visible notice while a platform admin is signed in as this tenant admin. */
export default function ImpersonationBanner() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const impersonation = useUserStore((s) => s.impersonation)
  const adminInfo = useUserStore((s) => s.adminInfo)
  const logout = useUserStore((s) => s.logout)
  const removeAllTabs = useTabsStore((s) => s.removeAllTabs)
  const [now, setNow] = useState(() => Date.now())

  const expiresAt = impersonation?.expires_at ? new Date(impersonation.expires_at).getTime() : 0

  useEffect(() => {
    if (!impersonation) return
    const id = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(id)
  }, [impersonation])

  if (!impersonation) return null

  const remaining = expiresAt ? expiresAt - now : 0
  const expired = !!expiresAt && remaining <= 0

  const exit = async () => {
    const loginPath = buildAdminLoginPath()
    removeAllTabs()
    await logout()
    // The console opened this tab; close it when the browser allows, else go to login.
    window.close()
    navigate(loginPath, { replace: true })
  }

  return (
    <Alert
      type="warning"
      showIcon
      banner
      message={
        <Space size={12} wrap>
          <span>
            {t('impersonate.banner', {
              operator: impersonation.platform_username || `#${impersonation.platform_admin_id}`,
              username: adminInfo?.username || '',
            })}
          </span>
          {expiresAt ? (
            <strong>{expired ? t('impersonate.expired') : t('impersonate.remaining', { time: formatRemaining(remaining) })}</strong>
          ) : null}
        </Space>
      }
      action={
        <Button size="small" danger onClick={() => void exit()}>
          {t('impersonate.exit')}
        </Button>
      }
    />
  )
}
