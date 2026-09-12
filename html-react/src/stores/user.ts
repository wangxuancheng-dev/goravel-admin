import { create } from 'zustand'
import { compact, map } from 'lodash-es'
import { getInfo, logout as logoutApi } from '@/api/auth'
import Storage from '@/utils/storage'
import logger from '@/utils/logger'
import type { AdminInfo, FeatureConfig, MenuNode } from '@/types'

const defaultConfig: FeatureConfig = {
  showButtonsWithoutPermission: false,
  isDeveloperAdmin: false,
  pprofEnabled: false,
  pprofTokenRequired: false,
  aiEnabled: false,
  ordersEnabled: true,
  paymentsEnabled: true,
  paymentGateways: null,
  devToolsEnabled: false,
  codeGeneratorEnabled: false,
  searchEnabled: false,
  searchDriver: 'null',
  otelEnabled: false,
}

function toBasicAdminInfo(adminInfo: AdminInfo): AdminInfo {
  return {
    id: adminInfo.id || (adminInfo as { ID?: number }).ID!,
    username: adminInfo.username || (adminInfo as { Username?: string }).Username || '',
    nickname: adminInfo.nickname || (adminInfo as { Nickname?: string }).Nickname,
    avatar: adminInfo.avatar || (adminInfo as { Avatar?: string }).Avatar,
    email: adminInfo.email || (adminInfo as { Email?: string }).Email,
    phone: adminInfo.phone || (adminInfo as { Phone?: string }).Phone,
    department_id: adminInfo.department_id || (adminInfo as { DepartmentID?: number }).DepartmentID,
    department: adminInfo.department || (adminInfo as { Department?: unknown }).Department,
    must_change_password: !!(
      adminInfo.must_change_password ?? (adminInfo as { mustChangePassword?: boolean }).mustChangePassword
    ),
    roles: (adminInfo.roles || (adminInfo as { Roles?: AdminInfo['roles'] }).Roles || []).map((role) => ({
      id: role.id || (role as { ID?: number }).ID!,
      name: role.name || (role as { Name?: string }).Name || '',
      slug: role.slug || (role as { Slug?: string }).Slug,
    })),
  }
}

function detectSuperAdmin(adminInfo: AdminInfo): boolean {
  if (adminInfo.is_super_admin === true || adminInfo.isSuperAdmin === true) return true
  return (adminInfo.roles || []).some(
    (role) =>
      (role.slug || (role as { Slug?: string }).Slug) === 'super-admin' &&
      (role.status ?? (role as { Status?: number }).Status ?? 1) === 1,
  )
}

interface UserState {
  token: string
  adminInfo: AdminInfo | null
  permissions: string[]
  menus: MenuNode[]
  isSuperAdmin: boolean
  isFetchingUserInfo: boolean
  userInfoFetched: boolean
  config: FeatureConfig

  isLoggedIn: () => boolean
  hasPermission: (permission: string) => boolean
  shouldShowButton: (permission: string) => boolean

  setToken: (token: string) => void
  setAdminInfo: (adminInfo: AdminInfo) => void
  setPermissions: (permissions: AdminInfo['permissions']) => void
  setMenus: (menus: MenuNode[]) => void
  setConfig: (config?: Record<string, unknown>) => void
  fetchUserInfo: (force?: boolean) => Promise<unknown>
  logout: (skipApiCall?: boolean) => Promise<void>
}

export const useUserStore = create<UserState>((set, get) => {
  const cachedAdmin = Storage.getItem<AdminInfo>('adminInfo', null)

  return {
    token: Storage.getItem<string>('token', '') || '',
    adminInfo: cachedAdmin,
    permissions: [],
    menus: [],
    isSuperAdmin: false,
    isFetchingUserInfo: false,
    userInfoFetched: !!cachedAdmin,
    config: { ...defaultConfig },

    isLoggedIn: () => !!get().token,

    hasPermission: (permission) => {
      const state = get()
      if (state.isSuperAdmin) return true
      return state.permissions.includes(permission)
    },

    shouldShowButton: (permission) => {
      const state = get()
      if (state.isSuperAdmin) return true
      if (state.permissions.includes(permission)) return true
      return state.config.showButtonsWithoutPermission
    },

    setToken: (token) => {
      set({ token })
      Storage.setItem('token', token)
    },

    setAdminInfo: (adminInfo) => {
      const basicInfo = toBasicAdminInfo(adminInfo)
      const isSuperAdmin = detectSuperAdmin(adminInfo)
      set({ adminInfo: basicInfo, isSuperAdmin })

      const saved = Storage.setItem('adminInfo', basicInfo)
      if (!saved) {
        const minimalInfo = {
          id: basicInfo.id,
          username: basicInfo.username,
          nickname: basicInfo.nickname,
          avatar: basicInfo.avatar,
        }
        if (Storage.setItem('adminInfo', minimalInfo)) {
          set({ adminInfo: minimalInfo as AdminInfo })
        } else {
          logger.error('Failed to save adminInfo even with minimal data')
        }
      }
    },

    setPermissions: (permissions) => {
      if (Array.isArray(permissions) && permissions.length > 0) {
        if (typeof permissions[0] === 'object' && permissions[0] !== null) {
          set({
            permissions: compact(
              map(permissions, (perm) =>
                typeof perm === 'string' ? perm : perm.slug || perm.Slug || '',
              ),
            ),
          })
        } else {
          set({ permissions: permissions as string[] })
        }
      } else {
        set({ permissions: [] })
      }
    },

    setMenus: (menus) => set({ menus }),

    setConfig: (config) => {
      set({
        config: {
          showButtonsWithoutPermission: !!(
            config?.show_buttons_without_permission || config?.showButtonsWithoutPermission
          ),
          isDeveloperAdmin: !!(config?.is_developer_admin || config?.isDeveloperAdmin),
          pprofEnabled: !!(config?.pprof_enabled || config?.pprofEnabled),
          pprofTokenRequired: !!(config?.pprof_token_required || config?.pprofTokenRequired),
          aiEnabled: !!(config?.ai_enabled || config?.aiEnabled),
          ordersEnabled: (config?.orders_enabled ?? config?.ordersEnabled ?? true) as boolean,
          paymentsEnabled: (config?.payments_enabled ?? config?.paymentsEnabled ?? false) as boolean,
          paymentGateways: (Array.isArray(config?.payment_gateways)
            ? config.payment_gateways
            : Array.isArray(config?.paymentGateways)
              ? config.paymentGateways
              : null) as string[] | null,
          devToolsEnabled: !!(config?.dev_tools_enabled || config?.devToolsEnabled),
          codeGeneratorEnabled: !!(config?.code_generator_enabled || config?.codeGeneratorEnabled),
          searchEnabled: !!(config?.search_enabled || config?.searchEnabled),
          searchDriver: (config?.search_driver || config?.searchDriver || 'null') as string,
          otelEnabled: !!(config?.otel_enabled || config?.otelEnabled),
        },
      })
    },

    fetchUserInfo: async (force = false) => {
      const state = get()

      if (state.isFetchingUserInfo && !force) {
        while (get().isFetchingUserInfo) {
          await new Promise((r) => setTimeout(r, 50))
        }
        return
      }

      if (state.userInfoFetched && !force && state.adminInfo && state.menus.length > 0) {
        return
      }

      const oldMenus = state.menus.length > 0 ? [...state.menus] : []
      const oldAdminInfo = state.adminInfo
      const oldPermissions = [...state.permissions]

      try {
        set({ isFetchingUserInfo: true })

        if (force || !get().userInfoFetched) {
          set({ menus: [], adminInfo: null, permissions: [] })
          Storage.removeItem('adminInfo')
        }

        const res = await getInfo()
        if (res.data?.admin) {
          get().setAdminInfo(res.data.admin)
          get().setPermissions(res.data.admin.permissions || [])

          const newMenus = res.data.admin.menus || []
          if (newMenus.length > 0) {
            get().setMenus(newMenus)
          } else if (oldMenus.length > 0 && !force) {
            logger.warn('New menus is empty, keeping old menus')
            get().setMenus(oldMenus)
          } else {
            get().setMenus([])
          }

          set({
            isSuperAdmin: detectSuperAdmin(res.data.admin),
            userInfoFetched: true,
          })

          logger.debug('User permissions:', get().permissions)
          logger.debug('Is super admin:', get().isSuperAdmin)
          logger.debug('Menus count:', get().menus.length)
        } else if (oldMenus.length > 0 && !force) {
          logger.warn('No admin data in response, restoring old data')
          set({
            menus: oldMenus,
            adminInfo: oldAdminInfo,
            permissions: oldPermissions,
          })
        }

        if (res.data?.config) {
          get().setConfig(res.data.config as Record<string, unknown>)
        }

        return res
      } catch (error) {
        if (oldMenus.length > 0 && !force) {
          logger.warn('Failed to fetch user info, restoring old data:', error)
          set({
            menus: oldMenus,
            adminInfo: oldAdminInfo,
            permissions: oldPermissions,
          })
          return
        }
        set({ userInfoFetched: false })
        await get().logout(true)
        throw error
      } finally {
        set({ isFetchingUserInfo: false })
      }
    },

    logout: async (skipApiCall = false) => {
      try {
        if (get().token && !skipApiCall) {
          await logoutApi()
        }
      } catch (error) {
        logger.error('Logout error:', error)
      } finally {
        set({
          token: '',
          adminInfo: null,
          permissions: [],
          menus: [],
          isSuperAdmin: false,
          userInfoFetched: false,
          isFetchingUserInfo: false,
          config: { ...defaultConfig },
        })
        Storage.removeItem('token')
        Storage.removeItem('adminInfo')
      }
    },
  }
})
