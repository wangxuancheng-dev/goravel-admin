<template>
  <div class="platform-login">
    <div class="platform-login__card">
      <h1>{{ $t('platform.title') }}</h1>
      <p class="subtitle">{{ $t('platform.login_hint') }}</p>
      <el-form ref="formRef" :model="form" :rules="rules" @submit.prevent>
        <el-form-item prop="username">
          <el-input v-model="form.username" :placeholder="$t('login.username')" size="large" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            :placeholder="$t('login.password')"
            size="large"
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-form-item v-if="needGoogleCode" prop="google_code">
          <el-input
            v-model="form.google_code"
            :placeholder="$t('platform.google_code_placeholder')"
            size="large"
            maxlength="6"
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-form-item v-if="showCaptcha && !needGoogleCode" prop="captcha_answer">
          <div class="captcha-row">
            <img
              v-if="captcha.image"
              :src="captcha.image"
              class="captcha-image"
              :alt="$t('login.captcha_alt')"
              @click.prevent="fetchCaptcha"
            />
            <el-button class="captcha-refresh" type="primary" size="small" text @click.prevent="fetchCaptcha">
              {{ $t('login.refresh_captcha') }}
            </el-button>
          </div>
          <el-input
            v-model="form.captcha_answer"
            :placeholder="$t('login.captcha_placeholder')"
            size="large"
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-button type="primary" size="large" class="submit" :loading="loading" @click="submit">
          {{ $t('login.login') }}
        </el-button>
      </el-form>
      <div class="footer hint">demo / demo123</div>
      <div class="footer">
        <a :href="tenantLoginUrl">{{ $t('platform.back_tenant_login') }}</a>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { completePlatformLogin, getPlatformLoginCaptcha, platformLogin } from '@/api/platform'
import { getTenantAdminLoginUrl } from '@/utils/tenant'

const { t } = useI18n()
const router = useRouter()
const formRef = ref(null)
const loading = ref(false)
const needGoogleCode = ref(false)
const showCaptcha = ref(false)
const tenantLoginUrl = getTenantAdminLoginUrl()
const form = reactive({ username: '', password: '', captcha_answer: '', google_code: '' })
const captcha = reactive({ id: '', image: '' })
const rules = computed(() => ({
  username: [{ required: true, message: t('login.username'), trigger: 'blur' }],
  password: [{ required: true, message: t('login.password'), trigger: 'blur' }],
  google_code: needGoogleCode.value
    ? [
        { required: true, message: t('login.google_code_required'), trigger: 'blur' },
        { pattern: /^\d{6}$/, message: t('login.google_code_format'), trigger: 'blur' }
      ]
    : [],
  captcha_answer:
    showCaptcha.value && !needGoogleCode.value
      ? [{ required: true, message: t('login.captcha_required'), trigger: 'blur' }]
      : []
}))

const fetchCaptcha = async () => {
  try {
    const res = await getPlatformLoginCaptcha()
    const info = res.data?.captcha || {}
    captcha.id = info.captcha_id || ''
    captcha.image = info.captcha_image || ''
    showCaptcha.value = true
    form.captcha_answer = ''
    formRef.value?.clearValidate?.(['captcha_answer'])
  } catch (error) {
    captcha.id = ''
    captcha.image = ''
    showCaptcha.value = true
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      const payload = {
        username: form.username.trim(),
        password: form.password
      }
      if (needGoogleCode.value) {
        payload.google_code = form.google_code
      } else if (showCaptcha.value) {
        payload.captcha_id = captcha.id
        payload.captcha_answer = form.captcha_answer
      }
      const res = await platformLogin(payload)
      await completePlatformLogin(res)
      ElMessage.success(t('login.login_success') || t('common.success'))
      router.replace('/platform/overview')
    } catch (error) {
      const code = error?.error_code || error?.errorCode || ''
      if (code === 'google_code_required') {
        needGoogleCode.value = true
        showCaptcha.value = false
        captcha.id = ''
        captcha.image = ''
        form.captcha_answer = ''
        form.google_code = ''
        ElMessage.warning(error?.translatedMessage || error?.message || t('login.google_code_required'))
        return
      }
      if (code === 'google_code_invalid') {
        form.google_code = ''
        if (!error?.__handled) {
          ElMessage.error(error?.translatedMessage || error?.message || t('login.login_failed'))
        }
        return
      }
      if (
        !needGoogleCode.value &&
        (code === 'captcha_required' || code === 'captcha_invalid' || code === 'captcha_expired')
      ) {
        await fetchCaptcha()
        if (code === 'captcha_required') {
          ElMessage.info(t('login.captcha_required'))
        } else if (!error?.__handled) {
          ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
        }
        return
      }
      if (showCaptcha.value && !needGoogleCode.value) {
        await fetchCaptcha()
      }
      if (!error?.__handled) {
        ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
      }
    } finally {
      loading.value = false
    }
  })
}
</script>

<style scoped>
.platform-login {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(145deg, #0f172a 0%, #1e293b 55%, #334155 100%);
  padding: 24px;
}
.platform-login__card {
  width: 100%;
  max-width: 420px;
  background: #fff;
  border-radius: 12px;
  padding: 36px 32px 28px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.25);
}
.platform-login__card h1 {
  margin: 0;
  font-size: 24px;
}
.subtitle {
  color: #64748b;
  margin: 8px 0 24px;
}
.captcha-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.captcha-image {
  height: 40px;
  border-radius: 4px;
  cursor: pointer;
  border: 1px solid #e2e8f0;
}
.submit {
  width: 100%;
  margin-top: 8px;
}
.footer {
  margin-top: 12px;
  text-align: center;
}
.footer.hint {
  color: #94a3b8;
  font-size: 13px;
}
</style>
