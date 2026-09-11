import { getField } from './normalizeFormData'

/**
 * Convert PascalCase / camelCase key to snake_case.
 */
function toSnakeCase(key) {
  if (!key || key === 'ID') return 'id'
  return key
    .replace(/([A-Z])/g, '_$1')
    .replace(/^_/, '')
    .toLowerCase()
}

/**
 * Normalize a single API entity: prefer snake_case keys, unify id/status aliases.
 * Nested objects and tree children are normalized recursively.
 */
export function normalizeEntity(entity) {
  if (entity == null || typeof entity !== 'object') {
    return entity
  }

  if (Array.isArray(entity)) {
    return entity.map(normalizeEntity)
  }

  const result = { ...entity }

  for (const key of Object.keys(entity)) {
    const snakeKey = toSnakeCase(key)
    const value = entity[key]

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

  return result
}

/**
 * Normalize a flat or tree list from API responses.
 */
export function normalizeTreeList(list) {
  if (!Array.isArray(list)) {
    return []
  }
  return list.map(normalizeEntity)
}

/**
 * Normalize paginated list response data in place.
 */
export function normalizeListResponse(res) {
  if (!res?.data) {
    return res
  }

  const list = res.data.list ?? res.data.data
  if (Array.isArray(list)) {
    // Always expose `list` so callers don't need legacy `data` fallbacks.
    res.data.list = normalizeTreeList(list)
  }

  return res
}

/**
 * Read entity field with snake_case / PascalCase fallback.
 */
export function entityField(row, fieldName, defaultValue = undefined) {
  return getField(row, fieldName, defaultValue)
}
