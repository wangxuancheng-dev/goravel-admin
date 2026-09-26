import { useCallback, useEffect, useState } from 'react'
import { App, Button, Space, Table, Tag, Typography } from 'antd'
import { useTranslation } from 'react-i18next'
import { getAuthTokens, revokeAuthToken, revokeOtherAuthTokens } from '@/api/auth'
import { useUnhandledError } from '@/hooks/useUnhandledError'

type SessionRow = {
  id: number
  name?: string
  browser?: string
  ip?: string
  os?: string
  last_used_at?: string
  created_at?: string
  is_current?: boolean
}

export default function ProfileSessionsPanel() {
  const { t } = useTranslation()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const [loading, setLoading] = useState(false)
  const [rows, setRows] = useState<SessionRow[]>([])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getAuthTokens()
      const list = (res.data?.tokens || []) as SessionRow[]
      setRows(list.map((item) => ({ ...item, id: Number(item.id) })))
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }, [showError, t])

  useEffect(() => {
    void load()
  }, [load])

  const onRevoke = (row: SessionRow) => {
    modal.confirm({
      title: t('profile.session_revoke_confirm'),
      onOk: async () => {
        try {
          await revokeAuthToken(row.id)
          message.success(t('common.operation_success'))
          await load()
        } catch (error) {
          showError(error, t('common.operation_failed'))
        }
      },
    })
  }

  const onRevokeOthers = () => {
    modal.confirm({
      title: t('profile.session_revoke_others_confirm'),
      onOk: async () => {
        try {
          await revokeOtherAuthTokens()
          message.success(t('common.operation_success'))
          await load()
        } catch (error) {
          showError(error, t('common.operation_failed'))
        }
      },
    })
  }

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button onClick={() => void load()}>{t('common.refresh')}</Button>
        <Button danger onClick={onRevokeOthers}>
          {t('profile.session_revoke_others')}
        </Button>
      </Space>
      <Typography.Paragraph type="secondary">{t('profile.session_hint')}</Typography.Paragraph>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={rows}
        pagination={false}
        columns={[
          {
            title: t('profile.session_device'),
            render: (_, row) => (
              <span>
                {[row.browser, row.os].filter(Boolean).join(' / ') || row.name || '-'}
                {row.is_current ? (
                  <Tag color="blue" style={{ marginLeft: 8 }}>
                    {t('profile.session_current')}
                  </Tag>
                ) : null}
              </span>
            ),
          },
          { title: 'IP', dataIndex: 'ip', width: 140 },
          { title: t('profile.session_last_used'), dataIndex: 'last_used_at', width: 180 },
          { title: t('table.created_at'), dataIndex: 'created_at', width: 180 },
          {
            title: t('table.operation'),
            width: 120,
            render: (_, row) =>
              row.is_current ? (
                '-'
              ) : (
                <Button type="link" danger onClick={() => onRevoke(row)}>
                  {t('profile.session_revoke')}
                </Button>
              ),
          },
        ]}
      />
    </div>
  )
}
