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
      <el-button
        size="small"
        :loading="batchLoading"
        :disabled="!(opsSummary.failed_provision > 0)"
        @click="retryFailedMigrates"
      >
        {{ $t('tenant.retry_failed_migrate') }}
      </el-button>
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
  >
    <template #provision_status="{ row }">
      <el-tooltip :content="row.last_migrate_error || row.last_op_message || ''" :disabled="!(row.last_migrate_error || row.last_op_message)">
        <el-tag :type="provisionTagType(row.provision_status)" size="small">
          {{ provisionLabel(row.provision_status) }}
        </el-tag>
      </el-tooltip>
    </template>
    <template #last_op="{ row }">
      <span class="op-meta">{{ row.last_op || '—' }} / {{ row.last_op_status || '—' }}</span>
    </template>
    <template #status="{ row }">
      <el-switch
        :model-value="Number(row.status) === 1"
        @change="(val) => onToggleStatus(row, val)"
      />
    </template>
    <template #actions="{ row }">
      <el-button link type="primary" :disabled="isBusy(row)" @click="openEdit(row)">{{ $t('common.edit') }}</el-button>
      <el-button link type="primary" @click="onPing(row)">{{ $t('tenant.op_ping') }}</el-button>
      <el-button link type="primary" :disabled="isBusy(row)" @click="openMigrate(row)">{{ $t('tenant.op_migrate') }}</el-button>
      <el-button link type="primary" :disabled="isBusy(row)" @click="onSeed(row)">{{ $t('tenant.op_seed') }}</el-button>
      <el-button link type="primary" :disabled="isBusy(row)" @click="onBackup(row)">{{ $t('tenant.op_backup') }}</el-button>
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
  getPlatformTenantList,
  getPlatformTenantOpsSummary,
  migratePlatformTenant,
  migratePlatformTenantBatch,
  pingPlatformTenant,
  platformHealth,
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
  return `${t('platform.health_driver')}: ${h.driver} · ${db} · ${tenants} · ${cli}`
})

const refreshOpsSummary = async () => {
  try {
    const res = await getPlatformTenantOpsSummary()
    opsSummary.value = res?.data?.summary || null
  } catch {
    // ignore
  }
}

onMounted(async () => {
  try {
    const res = await platformHealth()
    health.value = res?.data || null
  } catch {
    // ignore — list still usable
  }
  await refreshOpsSummary()
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
    ElMessage.success(t('tenant.batch_queued', { n }))
    await loadData()
    await refreshOpsSummary()
  } catch (e) {
    console.error(e)
  } finally {
    batchLoading.value = false
  }
}

const tableColumns = computed(() => [
  { field: 'id', title: t('table.id'), width: 70, sortable: true, key: 'id' },
  { field: 'code', title: t('tenant.code'), width: 110, key: 'code' },
  { field: 'name', title: t('tenant.name'), width: 120, key: 'name' },
  { field: 'driver', title: t('tenant.driver'), width: 90, key: 'driver' },
  { field: 'database', title: t('tenant.database'), width: 140, key: 'database' },
  { field: 'provision_status', title: t('tenant.provision_status'), width: 110, slot: 'provision_status', key: 'provision_status' },
  { field: 'last_op', title: t('tenant.last_op'), width: 140, slot: 'last_op', key: 'last_op' },
  { field: 'status', title: t('common.status'), width: 90, slot: 'status', key: 'status' },
  { field: 'created_at', title: t('table.created_at'), key: 'created_at' },
  { field: 'actions', title: t('common.operation'), width: 300, slot: 'actions', key: 'actions' }
])

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
    await pingPlatformTenant(row.id)
    ElMessage.success(t('tenant.ping_ok'))
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
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
</style>
