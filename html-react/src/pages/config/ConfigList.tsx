import { useCallback, useEffect, useState } from 'react'
import { App, Card, Form, Input, InputNumber, Select, Switch, Tabs } from 'antd'
import { useTranslation } from 'react-i18next'
import PageContainer from '@/components/PageContainer'
import PermissionButton from '@/components/PermissionButton'
import AttachmentImageField from '@/components/AttachmentImageField'
import { getConfigByGroup, saveConfig, testEmail } from '@/api/config'
import { entityField } from '@/utils/normalize'
import { notifyWebsiteConfigUpdated } from '@/utils/publicImage'
import { useUnhandledError } from '@/hooks/useUnhandledError'

function configsToForm(
  configs: Array<Record<string, unknown>> | undefined,
  keys: string[],
  parsers?: Record<string, (value: string) => unknown>,
) {
  const result: Record<string, unknown> = {}
  keys.forEach((key) => {
    result[key] = parsers?.[key]?.('') ?? ''
  })

  configs?.forEach((item) => {
    const key = String(entityField(item, 'key', '') ?? entityField(item, 'Key', '') ?? '')
    const raw = String(entityField(item, 'value', '') ?? entityField(item, 'Value', '') ?? '')
    if (keys.includes(key)) {
      result[key] = parsers?.[key] ? parsers[key](raw) : raw
    }
  })

  return result
}

function WebsiteConfigPanel() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getConfigByGroup('website')
      const values = configsToForm(res.data?.configs, [
        'site_enabled',
        'site_name',
        'site_url',
        'site_logo',
        'site_icp',
        'site_keywords',
        'site_description',
        'site_copyright',
      ])
      if (values.site_enabled === '') values.site_enabled = '1'
      form.setFieldsValue(values)
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }, [form, showError, t])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      await saveConfig('website', values as Record<string, unknown>)
      notifyWebsiteConfigUpdated()
      message.success(t('config.update_success'))
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Form form={form} layout="vertical" disabled={loading}>
      <Form.Item
        name="site_enabled"
        label={t('config.site_enabled')}
        valuePropName="checked"
        getValueProps={(value) => ({ checked: value === '1' || value === true })}
        getValueFromEvent={(checked: boolean) => (checked ? '1' : '0')}
      >
        <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
      </Form.Item>
      <Form.Item name="site_name" label={t('config.site_name')}>
        <Input placeholder={t('config.site_name_placeholder')} />
      </Form.Item>
      <Form.Item name="site_url" label={t('config.site_url')}>
        <Input placeholder={t('config.site_url_placeholder')} />
      </Form.Item>
      <Form.Item name="site_logo" label={t('config.site_logo')}>
        <AttachmentImageField placeholder={t('config.site_logo_placeholder')} />
      </Form.Item>
      <Form.Item name="site_icp" label={t('config.site_icp')}>
        <Input placeholder={t('config.site_icp_placeholder')} />
      </Form.Item>
      <Form.Item name="site_keywords" label={t('config.site_keywords')}>
        <Input placeholder={t('config.site_keywords_placeholder')} />
      </Form.Item>
      <Form.Item name="site_description" label={t('config.site_description')}>
        <Input placeholder={t('config.site_description_placeholder')} />
      </Form.Item>
      <Form.Item name="site_copyright" label={t('config.site_copyright')}>
        <Input.TextArea rows={3} placeholder={t('config.site_copyright_placeholder')} />
      </Form.Item>
      <PermissionButton permission="config.save" type="primary" loading={submitting} onClick={() => void handleSubmit()}>
        {t('common.save')}
      </PermissionButton>
    </Form>
  )
}

function EmailConfigPanel() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [testing, setTesting] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getConfigByGroup('email')
      const values = configsToForm(
        res.data?.configs,
        [
          'email_host',
          'email_port',
          'email_username',
          'email_password',
          'email_from',
          'email_from_name',
          'email_encryption',
          'email_timeout',
        ],
        {
          email_port: (v) => (v ? Number(v) : 587),
          email_timeout: (v) => (v ? Number(v) : 30),
        },
      )
      if (!values.email_encryption) values.email_encryption = 'tls'
      form.setFieldsValue(values)
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }, [form, showError, t])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      await saveConfig('email', values as Record<string, unknown>)
      message.success(t('config.update_success'))
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleTest = async () => {
    try {
      const values = await form.validateFields()
      setTesting(true)
      await testEmail(values as Record<string, unknown>)
      message.success(t('config.test_email_success'))
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('config.test_email_failed'))
    } finally {
      setTesting(false)
    }
  }

  return (
    <Form form={form} layout="vertical" disabled={loading}>
      <Form.Item name="email_host" label={t('config.email_host')}>
        <Input placeholder={t('config.email_host_placeholder')} />
      </Form.Item>
      <Form.Item name="email_port" label={t('config.email_port')}>
        <InputNumber min={1} max={65535} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="email_username" label={t('config.email_username')}>
        <Input autoComplete="off" placeholder={t('config.email_username_placeholder')} />
      </Form.Item>
      <Form.Item name="email_password" label={t('config.email_password')}>
        <Input.Password autoComplete="new-password" placeholder={t('config.email_password_placeholder')} />
      </Form.Item>
      <Form.Item name="email_from" label={t('config.email_from')}>
        <Input placeholder={t('config.email_from_placeholder')} />
      </Form.Item>
      <Form.Item name="email_from_name" label={t('config.email_from_name')}>
        <Input placeholder={t('config.email_from_name_placeholder')} />
      </Form.Item>
      <Form.Item name="email_encryption" label={t('config.email_encryption')}>
        <Select
          options={[
            { label: 'TLS', value: 'tls' },
            { label: 'SSL', value: 'ssl' },
            { label: 'None', value: '' },
          ]}
        />
      </Form.Item>
      <Form.Item name="email_timeout" label={t('config.email_timeout')}>
        <InputNumber min={1} max={300} style={{ width: '100%' }} addonAfter={t('config.email_timeout_unit')} />
      </Form.Item>
      <Form.Item>
        <PermissionButton permission="config.save" type="primary" loading={submitting} onClick={() => void handleSubmit()}>
          {t('common.save')}
        </PermissionButton>
        <PermissionButton
          permission="config.test_email"
          style={{ marginLeft: 8 }}
          loading={testing}
          onClick={() => void handleTest()}
        >
          {t('config.test_email')}
        </PermissionButton>
      </Form.Item>
    </Form>
  )
}

function CaptchaConfigPanel() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getConfigByGroup('captcha')
      const values = configsToForm(res.data?.configs, ['captcha_enabled', 'captcha_expire'], {
        captcha_enabled: (v) => v === '1' || v === 'true',
        captcha_expire: (v) => (v ? Number(v) : 120),
      })
      form.setFieldsValue(values)
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }, [form, showError, t])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      await saveConfig('captcha', {
        captcha_enabled: values.captcha_enabled ? '1' : '0',
        captcha_expire: String(values.captcha_expire ?? 120),
      })
      message.success(t('config.update_success'))
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Form form={form} layout="vertical" disabled={loading}>
      <Form.Item name="captcha_enabled" label={t('config.captcha_enabled')} valuePropName="checked">
        <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
      </Form.Item>
      <Form.Item
        name="captcha_expire"
        label={t('config.captcha_expire')}
        rules={[
          { required: true, message: t('config.captcha_expire_required') },
          {
            type: 'number',
            min: 30,
            message: t('config.captcha_expire_min', { seconds: 30 }),
          },
        ]}
      >
        <InputNumber min={30} max={600} style={{ width: '100%' }} addonAfter={t('config.captcha_expire_unit')} />
      </Form.Item>
      <PermissionButton permission="config.save" type="primary" loading={submitting} onClick={() => void handleSubmit()}>
        {t('common.save')}
      </PermissionButton>
    </Form>
  )
}

function StorageConfigPanel() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getConfigByGroup('storage')
      const values = configsToForm(res.data?.configs, ['file_disk', 'export_format'])
      if (!values.file_disk) values.file_disk = 'local'
      if (!values.export_format) values.export_format = 'csv'
      form.setFieldsValue(values)
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }, [form, showError, t])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      await saveConfig('storage', values as Record<string, unknown>)
      message.success(t('config.update_success'))
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Form form={form} layout="vertical" disabled={loading}>
      <Form.Item name="file_disk" label={t('config.file_disk')}>
        <Select
          options={[
            { label: 'local', value: 'local' },
            { label: 's3', value: 's3' },
            { label: 'oss', value: 'oss' },
            { label: 'cos', value: 'cos' },
            { label: 'qiniu', value: 'qiniu' },
            { label: 'minio', value: 'minio' },
          ]}
        />
      </Form.Item>
      <Form.Item name="export_format" label={t('config.export_format')}>
        <Select
          options={[
            { label: 'csv', value: 'csv' },
            { label: 'xlsx', value: 'xlsx' },
          ]}
        />
      </Form.Item>
      <PermissionButton permission="config.save" type="primary" loading={submitting} onClick={() => void handleSubmit()}>
        {t('common.save')}
      </PermissionButton>
    </Form>
  )
}

function CustomerServiceConfigPanel() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getConfigByGroup('customer_service')
      const values = configsToForm(res.data?.configs, [
        'cs_enabled',
        'cs_work_time',
        'cs_phone',
        'cs_email',
        'cs_wechat',
        'cs_wechat_qr',
        'cs_qq',
        'cs_telegram',
        'cs_whatsapp',
        'cs_online_url',
        'cs_custom_link',
        'cs_remark',
      ])
      if (values.cs_enabled === '') values.cs_enabled = '1'
      form.setFieldsValue(values)
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }, [form, showError, t])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      await saveConfig('customer_service', values as Record<string, unknown>)
      message.success(t('config.update_success'))
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Form form={form} layout="vertical" disabled={loading}>
      <Form.Item
        name="cs_enabled"
        label={t('config.cs_enabled')}
        valuePropName="checked"
        getValueProps={(value) => ({ checked: value === '1' || value === true })}
        getValueFromEvent={(checked: boolean) => (checked ? '1' : '0')}
      >
        <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
      </Form.Item>
      <Form.Item name="cs_work_time" label={t('config.cs_work_time')}>
        <Input placeholder={t('config.cs_work_time_placeholder')} />
      </Form.Item>
      <Form.Item name="cs_phone" label={t('config.cs_phone')}>
        <Input placeholder={t('config.cs_phone_placeholder')} />
      </Form.Item>
      <Form.Item name="cs_email" label={t('config.cs_email')}>
        <Input placeholder={t('config.cs_email_placeholder')} />
      </Form.Item>
      <Form.Item name="cs_wechat" label={t('config.cs_wechat')}>
        <Input placeholder={t('config.cs_wechat_placeholder')} />
      </Form.Item>
      <Form.Item name="cs_wechat_qr" label={t('config.cs_wechat_qr')}>
        <AttachmentImageField placeholder={t('config.cs_wechat_qr_placeholder')} />
      </Form.Item>
      <Form.Item name="cs_qq" label={t('config.cs_qq')}>
        <Input placeholder={t('config.cs_qq_placeholder')} />
      </Form.Item>
      <Form.Item name="cs_telegram" label={t('config.cs_telegram')}>
        <Input placeholder={t('config.cs_telegram_placeholder')} />
      </Form.Item>
      <Form.Item name="cs_whatsapp" label={t('config.cs_whatsapp')}>
        <Input placeholder={t('config.cs_whatsapp_placeholder')} />
      </Form.Item>
      <Form.Item name="cs_online_url" label={t('config.cs_online_url')}>
        <Input placeholder={t('config.cs_online_url_placeholder')} />
      </Form.Item>
      <Form.Item name="cs_custom_link" label={t('config.cs_custom_link')}>
        <Input placeholder={t('config.cs_custom_link_placeholder')} />
      </Form.Item>
      <Form.Item name="cs_remark" label={t('config.cs_remark')}>
        <Input.TextArea rows={3} placeholder={t('config.cs_remark_placeholder')} />
      </Form.Item>
      <PermissionButton permission="config.save" type="primary" loading={submitting} onClick={() => void handleSubmit()}>
        {t('common.save')}
      </PermissionButton>
    </Form>
  )
}

function LoginSecurityConfigPanel() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getConfigByGroup('login_security')
      const values = configsToForm(
        res.data?.configs,
        [
          'password_min_length',
          'password_require_letter',
          'password_require_number',
          'password_require_special',
          'anomaly_alert_enabled',
        ],
        {
          password_min_length: (v) => (v ? Number(v) : 8),
          password_require_letter: (v) => v === '' || v === '1' || v === 'true',
          password_require_number: (v) => v === '' || v === '1' || v === 'true',
          password_require_special: (v) => v === '1' || v === 'true',
          anomaly_alert_enabled: (v) => v === '' || v === '1' || v === 'true',
        },
      )
      if (values.password_min_length === '' || values.password_min_length == null) {
        values.password_min_length = 8
      }
      form.setFieldsValue(values)
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }, [form, showError, t])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      await saveConfig('login_security', {
        password_min_length: String(values.password_min_length ?? 8),
        password_require_letter: values.password_require_letter ? '1' : '0',
        password_require_number: values.password_require_number ? '1' : '0',
        password_require_special: values.password_require_special ? '1' : '0',
        anomaly_alert_enabled: values.anomaly_alert_enabled ? '1' : '0',
      })
      message.success(t('config.update_success'))
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Form form={form} layout="vertical" disabled={loading}>
      <Form.Item
        name="password_min_length"
        label={t('config.password_min_length')}
        rules={[
          { required: true, message: t('config.password_min_length_required') },
          { type: 'number', min: 4, message: t('config.password_min_length_min', { min: 4 }) },
        ]}
      >
        <InputNumber min={4} max={64} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="password_require_letter" label={t('config.password_require_letter')} valuePropName="checked">
        <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
      </Form.Item>
      <Form.Item name="password_require_number" label={t('config.password_require_number')} valuePropName="checked">
        <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
      </Form.Item>
      <Form.Item name="password_require_special" label={t('config.password_require_special')} valuePropName="checked">
        <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
      </Form.Item>
      <Form.Item name="anomaly_alert_enabled" label={t('config.anomaly_alert_enabled')} valuePropName="checked">
        <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
      </Form.Item>
      <PermissionButton permission="config.save" type="primary" loading={submitting} onClick={() => void handleSubmit()}>
        {t('common.save')}
      </PermissionButton>
    </Form>
  )
}

export default function ConfigList() {
  const { t } = useTranslation()

  return (
    <PageContainer title={t('menu.config')}>
      <Card>
        <Tabs
          items={[
            { key: 'website', label: t('config.website_config'), children: <WebsiteConfigPanel /> },
            { key: 'customer_service', label: t('config.customer_service_config'), children: <CustomerServiceConfigPanel /> },
            { key: 'email', label: t('config.email_config'), children: <EmailConfigPanel /> },
            { key: 'captcha', label: t('config.captcha_config'), children: <CaptchaConfigPanel /> },
            { key: 'storage', label: t('config.storage_config'), children: <StorageConfigPanel /> },
            { key: 'login_security', label: t('config.login_security_config'), children: <LoginSecurityConfigPanel /> },
          ]}
        />
      </Card>
    </PageContainer>
  )
}
