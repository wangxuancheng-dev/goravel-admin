<template>
  <ListPage
    ref="listPageRef"
    page-class="quote"
    :title="$t('menu.quote')"
    :add-button-text="$t('common.add')"
    :add-button-disabled="getButtonState('quote.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="quoteInitialSearchForm"
    i18n-prefix="quote"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :pagination="pagination"
    :show-toolbar="true"
    @add="handleAdd"
    @search="handleSearch"
    @reset="handleReset"
    @refresh="loadData"
    @page-change="loadData"
    @sort-change="handleSortChange"
  >
    <template #extra-buttons>
      <el-button
        type="success"
        :disabled="getButtonState('quote.export').disabled || isExporting"
        :loading="isExporting"
        @click="handleExport"
      >
        {{ $t("common.export") }}
      </el-button>
    </template>

    <template #operation="{ row }">
      <TableActionButtons
        :row="row"
        :primary-actions="operationActions"
        :get-button-state="getButtonState"
      />
    </template>

    <template #form>
      <QuoteForm
        v-model="dialogVisible"
        :edit-id="editId"
        @success="handleFormSuccess"
      />
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ElMessage, ElMessageBox } from "element-plus";
import ListPage from "@/components/ListPage.vue";
import TableActionButtons from "@/components/TableActionButtons.vue";
import QuoteForm from "./QuoteForm.vue";
import { useStandardListPage } from "@/composables/useStandardListPage";
import { createCrudActions } from "@/utils/listPageHelpers";

import { exportQuote } from "@/api/quote";

import { getQuoteList, deleteQuote, updateQuote } from "@/api/quote";
import logger from "@/utils/logger";
import ErrorHandler from "@/utils/errorHandler";
import {
  quoteInitialSearchForm,
  buildQuoteListParams,
  createQuoteSearchFields,
  createQuoteTableColumns,
} from "./quote.config";

const { t } = useI18n();
const router = useRouter();
const listPageRef = ref(null);

const isExporting = ref(false);

const {
  pagination,
  tableData,
  loading,
  searchForm,
  selectedIds,
  dialogVisible,
  editId,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
  handleSelectionChange,
  handleAdd,
  handleEdit,
  handleFormSuccess,
  handleDelete,
  getButtonState,
} = useStandardListPage({
  fetchApi: getQuoteList,
  initialSearchForm: quoteInitialSearchForm,
  buildParams: buildQuoteListParams,
  defaultSort: "id:desc",
  deleteApi: deleteQuote,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef),
});

const hasSelection = computed(() => selectedIds.value.length > 0);
const searchFields = computed(() => createQuoteSearchFields(t));
const tableColumns = computed(() =>
  createQuoteTableColumns(t, { enableBatchActions: false }),
);

const operationActions = computed(() =>
  createCrudActions(t, "quote", {
    onEdit: handleEdit,
    onDelete: handleDelete,
  }),
);

const handleBatchDelete = async () => {
  if (!selectedIds.value.length) return;
  try {
    await ElMessageBox.confirm(
      t("common.batch_delete_confirm", { count: selectedIds.value.length }),
      t("common.warning"),
      { type: "warning" },
    );
    await Promise.all(selectedIds.value.map((id) => deleteQuote(id)));
    ElMessage.success(t("common.operation_success"));
    await loadData();
  } catch (error) {
    if (error === "cancel" || error === "close") return;
    logger.error("Batch delete error:", error);
  }
};

const handleExport = async () => {
  if (isExporting.value) return;
  isExporting.value = true;
  try {
    const response = await exportQuote(searchForm);
    const fileUrl = response?.data?.file_url;
    if (fileUrl) {
      window.open(fileUrl, "_blank");
      ElMessage.success(t("export.success"));
    } else {
      ElMessage.success(t("export.success"));
      router.push("/exports");
    }
  } catch (error) {
    logger.error("Export error:", error);
    if (error.response?.status === 429) {
      ElMessage.warning(t("common.already_queued"));
    } else if (!error.__handled) {
      ElMessage.error(t("export.failed"));
      ErrorHandler.handle(error, { silent: true });
    }
  } finally {
    isExporting.value = false;
  }
};
</script>
