<template>
  <div class="notification-page">
    <ListPage
      ref="listPageRef"
      page-class="notification"
      :title="$t('notification.center')"
      :show-add-button="false"
      :search-form="searchForm"
      :search-fields="searchFields"
      :initial-search-values="notificationInitialSearchForm"
      i18n-prefix="notification"
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
      <template #header-actions>
        <el-button
          type="primary"
          :disabled="!canCreate"
          @click="handleAdd"
        >
          <el-icon><Plus /></el-icon>
          {{ $t('notification.create') }}
        </el-button>
        <el-button
          type="primary"
          plain
          :disabled="notificationStore.unreadCount === 0"
          @click="handleMarkAll"
        >
          {{ $t('notification.mark_all') }}
        </el-button>
      </template>

      <template #content="{ row }">
        <div class="text-truncate" :title="extractTextFromMarkdown(row.content)">
          {{ extractTextFromMarkdown(row.content) }}
        </div>
      </template>

      <template #type="{ row }">
        <el-tag size="small">
          {{ getNotificationTypeLabel(t, row.type) }}
        </el-tag>
      </template>

      <template #sender="{ row }">
        <template v-if="row.type === 'message' && row.sender_id === userStore.adminInfo?.id">
          <span class="text-gray-500">
            {{ $t('notification.sent_to') }}:
            <span v-if="row.receiver" class="text-blue-600">
              {{ row.receiver.nickname || row.receiver.username }}
            </span>
            <span v-else class="text-gray-400">-</span>
          </span>
        </template>
        <template v-else>
          <span v-if="row.sender">
            {{ row.sender.nickname || row.sender.username }}
          </span>
          <span v-else class="text-gray-400">
            {{ $t('notification.system') }}
          </span>
        </template>
      </template>

      <template #is_read="{ row }">
        <el-tag
          size="small"
          :type="row.is_read ? 'info' : 'danger'"
          effect="plain"
        >
          {{ row.is_read ? $t('notification.read') : $t('notification.unread') }}
        </el-tag>
      </template>

      <template #created_at="{ row }">
        {{ formatDate(row.created_at) }}
      </template>

      <template #operation="{ row }">
        <el-button type="primary" link @click="handleView(row)">
          {{ $t('common.view') }}
        </el-button>
        <el-button
          v-if="!row.is_read"
          type="primary"
          link
          @click="handleMarkRead(row)"
        >
          {{ $t('notification.mark_read') }}
        </el-button>
      </template>

      <template #form>
        <NotificationForm
          v-model="dialogVisible"
          @success="handleFormSuccess"
        />
      </template>
    </ListPage>

    <el-dialog
      v-model="viewDialogVisible"
      :title="$t('notification.detail')"
      width="800px"
    >
      <div v-if="currentNotification">
        <h3 class="detail-title">{{ currentNotification.title }}</h3>
        <div class="detail-meta">
          <el-tag size="small" class="mr-4">
            {{ getNotificationTypeLabel(t, currentNotification.type) }}
          </el-tag>
          <span class="mr-4">{{ formatDate(currentNotification.created_at) }}</span>
          <span>
            {{ $t('notification.table.sender') }}:
            {{ currentNotification.sender?.nickname || currentNotification.sender?.username || $t('notification.system') }}
          </span>
        </div>
        <RichContentHtml
          content-class="rich-text-content-view markdown-content"
          :content="currentNotification.content"
        />
      </div>
      <template #footer>
        <el-button @click="viewDialogVisible = false">
          {{ $t('common.close') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import dayjs from 'dayjs'
import { Plus } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import ListPage from '@/components/ListPage.vue'
import NotificationForm from './NotificationForm.vue'
import RichContentHtml from '@/components/RichContentHtml.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { useNotificationStore } from '@/store/notification'
import { useUserStore } from '@/store/user'
import { getNotificationList } from '@/api/notification'
import { extractTextFromMarkdown } from '@/utils/markdown'
import {
  notificationInitialSearchForm,
  transformNotificationRow,
  createNotificationSearchFields,
  createNotificationTableColumns,
  buildNotificationParams,
  getNotificationTypeLabel
} from './notification.config'

const { t } = useI18n()
const notificationStore = useNotificationStore()
const userStore = useUserStore()
const canCreate = computed(() => userStore.shouldShowButton('notification.store'))

const listPageRef = ref(null)
const viewDialogVisible = ref(false)
const currentNotification = ref(null)

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
  handleAdd,
} = useStandardListPage({
  fetchApi: getNotificationList,
  initialSearchForm: notificationInitialSearchForm,
  defaultSort: 'id:desc',
  normalizeRows: false,
  transformData: transformNotificationRow,
  buildParams: buildNotificationParams,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef),
  onLoadSuccess: (res) => {
    if (res?.data?.unread_count !== undefined) {
      notificationStore.unreadCount = res.data.unread_count || 0
    }
  }
})

const searchFields = computed(() => createNotificationSearchFields(t))
const tableColumns = computed(() => createNotificationTableColumns(t))

const handleView = (row) => {
  currentNotification.value = { ...row }
  viewDialogVisible.value = true
  if (!row.is_read) {
    handleMarkRead(row)
  }
}

const handleMarkRead = async (row) => {
  await notificationStore.markAsRead(row.id)
  row.is_read = true
  row.read_at = new Date().toISOString()
}

const handleMarkAll = async () => {
  await notificationStore.markAllRead()
  await loadData()
}

const formatDate = (value) => {
  if (!value) return ''
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss')
}

const handleFormSuccess = async () => {
  pagination.page = 1
  await loadData()
}
</script>

<style scoped>
.notification-page {
  padding: var(--card-padding);
}

.detail-title {
  font-size: 1.125rem;
  font-weight: 700;
  margin-bottom: 1rem;
}

.detail-meta {
  display: flex;
  align-items: center;
  color: #6b7280;
  margin-bottom: 1rem;
  font-size: 0.875rem;
}

.mr-4 {
  margin-right: 1rem;
}

.text-truncate {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  width: 100%;
  color: var(--text-color-regular);
}

html.dark .text-truncate {
  color: var(--el-text-color-regular, #cfd3dc);
}

.rich-text-content-view {
  min-height: 100px;
  max-height: 60vh;
  overflow-y: auto;
  white-space: normal;
  padding: var(--space-sm);
  border: 1px solid var(--border-color-light);
  border-radius: var(--border-radius-sm);
}

html.dark .rich-text-content-view {
  background-color: var(--el-bg-color, #1d1e1f);
  border-color: var(--el-border-color, #3d3e40);
  color: var(--el-text-color-regular, #cfd3dc);
}

.rich-text-content-view :deep(p) {
  margin: 0 0 var(--space-sm);
}

.rich-text-content-view :deep(img) {
  max-width: 100%;
  height: auto;
  display: block;
  margin: var(--space-sm) 0;
}

.markdown-content :deep(p) {
  margin: 0 0 var(--space-sm);
  line-height: 1.6;
}

.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3),
.markdown-content :deep(h4),
.markdown-content :deep(h5),
.markdown-content :deep(h6) {
  margin: var(--space-md) 0 var(--space-xs);
  font-weight: 600;
}

.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  margin: var(--space-sm) 0;
  padding-left: 30px;
}

.markdown-content :deep(li) {
  margin: 4px 0;
}

.markdown-content :deep(blockquote) {
  margin: var(--space-sm) 0;
  padding: var(--space-sm) 15px;
  border-left: 4px solid var(--el-color-primary);
  background-color: var(--bg-color-tertiary);
  color: var(--text-color-regular);
}

.markdown-content :deep(code) {
  background-color: var(--bg-color-tertiary);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  font-size: 0.9em;
}

.markdown-content :deep(pre) {
  background-color: var(--bg-color-tertiary);
  padding: var(--space-sm);
  border-radius: var(--border-radius-sm);
  overflow-x: auto;
  margin: var(--space-sm) 0;
}

.markdown-content :deep(pre code) {
  background-color: transparent;
  padding: 0;
}

.markdown-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: var(--space-sm) 0;
}

.markdown-content :deep(th),
.markdown-content :deep(td) {
  border: 1px solid var(--border-color-lighter);
  padding: 8px 12px;
  text-align: left;
}

.markdown-content :deep(th) {
  background-color: var(--bg-color-tertiary);
  font-weight: 600;
}
</style>
