<template>
  <el-container class="layout-container" :class="[`layout-${appStore.layoutSize}`, { 'layout-top-menu': appStore.menuMode === 'top' && !isMobile }]">
    <LayoutSidebar
      v-model:drawer-visible="drawerVisible"
      :is-mobile="isMobile"
      :menu-mode="appStore.menuMode"
      :sidebar-collapsed="appStore.sidebarCollapsed"
      :sidebar-effective-collapsed="sidebarEffectiveCollapsed"
      :active-menu="activeMenu"
      :menu-tree="menuTree"
      :system-title="systemTitle"
      :website-logo-url="websiteLogoUrl"
      @drawer-close="drawerVisible = false"
      @menu-select="handleMenuSelect"
    />

    <el-container direction="vertical">
      <LayoutHeader
        :is-mobile="isMobile"
        :is-xs="isXs"
        :menu-tree="menuTree"
        @toggle-sidebar="handleToggleSidebar"
        @open-drawer="drawerVisible = true"
        @lock-screen="handleLockScreen"
      />

      <ImpersonationBanner />

      <LayoutTopMenu
        :is-mobile="isMobile"
        :menu-mode="appStore.menuMode"
        :active-menu="activeMenu"
        :menu-tree="menuTree"
        @menu-select="handleMenuSelect"
      />

      <div class="tabs-wrapper" :class="{ 'mobile-hidden': isMobile }">
        <TabsView />
      </div>

      <el-main
        ref="mainContentRef"
        class="main-content"
        :class="{
          'main-content-iframe': isIframePage,
          'main-content--sidebar-narrowing': sidebarNarrowingLock
        }"
        :style="mainContentInlineStyle"
      >
        <el-watermark
          v-if="appStore.watermarkEnabled && !sidebarNarrowingLock"
          :content="watermarkText"
          :font="watermarkFont"
          :width="120"
          :height="48"
          :z-index="9"
          :gap="[80, 80]"
          :rotate="-22"
          class="main-watermark"
        >
          <div class="main-content-inner">
            <router-view v-slot="{ Component, route: routeItem }">
              <transition name="fade-transform" mode="out-in">
                <keep-alive>
                  <component
                    :is="Component"
                    :key="`${routeItem.path}-${tabsStore.getRefreshKey(routeItem.path)}`"
                  />
                </keep-alive>
              </transition>
            </router-view>
          </div>
        </el-watermark>
        <div v-else class="main-content-inner">
          <router-view v-slot="{ Component, route: routeItem }">
            <transition name="fade-transform" mode="out-in">
              <keep-alive>
                <component
                  :is="Component"
                  :key="`${routeItem.path}-${tabsStore.getRefreshKey(routeItem.path)}`"
                />
              </keep-alive>
            </transition>
          </router-view>
        </div>
      </el-main>
    </el-container>

    <LayoutLockScreen
      v-model:lock-dialog-visible="lockDialogVisible"
      v-model:pending-lock-password="pendingLockPassword"
      v-model:unlock-password="unlockPassword"
      :is-screen-locked="isScreenLocked"
      :lock-dialog-error="lockDialogError"
      :unlock-error="unlockError"
      :lock-input-name="lockInputName"
      :lock-dialog-input-name="lockDialogInputName"
      @confirm-lock="confirmLockScreen"
      @unlock-input="handleUnlockInput"
      @unlock="handleUnlockScreen"
      @go-login="goToLogin"
    />
  </el-container>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore } from '@/store/user'
import { useTabsStore } from '@/store/tabs'
import { useAppStore } from '@/store/app'
import TabsView from '@/components/TabsView.vue'
import LayoutSidebar from './components/LayoutSidebar.vue'
import LayoutHeader from './components/LayoutHeader.vue'
import LayoutTopMenu from './components/LayoutTopMenu.vue'
import LayoutLockScreen from './components/LayoutLockScreen.vue'
import ImpersonationBanner from './components/ImpersonationBanner.vue'
import { useResponsive } from '@/composables/useResponsive'
import { useLayoutWebsite } from '@/composables/useLayoutWebsite'
import { useLayoutMenu } from '@/composables/useLayoutMenu'
import { useLayoutLockScreen } from '@/composables/useLayoutLockScreen'
import { useLayoutSidebar } from '@/composables/useLayoutSidebar'
import { useLayoutLifecycle } from '@/composables/useLayoutLifecycle'

const { isMobile, isXs } = useResponsive()
const route = useRoute()
const userStore = useUserStore()
const tabsStore = useTabsStore()
const appStore = useAppStore()

const drawerVisible = ref(false)
const mainContentRef = ref(null)

const { systemTitle, websiteLogoUrl, loadWebsiteTitle } = useLayoutWebsite()

const { activeMenu, menuTree, handleMenuSelect } = useLayoutMenu({
  isMobile,
  onMobileMenuSelect: () => {
    drawerVisible.value = false
  }
})

const {
  sidebarNarrowingLock,
  sidebarEffectiveCollapsed,
  mainContentInlineStyle,
  handleToggleSidebar,
  cleanupSidebar
} = useLayoutSidebar(mainContentRef)

const {
  isScreenLocked,
  lockDialogVisible,
  pendingLockPassword,
  lockDialogError,
  unlockPassword,
  unlockError,
  lockInputName,
  lockDialogInputName,
  handleLockScreen,
  confirmLockScreen,
  handleUnlockInput,
  handleUnlockScreen,
  goToLogin
} = useLayoutLockScreen()

const watermarkText = computed(() => {
  const name = userStore.adminInfo?.nickname || userStore.adminInfo?.username || 'Admin'
  return name
})

const watermarkFont = computed(() => ({
  fontSize: 14,
  color: appStore.darkMode ? 'rgba(255, 255, 255, 0.12)' : 'rgba(0, 0, 0, 0.12)',
  fontWeight: 'normal'
}))

const isIframePage = computed(() => route.path === '/iframe')

useLayoutLifecycle({
  loadWebsiteTitle,
  cleanupSidebar
})
</script>

<style src="./styles/main-layout.css"></style>
<style src="./styles/layout-popper.css"></style>
