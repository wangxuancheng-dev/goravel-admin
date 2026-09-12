import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Progress,
  Row,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  CloudServerOutlined,
  DesktopOutlined,
  FolderOpenOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import type { EChartsOption } from 'echarts'
import { useTranslation } from 'react-i18next'
import { createSystemInfoSSE, getSystemInfo } from '@/api/monitor'
import PageContainer from '@/components/PageContainer'
import { THEME_COLORS, useAppStore } from '@/stores/app'
import { closeSSEConnection, createSSEConnection } from '@/utils/sse'
import type { ApiError } from '@/types'
import {
  formatBytes,
  formatDuration,
  formatLoad,
  formatNumber,
  formatPercent,
  formatPercentForProgress,
  formatUptime,
  getProgressColor,
} from './monitorFormat'
import './Monitor.scss'

const MAX_HISTORY_POINTS = 60
const POLL_INTERVAL_MS = 30000
const MIN_REFRESH_INTERVAL = 2000
const MAX_ERROR_COUNT = 5
const ERROR_WINDOW = 10000

interface SystemAlert {
  level?: string
  message?: string
  message_key?: string
  value?: number
}

interface ProcessInfo {
  status?: string
  type?: string
  pid?: number
  cpu?: number
  memory?: number
  version?: string
  host?: string
  threads?: number
  connections?: number
  max_connections?: number
  queries?: number
  uptime?: number
  slow_queries?: number
  table_locks_waited?: number
  innodb_row_lock_waits?: number
  threads_running?: number
  buffer_pool_size?: number
  active_connections?: number
  idle_connections?: number
  database_size?: number
  connected_clients?: number
  total_commands_processed?: number
  instantaneous_ops_per_sec?: number
  keyspace_hits?: number
  keyspace_misses?: number
  keyspace_hit_rate?: number
  used_memory_peak?: number
  db_size?: number
  process_name?: string
}

interface SystemInfo {
  os?: string
  alerts?: SystemAlert[]
  cpu?: Record<string, unknown>
  memory?: Record<string, unknown>
  disk?: Record<string, unknown>
  net?: Record<string, unknown>
  load?: Record<string, unknown>
  file_descriptors?: Record<string, unknown>
  runtime?: Record<string, unknown> & {
    memory?: Record<string, unknown>
  }
  system?: Record<string, unknown>
  app?: Record<string, unknown>
  health?: Record<string, unknown>
  websocket?: Record<string, unknown>
  processes?: {
    mysql?: ProcessInfo
    postgresql?: ProcessInfo
    redis?: ProcessInfo
    app?: ProcessInfo
  }
  process_top?: {
    by_cpu?: Record<string, unknown>[]
    by_memory?: Record<string, unknown>[]
  }
  tcp_connections?: Record<string, unknown> & { listening_ports?: number[] }
  disk_io?: Record<string, unknown>
  disk_partitions?: Record<string, unknown>[]
  network_interfaces?: Record<string, unknown>[]
}

interface HistoryData {
  cpu: number[]
  memory: number[]
  disk: number[]
  network: { sent: number[]; recv: number[]; total: number[] }
  timestamps: string[]
}

const emptyHistory = (): HistoryData => ({
  cpu: [],
  memory: [],
  disk: [],
  network: { sent: [], recv: [], total: [] },
  timestamps: [],
})

function getProcessStatusColor(status?: string): string {
  if (status === 'running' || status === 'connected' || status === 'sleep') return 'success'
  if (status === 'not_found' || status === 'disconnected') return 'error'
  if (status === 'error' || status === 'zombie' || status === 'stopped') return 'warning'
  return 'default'
}

function getProcessStatusLabel(status: string | undefined, t: (key: string) => string): string {
  if (status === 'running' || status === 'sleep') return t('monitor.process_running')
  if (status === 'connected') return t('monitor.process_connected')
  if (status === 'not_found') return t('monitor.process_not_found')
  if (status === 'disconnected') return t('monitor.process_disconnected')
  return status || '-'
}

function getConnectionPercentClass(connections: number, maxConnections: number): string {
  if (!maxConnections || maxConnections <= 0) return ''
  const percent = (connections / maxConnections) * 100
  if (percent >= 90) return 'connections-danger'
  if (percent >= 80) return 'connections-warning'
  return 'connections-safe'
}

function hasProcessTopUser(rows?: Record<string, unknown>[]): boolean {
  return !!rows?.some((r) => r?.user)
}

function SparklineChart({ option }: { option: EChartsOption }) {
  return <ReactECharts option={option} style={{ height: 180, marginTop: 15 }} notMerge lazyUpdate />
}

function ChartPanel({ option, height = 300 }: { option: EChartsOption; height?: number }) {
  return <ReactECharts option={option} style={{ height }} notMerge lazyUpdate />
}

export default function Monitor() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const darkMode = useAppStore((s) => s.darkMode)
  const themeColor = useAppStore((s) => s.themeColor)

  const primaryColor = useMemo(() => {
    const preset = THEME_COLORS.find((item) => item.key === themeColor) || THEME_COLORS[0]
    return preset.color
  }, [themeColor])

  const textColor = darkMode ? '#cfd3dc' : '#303133'
  const axisLineColor = darkMode ? '#3d3e40' : '#dcdfe6'
  const splitLineColor = darkMode ? '#3d3e40' : '#ebeef5'
  const tooltipBg = darkMode ? '#2d2d30' : '#fff'
  const tooltipBorder = darkMode ? '#3d3e40' : '#e4e7ed'

  const [systemInfo, setSystemInfo] = useState<SystemInfo>({ os: 'linux', alerts: [] })
  const [historyData, setHistoryData] = useState<HistoryData>(emptyHistory)
  const [refreshing, setRefreshing] = useState(false)

  const eventSourceRef = useRef<EventSource | null>(null)
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const errorCountRef = useRef(0)
  const lastErrorTimeRef = useRef(0)
  const lastRefreshTimeRef = useRef(0)

  const isLinux = systemInfo.os === 'linux'
  const hasTcpStats = !!systemInfo.tcp_connections
  const hasDiskIOStats = Number(systemInfo.disk_io?.total_read_bytes || 0) > 0
  const partitions = systemInfo.disk_partitions || []
  const hasSmallDiskPartitions = partitions.length > 0 && partitions.length <= 3
  const hasLargeDiskPartitions = partitions.length > 3

  const getAlertMessage = useCallback(
    (alert: SystemAlert) => {
      if (!alert) return ''
      if (alert.message) return alert.message
      if (alert.message_key && alert.value !== undefined) {
        return t(`messages.${alert.message_key}`, { value: alert.value.toFixed(2) })
      }
      return ''
    },
    [t],
  )

  const updateHistoryData = useCallback((info: SystemInfo) => {
    const timeStr = new Date().toLocaleTimeString()
    setHistoryData((prev) => {
      const next = {
        cpu: [...prev.cpu, Number(info.cpu?.percent || 0)],
        memory: [...prev.memory, Number(info.memory?.percent || 0)],
        disk: [...prev.disk, Number(info.disk?.percent || 0)],
        network: {
          sent: [...prev.network.sent, Number(info.net?.speed_sent_mbps || 0)],
          recv: [...prev.network.recv, Number(info.net?.speed_recv_mbps || 0)],
          total: [...prev.network.total, Number(info.net?.speed_total_mbps || 0)],
        },
        timestamps: [...prev.timestamps, timeStr],
      }
      if (next.cpu.length > MAX_HISTORY_POINTS) {
        next.cpu.shift()
        next.memory.shift()
        next.disk.shift()
        next.network.sent.shift()
        next.network.recv.shift()
        next.network.total.shift()
        next.timestamps.shift()
      }
      return next
    })
  }, [])

  const applySystemInfo = useCallback(
    (data: SystemInfo) => {
      setSystemInfo(data || {})
      updateHistoryData(data || {})
    },
    [updateHistoryData],
  )

  const cleanup = useCallback(() => {
    if (eventSourceRef.current) {
      closeSSEConnection(eventSourceRef.current)
      eventSourceRef.current = null
    }
    if (refreshTimerRef.current) {
      clearInterval(refreshTimerRef.current)
      refreshTimerRef.current = null
    }
  }, [])

  const startPolling = useCallback(() => {
    if (refreshTimerRef.current) return
    loadDataRef.current({ silent: true })
    refreshTimerRef.current = setInterval(() => {
      loadDataRef.current({ silent: true })
    }, POLL_INTERVAL_MS)
  }, [])

  const loadDataRef = useRef<(options?: { silent?: boolean }) => Promise<void>>(async () => {})

  const loadData = useCallback(
    async (options: { silent?: boolean } = {}) => {
      const { silent = false } = options
      if (refreshing) return

      const now = Date.now()
      const timeSinceLastRefresh = now - lastRefreshTimeRef.current
      if (!silent && timeSinceLastRefresh < MIN_REFRESH_INTERVAL) {
        const remainingTime = Math.ceil((MIN_REFRESH_INTERVAL - timeSinceLastRefresh) / 1000)
        message.warning(t('monitor.refresh_too_frequent', { seconds: remainingTime }))
        return
      }
      lastRefreshTimeRef.current = now

      setRefreshing(!silent)
      try {
        const res = await getSystemInfo()
        applySystemInfo((res.data as SystemInfo) || {})
      } catch (error) {
        const err = error as ApiError
        if (!silent && !err?.__handled) {
          const response = err?.response as { data?: { message?: string } } | undefined
          message.error(response?.data?.message || err?.message || t('error.default'))
        }
      } finally {
        setRefreshing(false)
      }
    },
    [refreshing, applySystemInfo, message, t],
  )

  loadDataRef.current = loadData

  const startSSEStream = useCallback(() => {
    try {
      errorCountRef.current = 0
      lastErrorTimeRef.current = 0
      const url = createSystemInfoSSE({ interval: 2 })
      eventSourceRef.current = createSSEConnection(url, {
        onMessage: (data) => {
          errorCountRef.current = 0
          lastErrorTimeRef.current = 0
          const payload = data as { type?: string; data?: SystemInfo }
          if (payload.type === 'system_info') {
            applySystemInfo(payload.data || {})
          }
        },
        onError: (_error, source) => {
          const now = Date.now()
          if (now - lastErrorTimeRef.current > ERROR_WINDOW) {
            errorCountRef.current = 0
          }
          errorCountRef.current += 1
          lastErrorTimeRef.current = now

          if (source?.readyState === EventSource.CLOSED) {
            message.warning(t('monitor.sse_connection_failed'))
            closeSSEConnection(eventSourceRef.current)
            eventSourceRef.current = null
            startPolling()
          } else if (errorCountRef.current >= MAX_ERROR_COUNT) {
            message.warning(t('monitor.sse_connection_failed'))
            closeSSEConnection(eventSourceRef.current)
            eventSourceRef.current = null
            startPolling()
          }
        },
        onOpen: () => {
          errorCountRef.current = 0
          lastErrorTimeRef.current = 0
        },
      })
    } catch {
      message.warning(t('monitor.sse_init_failed'))
      startPolling()
    }
  }, [applySystemInfo, message, startPolling, t])

  useEffect(() => {
    const hasData = Boolean(systemInfo.cpu?.model || systemInfo.memory?.total)
    if (!hasData) {
      loadData({ silent: true })
    }
    startSSEStream()
    return cleanup
  }, [cleanup, startSSEStream]) // eslint-disable-line react-hooks/exhaustive-deps

  const makeSparklineOption = useCallback(
    (
      label: string,
      color: string,
      data: number[],
      timestamps: string[],
      yMax = 100,
      valueSuffix = '%',
    ): EChartsOption => ({
      backgroundColor: 'transparent',
      grid: { top: 10, left: 30, right: 10, bottom: 30 },
      xAxis: {
        type: 'category',
        data: timestamps,
        axisLabel: { fontSize: 10, rotate: 45, color: textColor },
        axisLine: { lineStyle: { color: axisLineColor } },
      },
      yAxis: {
        type: 'value',
        max: yMax,
        axisLabel: {
          fontSize: 10,
          formatter: `{value}${valueSuffix}`,
          color: textColor,
        },
        axisLine: { lineStyle: { color: axisLineColor } },
        splitLine: { lineStyle: { color: splitLineColor } },
      },
      series: [
        {
          data,
          type: 'line',
          smooth: true,
          areaStyle: { opacity: 0.3 },
          lineStyle: { color, width: 2 },
          itemStyle: { color },
        },
      ],
      tooltip: {
        trigger: 'axis',
        backgroundColor: tooltipBg,
        borderColor: tooltipBorder,
        textStyle: { color: textColor },
        formatter: (params) => {
          const items = Array.isArray(params) ? params : [params]
          if (!items.length) return ''
          let result = `${items[0].name}<br/>`
          items.forEach((item) => {
            const value = typeof item.value === 'number' ? item.value.toFixed(1) : item.value
            result += `${item.marker}${label}: ${value}${valueSuffix}<br/>`
          })
          return result
        },
      },
    }),
    [textColor, axisLineColor, splitLineColor, tooltipBg, tooltipBorder],
  )

  const cpuChartOption = useMemo(
    () => makeSparklineOption(t('monitor.cpu'), '#F56C6C', historyData.cpu, historyData.timestamps),
    [historyData, makeSparklineOption, t],
  )

  const memoryChartOption = useMemo(
    () =>
      makeSparklineOption(t('monitor.memory'), primaryColor, historyData.memory, historyData.timestamps),
    [historyData, makeSparklineOption, primaryColor, t],
  )

  const diskChartOption = useMemo(
    () => makeSparklineOption(t('monitor.disk'), '#67C23A', historyData.disk, historyData.timestamps),
    [historyData, makeSparklineOption, t],
  )

  const networkChartOption = useMemo(
    (): EChartsOption => ({
      backgroundColor: 'transparent',
      grid: { top: 20, left: 40, right: 20, bottom: 30 },
      legend: {
        data: [t('monitor.net_send'), t('monitor.net_receive')],
        top: 5,
        textStyle: { fontSize: 11, color: textColor },
      },
      xAxis: {
        type: 'category',
        data: historyData.timestamps,
        axisLabel: { fontSize: 10, rotate: 45, color: textColor },
        axisLine: { lineStyle: { color: axisLineColor } },
      },
      yAxis: {
        type: 'value',
        axisLabel: { fontSize: 10, formatter: '{value} Mbps', color: textColor },
        axisLine: { lineStyle: { color: axisLineColor } },
        splitLine: { lineStyle: { color: splitLineColor } },
      },
      series: [
        {
          name: t('monitor.net_send'),
          data: historyData.network.sent,
          type: 'line',
          smooth: true,
          lineStyle: { color: primaryColor, width: 2 },
          itemStyle: { color: primaryColor },
        },
        {
          name: t('monitor.net_receive'),
          data: historyData.network.recv,
          type: 'line',
          smooth: true,
          lineStyle: { color: '#E6A23C', width: 2 },
          itemStyle: { color: '#E6A23C' },
        },
      ],
      tooltip: {
        trigger: 'axis',
        backgroundColor: tooltipBg,
        borderColor: tooltipBorder,
        textStyle: { color: textColor },
      },
    }),
    [historyData, primaryColor, t, textColor, axisLineColor, splitLineColor, tooltipBg, tooltipBorder],
  )

  const resourcePieOption = useMemo(
    (): EChartsOption => ({
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'item',
        backgroundColor: tooltipBg,
        borderColor: tooltipBorder,
        textStyle: { color: textColor },
      },
      legend: {
        orient: 'vertical',
        left: 'left',
        top: 'middle',
        textStyle: { fontSize: 12, color: textColor },
      },
      series: [
        {
          name: t('monitor.resource_usage'),
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['60%', '50%'],
          label: { show: true, color: textColor },
          data: [
            {
              value: Number(systemInfo.cpu?.percent || 0),
              name: t('monitor.cpu'),
              itemStyle: { color: '#F56C6C' },
            },
            {
              value: Number(systemInfo.memory?.percent || 0),
              name: t('monitor.memory'),
              itemStyle: { color: primaryColor },
            },
            {
              value: Number(systemInfo.disk?.percent || 0),
              name: t('monitor.disk'),
              itemStyle: { color: '#67C23A' },
            },
          ],
        },
      ],
    }),
    [systemInfo, primaryColor, t, textColor, tooltipBg, tooltipBorder],
  )

  const renderResourceCard = (
    title: string,
    icon: React.ReactNode,
    percent: number,
    usageLabel: string,
    infoItems: { label: string; value: React.ReactNode; highlight?: boolean }[],
    chartOption: EChartsOption,
    headerClass?: string,
  ) => (
    <Col xs={24} lg={8}>
      <Card
        className={`monitor-card ${headerClass || ''}`}
        title={
          <div className="monitor-card-header">
            <div className="monitor-card-title">
              {icon}
              <span>{title}</span>
            </div>
            <Button
              size="small"
              shape="circle"
              icon={<ReloadOutlined />}
              loading={refreshing}
              disabled={refreshing}
              onClick={() => loadData()}
            />
          </div>
        }
      >
        <div className="monitor-content">
          <div className="usage-header">
            <span>{usageLabel}</span>
            <span>{formatPercent(percent)}</span>
          </div>
          <Progress
            percent={formatPercentForProgress(percent)}
            strokeColor={getProgressColor(percent)}
            showInfo={false}
            strokeWidth={20}
          />
          <div className="info-grid">
            {infoItems.map((item) => (
              <div key={item.label} className={`info-item${item.label.length > 20 ? ' full-width' : ''}`}>
                <span className="info-label">{item.label}</span>
                <span className={`info-value${item.highlight ? ' highlight' : ''}`}>{item.value}</span>
              </div>
            ))}
          </div>
          <SparklineChart option={chartOption} />
        </div>
      </Card>
    </Col>
  )

  const renderProcessCard = (title: string, process?: ProcessInfo, extra?: React.ReactNode) => {
    if (!process) return null
    return (
      <Col xs={24} sm={12} lg={6}>
        <div className="process-card">
          <div className="process-header">
            <Tag color={getProcessStatusColor(process.status)}>{title}</Tag>
            {process.type ? (
              <Tag color={process.type === 'remote' ? 'warning' : 'success'}>
                {process.type === 'remote' ? t('monitor.process_remote') : t('monitor.process_local')}
              </Tag>
            ) : extra}
          </div>
          <div className="process-content">
            <div className="process-item">
              <span className="process-label">{t('monitor.process_status')}:</span>
              <span className="process-value">{getProcessStatusLabel(process.status, t)}</span>
            </div>
            {process.pid && process.pid > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_pid')}:</span>
                <span className="process-value">{process.pid}</span>
              </div>
            ) : null}
            {process.cpu !== undefined && process.cpu > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_cpu')}:</span>
                <span className="process-value">{formatPercent(process.cpu)}</span>
              </div>
            ) : null}
            {process.memory && process.memory > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_memory')}:</span>
                <span className="process-value">{formatBytes(process.memory)}</span>
              </div>
            ) : null}
            {process.version ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_version')}:</span>
                <span className="process-value">{process.version}</span>
              </div>
            ) : null}
            {process.host ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_host')}:</span>
                <span className="process-value">{process.host}</span>
              </div>
            ) : null}
            {process.connections !== undefined ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_connections')}:</span>
                <span className="process-value">
                  {formatNumber(process.connections)}
                  {process.max_connections ? (
                    <span className="connections-info">
                      {' '}
                      / {formatNumber(process.max_connections)}{' '}
                      <span
                        className={getConnectionPercentClass(
                          process.connections,
                          process.max_connections,
                        )}
                      >
                        (
                        {formatPercent(
                          (process.connections / process.max_connections) * 100,
                        )}
                        )
                      </span>
                    </span>
                  ) : null}
                </span>
              </div>
            ) : null}
            {process.queries !== undefined && process.queries > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_queries')}:</span>
                <span className="process-value">{formatNumber(process.queries)}</span>
              </div>
            ) : null}
            {process.uptime !== undefined && process.uptime > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_uptime')}:</span>
                <span className="process-value">{formatUptime(process.uptime)}</span>
              </div>
            ) : null}
            {process.slow_queries !== undefined && process.slow_queries > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_slow_queries')}:</span>
                <span className="process-value warning">{formatNumber(process.slow_queries)}</span>
              </div>
            ) : null}
            {process.table_locks_waited !== undefined && process.table_locks_waited > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_table_locks')}:</span>
                <span className="process-value warning">
                  {formatNumber(process.table_locks_waited)}
                </span>
              </div>
            ) : null}
            {process.innodb_row_lock_waits !== undefined && process.innodb_row_lock_waits > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_row_locks')}:</span>
                <span className="process-value warning">
                  {formatNumber(process.innodb_row_lock_waits)}
                </span>
              </div>
            ) : null}
            {process.threads_running !== undefined ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_threads_running')}:</span>
                <span className="process-value">{formatNumber(process.threads_running)}</span>
              </div>
            ) : null}
            {process.buffer_pool_size !== undefined && process.buffer_pool_size > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_buffer_pool')}:</span>
                <span className="process-value">{formatBytes(process.buffer_pool_size)}</span>
              </div>
            ) : null}
            {process.process_name ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.process_name')}:</span>
                <span className="process-value">{process.process_name}</span>
              </div>
            ) : null}
            {process.connected_clients !== undefined ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.redis_connected_clients')}:</span>
                <span className="process-value">{formatNumber(process.connected_clients)}</span>
              </div>
            ) : null}
            {process.db_size !== undefined && process.db_size > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.redis_db_size')}:</span>
                <span className="process-value">{formatNumber(process.db_size)}</span>
              </div>
            ) : null}
            {process.active_connections !== undefined ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.postgresql_active_connections')}:</span>
                <span className="process-value">{formatNumber(process.active_connections)}</span>
              </div>
            ) : null}
            {process.idle_connections !== undefined ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.postgresql_idle_connections')}:</span>
                <span className="process-value">{formatNumber(process.idle_connections)}</span>
              </div>
            ) : null}
            {process.database_size !== undefined && process.database_size > 0 ? (
              <div className="process-item">
                <span className="process-label">{t('monitor.postgresql_database_size')}:</span>
                <span className="process-value">{formatBytes(process.database_size)}</span>
              </div>
            ) : null}
          </div>
        </div>
      </Col>
    )
  }

  const processTopColumns = (showUser: boolean): ColumnsType<Record<string, unknown>> => [
    { title: '#', width: 48, render: (_, __, i) => i + 1 },
    { title: t('monitor.process_pid'), dataIndex: 'pid', width: 88 },
    { title: t('monitor.process_name'), dataIndex: 'name', ellipsis: true },
    ...(showUser
      ? [{ title: t('monitor.process_top_user'), dataIndex: 'user', width: 100, ellipsis: true }]
      : []),
    {
      title: t('monitor.process_cpu'),
      dataIndex: 'cpu_percent',
      width: 100,
      align: 'right' as const,
      render: (v) => formatPercent(Number(v || 0)),
    },
    {
      title: t('monitor.memory_usage'),
      dataIndex: 'memory_percent',
      width: 100,
      align: 'right' as const,
      render: (v) => formatPercent(Number(v || 0)),
    },
    {
      title: t('monitor.process_memory'),
      dataIndex: 'memory_bytes',
      align: 'right' as const,
      render: (v) => formatBytes(Number(v || 0)),
    },
  ]

  const runtimeMemory = systemInfo.runtime?.memory

  return (
    <PageContainer
      title={t('menu.monitor')}
      extra={
        <Button icon={<ReloadOutlined />} loading={refreshing} onClick={() => loadData()}>
          {t('common.refresh')}
        </Button>
      }
    >
      <div className="monitor-page">
        {(systemInfo.alerts || []).map((alert, index) => (
          <Alert
            key={index}
            type={alert.level === 'high' ? 'error' : 'warning'}
            showIcon
            message={getAlertMessage(alert)}
            style={{ marginBottom: 20 }}
          />
        ))}

        <Row gutter={[20, 20]}>
          {renderResourceCard(
            t('monitor.cpu'),
            <DesktopOutlined />,
            Number(systemInfo.cpu?.percent || 0),
            `${t('monitor.cpu_usage')} · ${t('monitor.cpu_usage_system', { cores: Number(systemInfo.cpu?.cores || 0) })}`,
            [
              { label: t('monitor.cpu_model'), value: String(systemInfo.cpu?.model || '-'), highlight: false },
              {
                label: t('monitor.cpu_logical_cores'),
                value: formatNumber(Number(systemInfo.cpu?.cores || 0)),
                highlight: true,
              },
              ...(systemInfo.cpu?.physical_cores
                ? [
                    {
                      label: t('monitor.cpu_physical_cores'),
                      value: formatNumber(Number(systemInfo.cpu?.physical_cores || 0)),
                    },
                  ]
                : []),
            ],
            cpuChartOption,
            'cpu-card',
          )}
          {renderResourceCard(
            t('monitor.memory'),
            <CloudServerOutlined />,
            Number(systemInfo.memory?.percent || 0),
            t('monitor.memory_usage'),
            [
              { label: t('monitor.memory_total'), value: formatBytes(Number(systemInfo.memory?.total || 0)) },
              {
                label: t('monitor.memory_used'),
                value: formatBytes(Number(systemInfo.memory?.used || 0)),
                highlight: true,
              },
              {
                label: t('monitor.memory_available'),
                value: formatBytes(Number(systemInfo.memory?.available || 0)),
              },
              { label: t('monitor.memory_free'), value: formatBytes(Number(systemInfo.memory?.free || 0)) },
            ],
            memoryChartOption,
          )}
          {renderResourceCard(
            t('monitor.disk'),
            <FolderOpenOutlined />,
            Number(systemInfo.disk?.percent || 0),
            t('monitor.disk_usage'),
            [
              { label: t('monitor.disk_total'), value: formatBytes(Number(systemInfo.disk?.total || 0)) },
              {
                label: t('monitor.disk_used'),
                value: formatBytes(Number(systemInfo.disk?.used || 0)),
                highlight: true,
              },
              { label: t('monitor.disk_free'), value: formatBytes(Number(systemInfo.disk?.free || 0)) },
              {
                label: t('monitor.disk_path'),
                value: String(systemInfo.disk?.path || '-'),
              },
            ],
            diskChartOption,
            'disk-card',
          )}
        </Row>

        <Row gutter={[20, 20]} style={{ marginTop: 20 }}>
          <Col xs={24} lg={12}>
            <Card className="monitor-card" title={t('monitor.resource_usage_chart')}>
              <ChartPanel option={resourcePieOption} />
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card className="monitor-card" title={t('monitor.network_speed_chart')}>
              {systemInfo.net?.speed_total_mbps !== undefined ? (
                <div
                  style={{
                    marginBottom: 16,
                    padding: 12,
                    background: 'var(--ant-color-fill-quaternary, #f5f5f5)',
                    borderRadius: 8,
                  }}
                >
                  <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 8 }}>
                    {t('monitor.bandwidth_speed')}
                  </div>
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      marginBottom: 6,
                    }}
                  >
                    <span style={{ fontSize: 12, opacity: 0.65 }}>{t('monitor.current_speed')}:</span>
                    <span style={{ fontSize: 16, fontWeight: 600 }}>
                      {formatNumber(Number(systemInfo.net?.speed_total_mbps || 0), 2)} Mbps
                    </span>
                  </div>
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      marginBottom: 6,
                    }}
                  >
                    <span style={{ fontSize: 12, opacity: 0.65 }}>{t('monitor.peak_speed')}:</span>
                    <span style={{ fontSize: 16, fontWeight: 600 }}>
                      {formatNumber(Number(systemInfo.net?.peak_total_mbps || 0), 2)} Mbps
                    </span>
                  </div>
                  <div style={{ fontSize: 11, opacity: 0.55, marginTop: 8 }}>
                    <span>↑{formatNumber(Number(systemInfo.net?.speed_sent_mbps || 0), 2)} Mbps</span>
                    <span style={{ margin: '0 8px' }}>|</span>
                    <span>↓{formatNumber(Number(systemInfo.net?.speed_recv_mbps || 0), 2)} Mbps</span>
                  </div>
                </div>
              ) : null}
              <ChartPanel option={networkChartOption} />
            </Card>
          </Col>
        </Row>

        {isLinux ? (
          <Row gutter={[20, 20]} style={{ marginTop: 20 }}>
            <Col xs={24} lg={12}>
              <Card className="monitor-card" title={t('monitor.load')}>
                <div className="load-display">
                  <div>
                    <span className="load-number">{formatLoad(Number(systemInfo.load?.load1 || 0))}</span>
                    <span className="load-percent">
                      ({formatPercent(Number(systemInfo.load?.load1_percent || 0))})
                    </span>
                  </div>
                  <div className="load-label">{t('monitor.load_current')}</div>
                </div>
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card className="monitor-card" title={t('monitor.file_descriptors')}>
                <div className="monitor-content">
                  <div className="usage-header">
                    <span>{t('monitor.fd_usage')}</span>
                    <span>{formatPercent(Number(systemInfo.file_descriptors?.percent || 0))}</span>
                  </div>
                  <Progress
                    percent={formatPercentForProgress(Number(systemInfo.file_descriptors?.percent || 0))}
                    strokeColor={getProgressColor(Number(systemInfo.file_descriptors?.percent || 0))}
                    showInfo={false}
                    strokeWidth={20}
                  />
                  <div className="info-grid">
                    <div className="info-item">
                      <span className="info-label">{t('monitor.fd_used')}</span>
                      <span className="info-value highlight">
                        {formatNumber(Number(systemInfo.file_descriptors?.used || 0))}
                      </span>
                    </div>
                    <div className="info-item">
                      <span className="info-label">{t('monitor.fd_free')}</span>
                      <span className="info-value">
                        {formatNumber(Number(systemInfo.file_descriptors?.free || 0))}
                      </span>
                    </div>
                    <div className="info-item full-width">
                      <span className="info-label">{t('monitor.fd_max')}</span>
                      <span className="info-value">
                        {formatNumber(Number(systemInfo.file_descriptors?.max || 0))}
                      </span>
                    </div>
                  </div>
                </div>
              </Card>
            </Col>
          </Row>
        ) : null}

        <Row gutter={[20, 20]} style={{ marginTop: 20 }}>
          <Col xs={24} lg={12}>
            <Card className="monitor-card" title={t('monitor.runtime')}>
              <div className="info-grid">
                <div className="info-item">
                  <span className="info-label">{t('monitor.goroutines')}</span>
                  <span className="info-value highlight">
                    {formatNumber(Number(systemInfo.runtime?.goroutines || 0))}
                  </span>
                </div>
                <div className="info-item">
                  <span className="info-label">{t('monitor.total_processes')}</span>
                  <span className="info-value highlight">
                    {formatNumber(Number(systemInfo.runtime?.total_processes || 0))}
                  </span>
                </div>
                {systemInfo.runtime?.num_cpu ? (
                  <div className="info-item">
                    <span className="info-label">{t('monitor.num_cpu')}</span>
                    <span className="info-value">{formatNumber(Number(systemInfo.runtime.num_cpu))}</span>
                  </div>
                ) : null}
                {systemInfo.runtime?.gomaxprocs ? (
                  <div className="info-item">
                    <span className="info-label">{t('monitor.gomaxprocs')}</span>
                    <span className="info-value">{formatNumber(Number(systemInfo.runtime.gomaxprocs))}</span>
                  </div>
                ) : null}
              </div>
              {runtimeMemory ? (
                <div style={{ marginTop: 15, paddingTop: 15, borderTop: '1px solid rgba(0,0,0,0.06)' }}>
                  <Typography.Text strong>{t('monitor.memory_stats')}</Typography.Text>
                  <div className="info-grid" style={{ marginTop: 10 }}>
                    {[
                      ['mem_alloc', runtimeMemory.alloc],
                      ['mem_total_alloc', runtimeMemory.total_alloc],
                      ['mem_sys', runtimeMemory.sys],
                      ['mem_heap_alloc', runtimeMemory.heap_alloc, true],
                      ['mem_heap_sys', runtimeMemory.heap_sys],
                      ['mem_heap_objects', runtimeMemory.heap_objects],
                      ['mem_num_gc', runtimeMemory.num_gc],
                    ].map(([key, value, highlight]) => (
                      <div key={String(key)} className="info-item">
                        <span className="info-label">{t(`monitor.${String(key)}`)}</span>
                        <span className={`info-value${highlight ? ' highlight' : ''}`}>
                          {String(key).includes('objects') || String(key).includes('num_gc')
                            ? formatNumber(Number(value || 0))
                            : formatBytes(Number(value || 0))}
                        </span>
                      </div>
                    ))}
                    {runtimeMemory.pause_total_ns ? (
                      <div className="info-item">
                        <span className="info-label">{t('monitor.mem_pause_total')}</span>
                        <span className="info-value">
                          {formatDuration(Number(runtimeMemory.pause_total_ns) / 1000000)}
                        </span>
                      </div>
                    ) : null}
                  </div>
                </div>
              ) : null}
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card className="monitor-card" title={t('monitor.system_info')}>
              <div className="info-grid">
                <div className="info-item full-width">
                  <span className="info-label">{t('monitor.hostname')}</span>
                  <span className="info-value highlight">{String(systemInfo.system?.hostname || '-')}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">{t('monitor.os')}</span>
                  <span className="info-value highlight">{String(systemInfo.system?.os || '-')}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">{t('monitor.arch')}</span>
                  <span className="info-value highlight">{String(systemInfo.system?.arch || '-')}</span>
                </div>
                <div className="info-item full-width">
                  <span className="info-label">{t('monitor.go_version')}</span>
                  <span className="info-value">{String(systemInfo.system?.go_version || '-')}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">{t('monitor_runtime.health_status')}</span>
                  <span className="info-value highlight">{String(systemInfo.health?.status || '-')}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">{t('monitor_runtime.alert_count')}</span>
                  <span className="info-value">{formatNumber(Number(systemInfo.health?.alert_count || 0))}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">{t('monitor_runtime.app_env')}</span>
                  <span className="info-value highlight">{String(systemInfo.app?.env || '-')}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">{t('monitor_runtime.app_debug')}</span>
                  <span className="info-value">{String(systemInfo.app?.debug ?? '-')}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">{t('monitor_runtime.queue_connection')}</span>
                  <span className="info-value">{String(systemInfo.app?.queue_connection || '-')}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">{t('monitor_runtime.cache_store')}</span>
                  <span className="info-value">{String(systemInfo.app?.cache_store || '-')}</span>
                </div>
                {systemInfo.websocket ? (
                  <>
                    <div className="info-item">
                      <span className="info-label">{t('monitor_runtime.ws_online_admins')}</span>
                      <span className="info-value highlight">
                        {formatNumber(Number(systemInfo.websocket.online_admins ?? 0))}
                      </span>
                    </div>
                    <div className="info-item">
                      <span className="info-label">{t('monitor_runtime.ws_connections')}</span>
                      <span className="info-value highlight">
                        {formatNumber(Number(systemInfo.websocket.connections ?? 0))}
                      </span>
                    </div>
                  </>
                ) : null}
              </div>
            </Card>
          </Col>
        </Row>

        {systemInfo.processes ? (
          <Card className="monitor-card" title={t('monitor.processes')} style={{ marginTop: 20 }}>
            <Row gutter={[20, 20]}>
              {renderProcessCard(t('monitor.process_mysql'), systemInfo.processes.mysql)}
              {renderProcessCard(t('monitor.process_postgresql'), systemInfo.processes.postgresql)}
              {renderProcessCard(t('monitor.process_redis'), systemInfo.processes.redis)}
              {renderProcessCard(
                t('monitor.process_app'),
                systemInfo.processes.app,
                <Tag>{t('monitor.process_local')}</Tag>,
              )}
            </Row>
          </Card>
        ) : null}

        {systemInfo.process_top &&
        ((systemInfo.process_top.by_cpu?.length || 0) > 0 ||
          (systemInfo.process_top.by_memory?.length || 0) > 0) ? (
          <Card
            className="monitor-card"
            title={
              <div className="process-ranking-header">
                <span>{t('monitor.process_ranking')}</span>
                <span className="process-ranking-hint">{t('monitor.process_ranking_hint')}</span>
              </div>
            }
            style={{ marginTop: 20 }}
          >
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <div className="process-top-subtitle">{t('monitor.process_top_by_cpu')}</div>
                <div className="process-top-cpu-hint">{t('monitor.process_top_cpu_hint')}</div>
                <Table
                  size="small"
                  bordered
                  pagination={false}
                  scroll={{ y: 380 }}
                  rowKey={(_, i) => `cpu-top-${i}`}
                  dataSource={systemInfo.process_top.by_cpu || []}
                  columns={processTopColumns(hasProcessTopUser(systemInfo.process_top.by_cpu))}
                />
              </Col>
              <Col xs={24} md={12} className="process-top-col-second">
                <div className="process-top-subtitle">{t('monitor.process_top_by_memory')}</div>
                <Table
                  size="small"
                  bordered
                  pagination={false}
                  scroll={{ y: 380 }}
                  rowKey={(_, i) => `mem-top-${i}`}
                  dataSource={systemInfo.process_top.by_memory || []}
                  columns={processTopColumns(hasProcessTopUser(systemInfo.process_top.by_memory))}
                />
              </Col>
            </Row>
          </Card>
        ) : null}

        {(hasTcpStats || hasDiskIOStats || hasSmallDiskPartitions) && (
          <Row gutter={[20, 20]} style={{ marginTop: 20 }}>
            {hasTcpStats ? (
              <Col xs={24} lg={hasDiskIOStats || hasSmallDiskPartitions ? 12 : 24}>
                <Card className="monitor-card" title={t('monitor.tcp_connections')}>
                  <div className="info-grid">
                    <div className="info-item">
                      <span className="info-label">{t('monitor.tcp_total')}</span>
                      <span className="info-value highlight">
                        {formatNumber(Number(systemInfo.tcp_connections?.total || 0))}
                      </span>
                    </div>
                    <div className="info-item">
                      <span className="info-label">{t('monitor.tcp_established')}</span>
                      <span className="info-value highlight">
                        {formatNumber(Number(systemInfo.tcp_connections?.established || 0))}
                      </span>
                    </div>
                    <div className="info-item">
                      <span className="info-label">{t('monitor.tcp_listen')}</span>
                      <span className="info-value">
                        {formatNumber(Number(systemInfo.tcp_connections?.listen || 0))}
                      </span>
                    </div>
                    <div className="info-item">
                      <span className="info-label">{t('monitor.tcp_time_wait')}</span>
                      <span className="info-value">
                        {formatNumber(Number(systemInfo.tcp_connections?.time_wait || 0))}
                      </span>
                    </div>
                  </div>
                  {(systemInfo.tcp_connections?.listening_ports || []).length > 0 ? (
                    <div style={{ marginTop: 15 }}>
                      <Typography.Text type="secondary">{t('monitor.tcp_listening_ports')}:</Typography.Text>
                      <div style={{ marginTop: 8 }}>
                        <Space wrap>
                          {(systemInfo.tcp_connections?.listening_ports || []).slice(0, 15).map((port) => (
                            <Tag key={port}>{port}</Tag>
                          ))}
                        </Space>
                      </div>
                    </div>
                  ) : null}
                </Card>
              </Col>
            ) : null}
            {hasDiskIOStats ? (
              <Col xs={24} lg={hasTcpStats || hasSmallDiskPartitions ? 12 : 24}>
                <Card className="monitor-card" title={t('monitor.disk_io')}>
                  <div className="info-grid">
                    <div className="info-item">
                      <span className="info-label">{t('monitor.disk_read_bytes')}</span>
                      <span className="info-value">
                        {formatBytes(Number(systemInfo.disk_io?.total_read_bytes || 0))}
                      </span>
                    </div>
                    <div className="info-item">
                      <span className="info-label">{t('monitor.disk_write_bytes')}</span>
                      <span className="info-value">
                        {formatBytes(Number(systemInfo.disk_io?.total_write_bytes || 0))}
                      </span>
                    </div>
                    <div className="info-item">
                      <span className="info-label">{t('monitor.disk_read_count')}</span>
                      <span className="info-value">
                        {formatNumber(Number(systemInfo.disk_io?.total_read_count || 0))}
                      </span>
                    </div>
                    <div className="info-item">
                      <span className="info-label">{t('monitor.disk_write_count')}</span>
                      <span className="info-value">
                        {formatNumber(Number(systemInfo.disk_io?.total_write_count || 0))}
                      </span>
                    </div>
                  </div>
                </Card>
              </Col>
            ) : null}
            {hasSmallDiskPartitions ? (
              <Col xs={24} lg={8}>
                <Card className="monitor-card" title={t('monitor.disk_partitions')}>
                  {partitions.map((part, index) => (
                    <div key={index} style={{ marginBottom: 12 }}>
                      <Typography.Text strong>
                        {String(part.mountpoint || part.device || index)}
                      </Typography.Text>
                      <Progress
                        percent={formatPercentForProgress(Number(part.percent || 0))}
                        strokeColor={getProgressColor(Number(part.percent || 0))}
                        size="small"
                        style={{ marginTop: 6 }}
                      />
                    </div>
                  ))}
                </Card>
              </Col>
            ) : null}
          </Row>
        )}

        {hasLargeDiskPartitions ? (
          <Card className="monitor-card" title={t('monitor.disk_partitions')} style={{ marginTop: 20 }}>
            <Table
              size="small"
              rowKey={(_, i) => `part-${i}`}
              dataSource={partitions}
              columns={[
                { title: t('monitor.partition_device'), dataIndex: 'device', width: 150 },
                { title: t('monitor.partition_mountpoint'), dataIndex: 'mountpoint' },
                { title: t('monitor.partition_fstype'), dataIndex: 'fstype', width: 120 },
                {
                  title: t('monitor.partition_total'),
                  dataIndex: 'total',
                  width: 120,
                  render: (v) => formatBytes(Number(v || 0)),
                },
                {
                  title: t('monitor.partition_used'),
                  dataIndex: 'used',
                  width: 120,
                  render: (v) => formatBytes(Number(v || 0)),
                },
                {
                  title: t('monitor.partition_free'),
                  dataIndex: 'free',
                  width: 120,
                  render: (v) => formatBytes(Number(v || 0)),
                },
                {
                  title: t('monitor.partition_percent'),
                  dataIndex: 'percent',
                  width: 180,
                  render: (v) => (
                    <Progress
                      percent={formatPercentForProgress(Number(v || 0))}
                      strokeColor={getProgressColor(Number(v || 0))}
                      size="small"
                    />
                  ),
                },
              ]}
            />
          </Card>
        ) : null}

        {(systemInfo.network_interfaces || []).length > 0 ? (
          <Card className="monitor-card" title={t('monitor.network_interfaces')} style={{ marginTop: 20 }}>
            <Table
              size="small"
              rowKey={(_, i) => `iface-${i}`}
              dataSource={systemInfo.network_interfaces}
              scroll={{ x: true }}
              columns={[
                { title: t('monitor.interface_name'), dataIndex: 'name', width: 140 },
                {
                  title: t('monitor.interface_bytes_sent'),
                  dataIndex: 'bytes_sent',
                  render: (v) => formatBytes(Number(v || 0)),
                },
                {
                  title: t('monitor.interface_bytes_recv'),
                  dataIndex: 'bytes_recv',
                  render: (v) => formatBytes(Number(v || 0)),
                },
                {
                  title: t('monitor.interface_packets_sent'),
                  dataIndex: 'packets_sent',
                  render: (v) => formatNumber(Number(v || 0)),
                },
                {
                  title: t('monitor.interface_packets_recv'),
                  dataIndex: 'packets_recv',
                  render: (v) => formatNumber(Number(v || 0)),
                },
              ]}
            />
          </Card>
        ) : null}
      </div>
    </PageContainer>
  )
}
