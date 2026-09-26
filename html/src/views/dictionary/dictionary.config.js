export const dictionaryInitialSearchForm = {
  type: ''
}

export function createDictionarySearchFields(t, typeOptions = []) {
  return [
    {
      prop: 'type',
      label: t('dictionary.type'),
      type: 'select',
      width: '200px',
      clearable: true,
      filterable: true,
      advanced: false,
      options: typeOptions
    }
  ]
}

export function createDictionaryTableColumns(t) {
  return [
    { field: 'id', title: t('table.id'), width: 80, sortable: true, key: 'id' },
    { field: 'type', title: t('dictionary.type'), sortable: false, key: 'type' },
    { field: 'label', title: t('dictionary.label'), sortable: false, key: 'label' },
    { field: 'value', title: t('dictionary.value'), sortable: false, key: 'value' },
    { field: 'translation_key', title: t('dictionary.translation_key'), sortable: false, key: 'translation_key' },
    { field: 'is_system', title: t('dictionary.is_system'), width: 100, sortable: false, slot: 'is_system', key: 'is_system' },
    { field: 'sort', title: t('common.sort'), width: 80, sortable: true, key: 'sort' },
    { field: 'status', title: t('table.status'), width: 80, sortable: false, slot: 'status', key: 'status' },
    { field: 'created_at', title: t('table.created_at'), sortable: true, key: 'created_at' },
    { title: t('table.operation'), width: 150, fixed: 'right', slot: 'operation', sortable: false, key: 'operation' }
  ]
}

export function isSystemDictionary(row) {
  return Number(row?.is_system ?? row?.IsSystem ?? 0) === 1
}
