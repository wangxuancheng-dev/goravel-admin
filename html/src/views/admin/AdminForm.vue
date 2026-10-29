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
import { normalizeFormData } from '../../utils/normalizeFormData'

import {
  createAdmin,
  updateAdmin
} from '../../api/admin'
import logger from '../../utils/logger'
import ErrorHandler from '../../utils/errorHandler'
import { describePasswordPolicy, passwordPolicyRule } from '../../utils/passwordPolicy'
import { useUserStore } from '../../store/user'

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
const userStore = useUserStore()
const formRef = ref(null)
const submitting = ref(false)
const loading = ref(false)

// 定义表单初始值的复用函数（返回新对象，避免引用问题）
const getFormInitialValue = () => ({
  id: null,
  username: '',
  password: '',
  nickname: '',
  email: '',
  phone: '',
  department_id: null,
  position_id: null,
  role_ids: [],
  status: "1",
  is_super_admin: false
})


const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const dialogTitle = computed(() => formData.id ? t('admin.edit_admin') : t('admin.add_admin'))

const formData = reactive(getFormInitialValue())

const isDefaultAdmin = computed(() => formData.is_super_admin === true && formData.id !== null)

const formRules = computed(() => ({
  username: [{ required: true, message: t('admin.username_required'), trigger: 'blur' }],
  password: formData.id
    ? []
    : [
        { required: true, message: t('admin.password_required'), trigger: 'blur' },
        passwordPolicyRule(userStore.config.passwordPolicy, t)
      ]
}))

const formFields = computed(() => {
  const fields = [
    { prop: 'username', label: t('table.username'), type: 'input', disabled: !!formData.id || loading.value },
    {
      prop: 'password',
      label: t('common.password'),
      type: 'password',
      placeholder: describePasswordPolicy(userStore.config.passwordPolicy, t),
      disabled: loading.value,
      visible: () => !formData.id   // 只在新增时显示
    },
    { prop: 'nickname', label: t('table.nickname'), type: 'input', disabled: loading.value },
    { prop: 'email', label: t('table.email'), type: 'input', disabled: loading.value },
    { prop: 'phone', label: t('table.phone'), type: 'input', disabled: loading.value },
    {
      prop: 'department_id',
      label: t('table.department'),
      type: 'tree-select',
      apiUrl: '/options?type=department',
      treeProps: { label: 'name', value: 'id', children: 'children' },
      clearable: true,
      disabled: isDefaultAdmin.value || loading.value,
    },
    {
      prop: 'position_id',
      label: t('table.position'),
      type: 'select',
      apiUrl: '/options?type=position',
      clearable: true,
      filterable: true,
      disabled: isDefaultAdmin.value || loading.value,
    },
    {
      prop: 'role_ids',
      label: t('table.roles'),
      type: 'select',
      multiple: true,
      apiUrl: '/options?type=role',
      placeholder: t('form.select_role'),
      disabled: isDefaultAdmin.value || loading.value
    },
    {
      prop: 'status',
      label: t('table.status'),
      type: 'radio',
      apiUrl: '/options?type=dictionary&dictionary_type=status',
      // options: [
      //   { label: t('common.enabled'), value: 1 },
      //   { label: t('common.disabled'), value: 0 }
      // ],
      disabled: isDefaultAdmin.value || loading.value
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

const loadDetail = async () => {}

const resetForm = () => {
  loading.value = false
  Object.assign(formData, getFormInitialValue())
  formRef.value?.resetFields()
}

const setFormData = async (data) => {
  loading.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 50))

    const normalized = normalizeFormData(data, {
      // role_ids: 'string-array',      // 多选角色
      department_id: 'number',       // 单选部门（tree-select）
      position_id: 'number',
      status: 'string',             // 单选状态
    })

    // 未设置部门/岗位时后端为 0；树形/下拉应显示为空，提交时再发 0
    if (normalized.department_id === 0 || normalized.department_id === '0') {
      normalized.department_id = null
    }
    if (normalized.position_id === 0 || normalized.position_id === '0') {
      normalized.position_id = null
    }

    Object.assign(formData, normalized)

  } finally {
    loading.value = false
  }
}

defineExpose({ setFormData })

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      const data = { ...formData }
      if (data.username) data.username = data.username.trim()
      const did = data.department_id
      data.department_id =
        did === null || did === undefined || did === '' || did === 0 || did === '0'
          ? 0
          : Number(did)
      const pid = data.position_id
      data.position_id =
        pid === null || pid === undefined || pid === '' || pid === 0 || pid === '0'
          ? 0
          : Number(pid)
      if (formData.id) {
        if (!data.password) delete data.password
        if (isDefaultAdmin.value) delete data.role_ids
        await updateAdmin(formData.id, data)
        ElMessage.success(t('admin.update_success'))
      } else {
        await createAdmin(data)
        ElMessage.success(t('admin.create_success'))
      }
      dialogVisible.value = false
      emit('success')
    } catch (error) {
      logger.error('Submit error:', error)
      if (!error.__handled && error.response?.data?.message) {
        ElMessage.error(error.response.data.message)
      } else if (!error.__handled && error.message) {
        ElMessage.error(error.message)
      }
    } finally {
      submitting.value = false
    }
  })
}

const handleCancel = () => { dialogVisible.value = false }

const handleDialogClose = () => { formRef.value?.resetFields() }
</script>
