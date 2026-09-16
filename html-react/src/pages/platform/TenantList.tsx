import { useEffect, useMemo, useState, type FocusEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
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
  Segmented,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Timeline,
  Tooltip,
  Typography,
  Badge,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { SettingOutlined } from '@ant-design/icons'
import {
  backupPlatformTenant,
  createPlatformTenant,
  deletePlatformTenant,
  forceDeletePlatformTenant,
  downloadPlatformTenantBackup,
  exportPlatformTenants,
  getPlatformTenantList,
  getPlatformTenantLoginLinks,
  getPlatformTenantDomains,
  createPlatformTenantDomain,
  verifyPlatformTenantDomain,
  setPrimaryPlatformTenantDomain,
  disablePlatformTenantDomain,
  deletePlatformTenantDomain,
  getPlatformTenantOpLogs,
  getPlatformTenantOpsSummary,
  getPlatformTenantOverview,
  getPlatformTenantSettings,
  listPlatformTenantBackups,
  migratePlatformTenant,
  migratePlatformTenantBatch,
  opsPlatformTenantBatch,
  pingPlatformTenant,
  platformHealth,
  prunePlatformTenantBackups,
  purgePlatformTenant,
  restorePlatformTenant,
  seedPlatformTenant,
  undeletePlatformTenant,
  updatePlatformTenant,
  updatePlatformTenantMaintenance,
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
import { isPlatformOwner } from '@/utils/platformRequest'

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
  schema_status?: string
  schema_migration_count?: number
  expected_migration_count?: number
  maintenance?: boolean
  maintenance_message?: string
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
  deleted_at?: string
  trashed?: boolean
  storage_limit_bytes?: number
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

const MB = 1024 * 1024
function mbToBytes(m?: number) {
  return Math.round((Number(m) || 0) * MB)
}
function bytesToMb(b?: number) {
  const n = Number(b) || 0
  if (n <= 0) return 0
  return Math.round(n / MB)
}
function formatQuota(used?: number, limit?: number) {
  const u = formatSize(used)
  const lim = Number(limit) || 0
  if (lim <= 0) return `${u} / ∞`
  return `${u} / ${formatSize(lim)}`
}

function backupDirOf(row: TenantRow) {
  if (row.backup_dir) return row.backup_dir
  if (row.code) return `storage/backups/tenants/${row.code}`
  return ''
}

interface TenantPingDetail {
  ok?: boolean
  latency_ms?: number
  host?: string
  port?: number
  database?: string
  error?: string
  last_op?: string
  last_op_status?: string
  last_op_message?: string
  last_migrate_error?: string
}

interface TenantOverviewData {
  database?: string
  table_count?: number
  database_bytes?: number
  admins_count?: number
  migrations_count?: number
  ping_ok?: boolean
  ping_ms?: number
  error?: string
}

interface TenantQuotaData {
  storage_limit_bytes?: number
  storage_used_bytes?: number


}

interface TenantOpLogRow {
  id?: number
  op?: string
  status?: string
  message?: string
  batch_id?: string
  operator_name?: string
  started_at?: string
  finished_at?: string
  created_at?: string
}

interface TenantLoginLinksData {
  query_url?: string
  hint?: string
  header?: string
  resolver?: string
  tenant_code?: string
  subdomain_url?: string
  primary_custom_url?: string
  active_custom_hosts?: string[]
}

interface TenantDomainRow {
  id: number
  host: string
  status: string
  ssl_mode: string
  is_primary?: boolean
  verify_token?: string
  last_check_error?: string
  dns_guide?: {
    txt_name?: string
    txt_value?: string
    domain_target?: string
    instruction?: string
  }
}

interface PlatformQueueStatus {
  connection?: string
  queue?: string
  pending?: number
  available?: boolean
  message?: string
}

export default function PlatformTenantList() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const isOwner = isPlatformOwner()
  const [createOpen, setCreateOpen] = useState(false)
  const [editing, setEditing] = useState<TenantRow | null>(null)
  const [migrateTarget, setMigrateTarget] = useState<TenantRow | null>(null)
  const [passwordTouched, setPasswordTouched] = useState(false)
  const [withSeed, setWithSeed] = useState(true)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()

  const unlockCredentialAutofill = (e: FocusEvent<HTMLInputElement>) => {
    e.currentTarget.removeAttribute('readonly')
  }
  const [healthDesc, setHealthDesc] = useState(t('platform.cli_ops_hint'))
  const [opsSummary, setOpsSummary] = useState<{
    failed_provision?: number
    busy?: number
    total?: number
    deleted?: number
    failed_purge?: number
    schema_aligned?: number
    schema_behind?: number
    schema_failed?: number
    schema_unknown?: number
    maintenance?: number
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
  const [backupKeep, setBackupKeep] = useState(10)
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([])
  const [exportLoading, setExportLoading] = useState(false)
  const [pingDetail, setPingDetail] = useState<TenantPingDetail | null>(null)
  const [pingRow, setPingRow] = useState<TenantRow | null>(null)
  const [detailExtraLoading, setDetailExtraLoading] = useState(false)
  const [detailOverview, setDetailOverview] = useState<TenantOverviewData | null>(null)
  const [detailQuota, setDetailQuota] = useState<TenantQuotaData | null>(null)
  const [detailOpLogs, setDetailOpLogs] = useState<TenantOpLogRow[]>([])
  const [detailLoginLinks, setDetailLoginLinks] = useState<TenantLoginLinksData | null>(null)
  const [detailDomains, setDetailDomains] = useState<TenantDomainRow[]>([])
  const [domainHost, setDomainHost] = useState('')
  const [domainSslMode, setDomainSslMode] = useState<'edge' | 'customer_cdn'>('edge')
  const [domainBusy, setDomainBusy] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<TenantRow | null>(null)
  const [deleteConfirmCode, setDeleteConfirmCode] = useState('')
  const [deleteDropDb, setDeleteDropDb] = useState(false)
  const [deleteLoading, setDeleteLoading] = useState(false)
  const [purgeTarget, setPurgeTarget] = useState<TenantRow | null>(null)
  const [purgeObjects, setPurgeObjects] = useState(true)
  const [purgeBackups, setPurgeBackups] = useState(true)
  const [purgeLoading, setPurgeLoading] = useState(false)
  const [forceDeleteTarget, setForceDeleteTarget] = useState<TenantRow | null>(null)
  const [forceDeleteConfirm, setForceDeleteConfirm] = useState('')
  const [forcePurgeObjects, setForcePurgeObjects] = useState(true)
  const [forcePurgeBackups, setForcePurgeBackups] = useState(true)
  const [forceDeleteLoading, setForceDeleteLoading] = useState(false)
  const [pruneKeep, setPruneKeep] = useState(10)
  const [pruneLoading, setPruneLoading] = useState(false)
  const [restoringName, setRestoringName] = useState('')

  const formatQueueHint = (q: PlatformQueueStatus | null, keep: number) => {
    if (!q) return ''
    const pending = Number(q.pending ?? 0)
    const parts = [t('platform.health_backup_keep', { n: keep })]
    if (q.available) {
      parts.push(t('tenant.queue_pending', { n: pending, queue: q.queue || 'long-running' }))
    }
    if (q.message) parts.push(String(q.message))
    return parts.join(' · ')
  }

  const refreshOpsSummary = async () => {
    try {
      const res = await getPlatformTenantOpsSummary()
      const summary = (res as { data?: { summary?: Record<string, unknown> } })?.data?.summary
      if (summary) {
        setOpsSummary({
          failed_provision: Number(summary.failed_provision ?? 0),
          busy: Number(summary.busy ?? 0),
          total: Number(summary.total ?? 0),
          deleted: Number(summary.deleted ?? 0),
          failed_purge: Number(summary.failed_purge ?? 0),
          schema_aligned: Number(summary.schema_aligned ?? 0),
          schema_behind: Number(summary.schema_behind ?? 0),
          schema_failed: Number(summary.schema_failed ?? 0),
          schema_unknown: Number(summary.schema_unknown ?? 0),
          maintenance: Number(summary.maintenance ?? 0),
        })
      }
    } catch {
      /* optional */
    }
  }

  const refreshHealthBanner = async () => {
    try {
      const res = await platformHealth()
      const h = (res as { data?: Record<string, unknown> })?.data
      if (!h) return
      const tenants = (h.tenants || {}) as { active?: number; total?: number }
      const db = h.database_ok ? t('platform.health_db_ok') : t('platform.health_db_bad')
      const keep = Number(h.backup_keep ?? backupKeep)
      if (h.backup_keep != null) setBackupKeep(keep)
      const q = (h.queue || null) as PlatformQueueStatus | null
      setHealthDesc(
        `${t('platform.health_driver')}: ${String(h.driver)} · ${db} · ${t('platform.health_tenants', {
          active: tenants.active ?? 0,
          total: tenants.total ?? 0,
        })} · ${formatQueueHint(q, keep)} · ${t('platform.cli_ops_hint')}`,
      )
    } catch {
      /* list still usable */
    }
  }

  useEffect(() => {
    void refreshHealthBanner()
    void getPlatformTenantSettings()
      .then((res) => {
        const data = (res as { data?: { backup_keep?: number; queue?: PlatformQueueStatus } })?.data
        if (data?.backup_keep != null) {
          setBackupKeep(Number(data.backup_keep))
          setPruneKeep(Number(data.backup_keep))
        }
      })
      .catch(() => {
        /* optional */
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
    handleSortChange,
    refresh,
  } = useListPage<TenantRow>({
    fetchApi: getPlatformTenantList,
    initialSearchForm: { code: '', name: '', status: '', provision_status: '', schema_status: '', maintenance: '', trashed: '' },
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
        schema_status: String(entityField(record, 'schema_status', '') ?? ''),
        schema_migration_count: Number(entityField(record, 'schema_migration_count', 0) ?? 0),
        expected_migration_count: Number(entityField(record, 'expected_migration_count', 0) ?? 0),
        maintenance: Boolean(entityField(record, 'maintenance', false)),
        maintenance_message: String(entityField(record, 'maintenance_message', '') ?? ''),
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
        deleted_at: String(entityField(record, 'deleted_at', '') ?? ''),
        trashed: Boolean(entityField(record, 'trashed', false)),
        storage_limit_bytes: Number(entityField(record, 'storage_limit_bytes', 0) ?? 0),
      }
    },
  })

  const isRecycleView = searchForm.trashed === 'only'

  useEffect(() => {
    const schemaFromQuery = String(searchParams.get('schema_status') || '').trim()
    const maintenanceFromQuery = String(searchParams.get('maintenance') || '').trim()
    if (!schemaFromQuery && !maintenanceFromQuery) return
    const next = { ...searchForm }
    let changed = false
    if (schemaFromQuery && String(searchForm.schema_status || '') !== schemaFromQuery) {
      next.schema_status = schemaFromQuery
      changed = true
    }
    if (maintenanceFromQuery && String(searchForm.maintenance || '') !== maintenanceFromQuery) {
      next.maintenance = maintenanceFromQuery
      changed = true
    }
    if (!changed) return
    onSearchFormChange(next)
    window.setTimeout(() => handleSearch(), 0)
    // Apply once from URL when landing from overview shortcuts.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams])

  const setListMode = (mode: 'active' | 'recycle') => {
    const next = mode === 'recycle' ? 'only' : ''
    onSearchFormChange({ ...searchForm, trashed: next })
    window.setTimeout(() => handleSearch(), 0)
  }

  const recycleOpTag = (row: TenantRow) => {
    if (!isRecycleView) return null
    if (row.last_op === 'purge') {
      if (row.last_op_status === 'queued' || row.last_op_status === 'running') {
        return { color: 'gold', label: t('tenant.purge_status_running') }
      }
      if (row.last_op_status === 'failed') {
        return { color: 'red', label: t('tenant.purge_status_failed') }
      }
      if (row.last_op_status === 'success') {
        return { color: 'green', label: t('tenant.purge_status_done') }
      }
    }
    if (row.last_op_status === 'queued' || row.last_op_status === 'running') {
      return { color: 'gold', label: t('tenant.op_busy') }
    }
    return null
  }

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
      const data = (res as {
        data?: { list?: typeof backupsList; backup_dir?: string; backup_keep?: number }
      })?.data
      setBackupsList(data?.list || [])
      setBackupsDir(data?.backup_dir || row.backup_dir || backupDirOf(row))
      if (data?.backup_keep != null) {
        setBackupKeep(Number(data.backup_keep))
        setPruneKeep(Number(data.backup_keep))
      }
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
      const blob = (await downloadPlatformTenantBackup(backupsRow.id, file.name)) as unknown as Blob
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

  const loadDetailExtra = async (row: TenantRow) => {
    setDetailExtraLoading(true)
    setDetailOverview(null)
    setDetailQuota(null)
    setDetailOpLogs([])
    setDetailLoginLinks(null)
    setDetailDomains([])
    setDomainHost('')
    try {
      const [overviewRes, logsRes, linksRes, domainsRes] = await Promise.all([
        getPlatformTenantOverview(row.id),
        getPlatformTenantOpLogs(row.id, { limit: 40 }),
        getPlatformTenantLoginLinks(row.id),
        getPlatformTenantDomains(row.id),
      ])
      setDetailOverview(
        (overviewRes as { data?: { overview?: TenantOverviewData } })?.data?.overview || null,
      )
      setDetailQuota(
        (overviewRes as { data?: { quota?: TenantQuotaData } })?.data?.quota || null,
      )
      setDetailOpLogs((logsRes as { data?: { list?: TenantOpLogRow[] } })?.data?.list || [])
      setDetailLoginLinks(
        (linksRes as { data?: { links?: TenantLoginLinksData } })?.data?.links || null,
      )
      setDetailDomains(
        (domainsRes as { data?: { list?: TenantDomainRow[] } })?.data?.list || [],
      )
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setDetailExtraLoading(false)
    }
  }

  const openDetail = (row: TenantRow) => {
    setDetailRow(row)
    void loadDetailExtra(row)
  }

  const closeDetail = () => {
    setDetailRow(null)
    setDetailOverview(null)
    setDetailQuota(null)
    setDetailOpLogs([])
    setDetailLoginLinks(null)
    setDetailDomains([])
    setDomainHost('')
  }

  const reloadDetailDomains = async (tenantId: string | number) => {
    try {
      const domainsRes = await getPlatformTenantDomains(tenantId)
      setDetailDomains(
        (domainsRes as { data?: { list?: TenantDomainRow[] } })?.data?.list || [],
      )
      const linksRes = await getPlatformTenantLoginLinks(tenantId)
      setDetailLoginLinks(
        (linksRes as { data?: { links?: TenantLoginLinksData } })?.data?.links || null,
      )
    } catch (error) {
      showError(error, t('common.operation_failed'))
    }
  }

  const onAddDomain = async () => {
    if (!detailRow || !domainHost.trim()) return
    setDomainBusy(true)
    try {
      await createPlatformTenantDomain(detailRow.id, {
        host: domainHost.trim(),
        ssl_mode: domainSslMode,
      })
      setDomainHost('')
      message.success(t('tenant.domain_add_success'))
      await reloadDetailDomains(detailRow.id)
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setDomainBusy(false)
    }
  }

  const onVerifyDomain = async (domainId: number) => {
    if (!detailRow) return
    setDomainBusy(true)
    try {
      await verifyPlatformTenantDomain(detailRow.id, domainId)
      message.success(t('tenant.domain_verify_success'))
      await reloadDetailDomains(detailRow.id)
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setDomainBusy(false)
    }
  }

  const onPrimaryDomain = async (domainId: number) => {
    if (!detailRow) return
    setDomainBusy(true)
    try {
      await setPrimaryPlatformTenantDomain(detailRow.id, domainId)
      message.success(t('common.success'))
      await reloadDetailDomains(detailRow.id)
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setDomainBusy(false)
    }
  }

  const onDisableDomain = async (domainId: number) => {
    if (!detailRow) return
    setDomainBusy(true)
    try {
      await disablePlatformTenantDomain(detailRow.id, domainId)
      message.success(t('common.success'))
      await reloadDetailDomains(detailRow.id)
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setDomainBusy(false)
    }
  }

  const onDeleteDomain = async (domainId: number) => {
    if (!detailRow) return
    setDomainBusy(true)
    try {
      await deletePlatformTenantDomain(detailRow.id, domainId)
      message.success(t('common.success'))
      await reloadDetailDomains(detailRow.id)
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setDomainBusy(false)
    }
  }

  const onPingRow = async (row: TenantRow) => {
    try {
      const res = await pingPlatformTenant(row.id)
      const data = (res as { data?: { ping?: TenantPingDetail } })?.data
      const ping = data?.ping || null
      setPingRow(row)
      setPingDetail(ping)
      if (ping?.ok) {
        message.success(t('tenant.ping_ok'))
      } else {
        message.warning(t('tenant.ping_failed'))
      }
    } catch (error) {
      showError(error, t('common.operation_failed'))
    }
  }

  const runBatchOps = (
    op: 'migrate' | 'seed' | 'backup',
    payload: {
      ids?: Array<string | number>
      provision_status?: string
      status?: number
      with_seed?: boolean
      limit?: number
    },
    confirmTitle: string,
    confirmContent: string,
  ) => {
    modal.confirm({
      title: confirmTitle,
      content: confirmContent,
      onOk: async () => {
        setBatchLoading(true)
        try {
          const res = await opsPlatformTenantBatch({ op, ...payload })
          const data = (res as { data?: { queued_count?: number; batch_id?: string } })?.data
          const n = Number(data?.queued_count ?? 0)
          const batch = data?.batch_id ? t('tenant.batch_id_suffix', { id: data.batch_id }) : ''
          message.success(t('tenant.batch_queued', { n, batch }))
          setSelectedRowKeys([])
          await refresh()
          await refreshOpsSummary()
          await refreshHealthBanner()
        } catch (error) {
          showError(error, t('common.operation_failed'))
        } finally {
          setBatchLoading(false)
        }
      },
    })
  }

  const exportCsv = async () => {
    setExportLoading(true)
    try {
      const blob = (await exportPlatformTenants(searchForm as Record<string, unknown>)) as unknown as Blob
      const url = window.URL.createObjectURL(blob instanceof Blob ? blob : new Blob([blob as unknown as BlobPart]))
      const a = document.createElement('a')
      a.href = url
      a.download = `tenants_${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '')}.csv`
      a.click()
      window.URL.revokeObjectURL(url)
      message.success(t('tenant.export_success'))
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setExportLoading(false)
    }
  }

  const restoreBackup = (file: { name: string }) => {
    if (!backupsRow) return
    modal.confirm({
      title: t('tenant.op_restore'),
      content: t('tenant.op_restore_confirm', { name: file.name }),
      onOk: async () => {
        setRestoringName(file.name)
        try {
          await restorePlatformTenant(backupsRow.id, { backup_name: file.name })
          message.success(t('tenant.op_queued'))
          setBackupsRow(null)
          await refresh()
        } catch (error) {
          showError(error, t('common.operation_failed'))
        } finally {
          setRestoringName('')
        }
      },
    })
  }

  const pruneBackups = async () => {
    if (!backupsRow) return
    modal.confirm({
      title: t('tenant.prune_backups'),
      content: t('tenant.prune_confirm', { keep: pruneKeep }),
      onOk: async () => {
        setPruneLoading(true)
        try {
          const res = await prunePlatformTenantBackups(backupsRow.id, { keep: pruneKeep })
          const removed = Number((res as { data?: { removed?: number } })?.data?.removed ?? 0)
          message.success(t('tenant.prune_success', { n: removed }))
          await openBackups(backupsRow)
        } catch (error) {
          showError(error, t('common.operation_failed'))
        } finally {
          setPruneLoading(false)
        }
      },
    })
  }

  const submitDelete = async () => {
    if (!deleteTarget) return
    setDeleteLoading(true)
    try {
      await deletePlatformTenant(deleteTarget.id, {
        confirm_code: deleteConfirmCode.trim(),
        drop_database: deleteDropDb,
      })
      message.success(t('tenant.delete_to_recycle'))
      setDeleteTarget(null)
      setDeleteConfirmCode('')
      setDeleteDropDb(false)
      closeDetail()
      setListMode('recycle')
      await refreshOpsSummary()
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setDeleteLoading(false)
    }
  }

  const onUndelete = (row: TenantRow) => {
    modal.confirm({
      title: t('tenant.undelete_confirm', { code: row.code }),
      onOk: async () => {
        try {
          await undeletePlatformTenant(row.id)
          message.success(t('tenant.undelete_success'))
          setListMode('active')
          await refreshOpsSummary()
        } catch (error) {
          showError(error, t('common.operation_failed'))
        }
      },
    })
  }

  const submitPurge = async () => {
    if (!purgeTarget) return
    setPurgeLoading(true)
    try {
      await purgePlatformTenant(purgeTarget.id, {
        purge_objects: purgeObjects,
        purge_backups: purgeBackups,
      })
      message.success(t('tenant.purge_queued'))
      setPurgeTarget(null)
      await refresh()
      await refreshOpsSummary()
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setPurgeLoading(false)
    }
  }

  const submitForceDelete = async () => {
    if (!forceDeleteTarget) return
    setForceDeleteLoading(true)
    try {
      const res = await forceDeletePlatformTenant(forceDeleteTarget.id, {
        confirm_code: forceDeleteConfirm.trim(),
        purge_objects: forcePurgeObjects,
        purge_backups: forcePurgeBackups,
      })
      const data = (res as { data?: { force_delete_queued?: boolean; purge_queued?: boolean } })?.data
      if (data?.force_delete_queued || data?.purge_queued) {
        message.success(t('tenant.force_delete_queued'))
      } else {
        message.success(t('tenant.force_delete_success'))
      }
      setForceDeleteTarget(null)
      setForceDeleteConfirm('')
      await refresh()
      await refreshOpsSummary()
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setForceDeleteLoading(false)
    }
  }

  const hasBusy = tableData.some(isTenantBusy)
  useEffect(() => {
    if (!hasBusy) return
    const timer = window.setInterval(() => {
      void refresh()
      void refreshOpsSummary()
      void refreshHealthBanner()
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
          const data = (res as { data?: { queued_count?: number; batch_id?: string } })?.data
          const n = Number(data?.queued_count ?? 0)
          const batch = data?.batch_id ? t('tenant.batch_id_suffix', { id: data.batch_id }) : ''
          message.success(t('tenant.batch_queued', { n, batch }))
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

  const schemaLabel = (status?: string) => {
    switch (status) {
      case 'aligned':
        return t('tenant.schema_aligned')
      case 'behind':
        return t('tenant.schema_behind')
      case 'failed':
        return t('tenant.schema_failed')
      case 'running':
        return t('tenant.schema_running')
      case 'unknown':
        return t('tenant.schema_unknown')
      default:
        return status || '—'
    }
  }

  const schemaColor = (status?: string) => {
    switch (status) {
      case 'aligned':
        return 'success'
      case 'behind':
      case 'running':
        return 'warning'
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
        title: t('tenant.schema_status'),
        dataIndex: 'schema_status',
        key: 'schema_status',
        width: 110,
        render: (status: string, row) => (
          <Tooltip
            title={
              row.schema_migration_count != null || row.expected_migration_count != null
                ? `${row.schema_migration_count ?? 0}/${row.expected_migration_count ?? '?'}${
                    row.last_migrate_error ? ` · ${row.last_migrate_error}` : ''
                  }`
                : row.last_migrate_error || undefined
            }
          >
            <Tag color={schemaColor(status)}>{schemaLabel(status)}</Tag>
          </Tooltip>
        ),
      },
      {
        title: t('tenant.last_op'),
        key: 'last_op',
        width: 140,
        render: (_, row) => {
          const tag = recycleOpTag(row)
          if (tag) {
            return (
              <Tooltip title={row.last_op_message || row.last_op_at || undefined}>
                <Tag color={tag.color}>{tag.label}</Tag>
              </Tooltip>
            )
          }
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
            disabled={!isOwner}
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
      {
        title: t('tenant.maintenance'),
        dataIndex: 'maintenance',
        key: 'maintenance',
        width: 100,
        render: (maintenance, row) => (
          <Switch
            checked={!!maintenance}
            disabled={!isOwner}
            title={t('tenant.maintenance_hint')}
            onChange={(checked) => {
              modal.confirm({
                title: t('tenant.maintenance'),
                content: checked ? t('tenant.maintenance_confirm_on') : t('tenant.maintenance_confirm_off'),
                onOk: async () => {
                  try {
                    await updatePlatformTenantMaintenance(row.id, { maintenance: !!checked })
                    message.success(t('common.update_success'))
                    await refresh()
                    await refreshOpsSummary()
                  } catch (error) {
                    showError(error, t('common.operation_failed'))
                    throw error
                  }
                },
              })
            }}
          />
        ),
      },
      { title: t('table.created_at'), dataIndex: 'created_at', key: 'created_at', width: 170 },
      {
        title: t('common.operation'),
        key: 'actions',
        width: isOwner ? 460 : 220,
        fixed: 'right',
        render: (_, row) => {
          const busy = isTenantBusy(row)
          if (isRecycleView) {
            return (
              <Space size={0} wrap>
                {isOwner ? (
                  <>
                    <Button type="link" size="small" disabled={busy} onClick={() => onUndelete(row)}>
                      {t('tenant.op_undelete')}
                    </Button>
                    <Button
                      type="link"
                      size="small"
                      danger
                      disabled={busy}
                      onClick={() => {
                        setForceDeleteConfirm('')
                        setForcePurgeObjects(true)
                        setForcePurgeBackups(true)
                        setForceDeleteTarget(row)
                      }}
                    >
                      {t('tenant.op_force_delete_short')}
                    </Button>
                    {row.last_op === 'purge' && row.last_op_status === 'failed' ? (
                      <Button
                        type="link"
                        size="small"
                        disabled={busy}
                        onClick={() => {
                          setPurgeObjects(true)
                          setPurgeBackups(true)
                          setPurgeTarget(row)
                        }}
                      >
                        {t('tenant.op_purge_retry')}
                      </Button>
                    ) : null}
                  </>
                ) : (
                  <Typography.Text type="secondary">—</Typography.Text>
                )}
              </Space>
            )
          }
          return (
            <Space size={0} wrap>
              <Button type="link" size="small" onClick={() => openDetail(row)}>
                {t('tenant.op_detail')}
              </Button>
              {isOwner ? (
                <Button
                  type="link"
                  size="small"
                  disabled={busy}
                  onClick={() => {
                    setPasswordTouched(false)
                    setEditing(row)
                    form.setFieldsValue({
                      name: row.name,
                      host: row.host,
                      port: row.port || 0,
                      username: row.username,
                      password: '',
                      database: row.database,
                      schema: row.schema,
                      storage_limit_mb: bytesToMb(row.storage_limit_bytes),
                    })
                  }}
                >
                  {t('common.edit')}
                </Button>
              ) : null}
              <Button type="link" size="small" onClick={() => void onPingRow(row)}>
                {t('tenant.op_ping')}
              </Button>
              {isOwner ? (
                <>
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
                </>
              ) : null}
              <Button type="link" size="small" onClick={() => void openBackups(row)}>
                {t('tenant.op_backups')}
              </Button>
            </Space>
          )
        },
      },
    ],
    [t, message, modal, refresh, refreshOpsSummary, showError, form, copyText, openBackups, runQueued, openDetail, onPingRow, isRecycleView, isOwner],
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
      await createPlatformTenant({
        ...values,
        password: passwordTouched && values.password ? values.password : undefined,
        storage_limit_bytes: mbToBytes(values.storage_limit_mb) || undefined,
      })
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
        storage_limit_bytes: mbToBytes(values.storage_limit_mb),
      }
      if (passwordTouched && values.password) payload.password = values.password
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
      title={isRecycleView ? t('tenant.recycle_bin') : t('menu.tenant')}
      extra={
        <Space wrap>
          {!isRecycleView ? (
            <>
              <Button loading={exportLoading} onClick={() => void exportCsv()}>
                {t('tenant.export_csv')}
              </Button>
              {isOwner ? (
                <>
                  <Button
                    type="primary"
                    loading={batchLoading}
                    disabled={selectedRowKeys.length < 1}
                    onClick={() =>
                      runBatchOps(
                        'migrate',
                        { ids: selectedRowKeys.map((id) => Number(id)), with_seed: false },
                        t('tenant.batch_migrate'),
                        t('tenant.batch_migrate_confirm', { n: selectedRowKeys.length }),
                      )
                    }
                  >
                    {t('tenant.batch_migrate')}
                  </Button>
                  <Button
                    loading={batchLoading}
                    disabled={selectedRowKeys.length < 1}
                    onClick={() =>
                      runBatchOps(
                        'seed',
                        { ids: selectedRowKeys.map((id) => Number(id)) },
                        t('tenant.batch_seed'),
                        t('tenant.batch_seed_confirm', { n: selectedRowKeys.length }),
                      )
                    }
                  >
                    {t('tenant.batch_seed')}
                  </Button>
                  <Button
                    loading={batchLoading}
                    disabled={selectedRowKeys.length < 1}
                    onClick={() =>
                      runBatchOps(
                        'backup',
                        { ids: selectedRowKeys.map((id) => Number(id)) },
                        t('tenant.batch_backup'),
                        t('tenant.batch_backup_confirm', { n: selectedRowKeys.length }),
                      )
                    }
                  >
                    {t('tenant.batch_backup')}
                  </Button>
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
                        storage_limit_mb: 0,
                      })
                      setPasswordTouched(false)
                      setCreateOpen(true)
                    }}
                  >
                    {t('tenant.add')}
                  </Button>
                </>
              ) : null}
            </>
          ) : null}
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
        description={healthDesc}
      />
      <Alert
        type="warning"
        showIcon
        style={{ marginBottom: 12 }}
        message={t('platform.migrate_current_binary_title')}
        description={t('platform.migrate_current_binary_desc')}
      />
      <Space wrap style={{ marginBottom: 12 }} size="middle">
        <Segmented
          value={isRecycleView ? 'recycle' : 'active'}
          onChange={(v) => setListMode(v === 'recycle' ? 'recycle' : 'active')}
          options={[
            { label: t('tenant.list_active'), value: 'active' },
            {
              label: (
                <span>
                  {t('tenant.recycle_bin')}
                  {(opsSummary?.deleted ?? 0) > 0 ? (
                    <Badge count={opsSummary?.deleted} style={{ marginLeft: 6 }} size="small" />
                  ) : null}
                </span>
              ),
              value: 'recycle',
            },
          ]}
        />
        {opsSummary ? (
          <Typography.Text type="secondary">
            {t('tenant.ops_summary', {
              failed: opsSummary.failed_provision ?? 0,
              busy: opsSummary.busy ?? 0,
              total: opsSummary.total ?? 0,
            })}
            {(opsSummary.schema_behind ?? 0) > 0 || (opsSummary.schema_failed ?? 0) > 0
              ? ` · ${t('tenant.schema_summary', {
                  aligned: opsSummary.schema_aligned ?? 0,
                  behind: opsSummary.schema_behind ?? 0,
                  failed: opsSummary.schema_failed ?? 0,
                  unknown: opsSummary.schema_unknown ?? 0,
                })}`
              : ''}
            {(opsSummary.failed_purge ?? 0) > 0
              ? ` · ${t('tenant.failed_purge_count', { n: opsSummary.failed_purge })}`
              : ''}
            {(opsSummary.maintenance ?? 0) > 0
              ? ` · ${t('tenant.ops_maintenance')} ${opsSummary.maintenance}`
              : ''}
          </Typography.Text>
        ) : null}
      </Space>
      {isRecycleView ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          message={t('tenant.recycle_banner_title')}
          description={t('tenant.recycle_banner_desc')}
        />
      ) : null}
      <SearchForm
        fields={
          isRecycleView
            ? [
                { name: 'code', label: t('tenant.code') },
                { name: 'name', label: t('tenant.name') },
              ]
            : [
                { name: 'code', label: t('tenant.code') },
                { name: 'name', label: t('tenant.name') },
                {
                  name: 'schema_status',
                  label: t('tenant.schema_status'),
                  type: 'select',
                  options: [
                    { label: t('tenant.schema_aligned'), value: 'aligned' },
                    { label: t('tenant.schema_behind'), value: 'behind' },
                    { label: t('tenant.schema_failed'), value: 'failed' },
                    { label: t('tenant.schema_running'), value: 'running' },
                    { label: t('tenant.schema_unknown'), value: 'unknown' },
                  ],
                },
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
                {
                  name: 'maintenance',
                  label: t('tenant.maintenance'),
                  type: 'select',
                  options: [
                    { label: t('tenant.maintenance_on'), value: '1' },
                    { label: t('tenant.maintenance_off'), value: '0' },
                  ],
                },
              ]
        }
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={() => {
          const keep = String(searchForm.trashed || '')
          onSearchFormChange({
            code: '',
            name: '',
            status: '',
            provision_status: '',
            schema_status: '',
            maintenance: '',
            trashed: keep,
          })
          window.setTimeout(() => handleSearch(), 0)
        }}
      />

      <Table
        rowKey="id"
        loading={loading}
        dataSource={tableData}
        columns={filteredColumns}
        scroll={{ x: isRecycleView ? 1100 : 1500 }}
        rowSelection={
          isRecycleView
            ? undefined
            : {
                selectedRowKeys,
                onChange: (keys) =>
                  setSelectedRowKeys(keys.map((k) => (typeof k === 'bigint' ? String(k) : k))),
              }
        }
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
        onClose={closeDetail}
        width={520}
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
              <Descriptions.Item label={t('tenant.maintenance')}>
                {detailRow.maintenance ? t('tenant.maintenance_on') : t('tenant.maintenance_off')}
                {detailRow.maintenance_message ? ` · ${detailRow.maintenance_message}` : ''}
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
            <Divider>{t('tenant.overview_title')}</Divider>
            {detailExtraLoading ? (
              <Typography.Text type="secondary">{t('common.loading')}</Typography.Text>
            ) : detailOverview ? (
              <Descriptions column={1} size="small" bordered>
                <Descriptions.Item label={t('tenant.overview_tables')}>
                  {detailOverview.table_count ?? '—'}
                </Descriptions.Item>
                <Descriptions.Item label={t('tenant.overview_db_size')}>
                  {formatSize(detailOverview.database_bytes)}
                </Descriptions.Item>
                <Descriptions.Item label={t('tenant.overview_admins')}>
                  {detailOverview.admins_count ?? '—'}
                </Descriptions.Item>
                <Descriptions.Item label={t('tenant.overview_migrations')}>
                  {detailOverview.migrations_count ?? '—'}
                </Descriptions.Item>
                <Descriptions.Item label={t('tenant.storage_used')}>
                  {formatQuota(detailQuota?.storage_used_bytes, detailQuota?.storage_limit_bytes)}
                </Descriptions.Item>
                <Descriptions.Item label={t('tenant.ping_detail')}>
                  {detailOverview.ping_ok
                    ? t('tenant.ping_latency', { ms: detailOverview.ping_ms ?? 0 })
                    : detailOverview.error || '—'}
                </Descriptions.Item>
              </Descriptions>
            ) : (
              <Typography.Text type="secondary">—</Typography.Text>
            )}
            <Divider>{t('tenant.login_links')}</Divider>
            {detailLoginLinks ? (
              <Space direction="vertical" style={{ width: '100%' }}>
                <Typography.Text type="secondary">{detailLoginLinks.hint}</Typography.Text>
                {detailLoginLinks.primary_custom_url ? (
                  <Space wrap>
                    <Typography.Text code>{detailLoginLinks.primary_custom_url}</Typography.Text>
                    <Button
                      type="link"
                      size="small"
                      onClick={() => void copyText(detailLoginLinks.primary_custom_url || '')}
                    >
                      {t('tenant.login_copy_url')}
                    </Button>
                  </Space>
                ) : null}
                {detailLoginLinks.subdomain_url ? (
                  <Space wrap>
                    <Typography.Text code>{detailLoginLinks.subdomain_url}</Typography.Text>
                    <Button
                      type="link"
                      size="small"
                      onClick={() => void copyText(detailLoginLinks.subdomain_url || '')}
                    >
                      {t('tenant.login_copy_url')}
                    </Button>
                  </Space>
                ) : null}
                <Space wrap>
                  <Typography.Text code>{detailLoginLinks.query_url}</Typography.Text>
                  <Button type="link" size="small" onClick={() => void copyText(detailLoginLinks.query_url || '')}>
                    {t('tenant.login_copy_url')}
                  </Button>
                </Space>
                {detailLoginLinks.header ? (
                  <Typography.Text type="secondary">
                    {t('tenant.login_header')}: {detailLoginLinks.header}
                  </Typography.Text>
                ) : null}
              </Space>
            ) : (
              <Typography.Text type="secondary">—</Typography.Text>
            )}
            <Divider>{t('tenant.domains_title')}</Divider>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
              {t('tenant.domains_hint')}
            </Typography.Paragraph>
            {isOwner ? (
              <Space wrap style={{ marginBottom: 12 }}>
                <Input
                  style={{ width: 220 }}
                  placeholder="crm.customer.com"
                  value={domainHost}
                  onChange={(e) => setDomainHost(e.target.value)}
                />
                <Select
                  style={{ width: 160 }}
                  value={domainSslMode}
                  onChange={(v) => setDomainSslMode(v)}
                  options={[
                    { value: 'edge', label: t('tenant.domain_ssl_edge') },
                    { value: 'customer_cdn', label: t('tenant.domain_ssl_cdn') },
                  ]}
                />
                <Button type="primary" loading={domainBusy} onClick={() => void onAddDomain()}>
                  {t('tenant.domain_add')}
                </Button>
              </Space>
            ) : null}
            {detailDomains.length > 0 ? (
              <Space direction="vertical" style={{ width: '100%' }} size="middle">
                {detailDomains.map((d) => (
                  <div key={d.id} style={{ border: '1px solid var(--ant-color-border)', padding: 8, borderRadius: 6 }}>
                    <Space wrap>
                      <Typography.Text strong>{d.host}</Typography.Text>
                      <Tag>{d.status}</Tag>
                      <Tag>{d.ssl_mode}</Tag>
                      {d.is_primary ? <Tag color="blue">{t('tenant.domain_primary')}</Tag> : null}
                    </Space>
                    {d.dns_guide?.txt_name ? (
                      <div style={{ marginTop: 6 }}>
                        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                          TXT {d.dns_guide.txt_name} = {d.dns_guide.txt_value}
                          {d.dns_guide.domain_target ? ` · CNAME → ${d.dns_guide.domain_target}` : ''}
                        </Typography.Text>
                      </div>
                    ) : null}
                    {d.last_check_error ? (
                      <Typography.Text type="danger" style={{ fontSize: 12, display: 'block' }}>
                        {d.last_check_error}
                      </Typography.Text>
                    ) : null}
                    {isOwner ? (
                      <Space wrap style={{ marginTop: 8 }}>
                        {d.status !== 'active' ? (
                          <Button size="small" loading={domainBusy} onClick={() => void onVerifyDomain(d.id)}>
                            {t('tenant.domain_verify')}
                          </Button>
                        ) : (
                          <Button size="small" loading={domainBusy} onClick={() => void onPrimaryDomain(d.id)}>
                            {t('tenant.domain_set_primary')}
                          </Button>
                        )}
                        {d.status !== 'disabled' ? (
                          <Button size="small" loading={domainBusy} onClick={() => void onDisableDomain(d.id)}>
                            {t('tenant.domain_disable')}
                          </Button>
                        ) : null}
                        <Button size="small" danger loading={domainBusy} onClick={() => void onDeleteDomain(d.id)}>
                          {t('common.delete')}
                        </Button>
                      </Space>
                    ) : null}
                  </div>
                ))}
              </Space>
            ) : (
              <Typography.Text type="secondary">—</Typography.Text>
            )}
            <Divider>{t('tenant.op_timeline')}</Divider>
            {detailOpLogs.length > 0 ? (
              <Timeline
                items={detailOpLogs.map((log) => ({
                  color:
                    log.status === 'success' ? 'green' : log.status === 'failed' ? 'red' : 'blue',
                  children: (
                    <div>
                      <Typography.Text strong>
                        {log.op} / {log.status}
                      </Typography.Text>
                      {log.operator_name ? (
                        <div>
                          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                            {t('tenant_op_log.operator')}: {log.operator_name}
                          </Typography.Text>
                        </div>
                      ) : null}
                      {log.batch_id ? (
                        <div>
                          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                            {t('tenant_op_log.batch_id')}: {log.batch_id}
                          </Typography.Text>
                        </div>
                      ) : null}
                      <div>
                        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                          {log.finished_at || log.started_at || log.created_at || '—'}
                        </Typography.Text>
                      </div>
                      {log.message ? (
                        <Typography.Paragraph style={{ marginBottom: 0, fontSize: 12 }}>
                          {log.message}
                        </Typography.Paragraph>
                      ) : null}
                    </div>
                  ),
                }))}
              />
            ) : (
              <Typography.Text type="secondary">—</Typography.Text>
            )}
            <Space style={{ marginTop: 16 }} wrap>
              <Button type="primary" onClick={() => void openBackups(detailRow)}>
                {t('tenant.op_backups')}
              </Button>
              {isOwner ? (
                <Button
                  danger
                  disabled={isTenantBusy(detailRow)}
                  onClick={() => {
                    setDeleteTarget(detailRow)
                    setDeleteConfirmCode('')
                    setDeleteDropDb(false)
                  }}
                >
                  {t('tenant.op_delete')}
                </Button>
              ) : null}
            </Space>
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
        <Space style={{ marginBottom: 12 }} wrap>
          <Typography.Text type="secondary">
            {t('tenant.backup_keep')}: {backupKeep}
          </Typography.Text>
          {isOwner ? (
            <>
              <InputNumber min={0} max={500} value={pruneKeep} onChange={(v) => setPruneKeep(Number(v) || 0)} />
              <Button loading={pruneLoading} onClick={() => void pruneBackups()}>
                {t('tenant.prune_backups')}
              </Button>
            </>
          ) : null}
        </Space>
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
              width: 260,
              render: (_, file) => (
                <Space size={0} wrap>
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
                  {isOwner ? (
                    <Button
                      type="link"
                      size="small"
                      disabled={!!backupsRow && isTenantBusy(backupsRow)}
                      loading={restoringName === file.name}
                      onClick={() => restoreBackup(file)}
                    >
                      {t('tenant.op_restore')}
                    </Button>
                  ) : null}
                </Space>
              ),
            },
          ]}
        />
      </Modal>

      <Modal
        title={t('tenant.ping_detail')}
        open={!!pingDetail}
        onCancel={() => {
          setPingDetail(null)
          setPingRow(null)
        }}
        footer={[
          <Button
            key="close"
            onClick={() => {
              setPingDetail(null)
              setPingRow(null)
            }}
          >
            {t('common.close')}
          </Button>,
        ]}
        destroyOnHidden
      >
        {pingDetail ? (
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label={t('tenant.code')}>{pingRow?.code}</Descriptions.Item>
            <Descriptions.Item label={t('common.status')}>
              {pingDetail.ok ? t('tenant.ping_ok') : t('tenant.ping_failed')}
            </Descriptions.Item>
            <Descriptions.Item label={t('tenant.ping_latency')}>
              {pingDetail.latency_ms ?? '—'} ms
            </Descriptions.Item>
            <Descriptions.Item label={t('tenant.host')}>
              {pingDetail.host || '—'}:{pingDetail.port ?? '—'}
            </Descriptions.Item>
            <Descriptions.Item label={t('tenant.database')}>{pingDetail.database || '—'}</Descriptions.Item>
            {pingDetail.error ? (
              <Descriptions.Item label={t('tenant.op_message')}>{pingDetail.error}</Descriptions.Item>
            ) : null}
          </Descriptions>
        ) : null}
      </Modal>

      <Modal
        title={t('tenant.op_delete')}
        open={!!deleteTarget}
        onCancel={() => {
          setDeleteTarget(null)
          setDeleteConfirmCode('')
          setDeleteDropDb(false)
        }}
        onOk={() => void submitDelete()}
        confirmLoading={deleteLoading}
        okButtonProps={{ danger: true }}
        destroyOnHidden
      >
        <Alert type="warning" showIcon style={{ marginBottom: 12 }} message={t('tenant.op_delete_confirm')} />
        <Form layout="vertical">
          <Form.Item label={t('tenant.delete_confirm_code', { code: deleteTarget?.code || '' })}>
            <Input value={deleteConfirmCode} onChange={(e) => setDeleteConfirmCode(e.target.value)} />
          </Form.Item>
          <Form.Item label={t('tenant.delete_drop_database')}>
            <Switch checked={deleteDropDb} onChange={setDeleteDropDb} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('tenant.op_purge_retry')}
        open={!!purgeTarget}
        onCancel={() => setPurgeTarget(null)}
        onOk={() => void submitPurge()}
        confirmLoading={purgeLoading}
        destroyOnHidden
      >
        <p>{t('tenant.purge_retry_hint', { code: purgeTarget?.code || '' })}</p>
        {purgeTarget?.last_op === 'purge' ? (
          <Alert
            style={{ marginBottom: 12 }}
            type={purgeTarget.last_op_status === 'failed' ? 'error' : 'info'}
            showIcon
            message={`${purgeTarget.last_op} / ${purgeTarget.last_op_status || '—'}`}
            description={purgeTarget.last_op_message || undefined}
          />
        ) : null}
        <Form layout="vertical">
          <Form.Item label={t('tenant.purge_objects')}>
            <Switch checked={purgeObjects} onChange={setPurgeObjects} />
          </Form.Item>
          <Form.Item label={t('tenant.purge_backups')}>
            <Switch checked={purgeBackups} onChange={setPurgeBackups} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('tenant.op_force_delete')}
        open={!!forceDeleteTarget}
        onCancel={() => {
          setForceDeleteTarget(null)
          setForceDeleteConfirm('')
        }}
        onOk={() => void submitForceDelete()}
        confirmLoading={forceDeleteLoading}
        okButtonProps={{ danger: true }}
        destroyOnHidden
      >
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          message={t('tenant.force_delete_hint', { code: forceDeleteTarget?.code || '' })}
        />
        <Form layout="vertical">
          <Form.Item label={t('tenant.delete_confirm_code', { code: forceDeleteTarget?.code || '' })}>
            <Input value={forceDeleteConfirm} onChange={(e) => setForceDeleteConfirm(e.target.value)} />
          </Form.Item>
          <Form.Item label={t('tenant.purge_objects')} extra={t('tenant.purge_objects_tip')}>
            <Switch checked={forcePurgeObjects} onChange={setForcePurgeObjects} />
          </Form.Item>
          <Form.Item label={t('tenant.purge_backups')} extra={t('tenant.purge_backups_tip')}>
            <Switch checked={forcePurgeBackups} onChange={setForcePurgeBackups} />
          </Form.Item>
        </Form>
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
        <Form form={form} layout="vertical" autoComplete="off">
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
            <Input
              name="tenant_db_username"
              autoComplete="off"
              readOnly
              onFocus={unlockCredentialAutofill}
              placeholder={t('tenant.username_placeholder')}
            />
          </Form.Item>
          <Form.Item name="password" label={t('tenant.password')}>
            <Input.Password
              name="tenant_db_password"
              autoComplete="new-password"
              readOnly
              onFocus={unlockCredentialAutofill}
              onChange={() => setPasswordTouched(true)}
              placeholder={t('tenant.password_placeholder')}
            />
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
          <Divider>{t('tenant.quota_section')}</Divider>
          <Form.Item
            name="storage_limit_mb"
            label={t('tenant.storage_limit_mb')}
            extra={t('tenant.quota_zero_unlimited')}
          >
            <InputNumber min={0} max={1048576} style={{ width: '100%' }} />
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
        <Form form={form} layout="vertical" autoComplete="off">
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
            <Input
              name="tenant_db_username"
              autoComplete="off"
              readOnly
              onFocus={unlockCredentialAutofill}
              placeholder={t('tenant.username_placeholder')}
            />
          </Form.Item>
          <Form.Item name="password" label={t('tenant.password')}>
            <Input.Password
              name="tenant_db_password"
              autoComplete="new-password"
              readOnly
              onFocus={unlockCredentialAutofill}
              onChange={() => setPasswordTouched(true)}
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
          <Divider>{t('tenant.quota_section')}</Divider>
          <Form.Item
            name="storage_limit_mb"
            label={t('tenant.storage_limit_mb')}
            extra={t('tenant.quota_zero_unlimited')}
          >
            <InputNumber min={0} max={1048576} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
