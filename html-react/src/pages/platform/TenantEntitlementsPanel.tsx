import { App, Button, Select, Space, Switch, Table, Tag, Typography } from 'antd'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  assignPlatformTenantPlan,
  deletePlatformTenantEntitlement,
  getPlatformPlans,
  getPlatformTenantEntitlements,
  recomputePlatformTenantEntitlements,
  setPlatformTenantEntitlement,
} from '@/api/platform'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { getPlatformAdmin } from '@/utils/platformRequest'

type FeatureRow = {
  key: string
  name: string
  type: string
  always_on?: boolean
  effective?: boolean
  has_override?: boolean
  override_value?: string
}

type EntitlementsData = {
  plan?: { code?: string; name?: string }
  catalog?: FeatureRow[]
  effective?: { version?: number; plan_code?: string }
}

export default function TenantEntitlementsPanel({ tenantId }: { tenantId: number }) {
  const { t } = useTranslation()
  const showError = useUnhandledError()
  const { message, modal } = App.useApp()
  const isViewer = getPlatformAdmin()?.role === 'viewer'
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<EntitlementsData | null>(null)
  const [plans, setPlans] = useState<{ code: string; name: string }[]>([])
  const [planCode, setPlanCode] = useState<string>()
  const [savingKey, setSavingKey] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [entRes, planRes] = await Promise.all([
        getPlatformTenantEntitlements(tenantId),
        getPlatformPlans(),
      ])
      const ent = (entRes as { data?: EntitlementsData })?.data || null
      setData(ent)
      setPlanCode(ent?.plan?.code || ent?.effective?.plan_code || undefined)
      const list = ((planRes as { data?: { list?: { code: string; name: string }[] } })?.data?.list ||
        (planRes as { list?: { code: string; name: string }[] })?.list ||
        []) as { code: string; name: string }[]
      setPlans(list)
    } catch (e) {
      showError(e, t('common.operation_failed'))
    } finally {
      setLoading(false)
    }
  }, [tenantId, showError, t])

  useEffect(() => {
    void load()
  }, [load])

  const planLabel = (code?: string) => {
    if (!code) return t('entitlement.assign_none')
    const p = plans.find((x) => x.code === code)
    return p ? `${p.name} (${p.code})` : code
  }

  // Assigning cancels the active subscription and applies the new plan immediately: confirm first.
  const onAssignPlan = () => {
    if (!planCode) return
    const currentCode = data?.plan?.code || data?.effective?.plan_code || undefined
    const to = planLabel(planCode)
    modal.confirm({
      title: t('entitlement.assign_confirm_title'),
      content:
        currentCode && currentCode === planCode
          ? t('entitlement.assign_confirm_same', { to })
          : t('entitlement.assign_confirm_change', { from: planLabel(currentCode), to }),
      okType: 'danger',
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: () => doAssignPlan(),
    })
  }

  const doAssignPlan = async () => {
    if (!planCode) return
    try {
      setLoading(true)
      await assignPlatformTenantPlan(tenantId, { plan_code: planCode })
      message.success(t('entitlement.plan_assigned'))
      await load()
    } catch (e) {
      showError(e, t('common.operation_failed'))
    } finally {
      setLoading(false)
    }
  }

  const onToggle = async (row: FeatureRow, enabled: boolean) => {
    setSavingKey(row.key)
    try {
      await setPlatformTenantEntitlement(tenantId, row.key, {
        enabled,
        reason: 'platform_ui',
      })
      message.success(t('entitlement.override_saved'))
      await load()
    } catch (e) {
      showError(e, t('common.operation_failed'))
    } finally {
      setSavingKey('')
    }
  }

  const onClearOverride = async (row: FeatureRow) => {
    setSavingKey(row.key)
    try {
      await deletePlatformTenantEntitlement(tenantId, row.key)
      message.success(t('entitlement.override_cleared'))
      await load()
    } catch (e) {
      showError(e, t('common.operation_failed'))
    } finally {
      setSavingKey('')
    }
  }

  const onRecompute = async () => {
    try {
      setLoading(true)
      await recomputePlatformTenantEntitlements(tenantId)
      message.success(t('entitlement.recomputed'))
      await load()
    } catch (e) {
      showError(e, t('common.operation_failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Typography.Text type="secondary">{t('entitlement.tenant_hint')}</Typography.Text>
      <Space wrap>
        <Select
          style={{ minWidth: 180 }}
          placeholder={t('entitlement.select_plan')}
          value={planCode}
          disabled={isViewer}
          options={plans.map((p) => ({ value: p.code, label: `${p.name} (${p.code})` }))}
          onChange={setPlanCode}
        />
        <Button type="primary" disabled={isViewer || !planCode} loading={loading} onClick={onAssignPlan}>
          {t('entitlement.assign_plan')}
        </Button>
        <Button disabled={isViewer} loading={loading} onClick={() => void onRecompute()}>
          {t('entitlement.recompute')}
        </Button>
        <Typography.Text type="secondary">
          {t('entitlement.version')}: {data?.effective?.version ?? '—'}
        </Typography.Text>
      </Space>
      <Table
        size="small"
        rowKey="key"
        loading={loading}
        pagination={false}
        dataSource={data?.catalog || []}
        locale={{ emptyText: t('entitlement.empty_catalog') }}
        columns={[
          {
            title: t('entitlement.feature'),
            dataIndex: 'name',
            render: (_, row) => (
              <Space>
                <span>{row.name}</span>
                <Typography.Text code>{row.key}</Typography.Text>
                {row.always_on ? <Tag color="green">{t('entitlement.always_on')}</Tag> : null}
              </Space>
            ),
          },
          {
            title: t('entitlement.enabled'),
            dataIndex: 'effective',
            width: 100,
            render: (v: boolean, row) => (
              <Switch
                checked={!!v}
                disabled={isViewer || row.type !== 'boolean' || !!row.always_on}
                loading={savingKey === row.key}
                onChange={(checked) => void onToggle(row, checked)}
              />
            ),
          },
          {
            title: t('entitlement.override'),
            width: 120,
            render: (_, row) =>
              row.has_override ? (
                <Button
                  type="link"
                  size="small"
                  disabled={isViewer}
                  loading={savingKey === row.key}
                  onClick={() => void onClearOverride(row)}
                >
                  {t('entitlement.clear_override')}
                </Button>
              ) : (
                '—'
              ),
          },
        ]}
      />
    </Space>
  )
}
