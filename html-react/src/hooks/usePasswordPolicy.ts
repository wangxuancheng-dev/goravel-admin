import { useEffect, useMemo, useState } from 'react'
import type { Rule } from 'antd/es/form'
import { useTranslation } from 'react-i18next'
import { platformInfo } from '@/api/platform'
import { useUserStore } from '@/stores/user'
import {
  DEFAULT_PASSWORD_POLICY,
  describePasswordPolicy,
  normalizePasswordPolicy,
  passwordPolicyError,
  type PasswordPolicy,
} from '@/utils/passwordPolicy'

function usePolicyRule(policy: PasswordPolicy) {
  const { t } = useTranslation()

  return useMemo(() => {
    const translate = (key: string, options?: Record<string, unknown>) => String(t(key, options))
    const rule: Rule = {
      validator: async (_, value) => {
        const msg = passwordPolicyError(value, policy, translate)
        if (msg) throw new Error(msg)
      },
    }
    return { policy, rule, hint: describePasswordPolicy(policy, translate) }
  }, [policy, t])
}

/** 当前租户的密码策略：表单校验 rule + 规则说明（hint）。 */
export function usePasswordPolicy() {
  return usePolicyRule(useUserStore((s) => s.config.passwordPolicy))
}

let platformPolicyRequest: Promise<PasswordPolicy> | null = null

/** 平台管理员的密码策略（取自 /platform/info，按页面会话缓存）。 */
export function usePlatformPasswordPolicy() {
  const [policy, setPolicy] = useState<PasswordPolicy>(DEFAULT_PASSWORD_POLICY)

  useEffect(() => {
    let active = true
    platformPolicyRequest ??= platformInfo()
      .then((res) => normalizePasswordPolicy((res.data as { password_policy?: unknown } | undefined)?.password_policy))
      .catch(() => {
        platformPolicyRequest = null
        return DEFAULT_PASSWORD_POLICY
      })
    void platformPolicyRequest.then((p) => active && setPolicy(p))
    return () => {
      active = false
    }
  }, [])

  return usePolicyRule(policy)
}
