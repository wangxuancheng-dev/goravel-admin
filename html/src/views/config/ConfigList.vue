<template>
  <div class="list-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ $t('menu.config') }}</span>
        </div>
      </template>

      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <el-tab-pane :label="$t('config.website_config')" name="website">
          <WebsiteConfig ref="websiteConfigRef" />
        </el-tab-pane>
        <el-tab-pane :label="$t('config.customer_service_config')" name="customer_service">
          <CustomerServiceConfig ref="customerServiceConfigRef" />
        </el-tab-pane>
        <el-tab-pane :label="$t('config.email_config')" name="email">
          <EmailConfig ref="emailConfigRef" />
        </el-tab-pane>
        <el-tab-pane :label="$t('config.captcha_config')" name="captcha">
          <CaptchaConfig ref="captchaConfigRef" />
        </el-tab-pane>
        <el-tab-pane :label="$t('config.storage_config')" name="storage">
          <StorageConfig ref="storageConfigRef" />
        </el-tab-pane>
        <el-tab-pane :label="$t('config.login_security_config')" name="login_security">
          <LoginSecurityConfig ref="loginSecurityConfigRef" />
        </el-tab-pane>
        <el-tab-pane :label="$t('config.oidc_config')" name="oidc">
          <OidcConfig ref="oidcConfigRef" />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import WebsiteConfig from './components/WebsiteConfig.vue'
import CustomerServiceConfig from './components/CustomerServiceConfig.vue'
import EmailConfig from './components/EmailConfig.vue'
import CaptchaConfig from './components/CaptchaConfig.vue'
import StorageConfig from './components/StorageConfig.vue'
import LoginSecurityConfig from './components/LoginSecurityConfig.vue'
import OidcConfig from './components/OidcConfig.vue'

const activeTab = ref('website')
const websiteConfigRef = ref(null)
const customerServiceConfigRef = ref(null)
const emailConfigRef = ref(null)
const captchaConfigRef = ref(null)
const storageConfigRef = ref(null)
const loginSecurityConfigRef = ref(null)
const oidcConfigRef = ref(null)

const handleTabChange = (tabName) => {
  // 切换tab时可以重新加载数据
  if (tabName === 'website' && websiteConfigRef.value) {
    websiteConfigRef.value.loadData()
  } else if (tabName === 'customer_service' && customerServiceConfigRef.value) {
    customerServiceConfigRef.value.loadData()
  } else if (tabName === 'email' && emailConfigRef.value) {
    emailConfigRef.value.loadData()
  } else if (tabName === 'captcha' && captchaConfigRef.value) {
    captchaConfigRef.value.loadData()
  } else if (tabName === 'storage' && storageConfigRef.value) {
    storageConfigRef.value.loadData()
  } else if (tabName === 'login_security' && loginSecurityConfigRef.value) {
    loginSecurityConfigRef.value.loadData()
  } else if (tabName === 'oidc' && oidcConfigRef.value) {
    oidcConfigRef.value.loadData()
  }
}
</script>

<style scoped>

</style>

