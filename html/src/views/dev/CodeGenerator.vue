<template>
  <div class="code-generator">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ $t('code_generator.title') }}</span>
          <div v-if="activeMode === 'manual'" class="card-header-actions">
            <el-button :loading="installing" @click="handleInstallModule">
              {{ $t('code_generator.install_module') }}
            </el-button>
            <el-button type="primary" :loading="generating" @click="handleGenerate">
              <el-icon><Document /></el-icon>
              {{ $t('code_generator.generate') }}
            </el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activeMode" type="border-card">
        <!-- 手动配置标签页 -->
        <el-tab-pane :label="$t('code_generator.manual_mode')" name="manual">
          <el-form :model="form" :rules="rules" ref="formRef" label-width="120px">
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item :label="$t('code_generator.select_table')">
                  <el-select 
                    v-model="selectedTable" 
                    filterable 
                    clearable 
                    @change="handleTableChange" 
                    :placeholder="$t('code_generator.select_table_placeholder')" 
                    style="width: 100%"
                  >
                    <el-option v-for="table in tables" :key="table" :label="table" :value="table" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item :label="$t('code_generator.module_name')" prop="module_name">
                  <el-input v-model="form.module_name" :placeholder="$t('code_generator.module_name_placeholder')" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item :label="$t('code_generator.table_name')" prop="table_name">
                  <el-input v-model="form.table_name" :placeholder="$t('code_generator.table_name_placeholder')" />
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item :label="$t('code_generator.menu_title')">
                  <el-input v-model="form.menu_title" :placeholder="$t('code_generator.menu_title_placeholder')" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item :label="$t('code_generator.parent_menu')">
                  <el-select
                    v-model="form.parent_menu_slug"
                    filterable
                    clearable
                    :placeholder="$t('code_generator.parent_menu_placeholder')"
                    style="width: 100%"
                  >
                    <el-option
                      v-for="item in parentMenuOptions"
                      :key="item.value || 'top'"
                      :label="item.label"
                      :value="item.value"
                    />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item :label="$t('code_generator.menu_sort')">
                  <el-input-number v-model="form.menu_sort" :min="0" style="width: 100%" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item :label="$t('code_generator.install_options')">
                  <el-checkbox v-model="form.install_enabled">{{ $t('code_generator.install_on_save') }}</el-checkbox>
                </el-form-item>
              </el-col>
            </el-row>

            <el-divider>{{ $t('code_generator.fields_config') }}</el-divider>

            <el-form-item :label="$t('code_generator.generated_files')">
              <el-checkbox-group v-model="form.files">
                <el-checkbox v-for="fileType in fileTypes" :key="fileType.value" :label="fileType.value">
                  {{ fileType.label }}
                </el-checkbox>
              </el-checkbox-group>
            </el-form-item>

            <el-form-item :label="$t('code_generator.function_options')">
              <el-checkbox-group v-model="form.options">
                <el-checkbox label="has_create">{{ $t('code_generator.has_create') }}</el-checkbox>
                <el-checkbox label="has_edit">{{ $t('code_generator.has_edit') }}</el-checkbox>
                <el-checkbox label="has_delete">{{ $t('code_generator.has_delete') }}</el-checkbox>
                <el-checkbox label="enable_batch_actions">{{ $t('code_generator.enable_batch_actions') }}</el-checkbox>
                <el-checkbox label="show_toolbar">{{ $t('code_generator.show_toolbar') }}</el-checkbox>
                <el-checkbox label="is_tree_list">{{ $t('code_generator.is_tree_list') }}</el-checkbox>
              </el-checkbox-group>
            </el-form-item>

            <el-form-item :label="$t('code_generator.export_mode')">
              <el-radio-group v-model="form.export_mode">
                <el-radio label="none">{{ $t('code_generator.export_mode_none') }}</el-radio>
                <el-radio label="sync">{{ $t('code_generator.export_mode_sync') }}</el-radio>
                <el-radio label="async">{{ $t('code_generator.export_mode_async') }}</el-radio>
              </el-radio-group>
            </el-form-item>

            <el-form-item :label="$t('code_generator.import_mode')">
              <el-radio-group v-model="form.import_mode">
                <el-radio label="none">{{ $t('code_generator.import_mode_none') }}</el-radio>
                <el-radio label="sync">{{ $t('code_generator.import_mode_sync') }}</el-radio>
                <el-radio label="async">{{ $t('code_generator.import_mode_async') }}</el-radio>
              </el-radio-group>
            </el-form-item>

            <el-button type="primary" @click="handleAddField">
              <el-icon><Plus /></el-icon>
              {{ $t('code_generator.add_field') }}
            </el-button>

            <el-table :data="form.fields" border style="margin-top: 20px">
              <el-table-column type="index" :label="$t('table.index')" width="60" />
              <el-table-column prop="name" :label="$t('code_generator.field_name')" width="150">
                <template #default="{ row }">
                  <el-input v-model="row.name" :placeholder="$t('code_generator.field_name_placeholder')" />
                </template>
              </el-table-column>
              <el-table-column prop="type" :label="$t('code_generator.field_type')" width="150">
                <template #default="{ row }">
                  <el-select v-model="row.type" :placeholder="$t('common.select')" @change="applyFieldTypeChange(row)">
                    <el-option
                      v-for="type in fieldTypes"
                      :key="type.value"
                      :label="type.label"
                      :value="type.value"
                    />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column prop="label" :label="$t('code_generator.field_label')" width="150">
                <template #default="{ row }">
                  <el-input v-model="row.label" :placeholder="$t('code_generator.field_label_placeholder')" />
                </template>
              </el-table-column>
              <el-table-column prop="form_type" :label="$t('code_generator.form_type')" width="150">
                <template #default="{ row }">
                  <el-select v-model="row.form_type" :placeholder="$t('common.select')">
                    <el-option
                      v-for="type in formTypesForField(row.type)"
                      :key="type.value"
                      :label="type.label"
                      :value="type.value"
                    />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column prop="search_type" :label="$t('code_generator.search_type')" width="120">
                <template #default="{ row }">
                  <el-select v-model="row.search_type" :placeholder="$t('common.select')">
                    <el-option label="LIKE" value="like" />
                    <el-option label="=" value="=" />
                    <el-option label=">" value=">" />
                    <el-option label=">=" value=">=" />
                    <el-option label="<" value="<" />
                    <el-option label="<=" value="<=" />
                    <el-option label="!=" value="!=" />
                    <el-option label="IN" value="in" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column prop="search_ui_type" :label="$t('code_generator.search_ui_type')" width="150">
                <template #default="{ row }">
                  <el-select v-model="row.search_ui_type" :placeholder="$t('common.select')">
                    <el-option
                      v-for="type in searchUiTypesForField(row.type)"
                      :key="type.value"
                      :label="type.label"
                      :value="type.value"
                    />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column :label="$t('code_generator.relation')" width="150">
                <template #default="{ row }">
                  <el-button size="small" @click="handleEditRelation(row)">
                    {{ row.relation ? $t('code_generator.edit_relation') : $t('code_generator.add_relation') }}
                  </el-button>
                </template>
              </el-table-column>
              <el-table-column :label="$t('code_generator.field_config')" width="100">
                <template #default="{ row }">
                  <el-button 
                    v-if="row.form_type === 'select' || row.form_type === 'radio' || row.form_type === 'checkbox' || row.type === 'decimal' || row.search_ui_type === 'select'"
                    size="small" 
                    @click="handleEditFieldConfig(row)"
                  >
                    <el-icon><Setting /></el-icon>
                  </el-button>
                </template>
              </el-table-column>
              <el-table-column :label="$t('code_generator.field_options')" width="300">
                <template #default="{ row }">
                  <el-checkbox v-model="row.required">{{ $t('code_generator.required') }}</el-checkbox>
                  <el-checkbox v-model="row.searchable">{{ $t('code_generator.searchable') }}</el-checkbox>
                  <el-checkbox v-model="row.sortable">{{ $t('code_generator.sortable') }}</el-checkbox>
                  <el-checkbox v-model="row.show_in_list">{{ $t('code_generator.show_in_list') }}</el-checkbox>
                  <el-checkbox v-model="row.show_in_form">{{ $t('code_generator.show_in_form') }}</el-checkbox>
                </template>
              </el-table-column>
              <el-table-column :label="$t('table.operation')" width="100" fixed="right">
                <template #default="{ $index }">
                  <el-button type="danger" size="small" @click="handleRemoveField($index)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </template>
              </el-table-column>
            </el-table>

            <el-divider>{{ $t('code_generator.code_preview') }}</el-divider>

            <el-tabs v-model="activeTab" type="border-card">
              <el-tab-pane
                v-for="fileType in fileTypes"
                :key="fileType.value"
                :label="fileType.label"
                :name="fileType.value"
              >
                <div class="code-preview">
                  <el-button
                    type="primary"
                    size="small"
                    @click="handlePreview(fileType.value)"
                    :loading="previewing === fileType.value"
                  >
                    {{ $t('code_generator.refresh_preview') }}
                  </el-button>
                  <pre v-if="previewCode[fileType.value]"><code>{{ previewCode[fileType.value] }}</code></pre>
                  <el-empty v-else :description="$t('code_generator.click_preview')" />
                </div>
              </el-tab-pane>
            </el-tabs>
          </el-form>
      </el-tab-pane>

        <!-- AI 辅助标签页（未配置 API Key 时显示配置说明） -->
        <el-tab-pane :label="$t('code_generator.ai_mode')" name="ai">
          <div class="ai-assistant">
            <el-alert
              v-if="!aiEnabled"
              :title="$t('code_generator.ai_not_configured_title')"
              type="warning"
              :closable="false"
              style="margin-bottom: 20px"
            >
              <p>{{ $t('code_generator.ai_not_configured_tip') }}</p>
            </el-alert>
            <template v-else>
            <el-alert
              :title="$t('code_generator.ai_mode_tip')"
              type="info"
              :closable="false"
              style="margin-bottom: 20px"
            />
            <el-form label-width="120px">
              <el-form-item :label="$t('code_generator.ai_example_title')">
                <div class="ai-example-prompts">
                  <el-tag
                    v-for="(prompt, index) in aiExamplePrompts"
                    :key="index"
                    class="ai-example-tag"
                    effect="plain"
                    @click="applyAIExample(prompt)"
                  >
                    {{ prompt }}
                  </el-tag>
                </div>
              </el-form-item>
              <el-form-item :label="$t('code_generator.ai_description')">
                <el-input
                  v-model="aiDescription"
                  type="textarea"
                  :rows="8"
                  :placeholder="$t('code_generator.ai_description_placeholder')"
                />
              </el-form-item>
              <el-form-item>
                <el-button
                  type="primary"
                  :loading="aiGenerating"
                  @click="handleGenerateWithAI"
                >
                  <el-icon><Document /></el-icon>
                  {{ $t('code_generator.ai_generate') }}
                </el-button>
                <el-button
                  v-if="aiLastError"
                  :disabled="aiGenerating"
                  @click="handleGenerateWithAI"
                >
                  {{ $t('code_generator.ai_retry') }}
                </el-button>
                <el-button @click="handleApplyAIConfig" :disabled="!aiGeneratedConfig">
                  {{ $t('code_generator.ai_apply_config') }}
                </el-button>
              </el-form-item>
            </el-form>

            <el-divider v-if="aiGeneratedConfig">{{ $t('code_generator.ai_generated_config') }}</el-divider>

            <div v-if="aiGeneratedConfig" class="ai-config-preview">
              <el-descriptions :column="2" border>
                <el-descriptions-item :label="$t('code_generator.module_name')">
                  {{ aiGeneratedConfig.module_name }}
                </el-descriptions-item>
                <el-descriptions-item :label="$t('code_generator.table_name')">
                  {{ aiGeneratedConfig.table_name }}
                </el-descriptions-item>
                <el-descriptions-item :label="$t('code_generator.fields_count')" :span="2">
                  {{ aiGeneratedConfig.fields?.length || 0 }}
                </el-descriptions-item>
              </el-descriptions>

              <el-table
                v-if="aiGeneratedConfig.fields && aiGeneratedConfig.fields.length > 0"
                :data="aiGeneratedConfig.fields"
                border
                style="margin-top: 20px"
                max-height="400"
              >
                <el-table-column type="index" :label="$t('table.index')" width="60" />
                <el-table-column prop="name" :label="$t('code_generator.field_name')" width="120" />
                <el-table-column prop="label" :label="$t('code_generator.field_label')" width="120" />
                <el-table-column prop="db_type" :label="$t('code_generator.field_type')" width="100" />
                <el-table-column prop="form_type" :label="$t('code_generator.form_type')" width="120" />
                <el-table-column prop="required" :label="$t('code_generator.required')" width="80">
                  <template #default="{ row }">
                    <el-tag v-if="row.required" type="success">{{ $t('common.yes') }}</el-tag>
                    <el-tag v-else type="info">{{ $t('common.no') }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="searchable" :label="$t('code_generator.searchable')" width="80">
                  <template #default="{ row }">
                    <el-tag v-if="row.searchable" type="success">{{ $t('common.yes') }}</el-tag>
                    <el-tag v-else type="info">{{ $t('common.no') }}</el-tag>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            </template>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog
      v-model="relationDialogVisible"
      :title="$t('code_generator.relation_config')"
      width="600px"
    >
      <el-form :model="relationForm" label-width="120px">
        <el-form-item :label="$t('code_generator.relation_table')">
          <el-input v-model="relationForm.table" :placeholder="$t('code_generator.relation_table_placeholder')" />
        </el-form-item>
        <el-form-item :label="$t('code_generator.relation_type')">
          <el-select v-model="relationForm.relation_type" :placeholder="$t('common.select')">
            <el-option label="belongsTo" value="belongsTo" />
            <el-option label="hasOne" value="hasOne" />
            <el-option label="hasMany" value="hasMany" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('code_generator.foreign_key')">
          <el-input v-model="relationForm.foreign_key" :placeholder="$t('code_generator.foreign_key_placeholder')" />
        </el-form-item>
        <el-form-item :label="$t('code_generator.display_field')">
          <el-input v-model="relationForm.display_field" :placeholder="$t('code_generator.display_field_placeholder')" />
        </el-form-item>
        <el-form-item 
          v-if="currentField && currentField.api_url"
          :label="$t('code_generator.is_tree')"
        >
          <el-checkbox v-model="relationForm.is_tree">{{ $t('code_generator.is_tree_desc') }}</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="relationDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSaveRelation">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
    <el-dialog
      v-model="fieldConfigDialogVisible"
      :title="$t('code_generator.field_config')"
      width="500px"
    >
      <el-form :model="fieldConfigForm" label-width="120px">
        <template v-if="currentField && currentField.type === 'decimal'">
          <el-form-item :label="$t('code_generator.precision')">
            <el-input-number v-model="fieldConfigForm.precision" :min="1" :max="65" />
          </el-form-item>
          <el-form-item :label="$t('code_generator.scale')">
            <el-input-number v-model="fieldConfigForm.scale" :min="0" :max="30" />
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item :label="$t('code_generator.option_type')">
            <el-radio-group v-model="fieldConfigForm.option_type">
              <el-radio label="dictionary">{{ $t('code_generator.option_type_dictionary') }}</el-radio>
              <el-radio label="api">{{ $t('code_generator.option_type_api') }}</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item 
            v-if="fieldConfigForm.option_type === 'dictionary'"
            :label="$t('code_generator.dictionary_key')"
          >
            <el-select 
              v-model="fieldConfigForm.dictionary" 
              :placeholder="$t('code_generator.dictionary_key_placeholder')"
              filterable
              allow-create
              default-first-option
              clearable
            >
              <el-option
                v-for="item in dictionaryTypes"
                :key="item"
                :label="item"
                :value="item"
              />
            </el-select>
          </el-form-item>
          <el-form-item 
            v-if="fieldConfigForm.option_type === 'api'"
            :label="$t('code_generator.api_url')"
          >
            <el-input v-model="fieldConfigForm.api_url" :placeholder="$t('code_generator.api_url_placeholder')" />
          </el-form-item>
          <el-form-item 
            v-if="fieldConfigForm.option_type === 'api'"
            :label="$t('code_generator.is_tree')"
          >
            <el-checkbox v-model="fieldConfigForm.is_tree">{{ $t('code_generator.is_tree_desc') }}</el-checkbox>
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="fieldConfigDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSaveFieldConfig">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>


  </div>
</template>

<script setup>
import { useCodeGenerator } from './code-generator/useCodeGenerator'

const {
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
} = useCodeGenerator()
</script>

<style scoped>
.code-generator {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header-actions {
  display: flex;
  gap: 8px;
}

.code-preview {
  position: relative;
}

.code-preview pre {
  background: var(--bg-color-tertiary);
  padding: 15px;
  border-radius: var(--border-radius-sm);
  max-height: 500px;
  overflow: auto;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  line-height: 1.5;
}

.code-preview code {
  color: var(--text-color-primary);
}

/* 暗黑模式样式 */
html.dark .code-preview pre {
  background: var(--el-bg-color) !important;
  border: 1px solid var(--el-border-color);
}

html.dark .code-preview code {
  color: var(--el-text-color-regular) !important;
}

.ai-example-prompts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.ai-example-tag {
  cursor: pointer;
  max-width: 100%;
  height: auto;
  white-space: normal;
  line-height: 1.4;
  padding: 6px 10px;
}
</style>
