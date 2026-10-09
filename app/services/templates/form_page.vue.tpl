<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="1000px"
    @close="handleDialogClose"
  >
    <div v-loading="loading">
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="120px"
      >
        <<range .FormFields>>
        <<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
        <<- if .ShowInForm>>
        <<- if or (eq .FormType "image-upload") (eq .FormType "file-upload") (eq .FormType "editor") (eq .FormType "markdown")>>
        <el-form-item :label="$t('<<.Name>>')" prop="<<.Name>>">
          <<- if eq .FormType "image-upload">>
          <ImageUpload
            v-model="formData.<<.Name>>"
            :height="400"
            :width="400"
          />
          <<- else if eq .FormType "file-upload">>
          <!-- File upload component, implement based on actual requirements. -->
          <el-upload
            v-model="formData.<<.Name>>"
            action="/api/upload/file"
            :show-file-list="false"
          >
            <el-button type="primary">Upload File</el-button>
          </el-upload>
          <<- else if eq .FormType "editor">>
          <WangEditor
            v-model="formData.<<.Name>>"
            :placeholder="$t('<<.Name>>_placeholder')"
            :height="400"
          />
          <<- else if eq .FormType "markdown">>
          <MarkdownEditor
            v-model="formData.<<.Name>>"
            :placeholder="$t('<<.Name>>_placeholder')"
            :height="400"
          />
          <<- end>>
        </el-form-item>
        <<- end>>
        <<- end>>
        <<- end>>
        <<- end>>
        <FormField
          v-for="f in formFields"
          :key="f.prop"
          :field="f"
          :model="formData"
        />
<<if .IsMasterDetail>>
        <el-divider content-position="left">{{ $t('common.detail_rows') }}</el-divider>
        <div style="margin-bottom: 12px">
          <el-button type="primary" link @click="addDetailRow">{{ $t('common.add') }}</el-button>
        </div>
        <el-table :data="formData.details" border size="small">
<<- range .DetailFormFields>>
          <el-table-column :label="$t('<<.Name>>')" min-width="120">
            <template #default="{ row }">
              <<if eq .FormType "number">>
              <el-input-number v-model="row.<<.Name>>" :controls="false" style="width: 100%" />
              <<else>>
              <el-input v-model="row.<<.Name>>" />
              <<end>>
            </template>
          </el-table-column>
<<- end>>
          <el-table-column :label="$t('common.operation')" width="80" fixed="right">
            <template #default="{ $index }">
              <el-button type="danger" link @click="removeDetailRow($index)">{{ $t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
<<end>>
      </el-form>
    </div>
    <template #footer>
      <el-button @click="handleCancel">{{ $t('common.cancel') }}</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $t('common.confirm') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import FormField from '../../components/Form/FormField.vue'
<<if .HasEditor>>
import WangEditor from '../../components/WangEditor.vue'
<<end>>
<<if .HasMarkdown>>
import MarkdownEditor from '../../components/MarkdownEditor.vue'
<<end>>
<<if .HasImageUpload>>
import ImageUpload from '../../components/ImageUpload.vue'
<<end>>
<<range .FormFields>>
<<- if and (eq .FormType "select") (or .Relation .ApiUrl)>>
<<- if .Relation>>
// Relation field: <<.Name>> -> <<.Relation.Table>>
<<- end>>
<<- end>>
<<- end>>
import {
  <<if .HasCreate>>create<<.ModelName>>,<<end>>
  <<if .HasEdit>>update<<.ModelName>>,<<end>>
  get<<.ModelName>>Detail
} from '../../api/<<.ModuleName>>'
import { mapFields, normalizeFormData } from '../../utils/normalizeFormData'
import ErrorHandler from '../../utils/errorHandler'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  editId: {
    type: [Number, String],
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'success'])

const { t } = useI18n()
const formRef = ref(null)
const submitting = ref(false)
const loading = ref(false)

// Reusable function to build initial form values.
const getFormInitialValue = () => ({
<<range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
  <<.Name>>: <<if eq .FormType "switch">>0<<else if eq .FormType "number">>0<<else if eq .FormType "date-picker">>null<<else if eq .FormType "datetime-picker">>null<<else if .Relation>>null<<else if or (eq .FormType "select") (eq .FormType "radio")>>null<<else if eq .FormType "checkbox">>[]<<else>>''<<end>>,
<<- end>>
<<- end>>
<<if .IsMasterDetail>>
  details: [{}],
<<end>>
})

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const dialogTitle = computed(() => {
  return formData.id ? t('<<.ModuleName>>.edit_<<.ModuleName>>') : t('<<.ModuleName>>.add_<<.ModuleName>>')
})

const formData = reactive(getFormInitialValue())

<<if .IsMasterDetail>>
const addDetailRow = () => {
  if (!Array.isArray(formData.details)) formData.details = []
  formData.details.push({})
}
const removeDetailRow = (index) => {
  formData.details.splice(index, 1)
  if (formData.details.length === 0) formData.details.push({})
}
<<end>>

const formRules = computed(() => {
  const rules = {}
<<range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
  <<- if .Required>>
  rules['<<.Name>>'] = [
    <<- if or (eq .FormType "select") (eq .FormType "radio") (eq .FormType "checkbox") (eq .FormType "switch") (eq .FormType "date-picker") (eq .FormType "datetime-picker")>>
    { required: true, message: t('common.select_required'), trigger: 'change' }
    <<- else>>
    { required: true, message: t('common.required'), trigger: 'blur' }
    <<- end>>
  ]
  <<- end>>
<<- end>>
<<- end>>
  return rules
})

// Schema-driven form fields.
const formFields = computed(() => {
  const fields = []
<<range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
  <<- if and .ShowInForm (ne .FormType "image-upload") (ne .FormType "file-upload") (ne .FormType "editor") (ne .FormType "markdown")>>
  fields.push({
    prop: '<<.Name>>',
    label: t('<<.Name>>'),
    type: <<if eq .FormType "input">>'input'<<else if eq .FormType "textarea">>'textarea'<<else if eq .FormType "select">><<- if and .ApiUrl (or (and .Relation .Relation.IsTree) .IsTree)>>'tree-select'<<else>>'select'<<- end>><<else if eq .FormType "radio">>'radio'<<else if eq .FormType "checkbox">>'checkbox'<<else if eq .FormType "switch">>'switch'<<else if eq .FormType "date-picker">>'date'<<else if eq .FormType "datetime-picker">>'datetime'<<else if eq .FormType "number">>'number'<<else>>'input'<<end>>,
    disabled: loading.value,
    <<- if eq .FormType "textarea">>
    rows: 4,
    <<- end>>
    <<- if .Relation>>
    <<- if .ApiUrl>>
    <<- if .Relation.IsTree>>
    apiUrl: '<<.ApiUrl>>',
    treeProps: { label: '<<.Relation.DisplayField>>', value: 'id', children: 'children' },
    clearable: true,
    <<- else>>
    apiUrl: '<<.ApiUrl>>',
    clearable: true,
    <<- if .Relation.DisplayField>>
    optionLabelKey: '<<.Relation.DisplayField>>',
    <<- end>>
    optionValueKey: 'id',
    <<- end>>
    <<- end>>
    <<- else if .ApiUrl>>
    <<- if or (eq .FormType "select") (eq .FormType "radio") (eq .FormType "checkbox")>>
    apiUrl: '<<.ApiUrl>>',
    <<- if .IsTree>>
    treeProps: { label: 'label', value: 'value', children: 'children' },
    <<- end>>
    clearable: true,
    <<- end>>
    <<- else if or (eq .FormType "select") (eq .FormType "radio") (eq .FormType "checkbox")>>
    <<- if .Dictionary>>
    apiUrl: '/options?type=dictionary&dictionary_type=<<.Dictionary>>',
    clearable: true,
    <<- end>>
    <<- end>>
    <<- if or (eq .FormType "date-picker") (eq .FormType "datetime-picker")>>
    clearable: true,
    <<- end>>
    <<- if eq .FormType "number">>
    min: 0,
    <<- end>>
    <<- if eq .FormType "switch">>
    props: {
      activeValue: 1,
      inactiveValue: 0
    },
    <<- end>>
  })
  <<- end>>
<<- end>>
<<- end>>
  return fields
})

watch(() => props.editId, async (newId) => {
  if (newId && dialogVisible.value) {
    await loadData()
  } else if (!newId && dialogVisible.value) {
    resetForm()
  }
}, { immediate: true })

watch(dialogVisible, (visible) => {
  if (visible) {
    if (props.editId) {
      loadData()
    } else {
      resetForm()
    }
  }
})

const loadData = async () => {
  if (!props.editId) {
    resetForm()
    return
  }

  loading.value = true
  try {
    const res = await get<<.ModelName>>Detail(props.editId)
    if (res.data && res.data.<<.ModuleName>>) {
      const data = res.data.<<.ModuleName>>
      const mapped = mapFields(data, getFormInitialValue())
      const normalizeRules = {}
<<range .FormFields>>
<<- if and (or (eq .FormType "radio") (eq .FormType "select") (eq .FormType "checkbox")) .Dictionary>>
      normalizeRules['<<.Name>>'] = 'string'
<<- end>>
<<- end>>
      const normalized = normalizeFormData(mapped, normalizeRules)
      Object.assign(formData, normalized)
<<if .IsMasterDetail>>
      formData.details = Array.isArray(data.details) && data.details.length ? data.details.map((row) => ({ ...row })) : [{}]
<<end>>
    }
  } catch (error) {
    ErrorHandler.handle(error)
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  Object.assign(formData, getFormInitialValue())
  formRef.value?.resetFields()
}

const handleDialogClose = () => {
  formRef.value?.resetFields()
}

const handleCancel = () => {
  dialogVisible.value = false
}

<<if .IsMasterDetail>>
const isBlankDetailRow = (row) => {
  if (!row || typeof row !== 'object') return true
<<- range .DetailFormFields>>
  {
    const v = row['<<.Name>>']
    if (v !== undefined && v !== null && String(v).trim() !== '') return false
  }
<<- end>>
  return true
}

const validateDetailRows = () => {
  const rows = Array.isArray(formData.details) ? formData.details : []
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i]
    if (isBlankDetailRow(row)) continue
<<- range .DetailFormFields>>
<<- if .Required>>
    {
      const v = row['<<.Name>>']
      if (v === undefined || v === null || String(v).trim() === '') {
        ElMessage.warning(t('common.required') + ': <<.Label>>')
        return false
      }
    }
<<- end>>
<<- end>>
  }
  return true
}
<<end>>

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return
<<if .IsMasterDetail>>
    if (!validateDetailRows()) return
<<end>>

    submitting.value = true
    try {
      const data = { ...formData }
      delete data.id
<<if .IsMasterDetail>>
      data.details = (Array.isArray(formData.details) ? formData.details : []).filter((row) => !isBlankDetailRow(row))
<<end>>
      <<if .HasEdit>>
      if (props.editId) {
        await update<<.ModelName>>(props.editId, data)
        ElMessage.success(t('common.update_success'))
      } else {
        <<if .HasCreate>>
        await create<<.ModelName>>(data)
        ElMessage.success(t('common.create_success'))
        <<end>>
      }
      <<else>>
      <<if .HasCreate>>
      await create<<.ModelName>>(data)
      ElMessage.success(t('common.create_success'))
      <<end>>
      <<end>>
      emit('success')
      dialogVisible.value = false
    } catch (error) {
      ErrorHandler.handle(error)
    } finally {
      submitting.value = false
    }
  })
}
</script>

<style scoped>
</style>
