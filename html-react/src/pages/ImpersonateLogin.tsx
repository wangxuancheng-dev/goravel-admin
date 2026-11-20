import { useEffect, useState } from 'react'
import { App, Typography } from 'antd'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { exchangeImpersonationTicket } from '@/api/auth'
import { useUserStore } from '@/stores/user'
import { buildAdminLoginPath, setTenantCode } from '@/utils/tenant'

type Outcome = { ok: boolean; tenantCode: string }

/**
 * One redemption per ticket for the whole module, not per component instance:
 * StrictMode re-runs the effect, and loading the user's menus rebuilds the router,
 * which remounts this page after the ticket has already been stripped from the URL.
 */
let current: { ticket: string; promise: Promise<Outcome> } | null = null

/** Landing page opened by the platform console: redeems a one-time support ticket. */
export default function ImpersonateLoginPage() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [status, setStatus] = useState(t('impersonate.processing'))

  useEffect(() => {
    let mounted = true
    const ticket = String(params.get('ticket') || '').trim()
    const tenantCode = String(params.get('tenant_code') || '').trim().toLowerCase()

    // A remount (no ticket left in the URL, or the same ticket) joins the run in flight.
    if (!current || (ticket && ticket !== current.ticket)) {
      // Keep the ticket out of history right away (it is single-use, but still).
      window.history.replaceState(window.history.state, '', window.location.pathname)
      const state = useUserStore.getState()
      const promise = (async (): Promise<Outcome> => {
        // Drop any session already in this browser before taking over (clears tenant code too).
        await state.logout(true)
        if (tenantCode) setTenantCode(tenantCode)
        if (!ticket) {
          message.error(t('messages.impersonation_ticket_invalid'))
          return { ok: false, tenantCode }
        }
        try {
          const res = await exchangeImpersonationTicket(ticket)
          const token = res.data?.token
          if (!token) throw new Error('no token')
          useUserStore.getState().setToken(token)
          await useUserStore.getState().fetchUserInfo(true)
          message.success(t('login.login_success'))
          return { ok: true, tenantCode }
        } catch {
          // request wrapper already toasts the server error
          return { ok: false, tenantCode }
        }
      })()
      current = { ticket, promise }
    }

    void current.promise.then((outcome) => {
      if (!mounted) return
      if (outcome.ok) {
        navigate('/', { replace: true })
        return
      }
      setStatus(t('impersonate.failed'))
      navigate(buildAdminLoginPath(outcome.tenantCode || undefined), { replace: true })
    })
    return () => {
      mounted = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount once
  }, [])

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Typography.Text>{status}</Typography.Text>
    </div>
  )
}
