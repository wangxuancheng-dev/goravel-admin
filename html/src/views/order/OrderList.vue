<template>
  <div class="order-list">
    <ListPage
      ref="listPageRef"
      page-class="order"
      :title="$t('menu.order')"
      :add-button-text="$t('order.add_order')"
      :add-button-disabled="getButtonState('order.store').disabled"
      :search-form="searchForm"
      :search-fields="searchFields"
      :initial-search-values="orderInitialSearchForm"
      i18n-prefix="order"
      :table-data="tableData"
      :loading="loading"
      :table-columns="tableColumns"
      :pagination="pagination"
      :hide-total-threshold="100000"
      show-toolbar
      @add="handleAdd"
      @search="handleSearch"
      @reset="handleReset"
      @refresh="loadData"
      @page-change="loadData"
    >
      <template #user_id="{ model, field }">
        <el-input v-model="model[field.prop]" :placeholder="t('order.user_id')" />
      </template>

      <template #extra-buttons>
        <el-button
          type="primary"
          :disabled="getButtonState('order.import').disabled || isImporting"
          :loading="isImporting"
          @click="handleImport"
        >
          <el-icon><Upload /></el-icon>
          {{ $t('common.import') }}
        </el-button>
        <el-button
          type="success"
          :disabled="getButtonState('order.export').disabled || isExporting"
          :loading="isExporting"
          @click="handleExport"
        >
          {{ $t('common.export') }}
        </el-button>
      </template>

      <template #table>
        <vxe-table
          ref="tableRef"
          :data="tableData"
          :loading="loading"
          border
          :size="vxeSize"
          :column-config="{ resizable: true }"
          height="600"
          :sort-config="{ multiple: false, trigger: 'default' }"
          @sort-change="handleSortChange"
        >
          <vxe-column type="expand" width="60">
            <template #content="{ row }">
              <div class="order-details-expand">
                <h4>{{ t('order.order_details') }}</h4>
                <vxe-table
                  :data="getOrderDetails(row)"
                  border
                  :size="vxeSize"
                  :show-header="true"
                >
                  <vxe-column field="product_name" :title="t('order.product_name')" width="200">
                    <template #default="{ row: detail }">
                      {{ detail.product_name || detail.ProductName || '-' }}
                    </template>
                  </vxe-column>
                  <vxe-column field="price" :title="t('order.price')" width="120">
                    <template #default="{ row: detail }">
                      {{ formatOrderAmount(detail.price || detail.Price) }}
                    </template>
                  </vxe-column>
                  <vxe-column field="quantity" :title="t('order.quantity')" width="100">
                    <template #default="{ row: detail }">
                      {{ detail.quantity || detail.Quantity || 0 }}
                    </template>
                  </vxe-column>
                  <vxe-column field="subtotal" :title="t('order.subtotal')" width="120">
                    <template #default="{ row: detail }">
                      {{ formatOrderAmount(detail.subtotal || detail.Subtotal) }}
                    </template>
                  </vxe-column>
                </vxe-table>
              </div>
            </template>
          </vxe-column>
          <template v-for="(column, index) in tableColumns">
            <vxe-column
              v-if="column.type"
              :key="`type-${column.type}-${index}`"
              :type="column.type"
              :width="column.width"
              :fixed="column.fixed"
            />
            <vxe-column
              v-else
              :key="column.field || column.slot || index"
              :field="column.field"
              :title="column.title"
              :width="column.width"
              :sortable="column.sortable"
              :fixed="column.fixed"
              :formatter="column.formatter"
            >
              <template v-if="column.slot === 'status'" #default="{ row }">
                <el-tag :type="getOrderStatusTagType(row.status)">
                  {{ getOrderStatusText(t, row.status) }}
                </el-tag>
              </template>
              <template v-else-if="column.slot === 'amount'" #default="{ row }">
                {{ formatOrderAmount(row.amount) }}
              </template>
              <template v-else-if="column.slot === 'operation'" #default="{ row }">
                <TableActionButtons
                  :row="row"
                  :primary-actions="getPrimaryActions(row)"
                  :get-button-state="getButtonState"
                  @action="handleAction"
                />
              </template>
            </vxe-column>
          </template>
        </vxe-table>
      </template>

      <template #form>
        <OrderForm v-model="dialogVisible" @success="handleFormSuccess" />
      </template>
    </ListPage>

    <input
      ref="fileInputRef"
      type="file"
      accept=".csv"
      style="display: none"
      @change="handleFileChange"
    />

    <el-dialog
      v-model="editDialogVisible"
      :title="$t('order.edit_order')"
      width="600px"
      :close-on-click-modal="false"
      @close="handleEditDialogClose"
    >
      <el-form
        ref="editFormRef"
        :model="editFormData"
        :rules="editFormRules"
        label-width="120px"
      >
        <el-form-item :label="$t('order.status')" prop="status">
          <el-select v-model="editFormData.status" :placeholder="$t('order.update_status_tip')" style="width: 100%">
            <el-option :label="$t('order.status_pending')" value="pending" />
            <el-option :label="$t('order.status_paid')" value="paid" />
            <el-option :label="$t('order.status_cancelled')" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('order.remark')" prop="remark">
          <el-input
            v-model="editFormData.remark"
            type="textarea"
            :rows="4"
            :placeholder="$t('order.remark_placeholder')"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="handleEditDialogClose">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="editSubmitting" @click="handleEditSubmit">
          {{ $t('common.confirm') }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="detailDialogVisible"
      :title="$t('order.detail')"
      width="80%"
      :close-on-click-modal="false"
    >
      <div v-if="orderDetail" class="order-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="$t('order.order_no')">
            {{ getOrderDetailField(orderDetail.order, 'order_no') }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('order.user_id')">
            {{ getOrderDetailField(orderDetail.order, 'user_id') }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('order.amount')">
            {{ formatOrderAmount(getOrderDetailField(orderDetail.order, 'amount', 0)) }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('order.status')">
            <el-tag :type="getOrderStatusTagType(orderDetail.order?.status)">
              {{ getOrderStatusText(t, orderDetail.order?.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('order.created_at')">
            {{ formatOrderTime(orderDetail.order?.created_at) }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('order.expire_at')">
            {{ formatOrderTime(orderDetail.order?.expire_at) }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('order.remark')">
            {{ getOrderDetailField(orderDetail.order, 'remark') }}
          </el-descriptions-item>
        </el-descriptions>

        <el-divider>{{ $t('order.details') }}</el-divider>

        <vxe-table :data="orderDetail.details || []" border :size="vxeSize">
          <vxe-column field="product_name" :title="$t('order.product_name')" />
          <vxe-column
            field="price"
            :title="$t('order.price')"
            :formatter="({ row }) => formatOrderAmount(row.price)"
          />
          <vxe-column field="quantity" :title="$t('order.quantity')" />
          <vxe-column
            field="subtotal"
            :title="$t('order.subtotal')"
            :formatter="({ row }) => formatOrderAmount(row.subtotal)"
          />
        </vxe-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted, computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQueuedExport } from '@/composables/useQueuedExport'
import { useCsvImport } from '@/composables/useCsvImport'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Upload } from '@element-plus/icons-vue'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import OrderForm from './OrderForm.vue'
import { useListPage } from '@/composables/useListPage'
import { usePermission } from '@/composables/usePermission'
import { useCrud } from '@/composables/useCrud'
import { useVxeTableSize } from '@/composables/useVxeTableSize'
import {
  getOrderList,
  getOrderDetail,
  updateOrder,
  deleteOrder,
  exportOrder,
  importOrder
} from '@/api/order'
import logger from '@/utils/logger'
import ErrorHandler from '@/utils/errorHandler'
import { validateTimeRange, ORDER_MAX_TIME_RANGE_MONTHS } from '@/utils/timeRangeValidator'
import {
  createOrderInitialSearchForm,
  createOrderSearchFields,
  createOrderTableColumns,
  formatOrderAmount,
  formatOrderTime,
  getOrderStatusText,
  getOrderStatusTagType,
  getOrderDetailField,
  getOrderDetails
} from './order.config'

const { getButtonState } = usePermission()
const { vxeSize } = useVxeTableSize()
const { t } = useI18n()
const listPageRef = ref(null)
const tableRef = ref(null)
const orderInitialSearchForm = createOrderInitialSearchForm()

const { dialogVisible, handleFormSuccess: handleFormSuccessCrud } = useCrud()
const detailDialogVisible = ref(false)
const orderDetail = ref(null)
const editDialogVisible = ref(false)
const editFormRef = ref(null)
const editSubmitting = ref(false)
const currentEditOrder = ref(null)
const editFormData = reactive({ status: '', remark: '' })

const editFormRules = computed(() => ({
  status: [{ required: true, message: t('order.status_required'), trigger: 'change' }]
}))

const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch: handleSearchBase,
  handleReset,
  handleSortChange,
  initDefaultSort
} = useListPage({
  fetchApi: getOrderList,
  initialSearchForm: orderInitialSearchForm,
  defaultSort: 'created_at:desc',
  normalizeRows: false,
  tableRef: computed(() => tableRef.value)
})

const searchFields = computed(() => createOrderSearchFields(t))
const tableColumns = computed(() => createOrderTableColumns(t))

const { isExporting, handleExport } = useQueuedExport({
  exportApi: exportOrder,
  getParams: () => searchForm
})

const { fileInputRef, isImporting, handleImport, handleFileChange } = useCsvImport({
  importApi: importOrder,
  onSuccess: loadData
})

const getCurrentTimeString = () => {
  const now = new Date()
  const pad = (n) => String(n).padStart(2, '0')
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
}

const validateTimeRangeForSearch = () => {
  if (!searchForm.start_time) return true
  const endTime = searchForm.end_time || getCurrentTimeString()
  const validation = validateTimeRange(searchForm.start_time, endTime, ORDER_MAX_TIME_RANGE_MONTHS)
  if (!validation.valid) {
    let errorMessage = validation.error
    if (validation.errorKey) {
      const translationKey = `order.${validation.errorKey}`
      errorMessage = validation.errorParams
        ? t(translationKey, validation.errorParams)
        : t(translationKey)
    }
    ElMessage.warning(errorMessage)
    return false
  }
  return true
}

const isInitialized = ref(false)
watch(
  () => [searchForm.start_time, searchForm.end_time],
  ([newStartTime, newEndTime], [oldStartTime, oldEndTime]) => {
    if (!isInitialized.value) {
      isInitialized.value = true
      return
    }
    if (newStartTime !== oldStartTime || newEndTime !== oldEndTime) {
      if (newStartTime) {
        const endTime = newEndTime || getCurrentTimeString()
        const validation = validateTimeRange(newStartTime, endTime, ORDER_MAX_TIME_RANGE_MONTHS)
        if (!validation.valid) {
          let errorMessage = validation.error
          if (validation.errorKey) {
            const translationKey = `order.${validation.errorKey}`
            errorMessage = validation.errorParams
              ? t(translationKey, validation.errorParams)
              : t(translationKey)
          }
          ElMessage.warning(errorMessage)
        }
      }
    }
  }
)

const handleSearch = () => {
  if (!validateTimeRangeForSearch()) return
  handleSearchBase()
}

const handleView = async (row) => {
  try {
    const orderNo = row.order_no
    const res = await getOrderDetail(row.id, orderNo ? { order_no: orderNo } : {})
    if (res.data) {
      orderDetail.value = res.data
      detailDialogVisible.value = true
    }
  } catch (error) {
    logger.error('View order detail error:', error)
    ErrorHandler.handle(error, { silent: true })
  }
}

const handleEdit = async (row) => {
  try {
    const orderNo = row.order_no
    const res = await getOrderDetail(row.id, orderNo ? { order_no: orderNo } : {})
    if (res.data?.order) {
      currentEditOrder.value = res.data.order
      editFormData.status = res.data.order.status || 'pending'
      editFormData.remark = res.data.order.remark || ''
      editDialogVisible.value = true
    }
  } catch (error) {
    logger.error('Get order detail for edit error:', error)
    ErrorHandler.handle(error, { silent: true })
  }
}

const handleEditDialogClose = () => {
  editDialogVisible.value = false
  currentEditOrder.value = null
  editFormData.status = ''
  editFormData.remark = ''
  editFormRef.value?.resetFields()
  editFormRef.value?.clearValidate()
}

const handleEditSubmit = async () => {
  if (!editFormRef.value) return
  try {
    await editFormRef.value.validate()
  } catch {
    return
  }
  if (!currentEditOrder.value) return

  editSubmitting.value = true
  try {
    const orderNo = currentEditOrder.value.order_no
    await updateOrder(currentEditOrder.value.id, {
      status: editFormData.status,
      remark: editFormData.remark || '',
      order_no: orderNo
    })
    ElMessage.success(t('common.update_success'))
    handleEditDialogClose()
    loadData()
  } catch (error) {
    logger.error('Update order error:', error)
    ErrorHandler.handle(error, { silent: true })
  } finally {
    editSubmitting.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(t('common.delete_confirm'), t('form.tip'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    const orderNo = row.order_no
    await deleteOrder(row.id, orderNo ? { order_no: orderNo } : {})
    ElMessage.success(t('common.delete_success'))
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      logger.error('Delete order error:', error)
      ErrorHandler.handle(error, { silent: true })
    }
  }
}

const getPrimaryActions = () => [
  { key: 'view', label: t('common.view'), type: 'info', permission: 'order.show', handler: handleView },
  { key: 'edit', label: t('common.edit'), type: 'primary', permission: 'order.update', handler: handleEdit },
  { key: 'delete', label: t('common.delete'), type: 'danger', permission: 'order.destroy', handler: handleDelete }
]

const handleAction = (command, row) => {
  if (command === 'view') handleView(row)
  if (command === 'edit') handleEdit(row)
  if (command === 'delete') handleDelete(row)
}

const handleAdd = () => {
  dialogVisible.value = true
}

const handleFormSuccess = () => {
  handleFormSuccessCrud(loadData)
}

onMounted(async () => {
  try {
    await loadData()
    await nextTick()
    initDefaultSort()
  } catch (error) {
    logger.error('OrderList onMounted error:', error)
    ErrorHandler.handle(error)
  }
})
</script>

<style scoped>
.order-details-expand {
  padding: var(--space-sm);
  background-color: var(--bg-color-tertiary);
}

.order-details-expand h4 {
  margin: var(--space-sm) 0;
  font-weight: bold;
  color: var(--text-color-primary);
}

html.dark .order-details-expand {
  background-color: var(--el-bg-color) !important;
}

html.dark .order-details-expand h4 {
  color: var(--el-text-color-primary) !important;
}

.order-detail {
  padding: 20px 0;
}
</style>
