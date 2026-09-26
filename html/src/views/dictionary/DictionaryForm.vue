<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="600px"
    @close="handleDialogClose"
  >
    <div v-loading="loading">
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="100px"
      >
        <FormField
          v-for="f in formFields"
          :key="f.prop"
          :field="f"
          :model="formData"
        />
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
import { getEnableDisableOptions } from '@/utils/options'
import { isSystemDictionary } from './dictionary.config'

import {
  getDictionaryDetail,
  createDictionary,
  updateDictionary
} from '../../api/dictionary'
import { mapFields } from '../../utils/normalizeFormData'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  editId: {
    type: [Number, String],
    default: null
  },
  typeOptions: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue', 'success'])

const { t } = useI18n()
const formRef = ref(null)
const submitting = ref(false)
const loading = ref(false)

const getFormInitialValue = () => ({
  id: null,
  type: '',
  label: '',
  value: '',
  translation_key: '',
  status: 1,
  is_system: 0,
  sort: 0
})

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const dialogTitle = computed(() => formData.id ? t('dictionary.edit_dictionary') : t('dictionary.add_dictionary'))

const formData = reactive(getFormInitialValue())
const isSystem = computed(() => isSystemDictionary(formData))

const formRules = computed(() => ({
  type: [{ required: true, message: t('dictionary.type_required'), trigger: 'change' }],
  label: [{ required: true, message: t('dictionary.label_required'), trigger: 'blur' }],
  value: [{ required: true, message: t('dictionary.value_required'), trigger: 'blur' }]
}))

const formFields = computed(() => {
  const fields = [
    {
      prop: 'type',
      label: t('dictionary.type'),
      type: 'select',
      disabled: loading.value || isSystem.value,
      filterable: true,
      clearable: false,
      options: props.typeOptions,
      props: {
        allowCreate: true,
        defaultFirstOption: true
      }
    },
    {
      prop: 'label',
      label: t('dictionary.label'),
      type: 'input',
      disabled: loading.value
    },
    {
      prop: 'value',
      label: t('dictionary.value'),
      type: 'input',
      disabled: loading.value || isSystem.value
    },
    {
      prop: 'translation_key',
      label: t('dictionary.translation_key'),
      type: 'input',
      disabled: loading.value,
      noValidate: true
    },
    {
      prop: 'status',
      label: t('table.status'),
      type: 'radio',
      disabled: loading.value || isSystem.value,
      options: getEnableDisableOptions(t)
    },
    {
      prop: 'sort',
      label: t('common.sort'),
      type: 'number',
      disabled: loading.value,
      min: 0,
      noValidate: true
    }
  ]
  return fields
})

watch(() => props.editId, async (newId) => {
  if (newId && dialogVisible.value) {
    await loadDetail(newId)
  } else if (!newId && dialogVisible.value) {
    resetForm()
  }
}, { immediate: true })

watch(dialogVisible, (visible) => {
  if (visible) {
    if (props.editId) {
      loadDetail(props.editId)
    } else {
      resetForm()
    }
  }
})

const loadDetail = async (id) => {
  loading.value = true
  try {
    const res = await getDictionaryDetail(id)
    if (res.data && res.data.dictionary) {
      const dict = res.data.dictionary
      const mapped = mapFields(dict, getFormInitialValue())
      Object.assign(formData, mapped)
    }
  } catch (error) {
    console.error('Load dictionary detail error:', error)
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  loading.value = false
  Object.assign(formData, getFormInitialValue())
  formRef.value?.resetFields()
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        if (formData.id) {
          await updateDictionary(formData.id, formData)
          ElMessage.success(t('dictionary.update_success'))
        } else {
          await createDictionary(formData)
          ElMessage.success(t('dictionary.create_success'))
        }
        dialogVisible.value = false
        emit('success')
      } catch (error) {
        console.error('Submit error:', error)
      } finally {
        submitting.value = false
      }
    }
  })
}

const handleCancel = () => {
  dialogVisible.value = false
}

const handleDialogClose = () => {
  formRef.value?.resetFields()
}
</script>
