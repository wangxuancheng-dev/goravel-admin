/** Form / search UI options constrained by DB field type (code generator). */

export const ALL_FORM_TYPE_VALUES = [
  'input',
  'textarea',
  'editor',
  'markdown',
  'image-upload',
  'select',
  'radio',
  'checkbox',
  'switch',
  'number',
  'date-picker',
  'datetime-picker',
] as const

export type FormTypeValue = (typeof ALL_FORM_TYPE_VALUES)[number]

export const ALL_SEARCH_UI_VALUES = [
  'input',
  'select',
  'date',
  'datetime',
  'daterange',
  'datetimerange',
] as const

export type SearchUiValue = (typeof ALL_SEARCH_UI_VALUES)[number]

const NUMERIC_TYPES = new Set([
  'integer',
  'bigInteger',
  'unsignedBigInteger',
  'unsignedTinyInteger',
  'decimal',
  'timestamp',
])

export function isNumericFieldType(dbType: string): boolean {
  return NUMERIC_TYPES.has(dbType)
}

export function defaultFormType(dbType: string): FormTypeValue {
  switch (dbType) {
    case 'text':
    case 'json':
      return 'textarea'
    case 'integer':
    case 'bigInteger':
    case 'unsignedBigInteger':
    case 'unsignedTinyInteger':
    case 'decimal':
    case 'timestamp':
      return 'number'
    case 'boolean':
      return 'switch'
    case 'date':
      return 'date-picker'
    case 'datetime':
      return 'datetime-picker'
    default:
      return 'input'
  }
}

export function allowedFormTypes(dbType: string): FormTypeValue[] {
  switch (dbType) {
    case 'string':
      return ['input', 'textarea', 'editor', 'markdown', 'image-upload', 'select', 'radio', 'checkbox']
    case 'text':
      return ['textarea', 'editor', 'markdown', 'input']
    case 'integer':
    case 'bigInteger':
    case 'unsignedBigInteger':
      return ['number', 'select', 'radio']
    case 'unsignedTinyInteger':
      return ['number', 'select', 'radio', 'switch']
    case 'decimal':
      return ['number', 'input']
    case 'boolean':
      return ['switch', 'radio', 'select']
    case 'date':
      return ['date-picker']
    case 'datetime':
      return ['datetime-picker', 'date-picker']
    case 'timestamp':
      return ['number', 'datetime-picker']
    case 'json':
      return ['textarea', 'input']
    default:
      return ['input', 'textarea', 'select', 'number']
  }
}

export function defaultSearchType(dbType: string): string {
  if (dbType === 'string' || dbType === 'text' || dbType === 'json') {
    return 'like'
  }
  return '='
}

export function allowedSearchUiTypes(dbType: string): SearchUiValue[] {
  switch (dbType) {
    case 'date':
      return ['date', 'daterange', 'select']
    case 'datetime':
      return ['datetime', 'datetimerange', 'date', 'daterange', 'select']
    case 'timestamp':
      return ['datetime', 'datetimerange', 'select', 'input']
    case 'integer':
    case 'bigInteger':
    case 'unsignedBigInteger':
    case 'unsignedTinyInteger':
    case 'decimal':
    case 'boolean':
      return ['input', 'select']
    case 'text':
    case 'json':
      return ['input', 'select']
    default:
      return [...ALL_SEARCH_UI_VALUES]
  }
}

export function defaultSearchUiType(dbType: string): SearchUiValue {
  const allowed = allowedSearchUiTypes(dbType)
  if (dbType === 'date') return allowed.includes('date') ? 'date' : allowed[0]
  if (dbType === 'datetime' || dbType === 'timestamp') {
    return allowed.includes('datetime') ? 'datetime' : allowed[0]
  }
  if (dbType === 'boolean') return allowed.includes('select') ? 'select' : allowed[0]
  return allowed.includes('input') ? 'input' : allowed[0]
}

export type FieldControlPatch = {
  type?: string
  form_type?: string
  search_type?: string
  search_ui_type?: string
}

/** Ensure form_type / search_* match the field DB type; used on type change and import. */
export function normalizeFieldControls<T extends FieldControlPatch>(field: T): T {
  const dbType = field.type || 'string'
  const formAllowed = allowedFormTypes(dbType)
  const searchUiAllowed = allowedSearchUiTypes(dbType)
  let formType = field.form_type || ''
  if (!formAllowed.includes(formType as FormTypeValue)) {
    formType = defaultFormType(dbType)
  }
  let searchUi = field.search_ui_type || ''
  if (!searchUiAllowed.includes(searchUi as SearchUiValue)) {
    searchUi = defaultSearchUiType(dbType)
  }
  let searchType = field.search_type || defaultSearchType(dbType)
  if (isNumericFieldType(dbType) || dbType === 'boolean' || dbType === 'date' || dbType === 'datetime') {
    if (searchType === 'like') {
      searchType = '='
    }
  }
  return {
    ...field,
    form_type: formType,
    search_ui_type: searchUi,
    search_type: searchType,
  }
}
