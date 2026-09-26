import { getField } from './normalizeFormData'
import { buildSearchParams } from './buildSearchParams'

/**
 * Append *_start / *_end query params from datetimerange/daterange form fields.
 */
export function buildRangeSearchParams(form, baseParams, fieldNames = []) {
  const params = buildSearchParams(form, baseParams)

  fieldNames.forEach((fieldName) => {
    const rangeValue = form[fieldName]
    if (!Array.isArray(rangeValue) || rangeValue.length !== 2) {
      delete params[`${fieldName}_start`]
      delete params[`${fieldName}_end`]
      return
    }

    const [start, end] = rangeValue
    if (start) {
      params[`${fieldName}_start`] = start
    } else {
      delete params[`${fieldName}_start`]
    }
    if (end) {
      params[`${fieldName}_end`] = end
    } else {
      delete params[`${fieldName}_end`]
    }
    delete params[fieldName]
  })

  return params
}

/**
 * Build standard edit/delete action buttons for TableActionButtons.
 */
export function createCrudActions(t, moduleName, handlers) {
  const actions = []

  if (handlers.onEdit) {
    actions.push({
      key: 'edit',
      label: t('common.edit'),
      type: 'primary',
      permission: `${moduleName}.update`,
      handler: handlers.onEdit
    })
  }

  if (handlers.onDelete) {
    actions.push({
      key: 'delete',
      label: t('common.delete'),
      type: 'danger',
      permission: `${moduleName}.destroy`,
      handler: handlers.onDelete,
      disabled: handlers.deleteDisabled
    })
  }

  return actions
}

/**
 * Read normalized status value from a row.
 */
export function rowStatus(row, defaultValue = 1) {
  return Number(getField(row, 'status', defaultValue))
}

/**
 * Read display text from a row field.
 */
export function rowField(row, field, defaultValue = '-') {
  const value = getField(row, field, defaultValue)
  return value === undefined || value === null || value === '' ? defaultValue : value
}
