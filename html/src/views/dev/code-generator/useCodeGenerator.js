import { ref, reactive, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Document, Delete, Setting } from '@element-plus/icons-vue'
import { getFieldTypes, getTables, getTableColumns, previewCode as previewCodeApi, generateCode, saveCode, installGeneratedModule, generateWithAI } from '../../../api/codeGenerator'
import { getDictionaryTypes } from '../../../api/dictionary'
import { getMenuTree } from '../../../api/menu'
import { isDev } from '../../../utils/env'
import { useUserStore } from '../../../store/user'
import logger from '../../../utils/logger'
import {
  ALL_FORM_TYPE_VALUES,
  allowedFormTypes,
  allowedSearchUiTypes,
  normalizeFieldControls,
} from './fieldFormOptions'

export function useCodeGenerator() {
  const { t } = useI18n()
  const userStore = useUserStore()

  if (!isDev()) {
    ElMessage.error(t('code_generator.dev_only'))
  }

  const formRef = ref(null)
  const generating = ref(false)
  const previewing = ref('')
  const activeTab = ref('model')
  const activeMode = ref('manual')
  const fieldTypesRaw = ref([])
  const dictionaryTypes = ref([])
  const tables = ref([])
  const selectedTable = ref('')
  const aiDescription = ref('')
  const aiGenerating = ref(false)
  const aiGeneratedConfig = ref(null)
  const aiLastError = ref(null)
  const aiEnabled = ref(false)
  const enabledFrontends = ref(['vue', 'react'])
  const installing = ref(false)
  const parentMenuOptions = ref([])

  const backendFileTypes = [
    { value: 'model', labelKey: 'file_model' },
    { value: 'controller', labelKey: 'file_controller' },
    { value: 'service', labelKey: 'file_service' },
    { value: 'request_create', labelKey: 'file_request_create' },
    { value: 'request_update', labelKey: 'file_request_update' },
    { value: 'export_job', labelKey: 'file_export_job' },
    { value: 'import_job', labelKey: 'file_import_job' },
  ]

  const vueFileTypes = [
    { value: 'api', labelKey: 'file_api' },
    { value: 'list_page_config', labelKey: 'file_list_page_config' },
    { value: 'list_page', labelKey: 'file_list_page' },
    { value: 'form_page', labelKey: 'file_form_page' },
  ]

  const reactFileTypes = [
    { value: 'react_api', labelKey: 'file_react_api' },
    { value: 'react_list_page', labelKey: 'file_react_list_page' },
    { value: 'react_list_page_config', labelKey: 'file_react_list_page_config' },
    { value: 'react_form_modal', labelKey: 'file_react_form_modal' },
  ]

  const buildDefaultFiles = (frontends) => {
    const files = ['model', 'controller', 'service', 'request_create', 'request_update']
    if (frontends.includes('vue')) {
      files.push('api', 'list_page_config', 'list_page', 'form_page')
    }
    if (frontends.includes('react')) {
      files.push('react_api', 'react_list_page')
    }
    return files
  }

  const aiExamplePrompts = computed(() => [
    t('code_generator.ai_example_product'),
    t('code_generator.ai_example_article'),
    t('code_generator.ai_example_guestbook'),
  ])

  const fieldTypes = computed(() => {
    return fieldTypesRaw.value.map(type => ({
      ...type,
      label: t(`code_generator.field_types.${type.value}`, type.label)
    }))
  })

  const previewCode = reactive({})
  const relationDialogVisible = ref(false)
  const fieldConfigDialogVisible = ref(false)
  const currentField = ref(null)
  const relationForm = reactive({
    table: '',
    relation_type: 'belongsTo',
    foreign_key: '',
    display_field: '',
    is_tree: false
  })
  const fieldConfigForm = reactive({
    option_type: 'dictionary',
    dictionary: '',
    api_url: '',
    is_tree: false,
    precision: 8,
    scale: 2
  })

  const form = reactive({
    module_name: '',
    table_name: '',
    menu_title: '',
    parent_menu_slug: '',
    menu_sort: 0,
    install_enabled: true,
    files: buildDefaultFiles(['vue', 'react']),
    options: ['has_create', 'has_edit', 'has_delete', 'show_toolbar'],
    export_mode: 'none',
    import_mode: 'none',
    fields: [
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
        api_url: ''
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
        api_url: ''
      }
    ]
  })

  const rules = {
    module_name: [
      { required: true, message: t('code_generator.module_name_required'), trigger: 'blur' }
    ],
    table_name: [
      { required: true, message: t('code_generator.table_name_required'), trigger: 'blur' }
    ]
  }

  const buildInstallConfig = () => ({
    enabled: form.install_enabled,
    menu_title: (form.menu_title || form.module_name || '').trim(),
    parent_menu_slug: form.parent_menu_slug || '',
    menu_sort: Number(form.menu_sort) || 0,
    frontend: enabledFrontends.value.includes('vue') ? 'vue' : 'react',
  })

  const flattenMenuOptions = (nodes, depth = 0) => {
    const result = []
    ;(nodes || []).forEach((node) => {
      const slug = String(node.slug || node.Slug || '')
      const title = String(node.title || node.Title || slug)
      if (slug) {
        result.push({ value: slug, label: `${'—'.repeat(depth)} ${title}`.trim() })
      }
      const children = node.children || node.Children
      if (Array.isArray(children) && children.length > 0) {
        result.push(...flattenMenuOptions(children, depth + 1))
      }
    })
    return result
  }

  const loadParentMenus = async () => {
    try {
      const response = await getMenuTree()
      const tree = Array.isArray(response.data) ? response.data : []
      parentMenuOptions.value = [
        { value: '', label: t('code_generator.parent_menu_top') },
        ...flattenMenuOptions(tree),
      ]
    } catch (error) {
      logger.error('Failed to load parent menus:', error)
    }
  }

  const buildSavePayload = (force = false) => ({
    module_name: form.module_name,
    table_name: form.table_name,
    fields: form.fields.map((field) => normalizeFieldControls(field)),
    files: form.files,
    force,
    options: buildGeneratorOptions(),
    install: buildInstallConfig(),
  })

  const saveGeneratedCode = async (force = false) => {
    const response = await saveCode(buildSavePayload(force))
    const savedFiles = response.data?.saved_files || []
    if (response.data?.install) {
      ElMessage.success(t('code_generator.save_and_install_success', { count: savedFiles.length }))
    } else {
      ElMessage.success(t('code_generator.save_success', { count: savedFiles.length }))
    }
  }

  const handleInstallModule = async () => {
    if (!formRef.value) return
    try {
      await formRef.value.validate()
    } catch {
      return
    }

    try {
      installing.value = true
      await installGeneratedModule({
        module_name: form.module_name,
        table_name: form.table_name,
        options: buildGeneratorOptions(),
        install: { ...buildInstallConfig(), enabled: true },
      })
      ElMessage.success(t('code_generator.install_success'))
    } catch (error) {
      logger.error('Failed to install module:', error)
      ElMessage.error(t('code_generator.install_failed'))
    } finally {
      installing.value = false
    }
  }

  const buildGeneratorOptions = () => ({
    has_export: form.export_mode !== 'none',
    export_async: form.export_mode === 'async',
    has_import: form.import_mode !== 'none',
    import_async: form.import_mode === 'async',
    has_create: form.options.includes('has_create'),
    has_edit: form.options.includes('has_edit'),
    has_delete: form.options.includes('has_delete'),
    enable_batch_actions: form.options.includes('enable_batch_actions'),
    show_toolbar: form.options.includes('show_toolbar'),
    is_tree_list: form.options.includes('is_tree_list')
  })

  const fileTypes = computed(() => {
    const types = backendFileTypes.map((item) => ({
      value: item.value,
      label: t(`code_generator.${item.labelKey}`),
    }))
    if (enabledFrontends.value.includes('vue')) {
      types.push(...vueFileTypes.map((item) => ({
        value: item.value,
        label: t(`code_generator.${item.labelKey}`),
      })))
    }
    if (enabledFrontends.value.includes('react')) {
      types.push(...reactFileTypes.map((item) => ({
        value: item.value,
        label: t(`code_generator.${item.labelKey}`),
      })))
    }
    return types
  })

  const formTypeLabel = (value) => t(`code_generator.form_types.${String(value).replace(/-/g, '_')}`, value)

  const formTypes = ALL_FORM_TYPE_VALUES.map((value) => ({
    value,
    label: formTypeLabel(value),
  }))

  const formTypesForField = (dbType) =>
    allowedFormTypes(dbType || 'string').map((value) => ({
      value,
      label: formTypeLabel(value),
    }))

  const searchUiTypesForField = (dbType) =>
    allowedSearchUiTypes(dbType || 'string').map((value) => ({
      value,
      label: t(`code_generator.search_ui_types.${value}`, value),
    }))

  const applyFieldTypeChange = (row) => {
    Object.assign(row, normalizeFieldControls(row))
  }

  const loadFieldTypes = async () => {
    try {
      const response = await getFieldTypes()
      fieldTypesRaw.value = response.data.field_types || []
      const frontends = response.data.frontends || ['vue', 'react']
      enabledFrontends.value = frontends
      form.files = buildDefaultFiles(frontends)
      aiEnabled.value = response.data.ai_enabled === true || userStore.config.aiEnabled
    } catch (error) {
      logger.error('Failed to load field types:', error)
      ElMessage.error(t('code_generator.load_field_types_failed'))
      aiEnabled.value = userStore.config.aiEnabled
    }
  }

  const applyAIExample = (prompt) => {
    aiDescription.value = prompt
  }

  const loadDictionaryTypes = async () => {
    try {
      const response = await getDictionaryTypes()
      dictionaryTypes.value = response.data.types || []
    } catch (error) {
      logger.error('Failed to load dictionary types:', error)
    }
  }

  const loadTables = async () => {
    try {
      const response = await getTables()
      tables.value = response.data.tables || []
    } catch (error) {
      logger.error('Failed to load tables:', error)
    }
  }

  const handleTableChange = async (val) => {
    if (!val) return
    form.table_name = val
    let moduleName = val
    if (moduleName.endsWith('s')) {
      moduleName = moduleName.slice(0, -1)
    }
    form.module_name = moduleName
    if (!form.menu_title) {
      form.menu_title = moduleName
    }

    try {
      const response = await getTableColumns(val)
      form.fields = (response.data.fields || []).map(field => {
        const fieldType = fieldTypesRaw.value.find(ft => ft.value === field.db_type)
        if (fieldType) {
          field.type = fieldType.value
        } else {
          if (field.db_type.includes('int')) field.type = 'integer'
          else if (field.db_type.includes('char') || field.db_type.includes('text')) field.type = 'string'
          else if (field.db_type.includes('date') || field.db_type.includes('time')) field.type = 'datetime'
          else if (field.db_type.includes('decimal') || field.db_type.includes('float') || field.db_type.includes('double')) field.type = 'decimal'
          else if (field.db_type.includes('bool')) field.type = 'boolean'
          else if (field.db_type.includes('json')) field.type = 'json'
          else field.type = 'string'
        }
        return normalizeFieldControls(field)
      })
      ElMessage.success(t('code_generator.fields_loaded'))
    } catch (error) {
      logger.error('Failed to load columns:', error)
      ElMessage.error(t('code_generator.load_columns_failed'))
    }
  }

  const handleAddField = () => {
    form.fields.push({
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
      api_url: ''
    })
  }

  const handleRemoveField = (index) => {
    form.fields.splice(index, 1)
  }

  const handleEditRelation = (row) => {
    currentField.value = row
    if (row.relation) {
      relationForm.table = row.relation.table
      relationForm.relation_type = row.relation.relation_type
      relationForm.foreign_key = row.relation.foreign_key
      relationForm.display_field = row.relation.display_field
      relationForm.is_tree = row.relation.is_tree || false
    } else {
      relationForm.table = ''
      relationForm.relation_type = 'belongsTo'
      relationForm.foreign_key = ''
      relationForm.display_field = ''
      relationForm.is_tree = false
    }
    relationDialogVisible.value = true
  }

  const handleSaveRelation = () => {
    if (!relationForm.table || !relationForm.foreign_key || !relationForm.display_field) {
      ElMessage.error(t('code_generator.relation_required'))
      return
    }
    currentField.value.relation = {
      table: relationForm.table,
      relation_type: relationForm.relation_type,
      foreign_key: relationForm.foreign_key,
      display_field: relationForm.display_field,
      alias: '',
      is_tree: relationForm.is_tree || false
    }
    relationDialogVisible.value = false
    ElMessage.success(t('code_generator.relation_saved'))
  }

  const handleEditFieldConfig = (row) => {
    currentField.value = row
    if (row.type === 'decimal') {
      fieldConfigForm.precision = row.precision || 8
      fieldConfigForm.scale = row.scale || 2
    } else if (row.dictionary) {
      fieldConfigForm.option_type = 'dictionary'
      fieldConfigForm.dictionary = row.dictionary
      fieldConfigForm.api_url = ''
      fieldConfigForm.is_tree = false
    } else if (row.api_url) {
      fieldConfigForm.option_type = 'api'
      fieldConfigForm.api_url = row.api_url
      fieldConfigForm.dictionary = ''
      fieldConfigForm.is_tree = (row.relation && row.relation.is_tree) || false
    } else {
      fieldConfigForm.option_type = 'dictionary'
      fieldConfigForm.dictionary = ''
      fieldConfigForm.api_url = ''
      fieldConfigForm.is_tree = false
    }
    loadDictionaryTypes()
    fieldConfigDialogVisible.value = true
  }

  const handleSaveFieldConfig = () => {
    if (currentField.value) {
      if (currentField.value.type === 'decimal') {
        currentField.value.precision = fieldConfigForm.precision
        currentField.value.scale = fieldConfigForm.scale
      } else if (fieldConfigForm.option_type === 'dictionary') {
        currentField.value.dictionary = fieldConfigForm.dictionary
        currentField.value.api_url = ''
        if (currentField.value.relation) {
          currentField.value.relation.is_tree = false
        }
      } else {
        currentField.value.dictionary = ''
        currentField.value.api_url = fieldConfigForm.api_url
        if (currentField.value.relation) {
          currentField.value.relation.is_tree = fieldConfigForm.is_tree || false
        } else if (fieldConfigForm.is_tree) {
          if (!currentField.value.relation) {
            currentField.value.relation = {
              table: '',
              relation_type: 'belongsTo',
              foreign_key: '',
              display_field: '',
              is_tree: fieldConfigForm.is_tree || false
            }
          }
        }
      }
    }
    fieldConfigDialogVisible.value = false
    ElMessage.success(t('code_generator.field_config_saved'))
  }

  const handlePreview = async (fileType) => {
    previewing.value = fileType
    try {
      const response = await previewCodeApi({
        module_name: form.module_name,
        table_name: form.table_name,
        fields: form.fields.map((field) => normalizeFieldControls(field)),
        file_type: fileType,
        options: buildGeneratorOptions()
      })
      previewCode[fileType] = response.data.code || ''
    } catch (error) {
      logger.error('Failed to preview code:', error)
      ElMessage.error(t('code_generator.preview_failed'))
    } finally {
      previewing.value = ''
    }
  }

  const handleGenerate = async () => {
    if (!formRef.value) return

    try {
      await formRef.value.validate()
    } catch (error) {
      return
    }

    try {
      await ElMessageBox.confirm(
        t('code_generator.generate_confirm'),
        t('common.confirm'),
        {
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
      )

      generating.value = true
      await saveGeneratedCode(false)
    } catch (error) {
      if (error !== 'cancel' && error !== 'close') {
        if (error.response && error.response.status === 409 && error.response.data && error.response.data.error_code === 'files_exist') {
          const existingFiles = error.response.data.files || []
          try {
            await ElMessageBox.confirm(
              t('code_generator.files_exist_confirm', {
                count: existingFiles.length,
                files: existingFiles
                  .map(
                    (f) =>
                      `<div style="padding-left:12px;font-family:monospace;font-size:12px;line-height:1.8;word-break:break-all">${f}</div>`
                  )
                  .join(''),
              }),
              t('common.warning'),
              {
                confirmButtonText: t('code_generator.overwrite'),
                cancelButtonText: t('common.cancel'),
                type: 'warning',
                dangerouslyUseHTMLString: true
              }
            )

            await saveGeneratedCode(true)
          } catch (innerError) {
            if (innerError !== 'cancel' && innerError !== 'close') {
              logger.error('Failed to overwrite code:', innerError)
              ElMessage.error(t('code_generator.generate_failed'))
            }
          }
        } else {
          logger.error('Failed to generate code:', error)
          ElMessage.error(t('code_generator.generate_failed'))
        }
      }
    } finally {
      generating.value = false
    }
  }

  const handleGenerateWithAI = async () => {
    if (!aiDescription.value || aiDescription.value.trim() === '') {
      ElMessage.warning(t('code_generator.ai_description_required'))
      return
    }

    try {
      aiGenerating.value = true
      aiLastError.value = null
      const response = await generateWithAI({
        description: aiDescription.value
      })

      if (response.data && response.data.config) {
        aiGeneratedConfig.value = response.data.config
        ElMessage.success(t('code_generator.ai_generate_success'))
      } else {
        aiLastError.value = t('code_generator.ai_generate_failed')
        ElMessage.error(t('code_generator.ai_generate_failed'))
      }
    } catch (error) {
      logger.error('Failed to generate with AI:', error)
      const errorMessage = error.response?.data?.message || error.message || t('code_generator.ai_generate_failed')
      aiLastError.value = errorMessage
      ElMessage.error(errorMessage)
    } finally {
      aiGenerating.value = false
    }
  }

  const handleApplyAIConfig = () => {
    if (!aiGeneratedConfig.value) {
      return
    }

    form.module_name = aiGeneratedConfig.value.module_name || ''
    form.table_name = aiGeneratedConfig.value.table_name || ''
    if (!form.menu_title) {
      form.menu_title = form.module_name
    }

    const fields = (aiGeneratedConfig.value.fields || []).map(field => {
      const dbType = field.db_type || field.type || 'string'

      let type = dbType
      const typeExists = fieldTypes.value.some(ft => ft.value === type)
      if (!typeExists) {
        if (dbType.includes('bigint')) {
          type = 'bigInteger'
        } else if (dbType.includes('int')) {
          type = 'integer'
        } else if (dbType.includes('char') || dbType.includes('text')) {
          type = 'string'
        } else if (dbType.includes('date') || dbType.includes('time')) {
          type = 'datetime'
        } else if (dbType.includes('decimal') || dbType.includes('float') || dbType.includes('double')) {
          type = 'decimal'
        } else if (dbType.includes('bool')) {
          type = 'boolean'
        } else if (dbType.includes('json')) {
          type = 'json'
        } else {
          type = 'string'
        }
      }

      let formType = field.form_type
      if (!formType) {
        if (dbType === 'text' || (field.name && (field.name.includes('content') || field.name.includes('description') || field.name.includes('detail')))) {
          formType = 'textarea'
        } else if (dbType === 'date') {
          formType = 'date-picker'
        } else if (dbType === 'datetime' || dbType === 'timestamp') {
          formType = 'datetime-picker'
        } else if (dbType === 'boolean') {
          formType = 'switch'
        } else if (dbType === 'json') {
          formType = 'textarea'
        } else {
          formType = 'input'
        }
      }

      let searchUIType = field.search_ui_type
      if (!searchUIType) {
        const isAtSuffixField = field.name && (field.name === 'created_at' || field.name === 'updated_at' || field.name.endsWith('_at'))
        if (isAtSuffixField) {
          searchUIType = dbType === 'date' ? 'daterange' : 'datetimerange'
        } else if (dbType === 'date') {
          searchUIType = 'date'
        } else if (dbType === 'datetime' || dbType === 'timestamp') {
          searchUIType = 'datetime'
        } else if (dbType === 'boolean') {
          searchUIType = 'select'
        } else {
          searchUIType = 'input'
        }
      }

      const searchType = field.search_type || (dbType === 'string' || dbType === 'text' ? 'like' : '=')

      let relation = field.relation || null
      if (relation && typeof relation === 'object') {
        relation = {
          table: relation.table || '',
          relation_type: relation.relation_type || 'belongsTo',
          foreign_key: relation.foreign_key || '',
          display_field: relation.display_field || '',
          alias: relation.alias || '',
          is_tree: relation.is_tree || false
        }
        if (!relation.table) {
          relation = null
        }
      }

      let precision = field.precision
      let scale = field.scale
      if (type === 'decimal') {
        if (!precision || precision === 0) {
          precision = 8
        }
        if (scale === undefined || scale === null) {
          scale = 2
        }
      }

      const mappedField = {
        ...field,
        type: type,
        label: field.label || field.name || '',
        required: field.required !== undefined ? field.required : false,
        searchable: field.searchable !== undefined ? field.searchable : true,
        sortable: field.sortable !== undefined ? field.sortable : false,
        show_in_list: field.show_in_list !== undefined ? field.show_in_list : true,
        show_in_form: field.show_in_form !== undefined ? field.show_in_form : true,
        show_in_detail: field.show_in_detail !== undefined ? field.show_in_detail : true,
        search_type: searchType,
        search_ui_type: searchUIType,
        form_type: formType,
        relation: relation,
        dictionary: field.dictionary || '',
        api_url: field.api_url || '',
        precision: precision,
        scale: scale
      }

      return normalizeFieldControls(mappedField)
    })

    form.fields = fields
    activeMode.value = 'manual'
    ElMessage.success(t('code_generator.ai_config_applied'))
  }

  onMounted(() => {
    loadFieldTypes()
    loadTables()
    loadParentMenus()
  })

  return {
    Plus,
    Document,
    Delete,
    Setting,
    formRef,
    generating,
    previewing,
    activeTab,
    activeMode,
    dictionaryTypes,
    tables,
    selectedTable,
    aiDescription,
    aiGenerating,
    aiGeneratedConfig,
    aiLastError,
    aiEnabled,
    aiExamplePrompts,
    fieldTypes,
    previewCode,
    relationDialogVisible,
    fieldConfigDialogVisible,
    currentField,
    relationForm,
    fieldConfigForm,
    form,
    rules,
    fileTypes,
    formTypes,
    formTypesForField,
    searchUiTypesForField,
    applyFieldTypeChange,
    handleTableChange,
    handleAddField,
    handleRemoveField,
    handleEditRelation,
    handleSaveRelation,
    handleEditFieldConfig,
    handleSaveFieldConfig,
    handlePreview,
    handleGenerate,
    handleInstallModule,
    installing,
    parentMenuOptions,
    applyAIExample,
    handleGenerateWithAI,
    handleApplyAIConfig,
  }
}
