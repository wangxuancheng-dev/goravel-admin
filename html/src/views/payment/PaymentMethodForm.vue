<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="760px"
    class="payment-method-dialog"
    destroy-on-close
    @close="handleDialogClose"
  >
    <div v-loading="loading" class="payment-method-form-body">
      <!-- 新建：分步引导 -->
      <el-steps
        v-if="!isEdit"
        :active="step"
        finish-status="success"
        align-center
        class="pm-steps"
      >
        <el-step :title="t('payment_method.step_choose_type')" />
        <el-step :title="t('payment_method.step_fill_config')" />
      </el-steps>

      <!-- Step 1：选择支付类型（点击即进入下一步） -->
      <div v-if="!isEdit && step === 0" class="type-picker">
        <p class="type-picker-hint">{{ t('payment_method.choose_type_hint') }}</p>
        <div class="type-grid">
          <div
            v-for="opt in typeOptions"
            :key="opt.value"
            role="button"
            tabindex="0"
            class="type-card"
            :class="{ active: formData.type === opt.value }"
            @click.stop="selectType(opt.value)"
            @keydown.enter.prevent="selectType(opt.value)"
            @keydown.space.prevent="selectType(opt.value)"
          >
            <span class="type-card-name">{{ opt.label }}</span>
            <span class="type-card-desc">{{ t(`payment_method.type_desc_${opt.value}`) }}</span>
          </div>
        </div>
      </div>

      <!-- Step 2 / 编辑：填写配置 -->
      <el-form
        v-show="isEdit || step === 1"
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="128px"
        class="pm-form"
      >
        <el-alert
          v-if="formData.type"
          :title="typeGuideTitle"
          type="info"
          :closable="false"
          show-icon
          class="pm-guide"
        >
          <template #default>
            <div>{{ typeGuideText }}</div>
          </template>
        </el-alert>

        <el-form-item v-if="isEdit" :label="t('payment_method.type')">
          <el-tag type="primary" size="large">{{ getPaymentMethodTypeLabel(t, formData.type) }}</el-tag>
          <span class="readonly-code">{{ t('payment_method.code') }}：{{ formData.code }}</span>
        </el-form-item>

        <el-form-item :label="t('payment_method.name')" prop="name">
          <el-input
            v-model="formData.name"
            :disabled="loading"
            :placeholder="t('payment_method.name_placeholder')"
            maxlength="50"
            show-word-limit
          />
          <div class="field-tip">{{ t('payment_method.name_tip') }}</div>
        </el-form-item>

        <el-form-item :label="t('table.status')" prop="is_active">
          <el-switch
            v-model="formData.is_active"
            :disabled="loading"
            :active-text="t('common.enabled')"
            :inactive-text="t('common.disabled')"
          />
        </el-form-item>

        <el-divider content-position="left">{{ t('payment_method.config_basic') }}</el-divider>
        <p class="section-hint">{{ t('payment_method.config_basic_hint') }}</p>

        <template v-for="field in basicConfigFields" :key="field.key">
          <el-form-item
            :label="t(`payment_method.${field.labelKey}`)"
            :prop="`config.${field.key}`"
            :rules="field.required ? [{ required: true, message: t('payment_method.field_required', { field: t(`payment_method.${field.labelKey}`) }), trigger: 'blur' }] : []"
          >
            <el-select
              v-if="field.type === 'select'"
              v-model="formData.config[field.key]"
              :disabled="loading"
              clearable
              :placeholder="t(`payment_method.${field.placeholderKey}`)"
              style="width: 100%"
            >
              <el-option
                v-for="opt in field.options || []"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
              />
            </el-select>
            <el-input
              v-else-if="field.type === 'textarea'"
              v-model="formData.config[field.key]"
              type="textarea"
              :rows="field.rows || 3"
              :disabled="loading"
              :placeholder="t(`payment_method.${field.placeholderKey}`)"
            />
            <el-input
              v-else
              v-model="formData.config[field.key]"
              :type="field.inputType || 'text'"
              :disabled="loading"
              :show-password="field.inputType === 'password'"
              :placeholder="t(`payment_method.${field.placeholderKey}`)"
            />
            <div v-if="field.tipKey" class="field-tip">{{ t(`payment_method.${field.tipKey}`) }}</div>
          </el-form-item>
        </template>

        <el-collapse v-model="advancedOpen" class="pm-advanced">
          <el-collapse-item name="advanced">
            <template #title>
              <span>{{ t('payment_method.advanced_settings') }}</span>
              <span class="advanced-sub">{{ t('payment_method.advanced_settings_hint') }}</span>
            </template>

            <template v-for="field in advancedConfigFields" :key="field.key">
              <el-form-item :label="t(`payment_method.${field.labelKey}`)">
                <el-select
                  v-if="field.type === 'select'"
                  v-model="formData.config[field.key]"
                  :disabled="loading"
                  clearable
                  :placeholder="t(`payment_method.${field.placeholderKey}`)"
                  style="width: 100%"
                >
                  <el-option
                    v-for="opt in field.options || []"
                    :key="opt.value"
                    :label="opt.label"
                    :value="opt.value"
                  />
                </el-select>
                <el-input
                  v-else-if="field.type === 'textarea'"
                  v-model="formData.config[field.key]"
                  type="textarea"
                  :rows="field.rows || 3"
                  :disabled="loading"
                  :placeholder="t(`payment_method.${field.placeholderKey}`)"
                />
                <el-input
                  v-else
                  v-model="formData.config[field.key]"
                  :type="field.inputType || 'text'"
                  :disabled="loading"
                  :show-password="field.inputType === 'password'"
                  :placeholder="t(`payment_method.${field.placeholderKey}`)"
                />
                <div v-if="field.tipKey" class="field-tip">{{ t(`payment_method.${field.tipKey}`) }}</div>
              </el-form-item>
            </template>

            <el-form-item :label="t('table.sort')">
              <el-input-number v-model="formData.sort" :min="0" :disabled="loading" />
              <div class="field-tip">{{ t('payment_method.sort_tip') }}</div>
            </el-form-item>

            <el-form-item :label="t('table.description')">
              <el-input
                v-model="formData.description"
                type="textarea"
                :rows="2"
                :disabled="loading"
                :placeholder="t('payment_method.description_placeholder')"
              />
            </el-form-item>
          </el-collapse-item>
        </el-collapse>
      </el-form>
    </div>

    <template #footer>
      <div class="pm-footer">
        <el-button @click="handleCancel">{{ t('common.cancel') }}</el-button>
        <el-button v-if="!isEdit && step === 1" @click="backToTypeStep">{{ t('payment_method.prev_step') }}</el-button>
        <el-button
          v-if="isEdit || step === 1"
          type="primary"
          :loading="submitting"
          @click="handleSubmit"
        >
          {{ t('common.confirm') }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import {
  getPaymentMethodDetail,
  createPaymentMethod,
  updatePaymentMethod
} from '../../api/paymentMethod'
import { mapFields, getField } from '../../utils/normalizeFormData'
import logger from '../../utils/logger'
import ErrorHandler from '../../utils/errorHandler'
import { useUserStore } from '@/store/user'
import {
  createPaymentMethodTypeOptions,
  getPaymentMethodTypeLabel,
  getConfigFieldsByGroup,
  createEmptyConfig,
  generatePaymentCode,
  normalizeConfigForForm,
  collectConfigPayload
} from './paymentMethod.config'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  editId: { type: [Number, String], default: null }
})

const emit = defineEmits(['update:modelValue', 'success'])

const { t } = useI18n()
const userStore = useUserStore()
const formRef = ref(null)
const submitting = ref(false)
const loading = ref(false)
const step = ref(0)
const advancedOpen = ref([])

const getFormInitialValue = () => ({
  id: null,
  name: '',
  code: '',
  type: '',
  config: {},
  is_active: true,
  sort: 0,
  description: ''
})

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const isEdit = computed(() => Boolean(formData.id || props.editId))

const dialogTitle = computed(() =>
  isEdit.value ? t('payment_method.edit_payment_method') : t('payment_method.add_payment_method')
)

const formData = reactive(getFormInitialValue())
const typeOptions = computed(() =>
  createPaymentMethodTypeOptions(t, userStore.config?.paymentGateways)
)

const basicConfigFields = computed(() => getConfigFieldsByGroup(formData.type, 'basic'))
const advancedConfigFields = computed(() => getConfigFieldsByGroup(formData.type, 'advanced'))

const typeGuideTitle = computed(() => {
  if (!formData.type) return ''
  return t('payment_method.guide_title', { type: getPaymentMethodTypeLabel(t, formData.type) })
})

const typeGuideText = computed(() => {
  if (!formData.type) return ''
  return t(`payment_method.guide_${formData.type}`)
})

const formRules = computed(() => ({
  name: [{ required: true, message: t('payment_method.name_required'), trigger: 'blur' }]
}))

watch(dialogVisible, (visible) => {
  if (!visible) return
  if (props.editId) {
    loadDetail(props.editId)
  } else {
    resetForm()
  }
})

watch(() => props.editId, (newId, oldId) => {
  if (!dialogVisible.value || newId === oldId) return
  if (newId) {
    loadDetail(newId)
  } else {
    resetForm()
  }
})

const selectType = (type) => {
  if (!type) return
  const prevTypeLabel = formData.type ? getPaymentMethodTypeLabel(t, formData.type) : ''
  formData.type = type
  formData.config = createEmptyConfig(type)
  formData.code = generatePaymentCode(type)
  if (!formData.name || formData.name === prevTypeLabel) {
    formData.name = getPaymentMethodTypeLabel(t, type)
  }
  step.value = 1
}

const backToTypeStep = () => {
  step.value = 0
}

const loadDetail = async (id) => {
  loading.value = true
  try {
    const response = await getPaymentMethodDetail(id)
    const data = response.data?.data || response.data || {}
    Object.assign(formData, mapFields(data, getFormInitialValue()))
    const paymentType = getField(data, 'type', formData.type) || formData.type
    formData.type = paymentType
    const configData = getField(data, 'config', {}) || {}
    formData.config = normalizeConfigForForm(paymentType, configData)
    step.value = 1
    advancedOpen.value = []
  } catch (error) {
    logger.error('Load payment method detail error:', error)
    ErrorHandler.handle(error, { silent: true })
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  loading.value = false
  submitting.value = false
  Object.assign(formData, getFormInitialValue())
  formData.config = {}
  step.value = 0
  advancedOpen.value = []
  // 等表单重新渲染后再清校验，避免隐藏态 resetFields 异常
  requestAnimationFrame(() => {
    formRef.value?.clearValidate?.()
  })
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    const config = collectConfigPayload(formData.type, formData.config)
    if (Object.keys(config).length === 0) {
      ElMessage.error(t('payment_method.config_required'))
      return
    }

    const missing = basicConfigFields.value.filter(
      (f) => f.required && !(config[f.key] && String(config[f.key]).trim())
    )
    if (missing.length > 0) {
      ElMessage.error(
        t('payment_method.field_required', {
          field: t(`payment_method.${missing[0].labelKey}`)
        })
      )
      return
    }

    submitting.value = true
    try {
      const data = {
        name: formData.name.trim(),
        sort: Number(formData.sort) || 0,
        config,
        is_active: formData.is_active,
        description: formData.description || ''
      }

      if (formData.id) {
        await updatePaymentMethod(formData.id, data)
        ElMessage.success(t('payment_method.update_success'))
      } else {
        data.code = formData.code || generatePaymentCode(formData.type)
        data.type = formData.type
        await createPaymentMethod(data)
        ElMessage.success(t('payment_method.create_success'))
      }
      dialogVisible.value = false
      emit('success')
    } catch (error) {
      logger.error('Submit error:', error)
      if (!error.__handled) {
        const msg = error.response?.data?.message || error.message
        if (msg) ElMessage.error(msg)
      }
    } finally {
      submitting.value = false
    }
  })
}

const handleCancel = () => {
  dialogVisible.value = false
}

const handleDialogClose = () => {
  resetForm()
}
</script>

<style scoped>
.pm-steps {
  margin-bottom: 20px;
}

.type-picker-hint {
  margin: 0 0 16px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.type-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.payment-method-form-body {
  position: relative;
}

.type-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  padding: 14px 16px;
  border: 1px solid #dcdfe6;
  border-radius: 10px;
  background: #fff;
  cursor: pointer;
  text-align: left;
  user-select: none;
  transition: border-color 0.15s, box-shadow 0.15s, background-color 0.15s;
}

.type-card:hover {
  border-color: #409eff;
}

.type-card:focus-visible {
  outline: 2px solid #409eff;
  outline-offset: 2px;
}

.type-card.active {
  border-color: #409eff;
  box-shadow: 0 0 0 1px #b3d8ff;
  background: #ecf5ff;
}

.type-card-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.type-card-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

.pm-guide {
  margin-bottom: 16px;
}

.section-hint,
.field-tip {
  margin: 0 0 12px 128px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.field-tip {
  margin: 6px 0 0;
}

.readonly-code {
  margin-left: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.pm-advanced {
  border: none;
  margin-top: 8px;
}

.pm-advanced :deep(.el-collapse-item__header) {
  font-weight: 500;
}

.advanced-sub {
  margin-left: 8px;
  font-size: 12px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
}

.pm-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

@media (max-width: 640px) {
  .type-grid {
    grid-template-columns: 1fr;
  }

  .section-hint {
    margin-left: 0;
  }
}
</style>
