import { useEffect, useState } from 'react'
import { App, Alert, Avatar, Button, Card, Form, Input, Modal, Space, Spin, Steps, Tabs, Typography } from 'antd'
import { LoadingOutlined, UserOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { updatePassword, updateProfile } from '@/api/profile'
import {
  bindGoogleAuthenticator,
  getGoogleAuthenticatorQRCode,
  getGoogleAuthenticatorStatus,
  unbindGoogleAuthenticator,
} from '@/api/auth'
import { useUserStore } from '@/stores/user'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { usePermission } from '@/hooks/usePermission'
import PageContainer from '@/components/PageContainer'
import ProfileSessionsPanel from './ProfileSessionsPanel'
import './Profile.scss'

type GoogleAuthQrData = {
  secret?: string
  qrCodeImage?: string
}

const DEFAULT_AVATARS = [
  'https://ui-avatars.com/api/?name=A&background=409EFF&color=fff&size=128',
  'https://ui-avatars.com/api/?name=B&background=67C23A&color=fff&size=128',
  'https://ui-avatars.com/api/?name=C&background=E6A23C&color=fff&size=128',
  'https://ui-avatars.com/api/?name=D&background=F56C6C&color=fff&size=128',
  'https://ui-avatars.com/api/?name=E&background=9C27B0&color=fff&size=128',
  'https://ui-avatars.com/api/?name=F&background=00BCD4&color=fff&size=128',
  'https://ui-avatars.com/api/?name=G&background=FF9800&color=fff&size=128',
  'https://ui-avatars.com/api/?name=H&background=4CAF50&color=fff&size=128',
  'https://ui-avatars.com/api/?name=I&background=2196F3&color=fff&size=128',
  'https://ui-avatars.com/api/?name=J&background=FF5722&color=fff&size=128',
  'https://ui-avatars.com/api/?name=K&background=795548&color=fff&size=128',
  'https://ui-avatars.com/api/?name=L&background=607D8B&color=fff&size=128',
  'https://ui-avatars.com/api/?name=M&background=3F51B5&color=fff&size=128',
  'https://ui-avatars.com/api/?name=N&background=009688&color=fff&size=128',
  'https://ui-avatars.com/api/?name=O&background=FFC107&color=333&size=128',
  'https://ui-avatars.com/api/?name=P&background=E91E63&color=fff&size=128',
  'https://ui-avatars.com/api/?name=Q&background=8BC34A&color=fff&size=128',
  'https://ui-avatars.com/api/?name=R&background=CDDC39&color=fff&size=128',
  'https://ui-avatars.com/api/?name=S&background=FFEB3B&color=333&size=128',
  'https://ui-avatars.com/api/?name=T&background=FF9800&color=fff&size=128',
  'https://ui-avatars.com/api/?name=U&background=9E9E9E&color=fff&size=128',
  'https://ui-avatars.com/api/?name=V&background=673AB7&color=fff&size=128',
  'https://ui-avatars.com/api/?name=W&background=00ACC1&color=fff&size=128',
  'https://ui-avatars.com/api/?name=X&background=5C6BC0&color=fff&size=128',
  'https://ui-avatars.com/api/?name=Y&background=F44336&color=fff&size=128',
  'https://ui-avatars.com/api/?name=Z&background=26A69A&color=fff&size=128',
]

export default function ProfilePage() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const googleAuthPerm = getButtonState('google_authenticator.manage')
  const adminInfo = useUserStore((s) => s.adminInfo)
  const passwordPerm = getButtonState('password.update')
  const mustChangePassword = !!adminInfo?.must_change_password
  const canManageOwnPassword = mustChangePassword || passwordPerm.show
  const canSubmitOwnPassword = mustChangePassword || !passwordPerm.disabled
  const fetchUserInfo = useUserStore((s) => s.fetchUserInfo)
  const [searchParams] = useSearchParams()
  const [activeTab, setActiveTab] = useState(searchParams.get('tab') === 'password' ? 'password' : 'basic')
  const [profileForm] = Form.useForm()
  const [passwordForm] = Form.useForm()
  const [bindForm] = Form.useForm()
  const [unbindForm] = Form.useForm()
  const [savingProfile, setSavingProfile] = useState(false)
  const [savingPassword, setSavingPassword] = useState(false)
  const [bound, setBound] = useState(false)
  const [qr, setQr] = useState<GoogleAuthQrData>({})
  const [loading2fa, setLoading2fa] = useState(false)
  const [bindStep, setBindStep] = useState(0)
  const [binding, setBinding] = useState(false)
  const [unbinding, setUnbinding] = useState(false)
  const [avatarOpen, setAvatarOpen] = useState(false)
  const [selectedAvatar, setSelectedAvatar] = useState('')
  const [savingAvatar, setSavingAvatar] = useState(false)

  useEffect(() => {
    profileForm.setFieldsValue({
      nickname: adminInfo?.nickname || '',
      email: adminInfo?.email || '',
      phone: adminInfo?.phone || '',
    })
  }, [adminInfo, profileForm])

  const load2fa = async () => {
    if (googleAuthPerm.disabled) return
    setLoading2fa(true)
    try {
      const statusRes = await getGoogleAuthenticatorStatus()
      const isBound = !!(statusRes.data as { is_bound?: boolean } | undefined)?.is_bound
      setBound(isBound)
      if (!isBound) {
        const qrRes = await getGoogleAuthenticatorQRCode()
        const data = qrRes.data as {
          secret?: string
          qr_code_image?: string
          qr_code_url?: string
        }
        setQr({
          secret: data?.secret,
          qrCodeImage: data?.qr_code_image,
        })
        setBindStep(0)
      } else {
        setQr({})
        setBindStep(0)
      }
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading2fa(false)
    }
  }

  const handleProfileSave = async () => {
    try {
      const values = await profileForm.validateFields()
      setSavingProfile(true)
      await updateProfile(values)
      message.success(t('common.update_success'))
      await fetchUserInfo(true)
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSavingProfile(false)
    }
  }

  const handlePasswordSave = async () => {
    if (!canSubmitOwnPassword) return
    try {
      const values = await passwordForm.validateFields()
      setSavingPassword(true)
      await updatePassword({
        old_password: values.old_password,
        new_password: values.new_password,
        confirm_password: values.confirm_password,
      })
      message.success(t('common.update_success'))
      passwordForm.resetFields()
      await fetchUserInfo(true)
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSavingPassword(false)
    }
  }

  const handleBind = async () => {
    if (!qr.secret) {
      message.warning(t('profile.save_secret_tip'))
      return
    }
    try {
      const values = await bindForm.validateFields()
      setBinding(true)
      await bindGoogleAuthenticator({ secret: qr.secret, code: values.code })
      message.success(t('profile.bind_success'))
      bindForm.resetFields()
      setBindStep(0)
      await load2fa()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('profile.bind_failed'))
    } finally {
      setBinding(false)
    }
  }

  const handleUnbind = async () => {
    try {
      const values = await unbindForm.validateFields()
      setUnbinding(true)
      await unbindGoogleAuthenticator({ code: values.code })
      message.success(t('profile.unbind_success'))
      unbindForm.resetFields()
      await load2fa()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('profile.unbind_failed'))
    } finally {
      setUnbinding(false)
    }
  }

  const openAvatarDialog = () => {
    setSelectedAvatar(adminInfo?.avatar || '')
    setAvatarOpen(true)
  }

  const handleSaveAvatar = async () => {
    if (!selectedAvatar) {
      message.warning(t('profile.please_select_avatar'))
      return
    }
    setSavingAvatar(true)
    try {
      await updateProfile({ avatar: selectedAvatar })
      message.success(t('profile.avatar_update_success'))
      setAvatarOpen(false)
      await fetchUserInfo(true)
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setSavingAvatar(false)
    }
  }

  return (
    <PageContainer title={t('menu.profile')}>
      {adminInfo?.must_change_password ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          message={t('profile.must_change_password_title')}
          description={t('profile.must_change_password_desc')}
        />
      ) : null}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={(key) => {
            setActiveTab(key)
            if (key === '2fa' && !googleAuthPerm.disabled) void load2fa()
            if (key === 'sessions') {
              /* panel self-loads */
            }
          }}
          items={([
            {
              key: 'basic',
              label: t('common.basic_info'),
              children: (
                <div style={{ maxWidth: 520 }}>
                  <div style={{ textAlign: 'center', marginBottom: 24 }}>
                    <Avatar size={100} src={adminInfo?.avatar} icon={<UserOutlined />} />
                    <div style={{ marginTop: 12 }}>
                      <Button type="link" onClick={openAvatarDialog}>
                        {t('profile.change_avatar')}
                      </Button>
                    </div>
                  </div>
                  <Form
                    form={profileForm}
                    layout="vertical"
                    onFinish={() => void handleProfileSave()}
                  >
                    <Form.Item label={t('table.username')}>
                      <Input value={adminInfo?.username || ''} disabled />
                    </Form.Item>
                    <Form.Item name="nickname" label={t('table.nickname')}>
                      <Input />
                    </Form.Item>
                    <Form.Item name="email" label={t('table.email')}>
                      <Input />
                    </Form.Item>
                    <Form.Item name="phone" label={t('table.phone')}>
                      <Input />
                    </Form.Item>
                    <Form.Item>
                      <Button type="primary" htmlType="submit" loading={savingProfile}>
                        {t('common.save')}
                      </Button>
                    </Form.Item>
                  </Form>
                </div>
              ),
            },
            ...(canManageOwnPassword
              ? [{
              key: 'password',
              label: t('common.change_password'),
              children: (
                <Form
                  form={passwordForm}
                  layout="vertical"
                  style={{ maxWidth: 480 }}
                  onFinish={() => void handlePasswordSave()}
                >
                  <Form.Item name="old_password" label={t('common.old_password')} rules={[{ required: true }]}>
                    <Input.Password />
                  </Form.Item>
                  <Form.Item name="new_password" label={t('common.new_password')} rules={[{ required: true }]}>
                    <Input.Password />
                  </Form.Item>
                  <Form.Item
                    name="confirm_password"
                    label={t('common.confirm_password')}
                    dependencies={['new_password']}
                    rules={[
                      { required: true },
                      ({ getFieldValue }) => ({
                        validator(_, value) {
                          if (!value || getFieldValue('new_password') === value) return Promise.resolve()
                          return Promise.reject(new Error(t('common.password_mismatch')))
                        },
                      }),
                    ]}
                  >
                    <Input.Password />
                  </Form.Item>
                  <Form.Item>
                    <Button type="primary" htmlType="submit" loading={savingPassword} disabled={!canSubmitOwnPassword}>
                      {t('common.save')}
                    </Button>
                  </Form.Item>
                </Form>
              ),
            }]
              : []),
            {
              key: '2fa',
              label: t('profile.google_authenticator', { defaultValue: 'Google Authenticator' }),
              disabled: googleAuthPerm.disabled,
              children: (
                <div className="profile-google-auth">
                  {bound ? (
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                      <Alert
                        type="success"
                        showIcon
                        message={t('profile.google_auth_bound', { defaultValue: '已绑定谷歌验证器' })}
                      />
                      <Form
                        form={unbindForm}
                        layout="vertical"
                        className="profile-google-auth__verify-form"
                        onFinish={() => void handleUnbind()}
                      >
                        <Form.Item
                          name="code"
                          label={t('profile.verification_code')}
                          rules={[
                            { required: true, message: t('profile.enter_6_digit_code') },
                            { pattern: /^\d{6}$/, message: t('login.google_code_format') },
                          ]}
                        >
                          <Input maxLength={6} placeholder={t('profile.enter_6_digit_code')} />
                        </Form.Item>
                        <Button danger htmlType="submit" loading={unbinding} disabled={googleAuthPerm.disabled}>
                          {t('profile.unbind')}
                        </Button>
                      </Form>
                    </Space>
                  ) : (
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                      <Alert
                        type="info"
                        showIcon
                        message={t('profile.google_auth_not_bound', { defaultValue: '尚未绑定谷歌验证器' })}
                      />
                      <Steps
                        className="profile-google-auth__steps"
                        current={bindStep}
                        items={[
                          { title: t('profile.step1_scan_qr') },
                          { title: t('profile.step2_verify') },
                        ]}
                      />
                      {bindStep === 0 ? (
                        <>
                          <Alert
                            className="profile-google-auth__tip"
                            type="warning"
                            showIcon
                            message={t('profile.scan_qr_tip')}
                          />
                          <div className="profile-google-auth__qr-container">
                            {loading2fa ? (
                              <div className="profile-google-auth__loading">
                                <Spin indicator={<LoadingOutlined spin />} />
                                <span>{t('common.loading')}</span>
                              </div>
                            ) : qr.qrCodeImage ? (
                              <div className="profile-google-auth__qr-wrapper">
                                <img
                                  src={qr.qrCodeImage}
                                  alt="QR Code"
                                  className="profile-google-auth__qr-image"
                                />
                                <div className="profile-google-auth__qr-info">
                                  <p>
                                    <strong>{t('profile.secret_key')}:</strong>{' '}
                                    <Typography.Text code copyable>
                                      {qr.secret}
                                    </Typography.Text>
                                  </p>
                                  <p>{t('profile.save_secret_tip')}</p>
                                </div>
                              </div>
                            ) : (
                              <Typography.Text type="secondary">{t('common.no_data')}</Typography.Text>
                            )}
                          </div>
                          <Space>
                            <Button type="primary" disabled={!qr.secret || googleAuthPerm.disabled} onClick={() => setBindStep(1)}>
                              {t('profile.next_step')}
                            </Button>
                            <Button onClick={() => void load2fa()} loading={loading2fa} disabled={googleAuthPerm.disabled}>
                              {t('common.refresh')}
                            </Button>
                          </Space>
                        </>
                      ) : (
                        <Form
                          form={bindForm}
                          layout="vertical"
                          className="profile-google-auth__verify-form"
                          onFinish={() => void handleBind()}
                        >
                          <Form.Item
                            name="code"
                            label={t('profile.verification_code')}
                            rules={[
                              { required: true, message: t('profile.enter_6_digit_code') },
                              { pattern: /^\d{6}$/, message: t('login.google_code_format') },
                            ]}
                          >
                            <Input maxLength={6} placeholder={t('profile.enter_6_digit_code')} />
                          </Form.Item>
                          <Space>
                            <Button onClick={() => setBindStep(0)}>{t('common.back')}</Button>
                            <Button type="primary" htmlType="submit" loading={binding} disabled={googleAuthPerm.disabled}>
                              {t('profile.bind')}
                            </Button>
                          </Space>
                        </Form>
                      )}
                    </Space>
                  )}
                </div>
              ),
            },
            {
              key: 'sessions',
              label: t('profile.sessions'),
              children: <ProfileSessionsPanel />,
            },
          ].filter((item) => item.key !== '2fa' || googleAuthPerm.show))}
        />
      </Card>

      <Modal
        open={avatarOpen}
        title={t('profile.change_avatar')}
        onCancel={() => setAvatarOpen(false)}
        onOk={() => void handleSaveAvatar()}
        confirmLoading={savingAvatar}
        okButtonProps={{ disabled: !selectedAvatar }}
        width={560}
        destroyOnHidden
      >
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(6, 1fr)',
            gap: 12,
            maxHeight: 400,
            overflowY: 'auto',
            padding: '8px 0',
          }}
        >
          {DEFAULT_AVATARS.map((avatar) => {
            const active = selectedAvatar === avatar
            return (
              <button
                key={avatar}
                type="button"
                onClick={() => setSelectedAvatar(avatar)}
                style={{
                  padding: 8,
                  borderRadius: 8,
                  border: `2px solid ${active ? 'var(--ant-color-primary)' : 'var(--ant-color-border)'}`,
                  background: active ? 'var(--ant-color-primary-bg)' : 'transparent',
                  cursor: 'pointer',
                }}
              >
                <Avatar size={56} src={avatar} icon={<UserOutlined />} />
              </button>
            )
          })}
        </div>
      </Modal>
    </PageContainer>
  )
}
