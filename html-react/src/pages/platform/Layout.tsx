import { Button, Form, Input, Layout, Menu, Modal, Typography, message } from 'antd'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { getPlatformAdmin } from '@/utils/platformRequest'
import { logoutPlatform, updatePlatformPassword } from '@/api/platform'
import { useUnhandledError } from '@/hooks/useUnhandledError'

const { Header, Sider, Content } = Layout

export default function PlatformLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const admin = getPlatformAdmin()
  const showError = useUnhandledError()
  const [pwdOpen, setPwdOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()

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

  return (
    <Layout style={{ minHeight: '100vh' }}>
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
          <Button type="link" onClick={() => setPwdOpen(true)} style={{ color: '#93c5fd' }}>
            {t('platform.change_password')}
          </Button>
          <Button type="link" onClick={() => void onLogout()} style={{ color: '#93c5fd' }}>
            {t('header.logout')}
          </Button>
        </div>
      </Header>
      <Layout>
        <Sider width={200} theme="light">
          <Menu
            mode="inline"
            selectedKeys={[location.pathname]}
            items={[
              {
                key: '/platform/tenants',
                label: t('menu.tenant'),
                onClick: () => navigate('/platform/tenants'),
              },
              {
                key: '/platform/tenant-op-logs',
                label: t('menu.tenant_op_log'),
                onClick: () => navigate('/platform/tenant-op-logs'),
              },
            ]}
          />
        </Sider>
        <Content style={{ padding: 16, background: '#f5f7fb' }}>
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
    </Layout>
  )
}
