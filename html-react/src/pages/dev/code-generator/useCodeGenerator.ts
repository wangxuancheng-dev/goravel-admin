import { useCallback, useEffect, useMemo, useState, createElement } from 'react'
import { App, Form } from 'antd'
import { useTranslation } from 'react-i18next'
import {
  generateWithAI,
  getFieldTypes,
  getTableColumns,
  getTables,
  installGeneratedModule,
  previewCode,
  saveCode,
  type CodeGeneratorField,
  type CodeGeneratorOptions,
  type ModuleInstallConfig,
} from '@/api/codeGenerator'
import { getDictionaryTypes } from '@/api/dictionary'
import { getMenuTree } from '@/api/menu'
import { useUserStore } from '@/stores/user'
import logger from '@/utils/logger'

const backendFileTypes = [
  { value: 'model', labelKey: 'file_model' },
  { value: 'controller', labelKey: 'file_controller' },
  { value: 'service', labelKey: 'file_service' },
  { value: 'request_create', labelKey: 'file_request_create' },
  { value: 'request_update', labelKey: 'file_request_update' },
  { value: 'export_job', labelKey: 'file_export_job' },
  { value: 'import_job', labelKey: 'file_import_job' },
] as const

const vueFileTypes = [
  { value: 'api', labelKey: 'file_api' },
  { value: 'list_page_config', labelKey: 'file_list_page_config' },
  { value: 'list_page', labelKey: 'file_list_page' },
  { value: 'form_page', labelKey: 'file_form_page' },
] as const

const reactFileTypes = [
  { value: 'react_api', labelKey: 'file_react_api' },
  { value: 'react_list_page', labelKey: 'file_react_list_page' },
  { value: 'react_list_page_config', labelKey: 'file_react_list_page_config' },
  { value: 'react_form_modal', labelKey: 'file_react_form_modal' },
] as const

function buildDefaultFiles(frontends: string[]) {
  const files = ['model', 'controller', 'service', 'request_create', 'request_update']
  if (frontends.includes('vue')) {
    files.push('api', 'list_page_config', 'list_page', 'form_page')
  }
  if (frontends.includes('react')) {
    files.push('react_api', 'react_list_page')
  }
  return files
}

function createDefaultFields(): CodeGeneratorField[] {
  return [
    {
      name: 'name',
      type: 'string',
      label: '名称',
      required: true,
      searchable: true,
      sortable: false,
      show_in_list: true,
      show_in_form: true,
      show_in_detail: true,
      is_primary_key: false,
      comment: '名称',
      search_type: 'like',
      search_ui_type: 'input',
      form_type: 'input',
      relation: null,
      dictionary: '',
      api_url: '',
    },
    {
      name: 'status',
      type: 'bool',
      label: '状态',
      required: false,
      searchable: true,
      sortable: false,
      show_in_list: true,
      show_in_form: true,
      show_in_detail: true,
      is_primary_key: false,
      comment: '状态',
      search_type: '=',
      search_ui_type: 'select',
      form_type: 'select',
      relation: null,
      dictionary: 'status',
      api_url: '',
    },
  ]
}

export function useCodeGenerator() {
  const { t } = useTranslation()
  const { message, modal } = App.useApp()
  const aiEnabledFromStore = useUserStore((s) => s.config.aiEnabled)

  const [form] = Form.useForm()
  const [generating, setGenerating] = useState(false)
  const [previewing, setPreviewing] = useState('')
  const [activeTab, setActiveTab] = useState('model')
  const [activeMode, setActiveMode] = useState('manual')
  const [fieldTypesRaw, setFieldTypesRaw] = useState<Array<{ value: string; label: string }>>([])
  const [dictionaryTypes, setDictionaryTypes] = useState<string[]>([])
  const [tables, setTables] = useState<string[]>([])
  const [selectedTable, setSelectedTable] = useState('')
  const [fields, setFields] = useState<CodeGeneratorField[]>(createDefaultFields)
  const [files, setFiles] = useState<string[]>(buildDefaultFiles(['vue', 'react']))
  const [options, setOptions] = useState<string[]>(['has_create', 'has_edit', 'has_delete', 'show_toolbar'])
  const [exportMode, setExportMode] = useState<'none' | 'sync' | 'async'>('none')
  const [importMode, setImportMode] = useState<'none' | 'sync' | 'async'>('none')
  const [installEnabled, setInstallEnabled] = useState(true)
  const [parentMenuSlug, setParentMenuSlug] = useState('')
  const [menuSort, setMenuSort] = useState(0)
  const [menuTitle, setMenuTitle] = useState('')
  const [parentMenuOptions, setParentMenuOptions] = useState<Array<{ value: string; label: string }>>([
    { value: '', label: '' },
  ])
  const [installing, setInstalling] = useState(false)
  const [previewCodeMap, setPreviewCodeMap] = useState<Record<string, string>>({})
  const [aiDescription, setAiDescription] = useState('')
  const [aiGenerating, setAiGenerating] = useState(false)
  const [aiGeneratedConfig, setAiGeneratedConfig] = useState<{
    module_name: string
    table_name: string
    fields: CodeGeneratorField[]
  } | null>(null)
  const [aiLastError, setAiLastError] = useState<string | null>(null)
  const [aiEnabled, setAiEnabled] = useState(false)
  const [enabledFrontends, setEnabledFrontends] = useState<string[]>(['vue', 'react'])
  const [relationDialogOpen, setRelationDialogOpen] = useState(false)
  const [fieldConfigDialogOpen, setFieldConfigDialogOpen] = useState(false)
  const [currentFieldIndex, setCurrentFieldIndex] = useState<number | null>(null)
  const [relationForm, setRelationForm] = useState({
    table: '',
    relation_type: 'belongsTo',
    foreign_key: '',
    display_field: '',
    is_tree: false,
  })
  const [fieldConfigForm, setFieldConfigForm] = useState({
    option_type: 'dictionary' as 'dictionary' | 'api',
    dictionary: '',
    api_url: '',
    is_tree: false,
    precision: 8,
    scale: 2,
  })

  const fieldTypes = useMemo(
    () =>
      fieldTypesRaw.map((type) => ({
        ...type,
        label: t(`code_generator.field_types.${type.value}`, type.label),
      })),
    [fieldTypesRaw, t],
  )

  const fileTypes = useMemo((): Array<{ value: string; label: string }> => {
    const types: Array<{ value: string; label: string }> = backendFileTypes.map((item) => ({
      value: item.value,
      label: t(`code_generator.${item.labelKey}`),
    }))
    if (enabledFrontends.includes('vue')) {
      types.push(
        ...vueFileTypes.map((item) => ({
          value: item.value,
          label: t(`code_generator.${item.labelKey}`),
        })),
      )
    }
    if (enabledFrontends.includes('react')) {
      types.push(
        ...reactFileTypes.map((item) => ({
          value: item.value,
          label: t(`code_generator.${item.labelKey}`),
        })),
      )
    }
    return types
  }, [enabledFrontends, t])

  const aiExamplePrompts = useMemo(
    () => [
      t('code_generator.ai_example_product'),
      t('code_generator.ai_example_article'),
      t('code_generator.ai_example_guestbook'),
    ],
    [t],
  )

  const formTypes = useMemo(
    () => [
      { value: 'input', label: t('code_generator.form_types.input') },
      { value: 'textarea', label: t('code_generator.form_types.textarea') },
      { value: 'editor', label: t('code_generator.form_types.editor') },
      { value: 'markdown', label: t('code_generator.form_types.markdown') },
      { value: 'image-upload', label: t('code_generator.form_types.image_upload') },
      { value: 'select', label: t('code_generator.form_types.select') },
      { value: 'radio', label: t('code_generator.form_types.radio') },
      { value: 'checkbox', label: t('code_generator.form_types.checkbox') },
      { value: 'switch', label: t('code_generator.form_types.switch') },
      { value: 'number', label: t('code_generator.form_types.number') },
      { value: 'date-picker', label: t('code_generator.form_types.date_picker') },
      { value: 'datetime-picker', label: t('code_generator.form_types.datetime_picker') },
    ],
    [t],
  )

  const buildGeneratorOptions = useCallback(
    (): CodeGeneratorOptions => ({
      has_export: exportMode !== 'none',
      export_async: exportMode === 'async',
      has_import: importMode !== 'none',
      import_async: importMode === 'async',
      has_create: options.includes('has_create'),
      has_edit: options.includes('has_edit'),
      has_delete: options.includes('has_delete'),
      enable_batch_actions: options.includes('enable_batch_actions'),
      show_toolbar: options.includes('show_toolbar'),
      is_tree_list: options.includes('is_tree_list'),
    }),
    [exportMode, importMode, options],
  )

  const buildInstallConfig = useCallback((): ModuleInstallConfig => {
    const values = form.getFieldsValue(['module_name', 'menu_title'])
    const resolvedTitle =
      menuTitle.trim() ||
      String(values.menu_title || '').trim() ||
      String(values.module_name || '').trim()
    return {
      enabled: installEnabled,
      menu_title: resolvedTitle,
      parent_menu_slug: parentMenuSlug,
      menu_sort: menuSort,
      frontend: enabledFrontends.includes('react') ? 'react' : 'vue',
    }
  }, [
    enabledFrontends,
    form,
    installEnabled,
    menuSort,
    menuTitle,
    parentMenuSlug,
  ])

  const flattenMenuOptions = useCallback((nodes: Array<Record<string, unknown>>, depth = 0): Array<{ value: string; label: string }> => {
    const result: Array<{ value: string; label: string }> = []
    nodes.forEach((node) => {
      const slug = String(node.slug || node.Slug || '')
      const title = String(node.title || node.Title || slug)
      if (slug) {
        result.push({ value: slug, label: `${'—'.repeat(depth)} ${title}`.trim() })
      }
      const children = (node.children || node.Children) as Array<Record<string, unknown>> | undefined
      if (Array.isArray(children) && children.length > 0) {
        result.push(...flattenMenuOptions(children, depth + 1))
      }
    })
    return result
  }, [])

  const loadParentMenus = useCallback(async () => {
    try {
      const response = await getMenuTree()
      const tree = Array.isArray(response.data) ? response.data : []
      setParentMenuOptions([
        { value: '', label: t('code_generator.parent_menu_top') },
        ...flattenMenuOptions(tree as Array<Record<string, unknown>>),
      ])
    } catch (error) {
      logger.error('Failed to load parent menus:', error)
    }
  }, [flattenMenuOptions, t])

  const loadFieldTypes = useCallback(async () => {
    try {
      const response = await getFieldTypes()
      const data = response.data
      setFieldTypesRaw(data?.field_types || [])
      const frontends = data?.frontends || ['vue', 'react']
      setEnabledFrontends(frontends)
      setFiles(buildDefaultFiles(frontends))
      setAiEnabled(data?.ai_enabled === true || aiEnabledFromStore)
    } catch (error) {
      logger.error('Failed to load field types:', error)
      message.error(t('code_generator.load_field_types_failed'))
      setAiEnabled(aiEnabledFromStore)
    }
  }, [aiEnabledFromStore, message, t])

  const loadDictionaryTypes = useCallback(async () => {
    try {
      const response = await getDictionaryTypes()
      setDictionaryTypes((response.data as { types?: string[] } | undefined)?.types || [])
    } catch (error) {
      logger.error('Failed to load dictionary types:', error)
    }
  }, [])

  const loadTables = useCallback(async () => {
    try {
      const response = await getTables()
      setTables(response.data?.tables || [])
    } catch (error) {
      logger.error('Failed to load tables:', error)
    }
  }, [])

  useEffect(() => {
    void loadFieldTypes()
    void loadTables()
    void loadParentMenus()
  }, [loadFieldTypes, loadTables, loadParentMenus])

  const handleTableChange = async (val: string) => {
    if (!val) return
    setSelectedTable(val)
    form.setFieldValue('table_name', val)
    let nextModuleName = val
    if (nextModuleName.endsWith('s')) {
      nextModuleName = nextModuleName.slice(0, -1)
    }
    form.setFieldValue('module_name', nextModuleName)
    if (!menuTitle.trim()) {
      setMenuTitle(nextModuleName)
      form.setFieldValue('menu_title', nextModuleName)
    }

    try {
      const response = await getTableColumns(val)
      const nextFields = (response.data?.fields || []).map((field: CodeGeneratorField) => {
        const fieldType = fieldTypesRaw.find((ft) => ft.value === field.db_type)
        if (fieldType) {
          field.type = fieldType.value
        } else if (field.db_type?.includes('int')) field.type = 'integer'
        else if (field.db_type?.includes('char') || field.db_type?.includes('text')) field.type = 'string'
        else if (field.db_type?.includes('date') || field.db_type?.includes('time')) field.type = 'datetime'
        else if (field.db_type?.includes('decimal') || field.db_type?.includes('float') || field.db_type?.includes('double'))
          field.type = 'decimal'
        else if (field.db_type?.includes('bool')) field.type = 'boolean'
        else if (field.db_type?.includes('json')) field.type = 'json'
        else field.type = 'string'
        return field
      })
      setFields(nextFields)
      message.success(t('code_generator.fields_loaded'))
    } catch (error) {
      logger.error('Failed to load columns:', error)
      message.error(t('code_generator.load_columns_failed'))
    }
  }

  const handleAddField = () => {
    setFields((prev) => [
      ...prev,
      {
        name: '',
        type: 'string',
        label: '',
        required: false,
        searchable: false,
        sortable: false,
        show_in_list: true,
        show_in_form: true,
        show_in_detail: true,
        is_primary_key: false,
        search_type: 'like',
        search_ui_type: 'input',
        form_type: 'input',
        relation: null,
        dictionary: '',
        api_url: '',
      },
    ])
  }

  const handleRemoveField = (index: number) => {
    setFields((prev) => prev.filter((_, i) => i !== index))
  }

  const updateField = (index: number, patch: Partial<CodeGeneratorField>) => {
    setFields((prev) => prev.map((field, i) => (i === index ? { ...field, ...patch } : field)))
  }

  const handleEditRelation = (index: number) => {
    const row = fields[index]
    setCurrentFieldIndex(index)
    if (row.relation) {
      setRelationForm({
        table: row.relation.table,
        relation_type: row.relation.relation_type,
        foreign_key: row.relation.foreign_key,
        display_field: row.relation.display_field,
        is_tree: row.relation.is_tree || false,
      })
    } else {
      setRelationForm({
        table: '',
        relation_type: 'belongsTo',
        foreign_key: '',
        display_field: '',
        is_tree: false,
      })
    }
    setRelationDialogOpen(true)
  }

  const handleSaveRelation = () => {
    if (!relationForm.table || !relationForm.foreign_key || !relationForm.display_field) {
      message.error(t('code_generator.relation_required'))
      return
    }
    if (currentFieldIndex == null) return
    updateField(currentFieldIndex, {
      relation: {
        table: relationForm.table,
        relation_type: relationForm.relation_type,
        foreign_key: relationForm.foreign_key,
        display_field: relationForm.display_field,
        alias: '',
        is_tree: relationForm.is_tree || false,
      },
    })
    setRelationDialogOpen(false)
    message.success(t('code_generator.relation_saved'))
  }

  const handleEditFieldConfig = (index: number) => {
    const row = fields[index]
    setCurrentFieldIndex(index)
    if (row.type === 'decimal') {
      setFieldConfigForm((prev) => ({
        ...prev,
        precision: row.precision || 8,
        scale: row.scale || 2,
      }))
    } else if (row.dictionary) {
      setFieldConfigForm({
        option_type: 'dictionary',
        dictionary: row.dictionary,
        api_url: '',
        is_tree: false,
        precision: 8,
        scale: 2,
      })
    } else if (row.api_url) {
      setFieldConfigForm({
        option_type: 'api',
        api_url: row.api_url,
        dictionary: '',
        is_tree: row.relation?.is_tree || false,
        precision: 8,
        scale: 2,
      })
    } else {
      setFieldConfigForm({
        option_type: 'dictionary',
        dictionary: '',
        api_url: '',
        is_tree: false,
        precision: 8,
        scale: 2,
      })
    }
    void loadDictionaryTypes()
    setFieldConfigDialogOpen(true)
  }

  const handleSaveFieldConfig = () => {
    if (currentFieldIndex == null) return
    const currentField = fields[currentFieldIndex]
    if (currentField.type === 'decimal') {
      updateField(currentFieldIndex, {
        precision: fieldConfigForm.precision,
        scale: fieldConfigForm.scale,
      })
    } else if (fieldConfigForm.option_type === 'dictionary') {
      updateField(currentFieldIndex, {
        dictionary: fieldConfigForm.dictionary,
        api_url: '',
        relation: currentField.relation ? { ...currentField.relation, is_tree: false } : currentField.relation,
      })
    } else {
      const relation = currentField.relation
        ? { ...currentField.relation, is_tree: fieldConfigForm.is_tree || false }
        : fieldConfigForm.is_tree
          ? {
              table: '',
              relation_type: 'belongsTo',
              foreign_key: '',
              display_field: '',
              is_tree: fieldConfigForm.is_tree || false,
            }
          : null
      updateField(currentFieldIndex, {
        dictionary: '',
        api_url: fieldConfigForm.api_url,
        relation,
      })
    }
    setFieldConfigDialogOpen(false)
    message.success(t('code_generator.field_config_saved'))
  }

  const handlePreview = async (fileType: string) => {
    setPreviewing(fileType)
    try {
      const values = await form.validateFields()
      const response = await previewCode({
        module_name: values.module_name,
        table_name: values.table_name,
        fields,
        file_type: fileType,
        options: buildGeneratorOptions(),
      })
      setPreviewCodeMap((prev) => ({ ...prev, [fileType]: String(response.data?.code || '') }))
    } catch (error) {
      logger.error('Failed to preview code:', error)
      message.error(t('code_generator.preview_failed'))
    } finally {
      setPreviewing('')
    }
  }

  const saveGeneratedCode = async (force: boolean) => {
    const values = await form.validateFields()
    const response = await saveCode({
      module_name: values.module_name,
      table_name: values.table_name,
      fields,
      files,
      force,
      options: buildGeneratorOptions(),
      install: buildInstallConfig(),
    })
    const savedFiles = response.data?.saved_files || []
    if (response.data?.install) {
      message.success(t('code_generator.save_and_install_success', { count: savedFiles.length }))
    } else {
      message.success(t('code_generator.save_success', { count: savedFiles.length }))
    }
  }

  const handleInstallModule = async () => {
    try {
      const values = await form.validateFields()
      setInstalling(true)
      await installGeneratedModule({
        module_name: values.module_name,
        table_name: values.table_name,
        options: buildGeneratorOptions(),
        install: { ...buildInstallConfig(), enabled: true },
      })
      message.success(t('code_generator.install_success'))
    } catch (error) {
      logger.error('Failed to install module:', error)
      message.error(t('code_generator.install_failed'))
    } finally {
      setInstalling(false)
    }
  }

  const formatExistingFilesHtml = (files: string[]) =>
    files.map((file) => `<div style="padding-left:12px;font-family:monospace;font-size:12px;line-height:1.8;word-break:break-all">${file}</div>`).join('')

  const buildFilesExistConfirmContent = (existingFiles: string[]) =>
    createElement('div', {
      dangerouslySetInnerHTML: {
        __html: t('code_generator.files_exist_confirm', {
          count: existingFiles.length,
          files: formatExistingFilesHtml(existingFiles),
        }),
      },
    })

  const handleGenerate = async () => {
    try {
      await form.validateFields()
    } catch {
      return
    }

    modal.confirm({
      title: t('common.confirm'),
      content: t('code_generator.generate_confirm'),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        setGenerating(true)
        try {
          await saveGeneratedCode(false)
        } catch (error: unknown) {
          const err = error as {
            response?: { status?: number; data?: { error_code?: string; files?: string[] } }
          }
          if (err.response?.status === 409 && err.response.data?.error_code === 'files_exist') {
            const existingFiles = err.response.data.files || []
            modal.confirm({
              title: t('common.warning'),
              content: buildFilesExistConfirmContent(existingFiles),
              okText: t('code_generator.overwrite'),
              cancelText: t('common.cancel'),
              onOk: async () => {
                setGenerating(true)
                try {
                  await saveGeneratedCode(true)
                } catch (innerError) {
                  logger.error('Failed to overwrite code:', innerError)
                  message.error(t('code_generator.generate_failed'))
                } finally {
                  setGenerating(false)
                }
              },
            })
          } else {
            logger.error('Failed to generate code:', error)
            message.error(t('code_generator.generate_failed'))
          }
        } finally {
          setGenerating(false)
        }
      },
    })
  }

  const handleGenerateWithAI = async () => {
    if (!aiDescription.trim()) {
      message.warning(t('code_generator.ai_description_required'))
      return
    }
    try {
      setAiGenerating(true)
      setAiLastError(null)
      const response = await generateWithAI({ description: aiDescription })
      if (response.data?.config) {
        setAiGeneratedConfig(response.data.config)
        message.success(t('code_generator.ai_generate_success'))
      } else {
        setAiLastError(t('code_generator.ai_generate_failed'))
        message.error(t('code_generator.ai_generate_failed'))
      }
    } catch (error: unknown) {
      logger.error('Failed to generate with AI:', error)
      const err = error as { response?: { data?: { message?: string } }; message?: string }
      const errorMessage = err.response?.data?.message || err.message || t('code_generator.ai_generate_failed')
      setAiLastError(errorMessage)
      message.error(errorMessage)
    } finally {
      setAiGenerating(false)
    }
  }

  const handleApplyAIConfig = () => {
    if (!aiGeneratedConfig) return

    form.setFieldsValue({
      module_name: aiGeneratedConfig.module_name || '',
      table_name: aiGeneratedConfig.table_name || '',
    })

    const mappedFields = (aiGeneratedConfig.fields || []).map((field) => {
      const dbType = field.db_type || field.type || 'string'
      let type = dbType
      const typeExists = fieldTypes.some((ft) => ft.value === type)
      if (!typeExists) {
        if (dbType.includes('bigint')) type = 'bigInteger'
        else if (dbType.includes('int')) type = 'integer'
        else if (dbType.includes('char') || dbType.includes('text')) type = 'string'
        else if (dbType.includes('date') || dbType.includes('time')) type = 'datetime'
        else if (dbType.includes('decimal') || dbType.includes('float') || dbType.includes('double')) type = 'decimal'
        else if (dbType.includes('bool')) type = 'boolean'
        else if (dbType.includes('json')) type = 'json'
        else type = 'string'
      }

      let formType = field.form_type
      if (!formType) {
        if (dbType === 'text' || field.name?.includes('content') || field.name?.includes('description')) {
          formType = 'textarea'
        } else if (dbType === 'date') formType = 'date-picker'
        else if (dbType === 'datetime' || dbType === 'timestamp') formType = 'datetime-picker'
        else if (dbType === 'boolean') formType = 'switch'
        else if (dbType === 'json') formType = 'textarea'
        else formType = 'input'
      }

      return {
        ...field,
        type,
        label: field.label || field.name || '',
        form_type: formType,
        dictionary: field.dictionary || '',
        api_url: field.api_url || '',
      } as CodeGeneratorField
    })

    setFields(mappedFields)
    setActiveMode('manual')
    message.success(t('code_generator.ai_config_applied'))
  }

  return {
    form,
    generating,
    previewing,
    activeTab,
    setActiveTab,
    activeMode,
    setActiveMode,
    dictionaryTypes,
    tables,
    selectedTable,
    setSelectedTable,
    aiDescription,
    setAiDescription,
    aiGenerating,
    aiGeneratedConfig,
    aiLastError,
    aiEnabled,
    aiExamplePrompts,
    fieldTypes,
    previewCodeMap,
    relationDialogOpen,
    setRelationDialogOpen,
    fieldConfigDialogOpen,
    setFieldConfigDialogOpen,
    currentFieldIndex,
    relationForm,
    setRelationForm,
    fieldConfigForm,
    setFieldConfigForm,
    fields,
    files,
    setFiles,
    options,
    setOptions,
    exportMode,
    setExportMode,
    importMode,
    setImportMode,
    installEnabled,
    setInstallEnabled,
    parentMenuSlug,
    setParentMenuSlug,
    parentMenuOptions,
    menuSort,
    setMenuSort,
    menuTitle,
    setMenuTitle,
    installing,
    handleInstallModule,
    fileTypes,
    formTypes,
    handleTableChange,
    handleAddField,
    handleRemoveField,
    updateField,
    handleEditRelation,
    handleSaveRelation,
    handleEditFieldConfig,
    handleSaveFieldConfig,
    handlePreview,
    handleGenerate,
    handleGenerateWithAI,
    handleApplyAIConfig,
    applyAIExample: setAiDescription,
  }
}
