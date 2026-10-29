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
        <!-- 配置式渲染表单字段 -->
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
// 引入配置式表单组件
import FormField from '../../components/Form/FormField.vue'
import { getEnableDisableOptions } from '@/utils/options'
import { createUser, updateUser, getUserDetail } from '../../api/user'
import { mapFields } from '../../utils/normalizeFormData'
import ErrorHandler from '../../utils/errorHandler'
import { describePasswordPolicy, passwordPolicyRule } from '../../utils/passwordPolicy'
import { useUserStore } from '../../store/user'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  editId: {
    type: Number,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'success'])

const { t } = useI18n()
const userStore = useUserStore()
const formRef = ref(null)
const loading = ref(false)
const submitting = ref(false)

// 定义表单初始值的复用函数（返回新对象，避免引用问题）
const getFormInitialValue = () => ({
  id: null,
  username: '',
  password: '',
  nickname: '',
  email: '',
  phone: '',
  status: 0 // 默认禁用状态
})

// 对话框显隐状态
const dialogVisible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

// 对话框标题
const dialogTitle = computed(() => {
  return props.editId ? t('user.edit_user') : t('user.add_user')
})

// 表单数据
const formData = reactive(getFormInitialValue())

// 动态验证规则
const formRules = computed(() => {
  const rules = {
    email: [
      { type: 'email', message: t('form.email_invalid'), trigger: 'blur' }
    ]
  }
  
  // 创建时，username 和 password 是必填的
  if (!props.editId) {
    rules.username = [
      { required: true, message: t('form.username_required'), trigger: 'blur' },
      { min: 3, message: t('form.username_min'), trigger: 'blur' },
      { max: 50, message: t('form.username_max'), trigger: 'blur' }
    ]
    rules.password = [
      { required: true, message: t('form.password_required'), trigger: 'blur' },
      passwordPolicyRule(userStore.config.passwordPolicy, t),
      { max: 50, message: t('form.password_max_length'), trigger: 'blur' }
    ]
  } else {
    // 编辑时，username 和 password 不是必填的（username 不可修改，password 可选）
    rules.password = [
      passwordPolicyRule(userStore.config.passwordPolicy, t),
      { max: 50, message: t('form.password_max_length'), trigger: 'blur' }
    ]
  }
  
  return rules
})

// 配置式表单字段
const formFields = computed(() => {
  const fields = [
    {
      prop: 'username',
      label: t('table.username'),
      type: 'input',
      // 编辑时禁用，加载中也禁用
      disabled: !!formData.id || loading.value
    },
    {
      prop: 'password',
      label: t('common.password'),
      type: 'password',
      placeholder: describePasswordPolicy(userStore.config.passwordPolicy, t),
      disabled: loading.value,
      // 仅新增时显示（和原代码 v-if="!formData.id" 逻辑一致）
      visible: () => !formData.id
    },
    {
      prop: 'nickname',
      label: t('table.nickname'),
      type: 'input',
      disabled: loading.value
    },
    {
      prop: 'email',
      label: t('table.email'),
      type: 'input',
      disabled: loading.value
    },
    {
      prop: 'phone',
      label: t('table.phone'),
      type: 'input',
      disabled: loading.value
    },
    {
      prop: 'status',
      label: t('table.status'),
      type: 'radio',
      disabled: loading.value,
      // 配置radio选项（和原代码 el-radio-group 逻辑一致）
      options: getEnableDisableOptions(t),
    }
  ]
  return fields
})

// 监听editId变化
watch(() => props.editId, async (newId) => {
  if (newId && dialogVisible.value) {
    await loadData()
  } else {
    resetForm()
  }
})

// 监听对话框显隐
watch(dialogVisible, (val) => {
  if (val && props.editId) {
    loadData()
  } else if (!val) {
    resetForm()
  }
})

// 加载用户详情
const loadData = async () => {
  if (!props.editId) {
    resetForm()
    return
  }

  loading.value = true
  try {
    const res = await getUserDetail(props.editId)
    if (res.code === 200 && res.data && res.data.user) {
      const user = res.data.user
      // 使用工具函数映射字段，自动处理 snake_case 和 PascalCase
      const mapped = mapFields(user, getFormInitialValue())
      // 编辑时不填充密码
      mapped.password = ''
      // 确保 status 是数字类型
      mapped.status = Number(mapped.status)
      Object.assign(formData, mapped)
    }
  } catch (error) {
    ErrorHandler.handle(error)
  } finally {
    loading.value = false
  }
}

// 重置表单
const resetForm = () => {
  Object.assign(formData, getFormInitialValue())
  formRef.value?.resetFields()
}

// 对话框关闭时重置表单
const handleDialogClose = () => {
  resetForm()
}

// 取消按钮
const handleCancel = () => {
  dialogVisible.value = false
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      if (props.editId) {
        // 编辑时，不提交 username（因为不可修改）
        // 确保 status 是数字类型（0 或 1）
        const updateData = { 
          ...formData,
          status: Number(formData.status) || 0
        }
        delete updateData.username
        delete updateData.id
        // 确保 status 是 0 或 1
        updateData.status = updateData.status === 1 ? 1 : 0
        await updateUser(props.editId, updateData)
        ElMessage.success(t('common.update_success'))
      } else {
        // 创建时，提交所有数据（包括 username）
        // 确保 status 是数字类型（0 或 1）
        const createData = { 
          ...formData,
          status: Number(formData.status) || 0
        }
        delete createData.id
        // 确保 status 是 0 或 1
        createData.status = createData.status === 1 ? 1 : 0
        await createUser(createData)
        ElMessage.success(t('common.create_success'))
      }
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