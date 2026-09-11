<template>
  <div class="login-page">

    <div class="login-page__left">
      <div class="login-page__brand">
        <div class="brand-logo">
          <el-icon :size="40"><Lock /></el-icon>
        </div>
        <h1 class="brand-title">{{ $t('login.title') }}</h1>
        <p class="brand-desc">{{ $t('login.page_description') }}</p>
      </div>
      <div class="login-page__deco">
        <span class="deco-circle deco-circle--1" />
        <span class="deco-circle deco-circle--2" />
        <span class="deco-circle deco-circle--3" />
        <span class="deco-line deco-line--1" />
        <span class="deco-line deco-line--2" />
      </div>
    </div>
    <!-- 右侧：表单区 -->
    <div class="login-page__right">
      <div class="login-page__toolbar">
        <div class="login-toolbar__theme">
          <button
            v-for="t in themeColorOptions"
            :key="t.key"
            type="button"
            class="login-theme-swatch"
            :class="{ active: appStore.themeColor === t.key }"
            :style="{ backgroundColor: t.color }"
            :title="t.key"
            @click="appStore.setThemeColor(t.key)"
          />
        </div>
        <DarkModeSwitch class="login-toolbar-dark" />
        <LanguageSwitch class="login-toolbar-switch" />
      </div>
      <div class="login-page__form-wrap">
        <div class="login-form-card">
          <h2 class="login-form-title">{{ $t('login.login') }}</h2>
          <p class="login-form-subtitle">{{ $t('login.page_description') }}</p>
          <el-form
            ref="loginFormRef"
            :model="loginForm"
            :rules="loginRules"
            class="login-form"
          >
            <el-form-item v-if="tenancyEnabled" prop="tenant_code">
              <el-input
                v-model="loginForm.tenant_code"
                :placeholder="$t('login.tenant_code')"
                size="large"
                class="login-input"
                @change="onTenantCodeChange"
              />
            </el-form-item>
            <el-form-item prop="username">
              <el-input
                v-model="loginForm.username"
                :placeholder="$t('login.username')"
                size="large"
                prefix-icon="User"
                class="login-input"
              />
            </el-form-item>
            <el-form-item prop="password">
              <el-input
                v-model="loginForm.password"
                type="password"
                :placeholder="$t('login.password')"
                size="large"
                prefix-icon="Lock"
                class="login-input"
                show-password
                @keyup.enter="handleLogin"
              />
            </el-form-item>
            <el-form-item v-if="needGoogleCode" prop="google_code">
              <el-input
                v-model="loginForm.google_code"
                :placeholder="$t('login.google_code_placeholder')"
                size="large"
                class="login-input"
                maxlength="6"
                @keyup.enter="handleLogin"
              />
            </el-form-item>
            <el-form-item v-if="captchaInfo.shouldShow && !needGoogleCode" prop="captcha_answer">
              <div class="captcha-row">
                <img
                  v-if="captchaInfo.image"
                  :src="captchaInfo.image"
                  class="captcha-image"
                  :alt="$t('login.captcha_alt')"
                  @click.prevent="fetchCaptcha"
                />
                <el-button
                  class="captcha-refresh"
                  type="primary"
                  size="small"
                  text
                  @click.prevent="fetchCaptcha"
                >
                  <el-icon class="refresh-icon"><Refresh /></el-icon>
                  <span>{{ $t('login.refresh_captcha') }}</span>
                </el-button>
              </div>
              <el-input
                v-model="loginForm.captcha_answer"
                :placeholder="$t('login.captcha_placeholder')"
                size="large"
                class="login-input"
                @keyup.enter="handleLogin"
              />
            </el-form-item>
            <el-form-item>
              <el-button
                type="primary"
                size="large"
                class="login-button"
                :loading="loading"
                @click="handleLogin"
              >
                {{ $t('login.login') }}
              </el-button>
            </el-form-item>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Lock, Refresh } from '@element-plus/icons-vue'
import { login, getLoginCaptcha } from '../api/auth'
import { useUserStore } from '../store/user'
import { useAppStore, THEME_COLORS } from '../store/app'
import LanguageSwitch from '../components/LanguageSwitch.vue'
import DarkModeSwitch from '../components/DarkModeSwitch.vue'
import { ERROR_CODES } from '../utils/request'
import Storage from '../utils/storage'
import {
  getTenantCode,
  isTenancyEnabled,
  resolveTenantCodeFromLocation,
  setTenantCode
} from '../utils/tenant'

const appStore = useAppStore()
const themeColorOptions = THEME_COLORS

const router = useRouter()
const userStore = useUserStore()
const { t } = useI18n()

const tenancyEnabled = isTenancyEnabled()
const loginFormRef = ref(null)
const loading = ref(false)

const loginForm = reactive({
  tenant_code: '',
  username: '',
  password: '',
  captcha_answer: '',
  google_code: ''
})

const captchaInfo = reactive({
  enabled: false,
  captcha_id: '',
  image: '',
  shouldShow: false // 是否应该显示图形验证码（需要先验证账号密码后才能确定）
})

const needGoogleCode = ref(false)

const loginRules = computed(() => ({
  tenant_code: tenancyEnabled
    ? [{ required: true, message: t('login.tenant_code_required'), trigger: 'blur' }]
    : [],
  username: [
    { required: true, message: t('login.username_required'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('login.password_required'), trigger: 'blur' }
  ],
  google_code: needGoogleCode.value
    ? [
        { required: true, message: t('login.google_code_required'), trigger: 'blur' },
        { pattern: /^\d{6}$/, message: t('login.google_code_format'), trigger: 'blur' }
      ]
    : [],
  captcha_answer: captchaInfo.shouldShow && !needGoogleCode.value
    ? [{ required: true, message: t('login.captcha_required'), trigger: 'blur' }]
    : []
}))

const onTenantCodeChange = () => {
  setTenantCode(loginForm.tenant_code)
  if (tenancyEnabled && loginForm.tenant_code) {
    checkCaptchaEnabled()
  }
}

// 获取图形验证码配置（不自动获取图片）
const checkCaptchaEnabled = async () => {
  try {
    if (tenancyEnabled) {
      setTenantCode(loginForm.tenant_code)
    }
    const res = await getLoginCaptcha({ check: true })
    const captcha = res.data?.captcha || {}
    captchaInfo.enabled = !!captcha.enabled
    // 不自动显示图形验证码，需要先验证账号密码
    captchaInfo.shouldShow = false
  } catch (error) {
    console.error('Check captcha enabled error:', error)
    captchaInfo.enabled = false
    captchaInfo.shouldShow = false
  }
}

// 获取图形验证码图片（当需要显示时才获取）
const fetchCaptcha = async () => {
  try {
    if (tenancyEnabled) {
      setTenantCode(loginForm.tenant_code)
    }
    const res = await getLoginCaptcha()
    const captcha = res.data?.captcha || {}
    captchaInfo.enabled = !!captcha.enabled
    captchaInfo.captcha_id = captcha.captcha_id || ''
    captchaInfo.image = captcha.captcha_image || ''
    captchaInfo.shouldShow = true
  } catch (error) {
    console.error('Fetch captcha error:', error)
    captchaInfo.enabled = false
    captchaInfo.captcha_id = ''
    captchaInfo.image = ''
    captchaInfo.shouldShow = false
  } finally {
    loginForm.captcha_answer = ''
    if (loginFormRef.value) {
      loginFormRef.value.clearValidate(['captcha_answer'])
    }
  }
}

onMounted(() => {
  const fromQuery = resolveTenantCodeFromLocation()
  const initialTenant = fromQuery || getTenantCode()
  if (initialTenant) {
    loginForm.tenant_code = initialTenant
    setTenantCode(initialTenant)
  }
  // 只检查图形验证码是否启用，不自动获取图片
  checkCaptchaEnabled()
})

const handleLogin = async () => {
  if (!loginFormRef.value) return
  
  // 先验证账号密码（不包含图形验证码）
  // 如果绑定了 2FA，后端会返回 google_code_required
  // 如果没有绑定 2FA 且图形验证码开启，后端会返回需要图形验证码的错误
  await loginFormRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true
      try {
        if (tenancyEnabled) {
          setTenantCode(loginForm.tenant_code)
        }
        const payload = {
          username: loginForm.username,
          password: loginForm.password
        }
        if (tenancyEnabled && loginForm.tenant_code) {
          payload.tenant_code = String(loginForm.tenant_code).trim().toLowerCase()
        }
        
        // 如果已经需要谷歌验证码，添加谷歌验证码
        if (needGoogleCode.value) {
          payload.google_code = loginForm.google_code
        }
        // 如果图形验证码应该显示，添加图形验证码
        else if (captchaInfo.shouldShow) {
          payload.captcha_id = captchaInfo.captcha_id
          payload.captcha_answer = loginForm.captcha_answer
        }
        // 否则，先只提交账号密码，让后端判断是否需要图形验证码或谷歌验证码
        
        const res = await login(payload)
        if (res.data && res.data.token) {
          const token = res.data.token
          // 登录时清除旧的数据，确保获取最新的数据
          userStore.menus = []
          userStore.adminInfo = null
          userStore.permissions = []
          Storage.removeItem('adminInfo')
          
          userStore.setToken(token)
          // 注意：登录接口返回的 admin 信息可能不完整，所以先不设置
          // 等待 fetchUserInfo() 获取完整的管理员信息（包括权限和菜单）
          // 等待一下确保token已保存
          await new Promise(resolve => setTimeout(resolve, 100))
          await userStore.fetchUserInfo()
          ElMessage.success(t('login.login_success'))
          router.push('/')
        } else {
          throw new Error(t('login.login_failed'))
        }
      } catch (error) {
        if (error?.__handled) {
          // 已在 axios 拦截器中提示
          return
        }
        
        // 使用错误码判断，简化逻辑
        // 优先从 error.errorCode 获取，如果没有则从 response.data.error_code 获取
        const errorCode = error.errorCode || error.response?.data?.error_code || ''
        const message = error.translatedMessage || error.message || error.response?.data?.message || ''
        
        // 根据错误码处理 UI 状态
        if (errorCode === ERROR_CODES.GOOGLE_CODE_REQUIRED) {
          // 绑定了 2FA，需要谷歌验证码，隐藏图形验证码
          needGoogleCode.value = true
          captchaInfo.shouldShow = false
          loginForm.google_code = ''
          loginForm.captcha_answer = ''
          if (loginFormRef.value) {
            loginFormRef.value.clearValidate(['captcha_answer'])
          }
          ElMessage.warning(message)
          return
        }
        
        if (errorCode === ERROR_CODES.GOOGLE_CODE_INVALID) {
          ElMessage.error(message)
          loginForm.google_code = ''
          return
        }
        
        if (errorCode === ERROR_CODES.ACCOUNT_DISABLED) {
          ElMessage.error(message)
          return
        }
        
        // 检查是否是验证码相关的错误
        if (errorCode === ERROR_CODES.CAPTCHA_INVALID || errorCode === ERROR_CODES.CAPTCHA_REQUIRED) {
          // 验证码错误，如果图形验证码开启且还没有显示，则显示图形验证码
          if (captchaInfo.enabled && !captchaInfo.shouldShow && !needGoogleCode.value) {
            await fetchCaptcha()
          }
          ElMessage.error(message)
          return
        }
        
        // 其他错误（可能是密码错误等）
        // 如果图形验证码开启且还没有显示，则显示图形验证码
        if (captchaInfo.enabled && !captchaInfo.shouldShow && !needGoogleCode.value) {
          await fetchCaptcha()
        }
        
        // 显示错误消息
        ElMessage.error(message)
      } finally {
        loading.value = false
        // 如果图形验证码已显示且不需要谷歌验证码，刷新图形验证码
        if (captchaInfo.shouldShow && !needGoogleCode.value) {
          await fetchCaptcha()
        }
      }
    }
  })
}
</script>

<style scoped>

.login-page {
  display: flex;
  min-height: 100vh;
  overflow: hidden;
}


.login-page__left {
  position: relative;
  width: 52%;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;
  background: linear-gradient(180deg, #2f3f57 0%, #243246 100%);
  overflow: hidden;
}

.login-page__brand {
  position: relative;
  z-index: 2;
  max-width: 380px;
}

.brand-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 60px;
  height: 60px;
  background: rgba(255, 255, 255, 0.14);
  border-radius: 12px;
  color: #fff;
  margin-bottom: 22px;
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.14);
}

.brand-title {
  font-size: 30px;
  font-weight: 700;
  color: #fff;
  margin: 0 0 10px;
  letter-spacing: 0;
  line-height: 1.25;
}

.brand-desc {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.76);
  margin: 0;
  line-height: 1.55;
}

/* 左侧装饰（几何图形） */
.login-page__deco {
  display: none;
}

/* ========== 右侧表单区 ========== */
.login-page__right {
  flex: 1;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  background: color-mix(in srgb, var(--bg-color-tertiary) 90%, var(--card-bg) 10%);
}

.login-page__toolbar {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  padding: 14px 22px 0;
  margin: 0;
  border: none;
  background: transparent;
  box-shadow: none;
  width: min(520px, calc(100% - 32px));
}

.login-toolbar__theme {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-right: 4px;
}

.login-theme-swatch {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  border: 2px solid transparent;
  cursor: pointer;
  padding: 0;
  flex-shrink: 0;
  transition: transform 0.15s ease, border-color 0.15s ease;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.login-theme-swatch:hover {
  transform: scale(1.12);
}

.login-theme-swatch.active {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px var(--card-bg);
}

.login-toolbar-dark :deep(.dark-mode-switch) {
  padding: 6px 10px;
  min-width: 40px;
  min-height: 40px;
}

.login-toolbar-switch :deep(.language-switch) {
  padding: 8px 14px;
  border-radius: var(--border-radius-lg);
  border: 1px solid var(--border-color-light);
  background: var(--card-bg);
  transition: all 0.2s ease;
}

.login-toolbar-switch :deep(.language-switch):hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

.login-page__form-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: min(560px, 100%);
  padding: 20px 24px 32px;
}

.login-form-card {
  width: 100%;
  max-width: 100%;
  padding: 40px 34px;
  background: var(--card-bg);
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, var(--border-color-light) 74%, transparent);
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.08);
}

.login-form-title {
  font-size: 22px;
  font-weight: 600;
  color: var(--text-color-primary);
  margin: 0;
  text-align: center;
  letter-spacing: 0;
}

.login-form-subtitle {
  margin: 8px 0 24px;
  text-align: center;
  font-size: 13px;
  line-height: 1.5;
  color: var(--text-color-secondary);
}

.login-form {
  margin-top: 0;
}

.login-form :deep(.el-form-item) {
  margin-bottom: 22px;
}

.login-form :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}

.login-input :deep(.el-input__wrapper) {
  border-radius: var(--border-radius-lg);
  box-shadow: 0 0 0 1px var(--border-color-light) inset;
  transition: all 0.2s ease;
  background: var(--card-bg);
  padding: 0 14px;
}

.login-input :deep(.el-input__wrapper):hover {
  box-shadow: 0 0 0 1px var(--text-color-placeholder) inset;
}

.login-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--el-color-primary) 30%, transparent) inset;
}

.login-input :deep(.el-input__inner) {
  font-size: 15px;
  color: var(--text-color-primary);
  height: 44px;
  line-height: 44px;
}

.login-input :deep(.el-input__inner::placeholder) {
  color: var(--text-color-secondary);
}

.login-button {
  width: 100%;
  height: 46px;
  font-size: 16px;
  font-weight: 500;
  border-radius: var(--border-radius-lg);
  background: var(--el-color-primary);
  border: none;
  box-shadow: 0 4px 10px color-mix(in srgb, var(--el-color-primary) 24%, transparent);
  transition: all 0.2s ease;
  margin-top: 8px;
}

.login-button:hover {
  background: var(--el-color-primary-light-3);
  box-shadow: 0 6px 14px color-mix(in srgb, var(--el-color-primary) 30%, transparent);
}

.login-button:active {
  background: var(--el-color-primary-dark-2);
  box-shadow: 0 2px 6px color-mix(in srgb, var(--el-color-primary) 30%, transparent);
}

.captcha-row {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 12px;
  margin-bottom: 12px;
}

.captcha-image {
  height: 40px;
  width: 170px;
  object-fit: cover;
  cursor: pointer;
  border-radius: var(--border-radius-lg);
  border: 1px solid var(--border-color-light);
  transition: all 0.2s ease;
}

.captcha-image:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px var(--el-color-primary);
}

.captcha-refresh {
  white-space: nowrap;
  padding: 0 8px;
  transition: all 0.2s ease;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.captcha-refresh:hover {
  color: var(--el-color-primary);
}

.captcha-refresh .refresh-icon {
  font-size: 16px;
  transition: transform 0.3s ease;
}

.captcha-refresh:hover .refresh-icon {
  transform: rotate(180deg);
}

/* ========== 响应式：小屏改为上下布局 ========== */
@media (max-width: 1024px) {
  .login-page {
    flex-direction: column;
  }

  .login-page__left {
    width: 100%;
    min-height: 230px;
    padding: 32px 24px;
  }

  .login-page__brand {
    max-width: 100%;
    text-align: center;
  }

  .brand-logo {
    margin-left: auto;
    margin-right: auto;
    margin-bottom: 20px;
  }

  .brand-title {
    font-size: 26px;
  }

  .brand-desc {
    font-size: 14px;
  }

  .login-page__right {
    min-height: auto;
  }

  .login-page__form-wrap {
    padding: 32px 24px;
  }

  .login-page__toolbar {
    padding: 12px 16px 0;
    width: calc(100% - 24px);
  }

  .login-form-card {
    padding: 36px 28px;
  }

  .login-form-title {
    font-size: 22px;
    margin-bottom: 28px;
  }
}

@media (max-width: 480px) {
  .login-page__left {
    min-height: 220px;
  }

  .brand-logo {
    width: 56px;
    height: 56px;
  }

  .brand-logo .el-icon {
    font-size: 28px;
  }

  .brand-title {
    font-size: 22px;
  }

  .login-form-card {
    padding: 28px 20px;
  }
}
</style>

