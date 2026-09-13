import { Suspense, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import {
  Navigate,
  Outlet,
  RouterProvider,
  createBrowserRouter,
  useLocation,
  useNavigate,
} from 'react-router-dom'
import { Spin } from 'antd'
import { useUserStore } from '@/stores/user'
import { setNavigator } from '@/utils/navigation'
import { convertMenusToRoutes } from './dynamicRoutes'
import { RouteErrorFallback } from './RouteErrorFallback'
import logger from '@/utils/logger'

// Shell routes are eager: Vite HMR often breaks lazy MainLayout with
// "Failed to fetch dynamically imported module ...?t=...".
import LoginPage from '../pages/Login'
import MainLayout from '../layouts/MainLayout'
import DashboardPage from '../pages/Dashboard'
import ProfilePage from '../pages/profile/Profile'
import NotFoundPage from '../pages/NotFound'
import PlatformLoginPage from '../pages/platform/Login'
import PlatformLayout from '../pages/platform/Layout'
import PlatformTenantListPage from '../pages/platform/TenantList'
import PlatformTenantOpLogListPage from '../pages/platform/TenantOpLogList'
import { getPlatformToken } from '@/utils/platformRequest'

function PageFallback({ fullscreen = false }: { fullscreen?: boolean }) {
  return (
    <div className={`page-fallback${fullscreen ? ' page-fallback--fullscreen' : ''}`}>
      <Spin size="large" />
    </div>
  )
}

function withSuspense(element: ReactNode) {
  return <Suspense fallback={<PageFallback />}>{element}</Suspense>
}

/** Registers navigate for axios 401 redirects. */
function NavigatorBridge() {
  const navigate = useNavigate()
  useEffect(() => {
    setNavigator((to, options) => {
      navigate(to, { replace: options?.replace })
    })
  }, [navigate])
  return null
}

function AuthGuard() {
  const location = useLocation()
  const token = useUserStore((s) => s.token)
  const mustChange = useUserStore((s) => !!s.adminInfo?.must_change_password)

  if (!token) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  if (mustChange && location.pathname !== '/profile') {
    return <Navigate to="/profile?tab=password" replace />
  }

  return (
    <>
      <NavigatorBridge />
      <Outlet />
    </>
  )
}

function GuestGuard() {
  const token = useUserStore((s) => s.token)
  if (token) return <Navigate to="/dashboard" replace />
  return (
    <>
      <NavigatorBridge />
      <Outlet />
    </>
  )
}

function menusSignature(menus: ReturnType<typeof useUserStore.getState>['menus']): string {
  if (!menus?.length) return ''
  try {
    return JSON.stringify(
      menus.map((m) => ({
        id: m.id ?? m.ID,
        path: m.Path ?? m.path,
        component: m.Component ?? m.component,
        type: m.Type ?? m.type,
        status: m.Status ?? m.status,
        link: m.LinkType ?? m.link_type,
        children: m.children ?? m.Children,
      })),
    )
  } catch {
    return String(menus.length)
  }
}

function PlatformAuthGuard() {
  const location = useLocation()
  const token = getPlatformToken()
  if (!token) {
    return <Navigate to="/platform/login" replace state={{ from: location.pathname }} />
  }
  return (
    <>
      <NavigatorBridge />
      <Outlet />
    </>
  )
}

function PlatformGuestGuard() {
  const token = getPlatformToken()
  if (token) return <Navigate to="/platform/tenants" replace />
  return (
    <>
      <NavigatorBridge />
      <Outlet />
    </>
  )
}

function buildRouter(dynamicChildren: ReturnType<typeof convertMenusToRoutes>) {
  return createBrowserRouter([
    {
      path: '/login',
      element: <GuestGuard />,
      errorElement: <RouteErrorFallback />,
      children: [
        {
          index: true,
          element: <LoginPage />,
          handle: { requiresAuth: false },
        },
      ],
    },
    {
      path: '/platform/login',
      element: <PlatformGuestGuard />,
      errorElement: <RouteErrorFallback />,
      children: [
        {
          index: true,
          element: <PlatformLoginPage />,
          handle: { requiresAuth: false, platform: true },
        },
      ],
    },
    {
      path: '/platform',
      element: <PlatformAuthGuard />,
      errorElement: <RouteErrorFallback />,
      children: [
        {
          element: <PlatformLayout />,
          errorElement: <RouteErrorFallback />,
          children: [
            { index: true, element: <Navigate to="/platform/tenants" replace /> },
            {
              path: 'tenants',
              element: <PlatformTenantListPage />,
              handle: { titleKey: 'menu.tenant', platform: true },
            },
            {
              path: 'tenant-op-logs',
              element: <PlatformTenantOpLogListPage />,
              handle: { titleKey: 'menu.tenant_op_log', platform: true },
            },
          ],
        },
      ],
    },
    {
      path: '/',
      element: <AuthGuard />,
      errorElement: <RouteErrorFallback />,
      children: [
        {
          element: <MainLayout />,
          errorElement: <RouteErrorFallback />,
          children: [
            { index: true, element: <Navigate to="/dashboard" replace /> },
            {
              path: 'dashboard',
              element: <DashboardPage />,
              handle: { titleKey: 'menu.dashboard' },
            },
            {
              path: 'profile',
              element: <ProfilePage />,
              handle: { titleKey: 'menu.profile' },
            },
            ...dynamicChildren.map((route) => ({
              ...route,
              element: withSuspense(route.element),
              errorElement: <RouteErrorFallback />,
            })),
            {
              path: '*',
              element: <NotFoundPage />,
              handle: { titleKey: 'notFound.title' },
            },
          ],
        },
      ],
    },
  ])
}

/**
 * Bootstrap menus before mounting the data router, so hard-refresh never hits
 * catch-all with empty dynamic routes, and AuthGuard is not remounted mid-fetch.
 */
export function AppRouter() {
  const token = useUserStore((s) => s.token)
  const menus = useUserStore((s) => s.menus)
  const fetchUserInfo = useUserStore((s) => s.fetchUserInfo)
  const [bootstrapped, setBootstrapped] = useState(() => !token)
  const routerRef = useRef<ReturnType<typeof createBrowserRouter> | null>(null)
  const menusKeyRef = useRef<string>('')

  useEffect(() => {
    let cancelled = false

    async function bootstrap() {
      if (!token) {
        if (!cancelled) setBootstrapped(true)
        return
      }

      if (!cancelled) setBootstrapped(false)

      try {
        const state = useUserStore.getState()
        const menusEmpty = !state.menus || state.menus.length === 0
        if (!state.userInfoFetched || menusEmpty) {
          await fetchUserInfo(!state.userInfoFetched)
        }
      } catch (error) {
        logger.error('Auth bootstrap failed:', error)
      } finally {
        if (!cancelled) setBootstrapped(true)
      }
    }

    void bootstrap()
    return () => {
      cancelled = true
    }
  }, [token, fetchUserInfo])

  const menuKey = useMemo(() => menusSignature(menus), [menus])
  const dynamicChildren = useMemo(() => convertMenusToRoutes(menus), [menuKey, menus])

  // Keep one router instance unless menu tree actually changed (avoids remount storms).
  if (!routerRef.current || menusKeyRef.current !== menuKey) {
    menusKeyRef.current = menuKey
    routerRef.current = buildRouter(dynamicChildren)
  }
  const router = routerRef.current

  if (!bootstrapped) {
    return <PageFallback fullscreen />
  }

  return <RouterProvider router={router} />
}
