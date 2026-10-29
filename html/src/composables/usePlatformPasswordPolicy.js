import { ref } from 'vue'
import { platformInfo } from '@/api/platform'
import { DEFAULT_PASSWORD_POLICY, normalizePasswordPolicy } from '@/utils/passwordPolicy'

let request = null

/** 平台管理员的密码策略（取自 /platform/info，按页面会话缓存）。 */
export function usePlatformPasswordPolicy() {
  const policy = ref({ ...DEFAULT_PASSWORD_POLICY })
  request ??= platformInfo()
    .then((res) => normalizePasswordPolicy(res?.data?.password_policy))
    .catch(() => {
      request = null
      return { ...DEFAULT_PASSWORD_POLICY }
    })
  request.then((p) => {
    policy.value = p
  })
  return policy
}
