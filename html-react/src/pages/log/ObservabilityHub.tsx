import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert,
  App,
  Button,
  Card,
  Descriptions,
  Divider,
  Empty,
  Input,
  InputNumber,
  Space,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import {
  getApiPerformanceOverview,
  getApiPerformanceTraces,
  getAuditTimeline,
  getPprofCpuHotspots,
  getPprofMemoryHotspots,
  getPprofStatus,
  getQueueDashboard,
  testQueueAlert,
  getSlowSqlTop,
  getTraceAggregate,
  verifyPprofToken,
} from '@/api/observability'
import PageContainer from '@/components/PageContainer'
import { useUserStore } from '@/stores/user'
import type { ApiError } from '@/types'

type HttpErrorResponse = {
  status?: number
  data?: { message?: string; retry_after?: number }
}

function getErrorResponse(error: unknown): HttpErrorResponse | undefined {
  return (error as ApiError)?.response as HttpErrorResponse | undefined
}

type TabKey = 'queue' | 'trace' | 'audit' | 'slowSql' | 'apiPerformance' | 'pprof'

interface QueueRow {
  name: string
  pending: number
  reserved: number
  delayed: number
  failed: number
  total: number
  stream_total: number | null
}

interface QueueConnection {
  connection?: string
  kind?: string
  driver_raw?: string
  redis_client?: string
  consumer_group?: string
  default_queue?: string
  fetch_error?: string
  is_default?: boolean
  queues?: Record<string, Record<string, unknown>>
}

interface TraceData {
  trace_id?: string
  request?: { method?: string; path?: string }
  operations?: Record<string, unknown>[]
  exceptions?: Record<string, unknown>[]
}

interface HotspotRow {
  key: number
  function: string
  flat_ms?: string
  flat_percent?: string
  cum_ms?: string
  cum_percent?: string
  flat_bytes?: string
  cum_bytes?: string
}

const QUEUE_KIND_COLORS: Record<string, string> = {
  database: 'success',
  redis_list: 'processing',
  redis_stream: 'warning',
  sync: 'default',
  other: 'error',
}

function queuesToRows(queues: Record<string, Record<string, unknown>> | undefined): QueueRow[] {
  if (!queues || typeof queues !== 'object') return []
  return Object.keys(queues).map((name) => {
    const s = queues[name] || {}
    return {
      name,
      pending: Number(s.pending ?? 0),
      reserved: Number(s.reserved ?? 0),
      delayed: s.delayed != null ? Number(s.delayed) : 0,
      failed: Number(s.failed ?? 0),
      total: Number(s.total ?? 0),
      stream_total: s.stream_total != null ? Number(s.stream_total) : null,
    }
  })
}

function formatHotspotBytes(bytes: number): string {
  const value = Number(bytes || 0)
  if (value >= 1024 * 1024 * 1024) return `${(value / (1024 * 1024 * 1024)).toFixed(2)} GB`
  if (value >= 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(2)} MB`
  if (value >= 1024) return `${(value / 1024).toFixed(2)} KB`
  return `${value} B`
}

export default function ObservabilityHub() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const hasPermission = useUserStore((s) => s.hasPermission)
  const config = useUserStore((s) => s.config)

  const hasAnyPermission = useCallback(
    (slugs: string[]) => slugs.some((slug) => hasPermission(slug)),
    [hasPermission],
  )

  const [tabAccess, setTabAccess] = useState({
    queue: hasAnyPermission(['observability.queue_dashboard']),
    trace: hasAnyPermission(['observability.trace']),
    audit: hasAnyPermission(['observability.audit_timeline']),
    slowSql: hasAnyPermission(['observability.slow_sql_top']),
    apiPerformance: hasAnyPermission([
      'observability.api_performance_overview',
      'observability.api_performance_traces',
      'observability.api_performance',
    ]),
    pprof: !!(config.isDeveloperAdmin && config.pprofEnabled),
  })

  const [pprofStatus, setPprofStatus] = useState({
    enabled: false,
    is_developer: false,
    token_required: false,
  })
  const [pprofStatusLoaded, setPprofStatusLoaded] = useState(false)
  const [pprofVerified, setPprofVerified] = useState(false)
  const [pprofVerifying, setPprofVerifying] = useState(false)
  const [pprofToken, setPprofToken] = useState('')

  const showPprofTab =
    pprofStatusLoaded &&
    pprofStatus.enabled &&
    pprofStatus.is_developer &&
    !!config.isDeveloperAdmin &&
    !!config.pprofEnabled

  const visibleTabs = useMemo(() => {
    const tabs: TabKey[] = []
    if (tabAccess.queue) tabs.push('queue')
    if (tabAccess.trace) tabs.push('trace')
    if (tabAccess.audit) tabs.push('audit')
    if (tabAccess.slowSql) tabs.push('slowSql')
    if (tabAccess.apiPerformance) tabs.push('apiPerformance')
    if (tabAccess.pprof && showPprofTab) tabs.push('pprof')
    return tabs
  }, [tabAccess, showPprofTab])

  const [activeTab, setActiveTab] = useState<TabKey>('queue')

  useEffect(() => {
    if (!visibleTabs.length) return
    if (!visibleTabs.includes(activeTab)) {
      setActiveTab(visibleTabs[0])
    }
  }, [visibleTabs, activeTab])

  const handleViewRequestError = useCallback(
    (error: unknown) => {
      const err = error as ApiError
      if (err?.__handled) return
      message.error(getErrorResponse(error)?.data?.message || err?.message || t('error.default'))
    },
    [message, t],
  )

  // Queue tab
  const [queueLoading, setQueueLoading] = useState(false)
  const [queueAlertTesting, setQueueAlertTesting] = useState(false)
  const [queueDashboard, setQueueDashboard] = useState<{
    default_connection: string
    connections: QueueConnection[]
    queue_alert?: {
      configured?: boolean
      source?: string
      url_masked?: string
      threshold?: number
    }
  }>({ default_connection: '', connections: [] })

  const canTestQueueAlert = hasAnyPermission(['observability.queue_alert_test'])

  const currentQueuePanel = useMemo(() => {
    const rows = queueDashboard.connections || []
    if (!rows.length) return null
    return (
      rows.find((item) => item?.is_default) ||
      rows.find((item) => item?.connection === queueDashboard.default_connection) ||
      rows[0]
    )
  }, [queueDashboard])

  const currentQueueRows = useMemo(
    () => queuesToRows(currentQueuePanel?.queues),
    [currentQueuePanel],
  )

  const loadQueue = useCallback(async () => {
    setQueueLoading(true)
    try {
      const res = await getQueueDashboard()
      const data = (res.data || {}) as Record<string, unknown>
      setQueueDashboard({
        default_connection: String(data.default_connection || ''),
        connections: (data.connections as QueueConnection[]) || [],
        queue_alert: (data.queue_alert as {
          configured?: boolean
          source?: string
          url_masked?: string
          threshold?: number
        }) || undefined,
      })
    } catch (error) {
      handleViewRequestError(error)
    } finally {
      setQueueLoading(false)
    }
  }, [handleViewRequestError])

  const onTestQueueAlert = useCallback(async () => {
    setQueueAlertTesting(true)
    try {
      await testQueueAlert()
      message.success(t('observability.queue_alert_test_ok'))
      await loadQueue()
    } catch (error) {
      handleViewRequestError(error)
    } finally {
      setQueueAlertTesting(false)
    }
  }, [handleViewRequestError, loadQueue, message, t])

  // Trace tab
  const [traceLoading, setTraceLoading] = useState(false)
  const [traceId, setTraceId] = useState('')
  const [traceData, setTraceData] = useState<TraceData>({})

  const loadTrace = useCallback(async () => {
    if (!traceId) {
      message.warning('trace_id is required')
      return
    }
    setTraceLoading(true)
    try {
      const res = await getTraceAggregate({ trace_id: traceId })
      setTraceData((res.data as TraceData) || {})
    } catch (error) {
      handleViewRequestError(error)
    } finally {
      setTraceLoading(false)
    }
  }, [traceId, handleViewRequestError, message])

  // Audit tab
  const [auditLoading, setAuditLoading] = useState(false)
  const [auditQuery, setAuditQuery] = useState({
    trace_id: '',
    keyword: '',
    page: 1,
    page_size: 20,
  })
  const [auditData, setAuditData] = useState<{ list: Record<string, unknown>[]; total: number }>({
    list: [],
    total: 0,
  })

  const loadAudit = useCallback(async () => {
    setAuditLoading(true)
    try {
      const res = await getAuditTimeline(auditQuery)
      const data = (res.data || {}) as Record<string, unknown>
      setAuditData({
        list: (data.list as Record<string, unknown>[]) || [],
        total: Number(data.total || 0),
      })
    } catch (error) {
      handleViewRequestError(error)
    } finally {
      setAuditLoading(false)
    }
  }, [auditQuery, handleViewRequestError])

  // Slow SQL tab
  const [slowSqlLoading, setSlowSqlLoading] = useState(false)
  const [slowSqlQuery, setSlowSqlQuery] = useState({ hours: 24, min_duration_ms: 200, limit: 20 })
  const [slowSqlData, setSlowSqlData] = useState<Record<string, unknown>[]>([])

  const loadSlowSql = useCallback(async () => {
    setSlowSqlLoading(true)
    try {
      const res = await getSlowSqlTop(slowSqlQuery)
      const data = (res.data || {}) as Record<string, unknown>
      setSlowSqlData((data.list as Record<string, unknown>[]) || [])
    } catch (error) {
      handleViewRequestError(error)
    } finally {
      setSlowSqlLoading(false)
    }
  }, [slowSqlQuery, handleViewRequestError])

  // API performance tab
  const [apiPerfLoading, setApiPerfLoading] = useState(false)
  const [apiPerfTraceLoading, setApiPerfTraceLoading] = useState(false)
  const [apiPerfQuery, setApiPerfQuery] = useState({ hours: 24, limit: 20 })
  const [apiPerfData, setApiPerfData] = useState<{
    slow_top: Record<string, unknown>[]
    error_top: Record<string, unknown>[]
    qps_top: Record<string, unknown>[]
  }>({ slow_top: [], error_top: [], qps_top: [] })
  const [apiPerfTraceData, setApiPerfTraceData] = useState<Record<string, unknown>[]>([])
  const [selectedApiEndpoint, setSelectedApiEndpoint] = useState({ method: '', route_template: '' })

  const loadApiPerformance = useCallback(async () => {
    setApiPerfLoading(true)
    try {
      const res = await getApiPerformanceOverview(apiPerfQuery)
      const data = (res.data || {}) as Record<string, unknown>
      setApiPerfData({
        slow_top: (data.slow_top as Record<string, unknown>[]) || [],
        error_top: (data.error_top as Record<string, unknown>[]) || [],
        qps_top: (data.qps_top as Record<string, unknown>[]) || [],
      })
    } catch (error) {
      if (getErrorResponse(error)?.status === 403) {
        setTabAccess((prev) => ({ ...prev, apiPerformance: false }))
        return
      }
      handleViewRequestError(error)
    } finally {
      setApiPerfLoading(false)
    }
  }, [apiPerfQuery, handleViewRequestError])

  const loadApiPerformanceTraces = useCallback(
    async (row: Record<string, unknown>) => {
      const method = String(row?.method || '')
      const route_template = String(row?.route_template || '')
      setSelectedApiEndpoint({ method, route_template })
      setApiPerfTraceLoading(true)
      try {
        const res = await getApiPerformanceTraces({
          method,
          route_template,
          hours: apiPerfQuery.hours,
          limit: apiPerfQuery.limit,
        })
        const data = (res.data || {}) as Record<string, unknown>
        setApiPerfTraceData((data.list as Record<string, unknown>[]) || [])
      } catch (error) {
        if (getErrorResponse(error)?.status === 403) {
          setTabAccess((prev) => ({ ...prev, apiPerformance: false }))
          return
        }
        handleViewRequestError(error)
      } finally {
        setApiPerfTraceLoading(false)
      }
    },
    [apiPerfQuery, handleViewRequestError],
  )

  const jumpToTrace = useCallback(
    (id: string) => {
      if (!id) return
      setTraceId(id)
      setActiveTab('trace')
      setTraceLoading(true)
      getTraceAggregate({ trace_id: id })
        .then((res) => setTraceData((res.data as TraceData) || {}))
        .catch(handleViewRequestError)
        .finally(() => setTraceLoading(false))
    },
    [handleViewRequestError],
  )

  // Pprof tab
  const [cpuHotspotForm, setCpuHotspotForm] = useState({ seconds: 3, top_n: 15 })
  const [memoryHotspotForm, setMemoryHotspotForm] = useState({ top_n: 25 })
  const [cpuHotspotLoading, setCpuHotspotLoading] = useState(false)
  const [memoryHotspotLoading, setMemoryHotspotLoading] = useState(false)
  const [cpuHotspotData, setCpuHotspotData] = useState<HotspotRow[]>([])
  const [memoryHotspotData, setMemoryHotspotData] = useState<HotspotRow[]>([])
  const [cpuSamplingRemaining, setCpuSamplingRemaining] = useState(0)
  const cpuSamplingTimer = useRef<ReturnType<typeof setInterval> | null>(null)

  const loadPprofStatus = useCallback(async () => {
    try {
      const res = await getPprofStatus()
      const data = (res.data || {}) as Record<string, unknown>
      setPprofStatus({
        enabled: !!data.enabled,
        is_developer: !!data.is_developer,
        token_required: !!data.token_required,
      })
      setPprofStatusLoaded(true)
    } catch (error) {
      if (getErrorResponse(error)?.status === 403) {
        setTabAccess((prev) => ({ ...prev, pprof: false }))
        setPprofStatusLoaded(true)
        setPprofStatus({ enabled: false, is_developer: false, token_required: false })
        return
      }
      setPprofStatusLoaded(true)
      setPprofStatus({ enabled: false, is_developer: false, token_required: false })
      handleViewRequestError(error)
    }
  }, [handleViewRequestError])

  const handlePprofApiError = useCallback(
    (error: unknown) => {
      const err = error as ApiError
      const response = getErrorResponse(error)
      const retryAfter = response?.data?.retry_after
      if (response?.status === 429 && retryAfter) {
        message.error(t('observability.pprof_rate_limited', { seconds: retryAfter }))
        return
      }
      message.error(response?.data?.message || err?.message || t('observability.pprof_unverified'))
    },
    [message, t],
  )

  const ensurePprofSamplingReady = useCallback(() => {
    if (!showPprofTab) return false
    if (pprofStatus.token_required) {
      if (!pprofToken) {
        message.warning(t('observability.pprof_token_placeholder'))
        return false
      }
      if (!pprofVerified) {
        message.warning(t('observability.pprof_verify'))
        return false
      }
    }
    return true
  }, [showPprofTab, pprofStatus.token_required, pprofToken, pprofVerified, message, t])

  const handleVerifyPprofToken = useCallback(async () => {
    if (!showPprofTab) return
    if (pprofStatus.token_required && !pprofToken) {
      message.warning(t('observability.pprof_token_placeholder'))
      return
    }
    setPprofVerifying(true)
    try {
      await verifyPprofToken({ token: pprofToken || '' })
      setPprofVerified(true)
      message.success(t('observability.pprof_verified'))
    } catch (error) {
      setPprofVerified(false)
      handlePprofApiError(error)
    } finally {
      setPprofVerifying(false)
    }
  }, [showPprofTab, pprofStatus.token_required, pprofToken, message, t, handlePprofApiError])

  const toCpuHotspotRows = (rows: Record<string, unknown>[]): HotspotRow[] =>
    rows.map((item, index) => ({
      key: index,
      function: String(item.function || '-'),
      flat_ms: `${((Number(item.flat_value || 0)) / 1e6).toFixed(2)} ms`,
      flat_percent: `${Number(item.flat_percent || 0).toFixed(2)}%`,
      cum_ms: `${((Number(item.cum_value || 0)) / 1e6).toFixed(2)} ms`,
      cum_percent: `${Number(item.cum_percent || 0).toFixed(2)}%`,
    }))

  const toMemoryHotspotRows = (rows: Record<string, unknown>[]): HotspotRow[] =>
    rows.map((item, index) => ({
      key: index,
      function: String(item.function || '-'),
      flat_bytes: formatHotspotBytes(Number(item.flat_value || 0)),
      flat_percent: `${Number(item.flat_percent || 0).toFixed(2)}%`,
      cum_bytes: formatHotspotBytes(Number(item.cum_value || 0)),
      cum_percent: `${Number(item.cum_percent || 0).toFixed(2)}%`,
    }))

  const loadCpuHotspots = useCallback(async () => {
    if (!ensurePprofSamplingReady()) return
    if (cpuHotspotLoading || memoryHotspotLoading) return
    setCpuHotspotLoading(true)
    setCpuSamplingRemaining(cpuHotspotForm.seconds)
    if (cpuSamplingTimer.current) clearInterval(cpuSamplingTimer.current)
    cpuSamplingTimer.current = setInterval(() => {
      setCpuSamplingRemaining((prev) => {
        if (prev <= 1) {
          if (cpuSamplingTimer.current) {
            clearInterval(cpuSamplingTimer.current)
            cpuSamplingTimer.current = null
          }
          return 0
        }
        return prev - 1
      })
    }, 1000)
    try {
      const res = await getPprofCpuHotspots({
        token: pprofToken || '',
        seconds: cpuHotspotForm.seconds,
        top_n: cpuHotspotForm.top_n,
      })
      const data = (res.data || {}) as Record<string, unknown>
      setCpuHotspotData(toCpuHotspotRows((data.list as Record<string, unknown>[]) || []))
    } catch (error) {
      handlePprofApiError(error)
    } finally {
      setCpuHotspotLoading(false)
      setCpuSamplingRemaining(0)
      if (cpuSamplingTimer.current) {
        clearInterval(cpuSamplingTimer.current)
        cpuSamplingTimer.current = null
      }
    }
  }, [
    ensurePprofSamplingReady,
    cpuHotspotLoading,
    memoryHotspotLoading,
    cpuHotspotForm,
    pprofToken,
    handlePprofApiError,
  ])

  const loadMemoryHotspots = useCallback(async () => {
    if (!ensurePprofSamplingReady()) return
    if (cpuHotspotLoading || memoryHotspotLoading) return
    setMemoryHotspotLoading(true)
    try {
      const res = await getPprofMemoryHotspots({
        token: pprofToken || '',
        top_n: memoryHotspotForm.top_n,
      })
      const data = (res.data || {}) as Record<string, unknown>
      setMemoryHotspotData(toMemoryHotspotRows((data.list as Record<string, unknown>[]) || []))
    } catch (error) {
      handlePprofApiError(error)
    } finally {
      setMemoryHotspotLoading(false)
    }
  }, [
    ensurePprofSamplingReady,
    cpuHotspotLoading,
    memoryHotspotLoading,
    memoryHotspotForm,
    pprofToken,
    handlePprofApiError,
  ])

  const openPprofPath = useCallback(
    (path: string) => {
      if (!showPprofTab) return
      if (pprofStatus.token_required) {
        if (!pprofToken) {
          message.warning(t('observability.pprof_token_placeholder'))
          return
        }
        if (!pprofVerified) {
          message.warning(t('observability.pprof_verify'))
          return
        }
      }
      const apiBaseURL = import.meta.env.VITE_API_BASE_URL as string | undefined
      const backendOrigin = apiBaseURL ? apiBaseURL.replace(/\/+$/, '') : window.location.origin
      const url = new URL(path, backendOrigin)
      if (pprofToken) url.searchParams.set('token', pprofToken)
      window.open(url.toString(), '_blank', 'noopener,noreferrer')
    },
    [showPprofTab, pprofStatus.token_required, pprofToken, pprofVerified, message, t],
  )

  useEffect(() => {
    if (tabAccess.slowSql) loadSlowSql()
    if (tabAccess.apiPerformance) loadApiPerformance()
    if (tabAccess.audit) loadAudit()
    if (tabAccess.queue) loadQueue()
    if (tabAccess.pprof) loadPprofStatus()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(
    () => () => {
      if (cpuSamplingTimer.current) {
        clearInterval(cpuSamplingTimer.current)
        cpuSamplingTimer.current = null
      }
    },
    [],
  )

  const searchRowStyle: React.CSSProperties = {
    display: 'flex',
    flexWrap: 'wrap',
    alignItems: 'center',
    gap: 12,
    marginBottom: 16,
  }

  const apiEndpointColumns = (showDuration = true): ColumnsType<Record<string, unknown>> => [
    { title: t('log.method'), dataIndex: 'method', width: 90 },
    { title: t('observability.api_route_template'), dataIndex: 'route_template', ellipsis: true },
    ...(showDuration
      ? [
          {
            title: t('observability.p95_ms'),
            dataIndex: 'p95_duration_ms',
            width: 120,
            render: (v: number) => Number(v || 0).toFixed(2),
          },
          {
            title: t('observability.p99_ms'),
            dataIndex: 'p99_duration_ms',
            width: 120,
            render: (v: number) => Number(v || 0).toFixed(2),
          },
          {
            title: t('observability.avg_ms'),
            dataIndex: 'avg_duration_ms',
            width: 120,
            render: (v: number) => Number(v || 0).toFixed(2),
          },
          {
            title: t('observability.max_ms'),
            dataIndex: 'max_duration_ms',
            width: 120,
            render: (v: number) => Number(v || 0).toFixed(2),
          },
        ]
      : []),
    {
      title: t('common.count'),
      dataIndex: 'count',
      width: 100,
    },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 140,
      render: (_, row) => (
        <Button type="link" onClick={() => loadApiPerformanceTraces(row)}>
          {t('observability.view_traces')}
        </Button>
      ),
    },
  ]

  const tabItems = [
    tabAccess.queue && {
      key: 'queue',
      label: t('observability.queue_tab'),
      children: (
        <>
          <Alert type="info" showIcon message={t('observability.queue_dashboard_hint')} style={{ marginBottom: 12 }} />
          <Card size="small" style={{ marginBottom: 12 }} title={t('observability.queue_alert_title')}>
            <Alert type="warning" showIcon message={t('observability.queue_alert_hint')} style={{ marginBottom: 12 }} />
            <Space wrap>
              <Tag color={queueDashboard.queue_alert?.configured ? 'success' : 'default'}>
                {queueDashboard.queue_alert?.configured
                  ? t('observability.queue_alert_configured', {
                      source: queueDashboard.queue_alert.source || '-',
                    })
                  : t('observability.queue_alert_not_configured')}
              </Tag>
              {queueDashboard.queue_alert?.configured ? (
                <>
                  <Typography.Text type="secondary">
                    {t('observability.queue_alert_threshold', {
                      n: queueDashboard.queue_alert.threshold ?? 100,
                    })}
                  </Typography.Text>
                  <Typography.Text type="secondary">
                    {t('observability.queue_alert_url', {
                      url: queueDashboard.queue_alert.url_masked || '***',
                    })}
                  </Typography.Text>
                </>
              ) : null}
              {canTestQueueAlert ? (
                <Button
                  loading={queueAlertTesting}
                  disabled={!queueDashboard.queue_alert?.configured}
                  onClick={() => void onTestQueueAlert()}
                >
                  {t('observability.queue_alert_test')}
                </Button>
              ) : null}
            </Space>
          </Card>
          <div style={searchRowStyle}>
            <span style={{ flex: 1 }}>
              {t('observability.queue_default')}:{' '}
              <Typography.Text code>{queueDashboard.default_connection || '-'}</Typography.Text>
            </span>
            <Button type="primary" loading={queueLoading} onClick={loadQueue}>
              {t('common.refresh')}
            </Button>
          </div>
          {currentQueuePanel ? (
            <Card
              size="small"
              title={
                <Space>
                  <strong>{currentQueuePanel.connection}</strong>
                  <Tag color={QUEUE_KIND_COLORS[currentQueuePanel.kind || ''] || 'default'}>
                    {t(`observability.queue_kind_${currentQueuePanel.kind || 'other'}`)}
                  </Tag>
                </Space>
              }
            >
              <Descriptions bordered size="small" column={4} style={{ marginBottom: 12 }}>
                <Descriptions.Item label={t('observability.queue_driver')}>
                  {currentQueuePanel.driver_raw || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('observability.queue_redis_client')}>
                  {currentQueuePanel.redis_client || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('observability.queue_consumer_group')}>
                  {currentQueuePanel.kind === 'redis_stream' ? currentQueuePanel.consumer_group || '-' : '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('observability.queue_default_queue')}>
                  {currentQueuePanel.default_queue || '-'}
                </Descriptions.Item>
              </Descriptions>
              {currentQueueRows.length ? (
                <Table
                  size="small"
                  bordered
                  pagination={false}
                  rowKey="name"
                  dataSource={currentQueueRows}
                  columns={[
                    {
                      title: t('observability.queue_name'),
                      dataIndex: 'name',
                      width: 180,
                      render: (v) => <Typography.Text code>{v}</Typography.Text>,
                    },
                    { title: t('observability.metric_pending'), dataIndex: 'pending', width: 100 },
                    { title: t('observability.metric_reserved'), dataIndex: 'reserved', width: 100 },
                    { title: t('observability.metric_delayed'), dataIndex: 'delayed', width: 100 },
                    { title: t('observability.metric_failed'), dataIndex: 'failed', width: 100 },
                    { title: t('observability.metric_total'), dataIndex: 'total', width: 100 },
                    ...(currentQueuePanel.kind === 'redis_stream'
                      ? [
                          {
                            title: t('observability.metric_stream_total'),
                            dataIndex: 'stream_total',
                            width: 150,
                            render: (v: number | null) => (v == null ? '—' : v),
                          },
                        ]
                      : []),
                  ]}
                />
              ) : (
                <Empty description={t('observability.queue_no_data')} />
              )}
              {currentQueuePanel.fetch_error ? (
                <Typography.Text type="danger" style={{ display: 'block', marginTop: 8 }}>
                  {currentQueuePanel.fetch_error}
                </Typography.Text>
              ) : null}
            </Card>
          ) : (
            <Empty description={t('observability.queue_panel_missing')} />
          )}
        </>
      ),
    },
    tabAccess.trace && {
      key: 'trace',
      label: t('observability.trace_tab'),
      children: (
        <>
          <div style={searchRowStyle}>
            <Input
              value={traceId}
              onChange={(e) => setTraceId(e.target.value)}
              placeholder={t('observability.trace_id_placeholder')}
              allowClear
              style={{ maxWidth: 360 }}
            />
            <Button type="primary" loading={traceLoading} onClick={loadTrace}>
              {t('common.search')}
            </Button>
          </div>
          {!traceLoading && !traceData.trace_id ? (
            <Empty description={t('observability.trace_empty')} />
          ) : traceData.trace_id ? (
            <>
              <Descriptions bordered column={2} size="small">
                <Descriptions.Item label={t('log.trace_id')}>{traceData.trace_id}</Descriptions.Item>
                <Descriptions.Item label={t('observability.trace_request')}>
                  {traceData.request?.method} {traceData.request?.path}
                </Descriptions.Item>
              </Descriptions>
              <Divider titlePlacement="start">{t('observability.trace_operations')}</Divider>
              <Table
                size="small"
                bordered
                pagination={false}
                rowKey={(_, i) => `op-${i}`}
                dataSource={traceData.operations || []}
                columns={[
                  { title: t('log.time'), dataIndex: 'created_at', width: 180 },
                  { title: t('log.method'), dataIndex: 'method', width: 90 },
                  { title: t('log.path'), dataIndex: 'path', ellipsis: true },
                  { title: t('observability.duration_ms'), dataIndex: 'duration', width: 120 },
                  { title: t('log.title'), dataIndex: 'title', ellipsis: true },
                ]}
              />
              <Divider titlePlacement="start">{t('observability.trace_exceptions')}</Divider>
              <Table
                size="small"
                bordered
                pagination={false}
                rowKey={(_, i) => `ex-${i}`}
                dataSource={traceData.exceptions || []}
                columns={[
                  { title: t('log.time'), dataIndex: 'created_at', width: 180 },
                  { title: t('log.level'), dataIndex: 'level', width: 100 },
                  { title: t('log.module'), dataIndex: 'module', width: 140 },
                  { title: t('log.message'), dataIndex: 'message', ellipsis: true },
                ]}
              />
            </>
          ) : null}
        </>
      ),
    },
    tabAccess.audit && {
      key: 'audit',
      label: t('observability.audit_tab'),
      children: (
        <>
          <div style={searchRowStyle}>
            <Input
              value={auditQuery.trace_id}
              onChange={(e) => setAuditQuery((q) => ({ ...q, trace_id: e.target.value }))}
              placeholder={t('observability.trace_id_placeholder')}
              allowClear
              style={{ maxWidth: 280 }}
            />
            <Input
              value={auditQuery.keyword}
              onChange={(e) => setAuditQuery((q) => ({ ...q, keyword: e.target.value }))}
              placeholder={t('observability.keyword_placeholder')}
              allowClear
              style={{ maxWidth: 320 }}
            />
            <Button type="primary" loading={auditLoading} onClick={() => loadAudit()}>
              {t('common.search')}
            </Button>
          </div>
          <Table
            size="small"
            bordered
            rowKey={(_, i) => `audit-${i}`}
            loading={auditLoading}
            dataSource={auditData.list}
            columns={[
              { title: t('log.time'), dataIndex: 'time', width: 180 },
              { title: t('observability.event_type'), dataIndex: 'type', width: 110 },
              { title: t('log.trace_id'), dataIndex: 'trace_id', width: 220, ellipsis: true },
              { title: t('log.module'), dataIndex: 'module', width: 130 },
              { title: t('log.message'), dataIndex: 'message', ellipsis: true },
            ]}
            pagination={{
              current: auditQuery.page,
              pageSize: auditQuery.page_size,
              total: auditData.total,
              showTotal: (total) => t('common.total', { total }),
              onChange: (page, pageSize) => {
                setAuditQuery((q) => ({ ...q, page, page_size: pageSize }))
                setAuditLoading(true)
                getAuditTimeline({ ...auditQuery, page, page_size: pageSize })
                  .then((res) => {
                    const data = (res.data || {}) as Record<string, unknown>
                    setAuditData({
                      list: (data.list as Record<string, unknown>[]) || [],
                      total: Number(data.total || 0),
                    })
                  })
                  .catch(handleViewRequestError)
                  .finally(() => setAuditLoading(false))
              },
            }}
          />
        </>
      ),
    },
    tabAccess.slowSql && {
      key: 'slowSql',
      label: t('observability.slow_sql_tab'),
      children: (
        <>
          <div style={searchRowStyle}>
            <span>{t('observability.slow_sql_hours')}</span>
            <InputNumber
              min={1}
              max={168}
              value={slowSqlQuery.hours}
              onChange={(v) => setSlowSqlQuery((q) => ({ ...q, hours: v ?? 24 }))}
            />
            <span>{t('observability.slow_sql_min_duration_ms')}</span>
            <InputNumber
              min={1}
              max={10000}
              value={slowSqlQuery.min_duration_ms}
              onChange={(v) => setSlowSqlQuery((q) => ({ ...q, min_duration_ms: v ?? 200 }))}
            />
            <span>{t('observability.slow_sql_limit')}</span>
            <InputNumber
              min={1}
              max={100}
              value={slowSqlQuery.limit}
              onChange={(v) => setSlowSqlQuery((q) => ({ ...q, limit: v ?? 20 }))}
            />
            <Button type="primary" loading={slowSqlLoading} onClick={loadSlowSql}>
              {t('common.search')}
            </Button>
          </div>
          <Table
            size="small"
            bordered
            rowKey={(_, i) => `sql-${i}`}
            loading={slowSqlLoading}
            dataSource={slowSqlData}
            columns={[
              {
                title: t('observability.max_ms'),
                dataIndex: 'max_duration_ms',
                width: 120,
                render: (v) => Number(v || 0).toFixed(2),
              },
              {
                title: t('observability.avg_ms'),
                dataIndex: 'avg_duration_ms',
                width: 120,
                render: (v) => Number(v || 0).toFixed(2),
              },
              { title: t('common.count'), dataIndex: 'count', width: 90 },
              {
                title: t('observability.normalized_sql'),
                dataIndex: 'normalized_sql',
                ellipsis: true,
                render: (v) =>
                  v ? (
                    <Tooltip title={String(v)}>
                      <Typography.Text code ellipsis style={{ maxWidth: 400 }}>
                        {String(v)}
                      </Typography.Text>
                    </Tooltip>
                  ) : (
                    '-'
                  ),
              },
              { title: t('observability.last_seen'), dataIndex: 'last_seen_at', width: 180 },
            ]}
          />
        </>
      ),
    },
    tabAccess.apiPerformance && {
      key: 'apiPerformance',
      label: t('observability.api_performance_tab'),
      children: (
        <>
          <div style={searchRowStyle}>
            <span>{t('observability.api_perf_hours')}</span>
            <InputNumber
              min={1}
              max={168}
              value={apiPerfQuery.hours}
              onChange={(v) => setApiPerfQuery((q) => ({ ...q, hours: v ?? 24 }))}
            />
            <span>{t('observability.api_perf_limit')}</span>
            <InputNumber
              min={1}
              max={100}
              value={apiPerfQuery.limit}
              onChange={(v) => setApiPerfQuery((q) => ({ ...q, limit: v ?? 20 }))}
            />
            <Button type="primary" loading={apiPerfLoading} onClick={loadApiPerformance}>
              {t('common.search')}
            </Button>
          </div>
          <Divider titlePlacement="start">{t('observability.api_perf_slow_top')}</Divider>
          <Table
            size="small"
            bordered
            pagination={false}
            rowKey={(_, i) => `slow-${i}`}
            dataSource={apiPerfData.slow_top}
            columns={apiEndpointColumns(true)}
          />
          <Divider titlePlacement="start">{t('observability.api_perf_error_top')}</Divider>
          <Table
            size="small"
            bordered
            pagination={false}
            rowKey={(_, i) => `err-${i}`}
            dataSource={apiPerfData.error_top}
            columns={[
              { title: t('log.method'), dataIndex: 'method', width: 90 },
              { title: t('observability.api_route_template'), dataIndex: 'route_template', ellipsis: true },
              {
                title: t('observability.error_rate'),
                dataIndex: 'error_rate',
                width: 120,
                render: (v) => `${(Number(v || 0) * 100).toFixed(2)}%`,
              },
              { title: t('observability.error_count'), dataIndex: 'error_count', width: 120 },
              { title: t('common.count'), dataIndex: 'count', width: 100 },
              {
                title: t('common.operation'),
                key: 'operation',
                width: 140,
                render: (_, row) => (
                  <Button type="link" onClick={() => loadApiPerformanceTraces(row)}>
                    {t('observability.view_traces')}
                  </Button>
                ),
              },
            ]}
          />
          <Divider titlePlacement="start">{t('observability.api_perf_qps_top')}</Divider>
          <Table
            size="small"
            bordered
            pagination={false}
            rowKey={(_, i) => `qps-${i}`}
            dataSource={apiPerfData.qps_top}
            columns={[
              { title: t('log.method'), dataIndex: 'method', width: 90 },
              { title: t('observability.api_route_template'), dataIndex: 'route_template', ellipsis: true },
              {
                title: t('observability.qps'),
                dataIndex: 'qps',
                width: 120,
                render: (v) => Number(v || 0).toFixed(4),
              },
              { title: t('common.count'), dataIndex: 'count', width: 100 },
              {
                title: t('common.operation'),
                key: 'operation',
                width: 140,
                render: (_, row) => (
                  <Button type="link" onClick={() => loadApiPerformanceTraces(row)}>
                    {t('observability.view_traces')}
                  </Button>
                ),
              },
            ]}
          />
          <Divider titlePlacement="start">{t('observability.api_perf_trace_drilldown')}</Divider>
          {selectedApiEndpoint.route_template ? (
            <Alert
              type="info"
              showIcon
              message={`${selectedApiEndpoint.method} ${selectedApiEndpoint.route_template}`}
              style={{ marginBottom: 12 }}
            />
          ) : null}
          <Table
            size="small"
            bordered
            loading={apiPerfTraceLoading}
            rowKey={(_, i) => `trace-${i}`}
            dataSource={apiPerfTraceData}
            columns={[
              { title: t('log.time'), dataIndex: 'occurred_at', width: 180 },
              {
                title: t('log.trace_id'),
                dataIndex: 'trace_id',
                ellipsis: true,
                render: (v) =>
                  v ? (
                    <Button type="link" onClick={() => jumpToTrace(String(v))}>
                      {String(v)}
                    </Button>
                  ) : (
                    '-'
                  ),
              },
              { title: t('log.status_code'), dataIndex: 'status_code', width: 110 },
              { title: t('observability.duration_ms'), dataIndex: 'duration_ms', width: 130 },
            ]}
          />
        </>
      ),
    },
    tabAccess.pprof &&
      showPprofTab && {
        key: 'pprof',
        label: t('observability.pprof_tab'),
        children: (
          <>
            {pprofStatus.token_required ? (
              <Alert type="warning" showIcon message={t('observability.pprof_hint')} style={{ marginBottom: 12 }} />
            ) : null}
            {pprofStatus.token_required ? (
              <div style={searchRowStyle}>
                <Input.Password
                  value={pprofToken}
                  onChange={(e) => setPprofToken(e.target.value)}
                  placeholder={t('observability.pprof_token_placeholder')}
                  style={{ maxWidth: 320 }}
                />
                <Button type="primary" loading={pprofVerifying} onClick={handleVerifyPprofToken}>
                  {t('observability.pprof_verify')}
                </Button>
                <Tag color={pprofVerified ? 'success' : 'default'}>
                  {pprofVerified ? t('observability.pprof_verified') : t('observability.pprof_unverified')}
                </Tag>
              </div>
            ) : null}
            <Space wrap style={{ marginBottom: 16 }}>
              <Button type="primary" onClick={() => openPprofPath('/debug/pprof/')}>
                Index
              </Button>
              <Button onClick={() => openPprofPath('/debug/pprof/heap')}>Heap</Button>
              <Button onClick={() => openPprofPath('/debug/pprof/goroutine')}>Goroutine</Button>
              <Button onClick={() => openPprofPath('/debug/pprof/allocs')}>Allocs</Button>
              <Button onClick={() => openPprofPath('/debug/pprof/profile')}>CPU Profile</Button>
              <Button onClick={() => openPprofPath('/debug/pprof/trace')}>Trace</Button>
              <Button onClick={() => openPprofPath('/debug/pprof/runtime')}>Runtime</Button>
            </Space>
            <Divider titlePlacement="start">{t('observability.cpu_hotspots_title')}</Divider>
            <div style={searchRowStyle}>
              <span>{t('observability.pprof_sample_seconds')}</span>
              <InputNumber
                min={1}
                max={120}
                value={cpuHotspotForm.seconds}
                onChange={(v) => setCpuHotspotForm((f) => ({ ...f, seconds: v ?? 3 }))}
              />
              <span>{t('observability.pprof_top_n')}</span>
              <InputNumber
                min={1}
                max={100}
                value={cpuHotspotForm.top_n}
                onChange={(v) => setCpuHotspotForm((f) => ({ ...f, top_n: v ?? 15 }))}
              />
              <Button
                type="primary"
                loading={cpuHotspotLoading}
                disabled={memoryHotspotLoading}
                onClick={loadCpuHotspots}
              >
                {t('observability.pprof_start_sampling')}
              </Button>
              {cpuHotspotLoading ? (
                <Tag color="warning">
                  {t('observability.pprof_sampling_in_progress', { seconds: cpuSamplingRemaining })}
                </Tag>
              ) : null}
            </div>
            <Table
              size="small"
              bordered
              pagination={false}
              rowKey="key"
              dataSource={cpuHotspotData}
              columns={[
                { title: t('common.sort'), width: 70, render: (_, __, i) => i + 1 },
                { title: t('observability.pprof_function'), dataIndex: 'function', ellipsis: true },
                { title: t('observability.pprof_cpu_self_ms'), dataIndex: 'flat_ms', width: 150 },
                { title: t('observability.pprof_self_percent'), dataIndex: 'flat_percent', width: 120 },
                { title: t('observability.pprof_cpu_cum_ms'), dataIndex: 'cum_ms', width: 150 },
                { title: t('observability.pprof_cum_percent'), dataIndex: 'cum_percent', width: 120 },
              ]}
            />
            <Divider titlePlacement="start">{t('observability.memory_hotspots_title')}</Divider>
            <div style={searchRowStyle}>
              <span>{t('observability.pprof_top_n')}</span>
              <InputNumber
                min={1}
                max={100}
                value={memoryHotspotForm.top_n}
                onChange={(v) => setMemoryHotspotForm((f) => ({ ...f, top_n: v ?? 25 }))}
              />
              <Button
                type="primary"
                loading={memoryHotspotLoading}
                disabled={cpuHotspotLoading}
                onClick={loadMemoryHotspots}
              >
                {t('common.refresh')}
              </Button>
            </div>
            <Table
              size="small"
              bordered
              pagination={false}
              rowKey="key"
              dataSource={memoryHotspotData}
              columns={[
                { title: t('common.sort'), width: 70, render: (_, __, i) => i + 1 },
                { title: t('observability.pprof_function'), dataIndex: 'function', ellipsis: true },
                { title: t('observability.pprof_mem_flat_bytes'), dataIndex: 'flat_bytes', width: 170 },
                { title: t('observability.pprof_self_percent'), dataIndex: 'flat_percent', width: 120 },
                { title: t('observability.pprof_mem_cum_bytes'), dataIndex: 'cum_bytes', width: 170 },
                { title: t('observability.pprof_cum_percent'), dataIndex: 'cum_percent', width: 120 },
              ]}
            />
          </>
        ),
      },
  ].filter(Boolean) as { key: TabKey; label: string; children: React.ReactNode }[]

  return (
    <PageContainer title={t('menu.observability')}>
      <Card>
        {tabItems.length ? (
          <Tabs activeKey={activeTab} onChange={(k) => setActiveTab(k as TabKey)} items={tabItems} />
        ) : (
          <Empty description={t('common.no_data')} />
        )}
      </Card>
    </PageContainer>
  )
}
