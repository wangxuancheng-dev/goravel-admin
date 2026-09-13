<template>
  <div class="platform-layout">
    <header class="platform-header">
      <div class="brand">{{ $t('platform.title') }}</div>
      <div class="actions">
        <span class="admin-name">{{ adminName }}</span>
        <el-button link type="primary" @click="pwdVisible = true">{{ $t('platform.change_password') }}</el-button>
        <el-button link type="primary" @click="onLogout">{{ $t('header.logout') }}</el-button>
      </div>
    </header>
    <div class="platform-body">
      <aside class="platform-aside">
        <el-menu :default-active="active" router>
          <el-menu-item index="/platform/tenants">{{ $t('menu.tenant') }}</el-menu-item>
          <el-menu-item index="/platform/tenant-op-logs">{{ $t('menu.tenant_op_log') }}</el-menu-item>
        </el-menu>
      </aside>
      <main class="platform-main">
        <router-view />
      </main>
    </div>

    <el-dialog v-model="pwdVisible" :title="$t('platform.change_password')" width="420px" destroy-on-close @closed="resetPwd">
      <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="100px">
        <el-form-item :label="$t('platform.old_password')" prop="old_password">
          <el-input v-model="pwdForm.old_password" type="password" show-password />
        </el-form-item>
        <el-form-item :label="$t('platform.new_password')" prop="new_password">
          <el-input v-model="pwdForm.new_password" type="password" show-password />
        </el-form-item>
        <el-form-item :label="$t('platform.confirm_password')" prop="confirm_password">
          <el-input v-model="pwdForm.confirm_password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="pwdSaving" @click="submitPwd">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { logoutPlatform, updatePlatformPassword } from '@/api/platform'
import { getPlatformAdmin } from '@/utils/platformRequest'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const active = computed(() => route.path)
const adminName = computed(() => {
  const admin = getPlatformAdmin()
  return admin?.name || admin?.username || ''
})

const pwdVisible = ref(false)
const pwdSaving = ref(false)
const pwdFormRef = ref()
const pwdForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: '',
})
const pwdRules = {
  old_password: [{ required: true, message: () => t('platform.old_password'), trigger: 'blur' }],
  new_password: [
    { required: true, message: () => t('platform.new_password'), trigger: 'blur' },
    { min: 6, message: () => t('platform.new_password'), trigger: 'blur' },
  ],
  confirm_password: [
    { required: true, message: () => t('platform.confirm_password'), trigger: 'blur' },
    {
      validator: (_r, v, cb) => {
        if (v !== pwdForm.new_password) cb(new Error(t('platform.confirm_password')))
        else cb()
      },
      trigger: 'blur',
    },
  ],
}

const resetPwd = () => {
  pwdForm.old_password = ''
  pwdForm.new_password = ''
  pwdForm.confirm_password = ''
}

const submitPwd = async () => {
  await pwdFormRef.value?.validate()
  pwdSaving.value = true
  try {
    await updatePlatformPassword({ ...pwdForm })
    ElMessage.success(t('platform.password_updated'))
    pwdVisible.value = false
  } catch (e) {
    if (!e?.__handled) ElMessage.error(e?.message || t('common.operation_failed'))
  } finally {
    pwdSaving.value = false
  }
}

const onLogout = async () => {
  await logoutPlatform()
  router.replace('/platform/login')
}
</script>

<style scoped>
.platform-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f5f7fb;
}
.platform-header {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  background: #0f172a;
  color: #fff;
}
.brand {
  font-weight: 600;
  letter-spacing: 0.02em;
}
.actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.admin-name {
  opacity: 0.85;
  font-size: 13px;
}
.platform-body {
  flex: 1;
  display: flex;
  min-height: 0;
}
.platform-aside {
  width: 200px;
  background: #fff;
  border-right: 1px solid #e5e7eb;
}
.platform-main {
  flex: 1;
  padding: 16px;
  overflow: auto;
}
</style>
