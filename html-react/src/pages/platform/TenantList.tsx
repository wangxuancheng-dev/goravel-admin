import { useEffect, useMemo, useState } from 'react'
import {
  Alert,
  App,
  Button,
  Descriptions,
  Divider,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { SettingOutlined } from '@ant-design/icons'
import {
  backupPlatformTenant,
  createPlatformTenant,
  downloadPlatformTenantBackup,
  getPlatformTenantList,
  getPlatformTenantOpsSummary,
  listPlatformTenantBackups,
  migratePlatformTenant,
  migratePlatformTenantBatch,
  pingPlatformTenant,
  platformHealth,
  seedPlatformTenant,
  updatePlatformTenant,
  updatePlatformTenantStatus,
} from '@/api/platform'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { useColumnSetting } from '@/hooks/useColumnSetting'
import PageContainer from '@/components/PageContainer'
import ColumnSettingDialog from '@/components/ColumnSettingDialog'
import SearchForm from '@/components/SearchForm'
import { entityField } from '@/utils/normalize'

interface TenantRow {
  id: number | string
  code?: string
  name?: string
  driver?: string
  isolation?: string
  host?: string
  port?: number
  database?: string
  schema?: string
  username?: string
  has_password?: boolean
  status?: number
  provision_status?: string
  last_op?: string
  last_op_status?: string
  last_op_message?: string
  last_migrate_error?: string
  last_backup_path?: string
  backup_dir?: string
  last_op_at?: string
  connection_name?: string
  migrated_at?: string
  created_at?: string
}

function isTenantBusy(row: TenantRow) {
  if (row.provision_status === 'migrating') return true
  return row.last_op_status === 'queued' || row.last_op_status === 'running'
}

function shortPath(path?: string) {
  const s = String(path || '')
  if (s.length <= 28) return s
  return `…${s.slice(-26)}`
}

function formatSize(n?: number) {
  const size = Number(n) || 0
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

function backupDirOf(row: TenantRow) {
  if (row.backup_dir) return row.backup_dir
  if (row.code) return `storage/backups/tenants/${row.code}`
  return ''
}

export default function PlatformTenantList() {
  const { t } = useTranslation()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const [createOpen, setCreateOpen] = useState(false)
  const [editing, setEditing] = useState<TenantRow | null>(null)
  const [migrateTarget, setMigrateTarget] = useState<TenantRow | null>(null)
  const [withSeed, setWithSeed] = useState(true)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()
  const [healthDesc, setHealthDesc] = useState(t('platform.cli_ops_hint'))
  const [opsSummary, setOpsSummary] = useState<{
    failed_provision?: number
    busy?: number
    total?: number
  } | null>(null)
  const [batchLoading, setBatchLoading] = useState(false)
  const [detailRow, setDetailRow] = useState<TenantRow | null>(null)
  const [backupsRow, setBackupsRow] = useState<TenantRow | null>(null)
  const [backupsLoading, setBackupsLoading] = useState(false)
  const [backupsList, setBackupsList] = useState<
    Array<{ name: string; path: string; size: number; mod_time: string }>
  >([])
  const [backupsDir, setBackupsDir] = useState('')
  const [downloadingName, setDownloadingName] = useState('')

  const refreshOpsSummary = async () => {
    try {
      const res = await getPlatformTenantOpsSummary()
      const summary = (res as { data?: { summary?: Record<string, unknown> } })?.data?.summary
      if (summary) {
        setOpsSummary({
          failed_provision: Number(summary.failed_provision ?? 0),
          busy: Number(summary.busy ?? 0),
          total: Number(summary.total ?? 0),
        })
      }
    } catch {
      /* optional */
    }
  }

  useEffect(() => {
    void platformHealth()
      .then((res) => {
        const h = (res as { data?: Record<string, unknown> })?.data
        if (!h) return
        const tenants = (h.tenants || {}) as { active?: number; total?: number }
        const db = h.database_ok ? t('platform.health_db_ok') : t('platform.health_db_bad')
        setHealthDesc(
          `${t('platform.health_driver')}: ${String(h.driver)} · ${db} · ${t('platform.health_tenants', {
            active: tenants.active ?? 0,
            total: tenants.total ?? 0,
          })} · ${t('platform.cli_ops_hint')}`,
        )
      })
      .catch(() => {
        /* list still usable */
      })
    void refreshOpsSummary()
  }, [t])

  const {
    tableData,
    loading,
    pagination,
    searchForm,
    onSearchFormChange,
    loadData,
    handleSearch,
    handleReset,
    handleSortChange,
    refresh,
  } = useListPage<TenantRow>({
    fetchApi: getPlatformTenantList,
    initialSearchForm: { code: '', name: '', status: '', provision_status: '' },
    defaultSort: 'id:desc',
    normalizeRows: false,
    transformData: (row) => {
      const record = row
      return {
        id: entityField(record, 'id', '')!,
        code: String(entityField(record, 'code', '') ?? ''),
        name: String(entityField(record, 'name', '') ?? ''),
        driver: String(entityField(record, 'driver', '') ?? ''),
        isolation: String(entityField(record, 'isolation', '') ?? ''),
        host: String(entityField(record, 'host', '') ?? ''),
        port: Number(entityField(record, 'port', 0) ?? 0),
        database: String(entityField(record, 'database', '') ?? ''),
        schema: String(entityField(record, 'schema', '') ?? ''),
        username: String(entityField(record, 'username', '') ?? ''),
        has_password: Boolean(entityField(record, 'has_password', false)),
        status: Number(entityField(record, 'status', 0) ?? 0),
        provision_status: String(entityField(record, 'provision_status', '') ?? ''),
        last_op: String(entityField(record, 'last_op', '') ?? ''),
        last_op_status: String(entityField(record, 'last_op_status', '') ?? ''),
        last_op_message: String(entityField(record, 'last_op_message', '') ?? ''),
        last_migrate_error: String(entityField(record, 'last_migrate_error', '') ?? ''),
        last_backup_path: String(entityField(record, 'last_backup_path', '') ?? ''),
        backup_dir: String(entityField(record, 'backup_dir', '') ?? ''),
        last_op_at: String(entityField(record, 'last_op_at', '') ?? ''),
        connection_name: String(entityField(record, 'connection_name', '') ?? ''),
        migrated_at: String(entityField(record, 'migrated_at', '') ?? ''),
        created_at: String(entityField(record, 'created_at', '') ?? ''),
      }
    },
  })

  const copyText = async (text?: string) => {
    const value = String(text || '')
    if (!value) return
    try {
      await navigator.clipboard.writeText(value)
      message.success(t('tenant.backup_copied'))
    } catch {
      message.error(t('common.operation_failed'))
    }
  }

  const openBackups = async (row: TenantRow) => {
    setBackupsRow(row)
    setBackupsList([])
    setBackupsDir(row.backup_dir || backupDirOf(row))
    setBackupsLoading(true)
    try {
      const res = await listPlatformTenantBackups(row.id)
      const data = (res as { data?: { list?: typeof backupsList; backup_dir?: string } })?.data
      setBackupsList(data?.list || [])
      setBackupsDir(data?.backup_dir || row.backup_dir || backupDirOf(row))
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setBackupsLoading(false)
    }
  }

  const downloadBackup = async (file: { name: string }) => {
    if (!backupsRow) return
    setDownloadingName(file.name)
    try {
      const blob = (await downloadPlatformTenantBackup(backupsRow.id, file.name)) as Blob
      const url = window.URL.createObjectURL(blob instanceof Blob ? blob : new Blob([blob as unknown as BlobPart]))
      const a = document.createElement('a')
      a.href = url
      a.download = file.name
      a.click()
      window.URL.revokeObjectURL(url)
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setDownloadingName('')
    }
  }

  const hasBusy = tableData.some(isTenantBusy)
  useEffect(() => {
    if (!hasBusy) return
    const timer = window.setInterval(() => {
      void refresh()
      void refreshOpsSummary()
    }, 3000)
    return () => window.clearInterval(timer)
  }, [hasBusy, refresh])

  const retryFailedMigrates = () => {
    modal.confirm({
      title: t('tenant.retry_failed_migrate'),
      content: t('tenant.retry_failed_migrate_confirm'),
      onOk: async () => {
        setBatchLoading(true)
        try {
          const res = await migratePlatformTenantBatch({ provision_status: 'failed', with_seed: false })
          const n = Number((res as { data?: { queued_count?: number } })?.data?.queued_count ?? 0)
          message.success(t('tenant.batch_queued', { n }))
          await refresh()
          await refreshOpsSummary()
        } catch (error) {
          showError(error, t('common.operation_failed'))
        } finally {
          setBatchLoading(false)
        }
      },
    })
  }

  const provisionLabel = (status?: string) => {
    switch (status) {
      case 'ready':
        return t('tenant.provision_ready')
      case 'migrating':
        return t('tenant.provision_migrating')
      case 'failed':
        return t('tenant.provision_failed')
      case 'pending':
      default:
        return t('tenant.provision_pending')
    }
  }

  const provisionColor = (status?: string) => {
    switch (status) {
      case 'ready':
        return 'success'
      case 'migrating':
        return 'processing'
      case 'failed':
        return 'error'
      default:
        return 'default'
    }
  }

  const runQueued = async (fn: () => Promise<unknown>) => {
    try {
      await fn()
      message.success(t('tenant.op_queued'))
      await refresh()
    } catch (error) {
      showError(error, t('common.operation_failed'))
    }
  }

  const columns = useMemo<ColumnsType<TenantRow>>(
    () => [
      { title: t('table.id'), dataIndex: 'id', key: 'id', width: 70, sorter: true },
      { title: t('tenant.code'), dataIndex: 'code', key: 'code', width: 110 },
      { title: t('tenant.name'), dataIndex: 'name', key: 'name', width: 120 },
      { title: t('tenant.driver'), dataIndex: 'driver', key: 'driver', width: 90 },
      { title: t('tenant.database'), dataIndex: 'database', key: 'database', width: 140 },
      {
        title: t('tenant.provision_status'),
        dataIndex: 'provision_status',
        key: 'provision_status',
        width: 110,
        render: (status: string, row) => (
          <Tooltip title={row.last_migrate_error || row.last_op_message || undefined}>
            <Tag color={provisionColor(status)}>{provisionLabel(status)}</Tag>
          </Tooltip>
        ),
      },
      {
        title: t('tenant.last_op'),
        key: 'last_op',
        width: 140,
        render: (_, row) => {
          if (!row.last_op && !row.last_op_status) return '—'
          return (
            <Tooltip title={row.last_op_message || row.last_op_at || undefined}>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {row.last_op || '—'} / {row.last_op_status || '—'}
              </Typography.Text>
            </Tooltip>
          )
        },
      },
      {
        title: t('tenant.backup_path'),
        key: 'last_backup_path',
        width: 200,
        render: (_, row) =>
          row.last_backup_path ? (
            <Space size={4}>
              <Tooltip title={row.last_backup_path}>
                <Typography.Text style={{ fontSize: 12 }}>{shortPath(row.last_backup_path)}</Typography.Text>
              </Tooltip>
              <Button type="link" size="small" onClick={() => void copyText(row.last_backup_path)}>
                {t('tenant.backup_copy_path')}
              </Button>
            </Space>
          ) : (
            '—'
          ),
      },
      {
        title: t('common.status'),
        dataIndex: 'status',
        key: 'status',
        width: 90,
        render: (status, row) => (
          <Switch
            checked={Number(status) === 1}
            onChange={async (checked) => {
              try {
                await updatePlatformTenantStatus(row.id, checked ? 1 : 0)
                message.success(t('common.update_success'))
                await refresh()
              } catch (error) {
                showError(error, t('common.operation_failed'))
              }
            }}
          />
        ),
      },
      { title: t('table.created_at'), dataIndex: 'created_at', key: 'created_at', width: 170 },
      {
        title: t('common.operation'),
        key: 'actions',
        width: 420,
        fixed: 'right',
        render: (_, row) => {
          const busy = isTenantBusy(row)
          return (
            <Space size={0} wrap>
              <Button type="link" size="small" onClick={() => setDetailRow(row)}>
                {t('tenant.op_detail')}
              </Button>
              <Button
                type="link"
                size="small"
                disabled={busy}
                onClick={() => {
                  setEditing(row)
                  form.setFieldsValue({
                    name: row.name,
                    host: row.host,
                    port: row.port || 0,
                    username: row.username,
                    password: '',
                    database: row.database,
                    schema: row.schema,
                  })
                }}
              >
                {t('common.edit')}
              </Button>
              <Button
                type="link"
                size="small"
                onClick={async () => {
                  try {
                    await pingPlatformTenant(row.id)
                    message.success(t('tenant.ping_ok'))
                  } catch (error) {
                    showError(error, t('common.operation_failed'))
                  }
                }}
              >
                {t('tenant.op_ping')}
              </Button>
              <Button
                type="link"
                size="small"
                disabled={busy}
                onClick={() => {
                  setWithSeed(true)
                  setMigrateTarget(row)
                }}
              >
                {t('tenant.op_migrate')}
              </Button>
              <Button
                type="link"
                size="small"
                disabled={busy}
                onClick={() => {
                  modal.confirm({
                    title: t('tenant.op_seed_confirm'),
                    onOk: () => runQueued(() => seedPlatformTenant(row.id)),
                  })
                }}
              >
                {t('tenant.op_seed')}
              </Button>
              <Button
                type="link"
                size="small"
                disabled={busy}
                onClick={() => {
                  modal.confirm({
                    title: t('tenant.op_backup_confirm'),
                    onOk: () => runQueued(() => backupPlatformTenant(row.id)),
                  })
                }}
              >
                {t('tenant.op_backup')}
              </Button>
              <Button type="link" size="small" onClick={() => void openBackups(row)}>
                {t('tenant.op_backups')}
              </Button>
            </Space>
          )
        },
      },
    ],
    [t, message, modal, refresh, showError, form, copyText, openBackups, runQueued],
  )

  const {
    filteredColumns,
    open: columnSettingOpen,
    openColumnSetting,
    closeColumnSetting,
    allColumns,
    visibleColumns,
    columnOrder,
    fixedColumns,
    handleConfirm: handleColumnSettingConfirm,
  } = useColumnSetting('platform-tenant', columns)

  const submitCreate = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      await createPlatformTenant(values)
      message.success(t('common.create_success'))
      setCreateOpen(false)
      await refresh()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSaving(false)
    }
  }

  const submitEdit = async () => {
    if (!editing) return
    try {
      const values = await form.validateFields()
      setSaving(true)
      const payload: Record<string, unknown> = {
        name: values.name,
        host: values.host,
        port: values.port || 0,
        username: values.username,
        database: values.database,
        schema: values.schema,
      }
      if (values.password) payload.password = values.password
      await updatePlatformTenant(editing.id, payload)
      message.success(t('common.update_success'))
      setEditing(null)
      await refresh()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <PageContainer
      title={t('menu.tenant')}
      extra={
        <Space wrap>
          <Button
            loading={batchLoading}
            disabled={(opsSummary?.failed_provision ?? 0) < 1}
            onClick={retryFailedMigrates}
          >
            {t('tenant.retry_failed_migrate')}
          </Button>
          <Button
            type="primary"
            onClick={() => {
              form.resetFields()
              form.setFieldsValue({
                driver: 'mysql',
                isolation: 'database',
                skip_create: false,
                port: 0,
              })
              setCreateOpen(true)
            }}
          >
            {t('tenant.add')}
          </Button>
          <Button icon={<SettingOutlined />} onClick={openColumnSetting}>
            {t('common.column_setting')}
          </Button>
        </Space>
      }
    >
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        message={t('platform.cli_ops_title')}
        description={
          <>
            <div>{healthDesc}</div>
            {opsSummary ? (
              <div style={{ marginTop: 6 }}>
                {t('tenant.ops_summary', {
                  failed: opsSummary.failed_provision ?? 0,
                  busy: opsSummary.busy ?? 0,
                  total: opsSummary.total ?? 0,
                })}
              </div>
            ) : null}
          </>
        }
      />
      <SearchForm
        fields={[
          { name: 'code', label: t('tenant.code') },
          { name: 'name', label: t('tenant.name') },
          {
            name: 'provision_status',
            label: t('tenant.provision_status_filter'),
            type: 'select',
            options: [
              { label: t('tenant.provision_pending'), value: 'pending' },
              { label: t('tenant.provision_migrating'), value: 'migrating' },
              { label: t('tenant.provision_ready'), value: 'ready' },
              { label: t('tenant.provision_failed'), value: 'failed' },
            ],
          },
          {
            name: 'status',
            label: t('common.status'),
            type: 'select',
            options: [
              { label: t('common.enabled'), value: '1' },
              { label: t('common.disabled'), value: '0' },
            ],
          },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />

      <Table
        rowKey="id"
        loading={loading}
        dataSource={tableData}
        columns={filteredColumns}
        scroll={{ x: 1500 }}
        pagination={{
          current: pagination.page,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
        }}
        onChange={(pager, _f, sorter) =>
          handlePaginatedTableChange({ pager, sorter, pagination, loadData, handleSortChange })
        }
      />

      <Drawer
        title={t('tenant.detail_title')}
        open={!!detailRow}
        onClose={() => setDetailRow(null)}
        width={440}
        destroyOnHidden
      >
        {detailRow ? (
          <>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label={t('tenant.code')}>{detailRow.code}</Descriptions.Item>
              <Descriptions.Item label={t('tenant.name')}>{detailRow.name}</Descriptions.Item>
              <Descriptions.Item label={t('tenant.driver')}>
                {detailRow.driver} / {detailRow.isolation}
              </Descriptions.Item>
              <Descriptions.Item label={t('tenant.database')}>{detailRow.database}</Descriptions.Item>
              <Descriptions.Item label={t('tenant.schema')}>{detailRow.schema || '—'}</Descriptions.Item>
              <Descriptions.Item label={t('tenant.host')}>
                {detailRow.host || '—'}:{detailRow.port || '—'}
              </Descriptions.Item>
              <Descriptions.Item label={t('tenant.connection_name')}>
                {detailRow.connection_name || '—'}
              </Descriptions.Item>
              <Descriptions.Item label={t('tenant.provision_status')}>
                {provisionLabel(detailRow.provision_status)}
              </Descriptions.Item>
              <Descriptions.Item label={t('tenant.last_op')}>
                {detailRow.last_op || '—'} / {detailRow.last_op_status || '—'}
              </Descriptions.Item>
              <Descriptions.Item label={t('tenant.last_op_at')}>{detailRow.last_op_at || '—'}</Descriptions.Item>
              <Descriptions.Item label={t('tenant.op_message')}>
                {detailRow.last_op_message || detailRow.last_migrate_error || '—'}
              </Descriptions.Item>
              <Descriptions.Item label={t('tenant.migrated_at')}>{detailRow.migrated_at || '—'}</Descriptions.Item>
              <Descriptions.Item label={t('tenant.backup_dir')}>
                <Space>
                  <span>{detailRow.backup_dir || backupDirOf(detailRow)}</span>
                  <Button
                    type="link"
                    size="small"
                    onClick={() => void copyText(detailRow.backup_dir || backupDirOf(detailRow))}
                  >
                    {t('tenant.backup_copy_path')}
                  </Button>
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label={t('tenant.backup_path')}>
                {detailRow.last_backup_path ? (
                  <Space>
                    <span>{detailRow.last_backup_path}</span>
                    <Button type="link" size="small" onClick={() => void copyText(detailRow.last_backup_path)}>
                      {t('tenant.backup_copy_path')}
                    </Button>
                  </Space>
                ) : (
                  '—'
                )}
              </Descriptions.Item>
            </Descriptions>
            <Button type="primary" style={{ marginTop: 16 }} onClick={() => void openBackups(detailRow)}>
              {t('tenant.op_backups')}
            </Button>
          </>
        ) : null}
      </Drawer>

      <Modal
        title={t('tenant.backup_list_title')}
        open={!!backupsRow}
        onCancel={() => setBackupsRow(null)}
        footer={null}
        width={720}
        destroyOnHidden
      >
        <Alert type="info" showIcon style={{ marginBottom: 12 }} message={t('tenant.backup_hint')} />
        {backupsDir ? (
          <Space style={{ marginBottom: 12 }}>
            <Typography.Text type="secondary">
              {t('tenant.backup_dir')}: {backupsDir}
            </Typography.Text>
            <Button type="link" size="small" onClick={() => void copyText(backupsDir)}>
              {t('tenant.backup_copy_path')}
            </Button>
          </Space>
        ) : null}
        <Table
          rowKey="name"
          size="small"
          loading={backupsLoading}
          dataSource={backupsList}
          pagination={false}
          locale={{ emptyText: t('tenant.backup_empty') }}
          columns={[
            { title: t('tenant.backup_name'), dataIndex: 'name', key: 'name' },
            {
              title: t('tenant.backup_size'),
              dataIndex: 'size',
              key: 'size',
              width: 100,
              render: (size: number) => formatSize(size),
            },
            { title: t('tenant.backup_time'), dataIndex: 'mod_time', key: 'mod_time', width: 170 },
            {
              title: t('common.operation'),
              key: 'actions',
              width: 180,
              render: (_, file) => (
                <Space size={0}>
                  <Button type="link" size="small" onClick={() => void copyText(file.path)}>
                    {t('tenant.backup_copy_path')}
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    loading={downloadingName === file.name}
                    onClick={() => void downloadBackup(file)}
                  >
                    {t('tenant.backup_download')}
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </Modal>

      <ColumnSettingDialog
        open={columnSettingOpen}
        onClose={closeColumnSetting}
        allColumns={allColumns}
        visibleColumns={visibleColumns}
        columnOrder={columnOrder}
        fixedColumns={fixedColumns}
        onConfirm={handleColumnSettingConfirm}
      />

      <Modal
        title={t('tenant.op_migrate')}
        open={!!migrateTarget}
        onCancel={() => setMigrateTarget(null)}
        onOk={() => {
          if (!migrateTarget) return
          void (async () => {
            try {
              await migratePlatformTenant(migrateTarget.id, { with_seed: withSeed })
              message.success(t('tenant.op_queued'))
              setMigrateTarget(null)
              await refresh()
            } catch (error) {
              showError(error, t('common.operation_failed'))
            }
          })()
        }}
        destroyOnHidden
      >
        <p>{t('tenant.op_migrate_confirm')}</p>
        <div>
          <Switch checked={withSeed} onChange={setWithSeed} />{' '}
          <span style={{ marginLeft: 8 }}>{t('tenant.op_with_seed')}</span>
        </div>
        {migrateTarget?.last_migrate_error ? (
          <Alert
            style={{ marginTop: 12 }}
            type="error"
            showIcon
            message={t('tenant.migrate_error')}
            description={migrateTarget.last_migrate_error}
          />
        ) : null}
      </Modal>

      <Modal
        title={t('tenant.add')}
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => void submitCreate()}
        confirmLoading={saving}
        destroyOnHidden
        width={640}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="code"
            label={t('tenant.code')}
            rules={[{ required: true, message: t('tenant.code_required') }]}
          >
            <Input placeholder={t('tenant.code_placeholder')} />
          </Form.Item>
          <Form.Item
            name="name"
            label={t('tenant.name')}
            rules={[{ required: true, message: t('tenant.name_required') }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="driver" label={t('tenant.driver')}>
            <Select
              options={[
                { value: 'mysql', label: 'mysql' },
                { value: 'postgres', label: 'postgres' },
              ]}
            />
          </Form.Item>
          <Form.Item name="isolation" label={t('tenant.isolation')}>
            <Select
              options={[
                { value: 'database', label: 'database' },
                { value: 'schema', label: 'schema' },
              ]}
            />
          </Form.Item>
          <Form.Item name="database" label={t('tenant.database')}>
            <Input placeholder={t('tenant.database_placeholder')} />
          </Form.Item>
          <Form.Item name="schema" label={t('tenant.schema')}>
            <Input placeholder={t('tenant.schema_placeholder')} />
          </Form.Item>
          <Divider>{t('tenant.remote_connection')}</Divider>
          <Form.Item name="host" label={t('tenant.host')}>
            <Input placeholder={t('tenant.host_placeholder')} />
          </Form.Item>
          <Form.Item name="port" label={t('tenant.port')}>
            <InputNumber min={0} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="username" label={t('tenant.username')}>
            <Input placeholder={t('tenant.username_placeholder')} />
          </Form.Item>
          <Form.Item name="password" label={t('tenant.password')}>
            <Input.Password placeholder={t('tenant.password_placeholder')} />
          </Form.Item>
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            message={t('tenant.migrate_cli_tip')}
          />
          <Form.Item
            name="skip_create"
            label={t('tenant.skip_create')}
            valuePropName="checked"
            extra={t('tenant.skip_create_tip')}
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('tenant.edit')}
        open={!!editing}
        onCancel={() => setEditing(null)}
        onOk={() => void submitEdit()}
        confirmLoading={saving}
        destroyOnHidden
        width={640}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label={t('tenant.name')}
            rules={[{ required: true, message: t('tenant.name_required') }]}
          >
            <Input />
          </Form.Item>
          <Divider>{t('tenant.remote_connection')}</Divider>
          <Form.Item name="host" label={t('tenant.host')}>
            <Input placeholder={t('tenant.host_placeholder')} />
          </Form.Item>
          <Form.Item name="port" label={t('tenant.port')}>
            <InputNumber min={0} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="username" label={t('tenant.username')}>
            <Input placeholder={t('tenant.username_placeholder')} />
          </Form.Item>
          <Form.Item name="password" label={t('tenant.password')}>
            <Input.Password
              placeholder={
                editing?.has_password ? t('tenant.password_keep') : t('tenant.password_placeholder')
              }
            />
          </Form.Item>
          <Form.Item name="database" label={t('tenant.database')}>
            <Input />
          </Form.Item>
          <Form.Item name="schema" label={t('tenant.schema')}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
