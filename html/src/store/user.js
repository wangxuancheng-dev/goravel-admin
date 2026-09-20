import { defineStore } from 'pinia'
import { compact, map } from 'lodash-es'
import { getInfo, logout } from '../api/auth'
import Storage from '../utils/storage'
import logger from '../utils/logger'
import { clearTenantCode, getTenantCode, resolveEffectiveTenantCode, resolveTenantCodeFromLocation, setTenantCode } from '../utils/tenant'

export const useUserStore = defineStore('user', {
  state: () => {
    const adminInfo = Storage.getItem('adminInfo', null)
    return {
      token: Storage.getItem('token', ''),
      adminInfo: adminInfo,
      tenant: Storage.getItem('tenantInfo', null),
      permissions: [],
      menus: [], // 菜单不缓存，每次刷新都从服务器重新获取
      isSuperAdmin: false, // 是否是超级管理员
      isFetchingUserInfo: false, // 是否正在获取用户信息，避免重复请求
      // 如果从 localStorage 恢复了用户信息，认为已获取过（但菜单仍需从服务器获取）
      userInfoFetched: !!adminInfo, // 用户信息是否已获取过（用于判断是否需要阻塞导航）
      config: {
        showButtonsWithoutPermission: false, // 是否显示无权限的按钮
        isDeveloperAdmin: false,
        pprofEnabled: false,
        pprofTokenRequired: false,
        aiEnabled: false,
        ordersEnabled: true,
        paymentsEnabled: true,
        scheduleDemoEnabled: true,
        paymentGateways: null,
        devToolsEnabled: false,
        codeGeneratorEnabled: false,
        searchEnabled: false,
        searchDriver: 'null',
        otelEnabled: false,
        tenancyEnabled: false
      }
    }
  },

  getters: {
    isLoggedIn: (state) => !!state.token,
    hasPermission: (state) => (permission) => {
      // 超级管理员拥有所有权限
      if (state.isSuperAdmin) {
        return true
      }
      return state.permissions.includes(permission)
    },
    // 检查是否应该显示按钮（考虑权限和配置）
    shouldShowButton: (state) => (permission) => {
      // 超级管理员总是显示所有按钮
      if (state.isSuperAdmin) {
        return true
      }
      const hasPerm = state.permissions.includes(permission)
      // 如果有权限，总是显示
      if (hasPerm) {
        return true
      }
      // 如果没有权限，根据配置决定是否显示
      return state.config.showButtonsWithoutPermission
    }
  },

  actions: {
    setToken(token) {
      this.token = token
      Storage.setItem('token', token)
    },

    setAdminInfo(adminInfo) {
      // 只存储必要的基本信息，避免 localStorage 配额超限
      // 不存储 permissions, menus 等大数据字段，它们已经单独存储
      // roles 只存储基本信息（id 和 name），用于显示
      const basicInfo = {
        id: adminInfo.id || adminInfo.ID,
        username: adminInfo.username || adminInfo.Username,
        nickname: adminInfo.nickname || adminInfo.Nickname,
        avatar: adminInfo.avatar || adminInfo.Avatar,
        email: adminInfo.email || adminInfo.Email,
        phone: adminInfo.phone || adminInfo.Phone,
        department_id: adminInfo.department_id || adminInfo.DepartmentID,
        department: adminInfo.department || adminInfo.Department,
        must_change_password: !!(adminInfo.must_change_password ?? adminInfo.mustChangePassword),
        // roles 只存储基本信息，避免存储完整的关联数据
        roles: (adminInfo.roles || adminInfo.Roles || []).map(role => ({
          id: role.id || role.ID,
          name: role.name || role.Name,
          slug: role.slug || role.Slug
        }))
      }
      this.adminInfo = basicInfo
      // 设置超级管理员标识（从后端返回或从角色判断）
      this.isSuperAdmin = adminInfo.is_super_admin === true || adminInfo.isSuperAdmin === true || 
        (adminInfo.roles || adminInfo.Roles || []).some(role => 
          (role.slug || role.Slug) === 'super-admin' && (role.status || role.Status) === 1
        )
      
      // 使用 Storage 工具保存，自动处理错误
      const saved = Storage.setItem('adminInfo', basicInfo)
      if (!saved) {
        // 如果保存失败，尝试只存储最基本信息
        const minimalInfo = {
          id: basicInfo.id,
          username: basicInfo.username,
          nickname: basicInfo.nickname,
          avatar: basicInfo.avatar
        }
        const minimalSaved = Storage.setItem('adminInfo', minimalInfo)
        if (minimalSaved) {
          this.adminInfo = minimalInfo
        } else {
          logger.error('Failed to save adminInfo even with minimal data')
        }
      }
    },

    setPermissions(permissions) {
      // 如果 permissions 是对象数组，提取 slug 字段；如果是字符串数组，直接使用
      if (Array.isArray(permissions) && permissions.length > 0) {
        if (typeof permissions[0] === 'object' && permissions[0] !== null) {
          // 对象数组，提取 slug 字段
          this.permissions = compact(map(permissions, perm => perm.slug || perm.Slug || perm))
        } else {
          // 字符串数组，直接使用
          this.permissions = permissions
        }
      } else {
        this.permissions = []
      }
    },

    setMenus(menus) {
      this.menus = menus
      // 菜单不缓存到 localStorage，每次刷新都从服务器重新获取
    },

    setTenant(tenant) {
      if (!tenant || (!tenant.code && !tenant.name && tenant.id == null)) {
        this.tenant = null
        Storage.removeItem('tenantInfo')
        return
      }
      const normalized = {
        id: tenant.id,
        code: tenant.code ? String(tenant.code).trim().toLowerCase() : undefined,
        name: tenant.name ? String(tenant.name).trim() : undefined
      }
      this.tenant = normalized
      Storage.setItem('tenantInfo', normalized)
      if (normalized.code) {
        setTenantCode(normalized.code)
      }
    },

    setConfig(config) {
      this.config = {
        showButtonsWithoutPermission: config?.show_buttons_without_permission || config?.showButtonsWithoutPermission || false,
        isDeveloperAdmin: config?.is_developer_admin || config?.isDeveloperAdmin || false,
        pprofEnabled: config?.pprof_enabled || config?.pprofEnabled || false,
        pprofTokenRequired: config?.pprof_token_required || config?.pprofTokenRequired || false,
        aiEnabled: config?.ai_enabled || config?.aiEnabled || false,
        ordersEnabled: config?.orders_enabled ?? config?.ordersEnabled ?? true,
        paymentsEnabled: config?.payments_enabled ?? config?.paymentsEnabled ?? true,
        scheduleDemoEnabled: config?.schedule_demo_enabled ?? config?.scheduleDemoEnabled ?? true,
        paymentGateways: Array.isArray(config?.payment_gateways)
          ? config.payment_gateways
          : (Array.isArray(config?.paymentGateways) ? config.paymentGateways : null),
		devToolsEnabled: config?.dev_tools_enabled || config?.devToolsEnabled || false,
        codeGeneratorEnabled: config?.code_generator_enabled || config?.codeGeneratorEnabled || false,
        searchEnabled: !!config?.search_enabled,
        searchDriver: config?.search_driver || 'null',
        otelEnabled: config?.otel_enabled || config?.otelEnabled || false,
        tenancyEnabled: !!(config?.tenancy_enabled || config?.tenancyEnabled)
      }
    },

    async fetchUserInfo(force = false) {
      // 如果正在获取中，且不是强制刷新，则等待当前请求完成
      if (this.isFetchingUserInfo && !force) {
        // 等待当前请求完成
        while (this.isFetchingUserInfo) {
          await new Promise(resolve => setTimeout(resolve, 50))
        }
        return Promise.resolve()
      }
      
      // 如果已经获取过且不是强制刷新，且菜单不为空，直接返回
      if (this.userInfoFetched && !force && this.adminInfo && this.menus.length > 0) {
        if (!this.tenant?.code) {
          const localCode = resolveEffectiveTenantCode() || getTenantCode() || resolveTenantCodeFromLocation()
          if (localCode) {
            this.setTenant({
              code: localCode,
              ...(this.tenant?.name ? { name: this.tenant.name } : {})
            })
          }
        }
        return Promise.resolve()
      }
      
      try {
        this.isFetchingUserInfo = true
        
        // 保存当前菜单数据，避免在请求失败时丢失
        const oldMenus = this.menus.length > 0 ? [...this.menus] : []
        const oldAdminInfo = this.adminInfo
        const oldPermissions = [...this.permissions]
        
        // 清除旧的数据，确保获取最新的数据（仅在强制刷新或首次加载时）
        if (force || !this.userInfoFetched) {
          this.menus = []
          this.adminInfo = null
          this.permissions = []
          Storage.removeItem('adminInfo')
        }
        
        const res = await getInfo()
        if (res.data && res.data.admin) {
          this.setAdminInfo(res.data.admin)
          // 设置权限（需要先设置，因为 setAdminInfo 可能会用到）
          this.setPermissions(res.data.admin.permissions || [])
          
          // 确保菜单数据存在且不为空
          const newMenus = res.data.admin.menus || []
          if (newMenus.length > 0) {
            this.setMenus(newMenus)
          } else if (oldMenus.length > 0 && !force) {
            // 如果新菜单为空但旧菜单存在，保留旧菜单（可能是接口返回问题）
            logger.warn('New menus is empty, keeping old menus')
            this.setMenus(oldMenus)
          } else {
            this.setMenus([])
          }
          
          // 设置超级管理员标识
          this.isSuperAdmin = res.data.admin.is_super_admin === true || res.data.admin.isSuperAdmin === true ||
            (res.data.admin.roles || []).some(role => 
              (role.slug || role.Slug) === 'super-admin' && (role.status || role.Status) === 1
            )
          
          // 标记用户信息已获取
          this.userInfoFetched = true
          
          // 调试：打印权限信息（开发环境）
          logger.debug('User permissions:', this.permissions)
          logger.debug('Is super admin:', this.isSuperAdmin)
          logger.debug('Menus count:', this.menus.length)
        } else {
          // 如果接口返回的数据中没有 admin 信息，恢复旧数据
          if (oldMenus.length > 0 && !force) {
            logger.warn('No admin data in response, restoring old data')
            this.menus = oldMenus
            this.adminInfo = oldAdminInfo
            this.permissions = oldPermissions
          }
        }
        // 保存配置信息
        if (res.data && res.data.config) {
          this.setConfig(res.data.config)
        }
        if (res.data && res.data.tenant) {
          this.setTenant(res.data.tenant)
        } else {
          const localCode = resolveEffectiveTenantCode() || getTenantCode() || resolveTenantCodeFromLocation()
          if (localCode) {
            this.setTenant({
              code: localCode,
              ...(this.tenant?.name ? { name: this.tenant.name } : {})
            })
          } else if (!res.data?.config?.tenancy_enabled && !res.data?.config?.tenancyEnabled) {
            this.setTenant(null)
          }
        }
        return res
      } catch (error) {
        // fetchUserInfo 失败时，如果旧数据存在，恢复旧数据
        if (oldMenus.length > 0 && !force) {
          logger.warn('Failed to fetch user info, restoring old data:', error)
          this.menus = oldMenus
          this.adminInfo = oldAdminInfo
          this.permissions = oldPermissions
          // 不标记为失败，允许继续使用旧数据
          return Promise.resolve()
        }
        
        // 如果没有旧数据或强制刷新，清除状态
        this.userInfoFetched = false
        this.logout(true)
        throw error
      } finally {
        this.isFetchingUserInfo = false
      }
    },

    async logout(skipApiCall = false) {
      try {
        // 如果有 token 且不需要跳过 API 调用，尝试调用后端登出接口
        if (this.token && !skipApiCall) {
          await logout()
        }
      } catch (error) {
        // 即使登出接口失败，也要清除本地状态
        logger.error('Logout error:', error)
      } finally {
        // 清除所有状态（同步执行，不等待）
        this.token = ''
        this.adminInfo = null
        this.tenant = null
        this.permissions = []
        this.menus = []
        this.isSuperAdmin = false
        this.userInfoFetched = false
        this.isFetchingUserInfo = false
        this.config = {
          showButtonsWithoutPermission: false,
          isDeveloperAdmin: false,
          pprofEnabled: false,
          pprofTokenRequired: false,
          aiEnabled: false,
          ordersEnabled: true,
          paymentsEnabled: true,
          scheduleDemoEnabled: true,
          paymentGateways: null,
          devToolsEnabled: false,
          codeGeneratorEnabled: false,
          searchEnabled: false,
          searchDriver: 'null',
          otelEnabled: false,
          tenancyEnabled: false
        }
        Storage.removeItem('token')
        Storage.removeItem('adminInfo')
        Storage.removeItem('tenantInfo')
        clearTenantCode()
        // 菜单不缓存，无需清除
      }
      // 返回 resolved promise 确保调用者可以继续
      return Promise.resolve()
    }
  }
})

