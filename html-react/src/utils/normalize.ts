import type { ApiResponse, PaginatedData } from '@/types'

function toSnakeCase(key: string): string {
  if (!key || key === 'ID') return 'id'
  return key
    .replace(/([A-Z])/g, '_$1')
    .replace(/^_/, '')
    .toLowerCase()
}

export function normalizeEntity<T>(entity: T): T {
  if (entity == null || typeof entity !== 'object') return entity

  if (Array.isArray(entity)) {
    return entity.map((item) => normalizeEntity(item)) as T
  }

  const source = entity as Record<string, unknown>
  const result: Record<string, unknown> = { ...source }

  for (const key of Object.keys(source)) {
    const snakeKey = toSnakeCase(key)
    const value = source[key]
    if (snakeKey !== key && result[snakeKey] === undefined && value !== undefined) {
      result[snakeKey] = value
    }
  }

  if (result.ID !== undefined && result.id === undefined) {
    result.id = result.ID
  }

  if (Array.isArray(result.children)) {
    result.children = normalizeTreeList(result.children)
  }

  return result as T
}

export function normalizeTreeList<T>(list: T[] | null | undefined): T[] {
  if (!Array.isArray(list)) return []
  return list.map((item) => normalizeEntity(item))
}

export function normalizeListResponse<T = unknown>(
  res: ApiResponse<PaginatedData<T> | unknown>,
): ApiResponse<PaginatedData<T>> {
  if (!res?.data || typeof res.data !== 'object') {
    return res as ApiResponse<PaginatedData<T>>
  }

  const data = res.data as PaginatedData<T> & Record<string, unknown>
  const list = data.list ?? data.data
  if (Array.isArray(list)) {
    const normalized = normalizeTreeList(list) as T[]
    // Always expose `list` so callers don't need legacy `data` fallbacks.
    data.list = normalized
  }

  return res as ApiResponse<PaginatedData<T>>
}

/** Read field with snake_case / PascalCase fallback. */
export function entityField<T = unknown>(
  row: Record<string, unknown> | null | undefined,
  fieldName: string,
  defaultValue?: T,
): T | undefined {
  if (!row) return defaultValue
  if (row[fieldName] !== undefined && row[fieldName] !== null) {
    return row[fieldName] as T
  }
  const pascal = fieldName
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join('')
  if (row[pascal] !== undefined && row[pascal] !== null) {
    return row[pascal] as T
  }
  if (fieldName === 'id' && row.ID !== undefined) return row.ID as T
  return defaultValue
}
