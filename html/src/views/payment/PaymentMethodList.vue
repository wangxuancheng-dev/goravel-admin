<template>
  <ListPage
    ref="listPageRef"
    page-class="payment-method"
    :title="$t('menu.payment_method')"
    :add-button-text="$t('payment_method.add_payment_method')"
    :add-button-disabled="getButtonState('payment_method.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="paymentMethodInitialSearchForm"
    i18n-prefix="payment_method"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :pagination="pagination"
    show-toolbar
    @add="handleAdd"
    @search="handleSearch"
    @reset="handleReset"
    @refresh="loadData"
    @page-change="loadData"
    @sort-change="handleSortChange"
  >
    <template #type="{ row }">
      <el-tag type="info" effect="plain">
        {{ getPaymentMethodTypeLabel(t, row.type) }}
      </el-tag>
    </template>

    <template #is_active="{ row }">
      <el-tag :type="row.is_active ? 'success' : 'danger'">
        {{ row.is_active ? $t('common.enabled') : $t('common.disabled') }}
      </el-tag>
    </template>

    <template #operation="{ row }">
      <TableActionButtons
        :row="row"
        :primary-actions="operationActions"
        :get-button-state="getButtonState"
      />
    </template>

    <template #form>
      <PaymentMethodForm
        ref="paymentMethodFormRef"
        v-model="dialogVisible"
        :edit-id="editId"
        @success="handleFormSuccess"
      />
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import PaymentMethodForm from './PaymentMethodForm.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { createCrudActions } from '@/utils/listPageHelpers'
import { getPaymentMethodList, deletePaymentMethod } from '@/api/paymentMethod'
import { useUserStore } from '@/store/user'
import {
  paymentMethodInitialSearchForm,
  createPaymentMethodSearchFields,
  createPaymentMethodTableColumns,
  getPaymentMethodTypeLabel
} from './paymentMethod.config'

const { t } = useI18n()
const userStore = useUserStore()
const listPageRef = ref(null)
const paymentMethodFormRef = ref(null)

const {
  pagination,
  tableData,
  loading,
  searchForm,
  dialogVisible,
  editId,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
  handleAdd,
  handleEdit,
  handleFormSuccess,
  handleDelete,
  getButtonState
} = useStandardListPage({
  fetchApi: getPaymentMethodList,
  initialSearchForm: paymentMethodInitialSearchForm,
  defaultSort: 'sort:asc,id:desc',
  deleteApi: deletePaymentMethod,
  normalizeRows: false,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const searchFields = computed(() =>
  createPaymentMethodSearchFields(t, userStore.config?.paymentGateways)
)
const tableColumns = computed(() => createPaymentMethodTableColumns(t))

const operationActions = computed(() =>
  createCrudActions(t, 'payment_method', {
    onEdit: handleEdit,
    onDelete: handleDelete
  })
)
</script>
