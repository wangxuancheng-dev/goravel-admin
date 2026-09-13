<template>
  <div class="platform-tenant-page">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      class="ops-banner"
      :title="$t('platform.cli_ops_title')"
      :description="healthDesc"
    />
    <div v-if="opsSummary" class="ops-summary-row">
      <span>{{ $t('tenant.ops_summary', { failed: opsSummary.failed_provision || 0, busy: opsSummary.busy || 0, total: opsSummary.total || 0 }) }}</span>
      <span v-if="queueInfo" class="queue-meta">{{ $t('tenant.queue_status', { conn: queueInfo.connection || '-', pending: queueInfo.pending ?? '-', msg: queueInfo.message || '' }) }}</span>
      <el-button
        size="small"
        :loading="batchLoading"
        :disabled="!(opsSummary.failed_provision > 0)"
        @click="retryFailedMigrates"
      >
        {{ $t('tenant.retry_failed_migrate') }}
      </el-button>
      <el-button
        size="small"
        :loading="batchLoading"
        :disabled="selectedRows.length < 1"
        @click="batchSeedSelected"
      >
        {{ $t('tenant.batch_seed') }}
      </el-button>
      <el-button
        size="small"
        :loading="batchLoading"
        :disabled="selectedRows.length < 1"
        @click="batchBackupSelected"
      >
        {{ $t('tenant.batch_backup') }}
      </el-button>
      <el-button size="small" @click="exportCsv">{{ $t('tenant.export_csv') }}</el-button>
    </div>
    <ListPage
    ref="listPageRef"
    page-class="platform-tenant"
    :title="$t('menu.tenant')"
    :add-button-text="$t('tenant.add')"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="initialSearchForm"
    i18n-prefix="tenant"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :pagination="pagination"
    show-toolbar
    @add="openCreate"
    @search="handleSearch"
    @reset="handleReset"
    @refresh="loadData"
    @page-change="loadData"
    @sort-change="handleSortChange"
    @selection-change="handleSelectionChange"
  >
    <template #provision_status="{ row }">
      <el-tooltip :content="row.last_migrate_error || row.last_op_message || ''" :disabled="!(row.last_migrate_error || row.last_op_message)">
        <el-tag :type="provisionTagType(row.provision_status)" size="small">
          {{ provisionLabel(row.provision_status) }}
        </el-tag>
      </el-tooltip>
    </template>
    <template #last_op="{ row }">
      <el-tooltip :content="row.last_op_message || row.last_op_at || ''" :disabled="!(row.last_op_message || row.last_op_at)">
        <span class="op-meta">{{ row.last_op || '—' }} / {{ row.last_op_status || '—' }}</span>
      </el-tooltip>
    </template>
    <template #last_backup_path="{ row }">
      <div v-if="row.last_backup_path" class="backup-cell">
        <el-tooltip :content="row.last_backup_path" placement="top">
          <span class="backup-path">{{ shortPath(row.last_backup_path) }}</span>
        </el-tooltip>
        <el-button link type="primary" @click="copyText(row.last_backup_path)">{{ $t('tenant.backup_copy_path') }}</el-button>
      </div>
      <span v-else class="op-meta">—</span>
    </template>
    <template #status="{ row }">
      <el-switch
        :model-value="Number(row.status) === 1"
        @change="(val) => onToggleStatus(row, val)"
      />
    </template>
    <template #actions="{ row }">
      <el-button link type="primary" @click="openDetail(row)">{{ $t('tenant.op_detail') }}</el-button>
      <el-button link type="primary" :disabled="isBusy(row)" @click="openEdit(row)">{{ $t('common.edit') }}</el-button>
      <el-button link type="primary" @click="onPing(row)">{{ $t('tenant.op_ping') }}</el-button>
      <el-button link type="primary" :disabled="isBusy(row)" @click="openMigrate(row)">{{ $t('tenant.op_migrate') }}</el-button>
      <el-dropdown trigger="click" @command="(cmd) => onMoreCommand(cmd, row)">
        <el-button link type="primary">{{ $t('tenant.op_more') }}</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="seed" :disabled="isBusy(row)">{{ $t('tenant.op_seed') }}</el-dropdown-item>
            <el-dropdown-item command="backup" :disabled="isBusy(row)">{{ $t('tenant.op_backup') }}</el-dropdown-item>
            <el-dropdown-item command="backups">{{ $t('tenant.op_backups') }}</el-dropdown-item>
            <el-dropdown-item command="overview">{{ $t('tenant.op_overview') }}</el-dropdown-item>
            <el-dropdown-item command="timeline">{{ $t('tenant.op_timeline') }}</el-dropdown-item>
            <el-dropdown-item command="login">{{ $t('tenant.op_login_link') }}</el-dropdown-item>
            <el-dropdown-item command="delete" divided>{{ $t('tenant.op_delete') }}</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </template>

    <template #form>
      <el-dialog
        v-model="dialogVisible"
        :title="editingId ? $t('tenant.edit') : $t('tenant.add')"
        width="640px"
        destroy-on-close
        @closed="resetForm"
      >
        <el-form ref="formRef" :model="form" :rules="formRules" label-width="110px">
          <el-form-item v-if="!editingId" :label="$t('tenant.code')" prop="code">
            <el-input v-model="form.code" :placeholder="$t('tenant.code_placeholder')" />
          </el-form-item>
          <el-form-item :label="$t('tenant.name')" prop="name">
            <el-input v-model="form.name" />
          </el-form-item>
          <template v-if="!editingId">
            <el-form-item :label="$t('tenant.driver')" prop="driver">
              <el-select v-model="form.driver" style="width: 100%">
                <el-option label="mysql" value="mysql" />
                <el-option label="postgres" value="postgres" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('tenant.isolation')" prop="isolation">
              <el-select v-model="form.isolation" style="width: 100%">
                <el-option label="database" value="database" />
                <el-option label="schema" value="schema" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('tenant.database')">
              <el-input v-model="form.database" :placeholder="$t('tenant.database_placeholder')" />
            </el-form-item>
            <el-form-item :label="$t('tenant.schema')">
              <el-input v-model="form.schema" :placeholder="$t('tenant.schema_placeholder')" />
            </el-form-item>
          </template>
          <el-divider content-position="left">{{ $t('tenant.remote_connection') }}</el-divider>
          <el-form-item :label="$t('tenant.host')">
            <el-input v-model="form.host" :placeholder="$t('tenant.host_placeholder')" />
          </el-form-item>
          <el-form-item :label="$t('tenant.port')">
            <el-input-number v-model="form.port" :min="0" :max="65535" controls-position="right" style="width: 100%" />
          </el-form-item>
          <el-form-item :label="$t('tenant.username')">
            <el-input v-model="form.username" :placeholder="$t('tenant.username_placeholder')" />
          </el-form-item>
          <el-form-item :label="$t('tenant.password')">
            <el-input
              v-model="form.password"
              type="password"
              show-password
              :placeholder="editingId && form.has_password ? $t('tenant.password_keep') : $t('tenant.password_placeholder')"
            />
          </el-form-item>
          <el-form-item v-if="editingId" :label="$t('tenant.database')">
            <el-input v-model="form.database" />
          </el-form-item>
          <el-form-item v-if="editingId" :label="$t('tenant.schema')">
            <el-input v-model="form.schema" />
          </el-form-item>
          <el-alert
            v-if="!editingId"
            type="info"
            :closable="false"
            show-icon
            class="migrate-tip"
            :title="$t('tenant.migrate_cli_tip')"
          />
          <el-form-item v-if="!editingId" :label="$t('tenant.skip_create')">
            <el-switch v-model="form.skip_create" />
            <div class="form-tip">{{ $t('tenant.skip_create_tip') }}</div>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" :loading="saving" @click="submitForm">{{ $t('common.confirm') }}</el-button>
        </template>
      </el-dialog>
    </template>
  </ListPage>

  <el-dialog v-model="migrateVisible" :title="$t('tenant.op_migrate')" width="480px" destroy-on-close>
    <p>{{ $t('tenant.op_migrate_confirm') }}</p>
    <div class="seed-row">
      <el-switch v-model="withSeed" />
      <span>{{ $t('tenant.op_with_seed') }}</span>
    </div>
    <el-alert
      v-if="migrateRow?.last_migrate_error"
      type="error"
      :closable="false"
      show-icon
      class="migrate-tip"
      :title="$t('tenant.migrate_error')"
      :description="migrateRow.last_migrate_error"
    />
    <template #footer>
      <el-button @click="migrateVisible = false">{{ $t('common.cancel') }}</el-button>
      <el-button type="primary" @click="submitMigrate">{{ $t('common.confirm') }}</el-button>
    </template>
  </el-dialog>

  <el-drawer v-model="detailVisible" :title="$t('tenant.detail_title')" size="440px" destroy-on-close>
    <template v-if="detailRow">
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item :label="$t('tenant.code')">{{ detailRow.code }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.name')">{{ detailRow.name }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.driver')">{{ detailRow.driver }} / {{ detailRow.isolation }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.database')">{{ detailRow.database }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.schema')">{{ detailRow.schema || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.host')">{{ detailRow.host || '—' }}:{{ detailRow.port || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.connection_name')">{{ detailRow.connection_name || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.provision_status')">{{ provisionLabel(detailRow.provision_status) }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.last_op')">{{ detailRow.last_op || '—' }} / {{ detailRow.last_op_status || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.last_op_at')">{{ detailRow.last_op_at || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.op_message')">{{ detailRow.last_op_message || detailRow.last_migrate_error || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.migrated_at')">{{ detailRow.migrated_at || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.backup_dir')">
          <div class="backup-cell">
            <span>{{ detailRow.backup_dir || backupDirOf(detailRow) }}</span>
            <el-button link type="primary" @click="copyText(detailRow.backup_dir || backupDirOf(detailRow))">{{ $t('tenant.backup_copy_path') }}</el-button>
          </div>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.backup_path')">
          <div v-if="detailRow.last_backup_path" class="backup-cell">
            <span>{{ detailRow.last_backup_path }}</span>
            <el-button link type="primary" @click="copyText(detailRow.last_backup_path)">{{ $t('tenant.backup_copy_path') }}</el-button>
          </div>
          <span v-else>—</span>
        </el-descriptions-item>
      </el-descriptions>
      <div class="drawer-actions">
        <el-button type="primary" @click="openBackups(detailRow)">{{ $t('tenant.op_backups') }}</el-button>
        <el-button @click="openOverview(detailRow)">{{ $t('tenant.op_overview') }}</el-button>
        <el-button @click="openTimeline(detailRow)">{{ $t('tenant.op_timeline') }}</el-button>
        <el-button @click="copyLoginLink(detailRow)">{{ $t('tenant.op_login_link') }}</el-button>
      </div>
    </template>
  </el-drawer>

  <el-dialog v-model="backupsVisible" :title="$t('tenant.backup_list_title')" width="780px" destroy-on-close>
    <el-alert type="info" :closable="false" show-icon class="migrate-tip" :title="$t('tenant.backup_hint')" />
    <div v-if="backupsMeta.dir" class="backup-dir-row">
      <span>{{ $t('tenant.backup_dir') }}: {{ backupsMeta.dir }}</span>
      <el-button link type="primary" @click="copyText(backupsMeta.dir)">{{ $t('tenant.backup_copy_path') }}</el-button>
      <span class="queue-meta">{{ $t('tenant.backup_keep_label', { n: backupsMeta.keep || settingsKeep || '-' }) }}</span>
      <el-button link type="warning" :loading="pruneLoading" @click="pruneBackups">{{ $t('tenant.backup_prune') }}</el-button>
    </div>
    <el-table v-loading="backupsLoading" :data="backupsList" size="small" empty-text="">
      <el-table-column prop="name" :label="$t('tenant.backup_name')" min-width="180" />
      <el-table-column prop="size" :label="$t('tenant.backup_size')" width="100">
        <template #default="{ row }">{{ formatSize(row.size) }}</template>
      </el-table-column>
      <el-table-column prop="mod_time" :label="$t('tenant.backup_time')" width="170" />
      <el-table-column :label="$t('common.operation')" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="copyText(row.path)">{{ $t('tenant.backup_copy_path') }}</el-button>
          <el-button link type="primary" :loading="downloadingName === row.name" @click="downloadBackup(row)">{{ $t('tenant.backup_download') }}</el-button>
          <el-button link type="danger" :disabled="isBusy(backupsRow)" @click="restoreBackup(row)">{{ $t('tenant.op_restore') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!backupsLoading && backupsList.length === 0" :description="$t('tenant.backup_empty')" />
  </el-dialog>

  <el-drawer v-model="overviewVisible" :title="$t('tenant.overview_title')" size="420px" destroy-on-close>
    <el-descriptions v-if="overviewData" :column="1" border size="small">
      <el-descriptions-item :label="$t('tenant.database')">{{ overviewData.database }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.driver')">{{ overviewData.driver }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.overview_ping')">{{ overviewData.ping_ok ? 'OK' : 'FAIL' }} / {{ overviewData.ping_ms }}ms</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.overview_tables')">{{ overviewData.table_count }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.overview_size')">{{ formatSize(overviewData.database_bytes) }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.overview_admins')">{{ overviewData.admins_count }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.overview_migrations')">{{ overviewData.migrations_count }}</el-descriptions-item>
      <el-descriptions-item v-if="overviewData.error" :label="$t('tenant.op_message')">{{ overviewData.error }}</el-descriptions-item>
    </el-descriptions>
  </el-drawer>

  <el-drawer v-model="timelineVisible" :title="$t('tenant.timeline_title')" size="520px" destroy-on-close>
    <el-timeline v-if="opLogs.length">
      <el-timeline-item v-for="item in opLogs" :key="item.id" :timestamp="item.finished_at || item.started_at || item.created_at" placement="top">
        <div>{{ item.op }} / {{ item.status }}</div>
        <div v-if="item.operator_name" class="op-meta">{{ $t('tenant_op_log.operator') }}: {{ item.operator_name }}</div>
        <div v-if="item.batch_id" class="op-meta">{{ $t('tenant_op_log.batch_id') }}: {{ item.batch_id }}</div>
        <div class="op-meta">{{ item.message || '—' }}</div>
      </el-timeline-item>
    </el-timeline>
    <el-empty v-else :description="$t('tenant.timeline_empty')" />
  </el-drawer>

  <el-dialog v-model="deleteVisible" :title="$t('tenant.op_delete')" width="480px" destroy-on-close>
    <p>{{ $t('tenant.delete_confirm_hint', { code: deleteRow?.code || '' }) }}</p>
    <el-form label-width="120px">
      <el-form-item :label="$t('tenant.confirm_code')">
        <el-input v-model="deleteConfirm" :placeholder="deleteRow?.code || ''" />
      </el-form-item>
      <el-form-item :label="$t('tenant.drop_database')">
        <el-switch v-model="deleteDropDb" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="deleteVisible = false">{{ $t('common.cancel') }}</el-button>
      <el-button type="danger" :loading="deleteLoading" @click="submitDelete">{{ $t('common.confirm') }}</el-button>
    </template>
  </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import {
  backupPlatformTenant,
  createPlatformTenant,
  deletePlatformTenant,
  downloadPlatformTenantBackup,
  exportPlatformTenants,
  getPlatformTenantList,
  getPlatformTenantLoginLinks,
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
  restorePlatformTenant,
  seedPlatformTenant,
  updatePlatformTenant,
  updatePlatformTenantStatus
} from '@/api/platform'

const { t } = useI18n()
const listPageRef = ref(null)
const formRef = ref(null)
const saving = ref(false)
const editingId = ref(null)
const health = ref(null)
const migrateVisible = ref(false)
const migrateRow = ref(null)
const withSeed = ref(true)
const opsSummary = ref(null)
const batchLoading = ref(false)
const selectedRows = ref([])
const detailVisible = ref(false)
const detailRow = ref(null)
const backupsVisible = ref(false)
const backupsRow = ref(null)
const backupsList = ref([])
const backupsMeta = reactive({ dir: '', last: '', keep: 0 })
const backupsLoading = ref(false)
const downloadingName = ref('')
const pruneLoading = ref(false)
const queueInfo = ref(null)
const settingsKeep = ref(0)
const overviewVisible = ref(false)
const overviewData = ref(null)
const timelineVisible = ref(false)
const opLogs = ref([])
const deleteVisible = ref(false)
const deleteRow = ref(null)
const deleteConfirm = ref('')
const deleteDropDb = ref(false)
const deleteLoading = ref(false)
let pollTimer = null

const healthDesc = computed(() => {
  const h = health.value
  const cli = t('platform.cli_ops_hint')
  if (!h) return cli
  const db = h.database_ok ? t('platform.health_db_ok') : t('platform.health_db_bad')
  const tenants = t('platform.health_tenants', {
    active: h.tenants?.active ?? 0,
    total: h.tenants?.total ?? 0
  })
  const q = h.queue?.connection ? ` · queue=${h.queue.connection}` : ''
  return `${t('platform.health_driver')}: ${h.driver} · ${db} · ${tenants}${q} · ${cli}`
})

const refreshOpsSummary = async () => {
  try {
    const res = await getPlatformTenantOpsSummary()
    opsSummary.value = res?.data?.summary || null
  } catch {
    // ignore
  }
}

const refreshSettings = async () => {
  try {
    const res = await getPlatformTenantSettings()
    settingsKeep.value = res?.data?.backup_keep || 0
    queueInfo.value = res?.data?.queue || null
  } catch {
    // ignore
  }
}

onMounted(async () => {
  try {
    const res = await platformHealth()
    health.value = res?.data || null
    if (res?.data?.queue) queueInfo.value = res.data.queue
    if (res?.data?.backup_keep) settingsKeep.value = res.data.backup_keep
  } catch {
    // ignore — list still usable
  }
  await refreshOpsSummary()
  await refreshSettings()
})

const initialSearchForm = { code: '', name: '', status: '', provision_status: '' }

const {
  pagination,
  tableData,
  loading,
  searchForm,
  dialogVisible,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange
} = useStandardListPage({
  fetchApi: getPlatformTenantList,
  initialSearchForm,
  defaultSort: 'id:desc'
})

const isBusy = (row) => {
  if (row?.provision_status === 'migrating') return true
  return row?.last_op_status === 'queued' || row?.last_op_status === 'running'
}

watch(
  tableData,
  (rows) => {
    const busy = Array.isArray(rows) && rows.some(isBusy)
    if (busy && !pollTimer) {
      pollTimer = window.setInterval(() => {
        loadData()
        refreshOpsSummary()
      }, 3000)
    } else if (!busy && pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  },
  { deep: true }
)

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

const provisionLabel = (status) => {
  switch (status) {
    case 'ready':
      return t('tenant.provision_ready')
    case 'migrating':
      return t('tenant.provision_migrating')
    case 'failed':
      return t('tenant.provision_failed')
    default:
      return t('tenant.provision_pending')
  }
}

const provisionTagType = (status) => {
  switch (status) {
    case 'ready':
      return 'success'
    case 'migrating':
      return 'warning'
    case 'failed':
      return 'danger'
    default:
      return 'info'
  }
}

const searchFields = computed(() => [
  { prop: 'code', label: t('tenant.code'), type: 'input', width: '180px' },
  { prop: 'name', label: t('tenant.name'), type: 'input', width: '180px' },
  {
    prop: 'provision_status',
    label: t('tenant.provision_status_filter'),
    type: 'select',
    width: '160px',
    options: [
      { label: t('tenant.provision_pending'), value: 'pending' },
      { label: t('tenant.provision_migrating'), value: 'migrating' },
      { label: t('tenant.provision_ready'), value: 'ready' },
      { label: t('tenant.provision_failed'), value: 'failed' }
    ]
  },
  {
    prop: 'status',
    label: t('common.status'),
    type: 'select',
    width: '140px',
    options: [
      { label: t('common.enabled'), value: '1' },
      { label: t('common.disabled'), value: '0' }
    ]
  }
])

const retryFailedMigrates = async () => {
  try {
    await ElMessageBox.confirm(t('tenant.retry_failed_migrate_confirm'), t('tenant.retry_failed_migrate'), {
      type: 'warning'
    })
  } catch {
    return
  }
  batchLoading.value = true
  try {
    const res = await migratePlatformTenantBatch({ provision_status: 'failed', with_seed: false })
    const n = res?.data?.queued_count ?? 0
    const batch = res?.data?.batch_id ? t('tenant.batch_id_suffix', { id: res.data.batch_id }) : ''
    ElMessage.success(t('tenant.batch_queued', { n, batch }))
    await loadData()
    await refreshOpsSummary()
  } catch (e) {
    console.error(e)
  } finally {
    batchLoading.value = false
  }
}

const tableColumns = computed(() => [
  { type: 'checkbox', width: 52, fixed: 'left', key: 'checkbox' },
  { field: 'id', title: t('table.id'), width: 70, sortable: true, key: 'id' },
  { field: 'code', title: t('tenant.code'), width: 110, key: 'code' },
  { field: 'name', title: t('tenant.name'), width: 120, key: 'name' },
  { field: 'driver', title: t('tenant.driver'), width: 90, key: 'driver' },
  { field: 'database', title: t('tenant.database'), width: 140, key: 'database' },
  { field: 'provision_status', title: t('tenant.provision_status'), width: 110, slot: 'provision_status', key: 'provision_status' },
  { field: 'last_op', title: t('tenant.last_op'), width: 140, slot: 'last_op', key: 'last_op' },
  { field: 'last_backup_path', title: t('tenant.backup_path'), minWidth: 180, slot: 'last_backup_path', key: 'last_backup_path' },
  { field: 'status', title: t('common.status'), width: 90, slot: 'status', key: 'status' },
  { field: 'created_at', title: t('table.created_at'), key: 'created_at' },
  { field: 'actions', title: t('common.operation'), width: 320, slot: 'actions', key: 'actions' }
])

const handleSelectionChange = (rows) => {
  selectedRows.value = Array.isArray(rows) ? rows : []
}

const form = reactive({
  code: '',
  name: '',
  driver: 'mysql',
  isolation: 'database',
  database: '',
  schema: '',
  host: '',
  port: 0,
  username: '',
  password: '',
  has_password: false,
  skip_create: false
})

const formRules = computed(() => {
  const rules = {
    name: [{ required: true, message: t('tenant.name_required'), trigger: 'blur' }]
  }
  if (!editingId.value) {
    rules.code = [{ required: true, message: t('tenant.code_required'), trigger: 'blur' }]
  }
  return rules
})

const openCreate = () => {
  editingId.value = null
  dialogVisible.value = true
}

const openEdit = (row) => {
  editingId.value = row.id
  form.name = row.name || ''
  form.host = row.host || ''
  form.port = Number(row.port) || 0
  form.username = row.username || ''
  form.password = ''
  form.database = row.database || ''
  form.schema = row.schema || ''
  form.has_password = !!row.has_password
  dialogVisible.value = true
}

const openMigrate = (row) => {
  migrateRow.value = row
  withSeed.value = true
  migrateVisible.value = true
}

const resetForm = () => {
  editingId.value = null
  form.code = ''
  form.name = ''
  form.driver = 'mysql'
  form.isolation = 'database'
  form.database = ''
  form.schema = ''
  form.host = ''
  form.port = 0
  form.username = ''
  form.password = ''
  form.has_password = false
  form.skip_create = false
}

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (editingId.value) {
        const payload = {
          name: form.name,
          host: form.host,
          port: form.port || 0,
          username: form.username,
          database: form.database,
          schema: form.schema
        }
        if (form.password) payload.password = form.password
        await updatePlatformTenant(editingId.value, payload)
        ElMessage.success(t('common.update_success'))
      } else {
        await createPlatformTenant({
          code: form.code,
          name: form.name,
          driver: form.driver,
          isolation: form.isolation,
          database: form.database || undefined,
          schema: form.schema || undefined,
          host: form.host || undefined,
          port: form.port || undefined,
          username: form.username || undefined,
          password: form.password || undefined,
          skip_create: form.skip_create
        })
        ElMessage.success(t('common.create_success'))
      }
      dialogVisible.value = false
      loadData()
    } catch (error) {
      if (!error?.__handled) {
        ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
      }
    } finally {
      saving.value = false
    }
  })
}

const onToggleStatus = async (row, enabled) => {
  const status = enabled ? 1 : 0
  try {
    await updatePlatformTenantStatus(row.id, status)
    row.status = status
    ElMessage.success(t('common.update_success'))
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
    loadData()
  }
}

const onPing = async (row) => {
  try {
    const res = await pingPlatformTenant(row.id)
    const ping = res?.data?.ping
    if (ping?.ok) {
      ElMessage.success(t('tenant.ping_ok_detail', { ms: ping.latency_ms ?? '-', host: ping.host || '-', db: ping.database || '-' }))
    } else {
      ElMessage.error(ping?.error || t('tenant.ping_fail'))
    }
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const onMoreCommand = (cmd, row) => {
  switch (cmd) {
    case 'seed':
      onSeed(row)
      break
    case 'backup':
      onBackup(row)
      break
    case 'backups':
      openBackups(row)
      break
    case 'overview':
      openOverview(row)
      break
    case 'timeline':
      openTimeline(row)
      break
    case 'login':
      copyLoginLink(row)
      break
    case 'delete':
      openDelete(row)
      break
    default:
      break
  }
}

const batchSeedSelected = async () => {
  const ids = selectedRows.value.map((r) => r.id).filter(Boolean)
  if (!ids.length) {
    ElMessage.warning(t('tenant.batch_need_selection'))
    return
  }
  try {
    await ElMessageBox.confirm(t('tenant.batch_seed_confirm', { n: ids.length }), { type: 'warning' })
  } catch {
    return
  }
  batchLoading.value = true
  try {
    const res = await opsPlatformTenantBatch({ op: 'seed', ids })
    const n = res?.data?.queued_count ?? 0
    const batch = res?.data?.batch_id ? t('tenant.batch_id_suffix', { id: res.data.batch_id }) : ''
    ElMessage.success(t('tenant.batch_queued', { n, batch }))
    selectedRows.value = []
    await loadData()
    await refreshOpsSummary()
  } catch (e) {
    console.error(e)
  } finally {
    batchLoading.value = false
  }
}

const batchBackupSelected = async () => {
  const ids = selectedRows.value.map((r) => r.id).filter(Boolean)
  if (!ids.length) {
    ElMessage.warning(t('tenant.batch_need_selection'))
    return
  }
  try {
    await ElMessageBox.confirm(t('tenant.batch_backup_confirm', { n: ids.length }), { type: 'warning' })
  } catch {
    return
  }
  batchLoading.value = true
  try {
    const res = await opsPlatformTenantBatch({ op: 'backup', ids })
    const n = res?.data?.queued_count ?? 0
    const batch = res?.data?.batch_id ? t('tenant.batch_id_suffix', { id: res.data.batch_id }) : ''
    ElMessage.success(t('tenant.batch_queued', { n, batch }))
    selectedRows.value = []
    await loadData()
    await refreshOpsSummary()
  } catch (e) {
    console.error(e)
  } finally {
    batchLoading.value = false
  }
}

const exportCsv = async () => {
  try {
    const blob = await exportPlatformTenants({ ...searchForm })
    const url = window.URL.createObjectURL(blob instanceof Blob ? blob : new Blob([blob]))
    const a = document.createElement('a')
    a.href = url
    a.download = `tenants_${Date.now()}.csv`
    a.click()
    window.URL.revokeObjectURL(url)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const openOverview = async (row) => {
  overviewVisible.value = true
  overviewData.value = null
  try {
    const res = await getPlatformTenantOverview(row.id)
    overviewData.value = res?.data?.overview || null
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const openTimeline = async (row) => {
  timelineVisible.value = true
  opLogs.value = []
  try {
    const res = await getPlatformTenantOpLogs(row.id, { limit: 40 })
    opLogs.value = res?.data?.list || []
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const copyLoginLink = async (row) => {
  try {
    const res = await getPlatformTenantLoginLinks(row.id)
    const links = res?.data?.links
    const text = links?.query_url || `/?tenant_code=${row.code}`
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('tenant.login_link_copied', { hint: links?.hint || '' }))
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const openDelete = (row) => {
  deleteRow.value = row
  deleteConfirm.value = ''
  deleteDropDb.value = false
  deleteVisible.value = true
}

const submitDelete = async () => {
  if (!deleteRow.value) return
  deleteLoading.value = true
  try {
    await deletePlatformTenant(deleteRow.value.id, {
      confirm_code: deleteConfirm.value,
      drop_database: deleteDropDb.value
    })
    ElMessage.success(t('common.delete_success'))
    deleteVisible.value = false
    loadData()
    refreshOpsSummary()
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    deleteLoading.value = false
  }
}

const restoreBackup = async (file) => {
  if (!backupsRow.value || !file?.name) return
  try {
    await ElMessageBox.confirm(t('tenant.op_restore_confirm', { name: file.name }), { type: 'warning' })
    await restorePlatformTenant(backupsRow.value.id, { backup_name: file.name })
    ElMessage.success(t('tenant.op_queued'))
    loadData()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const pruneBackups = async () => {
  if (!backupsRow.value) return
  const keep = backupsMeta.keep || settingsKeep.value || 10
  try {
    await ElMessageBox.confirm(t('tenant.backup_prune_confirm', { n: keep }), { type: 'warning' })
  } catch {
    return
  }
  pruneLoading.value = true
  try {
    const res = await prunePlatformTenantBackups(backupsRow.value.id, { keep })
    ElMessage.success(t('tenant.backup_prune_done', { n: res?.data?.removed ?? 0 }))
    await openBackups(backupsRow.value)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    pruneLoading.value = false
  }
}

const submitMigrate = async () => {
  if (!migrateRow.value) return
  try {
    await migratePlatformTenant(migrateRow.value.id, { with_seed: withSeed.value })
    ElMessage.success(t('tenant.op_queued'))
    migrateVisible.value = false
    loadData()
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const onSeed = async (row) => {
  try {
    await ElMessageBox.confirm(t('tenant.op_seed_confirm'), { type: 'warning' })
    await seedPlatformTenant(row.id)
    ElMessage.success(t('tenant.op_queued'))
    loadData()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const onBackup = async (row) => {
  try {
    await ElMessageBox.confirm(t('tenant.op_backup_confirm'), { type: 'warning' })
    await backupPlatformTenant(row.id)
    ElMessage.success(t('tenant.op_queued'))
    loadData()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const shortPath = (path) => {
  const s = String(path || '')
  if (s.length <= 28) return s
  return `…${s.slice(-26)}`
}

const backupDirOf = (row) => {
  if (row?.backup_dir) return row.backup_dir
  if (row?.code) return `storage/backups/tenants/${row.code}`
  return ''
}

const formatSize = (n) => {
  const size = Number(n) || 0
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

const copyText = async (text) => {
  const value = String(text || '')
  if (!value) return
  try {
    await navigator.clipboard.writeText(value)
    ElMessage.success(t('tenant.backup_copied'))
  } catch {
    ElMessage.error(t('common.operation_failed'))
  }
}

const openDetail = (row) => {
  detailRow.value = row
  detailVisible.value = true
}

const openBackups = async (row) => {
  backupsRow.value = row
  backupsVisible.value = true
  backupsList.value = []
  backupsMeta.dir = row.backup_dir || backupDirOf(row)
  backupsMeta.last = row.last_backup_path || ''
  backupsLoading.value = true
  try {
    const res = await listPlatformTenantBackups(row.id)
    backupsList.value = res?.data?.list || []
    backupsMeta.dir = res?.data?.backup_dir || backupsMeta.dir
    backupsMeta.last = res?.data?.last_backup_path || backupsMeta.last
    backupsMeta.keep = res?.data?.backup_keep || settingsKeep.value || 0
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    backupsLoading.value = false
  }
}

const downloadBackup = async (file) => {
  if (!backupsRow.value || !file?.name) return
  downloadingName.value = file.name
  try {
    const blob = await downloadPlatformTenantBackup(backupsRow.value.id, file.name)
    const url = window.URL.createObjectURL(blob instanceof Blob ? blob : new Blob([blob]))
    const a = document.createElement('a')
    a.href = url
    a.download = file.name
    a.click()
    window.URL.revokeObjectURL(url)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    downloadingName.value = ''
  }
}
</script>

<style scoped>
.ops-banner {
  margin-bottom: 12px;
}
.form-tip {
  margin-top: 4px;
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.4;
}
.migrate-tip {
  margin-bottom: 12px;
}
.op-meta {
  font-size: 12px;
  color: #64748b;
}
.seed-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 12px 0;
}
.backup-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.backup-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: #475569;
}
.backup-dir-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 13px;
  color: #64748b;
}
.drawer-actions {
  margin-top: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.ops-summary-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 13px;
  color: #475569;
}
.queue-meta {
  font-size: 12px;
  color: #64748b;
}
</style>
