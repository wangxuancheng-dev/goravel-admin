<template>
  <div class="attachment-image-field" :class="{ 'is-picker-only': pickerOnly }">
    <el-input
      v-if="!pickerOnly"
      :model-value="modelValue"
      :placeholder="placeholder || $t('config.site_logo_placeholder')"
      clearable
      @update:model-value="emitValue"
    >
      <template #append>
        <el-button @click="openPicker">
          <el-icon><Picture /></el-icon>
          {{ $t('common.select_from_attachment') }}
        </el-button>
      </template>
    </el-input>

    <div v-if="!pickerOnly && displayPreviewUrl" class="logo-preview">
      <el-image
        :src="displayPreviewUrl"
        :preview-src-list="[displayPreviewUrl]"
        fit="contain"
        class="logo-preview-image"
        :preview-teleported="true"
      >
        <template #error>
          <div class="logo-preview-error">
            <el-icon><Picture /></el-icon>
          </div>
        </template>
      </el-image>
    </div>

    <el-dialog
      v-model="pickerVisible"
      :title="$t('common.select_from_attachment')"
      width="780px"
      append-to-body
      destroy-on-close
      @opened="handlePickerOpened"
      @closed="handlePickerClosed"
    >
      <div class="picker-toolbar">
        <el-select
          v-model="categoryId"
          clearable
          :placeholder="$t('attachment.category')"
          style="width: 160px"
          @change="handleSearch"
        >
          <el-option
            v-for="opt in categoryOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>
        <el-input
          v-model="keyword"
          clearable
          :placeholder="$t('attachment.search_name_placeholder')"
          style="width: 280px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" @click="handleSearch">{{ $t('common.search') }}</el-button>
        <el-button @click="handleReset">{{ $t('common.reset') }}</el-button>
      </div>

      <div v-loading="loading" class="picker-grid">
        <div
          v-for="item in list"
          :key="item.id"
          class="picker-item"
          :class="{ active: selectedId === item.id }"
          @click="selectedId = item.id"
          @dblclick="confirmSelect(item)"
        >
          <el-image
            v-if="getImageUrl(item)"
            :src="getImageUrl(item)"
            fit="cover"
            class="picker-thumb"
            @load="handleImageLoad(item)"
            @error="handleImageError(item)"
          >
            <template #error>
              <div class="picker-thumb-error">
                <el-icon><Picture /></el-icon>
              </div>
            </template>
          </el-image>
          <div v-else class="picker-thumb-error">
            <el-icon v-if="getImageLoadingState(item) === 'loading'" class="is-loading"><Loading /></el-icon>
            <el-icon v-else><Picture /></el-icon>
          </div>
          <div class="picker-name" :title="getPickerNameTitle(item)">
            <div class="picker-name-main">{{ item.display_name || item.filename }}</div>
            <div v-if="item.display_name" class="picker-name-sub">{{ item.filename }}</div>
          </div>
        </div>
        <el-empty v-if="!loading && list.length === 0" :description="$t('common.no_data')" />
      </div>

      <div class="picker-pager">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="handlePageChange"
        />
      </div>

      <template #footer>
        <el-button @click="pickerVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :disabled="!selectedItem" @click="confirmSelect(selectedItem)">
          {{ $t('common.confirm') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onBeforeUnmount } from 'vue'
import { Picture, Loading } from '@element-plus/icons-vue'
import { getAttachmentList } from '@/api/attachment'
import { transformAttachmentRow } from '@/views/attachment/attachment.config'
import { useAttachmentImagePreview } from '@/composables/useAttachmentImagePreview'
import { PUBLIC_IMAGE_PATH_RE, resolveImageDisplayUrl } from '@/utils/publicImage'
import { toStablePublicAttachmentUrl } from '@/utils/attachmentUrl'
import request from '@/utils/request'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  placeholder: {
    type: String,
    default: ''
  },
  /** When true, only the picker dialog is rendered (for editors). */
  pickerOnly: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'change', 'select'])

const pickerVisible = ref(false)
const loading = ref(false)
const list = ref([])
const keyword = ref('')
const categoryId = ref('')
const categoryOptions = ref([])
const page = ref(1)
const pageSize = ref(12)
const total = ref(0)
const selectedId = ref(null)
const displayPreviewUrl = ref('')
let revokePreviewUrl = null

const {
  loadImageAsBlob,
  getImageUrl,
  getImageLoadingState,
  handleImageLoad,
  handleImageError
} = useAttachmentImagePreview()

const selectedItem = computed(() => list.value.find((item) => item.id === selectedId.value) || null)

const toSubmitUrl = (url) => {
  const value = String(url || '').trim()
  if (!value) return ''
  if (value.startsWith('/')) return value
  if (value.startsWith('http')) {
    try {
      const parsed = new URL(value)
      if (PUBLIC_IMAGE_PATH_RE.test(parsed.pathname)) {
        return `${parsed.pathname}${parsed.search || ''}`
      }
      return value
    } catch {
      return value
    }
  }
  return value
}

const emitValue = (value) => {
  const finalValue = toSubmitUrl(value)
  emit('update:modelValue', finalValue)
  emit('change', finalValue)
}

const revokePreviewBlob = () => {
  if (typeof revokePreviewUrl === 'function') {
    revokePreviewUrl()
    revokePreviewUrl = null
  }
  displayPreviewUrl.value = ''
}

const loadPreview = async (raw) => {
  revokePreviewBlob()
  const value = String(raw || '').trim()
  if (!value) return

  const result = await resolveImageDisplayUrl(value)
  if (String(props.modelValue || '').trim() !== value) {
    if (typeof result.revoke === 'function') result.revoke()
    return
  }
  revokePreviewUrl = result.revoke || null
  displayPreviewUrl.value = result.url || ''
}

watch(
  () => props.modelValue,
  (val) => {
    if (props.pickerOnly) return
    loadPreview(val)
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  revokePreviewBlob()
})

const ensureFileUrl = (item) => {
  if (!item) return ''
  if (item.file_url) return item.file_url
  if (item.id) return `/api/admin/public/images/${item.id}`
  return ''
}

/** 配置类长期引用：统一存公开预览路径，避免 S3 临时签名链过期 */
const toStablePublicUrl = (item) => toStablePublicAttachmentUrl(item)

const loadCategoryOptions = async () => {
  try {
    const res = await request({ url: '/options', method: 'get', params: { type: 'attachment_category' } })
    categoryOptions.value = res?.data?.options || []
  } catch {
    categoryOptions.value = []
  }
}

const loadList = async () => {
  loading.value = true
  try {
    const res = await getAttachmentList({
      page: page.value,
      page_size: pageSize.value,
      file_type: 'image',
      is_public: '1',
      keyword: keyword.value.trim() || undefined,
      category_id: categoryId.value || undefined
    })
    const rows = res?.data?.list || res?.data?.data || []
    list.value = rows.map((row) => {
      const item = transformAttachmentRow(row)
      item.file_url = ensureFileUrl(item)
      return item
    })
    total.value = Number(res?.data?.total || 0)
    await nextTick()
    list.value.forEach((item) => {
      if (item.file_type === 'image') {
        loadImageAsBlob(item)
      }
    })
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const getPickerNameTitle = (item) => {
  if (!item) return ''
  if (item.display_name) return `${item.display_name}\n${item.filename}`
  return item.filename || ''
}

const openPicker = () => {
  selectedId.value = null
  pickerVisible.value = true
}

const handlePickerOpened = () => {
  page.value = 1
  loadCategoryOptions()
  loadList()
}

const handlePickerClosed = () => {
  list.value = []
  selectedId.value = null
}

const handleSearch = () => {
  page.value = 1
  selectedId.value = null
  loadList()
}

const handleReset = () => {
  keyword.value = ''
  categoryId.value = ''
  page.value = 1
  selectedId.value = null
  loadList()
}

const handlePageChange = (newPage) => {
  page.value = newPage
  selectedId.value = null
  loadList()
}

const confirmSelect = (item) => {
  if (!item) return
  const url = toSubmitUrl(toStablePublicUrl(item))
  const alt = item.display_name || item.filename || 'image'
  emit('select', { url, alt, item })
  if (!props.pickerOnly) {
    emitValue(url)
  }
  pickerVisible.value = false
}

defineExpose({ openPicker })
</script>

<style scoped>
.attachment-image-field {
  width: 100%;
  max-width: 480px;
}

.attachment-image-field.is-picker-only {
  width: 0;
  height: 0;
  max-width: none;
  overflow: hidden;
}

.logo-preview {
  margin-top: 12px;
}

.logo-preview-image {
  width: 120px;
  height: 120px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: var(--el-fill-color-lighter);
}

.logo-preview-error {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-placeholder);
  font-size: 28px;
}

.picker-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.picker-grid {
  min-height: 280px;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.picker-item {
  border: 2px solid transparent;
  border-radius: 6px;
  padding: 8px;
  cursor: pointer;
  background: var(--el-fill-color-lighter);
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.picker-item:hover {
  border-color: var(--el-color-primary-light-5);
}

.picker-item.active {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px var(--el-color-primary-light-7);
}

.picker-thumb {
  width: 100%;
  height: 110px;
  border-radius: 4px;
  display: block;
}

.picker-thumb-error {
  width: 100%;
  height: 110px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-placeholder);
  background: var(--el-fill-color);
  border-radius: 4px;
  font-size: 28px;
}

.picker-name {
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--el-text-color-regular);
}

.picker-name-main {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.picker-name-sub {
  margin-top: 2px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.picker-pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

:deep(.el-empty) {
  grid-column: 1 / -1;
}
</style>
