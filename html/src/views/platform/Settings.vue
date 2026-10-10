<template>
  <div class="platform-settings">
    <el-card shadow="never" v-loading="loading">
      <template #header>{{ $t('platform.settings_captcha_title') }}</template>
      <el-alert
        v-if="isViewer"
        type="info"
        show-icon
        :closable="false"
        :title="$t('platform.settings_viewer_hint')"
        style="margin-bottom: 16px"
      />
      <el-form :model="form" label-width="120px" label-position="left" :disabled="isViewer">
        <el-form-item :label="$t('config.captcha_type')" prop="captcha_type">
          <el-select v-model="form.captcha_type" style="width: 240px">
            <el-option :label="$t('config.captcha_type_image')" value="image" />
            <el-option :label="$t('config.captcha_type_slide')" value="slide" />
          </el-select>
          <div class="tip">{{ $t('platform.settings_captcha_tip') }}</div>
        </el-form-item>
        <el-form-item v-if="!isViewer">
          <el-button type="primary" :loading="saving" @click="onSave">{{ $t('common.save') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
/** Landlord console settings (stored in platform_settings). */
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { getPlatformSettings, updatePlatformSettings } from '@/api/platform'
import { getPlatformAdmin } from '@/utils/platformRequest'

const { t } = useI18n()
const loading = ref(false)
const saving = ref(false)
const form = reactive({ captcha_type: 'image' })
const isViewer = computed(() => getPlatformAdmin()?.role === 'viewer')

const load = async () => {
  loading.value = true
  try {
    const res = await getPlatformSettings()
    form.captcha_type = res.data?.settings?.captcha_type === 'slide' ? 'slide' : 'image'
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    loading.value = false
  }
}

const onSave = async () => {
  saving.value = true
  try {
    await updatePlatformSettings({ captcha_type: form.captcha_type })
    ElMessage.success(t('config.update_success'))
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.platform-settings {
  padding: 4px;
}
.tip {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary, #64748b);
}
</style>
