<template>
  <div class="oidc-callback">
    <p>{{ statusText }}</p>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../store/user'
import { buildAdminLoginPath, setTenantCode } from '../utils/tenant'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const statusText = ref(t('login.oidc_processing'))

onMounted(async () => {
  const err = String(route.query.error || '').trim()
  const token = String(route.query.token || '').trim()
  const tenantCode = String(route.query.tenant_code || '').trim().toLowerCase()
  if (tenantCode) {
    setTenantCode(tenantCode)
  }
  if (err) {
    statusText.value = t('login.oidc_failed')
    ElMessage.error(t(`messages.${err}`, err) || t('login.oidc_failed'))
    router.replace(buildAdminLoginPath(tenantCode || undefined))
    return
  }
  if (!token) {
    statusText.value = t('login.oidc_failed')
    ElMessage.error(t('login.oidc_failed'))
    router.replace(buildAdminLoginPath(tenantCode || undefined))
    return
  }
  try {
    userStore.setToken(token)
    await userStore.fetchUserInfo(true)
    ElMessage.success(t('login.login_success'))
    statusText.value = t('login.login_success')
    const mustChange = !!userStore.adminInfo?.must_change_password
    router.replace(mustChange ? '/profile/password' : '/')
  } catch (e) {
    console.error(e)
    statusText.value = t('login.oidc_failed')
    ElMessage.error(t('login.oidc_failed'))
    router.replace(buildAdminLoginPath(tenantCode || undefined))
  }
})
</script>

<style scoped>
.oidc-callback {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
}
</style>