import { Button, Form, Input, Layout, Menu, Modal, Typography, message, theme } from 'antd'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { getPlatformAdmin } from '@/utils/platformRequest'
import {
  bindPlatformSecurity2FA,
  getPlatformSecurityQRCode,
  getPlatformSecurityStatus,
  logoutPlatform,
  unbindPlatformSecurity2FA,
  updatePlatformAllowedIPs,
  updatePlatformPassword,
} from '@/api/platform'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import DarkModeSwitch from '@/components/DarkModeSwitch'
import { useAppStore } from '@/stores/app'

const { Header, Sider, Content } = Layout

export default function PlatformLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const { token } = theme.useToken()
  const darkMode = useAppStore((s) => s.darkMode)
  const admin = getPlatformAdmin()
  const isViewer = admin?.role === 'viewer'
  const showError = useUnhandledError()
  const [pwdOpen, setPwdOpen] = useState(false)
  const [secOpen, setSecOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [secLoading, setSecLoading] = useState(false)
  const [bound2fa, setBound2fa] = useState(false)
  const [qr, setQr] = useState('')
  const [secret, setSecret] = useState('')
  const [form] = Form.useForm()
  const [secForm] = Form.useForm()
  const [bindForm] = Form.useForm()
  const [unbindForm] = Form.useForm()

  const onLogout = async () => {
    await logoutPlatform()
    navigate('/platform/login', { replace: true })
  }

  const onSubmitPwd = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      await updatePlatformPassword(values)
      message.success(t('platform.password_updated'))
      setPwdOpen(false)
      form.resetFields()
    } catch (e) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showError(e, t('platform.change_password'))
    } finally {
      setSaving(false)
    }
  }

  const openSecurity = async () => {
    setSecOpen(true)
    setSecLoading(true)
    setQr('')
    setSecret('')
    try {
      const res = await getPlatformSecurityStatus()
      const data = (res as { data?: { bound?: boolean; allowed_ips?: string } })?.data
      setBound2fa(!!data?.bound)
      secForm.setFieldsValue({ allowed_ips: data?.allowed_ips || '' })
    } catch (e) {
      showError(e, t('common.operation_failed'))
    } finally {
      setSecLoading(false)
    }
  }

  const loadQR = async () => {
    try {
      const res = await getPlatformSecurityQRCode()
      const data = (res as { data?: { qr_code?: string; secret?: string } })?.data
      setQr(data?.qr_code || '')
      setSecret(data?.secret || '')
    } catch (e) {
      showError(e, t('common.operation_failed'))
    }
  }

  const onBind2fa = async () => {
    try {
      const values = await bindForm.validateFields()
      await bindPlatformSecurity2FA({ secret, code: values.code })
      message.success(t('platform.bind_2fa_success'))
      setBound2fa(true)
      setQr('')
      setSecret('')
      bindForm.resetFields()
    } catch (e) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showError(e, t('common.operation_failed'))
    }
  }

  const onUnbind2fa = async () => {
    try {
      const values = await unbindForm.validateFields()
      await unbindPlatformSecurity2FA({ code: values.code })
      message.success(t('platform.unbind_2fa_success'))
      setBound2fa(false)
      unbindForm.resetFields()
    } catch (e) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showError(e, t('common.operation_failed'))
    }
  }

  const onSaveIPs = async () => {
    try {
      const values = await secForm.validateFields()
      await updatePlatformAllowedIPs({ allowed_ips: values.allowed_ips || '' })
      message.success(t('platform.allowed_ips_updated'))
    } catch (e) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showError(e, t('common.operation_failed'))
    }
  }

  return (
    <Layout style={{ minHeight: '100vh', background: token.colorBgLayout }}>
      <Header
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          background: '#0f172a',
          padding: '0 20px',
        }}
      >
        <Typography.Text strong style={{ color: '#fff', fontSize: 16 }}>
          {t('platform.title')}
        </Typography.Text>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Typography.Text style={{ color: 'rgba(255,255,255,0.85)' }}>
            {admin?.name || admin?.username || ''}
          </Typography.Text>
          {isViewer ? (
            <Typography.Text style={{ color: '#cbd5e1', fontSize: 12 }}>
              {t('platform.role_viewer')}
            </Typography.Text>
          ) : null}
          <DarkModeSwitch className="layout-header__icon-btn platform-header__icon-btn" />
          <Button type="link" onClick={() => setPwdOpen(true)} style={{ color: '#93c5fd' }}>
            {t('platform.change_password')}
          </Button>
          <Button type="link" onClick={() => void openSecurity()} style={{ color: '#93c5fd' }}>
            {t('platform.security')}
          </Button>
          <Button type="link" onClick={() => void onLogout()} style={{ color: '#93c5fd' }}>
            {t('header.logout')}
          </Button>
        </div>
      </Header>
      <Layout>
        <Sider
          width={200}
          theme={darkMode ? 'dark' : 'light'}
          style={{
            background: darkMode ? token.colorBgContainer : '#fff',
            borderRight: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          <Menu
            mode="inline"
            theme={darkMode ? 'dark' : 'light'}
            style={{ background: 'transparent', borderInlineEnd: 'none' }}
            selectedKeys={[location.pathname]}
            items={[
              {
                key: '/platform/overview',
                label: t('menu.platform_overview'),
                onClick: () => navigate('/platform/overview'),
              },
              {
                key: '/platform/tenants',
                label: t('menu.tenant'),
                onClick: () => navigate('/platform/tenants'),
              },
              {
                key: '/platform/admins',
                label: t('menu.platform_admin'),
                onClick: () => navigate('/platform/admins'),
              },
              {
                key: '/platform/tenant-op-logs',
                label: t('menu.tenant_op_log'),
                onClick: () => navigate('/platform/tenant-op-logs'),
              },
              {
                key: '/platform/alert-deliveries',
                label: t('menu.platform_alert'),
                onClick: () => navigate('/platform/alert-deliveries'),
              },
              {
                key: '/platform/system-logs',
                label: t('menu.system_log'),
                onClick: () => navigate('/platform/system-logs'),
              },
              {
                key: '/platform/login-logs',
                label: t('menu.login_log'),
                onClick: () => navigate('/platform/login-logs'),
              },
              {
                key: '/platform/operation-logs',
                label: t('menu.operation_log'),
                onClick: () => navigate('/platform/operation-logs'),
              },
            ]}
          />
        </Sider>
        <Content style={{ padding: 16, background: token.colorBgLayout }}>
          <Outlet />
        </Content>
      </Layout>

      <Modal
        title={t('platform.change_password')}
        open={pwdOpen}
        onCancel={() => {
          setPwdOpen(false)
          form.resetFields()
        }}
        onOk={() => void onSubmitPwd()}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="old_password"
            label={t('platform.old_password')}
            rules={[{ required: true }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="new_password"
            label={t('platform.new_password')}
            rules={[{ required: true, min: 6 }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="confirm_password"
            label={t('platform.confirm_password')}
            dependencies={['new_password']}
            rules={[
              { required: true },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) return Promise.resolve()
                  return Promise.reject(new Error(t('platform.confirm_password')))
                },
              }),
            ]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('platform.security')}
        open={secOpen}
        onCancel={() => setSecOpen(false)}
        footer={null}
        width={560}
        destroyOnClose
      >
        {secLoading ? (
          <Typography.Text type="secondary">{t('common.loading')}</Typography.Text>
        ) : (
          <>
            <Typography.Paragraph>
              {t('platform.2fa_status')}: {bound2fa ? t('common.yes') : t('common.no')}
            </Typography.Paragraph>
            {!bound2fa ? (
              <>
                <Button onClick={() => void loadQR()} style={{ marginBottom: 12 }}>
                  {t('platform.generate_2fa_qr')}
                </Button>
                {qr ? <img src={qr} alt="2fa" style={{ maxWidth: 200, display: 'block', marginBottom: 12 }} /> : null}
                {secret ? (
                  <Typography.Paragraph type="secondary" copyable>
                    {secret}
                  </Typography.Paragraph>
                ) : null}
                <Form form={bindForm} layout="vertical" onFinish={() => void onBind2fa()}>
                  <Form.Item name="code" label={t('platform.google_code')} rules={[{ required: true }]}>
                    <Input />
                  </Form.Item>
                  <Button type="primary" htmlType="submit" disabled={!secret}>
                    {t('platform.bind_2fa')}
                  </Button>
                </Form>
              </>
            ) : (
              <Form form={unbindForm} layout="vertical" onFinish={() => void onUnbind2fa()}>
                <Form.Item name="code" label={t('platform.google_code')} rules={[{ required: true }]}>
                  <Input />
                </Form.Item>
                <Button danger htmlType="submit">
                  {t('platform.unbind_2fa')}
                </Button>
              </Form>
            )}
            <Typography.Title level={5} style={{ marginTop: 24 }}>
              {t('platform.allowed_ips')}
            </Typography.Title>
            <Typography.Paragraph type="secondary">{t('platform.allowed_ips_hint')}</Typography.Paragraph>
            <Form form={secForm} layout="vertical" onFinish={() => void onSaveIPs()}>
              <Form.Item name="allowed_ips">
                <Input.TextArea rows={3} placeholder="1.2.3.4, 10.0.0.0/8" />
              </Form.Item>
              <Button type="primary" htmlType="submit">
                {t('common.save')}
              </Button>
            </Form>
          </>
        )}
      </Modal>
    </Layout>
  )
}
