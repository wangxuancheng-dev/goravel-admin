<template>
  <ListPage
    ref="listPageRef"
    page-class="tenant"
    :title="$t('menu.tenant')"
    :add-button-text="$t('tenant.add')"
    :add-button-disabled="getButtonState('tenant.store').disabled"
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
        :disabled="getButtonState('tenant.update_status').disabled"
        @change="(val) => onToggleStatus(row, val)"
      />
    </template>

    <template #form>
      <el-dialog
        v-model="dialogVisible"
        :title="$t('tenant.add')"
        width="520px"
        destroy-on-close
        @closed="resetCreateForm"
      >
        <el-form ref="formRef" :model="createForm" :rules="createRules" label-width="100px">
          <el-form-item :label="$t('tenant.code')" prop="code">
            <el-input v-model="createForm.code" :placeholder="$t('tenant.code_placeholder')" />
          </el-form-item>
          <el-form-item :label="$t('tenant.name')" prop="name">
            <el-input v-model="createForm.name" />
          </el-form-item>
          <el-form-item :label="$t('tenant.driver')" prop="driver">
            <el-select v-model="createForm.driver" style="width: 100%">
              <el-option label="mysql" value="mysql" />
              <el-option label="postgres" value="postgres" />
            </el-select>
          </el-form-item>
          <el-form-item :label="$t('tenant.isolation')" prop="isolation">
            <el-select v-model="createForm.isolation" style="width: 100%">
              <el-option label="database" value="database" />
              <el-option label="schema" value="schema" />
            </el-select>
          </el-form-item>
          <el-form-item :label="$t('tenant.migrate')">
            <el-switch v-model="createForm.migrate" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" :loading="saving" @click="submitCreate">{{ $t('common.confirm') }}</el-button>
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
import { createTenant, getTenantList, updateTenantStatus } from '@/api/tenant'

const { t } = useI18n()
const listPageRef = ref(null)
const formRef = ref(null)
const saving = ref(false)

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
  handleSortChange,
  getButtonState
} = useStandardListPage({
  fetchApi: getTenantList,
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
  { field: 'database', title: t('tenant.database'), key: 'database' },
  { field: 'schema', title: t('tenant.schema'), key: 'schema' },
  { field: 'status', title: t('common.status'), width: 100, slot: 'status', key: 'status' },
  { field: 'created_at', title: t('table.created_at'), key: 'created_at' }
])

const createForm = reactive({
  code: '',
  name: '',
  driver: 'mysql',
  isolation: 'database',
  migrate: true
})

const createRules = computed(() => ({
  code: [{ required: true, message: t('tenant.code_required'), trigger: 'blur' }],
  name: [{ required: true, message: t('tenant.name_required'), trigger: 'blur' }]
}))

const openCreate = () => {
  dialogVisible.value = true
}

const resetCreateForm = () => {
  createForm.code = ''
  createForm.name = ''
  createForm.driver = 'mysql'
  createForm.isolation = 'database'
  createForm.migrate = true
}

const submitCreate = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      await createTenant({ ...createForm })
      ElMessage.success(t('common.create_success'))
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
    await updateTenantStatus(row.id, status)
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
