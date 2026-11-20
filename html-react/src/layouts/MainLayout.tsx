import { Layout, theme, Watermark } from 'antd'
import { Outlet, useLocation, useMatches } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import LayoutHeader from './components/LayoutHeader'
import LayoutSidebar from './components/LayoutSidebar'
import LayoutTopMenu from './components/LayoutTopMenu'
import LayoutLockScreen from './components/LayoutLockScreen'
import TabsView from './components/TabsView'
import ImpersonationBanner from './components/ImpersonationBanner'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import { useTabsStore } from '@/stores/tabs'
import { useLayoutLockScreen } from '@/hooks/useLayoutLockScreen'
import { useResponsive } from '@/hooks/useResponsive'
import type { AppRouteMeta } from '@/router/dynamicRoutes'
import { resolveMenuTitle } from '@/utils/menuTitle'
import './MainLayout.scss'

const { Content } = Layout

export default function MainLayout() {
  const { t } = useTranslation()
  const location = useLocation()
  const matches = useMatches()
  const collapsed = useAppStore((s) => s.sidebarCollapsed)
  const darkMode = useAppStore((s) => s.darkMode)
  const menuMode = useAppStore((s) => s.menuMode)
  const watermarkEnabled = useAppStore((s) => s.watermarkEnabled)
  const adminInfo = useUserStore((s) => s.adminInfo)
  const addTab = useTabsStore((s) => s.addTab)
  const { token } = theme.useToken()
  const lockScreen = useLayoutLockScreen()
  const { isMobile, isXs } = useResponsive()
  const [drawerOpen, setDrawerOpen] = useState(false)

  const isTopMenu = menuMode === 'top' && !isMobile
  const showFixedSidebar = !isMobile && menuMode === 'sidebar'
  const bodyMarginLeft = showFixedSidebar ? (collapsed ? 64 : 220) : 0

  useEffect(() => {
    const match = [...matches].reverse().find((m) => (m.handle as AppRouteMeta | undefined)?.titleKey)
    const handle = match?.handle as AppRouteMeta | undefined
    const titleKey = handle?.titleKey
    const title = resolveMenuTitle(t, {
      titleKey,
      slug: handle?.menuSlug,
      fallback: location.pathname,
    })
    addTab({
      path: location.pathname,
      title,
      titleKey,
      name: String(match?.id || location.pathname),
    })
  }, [location.pathname, matches, addTab, t])

  useEffect(() => {
    setDrawerOpen(false)
  }, [location.pathname])

  useEffect(() => {
    if (!isMobile) setDrawerOpen(false)
  }, [isMobile])

  useEffect(() => {
    const handleFullscreenChange = () => {
      useAppStore.setState({ isFullscreen: !!document.fullscreenElement })
    }
    document.addEventListener('fullscreenchange', handleFullscreenChange)
    return () => document.removeEventListener('fullscreenchange', handleFullscreenChange)
  }, [])

  const watermarkText = adminInfo?.nickname || adminInfo?.username || 'Admin'

  return (
    <Layout
      className={`main-layout${darkMode ? ' main-layout--dark' : ''}${isMobile ? ' main-layout--mobile' : ''}${isXs ? ' main-layout--xs' : ''}`}
    >
      <LayoutSidebar
        isMobile={isMobile}
        drawerOpen={drawerOpen}
        onDrawerClose={() => setDrawerOpen(false)}
      />
      <Layout className="main-layout__body" style={{ marginLeft: bodyMarginLeft }}>
        <LayoutHeader
          onLockScreen={lockScreen.handleLockScreen}
          isMobile={isMobile}
          isXs={isXs}
          onOpenDrawer={() => setDrawerOpen(true)}
        />
        <ImpersonationBanner />
        {isTopMenu ? <LayoutTopMenu /> : null}
        {!isMobile ? <TabsView /> : null}
        {/*
          Watermark must NOT wrap page content: its root uses overflow:hidden,
          which breaks Ant Design Table sticky fixed columns.
        */}
        <div className="main-layout__main">
          {watermarkEnabled ? (
            <div className="main-layout__watermark-overlay" aria-hidden>
              <Watermark
                className="main-layout__watermark"
                content={watermarkText}
                font={{
                  fontSize: 14,
                  color: darkMode ? 'rgba(255, 255, 255, 0.12)' : 'rgba(0, 0, 0, 0.12)',
                }}
                gap={[80, 80]}
                rotate={-22}
              >
                <div className="main-layout__watermark-fill" />
              </Watermark>
            </div>
          ) : null}
          <Content className="main-layout__content" style={{ background: token.colorBgLayout }}>
            <div className="main-layout__page">
              <Outlet />
            </div>
          </Content>
        </div>
      </Layout>
      <LayoutLockScreen lockScreen={lockScreen} />
    </Layout>
  )
}
