import { Alert, Button, Card, Col, Row, Space, Spin, Tag, Typography } from 'antd'
import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { getPlatformOpsOverview } from '@/api/platform'

export default function PlatformOverview() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [overview, setOverview] = useState<Record<string, any> | null>(null)

  const summary = overview?.summary || null
  const queue = overview?.queue || null
  const alerts = overview?.alerts || null

  const schemaCards = useMemo(
    () => [
      { key: 'aligned', label: t('tenant.schema_aligned'), value: summary?.schema_aligned || 0, color: '#16a34a' },
      { key: 'behind', label: t('tenant.schema_behind'), value: summary?.schema_behind || 0, color: '#d97706' },
      { key: 'failed', label: t('tenant.schema_failed'), value: summary?.schema_failed || 0, color: '#dc2626' },
      { key: 'running', label: t('tenant.schema_running'), value: summary?.schema_running || 0, color: '#d97706' },
      { key: 'unknown', label: t('tenant.schema_unknown'), value: summary?.schema_unknown || 0, color: '#64748b' },
    ],
    [summary, t],
  )

  const load = async () => {
    setLoading(true)
    try {
      const res = await getPlatformOpsOverview()
      setOverview((res as any)?.data || null)
    } catch {
      setOverview(null)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const goTenants = (schemaStatus?: string) => {
    navigate(schemaStatus ? `/platform/tenants?schema_status=${schemaStatus}` : '/platform/tenants')
  }

  const goMaintenance = () => {
    navigate('/platform/tenants?maintenance=1')
  }

  return (
    <Spin spinning={loading}>
      <Space direction="vertical" size={16} style={{ width: '100%', maxWidth: 1100 }}>
        <Alert
          type="warning"
          showIcon
          message={t('platform.deploy_tip_title')}
          description={t('platform.deploy_tip_desc')}
        />

        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} md={6}>
            <Card size="small">
              <Typography.Text type="secondary">{t('platform.app_version')}</Typography.Text>
              <div style={{ fontSize: 22, fontWeight: 600 }}>{overview?.app_version || '—'}</div>
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card size="small">
              <Typography.Text type="secondary">{t('tenant.schema_expected')}</Typography.Text>
              <div style={{ fontSize: 22, fontWeight: 600 }}>
                {overview?.expected_migration_count ?? summary?.expected_migration_count ?? '—'}
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card size="small">
              <Typography.Text type="secondary">{t('tenant.ops_queue_pending')}</Typography.Text>
              <div style={{ fontSize: 22, fontWeight: 600 }}>{queue?.pending ?? '—'}</div>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {queue?.connection || '—'}
              </Typography.Text>
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card size="small" hoverable onClick={goMaintenance}>
              <Typography.Text type="secondary">{t('tenant.ops_maintenance')}</Typography.Text>
              <div
                style={{
                  fontSize: 22,
                  fontWeight: 600,
                  color: (summary?.maintenance || 0) > 0 ? '#d97706' : undefined,
                }}
              >
                {summary?.maintenance ?? 0}
              </div>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t('tenant.ops_total')} {summary?.total ?? 0}
              </Typography.Text>
            </Card>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          {schemaCards.map((item) => (
            <Col xs={12} sm={8} md={4} key={item.key}>
              <Card size="small" hoverable onClick={() => goTenants(item.key)}>
                <Typography.Text type="secondary">{item.label}</Typography.Text>
                <div style={{ fontSize: 22, fontWeight: 600, color: item.color }}>{item.value}</div>
              </Card>
            </Col>
          ))}
        </Row>

        <Card size="small" title={t('platform.alerts_title')}>
          <Space wrap>
            <Tag color={alerts?.tenant_ops?.configured ? 'success' : 'default'}>
              {t('tenant.alert_tenant_ops')}:{' '}
              {alerts?.tenant_ops?.configured ? t('tenant.alert_configured') : t('tenant.alert_not_configured')}
              {alerts?.tenant_ops?.source ? ` (${alerts.tenant_ops.source})` : ''}
            </Tag>
            <Tag color={alerts?.queue?.configured ? 'success' : 'default'}>
              {t('tenant.alert_queue')}:{' '}
              {alerts?.queue?.configured ? t('tenant.alert_configured') : t('tenant.alert_not_configured')}
              {alerts?.queue?.source ? ` (${alerts.queue.source})` : ''}
            </Tag>
          </Space>
        </Card>

        <Card
          size="small"
          title={t('platform.overview_actions')}
          extra={
            <Button type="link" onClick={() => void load()}>
              {t('common.refresh')}
            </Button>
          }
        >
          <Space wrap>
            <Button type="primary" onClick={() => goTenants('behind')}>
              {t('tenant.filter_schema_behind')}
            </Button>
            <Button danger onClick={() => goTenants('failed')}>
              {t('tenant.filter_schema_failed')}
            </Button>
            <Button onClick={() => goTenants('unknown')}>{t('tenant.filter_schema_unknown')}</Button>
            <Button onClick={goMaintenance}>{t('platform.filter_maintenance')}</Button>
            <Button onClick={() => navigate('/platform/tenants')}>{t('menu.tenant')}</Button>
            <Button onClick={() => navigate('/platform/tenant-op-logs')}>{t('menu.tenant_op_log')}</Button>
          </Space>
        </Card>
      </Space>
    </Spin>
  )
}
