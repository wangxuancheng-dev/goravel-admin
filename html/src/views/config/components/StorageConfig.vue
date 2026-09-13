<template>
  <div class="storage-config">
    <el-alert
      v-if="tenancyEnabled"
      type="warning"
      :closable="false"
      show-icon
      style="margin-bottom: 20px;"
      :title="$t('config.storage_disk_locked_title')"
      :description="$t('config.storage_disk_locked_tip')"
    />
    <el-alert
      v-else
      type="info"
      :closable="false"
      show-icon
      style="margin-bottom: 20px;"
    >
      <template #title>
        <div>
          <p style="margin: 0 0 8px 0; font-weight: 500;">{{ $t('config.storage_config_title') }}</p>
          <ul style="margin: 0; padding-left: 20px;">
            <li>{{ $t('config.storage_config_desc_1') }}</li>
            <li>{{ $t('config.storage_config_desc_2') }}</li>
            <li>{{ $t('config.storage_config_desc_3') }}</li>
          </ul>
        </div>
      </template>
    </el-alert>

    <el-form
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="150px"
      label-position="left"
    >
      <el-row :gutter="20">
        <el-col v-if="!tenancyEnabled" :span="12">
          <el-form-item :label="$t('config.file_disk')" prop="file_disk">
            <el-select v-model="formData.file_disk" :placeholder="$t('config.file_disk_placeholder')">
              <el-option label="local" value="local" />
              <el-option label="s3" value="s3" />
              <el-option label="oss" value="oss" />
              <el-option label="cos" value="cos" />
              <el-option label="qiniu" value="qiniu" />
              <el-option label="minio" value="minio" />
            </el-select>
            <div style="margin-top: 8px; color: var(--text-color-secondary); font-size: 12px;">
              {{ $t('config.storage_config_default_tip') }}
            </div>
          </el-form-item>
        </el-col>
        <el-col :span="tenancyEnabled ? 24 : 12">
          <el-form-item :label="$t('config.export_format')" prop="export_format">
            <el-select v-model="formData.export_format" :placeholder="$t('config.export_format_placeholder')">
              <el-option label="csv" value="csv" />
              <el-option label="xlsx" value="xlsx" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item>
        <el-button type="primary" @click="handleSubmit" :loading="submitting" :disabled="getButtonState('config.save').disabled">
          {{ $t('common.save') }}
        </el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { getConfigByGroup, saveConfig } from '../../../api/config'
import { usePermission } from '../../../composables/usePermission'
import { isTenancyEnabled } from '../../../utils/tenant'

const { t } = useI18n()
const { getButtonState } = usePermission()
const formRef = ref(null)
const submitting = ref(false)
const tenancyEnabled = isTenancyEnabled()

const formData = reactive({
  file_disk: 'local',
  export_format: 'csv'
})

const formRules = computed(() => {
  const rules = {
    export_format: [
      { required: true, message: t('config.export_format_required'), trigger: 'change' }
    ]
  }
  if (!tenancyEnabled) {
    rules.file_disk = [
      { required: true, message: t('config.file_disk_required'), trigger: 'change' }
    ]
  }
  return rules
})

const loadData = async () => {
  try {
    const res = await getConfigByGroup('storage')
    if (res.data && res.data.configs) {
      const configs = res.data.configs
      configs.forEach(config => {
        const key = config.Key || config.key
        const value = config.Value || config.value || ''
        
        // 兼容旧的字段名：export_disk 和 storage_disk
        if (key === 'file_disk' || key === 'export_disk' || key === 'storage_disk') {
          formData.file_disk = value
        }
        if (key === 'export_format') {
          formData.export_format = value || 'csv'
        }
      })
    }
  } catch (error) {
    console.error('Load storage config error:', error)
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        const configs = {
          export_format: formData.export_format
        }
        if (!tenancyEnabled) {
          configs.file_disk = formData.file_disk
        }

        await saveConfig('storage', configs)
        ElMessage.success(t('config.update_success'))
      } catch (error) {
        console.error('Submit error:', error)
        ElMessage.error(error?.translatedMessage || error?.message || t('config.update_failed'))
      } finally {
        submitting.value = false
      }
    }
  })
}

onMounted(() => {
  loadData()
})

defineExpose({
  loadData
})
</script>

<style scoped>
.storage-config {
  padding: 20px 0;
}

code {
  font-family: 'Courier New', monospace;
  font-size: 13px;
}
</style>
