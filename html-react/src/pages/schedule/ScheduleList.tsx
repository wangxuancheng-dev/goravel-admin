import { useEffect, useState } from 'react'
import { App, Button, Card, Descriptions, Modal, Space, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  getScheduleList,
  runSchedule,
  type ScheduleRunResult,
  type ScheduleTask,
} from '@/api/schedule'
import PageContainer from '@/components/PageContainer'
import PermissionButton from '@/components/PermissionButton'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { describeCron } from '@/utils/cronLabel'
import { logger } from '@/utils/logger'

function formatDuration(ms?: number) {
  if (ms === undefined || ms === null) return '-'
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(2)} s`
}

export default function ScheduleList() {
  const { t } = useTranslation()
  const { modal, message } = App.useApp()
  const showError = useUnhandledError()
  const [loading, setLoading] = useState(false)
  const [tableData, setTableData] = useState<ScheduleTask[]>([])
  const [runningCommand, setRunningCommand] = useState('')
  const [resultOpen, setResultOpen] = useState(false)
  const [resultData, setResultData] = useState<ScheduleRunResult | null>(null)

  const statusLabel = (status?: string) => {
    if (status === 'success') return t('schedule.status_success')
    if (status === 'failed') return t('schedule.status_failed')
    return t('schedule.status_never')
  }

  const triggeredByLabel = (value?: string) => {
    if (value === 'schedule') return t('schedule.triggered_schedule')
    if (value === 'manual') return t('schedule.triggered_manual')
    return value || '-'
  }

  const statusColor = (status?: string) => {
    if (status === 'success') return 'success'
    if (status === 'failed') return 'error'
    return 'default'
  }

  const showResult = (payload: ScheduleRunResult) => {
    setResultData(payload)
    setResultOpen(true)
  }

  const openResultFromRow = (row: ScheduleTask) => {
    showResult({
      command: row.command,
      status: row.last_status || 'never',
      error: row.last_error,
      output: row.last_output,
      duration_ms: row.last_duration_ms,
      run_at: row.last_run_at,
      triggered_by: row.last_triggered_by,
    })
  }

  const loadData = async () => {
    setLoading(true)
    try {
      const res = await getScheduleList()
      setTableData(res.data?.list || [])
    } catch (error) {
      logger.error('Failed to load schedules:', error)
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [])

  const handleRun = (row: ScheduleTask) => {
    modal.confirm({
      title: t('schedule.run_now'),
      content: t('schedule.run_confirm', { command: row.command }),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        setRunningCommand(row.command)
        try {
          const res = await runSchedule(row.command)
          const result = res.data?.result
          if (result?.status === 'failed') {
            message.warning(t('schedule.run_failed'))
          } else {
            message.success(t('schedule.run_success'))
          }
          if (result) showResult(result)
          await loadData()
        } catch (error) {
          showError(error, t('common.operation_failed'))
          await loadData()
          throw error
        } finally {
          setRunningCommand('')
        }
      },
    })
  }

  const columns: ColumnsType<ScheduleTask> = [
    {
      title: t('schedule.command'),
      dataIndex: 'command',
      key: 'command',
      minWidth: 220,
    },
    {
      title: t('schedule.description'),
      key: 'description',
      minWidth: 180,
      render: (_, row) => row.description || row.command || '-',
    },
    {
      title: t('schedule.cron'),
      dataIndex: 'cron',
      key: 'cron',
      minWidth: 220,
      render: (value?: string) => (
        <div>
          <code style={{ fontFamily: "Consolas, 'Courier New', monospace", fontSize: 12 }}>
            {value || '-'}
          </code>
          {value ? (
            <div style={{ marginTop: 4, color: 'rgba(0,0,0,0.45)', fontSize: 12, lineHeight: 1.4 }}>
              {describeCron(value, t)}
            </div>
          ) : null}
        </div>
      ),
    },
    {
      title: t('schedule.on_one_server'),
      dataIndex: 'on_one_server',
      key: 'on_one_server',
      width: 110,
      align: 'center',
      render: (value: boolean) => (
        <Tag color={value ? 'success' : 'default'}>{value ? t('common.yes') : t('common.no')}</Tag>
      ),
    },
    {
      title: t('schedule.last_status'),
      dataIndex: 'last_status',
      key: 'last_status',
      width: 110,
      align: 'center',
      render: (value?: string) => <Tag color={statusColor(value)}>{statusLabel(value)}</Tag>,
    },
    {
      title: t('schedule.last_run_at'),
      dataIndex: 'last_run_at',
      key: 'last_run_at',
      minWidth: 160,
      render: (value?: string) => value || '-',
    },
    {
      title: t('schedule.last_duration'),
      dataIndex: 'last_duration_ms',
      key: 'last_duration_ms',
      width: 110,
      align: 'right',
      render: (value?: number) => formatDuration(value),
    },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 180,
      fixed: 'right',
      align: 'center',
      render: (_, row) => (
        <Space size={0}>
          <PermissionButton
            permission="schedule.run"
            type="link"
            loading={runningCommand === row.command}
            disabled={!!runningCommand}
            onClick={() => handleRun(row)}
          >
            {t('schedule.run_now')}
          </PermissionButton>
          <Button
            type="link"
            disabled={!row.last_status || row.last_status === 'never'}
            onClick={() => openResultFromRow(row)}
          >
            {t('schedule.view_result')}
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <PageContainer title={t('menu.schedule')}>
      <Card
        title={
          <div>
            <div>{t('menu.schedule')}</div>
            <div style={{ marginTop: 6, fontSize: 13, fontWeight: 400, color: 'rgba(0,0,0,0.45)' }}>
              {t('schedule.hint')}
            </div>
          </div>
        }
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void loadData()}>
              {t('common.refresh')}
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={tableData}
          pagination={false}
          scroll={{ x: 1100 }}
        />
      </Card>

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
        destroyOnHidden
      >
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label={t('schedule.command')}>{resultData?.command || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('schedule.last_status')}>
            <Tag color={statusColor(resultData?.status)}>{statusLabel(resultData?.status)}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('schedule.last_run_at')}>{resultData?.run_at || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('schedule.last_duration')}>
            {formatDuration(resultData?.duration_ms)}
          </Descriptions.Item>
          <Descriptions.Item label={t('schedule.triggered_by')}>
            {triggeredByLabel(resultData?.triggered_by)}
          </Descriptions.Item>
          {resultData?.error ? (
            <Descriptions.Item label={t('schedule.error')}>
              <pre
                style={{
                  margin: 0,
                  maxHeight: 160,
                  overflow: 'auto',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  color: '#cf1322',
                  fontSize: 12,
                }}
              >
                {resultData.error}
              </pre>
            </Descriptions.Item>
          ) : null}
          <Descriptions.Item label={t('schedule.output')}>
            <pre
              style={{
                margin: 0,
                maxHeight: 320,
                overflow: 'auto',
                whiteSpace: 'pre',
                wordBreak: 'normal',
                fontSize: 12,
                lineHeight: 1.6,
                fontFamily: "Consolas, 'Courier New', ui-monospace, SFMono-Regular, Menlo, Monaco, monospace",
              }}
            >
              {resultData?.output || t('schedule.output_empty')}
            </pre>
          </Descriptions.Item>
        </Descriptions>
      </Modal>
    </PageContainer>
  )
}
