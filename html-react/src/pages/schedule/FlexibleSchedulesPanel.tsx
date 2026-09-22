import { useEffect, useState } from 'react'
import {
  App,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  theme,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { PlusOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  createFlexibleSchedule,
  deleteFlexibleSchedule,
  getFlexibleScheduleHandlers,
  getFlexibleScheduleList,
  previewFlexibleSchedule,
  runFlexibleSchedule,
  updateFlexibleSchedule,
  type FlexibleHandlerMeta,
  type FlexibleScheduleRow,
} from '@/api/schedule'
import PermissionButton from '@/components/PermissionButton'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { usePermission } from '@/hooks/usePermission'
import { describeCron } from '@/utils/cronLabel'
import { formatDateTimeDisplay } from '@/utils/dateUtils'
import { logger } from '@/utils/logger'

function formatDuration(ms?: number) {
  if (ms === undefined || ms === null) return '-'
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(2)} s`
}

export default function FlexibleSchedulesPanel() {
  const { t } = useTranslation()
  const { modal, message } = App.useApp()
  const { token } = theme.useToken()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const canUpdate = !getButtonState('flexible_schedule.update').disabled
  const [loading, setLoading] = useState(false)
  const [rows, setRows] = useState<FlexibleScheduleRow[]>([])
  const [handlers, setHandlers] = useState<FlexibleHandlerMeta[]>([])
  const [runningId, setRunningId] = useState<number | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<FlexibleScheduleRow | null>(null)
  const [saving, setSaving] = useState(false)
  const [previewRuns, setPreviewRuns] = useState<string[]>([])
  const [form] = Form.useForm()

  const mutedTextStyle = {
    color: token.colorTextSecondary,
    fontSize: 12,
    lineHeight: 1.4,
  } as const
  const hintTextStyle = {
    marginTop: 6,
    fontSize: 13,
    fontWeight: 400 as const,
    color: token.colorTextSecondary,
  }

  const statusLabel = (status?: string) => {
    if (status === 'success') return t('schedule.status_success')
    if (status === 'failed') return t('schedule.status_failed')
    if (status === 'skipped') return t('schedule.flexible_status_skipped')
    return t('schedule.status_never')
  }

  const statusColor = (status?: string) => {
    if (status === 'success') return 'success'
    if (status === 'failed') return 'error'
    if (status === 'skipped') return 'warning'
    return 'default'
  }

  const loadData = async () => {
    setLoading(true)
    try {
      const [listRes, handlerRes] = await Promise.all([
        getFlexibleScheduleList(),
        getFlexibleScheduleHandlers(),
      ])
      setRows(listRes.data?.list || [])
      setHandlers(handlerRes.data?.list || [])
    } catch (error) {
      logger.error('Failed to load flexible schedules:', error)
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({
      name: '',
      handler: handlers[0]?.key || 'schedule_test_log',
      cron_expr: '*/5 * * * *',
      timezone: 'UTC',
      tenant_id: 0,
      enabled: true,
    })
    setPreviewRuns([])
    setFormOpen(true)
  }

  const openEdit = (row: FlexibleScheduleRow) => {
    setEditing(row)
    form.setFieldsValue({
      name: row.name,
      handler: row.handler,
      cron_expr: row.cron_expr,
      timezone: row.timezone || 'UTC',
      tenant_id: row.tenant_id || 0,
      enabled: row.enabled,
    })
    setPreviewRuns(row.next_runs || [])
    setFormOpen(true)
  }

  const onPreview = async () => {
    try {
      const values = await form.validateFields(['cron_expr', 'timezone'])
      const res = await previewFlexibleSchedule({
        cron_expr: values.cron_expr,
        timezone: values.timezone || 'UTC',
        count: 5,
      })
      setPreviewRuns(res.data?.next_runs || [])
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) return
      showError(error, t('common.operation_failed'))
    }
  }

  const onSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      const payload = {
        name: values.name,
        handler: values.handler,
        cron_expr: values.cron_expr,
        timezone: values.timezone || 'UTC',
        tenant_id: Number(values.tenant_id) || 0,
        enabled: !!values.enabled,
      }
      if (editing) {
        await updateFlexibleSchedule(editing.id, payload)
        message.success(t('common.update_success'))
      } else {
        await createFlexibleSchedule(payload)
        message.success(t('common.create_success'))
      }
      setFormOpen(false)
      await loadData()
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSaving(false)
    }
  }

  const onToggle = async (row: FlexibleScheduleRow, enabled: boolean) => {
    try {
      await updateFlexibleSchedule(row.id, { enabled })
      message.success(t('common.update_success'))
      await loadData()
    } catch (error) {
      showError(error, t('common.operation_failed'))
    }
  }

  const onRun = (row: FlexibleScheduleRow) => {
    modal.confirm({
      title: t('schedule.run_now'),
      content: t('schedule.flexible_run_confirm', { name: row.name || row.handler }),
      onOk: async () => {
        setRunningId(row.id)
        try {
          const res = await runFlexibleSchedule(row.id, !row.enabled)
          const updated = res.data?.flexible_schedule
          if (updated?.last_status === 'failed') {
            message.warning(t('schedule.run_failed'))
          } else if (updated?.last_status === 'skipped') {
            message.warning(t('schedule.flexible_status_skipped'))
          } else {
            message.success(t('schedule.run_success'))
          }
          await loadData()
        } catch (error) {
          showError(error, t('common.operation_failed'))
          throw error
        } finally {
          setRunningId(null)
        }
      },
    })
  }

  const onDelete = (row: FlexibleScheduleRow) => {
    modal.confirm({
      title: t('common.delete'),
      content: t('schedule.flexible_delete_confirm', { name: row.name || row.handler }),
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteFlexibleSchedule(row.id)
          message.success(t('common.delete_success'))
          await loadData()
        } catch (error) {
          showError(error, t('common.operation_failed'))
          throw error
        }
      },
    })
  }

  const columns: ColumnsType<FlexibleScheduleRow> = [
    {
      title: t('schedule.flexible_name'),
      dataIndex: 'name',
      key: 'name',
      minWidth: 140,
    },
    {
      title: t('schedule.flexible_handler'),
      key: 'handler',
      minWidth: 160,
      render: (_, row) => (
        <div>
          <div>{row.handler_name || row.handler}</div>
          <code style={{ fontSize: 12, color: token.colorTextSecondary }}>{row.handler}</code>
        </div>
      ),
    },
    {
      title: t('schedule.cron'),
      dataIndex: 'cron_expr',
      key: 'cron_expr',
      minWidth: 200,
      render: (value?: string) => (
        <div>
          <code style={{ fontFamily: "Consolas, 'Courier New', monospace", fontSize: 12 }}>
            {value || '-'}
          </code>
          {value ? (
            <div style={{ marginTop: 4, ...mutedTextStyle }}>
              {describeCron(value, t)}
            </div>
          ) : null}
        </div>
      ),
    },
    {
      title: t('schedule.flexible_timezone'),
      dataIndex: 'timezone',
      key: 'timezone',
      width: 130,
    },
    {
      title: t('schedule.flexible_tenant'),
      key: 'tenant',
      width: 120,
      render: (_, row) =>
        row.tenant_id > 0 ? row.tenant_code || String(row.tenant_id) : t('schedule.flexible_tenant_all'),
    },
    {
      title: t('common.status'),
      dataIndex: 'enabled',
      key: 'enabled',
      width: 100,
      align: 'center',
      render: (enabled: boolean, row) => (
        <Switch
          checked={enabled}
          onChange={(v) => void onToggle(row, v)}
          disabled={!canUpdate}
        />
      ),
    },
    {
      title: t('schedule.last_status'),
      dataIndex: 'last_status',
      key: 'last_status',
      width: 100,
      align: 'center',
      render: (value?: string) => <Tag color={statusColor(value)}>{statusLabel(value)}</Tag>,
    },
    {
      title: t('schedule.last_run_at'),
      dataIndex: 'last_run_at',
      key: 'last_run_at',
      minWidth: 160,
      render: (v?: string) => formatDateTimeDisplay(v),
    },
    {
      title: t('schedule.last_duration'),
      dataIndex: 'last_duration_ms',
      key: 'last_duration_ms',
      width: 100,
      align: 'right',
      render: (v?: number) => formatDuration(v),
    },
    {
      title: t('schedule.flexible_next_runs'),
      dataIndex: 'next_runs',
      key: 'next_runs',
      minWidth: 180,
      render: (runs?: string[]) =>
        runs && runs.length ? (
          <div style={{ fontSize: 12, lineHeight: 1.5 }}>
            {runs.slice(0, 2).map((r) => (
              <div key={r}>{formatDateTimeDisplay(r)}</div>
            ))}
          </div>
        ) : (
          '-'
        ),
    },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 220,
      fixed: 'right',
      render: (_, row) => (
        <Space size={0} wrap>
          <PermissionButton permission="flexible_schedule.update" type="link" onClick={() => openEdit(row)}>
            {t('common.edit')}
          </PermissionButton>
          <PermissionButton
            permission="flexible_schedule.run"
            type="link"
            loading={runningId === row.id}
            onClick={() => onRun(row)}
          >
            {t('schedule.run_now')}
          </PermissionButton>
          <PermissionButton
            permission="flexible_schedule.delete"
            type="link"
            danger
            onClick={() => onDelete(row)}
          >
            {t('common.delete')}
          </PermissionButton>
        </Space>
      ),
    },
  ]

  return (
    <>
      <Card
        style={{ marginTop: 16 }}
        title={
          <div>
            <div>{t('schedule.flexible_title')}</div>
            <div style={hintTextStyle}>
              {t('schedule.flexible_hint')}
            </div>
          </div>
        }
        extra={
          <Space>
            <Button onClick={() => void loadData()} loading={loading}>
              {t('common.refresh')}
            </Button>
            <PermissionButton permission="flexible_schedule.create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              {t('common.add')}
            </PermissionButton>
          </Space>
        }
      >
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1400 }}
          pagination={false}
        />
      </Card>

      <Modal
        title={editing ? t('common.edit') : t('common.add')}
        open={formOpen}
        onCancel={() => setFormOpen(false)}
        onOk={() => void onSave()}
        confirmLoading={saving}
        destroyOnClose
        width={640}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 12 }}>
          <Form.Item name="name" label={t('schedule.flexible_name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="handler" label={t('schedule.flexible_handler')} rules={[{ required: true }]}>
            <Select
              options={handlers.map((h) => ({
                value: h.key,
                label: `${h.name} (${h.key})`,
              }))}
            />
          </Form.Item>
          <Form.Item
            name="cron_expr"
            label={t('schedule.cron')}
            extra={t('schedule.flexible_cron_tip')}
            rules={[{ required: true }]}
          >
            <Input placeholder="0 20 * * *" />
          </Form.Item>
          <Form.Item name="timezone" label={t('schedule.flexible_timezone')} rules={[{ required: true }]}>
            <Input placeholder="UTC" />
          </Form.Item>
          <Form.Item
            name="tenant_id"
            label={t('schedule.flexible_tenant')}
            extra={t('schedule.flexible_tenant_tip')}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="enabled" label={t('common.status')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Space style={{ marginBottom: 12 }}>
            <Button onClick={() => void onPreview()}>{t('schedule.flexible_preview')}</Button>
          </Space>
          {previewRuns.length > 0 ? (
            <div style={{ marginBottom: 8, fontSize: 12, color: token.colorTextSecondary }}>
              <div>{t('schedule.flexible_next_runs')}:</div>
              {previewRuns.map((r) => (
                <div key={r}>{formatDateTimeDisplay(r)}</div>
              ))}
            </div>
          ) : null}
        </Form>
      </Modal>
    </>
  )
}
