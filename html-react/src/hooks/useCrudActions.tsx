import { useCallback, useState, type ReactNode } from 'react'
import { App, Button, Space } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import PermissionButton from '@/components/PermissionButton'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { promptSensitiveConfirm } from '@/utils/sensitiveConfirm'

interface UseCrudActionsOptions {
  createPermission?: string
  deletePermission?: string
  onRefresh: () => Promise<void> | void
  onCreate?: () => void
  deleteApi?: (id: string | number, data?: Record<string, unknown>) => Promise<unknown>
  /** When true, prompt for TOTP/password (confirm_code) before delete */
  requireSensitiveConfirm?: boolean
}

export function useCrudActions(options: UseCrudActionsOptions) {
  const { t } = useTranslation()
  const { modal, message } = App.useApp()
  const showError = useUnhandledError()
  const [deleting, setDeleting] = useState(false)

  const confirmDelete = useCallback(
    (id: string | number, name?: string) => {
      if (!options.deleteApi) return

      const runDelete = async (payload?: Record<string, unknown>) => {
        setDeleting(true)
        try {
          await options.deleteApi?.(id, payload)
          message.success(t('common.delete_success'))
          await options.onRefresh()
        } catch (error) {
          showError(error, t('common.operation_failed'))
          throw error
        } finally {
          setDeleting(false)
        }
      }

      if (options.requireSensitiveConfirm) {
        void (async () => {
          try {
            const confirmCode = await promptSensitiveConfirm(modal, t, {
              title: name
                ? `${t('common.delete_confirm')} — ${name}`
                : t('common.delete_confirm'),
            })
            await runDelete({ confirm_code: confirmCode })
          } catch (error) {
            if ((error as Error)?.message === 'cancel') return
            if ((error as Error)?.message === t('common.sensitive_confirm_required')) {
              message.error(t('common.sensitive_confirm_required'))
            }
          }
        })()
        return
      }

      modal.confirm({
        title: t('common.delete_confirm'),
        content: name ? `${name}` : undefined,
        okType: 'danger',
        onOk: () => runDelete(),
      })
    },
    [modal, message, options, showError, t],
  )

  const toolbar: ReactNode = (
    <Space>
      {options.onCreate && (
        <PermissionButton
          permission={options.createPermission || ''}
          type="primary"
          icon={<PlusOutlined />}
          onClick={options.onCreate}
        >
          {t('common.add')}
        </PermissionButton>
      )}
      <Button icon={<ReloadOutlined />} onClick={() => void options.onRefresh()}>
        {t('common.refresh')}
      </Button>
    </Space>
  )

  return { toolbar, confirmDelete, deleting }
}
