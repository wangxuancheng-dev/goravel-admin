<template>
  <el-header class="header">
    <div class="header-left">
      <el-button
        v-if="isMobile"
        type="text"
        class="collapse-btn mobile-menu-btn"
        @click="$emit('open-drawer')"
      >
        <el-icon><Menu /></el-icon>
      </el-button>
      <el-button
        v-else-if="appStore.menuMode === 'sidebar'"
        type="text"
        class="collapse-btn"
        @click="$emit('toggle-sidebar')"
      >
        <el-icon><Fold v-if="!appStore.sidebarCollapsed" /><Expand v-else /></el-icon>
      </el-button>
      <el-tag
        v-if="tenantLabel"
        class="header-tenant-tag"
        type="primary"
        effect="plain"
        :title="`${$t('header.current_tenant')}: ${tenantLabel}`"
      >
        {{ $t('header.current_tenant') }}: {{ tenantLabel }}
      </el-tag>
      <BreadcrumbView :class="{ 'mobile-hidden': isXs }" />
    </div>
    <div class="header-right">
      <MenuSearch v-if="!isMobile" :menus="menuTree" />
      <el-button
        v-if="!isMobile"
        type="text"
        class="header-btn"
        @click="appStore.toggleFullscreen"
        :title="$t('header.fullscreen')"
      >
        <el-icon class="header-icon-fixed">
          <FullScreen v-if="!appStore.isFullscreen" />
          <Aim v-else />
        </el-icon>
      </el-button>
      <el-dropdown
        v-if="!isMobile"
        @command="handleLayoutSizeChange"
        class="layout-size-dropdown"
        popper-class="layout-size-popper"
      >
        <el-button type="text" class="header-btn" :title="$t('header.layout_size')">
          <el-icon class="header-icon-fixed"><Operation /></el-icon>
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="large" :class="{ 'is-active': appStore.layoutSize === 'large' }">
              <span class="layout-size-option">
                <span class="layout-size-option-left">
                  <span class="layout-density layout-density-large" aria-hidden="true">
                    <i></i><i></i><i></i>
                  </span>
                  <span>{{ $t('header.layout_size_large') }}</span>
                </span>
                <el-icon v-if="appStore.layoutSize === 'large'" class="layout-size-option-check"><Check /></el-icon>
              </span>
            </el-dropdown-item>
            <el-dropdown-item command="default" :class="{ 'is-active': appStore.layoutSize === 'default' }">
              <span class="layout-size-option">
                <span class="layout-size-option-left">
                  <span class="layout-density layout-density-default" aria-hidden="true">
                    <i></i><i></i><i></i>
                  </span>
                  <span>{{ $t('header.layout_size_default') }}</span>
                </span>
                <el-icon v-if="appStore.layoutSize === 'default'" class="layout-size-option-check"><Check /></el-icon>
              </span>
            </el-dropdown-item>
            <el-dropdown-item command="small" :class="{ 'is-active': appStore.layoutSize === 'small' }">
              <span class="layout-size-option">
                <span class="layout-size-option-left">
                  <span class="layout-density layout-density-small" aria-hidden="true">
                    <i></i><i></i><i></i>
                  </span>
                  <span>{{ $t('header.layout_size_small') }}</span>
                </span>
                <el-icon v-if="appStore.layoutSize === 'small'" class="layout-size-option-check"><Check /></el-icon>
              </span>
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-popover
        v-if="!isMobile"
        placement="bottom-end"
        :width="300"
        trigger="click"
        popper-class="settings-popover"
      >
        <template #reference>
          <el-button
            type="text"
            class="header-btn"
            :title="$t('header.settings')"
          >
            <el-icon class="header-icon-fixed"><Setting /></el-icon>
          </el-button>
        </template>
        <div class="settings-panel">
          <div class="settings-title">{{ $t('header.settings') }}</div>
          <div class="settings-item settings-item-menu-mode">
            <span class="settings-label">{{ $t('header.menu_mode') }}</span>
            <div class="menu-mode-toggle" role="tablist" :aria-label="$t('header.menu_mode')">
              <button
                type="button"
                class="menu-mode-btn"
                :class="{ active: appStore.menuMode === 'sidebar' }"
                @click="appStore.setMenuMode('sidebar')"
              >
                <el-icon><Fold /></el-icon>
                <span>{{ $t('header.menu_mode_sidebar') }}</span>
              </button>
              <button
                type="button"
                class="menu-mode-btn"
                :class="{ active: appStore.menuMode === 'top' }"
                @click="appStore.setMenuMode('top')"
              >
                <el-icon><Menu /></el-icon>
                <span>{{ $t('header.menu_mode_top') }}</span>
              </button>
            </div>
          </div>
          <div class="settings-item">
            <span class="settings-label">{{ $t('header.watermark') }}</span>
            <el-switch v-model="appStore.watermarkEnabled" @change="appStore.setWatermarkEnabled(appStore.watermarkEnabled)" />
          </div>
          <div class="settings-item settings-item-theme">
            <span class="settings-label">{{ $t('header.theme_color') }}</span>
            <div class="theme-color-swatches">
              <button
                v-for="theme in themeColorOptions"
                :key="theme.key"
                type="button"
                class="theme-swatch"
                :class="{ active: appStore.themeColor === theme.key }"
                :style="{ backgroundColor: theme.color }"
                :title="theme.key"
                @click="appStore.setThemeColor(theme.key)"
              />
            </div>
          </div>
        </div>
      </el-popover>
      <NotificationBell />
      <DarkModeSwitch />
      <LanguageSwitch :class="{ 'mobile-hidden': isXs }" />
      <el-button
        type="text"
        class="header-btn"
        :class="{ 'mobile-hidden': isXs }"
        :title="$t('header.lock_screen')"
        @click="$emit('lock-screen')"
      >
        <el-icon class="header-icon-fixed"><Lock /></el-icon>
      </el-button>
      <TimezoneSwitch :class="{ 'mobile-hidden': isMobile }" />
      <el-dropdown
        @command="handleCommand"
        class="user-dropdown"
        popper-class="user-account-popper"
        placement="bottom-end"
        :teleported="true"
      >
        <span class="user-info">
          <el-avatar
            v-if="userStore.adminInfo?.avatar"
            :size="isMobile ? 28 : 32"
            :src="userStore.adminInfo.avatar"
            class="user-avatar"
          >
            <el-icon><User /></el-icon>
          </el-avatar>
          <el-icon v-else class="user-icon"><User /></el-icon>
          <span class="user-name" :class="{ 'mobile-hidden': isXs }">
            {{ userStore.adminInfo?.nickname || userStore.adminInfo?.username }}
          </span>
          <el-icon class="el-icon--right" :class="{ 'mobile-hidden': isMobile }">
            <ArrowDown />
          </el-icon>
        </span>
        <template #dropdown>
          <div class="user-account-panel">
            <div class="user-account-header">
              <el-avatar
                v-if="userStore.adminInfo?.avatar"
                :size="48"
                :src="userStore.adminInfo.avatar"
                class="user-account-avatar"
              >
                <el-icon><User /></el-icon>
              </el-avatar>
              <el-avatar v-else :size="48" class="user-account-avatar user-account-avatar--placeholder">
                <el-icon><User /></el-icon>
              </el-avatar>
              <div class="user-account-meta">
                <div class="user-account-name-row">
                  <span class="user-account-name">{{ userAccountDisplayName }}</span>
                  <el-tag
                    v-if="userStore.isSuperAdmin"
                    size="small"
                    type="warning"
                    effect="plain"
                    class="user-account-badge"
                  >
                    {{ $t('header.super_admin') }}
                  </el-tag>
                </div>
                <div v-if="userAccountSubtitle" class="user-account-sub">{{ userAccountSubtitle }}</div>
                <div v-if="userAccountDepartment" class="user-account-dept">
                  <el-icon class="user-account-dept-icon"><OfficeBuilding /></el-icon>
                  <span>{{ userAccountDepartment }}</span>
                </div>
                <div
                  v-if="userAccountRolePreview.visible.length || userAccountShowAllPermissionsHint"
                  class="user-account-roles"
                >
                  <span class="user-account-roles-label">{{ $t('header.account_roles') }}</span>
                  <div class="user-account-roles-tags">
                    <template v-if="userAccountRolePreview.visible.length">
                      <el-tag
                        v-for="(name, idx) in userAccountRolePreview.visible"
                        :key="`role-${idx}-${name}`"
                        size="small"
                        effect="plain"
                        type="info"
                        class="user-account-role-tag"
                      >
                        {{ name }}
                      </el-tag>
                      <el-tag
                        v-if="userAccountRolePreview.more > 0"
                        size="small"
                        effect="plain"
                        type="info"
                        class="user-account-role-tag user-account-role-tag--more"
                      >
                        +{{ userAccountRolePreview.more }}
                      </el-tag>
                    </template>
                    <el-tag
                      v-else-if="userAccountShowAllPermissionsHint"
                      size="small"
                      effect="plain"
                      type="success"
                      class="user-account-role-tag user-account-role-tag--all"
                    >
                      {{ $t('header.all_permissions_hint') }}
                    </el-tag>
                  </div>
                </div>
              </div>
            </div>
            <el-dropdown-menu class="user-account-menu">
              <el-dropdown-item command="profile" class="user-account-item">
                <span class="user-account-item-inner">
                  <span class="user-account-item-left">
                    <el-icon class="user-account-item-icon"><User /></el-icon>
                    <span class="user-account-item-text">
                      <span class="user-account-item-title">{{ $t('header.profile') }}</span>
                      <span class="user-account-item-desc">{{ $t('header.profile_desc') }}</span>
                    </span>
                  </span>
                  <el-icon class="user-account-item-chevron"><ArrowRight /></el-icon>
                </span>
              </el-dropdown-item>
              <el-dropdown-item command="logout" class="user-account-item user-account-item--logout">
                <span class="user-account-item-inner">
                  <span class="user-account-item-left">
                    <el-icon class="user-account-item-icon user-account-item-icon--danger"><SwitchButton /></el-icon>
                    <span class="user-account-item-text">
                      <span class="user-account-item-title">{{ $t('header.logout') }}</span>
                      <span class="user-account-item-desc user-account-item-desc--danger">{{ $t('header.logout_desc') }}</span>
                    </span>
                  </span>
                </span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </div>
        </template>
      </el-dropdown>
    </div>
  </el-header>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessageBox } from 'element-plus'
import {
  Fold,
  Expand,
  Setting,
  User,
  ArrowDown,
  ArrowRight,
  FullScreen,
  Aim,
  Menu,
  Operation,
  OfficeBuilding,
  SwitchButton,
  Check,
  Lock
} from '@element-plus/icons-vue'
import { useUserStore } from '@/store/user'
import { useTabsStore } from '@/store/tabs'
import { useAppStore, THEME_COLORS } from '@/store/app'
import LanguageSwitch from '@/components/LanguageSwitch.vue'
import TimezoneSwitch from '@/components/TimezoneSwitch.vue'
import NotificationBell from '@/components/NotificationBell.vue'
import DarkModeSwitch from '@/components/DarkModeSwitch.vue'
import BreadcrumbView from '@/components/BreadcrumbView.vue'
import MenuSearch from '@/components/MenuSearch.vue'
import { useLayoutAccount } from '@/composables/useLayoutAccount'
import { getTenantCode, resolveTenantCodeFromLocation } from '@/utils/tenant'

const props = defineProps({
  isMobile: { type: Boolean, default: false },
  isXs: { type: Boolean, default: false },
  menuTree: { type: Array, default: () => [] }
})

defineEmits(['toggle-sidebar', 'open-drawer', 'lock-screen'])

const router = useRouter()
const { t } = useI18n()
const userStore = useUserStore()
const tabsStore = useTabsStore()
const appStore = useAppStore()
const themeColorOptions = THEME_COLORS

const {
  userAccountDisplayName,
  userAccountSubtitle,
  userAccountDepartment,
  userAccountRolePreview,
  userAccountShowAllPermissionsHint
} = useLayoutAccount()

const tenantLabel = computed(() => {
  const code = String(
    userStore.tenant?.code || getTenantCode() || resolveTenantCodeFromLocation() || '',
  ).trim()
  const name = String(userStore.tenant?.name || '').trim()
  if (!code && !name) return ''
  if (name && code && name.toLowerCase() !== code.toLowerCase()) {
    return props.isXs ? code : `${name} (${code})`
  }
  return name || code
})

const handleLayoutSizeChange = (size) => {
  appStore.setLayoutSize(size)
}

const handleCommand = async (command) => {
  if (command === 'profile') {
    router.push('/profile')
  } else if (command === 'logout') {
    try {
      await ElMessageBox.confirm(t('header.logout_confirm'), t('common.confirm'), {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      })
      await userStore.logout()
      tabsStore.removeAllTabs()
      router.push('/login')
    } catch {
      // cancelled
    }
  }
}
</script>
