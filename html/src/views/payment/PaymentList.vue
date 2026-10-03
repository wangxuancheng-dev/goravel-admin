<template>
  <div class="payment-list">
    <ListPage
      ref="listPageRef"
      page-class="payment"
      :title="$t('menu.payment')"
      :show-add-button="false"
      :search-form="searchForm"
      :search-fields="searchFields"
      :initial-search-values="paymentInitialSearchForm"
      i18n-prefix="payment"
      :table-data="tableData"
      :loading="loading"
      :table-columns="tableColumns"
      :pagination="pagination"
      :hide-total-threshold="100000"
      show-toolbar
      @search="handleSearch"
      @reset="handleReset"
      @refresh="loadData"
      @page-change="loadData"
      @sort-change="handleSortChange"
    >
      <template #extra-buttons>
        <el-button
          type="success"
          :disabled="getButtonState('payment.export').disabled || isExporting"
          :loading="isExporting"
          @click="handleExport"
        >
          <el-icon><Download /></el-icon>
          {{ $t('common.export') }}
        </el-button>
      </template>

      <template #status="{ row }">
        <el-tag :type="getPaymentStatusType(row.status)" size="small">
          {{ getPaymentStatusText(t, row.status) }}
        </el-tag>
      </template>

      <template #payment_method="{ row }">
        {{ getPaymentMethodName(row.payment_method) }}
      </template>

      <template #amount="{ row }">
        {{ formatPaymentAmount(row.amount) }}
      </template>

      <template #operation="{ row }">
        <el-button
          type="primary"
          size="small"
          :disabled="getButtonState('payment.show').disabled"
          @click="handleView(row)"
        >
          {{ $t('common.view') }}
        </el-button>
      </template>
    </ListPage>

    <el-dialog
      v-model="detailDialogVisible"
      :title="$t('payment.payment_detail')"
      width="800px"
    >
      <div v-loading="detailLoading">
        <el-descriptions v-if="paymentDetail" :column="2" border>
          <el-descriptions-item :label="$t('payment.payment_no')">
            {{ paymentDetail.payment_no }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('payment.order_no')">
            {{ paymentDetail.order_no }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('payment.payment_method')">
            {{ getPaymentMethodName(paymentDetail.payment_method) }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('payment.user_id')">
            {{ paymentDetail.user_id }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('payment.amount')">
            {{ formatPaymentAmount(paymentDetail.amount) }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('payment.status')">
            <el-tag :type="getPaymentStatusType(paymentDetail.status)" size="small">
              {{ getPaymentStatusText(t, paymentDetail.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('payment.third_party_no')">
            {{ paymentDetail.third_party_no || '-' }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('payment.pay_time')">
            {{ paymentDetail.pay_time || '-' }}
          </el-descriptions-item>
          <el-descriptions-item
            v-if="paymentDetail.fail_reason"
            :label="$t('payment.fail_reason')"
            :span="2"
          >
            {{ paymentDetail.fail_reason }}
          </el-descriptions-item>
          <el-descriptions-item
            v-if="paymentDetail.remark"
            :label="$t('payment.remark')"
            :span="2"
          >
            {{ paymentDetail.remark }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('table.created_at')">
            {{ formatPaymentDateTime(paymentDetail.created_at) }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('table.updated_at')">
            {{ formatPaymentDateTime(paymentDetail.updated_at) }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Download } from '@element-plus/icons-vue'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { getPaymentList, getPaymentDetail, exportPayments } from '@/api/payment'
import logger from '@/utils/logger'
import ErrorHandler from '@/utils/errorHandler'
import { normalizeEntity } from '@/utils/normalize'
import {
  createPaymentInitialSearchForm,
  createPaymentSearchFields,
  createPaymentTableColumns,
  getPaymentStatusType,
  getPaymentStatusText,
  getPaymentMethodName,
  formatPaymentAmount,
  formatPaymentDateTime
} from './payment.config'

const { t } = useI18n()
const router = useRouter()
const listPageRef = ref(null)
const detailDialogVisible = ref(false)
const detailLoading = ref(false)
const paymentDetail = ref(null)
const isExporting = ref(false)
const paymentInitialSearchForm = createPaymentInitialSearchForm()

const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
  getButtonState
} = useStandardListPage({
  fetchApi: getPaymentList,
  initialSearchForm: paymentInitialSearchForm,
  defaultSort: 'created_at:desc',
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const searchFields = computed(() => createPaymentSearchFields(t))
const tableColumns = computed(() => createPaymentTableColumns(t))

const handleView = async (row) => {
  detailDialogVisible.value = true
  detailLoading.value = true
  try {
    const response = await getPaymentDetail(row.payment_no)
    paymentDetail.value = normalizeEntity(response.data?.data || response.data || {})
  } catch (error) {
    logger.error('Load payment detail error:', error)
    ErrorHandler.handle(error)
  } finally {
    detailLoading.value = false
  }
}

const handleExport = async () => {
  if (isExporting.value) return
  isExporting.value = true
  try {
    const response = await exportPayments({
      ...searchForm,
      order_by: 'created_at:desc'
    })
    const exportId = response.data?.data?.export_id || response.data?.export_id
    if (!exportId) {
      ElMessage.error(t('export.failed'))
      return
    }
    ElMessage.success(t('export.task_submitted'))
    router.push('/exports')
  } catch (error) {
    logger.error('Export payments error:', error)
    if (error.response?.status === 429) {
      ElMessage.warning(t('common.already_queued'))
    } else if (!error.__handled) {
      ErrorHandler.handle(error, { silent: true })
    }
  } finally {
    isExporting.value = false
  }
}
</script>
