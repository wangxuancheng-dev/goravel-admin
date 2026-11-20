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

    <el-card shadow="never" v-loading="loading" class="settings-card">
      <template #header>{{ $t('platform.settings_expiry_title') }}</template>
      <el-alert
        type="info"
        show-icon
        :closable="false"
        :title="$t('platform.settings_expiry_desc')"
        style="margin-bottom: 16px"
      />
      <el-form :model="expiry" label-width="120px" label-position="left" :disabled="isViewer">
        <el-form-item :label="$t('platform.settings_expiry_action')">
          <el-select v-model="expiry.action" style="width: 320px">
            <el-option :label="$t('platform.settings_expiry_action_downgrade')" value="downgrade" />
            <el-option :label="$t('platform.settings_expiry_action_suspend')" value="suspend" />
            <el-option :label="$t('platform.settings_expiry_action_notify_only')" value="notify_only" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('platform.settings_expiry_grace_days')">
          <el-input-number
            v-model="expiry.grace_days"
            :min="0"
            :max="90"
            :precision="0"
            :disabled="isViewer || expiry.action === 'notify_only'"
          />
          <div class="tip">{{ $t('platform.settings_expiry_grace_tip') }}</div>
        </el-form-item>
        <el-form-item :label="$t('platform.settings_expiry_remind_days')">
          <el-select v-model="expiry.remind_days" multiple :multiple-limit="5" style="width: 320px">
            <el-option
              v-for="d in remindDayOptions"
              :key="d"
              :label="$t('platform.settings_expiry_days_before', { n: d })"
              :value="d"
            />
          </el-select>
          <div class="tip">{{ $t('platform.settings_expiry_remind_tip') }}</div>
        </el-form-item>
        <el-form-item :label="$t('platform.settings_expiry_locale')">
          <el-select v-model="expiry.notice_locale" style="width: 240px">
            <el-option label="简体中文" value="cn" />
            <el-option label="English" value="en" />
          </el-select>
          <div class="tip">{{ $t('platform.settings_expiry_locale_tip') }}</div>
        </el-form-item>
        <el-form-item v-if="!isViewer">
          <el-button type="primary" :loading="savingExpiry" @click="onSaveExpiry">{{ $t('common.save') }}</el-button>
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
const savingExpiry = ref(false)
const form = reactive({ captcha_type: 'image' })
const remindDayOptions = [30, 14, 7, 3, 1]
const expiry = reactive({ action: 'downgrade', grace_days: 0, remind_days: [7, 3, 1], notice_locale: 'cn' })
const isViewer = computed(() => getPlatformAdmin()?.role === 'viewer')

const showError = (error) => {
  if (!error?.__handled) {
    ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
  }
}

const load = async () => {
  loading.value = true
  try {
    const res = await getPlatformSettings()
    const settings = res.data?.settings || {}
    form.captcha_type = settings.captcha_type === 'slide' ? 'slide' : 'image'
    if (settings.subscription_expiry) {
      Object.assign(expiry, settings.subscription_expiry, {
        remind_days: Array.isArray(settings.subscription_expiry.remind_days) ? settings.subscription_expiry.remind_days : []
      })
    }
  } catch (error) {
    showError(error)
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
    showError(error)
  } finally {
    saving.value = false
  }
}

const onSaveExpiry = async () => {
  savingExpiry.value = true
  try {
    await updatePlatformSettings({
      subscription_expiry: {
        action: expiry.action,
        grace_days: Number(expiry.grace_days) || 0,
        remind_days: expiry.remind_days.map(Number),
        notice_locale: expiry.notice_locale
      }
    })
    ElMessage.success(t('config.update_success'))
  } catch (error) {
    showError(error)
  } finally {
    savingExpiry.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.platform-settings {
  padding: 4px;
}
.settings-card {
  margin-top: 16px;
}
.tip {
  width: 100%;
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary, #64748b);
}
</style>
