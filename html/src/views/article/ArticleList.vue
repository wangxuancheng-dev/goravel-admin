<template>
  <ListPage
    ref="listPageRef"
    page-class="article"
    :title="$t('menu.article')"
    :add-button-text="$t('common.add')"
    :add-button-disabled="getButtonState('article.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="articleInitialSearchForm"
    i18n-prefix="article"
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
        :disabled="getButtonState('article.export').disabled || isExporting"
        :loading="isExporting"
        @click="handleExport"
      >
        {{ $t("common.export") }}
      </el-button>
    </template>

    <template #admin_id="{ row }">
      {{ getadminDisplayName(row.admin || row.admin_id) }}
    </template>
    <template #content="{ row }">
      <div class="text-truncate" :title="extractTextFromMarkdown(row.content)">
        {{ extractTextFromMarkdown(row.content).slice(0, 120) || "-" }}
      </div>
    </template>

    <template #operation="{ row }">
      <TableActionButtons
        :row="row"
        :primary-actions="operationActions"
        :get-button-state="getButtonState"
      />
    </template>

    <template #form>
      <ArticleForm
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
import ArticleForm from "./ArticleForm.vue";
import { useStandardListPage } from "@/composables/useStandardListPage";
import { createCrudActions } from "@/utils/listPageHelpers";

import { extractTextFromMarkdown } from "@/utils/markdown";

import { exportArticle } from "@/api/article";

import { getArticleList, deleteArticle, updateArticle } from "@/api/article";
import logger from "@/utils/logger";
import ErrorHandler from "@/utils/errorHandler";
import {
  articleInitialSearchForm,
  buildArticleListParams,
  createArticleSearchFields,
  createArticleTableColumns,
  getadminDisplayName,
} from "./article.config";

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
  fetchApi: getArticleList,
  initialSearchForm: articleInitialSearchForm,
  buildParams: buildArticleListParams,
  defaultSort: "id:desc",
  deleteApi: deleteArticle,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef),
  normalizeRows: false,
});

const hasSelection = computed(() => selectedIds.value.length > 0);
const searchFields = computed(() => createArticleSearchFields(t));
const tableColumns = computed(() =>
  createArticleTableColumns(t, { enableBatchActions: false }),
);

const operationActions = computed(() =>
  createCrudActions(t, "article", {
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
    await Promise.all(selectedIds.value.map((id) => deleteArticle(id)));
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
    const response = await exportArticle(searchForm);
    const exportId = response?.data?.export_id || response?.data?.id;
    ElMessage.success(
      exportId ? t("export.task_submitted") : t("common.operation_success"),
    );
    router.push("/exports");
  } catch (error) {
    logger.error("Export error:", error);
    if (error.response?.status === 429) {
      ElMessage.warning(t("common.already_queued"));
    } else if (!error.__handled) {
      ErrorHandler.handle(error, { silent: true });
    }
  } finally {
    isExporting.value = false;
  }
};
</script>

<style scoped>
.text-truncate {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
