<template>
  <ListPage
    ref="listPageRef"
    page-class="user-balance-log"
    :show-add-button="false"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="initialSearchValues"
    i18n-prefix="user"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :table-key="`table-${tableColumns.length}`"
    :pagination="pagination"
    :table-height="500"
    show-toolbar
    show-column-setting
    :visible-columns="visibleColumns"
    :all-columns="allColumns"
    :default-visible-columns="defaultVisibleColumns"
    :column-order="columnOrder"
    :fixed-columns="fixedColumns"
    :on-column-setting-confirm="handleColumnSettingConfirm"
    @search="handleSearchWithStats"
    @reset="handleResetWithStats"
    @refresh="refreshAll"
    @page-change="loadData"
  >
    <template #header-title>
      <div class="header-left">
        <el-button type="primary" link :icon="ArrowLeft" @click="handleBack">
          {{ $t('user.back') }}
        </el-button>
        <span>{{ $t('user.balance_logs') }}</span>
        <span v-if="searchForm.user_id" class="user-id-display">
          ({{ $t('user.user_id') }}: {{ searchForm.user_id }})
        </span>
      </div>
    </template>

    <template #before-table>
      <el-card v-if="statistics" class="statistics-card" shadow="never">
        <div class="statistics">
          <div class="stat-item">
            <span class="label">{{ $t('user.total_income') }}:</span>
            <span class="value income">¥{{ formatMoney(statistics.total_income) }}</span>
          </div>
          <div class="stat-item">
            <span class="label">{{ $t('user.total_expense') }}:</span>
            <span class="value expense">¥{{ formatMoney(statistics.total_expense) }}</span>
          </div>
          <div class="stat-item">
            <span class="label">{{ $t('user.total_refund') }}:</span>
            <span class="value refund">¥{{ formatMoney(statistics.total_refund) }}</span>
          </div>
          <div class="stat-item">
            <span class="label">{{ $t('user.current_balance') }}:</span>
            <span class="value balance">¥{{ formatMoney(statistics.current_balance) }}</span>
          </div>
        </div>
      </el-card>
    </template>

    <template #type="{ row }">
      <el-tag v-if="row.type === 'income'" type="success">{{ $t('user.balance_income') }}</el-tag>
      <el-tag v-else-if="row.type === 'expense'" type="danger">{{ $t('user.balance_expense') }}</el-tag>
      <el-tag v-else-if="row.type === 'refund'" type="warning">{{ $t('user.balance_refund') }}</el-tag>
    </template>

    <template #amount="{ row }">
      <span :style="{ color: row.type === 'expense' ? 'var(--el-color-danger)' : 'var(--el-color-success)' }">
        {{ row.type === 'expense' ? '-' : '+' }}¥{{ formatMoney(row.amount) }}
      </span>
    </template>

    <template #balance="{ row }">
      <span style="color: var(--el-color-primary); font-weight: bold;">¥{{ formatMoney(row.balance) }}</span>
    </template>

    <template #source="{ row }">
      <span v-if="row.source === 'order'">{{ $t('user.source_order') }}</span>
      <span v-else-if="row.source === 'recharge'">{{ $t('user.source_recharge') }}</span>
      <span v-else-if="row.source === 'withdraw'">{{ $t('user.source_withdraw') }}</span>
      <span v-else-if="row.source === 'manual'">{{ $t('user.source_manual') }}</span>
      <span v-else>{{ row.source }}</span>
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'
import ListPage from '@/components/ListPage.vue'
import { useListPage } from '@/composables/useListPage'
import { useColumnSetting } from '@/composables/useColumnSetting'
import { getUserBalanceLogList, getUserBalanceStatistics } from '@/api/userBalanceLog'
import ErrorHandler from '@/utils/errorHandler'
import {
  createUserBalanceLogInitialSearch,
  createUserBalanceLogSearchFields,
  createUserBalanceLogTableColumns,
  formatMoney,
  buildBalanceLogParams
} from './userBalanceLog.config'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const userId = route.query.user_id || ''
const initialSearchValues = createUserBalanceLogInitialSearch(userId)
const statistics = ref(null)
const listPageRef = ref(null)

const allTableColumns = computed(() => createUserBalanceLogTableColumns(t))

const {
  tableColumns,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('user_balance_log', allTableColumns)

const searchFields = computed(() => createUserBalanceLogSearchFields(t))

const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch,
  handleReset
} = useListPage({
  fetchApi: getUserBalanceLogList,
  initialSearchForm: initialSearchValues,
  buildParams: buildBalanceLogParams,
  defaultSort: 'id:desc',
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

pagination.pageSize = 20

const loadStatistics = async () => {
  if (!searchForm.user_id) return
  try {
    const res = await getUserBalanceStatistics({
      user_id: searchForm.user_id,
      start_time: searchForm.start_time,
      end_time: searchForm.end_time
    })
    if (res.code === 200) {
      statistics.value = res.data.statistics
    }
  } catch (error) {
    ErrorHandler.handle(error)
  }
}

const refreshAll = async () => {
  await loadData()
  await loadStatistics()
}

const handleSearchWithStats = () => {
  handleSearch()
  loadStatistics()
}

const handleResetWithStats = (...args) => {
  handleReset(...args)
  loadStatistics()
}

const handleBack = () => {
  if (window.history.length > 1) {
    router.go(-1)
  } else {
    router.push('/users')
  }
}

onMounted(() => {
  if (searchForm.user_id) {
    loadData()
    loadStatistics()
  }
})
</script>

<style scoped>
.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-id-display {
  color: var(--text-color-secondary);
  font-size: 14px;
}

.statistics-card {
  margin-bottom: 20px;
}

.statistics {
  display: flex;
  gap: 30px;
  flex-wrap: wrap;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.stat-item .label {
  font-weight: 500;
}

.stat-item .value {
  font-size: 18px;
  font-weight: bold;
}

.stat-item .value.income {
  color: var(--el-color-success);
}

.stat-item .value.expense {
  color: var(--el-color-danger);
}

.stat-item .value.refund {
  color: var(--el-color-warning);
}

.stat-item .value.balance {
  color: var(--el-color-primary);
}
</style>
