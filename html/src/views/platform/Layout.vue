<template>
  <div class="platform-layout">
    <header class="platform-header">
      <div class="brand-wrap">
        <button type="button" class="header-icon-btn" aria-label="toggle sidebar" @click="appStore.toggleSidebar()">
          <el-icon><Expand v-if="collapsed" /><Fold v-else /></el-icon>
        </button>
        <div class="brand">{{ $t('platform.title') }}</div>
      </div>
      <div class="actions">
        <span class="admin-name">{{ adminName }}</span>
        <el-tag v-if="isViewer" size="small" type="info" effect="plain" class="role-tag">{{ $t('platform.role_viewer') }}</el-tag>
        <el-popover placement="bottom-end" trigger="click" :width="240">
          <template #reference>
            <button type="button" class="header-icon-btn" :title="$t('header.theme_color')">
              <el-icon><Setting /></el-icon>
            </button>
          </template>
          <div class="theme-settings">
            <div class="theme-settings-label">{{ $t('header.theme_color') }}</div>
            <div class="theme-color-swatches">
              <button
                v-for="item in THEME_COLORS"
                :key="item.key"
                type="button"
                class="theme-swatch"
                :class="{ active: appStore.themeColor === item.key }"
                :style="{ backgroundColor: item.color }"
                :title="item.key"
                @click="appStore.setThemeColor(item.key)"
              />
            </div>
          </div>
        </el-popover>
        <DarkModeSwitch class="platform-header-dark" />
        <LanguageSwitch class="platform-header-lang" />
        <TimezoneSwitch class="platform-header-tz" />
        <el-button link type="primary" @click="pwdVisible = true">{{ $t('platform.change_password') }}</el-button>
        <el-button link type="primary" @click="onLogout">{{ $t('header.logout') }}</el-button>
      </div>
    </header>
    <div class="platform-body">
      <aside class="platform-aside" :class="{ collapsed }">
        <el-menu :default-active="active" :collapse="collapsed" :collapse-transition="false" router>
          <el-menu-item index="/platform/overview">
            <el-icon><Odometer /></el-icon>
            <template #title>{{ $t('menu.platform_overview') }}</template>
          </el-menu-item>
          <el-menu-item index="/platform/tenants">
            <el-icon><OfficeBuilding /></el-icon>
            <template #title>{{ $t('menu.tenant') }}</template>
          </el-menu-item>
          <el-menu-item index="/platform/entitlements">
            <el-icon><Medal /></el-icon>
            <template #title>{{ $t('menu.entitlement') }}</template>
          </el-menu-item>
          <el-menu-item index="/platform/admins">
            <el-icon><User /></el-icon>
            <template #title>{{ $t('menu.platform_admin') }}</template>
          </el-menu-item>
          <el-menu-item index="/platform/settings">
            <el-icon><Tools /></el-icon>
            <template #title>{{ $t('menu.platform_settings') }}</template>
          </el-menu-item>
          <el-menu-item index="/platform/tenant-op-logs">
            <el-icon><Clock /></el-icon>
            <template #title>{{ $t('menu.tenant_op_log') }}</template>
          </el-menu-item>
          <el-menu-item index="/platform/alert-deliveries">
            <el-icon><Bell /></el-icon>
            <template #title>{{ $t('menu.platform_alert') }}</template>
          </el-menu-item>
          <el-menu-item index="/platform/system-logs">
            <el-icon><Document /></el-icon>
            <template #title>{{ $t('menu.system_log') }}</template>
          </el-menu-item>
          <el-menu-item index="/platform/login-logs">
            <el-icon><Key /></el-icon>
            <template #title>{{ $t('menu.login_log') }}</template>
          </el-menu-item>
          <el-menu-item index="/platform/operation-logs">
            <el-icon><Tickets /></el-icon>
            <template #title>{{ $t('menu.operation_log') }}</template>
          </el-menu-item>
        </el-menu>
      </aside>
      <main class="platform-main">
        <!-- The API converts times per X-Timezone: remount the page to refetch on a switch. -->
        <router-view :key="appStore.timezone" />
      </main>
    </div>

    <el-dialog v-model="pwdVisible" :title="$t('platform.change_password')" width="420px" destroy-on-close @closed="resetPwd">
      <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="100px">
        <el-form-item :label="$t('platform.old_password')" prop="old_password">
          <el-input v-model="pwdForm.old_password" type="password" show-password />
        </el-form-item>
        <el-form-item :label="$t('platform.new_password')" prop="new_password">
          <el-input
            v-model="pwdForm.new_password"
            type="password"
            show-password
            :placeholder="describePasswordPolicy(passwordPolicy, t)"
          />
        </el-form-item>
        <el-form-item :label="$t('platform.confirm_password')" prop="confirm_password">
          <el-input v-model="pwdForm.confirm_password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="pwdSaving" @click="submitPwd">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import {
  Bell,
  Clock,
  Document,
  Expand,
  Fold,
  Key,
  Medal,
  Odometer,
  OfficeBuilding,
  Setting,
  Tickets,
  Tools,
  User,
} from '@element-plus/icons-vue'
import { useAppStore, THEME_COLORS } from '@/store/app'
import { logoutPlatform, updatePlatformPassword } from '@/api/platform'
import { getPlatformAdmin } from '@/utils/platformRequest'
import DarkModeSwitch from '@/components/DarkModeSwitch.vue'
import LanguageSwitch from '@/components/LanguageSwitch.vue'
import TimezoneSwitch from '@/components/TimezoneSwitch.vue'
import { describePasswordPolicy, passwordPolicyRule } from '@/utils/passwordPolicy'
import { usePlatformPasswordPolicy } from '@/composables/usePlatformPasswordPolicy'

const { t } = useI18n()
const passwordPolicy = usePlatformPasswordPolicy()
const route = useRoute()
const router = useRouter()
const active = computed(() => route.path)
const appStore = useAppStore()
const collapsed = computed(() => appStore.sidebarCollapsed)
const platformAdmin = computed(() => getPlatformAdmin())
const adminName = computed(() => platformAdmin.value?.name || platformAdmin.value?.username || '')
const isViewer = computed(() => platformAdmin.value?.role === 'viewer')

const pwdVisible = ref(false)
const pwdSaving = ref(false)
const pwdFormRef = ref()
const pwdForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: '',
})
const pwdRules = {
  old_password: [{ required: true, message: () => t('platform.old_password'), trigger: 'blur' }],
  new_password: [
    { required: true, message: () => t('platform.new_password'), trigger: 'blur' },
    passwordPolicyRule(() => passwordPolicy.value, t),
  ],
  confirm_password: [
    { required: true, message: () => t('platform.confirm_password'), trigger: 'blur' },
    {
      validator: (_r, v, cb) => {
        if (v !== pwdForm.new_password) cb(new Error(t('platform.confirm_password')))
        else cb()
      },
      trigger: 'blur',
    },
  ],
}

const resetPwd = () => {
  pwdForm.old_password = ''
  pwdForm.new_password = ''
  pwdForm.confirm_password = ''
}

const submitPwd = async () => {
  await pwdFormRef.value?.validate()
  pwdSaving.value = true
  try {
    await updatePlatformPassword({ ...pwdForm })
    ElMessage.success(t('platform.password_updated'))
    pwdVisible.value = false
  } catch (e) {
    if (!e?.__handled) ElMessage.error(e?.message || t('common.operation_failed'))
  } finally {
    pwdSaving.value = false
  }
}

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
  background: var(--el-bg-color-page, #f5f7fb);
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
.brand-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
}
.header-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 36px;
  min-height: 36px;
  padding: 6px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: rgba(255, 255, 255, 0.85);
  font-size: 18px;
  cursor: pointer;
}
.header-icon-btn:hover {
  background-color: rgba(255, 255, 255, 0.1);
  color: #fff;
}
.theme-settings-label {
  font-size: 13px;
  color: var(--el-text-color-regular);
  margin-bottom: 10px;
}
.theme-color-swatches {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.theme-swatch {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  padding: 0;
  box-shadow: 0 0 0 1px var(--el-border-color);
  transition: transform 0.15s;
}
.theme-swatch:hover {
  transform: scale(1.1);
}
.theme-swatch.active {
  border-color: var(--el-bg-color, #fff);
  box-shadow: 0 0 0 2px var(--el-color-primary);
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
.role-tag {
  border-color: rgba(255, 255, 255, 0.35);
  color: #e2e8f0;
  background: transparent;
}
.platform-header-dark :deep(.dark-mode-switch) {
  color: rgba(255, 255, 255, 0.85);
  min-width: 36px;
  min-height: 36px;
  padding: 6px;
}
.platform-header-dark :deep(.dark-mode-switch:hover) {
  background-color: rgba(255, 255, 255, 0.1);
  color: #fff;
}
/* Timezone select on the always-dark platform header. */
.platform-header-tz {
  width: 200px;
}
.platform-header-tz :deep(.el-select__wrapper) {
  background: rgba(255, 255, 255, 0.08);
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.2) inset;
}
.platform-header-tz :deep(.el-select__wrapper.is-hovering:not(.is-focused)) {
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.4) inset;
}
/* Element Plus renders the selected label in __placeholder (without is-transparent). */
.platform-header-tz :deep(.el-select__placeholder:not(.is-transparent)),
.platform-header-tz :deep(.el-select__selected-item),
.platform-header-tz :deep(.el-select__input),
.platform-header-tz :deep(.el-select__caret) {
  color: rgba(255, 255, 255, 0.85);
}
@media (max-width: 768px) {
  .platform-header-tz {
    display: none;
  }
}
.platform-header-lang :deep(.language-switch) {
  color: rgba(255, 255, 255, 0.85);
  min-width: 36px;
  min-height: 36px;
  padding: 6px;
}
.platform-header-lang :deep(.language-switch:hover) {
  background-color: rgba(255, 255, 255, 0.1);
  color: #fff;
}
.platform-body {
  flex: 1;
  display: flex;
  min-height: 0;
}
.platform-aside {
  width: 200px;
  flex-shrink: 0;
  background: var(--el-bg-color, #fff);
  border-right: 1px solid var(--el-border-color-light, #e5e7eb);
  transition: width 0.2s;
}
.platform-aside.collapsed {
  width: 64px;
}
.platform-aside :deep(.el-menu) {
  border-right: none;
  background: transparent;
}
.platform-main {
  flex: 1;
  padding: 16px;
  overflow: auto;
}
</style>
