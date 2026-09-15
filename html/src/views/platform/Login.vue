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
        <el-button type="primary" size="large" class="submit" :loading="loading" @click="submit">
          {{ $t('login.login') }}
        </el-button>
      </el-form>
      <div class="footer">
        <router-link to="/login">{{ $t('platform.back_tenant_login') }}</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { completePlatformLogin, platformLogin } from '@/api/platform'

const { t } = useI18n()
const router = useRouter()
const formRef = ref(null)
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const rules = computed(() => ({
  username: [{ required: true, message: t('login.username'), trigger: 'blur' }],
  password: [{ required: true, message: t('login.password'), trigger: 'blur' }]
}))

const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      const res = await platformLogin({
        username: form.username.trim(),
        password: form.password
      })
      await completePlatformLogin(res)
      ElMessage.success(t('login.login_success') || t('common.success'))
      router.replace('/platform/overview')
    } catch (error) {
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
  font-size: 22px;
  color: #0f172a;
}
.subtitle {
  margin: 8px 0 24px;
  color: #64748b;
  font-size: 13px;
}
.submit {
  width: 100%;
}
.footer {
  margin-top: 18px;
  text-align: center;
  font-size: 13px;
}
</style>
