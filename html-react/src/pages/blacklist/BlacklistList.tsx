import { useEffect, useState } from 'react'
import { Alert, Tag } from 'antd'
import { useTranslation } from 'react-i18next'
import SimpleCrudPage from '@/components/SimpleCrudPage'
import {
  createBlacklist,
  deleteBlacklist,
  getBlacklistList,
  updateBlacklist,
} from '@/api/blacklist'

interface BlacklistRow {
  id: number | string
  ip?: string
  remark?: string
  status?: number
  created_at?: string
  name?: string
}

function formatBlacklistIP(ip?: string) {
  if (!ip) return '-'
  if (ip.includes('-')) {
    const parts = ip.split('-')
    if (parts.length === 2) {
      return `${parts[0].trim()} ~ ${parts[1].trim()}`
    }
  }
  return ip
}

export default function BlacklistList() {
  const { t } = useTranslation()
  const [clientIp, setClientIp] = useState('')

  useEffect(() => {
    void getBlacklistList({ page: 1, page_size: 1 })
      .then((res) => {
        const ip = String((res.data as { client_ip?: string } | undefined)?.client_ip || '')
        if (ip) setClientIp(ip)
      })
      .catch(() => {})
  }, [])

  return (
    <>
      {clientIp ? (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message={t('blacklist.client_ip_tip', { ip: clientIp })}
        />
      ) : null}
      <SimpleCrudPage<BlacklistRow>
        title={t('menu.blacklist')}
        permissions={{
          store: 'blacklist.store',
          update: 'blacklist.update',
          destroy: 'blacklist.destroy',
        }}
        fetchApi={getBlacklistList}
        createApi={createBlacklist}
        updateApi={updateBlacklist}
        deleteApi={deleteBlacklist}
        requireSensitiveConfirmOnCreate
        requireSensitiveConfirmOnDelete
        createTitle={t('blacklist.add_blacklist')}
        editTitle={t('blacklist.edit_blacklist')}
        initialSearchForm={{ ip: '', status: '' }}
        searchFields={[
          { name: 'ip', label: t('blacklist.ip') },
          {
            name: 'status',
            label: t('common.status'),
            type: 'select',
            options: [
              { label: t('blacklist.enabled'), value: '1' },
              { label: t('blacklist.disabled'), value: '0' },
            ],
          },
        ]}
        formFields={[
          { name: 'ip', label: t('blacklist.ip'), type: 'textarea', required: true },
          { name: 'remark', label: t('blacklist.remark'), type: 'textarea' },
          { name: 'status', label: t('common.status'), type: 'status' },
        ]}
        transformRow={(row) => ({
          id: row.id as string | number,
          ip: String(row.ip ?? ''),
          remark: String(row.remark ?? ''),
          status: Number(row.status ?? 1),
          created_at: String(row.created_at ?? ''),
          name: String(row.ip ?? ''),
        })}
        columns={[
          { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
          {
            title: t('blacklist.ip'),
            dataIndex: 'ip',
            render: (ip: string) => <span style={{ wordBreak: 'break-all' }}>{formatBlacklistIP(ip)}</span>,
          },
          { title: t('blacklist.remark'), dataIndex: 'remark', ellipsis: true },
          {
            title: t('common.status'),
            dataIndex: 'status',
            width: 100,
            render: (status: number) =>
              status === 1 ? (
                <Tag color="error">{t('blacklist.enabled')}</Tag>
              ) : (
                <Tag>{t('blacklist.disabled')}</Tag>
              ),
          },
          { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
        ]}
      />
    </>
  )
}
