import request from '@/utils/request'

export interface CodeGeneratorField {
  name: string
  type: string
  label: string
  required?: boolean
  searchable?: boolean
  sortable?: boolean
  show_in_list?: boolean
  show_in_form?: boolean
  show_in_detail?: boolean
  is_primary_key?: boolean
  comment?: string
  search_type?: string
  search_ui_type?: string
  form_type?: string
  relation?: {
    table: string
    relation_type: string
    foreign_key: string
    display_field: string
    alias?: string
    is_tree?: boolean
  } | null
  dictionary?: string
  api_url?: string
  precision?: number
  scale?: number
  db_type?: string
}

export interface CodeGeneratorOptions {
  has_create?: boolean
  has_edit?: boolean
  has_delete?: boolean
  has_export?: boolean
  export_async?: boolean
  has_import?: boolean
  import_async?: boolean
  enable_batch_actions?: boolean
  show_toolbar?: boolean
  is_tree_list?: boolean
}

export interface ModuleInstallConfig {
  enabled?: boolean
  menu_title?: string
  parent_menu_slug?: string
  menu_sort?: number
  frontend?: string
}

export interface ModuleInstallResult {
  menu_id: number
  menu_slug: string
  permission_ids: number[]
  manifest_path?: string
}

export interface CodeGeneratorPayload {
  module_name: string
  table_name: string
  fields: CodeGeneratorField[]
  files?: string[]
  file_type?: string
  force?: boolean
  options?: CodeGeneratorOptions
  install?: ModuleInstallConfig
}

export function getFieldTypes() {
  return request<{
    field_types: Array<{ value: string; label: string }>
    frontends: string[]
    ai_enabled?: boolean
  }>({
    url: '/code-generator/field-types',
    method: 'get',
  })
}

export function getTables() {
  return request<{ tables: string[] }>({
    url: '/code-generator/tables',
    method: 'get',
  })
}

export function getTableColumns(tableName: string) {
  return request<{ fields: CodeGeneratorField[] }>({
    url: '/code-generator/table-columns',
    method: 'get',
    params: { table_name: tableName },
  })
}

export function previewCode(data: CodeGeneratorPayload) {
  return request<{ code: string }>({
    url: '/code-generator/preview',
    method: 'post',
    data,
  })
}

export function saveCode(data: CodeGeneratorPayload) {
  return request<{ saved_files: string[]; install?: ModuleInstallResult }>({
    url: '/code-generator/save',
    method: 'post',
    data,
  })
}

export function installGeneratedModule(data: Pick<CodeGeneratorPayload, 'module_name' | 'table_name' | 'options' | 'install'>) {
  return request<{ install: ModuleInstallResult }>({
    url: '/code-generator/install-module',
    method: 'post',
    data,
  })
}

export function generateWithAI(data: { description: string }) {
  return request<{
    config: {
      module_name: string
      table_name: string
      fields: CodeGeneratorField[]
    }
  }>({
    url: '/code-generator/generate-with-ai',
    method: 'post',
    data,
    timeout: 300000,
  })
}
