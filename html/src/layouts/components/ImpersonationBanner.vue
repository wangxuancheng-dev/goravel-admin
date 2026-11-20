<template>
  <el-alert v-if="impersonation" type="warning" show-icon :closable="false" class="impersonation-banner">
    <template #title>
      <div class="impersonation-banner__body">
        <span>
          {{
            $t('impersonate.banner', {
              operator: impersonation.platform_username || `#${impersonation.platform_admin_id}`,
              username: userStore.adminInfo?.username || '',
            })
          }}
        </span>
        <strong v-if="expiresAt">
          {{ expired ? $t('impersonate.expired') : $t('impersonate.remaining', { time: remainingText }) }}
        </strong>
        <el-button size="small" type="danger" plain @click="exit">{{ $t('impersonate.exit') }}</el-button>
      </div>
    </template>
  </el-alert>
</template>

<script setup>
// Always-visible notice while a platform admin is signed in as this tenant admin.
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/store/user'
import { useTabsStore } from '@/store/tabs'
import { resetDynamicRoutes } from '@/router'
import { buildAdminLoginPath } from '@/utils/tenant'

const router = useRouter()
const userStore = useUserStore()
const tabsStore = useTabsStore()
const now = ref(Date.now())
let timer = null

const impersonation = computed(() => userStore.impersonation)
const expiresAt = computed(() =>
  impersonation.value?.expires_at ? new Date(impersonation.value.expires_at).getTime() : 0,
)
const remaining = computed(() => (expiresAt.value ? expiresAt.value - now.value : 0))
const expired = computed(() => !!expiresAt.value && remaining.value <= 0)
const remainingText = computed(() => {
  const total = Math.max(0, Math.floor(remaining.value / 1000))
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`
})

const stopTimer = () => {
  if (timer) {
    window.clearInterval(timer)
    timer = null
  }
}

watch(
  impersonation,
  (val) => {
    stopTimer()
    if (val) {
      now.value = Date.now()
      timer = window.setInterval(() => {
        now.value = Date.now()
      }, 1000)
    }
  },
  { immediate: true },
)

onBeforeUnmount(stopTimer)

const exit = async () => {
  const loginPath = buildAdminLoginPath()
  tabsStore.removeAllTabs()
  await userStore.logout()
  resetDynamicRoutes()
  // The console opened this tab; close it when the browser allows, else go to login.
  window.close()
  router.replace(loginPath)
}
</script>

<style scoped>
.impersonation-banner {
  border-radius: 0;
}
.impersonation-banner__body {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
</style>
