<template>
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
    <template #status="{ row }">
      <el-switch
        :model-value="Number(row.status) === 1"
        @change="(val) => onToggleStatus(row, val)"
      />
    </template>
    <template #actions="{ row }">
      <el-button link type="primary" @click="openEdit(row)">{{ $t('common.edit') }}</el-button>
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
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import {
  createPlatformTenant,
  getPlatformTenantList,
  updatePlatformTenant,
  updatePlatformTenantStatus
} from '@/api/platform'

const { t } = useI18n()
const listPageRef = ref(null)
const formRef = ref(null)
const saving = ref(false)
const editingId = ref(null)

const initialSearchForm = { code: '', name: '', status: '' }

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

const searchFields = computed(() => [
  { prop: 'code', label: t('tenant.code'), type: 'input', width: '180px' },
  { prop: 'name', label: t('tenant.name'), type: 'input', width: '180px' },
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

const tableColumns = computed(() => [
  { field: 'id', title: t('table.id'), width: 80, sortable: true, key: 'id' },
  { field: 'code', title: t('tenant.code'), key: 'code' },
  { field: 'name', title: t('tenant.name'), key: 'name' },
  { field: 'driver', title: t('tenant.driver'), width: 100, key: 'driver' },
  { field: 'isolation', title: t('tenant.isolation'), width: 110, key: 'isolation' },
  { field: 'host', title: t('tenant.host'), key: 'host' },
  { field: 'database', title: t('tenant.database'), key: 'database' },
  { field: 'schema', title: t('tenant.schema'), key: 'schema' },
  { field: 'status', title: t('common.status'), width: 100, slot: 'status', key: 'status' },
  { field: 'created_at', title: t('table.created_at'), key: 'created_at' },
  { field: 'actions', title: t('common.operation'), width: 100, slot: 'actions', key: 'actions' }
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
</script>

<style scoped>
.form-tip {
  margin-top: 4px;
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.4;
}
.migrate-tip {
  margin-bottom: 12px;
}
</style>
