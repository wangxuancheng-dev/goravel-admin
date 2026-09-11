<template>
  <div class="platform-layout">
    <header class="platform-header">
      <div class="brand">{{ $t('platform.title') }}</div>
      <div class="actions">
        <span class="admin-name">{{ adminName }}</span>
        <el-button link type="primary" @click="onLogout">{{ $t('header.logout') }}</el-button>
      </div>
    </header>
    <div class="platform-body">
      <aside class="platform-aside">
        <el-menu :default-active="active" router>
          <el-menu-item index="/platform/tenants">{{ $t('menu.tenant') }}</el-menu-item>
        </el-menu>
      </aside>
      <main class="platform-main">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { logoutPlatform } from '@/api/platform'
import { getPlatformAdmin } from '@/utils/platformRequest'

const route = useRoute()
const router = useRouter()
const active = computed(() => route.path)
const adminName = computed(() => {
  const admin = getPlatformAdmin()
  return admin?.name || admin?.username || ''
})

const onLogout = async () => {
  await logoutPlatform()
  router.replace('/platform/login')
}
</script>

<style scoped>
.platform-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f5f7fb;
}
.platform-header {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  background: #0f172a;
  color: #fff;
}
.brand {
  font-weight: 600;
  letter-spacing: 0.02em;
}
.actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.admin-name {
  opacity: 0.85;
  font-size: 13px;
}
.platform-body {
  flex: 1;
  display: flex;
  min-height: 0;
}
.platform-aside {
  width: 200px;
  background: #fff;
  border-right: 1px solid #e5e7eb;
}
.platform-main {
  flex: 1;
  padding: 16px;
  overflow: auto;
}
</style>
