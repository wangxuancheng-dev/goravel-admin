import { Form, Input } from 'antd'
import type { TFunction } from 'i18next'

type ConfirmFn = (config: {
  title?: React.ReactNode
  content?: React.ReactNode
  okText?: string
  cancelText?: string
  onOk?: () => void | Promise<void>
  onCancel?: () => void
}) => void

/**
 * Prompt for TOTP / current password used as confirm_code on sensitive admin ops.
 * Resolves with the code, or rejects if cancelled / empty.
 */
export function promptSensitiveConfirm(
  modal: { confirm: ConfirmFn },
  t: TFunction,
  options?: { title?: string },
): Promise<string> {
  let confirmCode = ''
  return new Promise((resolve, reject) => {
    modal.confirm({
      title: options?.title || t('common.sensitive_confirm'),
      content: (
        <Form layout="vertical" style={{ marginTop: 12 }}>
          <Form.Item
            label={t('common.sensitive_confirm_hint')}
            required
            style={{ marginBottom: 0 }}
          >
            <Input.Password
              autoFocus
              placeholder={t('common.sensitive_confirm_placeholder')}
              onChange={(e) => {
                confirmCode = e.target.value
              }}
            />
          </Form.Item>
        </Form>
      ),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: () => {
        const code = confirmCode.trim()
        if (!code) {
          return Promise.reject(new Error(t('common.sensitive_confirm_required')))
        }
        resolve(code)
      },
      onCancel: () => {
        reject(new Error('cancel'))
      },
    })
  })
}
