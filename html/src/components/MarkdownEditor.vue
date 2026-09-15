<template>
  <div class="markdown-editor-wrapper" :class="{ 'dark-mode': isDark }" :style="{ width: normalizedWidth }">
    <div class="markdown-editor-media-bar">
      <el-button size="small" @click="openMediaPicker">
        {{ $t('attachment.select_from_library') }}
      </el-button>
    </div>
    <MdEditor
      ref="mdEditorRef"
      v-model="markdownValue"
      :height="height"
      :placeholder="placeholder"
      :toolbars="toolbars"
      :language="editorLanguage"
      :theme="isDark ? 'dark' : 'light'"
      :sanitize="sanitizeHtml"
      @onUploadImg="handleUploadImg"
      @onChange="handleChange"
    />
    <AttachmentImageField
      ref="mediaPickerRef"
      picker-only
      @select="handleMediaSelect"
    />
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '../store/app'
import axios from 'axios'
import { resolveUploadStorageUrl } from '@/utils/attachmentUrl'
import { buildAdminAuthHeaders } from '@/utils/authHeaders'
import { withTenantQuery } from '@/utils/tenant'
import { sanitizeHtml } from '@/utils/markdown'
import AttachmentImageField from './AttachmentImageField.vue'

const { t, locale } = useI18n()
const appStore = useAppStore()

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  width: {
    type: [String, Number],
    default: '100%'
  },
  height: {
    type: Number,
    default: 300
  },
  placeholder: {
    type: String,
    default: '请输入内容...'
  },
  toolbars: {
    type: Array,
    default: () => [
      'bold', 'underline', 'italic', '-',
      'title', 'strikeThrough', 'sub', 'sup',
      'quote', 'unorderedList', 'orderedList', 'task', '-',
      'codeRow', 'code', 'link', 'image', 'table', '-',
      'revoke', 'next', 'save',
      '-',
      'pageFullscreen', 'fullscreen', 'preview', 'catalog'
    ]
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

const markdownValue = ref(props.modelValue)
const mediaPickerRef = ref(null)
const mdEditorRef = ref(null)
const isDark = computed(() => appStore.darkMode)
const editorLanguage = computed(() => (locale.value === 'en-US' ? 'en-US' : 'zh-CN'))
const normalizedWidth = computed(() =>
  typeof props.width === 'number' ? `${props.width}px` : props.width
)

// 监听 props 变化
watch(() => props.modelValue, (newVal) => {
  if (newVal !== markdownValue.value) {
    markdownValue.value = newVal
  }
})

// 上传配置
const uploadAction = computed(() => {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL
  const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
  if (apiBaseURL) {
    const base = apiBaseURL.replace(/\/+$/, '')
    const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    return `${base}${prefix}/attachments/upload`
  }
  return `${apiPrefix}/attachments/upload`
})

const uploadHeaders = computed(() => buildAdminAuthHeaders())

// 处理图片上传
const handleUploadImg = async (files, callback) => {
  const formData = new FormData()
  const file = files[0]
  
  if (!file) {
    callback([])
    return
  }

  // 检查文件大小（5M）
  if (file.size > 5 * 1024 * 1024) {
    callback([])
    return
  }

  // 检查文件类型
  if (!file.type.startsWith('image/')) {
    callback([])
    return
  }

  formData.append('file', file)
  formData.append('is_public', '1')

  try {
    const response = await axios.post(uploadAction.value, formData, {
      headers: uploadHeaders.value,
      timeout: 10000
    })

    if (response.data.code === 200 && response.data.data) {
      const url = withTenantQuery(resolveUploadStorageUrl(response.data.data))
      callback(url ? [url] : [])
    } else {
      console.error('Upload error', response.data)
      callback([])
    }
  } catch (error) {
    console.error('Upload error', error)
    callback([])
  }
}

// 处理内容变化
const handleChange = (value) => {
  markdownValue.value = value
  emit('update:modelValue', value)
  emit('change', value)
}

const openMediaPicker = () => {
  mediaPickerRef.value?.openPicker?.()
}

const escapeMdAlt = (value) => String(value || 'image').replace(/[[\]]/g, '')

const handleMediaSelect = ({ url, alt }) => {
  const displayUrl = withTenantQuery(url)
  if (!displayUrl) {
    ElMessage.warning(t('attachment.editor_public_required'))
    return
  }
  const snippet = `![${escapeMdAlt(alt)}](${displayUrl})`
  const editor = mdEditorRef.value
  if (editor && typeof editor.insert === 'function') {
    editor.insert(() => ({
      targetValue: snippet,
      select: false,
      deviationStart: 0,
      deviationEnd: 0
    }))
    return
  }
  const next = `${markdownValue.value || ''}${markdownValue.value ? '\n' : ''}${snippet}`
  handleChange(next)
}
</script>

<style scoped>
.markdown-editor-wrapper {
  width: 100%;
  min-width: 0;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  overflow: hidden;
  transition: border-color 0.3s;
}

.markdown-editor-media-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid #dcdfe6;
  background: #fafafa;
}

.markdown-editor-wrapper :deep(.md-editor) {
  width: 100%;
  min-width: 0;
  border: none;
}

.markdown-editor-wrapper :deep(.md-editor-toolbar-wrapper) {
  transition: background-color 0.3s, border-color 0.3s;
}

.markdown-editor-wrapper :deep(.md-editor-input-wrapper),
.markdown-editor-wrapper :deep(.md-editor-preview-wrapper) {
  transition: background-color 0.3s, color 0.3s;
}

/* ========== 暗黑模式 — 覆盖 md-editor-v3 内置 dark 主题的深色 ========== */

/* --- 用 Element Plus 变量覆盖 md-editor-v3 的 CSS 变量 --- */
.dark-mode :deep(.md-editor-dark) {
  --md-bk-color: var(--el-bg-color-overlay);
  --md-bk-color-outstand: var(--el-fill-color-blank);
  --md-bk-hover: var(--el-fill-color-light);
  --md-border-color: var(--el-border-color);
  --md-border-hover-color: var(--el-border-color-lighter);
  --md-border-active-color: var(--el-color-primary);
  --md-color: var(--el-text-color-regular);
  --md-hover-color: var(--el-text-color-primary);
}

/* --- 外框 --- */
.dark-mode {
  border-color: var(--el-border-color);
}

.dark-mode .markdown-editor-media-bar {
  background-color: var(--el-bg-color-overlay);
  border-bottom-color: var(--el-border-color);
}

/* --- 工具栏 --- */
.dark-mode :deep(.md-editor-toolbar-wrapper) {
  background-color: var(--el-bg-color-overlay);
  border-bottom-color: var(--el-border-color);
}

.dark-mode :deep(.md-editor-toolbar li > span) {
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.md-editor-toolbar li > span:hover) {
  background-color: var(--el-fill-color-light);
}

.dark-mode :deep(.md-editor-toolbar .active > span) {
  background-color: var(--el-fill-color);
  color: var(--el-text-color-primary);
}

/* --- 工具栏下拉面板 --- */
.dark-mode :deep(.md-editor-dropdown) {
  background-color: var(--el-bg-color-overlay);
  border-color: var(--el-border-color);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.dark-mode :deep(.md-editor-dropdown li) {
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.md-editor-dropdown li:hover) {
  background-color: var(--el-fill-color-light);
}

/* --- 编辑区域 --- */
.dark-mode :deep(.md-editor-input-wrapper) {
  background-color: var(--el-fill-color-blank);
}

.dark-mode :deep(.md-editor-input-wrapper .inputWrapper textarea),
.dark-mode :deep(.cm-editor),
.dark-mode :deep(.cm-content) {
  color: var(--el-text-color-regular);
  caret-color: var(--el-text-color-regular);
}

.dark-mode :deep(.cm-editor .cm-gutters) {
  background-color: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  color: var(--el-text-color-secondary);
}

.dark-mode :deep(.cm-editor .cm-activeLineGutter) {
  background-color: var(--el-fill-color);
}

.dark-mode :deep(.cm-editor .cm-activeLine) {
  background-color: var(--el-fill-color-light);
}

.dark-mode :deep(.cm-editor .cm-cursor) {
  border-left-color: var(--el-text-color-regular);
}

/* --- 预览区域 --- */
.dark-mode :deep(.md-editor-preview-wrapper) {
  background-color: var(--el-fill-color-blank);
}

.dark-mode :deep(.md-editor-preview) {
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.md-editor-preview h1),
.dark-mode :deep(.md-editor-preview h2),
.dark-mode :deep(.md-editor-preview h3),
.dark-mode :deep(.md-editor-preview h4),
.dark-mode :deep(.md-editor-preview h5),
.dark-mode :deep(.md-editor-preview h6) {
  color: var(--el-text-color-primary);
  border-color: var(--el-border-color);
}

.dark-mode :deep(.md-editor-preview a) {
  color: var(--el-color-primary);
}

.dark-mode :deep(.md-editor-preview blockquote) {
  border-left-color: var(--el-border-color);
  color: var(--el-text-color-secondary);
  background-color: var(--el-fill-color-light);
}

.dark-mode :deep(.md-editor-preview code) {
  background-color: var(--el-fill-color);
  color: var(--el-color-danger);
}

.dark-mode :deep(.md-editor-preview pre) {
  background-color: var(--el-fill-color-darker);
  border-color: var(--el-border-color);
}

.dark-mode :deep(.md-editor-preview pre code) {
  background-color: transparent;
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.md-editor-preview table) {
  border-color: var(--el-border-color);
}

.dark-mode :deep(.md-editor-preview td),
.dark-mode :deep(.md-editor-preview th) {
  border-color: var(--el-border-color);
}

.dark-mode :deep(.md-editor-preview th) {
  background-color: var(--el-fill-color);
}

.dark-mode :deep(.md-editor-preview hr) {
  border-color: var(--el-border-color);
}

/* --- 占位符 --- */
.dark-mode :deep(.md-editor-input-wrapper .placeholder-wrapper) {
  color: var(--el-text-color-placeholder);
}

/* --- 底部状态栏 --- */
.dark-mode :deep(.md-editor-footer) {
  background-color: var(--el-bg-color-overlay);
  border-top-color: var(--el-border-color);
  color: var(--el-text-color-secondary);
}

/* --- 弹窗 / 模态框 --- */
.dark-mode :deep(.md-editor-modal) {
  background-color: var(--el-bg-color-overlay);
  border-color: var(--el-border-color);
  color: var(--el-text-color-regular);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.5);
}

.dark-mode :deep(.md-editor-modal input),
.dark-mode :deep(.md-editor-modal textarea),
.dark-mode :deep(.md-editor-modal select) {
  background-color: var(--el-fill-color-blank);
  border-color: var(--el-border-color);
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.md-editor-modal input:focus),
.dark-mode :deep(.md-editor-modal textarea:focus) {
  border-color: var(--el-color-primary);
}

.dark-mode :deep(.md-editor-modal button) {
  background-color: var(--el-fill-color);
  color: var(--el-text-color-regular);
  border-color: var(--el-border-color);
}

.dark-mode :deep(.md-editor-modal button:hover) {
  background-color: var(--el-fill-color-light);
}

/* --- 目录 (catalog) --- */
.dark-mode :deep(.md-editor-catalog-editor) {
  background-color: var(--el-bg-color-overlay);
  border-color: var(--el-border-color);
}

.dark-mode :deep(.md-editor-catalog-editor a) {
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.md-editor-catalog-editor a:hover),
.dark-mode :deep(.md-editor-catalog-editor .active > a) {
  color: var(--el-color-primary);
}

/* --- 编辑 / 预览分隔线 --- */
.dark-mode :deep(.md-editor-resize-operate) {
  border-color: var(--el-border-color);
}

/* --- 全屏模式 --- */
.dark-mode :deep(.md-editor-fullscreen) {
  background-color: var(--el-bg-color-overlay);
}

/* --- 滚动条 --- */
.dark-mode :deep(.md-editor ::-webkit-scrollbar-thumb) {
  background-color: var(--el-fill-color);
}

.dark-mode :deep(.md-editor ::-webkit-scrollbar-thumb:hover) {
  background-color: var(--el-fill-color-dark);
}

.dark-mode :deep(.md-editor ::-webkit-scrollbar-track) {
  background-color: var(--el-bg-color-overlay);
}
</style>
