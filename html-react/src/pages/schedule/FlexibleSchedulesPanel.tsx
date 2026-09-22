import { useEffect, useState } from 'react'
import {
  App,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  Modal,
  Space,
  Switch,
  Table,
  Tag,
  theme,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import {
  getFlexibleScheduleList,
  previewFlexibleSchedule,
  runFlexibleSchedule,
  updateFlexibleSchedule,
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
  const [runningId, setRunningId] = useState<number | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<FlexibleScheduleRow | null>(null)
  const [saving, setSaving] = useState(false)
  const [previewRuns, setPreviewRuns] = useState<string[]>([])
  const [resultOpen, setResultOpen] = useState(false)
  const [resultRow, setResultRow] = useState<FlexibleScheduleRow | null>(null)
  const [form] = Form.useForm()

  const mutedTextStyle = {
    color: token.colorTextSecondary,
    fontSize: 12,
    lineHeight: 1.4,
  } as const

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
      const listRes = await getFlexibleScheduleList()
      setRows(listRes.data?.list || [])
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

  const openEdit = (row: FlexibleScheduleRow) => {
    setEditing(row)
    form.setFieldsValue({
      cron_expr: row.cron_expr,
      enabled: row.enabled,
    })
    setPreviewRuns(row.next_runs || [])
    setFormOpen(true)
  }

  const onPreview = async () => {
    try {
      const values = await form.validateFields(['cron_expr'])
      const res = await previewFlexibleSchedule({
        cron_expr: values.cron_expr,
        count: 5,
      })
      setPreviewRuns(res.data?.next_runs || [])
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) return
      showError(error, t('common.operation_failed'))
    }
  }

  const onSave = async () => {
    if (!editing) return
    try {
      const values = await form.validateFields()
      setSaving(true)
      await updateFlexibleSchedule(editing.id, {
        cron_expr: values.cron_expr,
        enabled: !!values.enabled,
      })
      message.success(t('common.update_success'))
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

  const openResult = (row: FlexibleScheduleRow) => {
    setResultRow(row)
    setResultOpen(true)
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
          if (updated) openResult(updated)
        } catch (error) {
          showError(error, t('common.operation_failed'))
          throw error
        } finally {
          setRunningId(null)
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
            <div style={{ marginTop: 4, ...mutedTextStyle }}>{describeCron(value, t)}</div>
          ) : null}
        </div>
      ),
    },
    {
      title: t('common.status'),
      dataIndex: 'enabled',
      key: 'enabled',
      width: 100,
      align: 'center',
      render: (enabled: boolean, row) => (
        <Switch checked={enabled} onChange={(v) => void onToggle(row, v)} disabled={!canUpdate} />
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
      width: 240,
      fixed: 'right',
      align: 'center',
      render: (_, row) => (
        <Space size={0}>
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
          <Button
            type="link"
            disabled={!row.last_status || row.last_status === 'never'}
            onClick={() => openResult(row)}
          >
            {t('schedule.view_result')}
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <>
      <Card
        style={{ marginTop: 16 }}
        title={t('schedule.flexible_title')}
        extra={
          <Button onClick={() => void loadData()} loading={loading}>
            {t('common.refresh')}
          </Button>
        }
      >
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1200 }}
          pagination={false}
        />
      </Card>

      <Modal
        title={t('common.edit')}
        open={formOpen}
        onCancel={() => setFormOpen(false)}
        onOk={() => void onSave()}
        confirmLoading={saving}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 12 }}>
          <Form.Item
            name="cron_expr"
            label={t('schedule.cron')}
            extra={t('schedule.flexible_cron_tip')}
            rules={[{ required: true }]}
          >
            <Input placeholder="*/5 * * * *" />
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

      <Modal
        open={resultOpen}
        title={t('schedule.result_title')}
        onCancel={() => setResultOpen(false)}
        footer={[
          <Button key="close" type="primary" onClick={() => setResultOpen(false)}>
            {t('common.close')}
          </Button>,
        ]}
        width={720}
        destroyOnClose
      >
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label={t('schedule.flexible_name')}>
            {resultRow?.name || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('schedule.flexible_handler')}>
            {resultRow?.handler_name || resultRow?.handler || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('schedule.last_status')}>
            <Tag color={statusColor(resultRow?.last_status)}>{statusLabel(resultRow?.last_status)}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('schedule.last_run_at')}>
            {formatDateTimeDisplay(resultRow?.last_run_at)}
          </Descriptions.Item>
          <Descriptions.Item label={t('schedule.last_duration')}>
            {formatDuration(resultRow?.last_duration_ms)}
          </Descriptions.Item>
          {resultRow?.last_error ? (
            <Descriptions.Item label={t('schedule.error')}>
              <pre
                style={{
                  margin: 0,
                  maxHeight: 160,
                  overflow: 'auto',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  color: token.colorError,
                }}
              >
                {resultRow.last_error}
              </pre>
            </Descriptions.Item>
          ) : null}
          <Descriptions.Item label={t('schedule.output')}>
            <pre
              style={{
                margin: 0,
                maxHeight: 320,
                overflow: 'auto',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                fontSize: 12,
                lineHeight: 1.6,
                fontFamily: "Consolas, 'Courier New', ui-monospace, SFMono-Regular, Menlo, Monaco, monospace",
              }}
            >
              {resultRow?.last_output || t('schedule.output_empty')}
            </pre>
          </Descriptions.Item>
        </Descriptions>
      </Modal>
    </>
  )
}
