import { entityField } from '@/utils/normalize'

export const adminInitialSearchForm = {
  username: '',
  status: '',
  is_2fa_bound: '',
}

export type AdminSearchForm = typeof adminInitialSearchForm

export interface AdminRow {
  id: number | string
  username: string
  nickname?: string
  email?: string
  phone?: string
  status?: number
  created_at?: string
  is_2fa_bound?: boolean
  department?: { name?: string } | string
  position?: { name?: string } | string
  roles?: Array<{ id?: number | string; name?: string }>
}

export const adminProtectedIds = new Set([1, 2])

export function transformAdminRow(row: Record<string, unknown>): AdminRow {
  const rawRoles = (entityField(row, 'roles', []) as Array<Record<string, unknown>>) || []
  const roles = rawRoles.map((role) => ({
    id: entityField(role, 'id', '') as number | string,
    name: String(entityField(role, 'name', '') ?? ''),
  }))

  return {
    id: entityField(row, 'id', '')!,
    username: String(entityField(row, 'username', '') ?? ''),
    nickname: String(entityField(row, 'nickname', '') ?? ''),
    email: String(entityField(row, 'email', '') ?? ''),
    phone: String(entityField(row, 'phone', '') ?? ''),
    status: Number(entityField(row, 'status', 0) ?? 0),
    created_at: String(entityField(row, 'created_at', '') ?? ''),
    is_2fa_bound: !!(entityField(row, 'is_2fa_bound', false) || entityField(row, 'Is2faBound', false)),
    department: (entityField(row, 'department', null) as AdminRow['department']) || undefined,
    position: (entityField(row, 'position', null) as AdminRow['position']) || undefined,
    roles,
  }
}

export function getAdminDeptName(row: AdminRow) {
  if (!row.department) return '-'
  return typeof row.department === 'string' ? row.department : row.department.name || '-'
}

export function getAdminPositionName(row: AdminRow) {
  if (!row.position) return '-'
  return typeof row.position === 'string' ? row.position : row.position.name || '-'
}

export function getAdminRoleNames(row: AdminRow) {
  if (!row.roles?.length) return '-'
  return row.roles.map((r) => r.name).filter(Boolean).join(', ') || '-'
}
