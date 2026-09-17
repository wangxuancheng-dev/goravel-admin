import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useUserStore } from '@/stores/user'
import { useTabsStore } from '@/stores/tabs'
import { buildAdminLoginPath } from '@/utils/tenant'

export function useLayoutLockScreen() {
  const navigate = useNavigate()
  const logout = useUserStore((s) => s.logout)
  const removeAllTabs = useTabsStore((s) => s.removeAllTabs)
  const { t } = useTranslation()

  const [isScreenLocked, setIsScreenLocked] = useState(false)
  const [lockPassword, setLockPassword] = useState('')
  const [unlockPassword, setUnlockPassword] = useState('')
  const [unlockError, setUnlockError] = useState('')
  const [lockDialogVisible, setLockDialogVisible] = useState(false)
  const [pendingLockPassword, setPendingLockPassword] = useState('')
  const [lockDialogError, setLockDialogError] = useState('')

  const [lockInputName] = useState(() => `lock-screen-password-${Math.random().toString(36).slice(2)}`)
  const [lockDialogInputName] = useState(() => `set-lock-password-${Math.random().toString(36).slice(2)}`)

  const handleLockScreen = () => {
    setPendingLockPassword('')
    setLockDialogError('')
    setLockDialogVisible(true)
  }

  const cancelLockDialog = () => {
    setLockDialogVisible(false)
  }

  const confirmLockScreen = () => {
    if (!pendingLockPassword || !pendingLockPassword.trim()) {
      setLockDialogError(t('header.lock_password_required'))
      return
    }

    setLockPassword(pendingLockPassword.trim())
    setUnlockPassword('')
    setUnlockError('')
    setPendingLockPassword('')
    setLockDialogError('')
    setLockDialogVisible(false)
    setIsScreenLocked(true)
  }

  const handleUnlockInput = (value: string) => {
    setUnlockPassword(value)
    if (unlockError) {
      setUnlockError('')
    }
  }

  const handleUnlockScreen = () => {
    if (!unlockPassword.trim()) {
      setUnlockError(t('header.lock_password_required'))
      return
    }

    if (unlockPassword !== lockPassword) {
      setUnlockError(t('header.lock_password_invalid'))
      return
    }

    setIsScreenLocked(false)
    setLockPassword('')
    setUnlockPassword('')
    setUnlockError('')
  }

  const goToLogin = async () => {
    const loginPath = buildAdminLoginPath()
    try {
      await logout()
    } finally {
      removeAllTabs()
      setIsScreenLocked(false)
      setLockPassword('')
      setUnlockPassword('')
      setUnlockError('')
      setPendingLockPassword('')
      setLockDialogError('')
      setLockDialogVisible(false)
      navigate(loginPath)
    }
  }

  return {
    isScreenLocked,
    lockDialogVisible,
    pendingLockPassword,
    setPendingLockPassword,
    lockDialogError,
    unlockPassword,
    unlockError,
    lockInputName,
    lockDialogInputName,
    handleLockScreen,
    cancelLockDialog,
    confirmLockScreen,
    handleUnlockInput,
    handleUnlockScreen,
    goToLogin,
  }
}

export type UseLayoutLockScreenReturn = ReturnType<typeof useLayoutLockScreen>
