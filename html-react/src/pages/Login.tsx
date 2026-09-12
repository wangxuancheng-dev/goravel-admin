import { useEffect, useMemo, useState } from 'react'
import { App, Button, Form, Input, Space, Typography, theme } from 'antd'
import { LockOutlined, UserOutlined, ReloadOutlined, BankOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { login, getLoginCaptcha } from '@/api/auth'
import { useUserStore } from '@/stores/user'
import { useAppStore, THEME_COLORS } from '@/stores/app'
import { ERROR_CODES, type ApiError } from '@/types'
import LanguageSwitch from '@/components/LanguageSwitch'
import DarkModeSwitch from '@/components/DarkModeSwitch'
import {
  getTenantCode,
  isTenancyEnabled,
  resolveTenantCodeFromLocation,
  setTenantCode,
} from '@/utils/tenant'
import './Login.scss'

interface LoginFormValues {
  tenant_code?: string
  username: string
  password: string
  google_code?: string
  captcha_answer?: string
}

export default function LoginPage() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const navigate = useNavigate()
  const [form] = Form.useForm<LoginFormValues>()
  const [loading, setLoading] = useState(false)
  const [needGoogleCode, setNeedGoogleCode] = useState(false)
  const tenancyEnabled = useMemo(() => isTenancyEnabled(), [])
  const [captcha, setCaptcha] = useState<{
    enabled: boolean
    id: string
    image: string
    shouldShow: boolean
  }>({
    enabled: false,
    id: '',
    image: '',
    shouldShow: false,
  })
  const setToken = useUserStore((s) => s.setToken)
  const fetchUserInfo = useUserStore((s) => s.fetchUserInfo)
  const themeColor = useAppStore((s) => s.themeColor)
  const setThemeColor = useAppStore((s) => s.setThemeColor)
  const { token } = theme.useToken()

  /** Check whether captcha is enabled (do not show image yet). */
  const checkCaptchaEnabled = async () => {
    try {
      if (tenancyEnabled) {
        setTenantCode(form.getFieldValue('tenant_code'))
      }
      const res = await getLoginCaptcha({ check: true })
      const info = res.data?.captcha
      setCaptcha((prev) => ({
        ...prev,
        enabled: !!info?.enabled,
        shouldShow: false,
        id: '',
        image: '',
      }))
    } catch {
      setCaptcha({ enabled: false, id: '', image: '', shouldShow: false })
    }
  }

  /** Fetch captcha image and show the field. */
  const fetchCaptcha = async () => {
    try {
      if (tenancyEnabled) {
        setTenantCode(form.getFieldValue('tenant_code'))
      }
      const res = await getLoginCaptcha()
      const info = res.data?.captcha
      setCaptcha({
        enabled: !!info?.enabled,
        id: info?.captcha_id || '',
        image: info?.captcha_image || '',
        shouldShow: true,
      })
      form.setFieldValue('captcha_answer', undefined)
    } catch {
      setCaptcha({ enabled: false, id: '', image: '', shouldShow: false })
    }
  }

  useEffect(() => {
    const fromQuery = resolveTenantCodeFromLocation()
    const initialTenant = fromQuery || getTenantCode()
    if (initialTenant) {
      form.setFieldValue('tenant_code', initialTenant)
      setTenantCode(initialTenant)
    }
    void checkCaptchaEnabled()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount only
  }, [])

  const handleSubmit = async (values: LoginFormValues) => {
    setLoading(true)
    try {
      if (tenancyEnabled) {
        setTenantCode(values.tenant_code)
      }
      const payload = {
        username: values.username,
        password: values.password,
        ...(tenancyEnabled && values.tenant_code
          ? { tenant_code: String(values.tenant_code).trim().toLowerCase() }
          : {}),
        ...(needGoogleCode ? { google_code: values.google_code } : {}),
        ...(!needGoogleCode && captcha.shouldShow
          ? { captcha_id: captcha.id, captcha_answer: values.captcha_answer }
          : {}),
      }

      const res = await login(payload)
      const tokenValue = (res.data as { token?: string } | undefined)?.token
      if (tokenValue) {
        setToken(tokenValue)
      }

      await fetchUserInfo(true)
      message.success(t('login.login_success'))
      const mustChange = !!useUserStore.getState().adminInfo?.must_change_password
      navigate(mustChange ? '/profile?tab=password' : '/dashboard', { replace: true })
    } catch (error) {
      const err = error as ApiError
      const code = err.errorCode || ''

      if (code === ERROR_CODES.GOOGLE_CODE_REQUIRED) {
        setNeedGoogleCode(true)
        setCaptcha((prev) => ({ ...prev, shouldShow: false, id: '', image: '' }))
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

      // Show captcha after captcha errors or other login failures when captcha is enabled
      const captchaError =
        code === ERROR_CODES.CAPTCHA_REQUIRED ||
        code === ERROR_CODES.CAPTCHA_INVALID ||
        code === ERROR_CODES.CAPTCHA_EXPIRED

      if (captcha.enabled && !needGoogleCode && (!captcha.shouldShow || captchaError)) {
        await fetchCaptcha()
      } else if (captcha.shouldShow && !needGoogleCode) {
        await fetchCaptcha()
      }

      if (!err.__handled) {
        message.error(err.translatedMessage || err.message || t('login.login_failed'))
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page" style={{ ['--login-primary' as string]: token.colorPrimary }}>
      <div className="login-page__left">
        <div className="login-page__brand">
          <div className="brand-logo">
            <LockOutlined />
          </div>
          <Typography.Title level={2} className="brand-title">
            {t('login.title')}
          </Typography.Title>
          <Typography.Paragraph className="brand-desc">{t('login.page_description')}</Typography.Paragraph>
        </div>
        <div className="login-page__deco" aria-hidden>
          <span className="deco-circle deco-circle--1" />
          <span className="deco-circle deco-circle--2" />
          <span className="deco-circle deco-circle--3" />
        </div>
      </div>

      <div className="login-page__right">
        <div className="login-page__toolbar">
          <div className="login-toolbar__theme">
            {THEME_COLORS.map((item) => (
              <button
                key={item.key}
                type="button"
                className={`login-theme-swatch ${themeColor === item.key ? 'active' : ''}`}
                style={{ backgroundColor: item.color }}
                onClick={() => setThemeColor(item.key)}
                title={item.key}
              />
            ))}
          </div>
          <DarkModeSwitch />
          <LanguageSwitch />
        </div>

        <div className="login-page__form-wrap">
          <div className="login-form-card">
            <Typography.Title level={3}>{t('login.login')}</Typography.Title>
            <Typography.Paragraph type="secondary">{t('login.page_description')}</Typography.Paragraph>

            <Form form={form} layout="vertical" size="large" onFinish={handleSubmit} requiredMark={false}>
              {tenancyEnabled && (
                <Form.Item
                  name="tenant_code"
                  rules={[{ required: true, message: t('login.tenant_code_required') }]}
                >
                  <Input
                    prefix={<BankOutlined />}
                    placeholder={t('login.tenant_code_placeholder')}
                    autoComplete="organization"
                    onChange={(e) => {
                      setTenantCode(e.target.value)
                    }}
                    onBlur={(e) => {
                      setTenantCode(e.target.value)
                      if (tenancyEnabled && String(e.target.value || '').trim()) {
                        void checkCaptchaEnabled()
                      }
                    }}
                  />
                </Form.Item>
              )}

              <Form.Item
                name="username"
                rules={[{ required: true, message: t('login.username_required') }]}
              >
                <Input prefix={<UserOutlined />} placeholder={t('login.username')} autoComplete="username" />
              </Form.Item>

              <Form.Item
                name="password"
                rules={[{ required: true, message: t('login.password_required') }]}
              >
                <Input.Password
                  prefix={<LockOutlined />}
                  placeholder={t('login.password')}
                  autoComplete="current-password"
                />
              </Form.Item>

              {needGoogleCode && (
                <Form.Item
                  name="google_code"
                  rules={[
                    { required: true, message: t('login.google_code_required') },
                    { pattern: /^\d{6}$/, message: t('login.google_code_format') },
                  ]}
                >
                  <Input placeholder={t('login.google_code_placeholder')} maxLength={6} />
                </Form.Item>
              )}

              {captcha.shouldShow && !needGoogleCode && (
                <>
                  <div className="captcha-row">
                    {captcha.image ? (
                      <img
                        src={captcha.image}
                        alt={t('login.captcha_alt')}
                        className="captcha-image"
                        onClick={() => void fetchCaptcha()}
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
                    <Input placeholder={t('login.captcha_placeholder')} />
                  </Form.Item>
                </>
              )}

              <Form.Item>
                <Button type="primary" htmlType="submit" block loading={loading}>
                  {t('login.login')}
                </Button>
              </Form.Item>
            </Form>

            <Space className="login-hint" size={4}>
              <Typography.Text type="secondary">demo / demo123</Typography.Text>
            </Space>
            {tenancyEnabled && (
              <div style={{ marginTop: 12, textAlign: 'center' }}>
                <Button type="link" onClick={() => navigate('/platform/login')}>
                  {t('platform.title')}
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
