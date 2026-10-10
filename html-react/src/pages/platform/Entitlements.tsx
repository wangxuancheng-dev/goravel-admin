import { Button, Card, Form, Input, InputNumber, Modal, Space, Switch, Table, Tag, message } from 'antd'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  getPlatformFeatures,
  getPlatformPlanEntitlements,
  getPlatformPlans,
  registerPlatformModuleFeature,
  updatePlatformPlan,
  upsertPlatformPlan,
} from '@/api/platform'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { getPlatformAdmin } from '@/utils/platformRequest'

type Feature = {
  id?: number
  key: string
  name: string
  description?: string
  type: string
  default_value?: string
  menu_slugs?: string[]
  always_on?: boolean
  status?: number
}

type Plan = {
  id?: number
  code: string
  name: string
  description?: string
  is_default?: boolean
  status?: number
}

export default function PlatformEntitlementsPage() {
  const { t } = useTranslation()
  const showError = useUnhandledError()
  const isViewer = getPlatformAdmin()?.role === 'viewer'
  const [features, setFeatures] = useState<Feature[]>([])
  const [plans, setPlans] = useState<Plan[]>([])
  const [loading, setLoading] = useState(false)
  const [moduleOpen, setModuleOpen] = useState(false)
  const [planOpen, setPlanOpen] = useState(false)
  const [planEntOpen, setPlanEntOpen] = useState(false)
  const [editingPlan, setEditingPlan] = useState<Plan | null>(null)
  const [moduleForm] = Form.useForm()
  const [planForm] = Form.useForm()
  const [planEntForm] = Form.useForm()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [fRes, pRes] = await Promise.all([getPlatformFeatures(), getPlatformPlans()])
      setFeatures(
        ((fRes as { data?: { list?: Feature[] } })?.data?.list ||
          (fRes as { list?: Feature[] })?.list ||
          []) as Feature[],
      )
      setPlans(
        ((pRes as { data?: { list?: Plan[] } })?.data?.list ||
          (pRes as { list?: Plan[] })?.list ||
          []) as Plan[],
      )
    } catch (e) {
      showError(e, t('common.operation_failed'))
    } finally {
      setLoading(false)
    }
  }, [showError, t])

  useEffect(() => {
    void load()
  }, [load])

  const submitModule = async () => {
    try {
      const values = await moduleForm.validateFields()
      await registerPlatformModuleFeature({
        module_name: values.module_name,
        display_name: values.display_name,
        menu_slug: values.menu_slug,
        always_on: !!values.always_on,
        row_quota: !!values.row_quota,
      })
      message.success(t('entitlement.module_registered'))
      setModuleOpen(false)
      moduleForm.resetFields()
      await load()
    } catch (e) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showError(e, t('common.operation_failed'))
    }
  }

  const submitPlan = async () => {
    try {
      const values = await planForm.validateFields()
      await upsertPlatformPlan({
        ...values,
        is_public: true,
        status: 1,
      })
      message.success(t('common.save_success'))
      setPlanOpen(false)
      planForm.resetFields()
      await load()
    } catch (e) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showError(e, t('common.operation_failed'))
    }
  }

  const openPlanEntitlements = async (plan: Plan) => {
    setEditingPlan(plan)
    setPlanEntOpen(true)
    try {
      const res = await getPlatformPlanEntitlements(plan.code)
      const list = ((res as { data?: { list?: { feature_key?: string; value?: string }[] } })?.data?.list ||
        (res as { list?: { feature_key?: string; value?: string }[] })?.list ||
        []) as { feature_key?: string; value?: string }[]
      const lines =
        list.length > 0
          ? list
              .map((row) => {
                const key = String(row.feature_key || '').trim()
                if (!key) return ''
                return `${key}=${String(row.value ?? '')}`
              })
              .filter(Boolean)
              .join('\n')
          : features
              .map((f) => {
                if (f.type === 'boolean') return `${f.key}=false`
                if (f.type === 'limit') return `${f.key}=0`
                return ''
              })
              .filter(Boolean)
              .join('\n')
      planEntForm.setFieldsValue({ entitlements_text: lines })
    } catch (e) {
      showError(e, t('common.operation_failed'))
      planEntForm.setFieldsValue({ entitlements_text: '' })
    }
  }

  const submitPlanEntitlements = async () => {
    if (!editingPlan) return
    try {
      const values = await planEntForm.validateFields()
      const entitlements: Record<string, string> = {}
      String(values.entitlements_text || '')
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean)
        .forEach((line) => {
          const idx = line.indexOf('=')
          if (idx <= 0) return
          const key = line.slice(0, idx).trim()
          const value = line.slice(idx + 1).trim()
          if (key) entitlements[key] = value
        })
      await updatePlatformPlan(editingPlan.code, {
        code: editingPlan.code,
        name: editingPlan.name,
        description: editingPlan.description,
        is_default: editingPlan.is_default,
        status: editingPlan.status || 1,
        entitlements,
      })
      message.success(t('entitlement.plan_entitlements_saved'))
      setPlanEntOpen(false)
      await load()
    } catch (e) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showError(e, t('common.operation_failed'))
    }
  }

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Card
        title={t('entitlement.catalog_title')}
        extra={
          <Button type="primary" disabled={isViewer} onClick={() => setModuleOpen(true)}>
            {t('entitlement.register_module')}
          </Button>
        }
      >
        <p style={{ color: 'rgba(0,0,0,0.45)', marginTop: 0 }}>{t('entitlement.catalog_hint')}</p>
        <p style={{ color: 'rgba(0,0,0,0.45)' }}>{t('entitlement.handwritten_hint')}</p>
        <Table
          rowKey="key"
          loading={loading}
          dataSource={features}
          pagination={false}
          columns={[
            { title: t('entitlement.feature'), dataIndex: 'name' },
            { title: 'Key', dataIndex: 'key', render: (v) => <code>{v}</code> },
            { title: t('entitlement.type'), dataIndex: 'type', width: 100 },
            {
              title: t('entitlement.always_on'),
              dataIndex: 'always_on',
              width: 100,
              render: (v) => (v ? <Tag color="green">{t('common.yes')}</Tag> : '—'),
            },
          ]}
        />
      </Card>

      <Card
        title={t('entitlement.plans_title')}
        extra={
          <Button type="primary" disabled={isViewer} onClick={() => setPlanOpen(true)}>
            {t('entitlement.add_plan')}
          </Button>
        }
      >
        <Table
          rowKey="code"
          loading={loading}
          dataSource={plans}
          pagination={false}
          columns={[
            { title: t('entitlement.plan'), dataIndex: 'name' },
            { title: 'Code', dataIndex: 'code', render: (v) => <code>{v}</code> },
            {
              title: t('entitlement.default'),
              dataIndex: 'is_default',
              width: 100,
              render: (v) => (v ? <Tag color="green">{t('common.yes')}</Tag> : '—'),
            },
            {
              title: t('common.actions'),
              width: 160,
              render: (_, row) => (
                <Button type="link" size="small" disabled={isViewer} onClick={() => void openPlanEntitlements(row)}>
                  {t('entitlement.edit_plan_entitlements')}
                </Button>
              ),
            },
          ]}
        />
      </Card>

      <Modal
        open={moduleOpen}
        title={t('entitlement.register_module')}
        onCancel={() => setModuleOpen(false)}
        onOk={() => void submitModule()}
        destroyOnHidden
      >
        <p style={{ color: 'rgba(0,0,0,0.45)' }}>{t('entitlement.handwritten_hint')}</p>
        <Form form={moduleForm} layout="vertical">
          <Form.Item name="module_name" label={t('entitlement.module_name')} rules={[{ required: true }]}>
            <Input placeholder="guestbook" />
          </Form.Item>
          <Form.Item name="display_name" label={t('entitlement.feature')}>
            <Input />
          </Form.Item>
          <Form.Item name="menu_slug" label={t('entitlement.menu_slugs')}>
            <Input placeholder="guestbook" />
          </Form.Item>
          <Form.Item name="always_on" label={t('entitlement.always_on')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="row_quota" label={t('entitlement.row_quota')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={planOpen}
        title={t('entitlement.add_plan')}
        onCancel={() => setPlanOpen(false)}
        onOk={() => void submitPlan()}
        destroyOnHidden
      >
        <Form form={planForm} layout="vertical">
          <Form.Item name="code" label="Code" rules={[{ required: true }]}>
            <Input placeholder="pro" />
          </Form.Item>
          <Form.Item name="name" label={t('entitlement.plan')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label={t('entitlement.description')}>
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="is_default" label={t('entitlement.default')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="price_monthly" label={t('entitlement.price_monthly')}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={planEntOpen}
        title={t('entitlement.edit_plan_entitlements')}
        onCancel={() => setPlanEntOpen(false)}
        onOk={() => void submitPlanEntitlements()}
        width={640}
        destroyOnHidden
      >
        <p style={{ color: 'rgba(0,0,0,0.45)' }}>{t('entitlement.plan_entitlements_hint')}</p>
        <Form form={planEntForm} layout="vertical">
          <Form.Item name="entitlements_text" label={t('entitlement.plan_entitlements')}>
            <Input.TextArea rows={12} placeholder={'module.guestbook=true\nquota.module.guestbook.rows=1000'} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  )
}
