import { useState } from 'react'
import { App, Button, Form, Input, Typography, theme } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { completePlatformLogin, getPlatformLoginCaptcha, platformLogin } from '@/api/platform'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { getTenantAdminLoginUrl } from '@/utils/tenant'
import { ERROR_CODES, type ApiError } from '@/types'
import DarkModeSwitch from '@/components/DarkModeSwitch'
import LanguageSwitch from '@/components/LanguageSwitch'

export default function PlatformLogin() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const { token } = theme.useToken()
  const navigate = useNavigate()
  const showError = useUnhandledError()
  const [loading, setLoading] = useState(false)
  const [needGoogleCode, setNeedGoogleCode] = useState(false)
  const [captcha, setCaptcha] = useState({ id: '', image: '', shouldShow: false })
  const [form] = Form.useForm()

  const fetchCaptcha = async () => {
    try {
      const res = await getPlatformLoginCaptcha()
      const info = (res as { data?: { captcha?: { captcha_id?: string; captcha_image?: string } } })?.data
        ?.captcha
      setCaptcha({
        id: info?.captcha_id || '',
        image: info?.captcha_image || '',
        shouldShow: true,
      })
      form.setFieldValue('captcha_answer', undefined)
    } catch (error) {
      setCaptcha({ id: '', image: '', shouldShow: true })
      showError(error, t('common.operation_failed'))
    }
  }

  const onFinish = async (values: {
    username: string
    password: string
    captcha_answer?: string
    google_code?: string
  }) => {
    setLoading(true)
    try {
      const payload: {
        username: string
        password: string
        google_code?: string
        captcha_id?: string
        captcha_answer?: string
      } = {
        username: String(values.username || '').trim(),
        password: values.password,
      }
      if (needGoogleCode) {
        payload.google_code = values.google_code
      } else if (captcha.shouldShow) {
        payload.captcha_id = captcha.id
        payload.captcha_answer = values.captcha_answer
      }
      const res = await platformLogin(payload)
      completePlatformLogin(res as { data?: { token?: string; admin?: unknown } })
      message.success(t('login.login_success'))
      navigate('/platform/overview', { replace: true })
    } catch (error) {
      const err = error as ApiError
      const code = err.errorCode || ''

      if (code === ERROR_CODES.GOOGLE_CODE_REQUIRED) {
        setNeedGoogleCode(true)
        setCaptcha({ id: '', image: '', shouldShow: false })
        form.setFieldValue('captcha_answer', undefined)
        form.setFieldValue('google_code', undefined)
        message.warning(err.message || t('login.google_code_required'))
        return
      }

      if (code === ERROR_CODES.GOOGLE_CODE_INVALID) {
        form.setFieldValue('google_code', undefined)
        if (!err.__handled) {
          message.error(err.translatedMessage || err.message || t('login.login_failed'))
        }
        return
      }

      const captchaError =
        code === ERROR_CODES.CAPTCHA_REQUIRED ||
        code === ERROR_CODES.CAPTCHA_INVALID ||
        code === ERROR_CODES.CAPTCHA_EXPIRED

      if (!needGoogleCode && captchaError) {
        await fetchCaptcha()
        if (!err.__handled && code !== ERROR_CODES.CAPTCHA_REQUIRED) {
          message.error(err.translatedMessage || err.message || t('common.operation_failed'))
        } else if (code === ERROR_CODES.CAPTCHA_REQUIRED) {
          message.info(t('login.captcha_required'))
        }
        return
      }

      if (captcha.shouldShow && !needGoogleCode) {
        await fetchCaptcha()
      }
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
        position: 'relative',
      }}
    >
      <div style={{ position: 'absolute', top: 20, right: 20, display: 'flex', alignItems: 'center', gap: 4 }}>
        <DarkModeSwitch className="layout-header__icon-btn platform-header__icon-btn" />
        <LanguageSwitch className="layout-header__icon-btn platform-header__icon-btn" />
      </div>
      <div
        style={{
          width: '100%',
          maxWidth: 420,
          background: token.colorBgContainer,
          borderRadius: 12,
          padding: '36px 32px 28px',
          boxShadow: '0 20px 50px rgba(0,0,0,0.25)',
          border: `1px solid ${token.colorBorderSecondary}`,
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
          {needGoogleCode ? (
            <Form.Item
              name="google_code"
              label={t('platform.google_code')}
              rules={[
                { required: true, message: t('login.google_code_required') },
                { pattern: /^\d{6}$/, message: t('login.google_code_format') },
              ]}
            >
              <Input size="large" placeholder={t('platform.google_code_placeholder')} maxLength={6} />
            </Form.Item>
          ) : null}
          {captcha.shouldShow && !needGoogleCode ? (
            <>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                {captcha.image ? (
                  <img
                    src={captcha.image}
                    alt={t('login.captcha_alt')}
                    onClick={() => void fetchCaptcha()}
                    style={{
                      height: 40,
                      borderRadius: 4,
                      cursor: 'pointer',
                      border: `1px solid ${token.colorBorder}`,
                    }}
                  />
                ) : null}
                <Button type="link" icon={<ReloadOutlined />} onClick={() => void fetchCaptcha()}>
                  {t('login.refresh_captcha')}
                </Button>
              </div>
              <Form.Item
                name="captcha_answer"
                rules={[{ required: true, message: t('login.captcha_required') }]}
              >
                <Input size="large" placeholder={t('login.captcha_placeholder')} />
              </Form.Item>
            </>
          ) : null}
          <Button type="primary" htmlType="submit" size="large" block loading={loading}>
            {t('login.login')}
          </Button>
        </Form>
        <div style={{ marginTop: 16, textAlign: 'center' }}>
          <Typography.Text type="secondary">demo / demo123</Typography.Text>
        </div>
        <div style={{ marginTop: 8, textAlign: 'center' }}>
          <a href={getTenantAdminLoginUrl()}>{t('platform.back_tenant_login')}</a>
        </div>
      </div>
    </div>
  )
}
