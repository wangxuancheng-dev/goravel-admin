<template>
  <div class="platform-login">
    <div class="platform-login__toolbar">
      <DarkModeSwitch class="platform-login-dark" />
      <LanguageSwitch class="platform-login-lang" />
    </div>
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
        <el-form-item v-if="showCaptcha && !needGoogleCode && captcha.type === 'slide' && captcha.slide">
          <SlideCaptcha
            :key="captcha.id"
            :data="captcha.slide"
            @confirm="onSlideConfirm"
            @refresh="fetchCaptcha"
          />
        </el-form-item>
        <el-form-item v-if="showCaptcha && !needGoogleCode && captcha.type !== 'slide'" prop="captcha_answer">
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
import { reactive, ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { completePlatformLogin, getPlatformLoginCaptcha, platformLogin } from '@/api/platform'
import { getTenantAdminLoginUrl } from '@/utils/tenant'
import DarkModeSwitch from '@/components/DarkModeSwitch.vue'
import LanguageSwitch from '@/components/LanguageSwitch.vue'
import SlideCaptcha from '@/components/SlideCaptcha.vue'

const { t } = useI18n()
const router = useRouter()
const formRef = ref(null)
const loading = ref(false)
const needGoogleCode = ref(false)
const showCaptcha = ref(false)
const tenantLoginUrl = getTenantAdminLoginUrl()
const form = reactive({ username: '', password: '', captcha_answer: '', google_code: '' })
// type (image | slide) is decided by the backend env PLATFORM_CAPTCHA_TYPE
const captcha = reactive({ type: 'image', id: '', image: '', slide: null })
// Slide mode: x offset recorded after the user releases the slider (verified on login)
const slideAnswer = ref('')
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
    showCaptcha.value && !needGoogleCode.value && captcha.type !== 'slide'
      ? [{ required: true, message: t('login.captcha_required'), trigger: 'blur' }]
      : []
}))

const fetchCaptcha = async () => {
  try {
    const res = await getPlatformLoginCaptcha()
    const info = res.data?.captcha || {}
    const isSlide = info.type === 'slide'
    captcha.type = isSlide ? 'slide' : 'image'
    captcha.id = info.captcha_id || ''
    captcha.image = info.captcha_image || ''
    captcha.slide = isSlide
      ? {
          captcha_id: info.captcha_id || '',
          master_image: info.master_image || '',
          tile_image: info.tile_image || '',
          master_width: info.master_width || 0,
          master_height: info.master_height || 0,
          tile_width: info.tile_width || 0,
          tile_height: info.tile_height || 0,
          tile_x: info.tile_x || 0,
          tile_y: info.tile_y || 0
        }
      : null
    slideAnswer.value = ''
    showCaptcha.value = true
    form.captcha_answer = ''
    formRef.value?.clearValidate?.(['captcha_answer'])
  } catch (error) {
    captcha.type = 'image'
    captcha.id = ''
    captcha.image = ''
    captcha.slide = null
    slideAnswer.value = ''
    showCaptcha.value = true
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const onSlideConfirm = (answer) => {
  slideAnswer.value = answer
}

// Image mode keeps the legacy flow (captcha appears after the first submit).
// Slide mode shows the slider up front, so probe the configured type first.
onMounted(async () => {
  try {
    const res = await getPlatformLoginCaptcha({ check: true })
    if (res.data?.captcha?.type === 'slide') {
      await fetchCaptcha()
    }
  } catch {
    // ignore: the normal submit flow still requests a captcha when required
  }
})

const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    if (!needGoogleCode.value && showCaptcha.value && captcha.type === 'slide' && !slideAnswer.value) {
      ElMessage.warning(t('login.slide_required'))
      return
    }
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
        payload.captcha_answer = captcha.type === 'slide' ? slideAnswer.value : form.captcha_answer
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
        captcha.type = 'image'
        captcha.id = ''
        captcha.image = ''
        captcha.slide = null
        slideAnswer.value = ''
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
  position: relative;
}
.platform-login__toolbar {
  position: absolute;
  top: 20px;
  right: 20px;
  display: flex;
  align-items: center;
  gap: 4px;
}
.platform-login-dark :deep(.dark-mode-switch) {
  color: rgba(255, 255, 255, 0.85);
  min-width: 36px;
  min-height: 36px;
  padding: 6px;
}
.platform-login-dark :deep(.dark-mode-switch:hover) {
  background-color: rgba(255, 255, 255, 0.1);
  color: #fff;
}
.platform-login-lang :deep(.language-switch) {
  color: rgba(255, 255, 255, 0.85);
  min-width: 36px;
  min-height: 36px;
  padding: 6px;
}
.platform-login-lang :deep(.language-switch:hover) {
  background-color: rgba(255, 255, 255, 0.1);
  color: #fff;
}
.platform-login__card {
  width: 100%;
  max-width: 420px;
  background: var(--el-bg-color, #fff);
  border-radius: 12px;
  padding: 36px 32px 28px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.25);
  border: 1px solid var(--el-border-color-lighter, transparent);
}
.platform-login__card h1 {
  margin: 0;
  font-size: 24px;
  color: var(--el-text-color-primary, inherit);
}
.subtitle {
  color: var(--el-text-color-secondary, #64748b);
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
  border: 1px solid var(--el-border-color, #e2e8f0);
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
  color: var(--el-text-color-placeholder, #94a3b8);
  font-size: 13px;
}
</style>
