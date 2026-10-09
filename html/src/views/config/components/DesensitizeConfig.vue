<template>
  <div class="desensitize-config">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      style="margin-bottom: 16px"
      :title="$t('config.desensitize_tip')"
    />
    <el-form
      ref="formRef"
      :model="formData"
      label-width="160px"
      label-position="left"
    >
      <el-form-item :label="$t('config.desensitize_enabled')" prop="enabled">
        <el-switch
          v-model="formData.enabled"
          :active-text="$t('common.enabled')"
          :inactive-text="$t('common.disabled')"
        />
      </el-form-item>

      <el-form-item :label="$t('config.desensitize_bypass_roles')" prop="bypass_role_slugs">
        <el-input
          v-model="formData.bypass_role_slugs"
          :placeholder="$t('config.desensitize_bypass_roles_placeholder')"
        />
        <div class="tip">{{ $t('config.desensitize_bypass_roles_tip') }}</div>
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
  enabled: true,
  bypass_role_slugs: 'super-admin'
})

const loadData = async () => {
  try {
    const res = await getConfigByGroup('desensitize')
    if (res.data && res.data.configs) {
      const configs = res.data.configs
      configs.forEach(config => {
        const key = config.Key || config.key
        let value = config.Value || config.value || ''
        if (key === 'enabled') {
          formData.enabled = value === '1' || value === 'true' || value === true
        } else if (key === 'bypass_role_slugs' && value) {
          formData.bypass_role_slugs = value
        }
      })
    }
  } catch (error) {
    console.error('Load desensitize config error:', error)
  }
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    await saveConfig('desensitize', {
      enabled: formData.enabled ? '1' : '0',
      bypass_role_slugs: formData.bypass_role_slugs || 'super-admin'
    })
    ElMessage.success(t('config.update_success'))
  } catch (error) {
    console.error('Submit desensitize config error:', error)
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadData()
})

defineExpose({
  loadData
})
</script>

<style scoped>
.desensitize-config {
  padding: 20px 0;
}
.tip {
  margin-top: 6px;
  color: var(--text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}
</style>
