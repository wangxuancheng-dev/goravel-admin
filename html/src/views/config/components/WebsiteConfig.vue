<template>
  <div class="website-config">
    <el-form
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      label-position="left"
    >
      <el-form-item :label="$t('config.site_enabled')" prop="site_enabled">
        <el-switch
          v-model="formData.site_enabled"
          active-value="1"
          inactive-value="0"
          :active-text="$t('common.enabled')"
          :inactive-text="$t('common.disabled')"
        />
      </el-form-item>

      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item :label="$t('config.site_name')" prop="site_name">
            <el-input v-model="formData.site_name" :placeholder="$t('config.site_name_placeholder')" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item :label="$t('config.site_url')" prop="site_url">
            <el-input v-model="formData.site_url" :placeholder="$t('config.site_url_placeholder')" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item :label="$t('config.site_logo')" prop="site_logo">
        <AttachmentImageField
          v-model="formData.site_logo"
          :placeholder="$t('config.site_logo_placeholder')"
        />
      </el-form-item>

      <el-form-item :label="$t('config.site_icp')" prop="site_icp">
        <el-input v-model="formData.site_icp" :placeholder="$t('config.site_icp_placeholder')" style="max-width: 480px" />
      </el-form-item>

      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item :label="$t('config.site_keywords')" prop="site_keywords">
            <el-input v-model="formData.site_keywords" :placeholder="$t('config.site_keywords_placeholder')" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item :label="$t('config.site_description')" prop="site_description">
            <el-input v-model="formData.site_description" :placeholder="$t('config.site_description_placeholder')" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item :label="$t('config.site_copyright')" prop="site_copyright">
        <el-input v-model="formData.site_copyright" type="textarea" :rows="3" :placeholder="$t('config.site_copyright_placeholder')" />
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
import { forOwn } from 'lodash-es'
import { getConfigByGroup, saveConfig } from '../../../api/config'
import { usePermission } from '../../../composables/usePermission'
import AttachmentImageField from '../../../components/AttachmentImageField.vue'
import { notifyWebsiteConfigUpdated } from '../../../utils/publicImage'

const { t } = useI18n()
const { getButtonState } = usePermission()
const formRef = ref(null)
const submitting = ref(false)

const formData = reactive({
  site_enabled: '1',
  site_name: '',
  site_url: '',
  site_logo: '',
  site_icp: '',
  site_keywords: '',
  site_description: '',
  site_copyright: ''
})

const formRules = {}

const loadData = async () => {
  try {
    const res = await getConfigByGroup('website')
    if (res.data && res.data.configs) {
      const configs = res.data.configs
      // 将配置数组转换为表单对象
      configs.forEach(config => {
        const key = config.Key || config.key
        const value = config.Value || config.value || ''
        if (formData.hasOwnProperty(key)) {
          formData[key] = value
        }
      })
    }
  } catch (error) {
    console.error('Load website config error:', error)
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        // 将表单数据转换为配置对象
        const configs = {}
        forOwn(formData, (value, key) => {
          configs[key] = value
        })

        await saveConfig('website', configs)
        ElMessage.success(t('config.update_success'))
        notifyWebsiteConfigUpdated()
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

// 暴露方法供父组件调用
defineExpose({
  loadData
})
</script>

<style scoped>
.website-config {
  padding: 20px 0;
}
</style>

