<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="700px"
    @close="handleDialogClose"
  >
    <div v-loading="loading">
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="120px"
      >
        <!-- 配置式渲染表单字段，替代硬编码的el-form-item -->
        <FormField
          v-for="f in formFields"
          :key="f.prop"
          :field="f"
          :model="formData"
        >
          <!-- IP字段的额外提示文字插槽 -->
          <template v-if="f.prop === 'ip'">
            <div style="margin-top: 8px; color: #909399; font-size: 12px;">
              {{ $t('blacklist.ip_tip') }}
            </div>
          </template>
        </FormField>
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

import {
  getBlacklistDetail,
  createBlacklist,
  updateBlacklist
} from '../../api/blacklist'
import { mapFields } from '../../utils/normalizeFormData'
import { promptSensitiveConfirm } from '../../utils/sensitiveConfirm'

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

// 对话框显隐状态（双向绑定）
const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// 定义表单初始值的复用函数（返回新对象，避免引用问题）
const getFormInitialValue = () => ({
  id: null,
  ip: '',
  remark: '',
  status: 1
})

// 对话框标题（新增/编辑区分）
const dialogTitle = computed(() => formData.id ? t('blacklist.edit_blacklist') : t('blacklist.add_blacklist'))

// 表单数据
const formData = reactive(getFormInitialValue())

// 表单验证规则（保留原有IP验证逻辑）
const formRules = computed(() => ({
  ip: [
    { required: true, message: t('blacklist.ip_required'), trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (!value || value.trim() === '') {
          callback(new Error(t('blacklist.ip_required')))
          return
        }
        // 前端简单验证，后端会做详细验证
        const ipList = value.split(',')
        for (const ip of ipList) {
          const trimmedIP = ip.trim()
          if (trimmedIP === '') continue
          // 简单检查：至少包含点或斜杠或横线
          if (!trimmedIP.includes('.') && !trimmedIP.includes('/') && !trimmedIP.includes('-')) {
            callback(new Error(t('blacklist.ip_format_error')))
            return
          }
        }
        callback()
      },
      trigger: 'blur'
    }
  ]
}))

// 配置式表单字段（核心改造点：和admin页面统一风格）
const formFields = computed(() => {
  const fields = [
    {
      prop: 'ip',
      label: t('blacklist.ip'),
      type: 'textarea', // 对应原IP的textarea输入框
      rows: 4, // 行数和原代码一致
      placeholder: t('blacklist.ip_placeholder'),
      disabled: loading.value // 加载中禁用
    },
    {
      prop: 'remark',
      label: t('blacklist.remark'),
      type: 'textarea', // 备注的textarea
      rows: 3,
      placeholder: t('blacklist.remark_placeholder'),
      disabled: loading.value
    },
    {
      prop: 'status',
      label: t('table.status'),
      type: 'radio', // 单选按钮类型
      disabled: loading.value,
      // 配置radio的选项（和原代码一致）
      options: [
        { label: t('blacklist.enabled'), value: 1 },
        { label: t('blacklist.disabled'), value: 0 }
      ]
    }
  ]
  return fields
})

// 监听editId变化，加载详情
watch(() => props.editId, async (newId) => {
  if (newId && dialogVisible.value) {
    await loadDetail(newId)
  } else if (!newId && dialogVisible.value) {
    resetForm()
  }
}, { immediate: true })

// 监听对话框显隐，重置/加载表单
watch(dialogVisible, (visible) => {
  if (visible) {
    if (props.editId) {
      loadDetail(props.editId)
    } else {
      resetForm()
    }
  }
})

// 加载黑名单详情
const loadDetail = async (id) => {
  loading.value = true
  try {
    const res = await getBlacklistDetail(id)
    if (res.data && res.data.blacklist) {
      const blacklist = res.data.blacklist
      // 使用工具函数映射字段，自动处理 snake_case 和 PascalCase
      const mapped = mapFields(blacklist, getFormInitialValue())
      Object.assign(formData, mapped)
    }
  } catch (error) {
    console.error('Load blacklist detail error:', error)
  } finally {
    loading.value = false
  }
}

// 重置表单
const resetForm = () => {
  loading.value = false
  Object.assign(formData, {
    id: null,
    ip: '',
    remark: '',
    status: 1
  })
  formRef.value?.resetFields()
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        // 只处理需要转换的字段（去除首尾空格）
        const submitData = {
          ...formData,
          ip: formData.ip.trim(),
          remark: formData.remark.trim()
        }
        if (formData.id) {
          await updateBlacklist(formData.id, submitData)
          ElMessage.success(t('blacklist.update_success'))
        } else {
          const confirmCode = await promptSensitiveConfirm(t)
          await createBlacklist({ ...submitData, confirm_code: confirmCode })
          ElMessage.success(t('blacklist.create_success'))
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

// 取消按钮
const handleCancel = () => {
  dialogVisible.value = false
}

// 对话框关闭时重置表单
const handleDialogClose = () => {
  formRef.value?.resetFields()
}
</script>