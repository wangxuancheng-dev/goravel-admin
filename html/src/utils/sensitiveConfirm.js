import { ElMessageBox } from 'element-plus'

/**
 * Prompt for TOTP / current password used as confirm_code on sensitive admin ops.
 * @param {Function} t - vue-i18n t
 * @returns {Promise<string>} confirm_code
 */
export async function promptSensitiveConfirm(t) {
  const { value } = await ElMessageBox.prompt(
    t('common.sensitive_confirm_hint'),
    t('common.sensitive_confirm'),
    {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      inputType: 'password',
      inputPlaceholder: t('common.sensitive_confirm_placeholder'),
      inputValidator: (v) => {
        if (!v || !String(v).trim()) {
          return t('common.sensitive_confirm_required')
        }
        return true
      }
    }
  )
  return String(value).trim()
}

/**
 * Wrap a delete API so it prompts for confirm_code then sends it in the DELETE body.
 * @param {(id: string|number, data?: object) => Promise<any>} deleteApi
 * @param {Function} t
 */
export function withSensitiveDelete(deleteApi, t) {
  return async (id) => {
    const confirmCode = await promptSensitiveConfirm(t)
    return deleteApi(id, { confirm_code: confirmCode })
  }
}
