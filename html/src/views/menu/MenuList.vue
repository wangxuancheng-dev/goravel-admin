<template>
  <TreeListPage
    ref="treeListPageRef"
    page-class="menu"
    :title="$t('menu.menu')"
    :add-button-text="$t('menu_management.add_menu')"
    :add-button-disabled="getButtonState('menu.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="menuInitialSearchForm"
    i18n-prefix="menu"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :table-key="`table-${tableColumns.length}-${JSON.stringify(columnOrder)}`"
    :default-expand-all="isExpanded"
    :visible-columns="visibleColumns"
    :all-columns="allColumns"
    :default-visible-columns="defaultVisibleColumns"
    :column-order="columnOrder"
    :fixed-columns="fixedColumns"
    :on-column-setting-confirm="handleColumnSettingConfirm"
    :show-column-setting="false"
    @add="handleAdd"
    @search="handleSearch"
    @reset="handleReset"
    @refresh="loadData"
  >
    <template #header-extra>
      <el-button @click="handleToggleExpand">
        <el-icon><component :is="isExpanded ? Fold : Expand" /></el-icon>
        {{ isExpanded ? $t('menu_management.collapse_all') : $t('menu_management.expand_all') }}
      </el-button>
    </template>

    <template #type="{ row }">
      <el-tag :type="getMenuTypeTagType(row.type)">
        {{ getMenuTypeLabel(t, row.type) }}
      </el-tag>
    </template>

    <template #link_type="{ row }">
      <el-tag :type="row.link_type === 1 ? 'primary' : 'success'">
        {{ row.link_type === 1 ? $t('menu_management.link_type_internal') : $t('menu_management.link_type_external') }}
      </el-tag>
    </template>

    <template #open_type="{ row }">
      <span v-if="row.link_type === 2">
        <el-tag :type="row.open_type === 1 ? 'info' : 'warning'">
          {{ row.open_type === 1 ? $t('menu_management.open_type_iframe') : $t('menu_management.open_type_new_window') }}
        </el-tag>
      </span>
      <span v-else>-</span>
    </template>

    <template #no_cache="{ row }">
      <el-tooltip
        v-if="row.no_cache === 1"
        :content="$t('menu_management.no_cache_no')"
        placement="top"
      >
        <el-tag type="warning">{{ $t('common.no') }}</el-tag>
      </el-tooltip>
      <el-tag v-else type="success">{{ $t('common.yes') }}</el-tag>
    </template>

    <template #icon="{ row }">
      <span v-if="getIconComponent(row.icon)" class="menu-icon-preview">
        <el-icon><component :is="getIconComponent(row.icon)" /></el-icon>
        <span class="menu-icon-name">{{ normalizeIconName(row.icon) }}</span>
      </span>
      <span v-else>-</span>
    </template>

    <template #status="{ row }">
      <el-tag :type="row.status === 1 ? 'success' : 'danger'">
        {{ row.status === 1 ? $t('common.enabled') : $t('common.disabled') }}
      </el-tag>
    </template>

    <template #is_hidden="{ row }">
      <el-tag :type="row.is_hidden === 0 ? 'success' : 'info'">
        {{ row.is_hidden === 0 ? $t('menu_management.is_hidden_show') : $t('menu_management.is_hidden_hide') }}
      </el-tag>
    </template>

    <template #operation="{ row }">
      <el-button
        type="primary"
        link
        :disabled="getButtonState('menu.update').disabled"
        @click="handleEdit(row)"
      >
        {{ $t('common.edit') }}
      </el-button>
      <el-button
        type="danger"
        link
        :disabled="getButtonState('menu.destroy').disabled"
        @click="handleDelete(row)"
      >
        {{ $t('common.delete') }}
      </el-button>
    </template>

    <template #form>
      <MenuForm
        v-model="dialogVisible"
        :edit-id="editId"
        :menu-options="menuOptions"
        @success="handleFormSuccess"
      />
    </template>
  </TreeListPage>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Fold, Expand } from '@element-plus/icons-vue'
import TreeListPage from '@/components/TreeListPage.vue'
import MenuForm from './MenuForm.vue'
import { useStandardTreeListPage } from '@/composables/useStandardTreeListPage'
import { useTreeExpand } from '@/composables/useTreeExpand'
import { useColumnSetting } from '@/composables/useColumnSetting'
import { useElTableColumns } from '@/composables/useElTableColumns'
import { getMenuList, deleteMenu } from '@/api/menu'
import {
  menuInitialSearchForm,
  createMenuSearchFields,
  createMenuTableColumns,
  normalizeIconName,
  getIconComponent,
  mapMenuTreeOptions,
  getMenuTypeTagType,
  getMenuTypeLabel
} from './menu.config'

const { t } = useI18n()
const treeListPageRef = ref(null)

const {
  searchForm,
  tableData,
  loading,
  dialogVisible,
  editId,
  loadData,
  handleSearch,
  handleReset,
  handleAdd,
  handleEdit,
  handleDelete,
  handleFormSuccess,
  getButtonState
} = useStandardTreeListPage({
  fetchApi: getMenuList,
  initialSearchForm: menuInitialSearchForm,
  normalizeRows: false,
  deleteApi: deleteMenu
})

// Default expanded so disabled / nested menus stay easy to find after refresh.
const { isExpanded, handleToggleExpand } = useTreeExpand(treeListPageRef, tableData, true)

const searchFields = computed(() => createMenuSearchFields())
const allTableColumns = computed(() => createMenuTableColumns(t))

const {
  tableColumns: tableColumnsConfig,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('menu', allTableColumns)

const tableColumns = useElTableColumns(tableColumnsConfig, visibleColumns, columnOrder, fixedColumns)
const menuOptions = computed(() => mapMenuTreeOptions(tableData.value))
</script>

<style scoped>
.menu-icon-preview {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.menu-icon-name {
  font-size: 12px;
  color: var(--text-color-regular);
}
</style>
