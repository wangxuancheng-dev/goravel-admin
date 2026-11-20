import { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Card, Form, InputNumber, Select, Space, message } from 'antd'
import { useTranslation } from 'react-i18next'
import { getPlatformSettings, updatePlatformSettings, type SubscriptionExpiryPolicy } from '@/api/platform'
import { getPlatformAdmin } from '@/utils/platformRequest'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import PageContainer from '@/components/PageContainer'

interface SettingsForm {
  captcha_type: 'image' | 'slide'
}

const REMIND_DAY_OPTIONS = [30, 14, 7, 3, 1]

const defaultExpiry: SubscriptionExpiryPolicy = {
  action: 'downgrade',
  grace_days: 0,
  remind_days: [7, 3, 1],
  notice_locale: 'cn',
}

/** Landlord console settings (stored in platform_settings). */
export default function PlatformSettingsPage() {
  const { t } = useTranslation()
  const showError = useUnhandledError()
  const [form] = Form.useForm<SettingsForm>()
  const [expiryForm] = Form.useForm<SubscriptionExpiryPolicy>()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [savingExpiry, setSavingExpiry] = useState(false)
  const isViewer = getPlatformAdmin()?.role === 'viewer'
  const expiryAction = Form.useWatch('action', expiryForm)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getPlatformSettings()
      const settings = (
        res as {
          data?: { settings?: { captcha_type?: string; subscription_expiry?: Partial<SubscriptionExpiryPolicy> } }
        }
      )?.data?.settings
      form.setFieldsValue({ captcha_type: settings?.captcha_type === 'slide' ? 'slide' : 'image' })
      expiryForm.setFieldsValue({ ...defaultExpiry, ...(settings?.subscription_expiry || {}) })
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }, [form, expiryForm, showError, t])

  useEffect(() => {
    void load()
  }, [load])

  const onSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      await updatePlatformSettings({ captcha_type: values.captcha_type })
      message.success(t('config.update_success'))
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSaving(false)
    }
  }

  const onSaveExpiry = async () => {
    try {
      const values = await expiryForm.validateFields()
      setSavingExpiry(true)
      await updatePlatformSettings({
        subscription_expiry: {
          action: values.action,
          grace_days: Number(values.grace_days) || 0,
          remind_days: (values.remind_days || []).map(Number),
          notice_locale: values.notice_locale,
        },
      })
      message.success(t('config.update_success'))
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSavingExpiry(false)
    }
  }

  return (
    <PageContainer title={t('menu.platform_settings')}>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Card title={t('platform.settings_captcha_title')} loading={loading}>
          {isViewer ? (
            <Alert type="info" showIcon style={{ marginBottom: 16 }} message={t('platform.settings_viewer_hint')} />
          ) : null}
          <Form form={form} layout="vertical" disabled={isViewer} style={{ maxWidth: 480 }}>
            <Form.Item
              name="captcha_type"
              label={t('config.captcha_type')}
              extra={t('platform.settings_captcha_tip')}
            >
              <Select
                options={[
                  { value: 'image', label: t('config.captcha_type_image') },
                  { value: 'slide', label: t('config.captcha_type_slide') },
                ]}
              />
            </Form.Item>
            {!isViewer ? (
              <Button type="primary" loading={saving} onClick={() => void onSave()}>
                {t('common.save')}
              </Button>
            ) : null}
          </Form>
        </Card>

        <Card title={t('platform.settings_expiry_title')} loading={loading}>
          <Alert type="info" showIcon style={{ marginBottom: 16 }} message={t('platform.settings_expiry_desc')} />
          <Form
            form={expiryForm}
            layout="vertical"
            disabled={isViewer}
            style={{ maxWidth: 480 }}
            initialValues={defaultExpiry}
          >
            <Form.Item name="action" label={t('platform.settings_expiry_action')}>
              <Select
                options={[
                  { value: 'downgrade', label: t('platform.settings_expiry_action_downgrade') },
                  { value: 'suspend', label: t('platform.settings_expiry_action_suspend') },
                  { value: 'notify_only', label: t('platform.settings_expiry_action_notify_only') },
                ]}
              />
            </Form.Item>
            <Form.Item
              name="grace_days"
              label={t('platform.settings_expiry_grace_days')}
              extra={t('platform.settings_expiry_grace_tip')}
            >
              <InputNumber min={0} max={90} precision={0} disabled={isViewer || expiryAction === 'notify_only'} />
            </Form.Item>
            <Form.Item
              name="remind_days"
              label={t('platform.settings_expiry_remind_days')}
              extra={t('platform.settings_expiry_remind_tip')}
              rules={[
                {
                  validator: (_, v: number[] | undefined) =>
                    (v || []).length <= 5 ? Promise.resolve() : Promise.reject(new Error(t('platform.settings_expiry_remind_max'))),
                },
              ]}
            >
              <Select
                mode="multiple"
                options={REMIND_DAY_OPTIONS.map((d) => ({ value: d, label: t('platform.settings_expiry_days_before', { n: d }) }))}
              />
            </Form.Item>
            <Form.Item
              name="notice_locale"
              label={t('platform.settings_expiry_locale')}
              extra={t('platform.settings_expiry_locale_tip')}
            >
              <Select
                options={[
                  { value: 'cn', label: '简体中文' },
                  { value: 'en', label: 'English' },
                ]}
              />
            </Form.Item>
            {!isViewer ? (
              <Button type="primary" loading={savingExpiry} onClick={() => void onSaveExpiry()}>
                {t('common.save')}
              </Button>
            ) : null}
          </Form>
        </Card>
      </Space>
    </PageContainer>
  )
}
