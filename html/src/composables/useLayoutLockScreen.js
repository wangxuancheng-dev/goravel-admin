import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/store/user'
import { useTabsStore } from '@/store/tabs'
import { buildAdminLoginPath } from '@/utils/tenant'

export function useLayoutLockScreen() {
  const router = useRouter()
  const userStore = useUserStore()
  const tabsStore = useTabsStore()
  const { t } = useI18n()

  const isScreenLocked = ref(false)
  const lockPassword = ref('')
  const unlockPassword = ref('')
  const unlockError = ref('')
  const lockInputName = `lock-screen-password-${Math.random().toString(36).slice(2)}`
  const lockDialogInputName = `set-lock-password-${Math.random().toString(36).slice(2)}`
  const lockDialogVisible = ref(false)
  const pendingLockPassword = ref('')
  const lockDialogError = ref('')

  const handleLockScreen = () => {
    pendingLockPassword.value = ''
    lockDialogError.value = ''
    lockDialogVisible.value = true
  }

  const confirmLockScreen = () => {
    if (!pendingLockPassword.value || !pendingLockPassword.value.trim()) {
      lockDialogError.value = t('header.lock_password_required')
      return
    }

    lockPassword.value = pendingLockPassword.value.trim()
    unlockPassword.value = ''
    unlockError.value = ''
    pendingLockPassword.value = ''
    lockDialogError.value = ''
    lockDialogVisible.value = false
    isScreenLocked.value = true
  }

  const handleUnlockInput = () => {
    if (unlockError.value) {
      unlockError.value = ''
    }
  }

  const handleUnlockScreen = () => {
    if (!unlockPassword.value.trim()) {
      unlockError.value = t('header.lock_password_required')
      return
    }

    if (unlockPassword.value !== lockPassword.value) {
      unlockError.value = t('header.lock_password_invalid')
      return
    }

    isScreenLocked.value = false
    lockPassword.value = ''
    unlockPassword.value = ''
    unlockError.value = ''
  }

  const goToLogin = async () => {
    const loginPath = buildAdminLoginPath()
    try {
      await userStore.logout()
    } finally {
      tabsStore.removeAllTabs()
      isScreenLocked.value = false
      lockPassword.value = ''
      unlockPassword.value = ''
      unlockError.value = ''
      pendingLockPassword.value = ''
      lockDialogError.value = ''
      lockDialogVisible.value = false
      router.push(loginPath)
    }
  }

  return {
    isScreenLocked,
    lockDialogVisible,
    pendingLockPassword,
    lockDialogError,
    unlockPassword,
    unlockError,
    lockInputName,
    lockDialogInputName,
    handleLockScreen,
    confirmLockScreen,
    handleUnlockInput,
    handleUnlockScreen,
    goToLogin
  }
}
