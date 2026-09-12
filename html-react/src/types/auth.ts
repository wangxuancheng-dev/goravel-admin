export interface AdminRole {
  id: number | string
  name: string
  slug?: string
  status?: number
}

export interface AdminInfo {
  id: number | string
  username: string
  nickname?: string
  avatar?: string
  email?: string
  phone?: string
  department_id?: number | string
  department?: unknown
  roles?: AdminRole[]
  permissions?: Array<string | { slug?: string; Slug?: string }>
  menus?: MenuNode[]
  is_super_admin?: boolean
  isSuperAdmin?: boolean
}

export interface FeatureConfig {
  showButtonsWithoutPermission: boolean
  isDeveloperAdmin: boolean
  pprofEnabled: boolean
  pprofTokenRequired: boolean
  aiEnabled: boolean
  ordersEnabled: boolean
  paymentsEnabled: boolean
  paymentGateways: string[] | null
  devToolsEnabled: boolean
  codeGeneratorEnabled: boolean
  searchEnabled: boolean
  searchDriver: string
  otelEnabled: boolean
}

export interface UserInfoPayload {
  admin: AdminInfo
  config?: Record<string, unknown>
}

export interface LoginPayload {
  username: string
  password: string
  google_code?: string
  captcha_id?: string
  captcha_answer?: string
}

export interface CaptchaInfo {
  captcha?: {
    enabled?: boolean
    captcha_id?: string
    captcha_image?: string
  }
}

export interface MenuNode {
  id?: number | string
  ID?: number | string
  name?: string
  Name?: string
  title?: string
  Title?: string
  path?: string
  Path?: string
  component?: string
  Component?: string
  icon?: string
  Icon?: string
  slug?: string
  Slug?: string
  type?: number
  Type?: number
  status?: number
  Status?: number
  link_type?: number
  LinkType?: number
  sort?: number
  Sort?: number
  no_cache?: number
  NoCache?: number
  children?: MenuNode[]
  Children?: MenuNode[]
}
