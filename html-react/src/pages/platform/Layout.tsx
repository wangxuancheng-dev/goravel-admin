import { Button, Layout, Menu, Typography } from 'antd'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { getPlatformAdmin } from '@/utils/platformRequest'
import { logoutPlatform } from '@/api/platform'

const { Header, Sider, Content } = Layout

export default function PlatformLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const admin = getPlatformAdmin()

  const onLogout = async () => {
    await logoutPlatform()
    navigate('/platform/login', { replace: true })
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
            ]}
          />
        </Sider>
        <Content style={{ padding: 16, background: '#f5f7fb' }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
