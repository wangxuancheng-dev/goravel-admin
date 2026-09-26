import { useEffect, useState } from 'react'
import { Alert, Tag } from 'antd'
import { useTranslation } from 'react-i18next'
import SimpleCrudPage from '@/components/SimpleCrudPage'
import {
  createAllowlist,
  deleteAllowlist,
  getAllowlistList,
  updateAllowlist,
} from '@/api/allowlist'

interface AllowlistRow {
  id: number | string
  ip?: string
  remark?: string
  status?: number
  created_at?: string
  name?: string
}

export default function AllowlistList() {
  const { t } = useTranslation()
  const [clientIp, setClientIp] = useState('')

  useEffect(() => {
    void getAllowlistList({ page: 1, page_size: 1 })
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
          message={t('allowlist.client_ip_tip', { ip: clientIp })}
        />
      ) : null}
      <SimpleCrudPage<AllowlistRow>
        title={t('menu.allowlist')}
        permissions={{
          store: 'allowlist.store',
          update: 'allowlist.update',
          destroy: 'allowlist.destroy',
        }}
        fetchApi={getAllowlistList}
        createApi={createAllowlist}
        updateApi={updateAllowlist}
        deleteApi={deleteAllowlist}
        requireSensitiveConfirmOnCreate
        requireSensitiveConfirmOnDelete
        createTitle={t('allowlist.add_allowlist')}
        editTitle={t('allowlist.edit_allowlist')}
        initialSearchForm={{ ip: '', status: '' }}
        searchFields={[
          { name: 'ip', label: t('allowlist.ip') },
          {
            name: 'status',
            label: t('common.status'),
            type: 'select',
            options: [
              { label: t('allowlist.enabled'), value: '1' },
              { label: t('allowlist.disabled'), value: '0' },
            ],
          },
        ]}
        formFields={[
          { name: 'ip', label: t('allowlist.ip'), type: 'textarea', required: true },
          { name: 'remark', label: t('allowlist.remark'), type: 'textarea' },
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
          { title: t('allowlist.ip'), dataIndex: 'ip', ellipsis: true },
          { title: t('allowlist.remark'), dataIndex: 'remark', ellipsis: true },
          {
            title: t('common.status'),
            dataIndex: 'status',
            width: 100,
            render: (v: number) => (
              <Tag color={v === 1 ? 'success' : 'default'}>
                {v === 1 ? t('allowlist.enabled') : t('allowlist.disabled')}
              </Tag>
            ),
          },
          { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
        ]}
      />
    </>
  )
}
