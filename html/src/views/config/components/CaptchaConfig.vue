<template>
  <div class="captcha-config">
    <el-form
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      label-position="left"
    >
      <el-form-item :label="$t('config.captcha_enabled')" prop="captcha_enabled">
        <el-switch
          v-model="formData.captcha_enabled"
          :active-text="$t('common.enabled')"
          :inactive-text="$t('common.disabled')"
        />
        <span style="margin-left: 10px; color: var(--text-color-secondary);">{{ $t('config.captcha_enabled_tip') }}</span>
      </el-form-item>

      <el-form-item :label="$t('config.captcha_type')" prop="captcha_type">
        <el-select v-model="formData.captcha_type" style="width: 220px">
          <el-option :label="$t('config.captcha_type_image')" value="image" />
          <el-option :label="$t('config.captcha_type_slide')" value="slide" />
        </el-select>
        <span style="margin-left: 10px; color: var(--text-color-secondary);">{{ $t('config.captcha_type_tip') }}</span>
      </el-form-item>

      <el-form-item :label="$t('config.captcha_expire')" prop="captcha_expire">
        <el-input-number
          v-model="formData.captcha_expire"
          :min="30"
          :max="600"
          :placeholder="$t('config.captcha_expire_placeholder')"
        />
        <span style="margin-left: 10px; color: var(--text-color-secondary);">{{ $t('config.captcha_expire_unit') }}</span>
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
  captcha_enabled: false,
  captcha_type: 'image',
  captcha_expire: 120
})

const formRules = {
  captcha_expire: [
    { required: true, message: t('config.captcha_expire_required'), trigger: 'blur' },
    {
      type: 'number',
      min: 30,
      message: t('config.captcha_expire_min', { seconds: 30 }),
      trigger: 'blur'
    }
  ]
}

const loadData = async () => {
  try {
    const res = await getConfigByGroup('captcha')
    if (res.data && res.data.configs) {
      const configs = res.data.configs
      configs.forEach(config => {
        const key = config.Key || config.key
        let value = config.Value || config.value || ''
        
        if (key === 'captcha_enabled') {
          value = value === '1' || value === 'true' || value === true
        } else if (key === 'captcha_type') {
          value = value === 'slide' ? 'slide' : 'image'
        } else if (key === 'captcha_expire') {
          value = value ? parseInt(value) : 120
        }
        
        if (formData.hasOwnProperty(key)) {
          formData[key] = value
        }
      })
    }
  } catch (error) {
    console.error('Load captcha config error:', error)
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        const configs = {
          captcha_enabled: formData.captcha_enabled ? '1' : '0',
          captcha_type: formData.captcha_type === 'slide' ? 'slide' : 'image',
          captcha_expire: formData.captcha_expire
        }

        await saveConfig('captcha', configs)
        ElMessage.success(t('config.update_success'))
      } catch (error) {
        console.error('Submit error:', error)
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
.captcha-config {
  padding: 20px 0;
}
</style>

