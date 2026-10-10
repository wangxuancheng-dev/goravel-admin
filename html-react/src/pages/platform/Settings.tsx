import { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Card, Form, Select, message } from 'antd'
import { useTranslation } from 'react-i18next'
import { getPlatformSettings, updatePlatformSettings } from '@/api/platform'
import { getPlatformAdmin } from '@/utils/platformRequest'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import PageContainer from '@/components/PageContainer'

interface SettingsForm {
  captcha_type: 'image' | 'slide'
}

/** Landlord console settings (stored in platform_settings). */
export default function PlatformSettingsPage() {
  const { t } = useTranslation()
  const showError = useUnhandledError()
  const [form] = Form.useForm<SettingsForm>()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const isViewer = getPlatformAdmin()?.role === 'viewer'

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getPlatformSettings()
      const settings = (res as { data?: { settings?: { captcha_type?: string } } })?.data?.settings
      form.setFieldsValue({ captcha_type: settings?.captcha_type === 'slide' ? 'slide' : 'image' })
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }, [form, showError, t])

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

  return (
    <PageContainer title={t('menu.platform_settings')}>
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
    </PageContainer>
  )
}
