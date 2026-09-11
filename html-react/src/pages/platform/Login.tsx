import { useState } from 'react'
import { App, Button, Form, Input, Typography } from 'antd'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { completePlatformLogin, platformLogin } from '@/api/platform'
import { useUnhandledError } from '@/hooks/useUnhandledError'

export default function PlatformLogin() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const navigate = useNavigate()
  const showError = useUnhandledError()
  const [loading, setLoading] = useState(false)
  const [form] = Form.useForm()

  const onFinish = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      const res = await platformLogin({
        username: String(values.username || '').trim(),
        password: values.password,
      })
      completePlatformLogin(res as { data?: { token?: string; admin?: unknown } })
      message.success(t('login.login_success'))
      navigate('/platform/tenants', { replace: true })
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'linear-gradient(145deg, #0f172a 0%, #1e293b 55%, #334155 100%)',
        padding: 24,
      }}
    >
      <div
        style={{
          width: '100%',
          maxWidth: 420,
          background: '#fff',
          borderRadius: 12,
          padding: '36px 32px 28px',
          boxShadow: '0 20px 50px rgba(0,0,0,0.25)',
        }}
      >
        <Typography.Title level={3} style={{ marginTop: 0 }}>
          {t('platform.title')}
        </Typography.Title>
        <Typography.Paragraph type="secondary">{t('platform.login_hint')}</Typography.Paragraph>
        <Form form={form} layout="vertical" onFinish={(v) => void onFinish(v)}>
          <Form.Item
            name="username"
            rules={[{ required: true, message: t('login.username_required') }]}
          >
            <Input size="large" placeholder={t('login.username')} />
          </Form.Item>
          <Form.Item
            name="password"
            rules={[{ required: true, message: t('login.password_required') }]}
          >
            <Input.Password size="large" placeholder={t('login.password')} />
          </Form.Item>
          <Button type="primary" htmlType="submit" size="large" block loading={loading}>
            {t('login.login')}
          </Button>
        </Form>
        <div style={{ marginTop: 16, textAlign: 'center' }}>
          <Link to="/login">{t('platform.back_tenant_login')}</Link>
        </div>
      </div>
    </div>
  )
}
