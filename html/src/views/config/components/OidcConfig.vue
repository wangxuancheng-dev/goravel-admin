<template>
  <div class="oidc-config">
    <el-alert type="info" :closable="false" show-icon style="margin-bottom: 16px">
      {{ $t('config.oidc_hint') }}
    </el-alert>
    <el-form ref="formRef" :model="formData" label-width="160px" label-position="left">
      <el-form-item :label="$t('config.oidc_enabled')">
        <el-switch v-model="formData.enabled" />
      </el-form-item>
      <el-form-item :label="$t('config.oidc_issuer')">
        <el-input v-model="formData.issuer" placeholder="https://login.microsoftonline.com/{tenant}/v2.0" />
      </el-form-item>
      <el-form-item :label="$t('config.oidc_client_id')">
        <el-input v-model="formData.client_id" />
      </el-form-item>
      <el-form-item :label="$t('config.oidc_client_secret')">
        <el-input v-model="formData.client_secret" type="password" show-password :placeholder="$t('config.keep_blank_to_preserve')" />
      </el-form-item>
      <el-form-item :label="$t('config.oidc_scopes')">
        <el-input v-model="formData.scopes" placeholder="openid profile email" />
      </el-form-item>
      <el-form-item :label="$t('config.oidc_button_label')">
        <el-input v-model="formData.button_label" />
      </el-form-item>
      <el-form-item :label="$t('config.oidc_auto_provision')">
        <el-switch v-model="formData.auto_provision" />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="submitting" :disabled="getButtonState('config.save').disabled" @click="handleSubmit">
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
  enabled: false,
  issuer: '',
  client_id: '',
  client_secret: '',
  scopes: 'openid profile email',
  button_label: '',
  auto_provision: false
})

const parseBool = (value) => value === '1' || value === 'true' || value === true

const loadData = async () => {
  try {
    const res = await getConfigByGroup('oidc')
    const configs = res.data?.configs || []
    configs.forEach((config) => {
      const key = config.Key || config.key
      let value = config.Value ?? config.value ?? ''
      if (key === 'enabled' || key === 'auto_provision') value = parseBool(value)
      if (Object.prototype.hasOwnProperty.call(formData, key)) formData[key] = value
    })
  } catch (error) {
    console.error('Load oidc config error:', error)
  }
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    await saveConfig('oidc', {
      enabled: formData.enabled ? '1' : '0',
      issuer: String(formData.issuer || ''),
      client_id: String(formData.client_id || ''),
      client_secret: String(formData.client_secret || ''),
      scopes: String(formData.scopes || 'openid profile email'),
      button_label: String(formData.button_label || ''),
      auto_provision: formData.auto_provision ? '1' : '0'
    })
    ElMessage.success(t('config.update_success'))
    formData.client_secret = ''
    await loadData()
  } catch (error) {
    console.error('Submit oidc config error:', error)
  } finally {
    submitting.value = false
  }
}

onMounted(() => { loadData() })
defineExpose({ loadData })
</script>