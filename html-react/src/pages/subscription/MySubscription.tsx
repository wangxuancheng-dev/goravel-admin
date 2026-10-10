import { useCallback, useEffect, useMemo, useState } from 'react'
import { Alert, Card, Descriptions, Empty, Progress, Space, Spin, Table, Tag, Typography } from 'antd'
import { useTranslation } from 'react-i18next'
import { getMyPlan, type MyPlanFeature, type MyPlanLimit, type MyPlanPayload } from '@/api/subscription'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import PageContainer from '@/components/PageContainer'

function formatDate(value?: string | null): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString()
}

function daysLeft(value?: string | null): number | null {
  if (!value) return null
  const end = new Date(value).getTime()
  if (Number.isNaN(end)) return null
  return Math.ceil((end - Date.now()) / 86400000)
}

export default function MySubscriptionPage() {
  const { t } = useTranslation()
  const showError = useUnhandledError()
  const [loading, setLoading] = useState(true)
  const [data, setData] = useState<MyPlanPayload | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getMyPlan()
      setData((res?.data as MyPlanPayload) || null)
    } catch (e) {
      showError(e, t('common.operation_failed'))
    } finally {
      setLoading(false)
    }
  }, [showError, t])

  useEffect(() => {
    void load()
  }, [load])

  const features = useMemo(() => data?.features || [], [data])
  const limits = useMemo(() => data?.limits || [], [data])
  const sub = data?.subscription
  const remainingDays = daysLeft(sub?.ends_at)
  const expiringSoon = remainingDays !== null && remainingDays <= 7

  const kindLabel = (kind: MyPlanFeature['kind']) => {
    if (kind === 'capability') return t('subscription.kind_capability')
    if (kind === 'module') return t('subscription.kind_module')
    return t('subscription.kind_other')
  }

  if (loading && !data) {
    return (
      <PageContainer title={t('menu.subscription')}>
        <Spin />
      </PageContainer>
    )
  }

  if (!data || data.enabled === false) {
    return (
      <PageContainer title={t('menu.subscription')}>
        <Empty description={t('subscription.not_available')} />
      </PageContainer>
    )
  }

  return (
    <PageContainer title={t('menu.subscription')}>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {expiringSoon ? (
          <Alert
            type="warning"
            showIcon
            message={
              (remainingDays as number) < 0
                ? t('subscription.expired_hint')
                : t('subscription.expiring_hint', { days: remainingDays })
            }
          />
        ) : null}

        <Card title={t('subscription.current_plan')} loading={loading}>
          <Descriptions column={{ xs: 1, sm: 2 }} size="small">
            <Descriptions.Item label={t('subscription.plan_name')}>
              {data.plan?.name || '—'}
              {data.plan?.code ? (
                <Typography.Text code style={{ marginLeft: 8 }}>
                  {data.plan.code}
                </Typography.Text>
              ) : null}
            </Descriptions.Item>
            <Descriptions.Item label={t('subscription.status')}>
              {sub?.status ? <Tag color={sub.status === 'active' ? 'green' : 'default'}>{sub.status}</Tag> : '—'}
            </Descriptions.Item>
            <Descriptions.Item label={t('subscription.starts_at')}>{formatDate(sub?.starts_at)}</Descriptions.Item>
            <Descriptions.Item label={t('subscription.ends_at')}>
              {sub?.ends_at ? formatDate(sub.ends_at) : t('subscription.no_expiry')}
            </Descriptions.Item>
            {sub?.trial_ends_at ? (
              <Descriptions.Item label={t('subscription.trial_ends_at')}>{formatDate(sub.trial_ends_at)}</Descriptions.Item>
            ) : null}
            {data.plan?.description ? (
              <Descriptions.Item label={t('subscription.description')} span={2}>
                {data.plan.description}
              </Descriptions.Item>
            ) : null}
          </Descriptions>
          <Typography.Paragraph type="secondary" style={{ marginTop: 12, marginBottom: 0 }}>
            {t('subscription.upgrade_hint')}
          </Typography.Paragraph>
        </Card>

        {limits.length > 0 ? (
          <Card title={t('subscription.quota_title')} loading={loading}>
            <Space direction="vertical" size="middle" style={{ width: '100%' }}>
              {limits.map((l: MyPlanLimit) => {
                // limit 0 (not unlimited) means no quota at all: show as full
                const percent = l.unlimited ? 0 : l.limit <= 0 ? 100 : Math.min(100, Math.round((l.used / l.limit) * 100))
                return (
                  <div key={l.key}>
                    <Space style={{ marginBottom: 4 }}>
                      <span>{l.name}</span>
                      <Typography.Text type="secondary">
                        {l.unlimited
                          ? t('subscription.quota_unlimited', { used: l.used })
                          : t('subscription.quota_usage', { used: l.used, limit: l.limit })}
                      </Typography.Text>
                    </Space>
                    {l.unlimited ? null : (
                      <Progress percent={percent} status={percent >= 100 ? 'exception' : percent >= 80 ? 'active' : 'normal'} />
                    )}
                  </div>
                )
              })}
            </Space>
          </Card>
        ) : null}

        <Card title={t('subscription.features_title')} loading={loading}>
          <Table<MyPlanFeature>
            size="small"
            rowKey="key"
            pagination={false}
            dataSource={features}
            locale={{ emptyText: t('subscription.features_empty') }}
            columns={[
              { title: t('subscription.feature'), dataIndex: 'name' },
              {
                title: t('subscription.kind'),
                dataIndex: 'kind',
                width: 120,
                render: (v: MyPlanFeature['kind']) => <Tag>{kindLabel(v)}</Tag>,
              },
              {
                title: t('subscription.availability'),
                dataIndex: 'enabled',
                width: 160,
                render: (v: boolean) =>
                  v ? (
                    <Tag color="green">{t('subscription.enabled')}</Tag>
                  ) : (
                    <Tag color="orange">{t('subscription.locked')}</Tag>
                  ),
              },
            ]}
          />
        </Card>
      </Space>
    </PageContainer>
  )
}
