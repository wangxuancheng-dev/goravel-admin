import { Button, Result } from 'antd'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

/** Legacy menu entry: tenant mgmt moved to platform console. */
export default function TenantList() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  return (
    <Result
      status="info"
      title={t('platform.moved_title')}
      subTitle={t('platform.moved_hint')}
      extra={
        <Button type="primary" onClick={() => navigate('/platform/tenants')}>
          {t('platform.go_platform')}
        </Button>
      }
    />
  )
}
