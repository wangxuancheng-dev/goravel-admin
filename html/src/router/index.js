import { createRouter, createWebHistory } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../store/user'
import logger from '../utils/logger'
import { flattenTree } from '../utils/tree'
import { getPlatformToken } from '../utils/platformRequest'

/**
 * 带重试和错误处理的动态导入包装函数
 * @param {Function} importFn - 动态导入函数
 * @param {number} maxRetries - 最大重试次数，默认 3
 * @param {number} timeout - 超时时间（毫秒），默认 30 秒
 * @returns {Promise} 导入的模块
 */
function lazyLoad(importFn, maxRetries = 3, timeout = 10000) {
  return new Promise((resolve, reject) => {
    let retryCount = 0
    
    const attemptLoad = () => {
      // 创建超时 Promise
      const timeoutPromise = new Promise((_, timeoutReject) => {
        setTimeout(() => {
          timeoutReject(new Error('模块加载超时'))
        }, timeout)
      })
      
      // 创建加载 Promise
      const loadPromise = importFn().catch(err => {
        // 如果是网络错误或加载失败，可以重试
        if (err.message && (
          err.message.includes('Failed to fetch') ||
          err.message.includes('Loading chunk') ||
          err.message.includes('Loading CSS chunk')
        )) {
          throw err
        }
        throw err
      })
      
      // 竞争加载和超时
      Promise.race([loadPromise, timeoutPromise])
        .then(module => {
          resolve(module)
        })
        .catch(error => {
          retryCount++
          
          if (retryCount < maxRetries) {
            logger.warn(`模块加载失败，正在重试 (${retryCount}/${maxRetries}):`, error.message)
            // 指数退避：1秒、2秒、4秒
            const delay = Math.min(1000 * Math.pow(2, retryCount - 1), 5000)
            setTimeout(() => {
              attemptLoad()
            }, delay)
          } else {
            logger.error('模块加载失败，已达到最大重试次数:', error)
            ElMessage.error({
              message: '页面加载失败，请刷新页面重试',
              duration: 5000,
              showClose: true
            })
            reject(error)
          }
        })
    }
    
    attemptLoad()
  })
}

// 固定路由（不需要从接口获取）
const staticRoutes = [
  {
    path: '/login',
    name: 'Login',
    component: () => lazyLoad(() => import('../views/Login.vue')),
    meta: { requiresAuth: false }
  },
  {
    path: '/platform/login',
    name: 'PlatformLogin',
    component: () => lazyLoad(() => import('../views/platform/Login.vue')),
    meta: { requiresAuth: false, platform: true }
  },
  {
    path: '/platform',
    name: 'PlatformLayout',
    component: () => lazyLoad(() => import('../views/platform/Layout.vue')),
    redirect: '/platform/overview',
    meta: { requiresAuth: true, platform: true },
    children: [
      {
        path: 'overview',
        name: 'PlatformOverview',
        component: () => lazyLoad(() => import('../views/platform/Overview.vue')),
        meta: { titleKey: 'menu.platform_overview', platform: true, requiresAuth: true }
      },
      {
        path: 'tenants',
        name: 'PlatformTenants',
        component: () => lazyLoad(() => import('../views/platform/TenantList.vue')),
        meta: { titleKey: 'menu.tenant', platform: true, requiresAuth: true }
      },
      {
        path: 'tenant-op-logs',
        name: 'PlatformTenantOpLogs',
        component: () => lazyLoad(() => import('../views/platform/TenantOpLogList.vue')),
        meta: { titleKey: 'menu.tenant_op_log', platform: true, requiresAuth: true }
      }
    ]
  },
  {
    path: '/',
    name: 'MainLayout',
    component: () => lazyLoad(() => import('../layouts/MainLayout.vue')),
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => lazyLoad(() => import('../views/Dashboard.vue')),
        meta: { titleKey: 'menu.dashboard' }
      },
      {
        path: 'profile',
        name: 'Profile',
        component: () => lazyLoad(() => import('../views/profile/Profile.vue')),
        meta: { titleKey: 'menu.profile' }
      },
      {
        path: 'iframe',
        name: 'Iframe',
        component: () => lazyLoad(() => import('../views/iframe/IframeView.vue')),
        meta: { titleKey: 'menu.external_link' }
      },
      {
        path: 'dev/form-demo',
        name: 'FormDemo',
        component: () => lazyLoad(() => import('../views/dev/FormDemo.vue')),
        meta: { titleKey: 'menu.form_demo' }
      },
      {
        // 404 路由，必须放在最后，作为 catch-all 路由
        // 在子路由中使用相对路径（不带前导斜杠）
        path: ':pathMatch(.*)*',
        name: 'NotFound',
        component: () => lazyLoad(() => import('../views/NotFound.vue')),
        meta: { titleKey: 'notFound.title' }
      }
    ]
  }
]

/**
 * 使用 import.meta.glob 动态导入所有 views 目录下的 Vue 组件
 * 这样菜单的 component 字段可以直接设置路径，无需在此维护映射表
 * 
 * 支持的 component 格式：
 * - 'admin/AdminList' -> ../views/admin/AdminList.vue
 * - 'log/OperationLogList' -> ../views/log/OperationLogList.vue
 * - 'views/admin/AdminList.vue' -> ../views/admin/AdminList.vue (完整路径)
 */
const viewModules = import.meta.glob('../views/**/*.vue')

/**
 * 获取组件的导入函数
 * @param {string} component - 菜单的 component 字段
 *   支持格式：
 *   - 'admin/AdminList' -> ../views/admin/AdminList.vue
 *   - 'log/OperationLogList' -> ../views/log/OperationLogList.vue
 *   - 'views/admin/AdminList.vue' -> ../views/admin/AdminList.vue
 * @returns {Function|null} 组件导入函数，如果不存在则返回 null
 */
function getComponentImport(component) {
  if (!component || component === 'Layout') {
    return null
  }

  // 标准化路径：将 component 转换为 import.meta.glob 的键格式
  let modulePath = component
  
  // 移除开头的 views/ 或 ../views/（如果有）
  modulePath = modulePath.replace(/^(\.\.\/)?views\//, '')
  
  // 移除结尾的 .vue（如果有）
  modulePath = modulePath.replace(/\.vue$/, '')
  
  // 构建可能的路径列表（支持 article/ArticleList 和 views/article/ArticleList 等多种格式）
  const possiblePaths = [
    `../views/${modulePath}.vue`,
    `../views/${modulePath}/index.vue`, // 支持 index.vue
    `../views/${modulePath.replace(/^\//, '')}.vue`, // 确保没有双斜杠
  ]

  // 查找模块
  let moduleImport = null
  let fullPath = ''

  for (const path of possiblePaths) {
    if (viewModules[path]) {
      moduleImport = viewModules[path]
      fullPath = path
      break
    }
  }
  
  if (moduleImport) {
    return () => lazyLoad(moduleImport)
  }

  // 如果找不到，记录警告
  logger.warn(`Component not found: ${component} (tried: ${possiblePaths.join(', ')})`)
  logger.debug('Available modules:', Object.keys(viewModules))
  return null
}

/**
 * 将菜单数据转换为路由配置
 * @param {Array} menus - 菜单数组（支持树形结构）
 * @returns {Array} 路由配置数组
 */
function convertMenusToRoutes(menus) {
  if (!menus || !Array.isArray(menus)) {
    return []
  }

  // 先将树形结构扁平化
  const flatMenus = flattenTree(menus, 'children')
  const routes = []
  const processedPaths = new Set() // 避免重复路由

  flatMenus.forEach(menu => {
    // 只处理类型为菜单（type === 2）且状态为启用（status === 1）的菜单
    const type = menu.Type !== undefined ? menu.Type : (menu.type !== undefined ? menu.type : 1)
    const status = menu.Status !== undefined ? menu.Status : (menu.status !== undefined ? menu.status : 1)
    const linkType = menu.LinkType !== undefined ? menu.LinkType : (menu.link_type !== undefined ? menu.link_type : 1)
    
    // 如果有组件路径，即使类型不是菜单（可能是误操作或历史数据），也尝试生成路由
    // 但必须是启用的，且不是按钮类型（type === 3）
    const component = menu.Component || menu.component || ''
    if (status !== 1 || type === 3) {
      return
    }
    
    // 如果是目录类型（type === 1）且没有组件路径，跳过（仅作为父级菜单）
    if (type === 1 && !component) {
      return
    }

    const path = menu.Path || menu.path || ''
    
    // 如果没有路径，跳过
    if (!path || path === '/') {
      return
    }

    // 避免重复路由
    if (processedPaths.has(path)) {
      return
    }
    processedPaths.add(path)

    // 处理路径：移除前导斜杠，子路由使用相对路径（不带前导斜杠）
    // 静态路由中的子路由都是相对路径，如 "dashboard", "profile"
    const routePath = path.startsWith('/') ? path.slice(1) : path
    
    // 生成路由名称（从路径转换，如 "admins" -> "Admins", "user-balance-logs" -> "UserBalanceLogs"）
    const routeName = routePath
      .split('/')  // 先按 / 分割，处理多级路径
      .filter(Boolean) // 移除空字符串
      .map(part => {
         // 处理带连字符的部分
         return part.split('-')
           .map(p => p.charAt(0).toUpperCase() + p.slice(1))
           .join('')
      })
      .join('')

    // 生成 titleKey
    // 翻译文件中的键通常是 menu.xxx_management 格式
    // 但有些菜单的 slug 可能已经包含了 _management，所以需要智能处理
    const slug = menu.Slug || menu.slug || routePath
    
    // 如果 slug 是以 / 开头的路径（如 /articles），去掉开头的 /
    const cleanSlug = slug.startsWith('/') ? slug.slice(1) : slug
    
    let titleKey = `menu.${cleanSlug}`
    
    // 如果 slug 不包含 _management，尝试添加后缀
    // 但先检查原始键是否存在，如果存在就不添加后缀
    // 注意：这里我们无法直接检查翻译键，所以先尝试添加 _management
    // BreadcrumbView 会使用智能翻译函数来处理
    
    // 是否缓存：no_cache 为 1 时每次进页面刷新接口
    const noCache = menu.no_cache === 1 || menu.NoCache === 1

    // 构建路由配置
    const route = {
      path: routePath,
      name: routeName,
      meta: {
        titleKey: titleKey,
        menuId: menu.id || menu.ID,
        menuSlug: cleanSlug, // 保存 slug，供 BreadcrumbView 使用
        noCache: !!noCache
      }
    }

    // 如果是外部链接（linkType === 2），使用 iframe 组件
    if (linkType === 2) {
      route.component = () => lazyLoad(() => import('../views/iframe/IframeView.vue'))
      route.meta.externalUrl = path
    } else {
      // 内部页面，根据 component 字段获取组件导入函数
      const componentImport = getComponentImport(component)
      if (componentImport) {
        route.component = componentImport
      } else {
        // Layout 组件视为目录，不需要生成路由
        if (component === 'Layout') {
          return
        }
        // 如果无法获取组件导入函数，跳过（可能是目录类型或其他特殊类型）
        logger.warn(`Skipping route ${path} due to missing component import for: ${component}`)
        return
      }
    }

    routes.push(route)
  })

  return routes
}

// 标记是否已经添加过动态路由
let dynamicRoutesAdded = false

// 防止路由守卫在接口异常/菜单为空时陷入 next(to.fullPath) 死循环（从而导致 /info 无限请求）
const navigationRetryState = new Map() // fullPath -> { count: number, lastAt: number }
const NAVIGATION_RETRY_WINDOW = 3000 // 3 秒窗口
const MAX_RETRIES_PER_PATH = 1

let lastMenuRefreshAttemptAt = 0
const MENU_REFRESH_COOLDOWN = 5000 // 5 秒内不重复刷新菜单

function canRetryNavigation(fullPath) {
  const now = Date.now()
  const state = navigationRetryState.get(fullPath)
  if (!state || now - state.lastAt > NAVIGATION_RETRY_WINDOW) {
    navigationRetryState.set(fullPath, { count: 0, lastAt: now })
    return true
  }
  return state.count < MAX_RETRIES_PER_PATH
}

function markNavigationRetried(fullPath) {
  const now = Date.now()
  const state = navigationRetryState.get(fullPath) || { count: 0, lastAt: now }
  state.count += 1
  state.lastAt = now
  navigationRetryState.set(fullPath, state)
}

// 初始路由（只包含固定路由）
const routes = [...staticRoutes]

const router = createRouter({
  history: createWebHistory(),
  routes
})

/**
 * 动态添加路由
 * @param {Array} menus - 菜单数组
 */
function addDynamicRoutes(menus) {
  if (!menus || menus.length === 0) {
    return
  }

  const dynamicRoutes = convertMenusToRoutes(menus)
  
  if (dynamicRoutes.length === 0) {
    logger.warn('No dynamic routes to add')
    return
  }

  // 检查路由是否已存在，避免重复添加
  const existingRoutes = router.getRoutes()
  const existingPaths = new Set(
    existingRoutes
      .filter(route => route.path !== '/' && route.path !== '/login')
      .flatMap(route => route.children || [])
      .map(child => child.path)
  )

  // 只添加不存在的路由
  const routesToAdd = dynamicRoutes.filter(route => !existingPaths.has(route.path))
  
  if (routesToAdd.length === 0) {
    logger.debug('All dynamic routes already exist')
    return
  }

  // 找到主布局路由（path === '/' 或 name === 'MainLayout'）
  const mainLayoutRoute = existingRoutes.find(route => route.path === '/' || route.name === 'MainLayout')
  
  if (!mainLayoutRoute) {
    logger.error('Main layout route not found')
    return
  }

  // 添加新路由到主布局路由
  // Vue Router 的子路由路径应该是相对路径（不带前导斜杠）
  // 但为了匹配 URL 路径（如 /users），我们需要确保路径格式正确
  const parentName = mainLayoutRoute.name || 'MainLayout'
  routesToAdd.forEach(route => {
    // 子路由路径应该是相对路径（不带前导斜杠）
    // 这样 Vue Router 会自动处理路径匹配
    const routePath = route.path.startsWith('/') ? route.path.slice(1) : route.path
    
    const routeConfig = {
      ...route,
      path: routePath
    }
    
    try {
      router.addRoute(parentName, routeConfig)
      logger.debug(`Added route: ${routePath} (parent: ${parentName})`)
    } catch (error) {
      logger.error(`Failed to add route ${routePath}:`, error)
    }
  })
  
  logger.debug(`Added ${routesToAdd.length} dynamic routes:`, routesToAdd.map(r => r.path))
}

/**
 * 重置动态路由标志（在登出时调用）
 */
export function resetDynamicRoutes() {
  dynamicRoutesAdded = false
}

router.beforeEach((to, from, next) => {
  const userStore = useUserStore()
  const isPlatformRoute = to.matched.some((r) => r.meta?.platform) || String(to.path || '').startsWith('/platform')

  if (isPlatformRoute) {
    const platformToken = getPlatformToken()
    if (to.meta.requiresAuth === false || to.path === '/platform/login') {
      if (platformToken && to.path === '/platform/login') {
        next('/platform/overview')
      } else {
        next()
      }
      return
    }
    if (!platformToken) {
      next('/platform/login')
      return
    }
    next()
    return
  }

  if (to.meta.requiresAuth === false) {
    // 登录页面，如果已登录则跳转到首页
    if (userStore.isLoggedIn) {
      next('/')
    } else {
      // 如果未登录，重置动态路由标志
      if (dynamicRoutesAdded) {
        dynamicRoutesAdded = false
      }
      next()
    }
  } else {
    // 需要认证的页面
    if (!userStore.isLoggedIn) {
      // 如果没有token，重置动态路由标志并跳转到登录页
      if (dynamicRoutesAdded) {
        dynamicRoutesAdded = false
      }
      next('/login')
    } else {
      // 优化：只在首次加载（从登录页或刷新页面）时才阻塞导航
      // 如果用户信息已获取过，允许导航继续，菜单可以在后台异步加载
      const isFirstLoad = !from.name || from.name === 'Login'
      
      // 检查菜单是否为空（即使 userInfoFetched 为 true，菜单也可能被意外清空）
      const menusEmpty = !userStore.menus || userStore.menus.length === 0
      
      // 如果用户信息已获取过，但菜单为空，需要重新获取菜单（不阻塞导航）
      if (userStore.userInfoFetched && menusEmpty && userStore.adminInfo) {
        const now = Date.now()
        // 冷却期内不重复拉取菜单，避免 /info 被频繁请求
        if (now - lastMenuRefreshAttemptAt < MENU_REFRESH_COOLDOWN) {
          const resolved = router.resolve(to.path)
          if (!resolved.name && to.path !== '/dashboard') {
            next('/dashboard')
          } else {
            next()
          }
          return
        }

        lastMenuRefreshAttemptAt = now

        // 阻塞导航，等待菜单加载完成（因为当前路由可能依赖动态路由）
        userStore.fetchUserInfo().then(() => {
          // 获取菜单后添加动态路由
          if (userStore.menus && userStore.menus.length > 0 && !dynamicRoutesAdded) {
            addDynamicRoutes(userStore.menus)
            dynamicRoutesAdded = true
          }

          // 如果菜单仍为空，不再对当前路径做 next(to.fullPath) 递归重试，直接放行或降级
          if (!userStore.menus || userStore.menus.length === 0) {
            const resolved = router.resolve(to.path)
            if (!resolved.name && to.path !== '/dashboard') {
              next('/dashboard')
            } else {
              next()
            }
            return
          }

          // 只有在允许重试时才 next(to.fullPath)，避免死循环
          if (canRetryNavigation(to.fullPath)) {
            markNavigationRetried(to.fullPath)
            next(to.fullPath)
          } else {
            next('/dashboard')
          }
        }).catch((error) => {
          logger.error('Failed to refresh menus:', error)
          // 获取失败时不再反复重试，直接回到 dashboard（避免触发更多 /info）
          next('/dashboard')
        })
        return
      }
      
      // 如果用户信息已获取过且菜单不为空，检查是否需要添加动态路由
      if (userStore.userInfoFetched && !menusEmpty) {
        // 如果还没有添加动态路由，现在添加
        if (!dynamicRoutesAdded && userStore.menus && userStore.menus.length > 0) {
          addDynamicRoutes(userStore.menus)
          dynamicRoutesAdded = true
          // 路由添加后，使用 next() 重试导航
          if (canRetryNavigation(to.fullPath)) {
            markNavigationRetried(to.fullPath)
            next(to.fullPath)
          } else {
            next('/dashboard')
          }
          return
        }
        // 检查路由是否存在（只检查路径，不包含查询参数）
        const route = router.resolve(to.path)
        if (!route.name && to.path !== '/') {
          // 路由不存在，可能是路径不匹配；只允许重试一次，避免无限循环
          if (canRetryNavigation(to.fullPath)) {
            markNavigationRetried(to.fullPath)
            next(to.fullPath)
          } else {
            next('/dashboard')
          }
          return
        }
        if (
          userStore.adminInfo?.must_change_password &&
          to.path !== '/profile' &&
          to.path !== '/login'
        ) {
          next({ path: '/profile', query: { tab: 'password' } })
          return
        }
        next()
        return
      }
      
      // 首次加载：如果用户信息不存在或菜单为空，需要获取
      if (!userStore.adminInfo || menusEmpty) {
        // 阻塞导航，等待用户信息加载完成
        userStore.fetchUserInfo().then(() => {
          // 获取菜单后添加动态路由
          if (userStore.menus && userStore.menus.length > 0 && !dynamicRoutesAdded) {
            addDynamicRoutes(userStore.menus)
            dynamicRoutesAdded = true
          }
          // 菜单为空时不递归重试，避免 /info 无限请求
          if (!userStore.menus || userStore.menus.length === 0) {
            const resolved = router.resolve(to.path)
            if (!resolved.name && to.path !== '/dashboard') {
              next('/dashboard')
            } else {
              next()
            }
            return
          }

          // 路由添加后，使用 next() 重试导航（仅一次）
          if (canRetryNavigation(to.fullPath)) {
            markNavigationRetried(to.fullPath)
            next(to.fullPath)
          } else {
            next('/dashboard')
          }
        }).catch((error) => {
          // 如果获取用户信息失败（可能是401），拦截器会处理跳转
          // 这里只需要阻止导航
          next(false)
        })
      } else {
        // 用户信息已存在，检查是否需要添加动态路由
        if (!dynamicRoutesAdded && userStore.menus && userStore.menus.length > 0) {
          addDynamicRoutes(userStore.menus)
          dynamicRoutesAdded = true
          // 路由添加后，使用 next() 重试导航
          if (canRetryNavigation(to.fullPath)) {
            markNavigationRetried(to.fullPath)
            next(to.fullPath)
          } else {
            next('/dashboard')
          }
          return
        }
        // 检查路由是否存在（只检查路径，不包含查询参数）
        const route = router.resolve(to.path)
        if (!route.name && to.path !== '/') {
          // 路由不存在，可能是路径不匹配；只允许重试一次，避免无限循环
          if (canRetryNavigation(to.fullPath)) {
            markNavigationRetried(to.fullPath)
            next(to.fullPath)
          } else {
            next('/dashboard')
          }
          return
        }
        // 标记为已获取，避免后续路由切换时重复检查
        userStore.userInfoFetched = true
        next()
      }
    }
  }
})

// 捕获路由错误（包括动态导入失败和路由未找到）
router.onError((error) => {
  logger.error('Router error:', error)
  
  // 如果是路由未找到的错误，尝试重新加载动态路由
  if (error.message && (
    error.message.includes('No match found') ||
    error.message.includes('No match') ||
    error.name === 'NavigationFailure'
  )) {
    const userStore = useUserStore()
    // 如果用户已登录但路由未找到，可能是动态路由未加载
    if (userStore.isLoggedIn && userStore.menus && userStore.menus.length > 0 && !dynamicRoutesAdded) {
      logger.warn('Route not found, attempting to reload dynamic routes')
      addDynamicRoutes(userStore.menus)
      dynamicRoutesAdded = true
      // 重试导航
      router.push(router.currentRoute.value.fullPath).catch(() => {
        // 如果重试失败，跳转到首页
        router.push('/').catch(() => {})
      })
    }
  }
  
  // 检查是否是动态导入失败
  if (error.message && (
    error.message.includes('Failed to fetch dynamically imported module') ||
    error.message.includes('Loading chunk') ||
    error.message.includes('Loading CSS chunk') ||
    error.name === 'ChunkLoadError'
  )) {
    ElMessage.error({
      message: '页面加载失败，请刷新页面重试',
      duration: 5000,
      showClose: true
    })
    
    // 可以尝试重新加载页面
    const retry = () => {
      window.location.reload()
    }
    
    // 延迟 2 秒后自动刷新，给用户时间看到错误提示
    setTimeout(retry, 2000)
  } else {
    ElMessage.error({
      message: '路由导航失败，请刷新页面',
      duration: 3000
    })
  }
})

export default router

