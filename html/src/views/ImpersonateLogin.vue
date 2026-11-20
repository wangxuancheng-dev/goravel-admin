<template>
  <div class="impersonate-login">
    <p>{{ statusText }}</p>
  </div>
</template>

<script setup>
// Landing page opened by the platform console: redeems a one-time support ticket.
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { exchangeImpersonationTicket } from '../api/auth'
import { useUserStore } from '../store/user'
import { useTabsStore } from '../store/tabs'
import { resetDynamicRoutes } from '../router'
import { buildAdminLoginPath, setTenantCode } from '../utils/tenant'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const tabsStore = useTabsStore()
const statusText = ref(t('impersonate.processing'))

onMounted(async () => {
  const ticket = String(route.query.ticket || '').trim()
  const tenantCode = String(route.query.tenant_code || '').trim().toLowerCase()
  // Keep the ticket out of history right away (it is single-use, but still).
  window.history.replaceState(window.history.state, '', window.location.pathname)
  const fail = () => {
    statusText.value = t('impersonate.failed')
    router.replace(buildAdminLoginPath(tenantCode || undefined))
  }

  // Drop any session already in this browser before taking over (clears tenant code too).
  await userStore.logout(true)
  tabsStore.removeAllTabs()
  resetDynamicRoutes()
  if (tenantCode) setTenantCode(tenantCode)

  if (!ticket) {
    ElMessage.error(t('messages.impersonation_ticket_invalid'))
    fail()
    return
  }
  try {
    const res = await exchangeImpersonationTicket(ticket)
    const token = res?.data?.token
    if (!token) throw new Error('no token')
    userStore.setToken(token)
    await userStore.fetchUserInfo(true)
    ElMessage.success(t('login.login_success'))
    router.replace('/')
  } catch {
    // request interceptor already shows the server error
    fail()
  }
})
</script>

<style scoped>
.impersonate-login {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
}
</style>
