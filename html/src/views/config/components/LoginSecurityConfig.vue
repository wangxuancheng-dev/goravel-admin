<template>
  <div class="login-security-config">
    <el-form
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="160px"
      label-position="left"
    >
      <el-form-item :label="$t('config.password_min_length')" prop="password_min_length">
        <el-input-number
          v-model="formData.password_min_length"
          :min="4"
          :max="64"
        />
      </el-form-item>
      <el-form-item :label="$t('config.password_require_letter')" prop="password_require_letter">
        <el-switch v-model="formData.password_require_letter" />
      </el-form-item>
      <el-form-item :label="$t('config.password_require_number')" prop="password_require_number">
        <el-switch v-model="formData.password_require_number" />
      </el-form-item>
      <el-form-item :label="$t('config.password_require_special')" prop="password_require_special">
        <el-switch v-model="formData.password_require_special" />
      </el-form-item>
      <el-form-item :label="$t('config.anomaly_alert_enabled')" prop="anomaly_alert_enabled">
        <el-switch v-model="formData.anomaly_alert_enabled" />
        <span style="margin-left: 10px; color: var(--text-color-secondary);">{{ $t('config.anomaly_alert_enabled_tip') }}</span>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="handleSubmit" :loading="submitting" :disabled="getButtonState('config.save').disabled">
          {{ $t('common.save') }}
        </el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { getConfigByGroup, saveConfig } from '../../../api/config'
import { usePermission } from '../../../composables/usePermission'

const { t } = useI18n()
const { getButtonState } = usePermission()
const formRef = ref(null)
const submitting = ref(false)

const formData = reactive({
  password_min_length: 8,
  password_require_letter: true,
  password_require_number: true,
  password_require_special: false,
  anomaly_alert_enabled: true
})

const formRules = {
  password_min_length: [
    { required: true, message: t('config.password_min_length_required'), trigger: 'blur' },
    { type: 'number', min: 4, message: t('config.password_min_length_min', { min: 4 }), trigger: 'blur' }
  ]
}

const loadData = async () => {
  try {
    const res = await getConfigByGroup('login_security')
    if (res.data && res.data.configs) {
      res.data.configs.forEach((config) => {
        const key = config.Key || config.key
        let value = config.Value || config.value || ''
        if (key === 'password_min_length') {
          value = value ? parseInt(value, 10) : 8
        } else if (
          key === 'password_require_letter' ||
          key === 'password_require_number' ||
          key === 'password_require_special' ||
          key === 'anomaly_alert_enabled'
        ) {
          value = value === '1' || value === 'true' || value === true
        }
        if (Object.prototype.hasOwnProperty.call(formData, key)) {
          formData[key] = value
        }
      })
    }
  } catch (error) {
    console.error('Load login_security config error:', error)
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await saveConfig('login_security', {
        password_min_length: String(formData.password_min_length),
        password_require_letter: formData.password_require_letter ? '1' : '0',
        password_require_number: formData.password_require_number ? '1' : '0',
        password_require_special: formData.password_require_special ? '1' : '0',
        anomaly_alert_enabled: formData.anomaly_alert_enabled ? '1' : '0'
      })
      ElMessage.success(t('config.update_success'))
    } catch (error) {
      console.error('Submit login_security config error:', error)
    } finally {
      submitting.value = false
    }
  })
}

onMounted(() => {
  loadData()
})

defineExpose({ loadData })
</script>

<style scoped>
.login-security-config {
  padding: 20px 0;
}
</style>
