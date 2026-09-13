<template>
  <div class="wang-editor-wrapper" :class="{ 'dark-mode': isDark }" :style="{ width: normalizedWidth }">
    <Toolbar
      :editor="editorRef"
      :defaultConfig="toolbarConfig"
      :mode="mode"
    />
    <div class="wang-editor-media-bar">
      <el-button size="small" @click="openMediaPicker">
        {{ $t('attachment.select_from_library') }}
      </el-button>
    </div>
    <Editor
      :style="{ height: height + 'px', overflowY: 'hidden' }"
      v-model="valueHtml"
      :defaultConfig="editorConfig"
      :mode="mode"
      @onCreated="handleCreated"
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
import '@wangeditor/editor/dist/css/style.css' // 引入 css
import { onBeforeUnmount, ref, shallowRef, onMounted, watch, computed } from 'vue'
import { Editor, Toolbar } from '@wangeditor/editor-for-vue'
import { ElMessage } from 'element-plus'
import Storage from '../utils/storage'
import { resolveUploadStorageUrl } from '@/utils/attachmentUrl'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '../store/app'
import AttachmentImageField from './AttachmentImageField.vue'

const { t, locale } = useI18n()
const appStore = useAppStore()

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  mode: {
    type: String,
    default: 'default' // 'default' or 'simple'
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
  excludeToolbarKeys: {
    type: Array,
    default: () => ['group-video']
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

// 编辑器实例，必须用 shallowRef
const editorRef = shallowRef()
const mediaPickerRef = ref(null)

// 内容 HTML
const valueHtml = ref('')

const isDark = computed(() => appStore.darkMode)
const normalizedWidth = computed(() =>
  typeof props.width === 'number' ? `${props.width}px` : props.width
)

// 初始化内容
onMounted(() => {
    valueHtml.value = props.modelValue
})

// 监听 props 变化 (外部修改 v-model)
watch(() => props.modelValue, (newVal) => {
    // 1. 基本检查：如果新值与当前绑定值相等，直接跳过
    if (newVal === valueHtml.value) return

    // 2. 深度检查：如果编辑器实例存在，检查编辑器实际内容是否与新值一致
    // 这步至关重要，因为 valueHtml.value 可能因为 v-model 的更新机制略有滞后
    // 而 editor.getHtml() 是编辑器当前的真实状态
    const editor = editorRef.value
    if (editor) {
        const currentHtml = editor.getHtml()
        if (newVal === currentHtml) return
    }

    // 只有确实不一致时才更新，避免 Slate 内部状态混乱导致 "Cannot find a descendant" 错误
    valueHtml.value = newVal
})

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

const uploadHeaders = computed(() => {
  const token = Storage.getItem('token', '') || ''
  return {
    'Authorization': `Bearer ${typeof token === 'string' ? token.trim() : ''}`,
    'Accept-Language': locale.value === 'en-US' ? 'en-US' : 'zh-CN'
  }
})

const toolbarConfig = computed(() => ({
    excludeKeys: props.excludeToolbarKeys
}))
const editorConfig = { 
    placeholder: props.placeholder,
    MENU_CONF: {
        uploadImage: {
            server: uploadAction.value,
            fieldName: 'file',
            headers: uploadHeaders.value,
            maxFileSize: 5 * 1024 * 1024, // 5M
            maxNumberOfFiles: 10,
            allowedFileTypes: ['image/*'],
            metaWithUrl: false,
            withCredentials: false,
            timeout: 10 * 1000, // 10 秒
            meta: {
                is_public: '1'
            },
            customInsert(res, insertFn) {
                if (res.code === 200 && res.data) {
                    const url = resolveUploadStorageUrl(res.data)
                    if (!url) {
                        console.error('Upload error: missing file url', res)
                        return
                    }
                    insertFn(url, res.data.filename, url)
                } else {
                    console.error('Upload error', res)
                }
            }
        }
    }
}

// 监听上传配置变化（如 token 更新）
watch([uploadAction, uploadHeaders], () => {
    if (editorConfig.MENU_CONF.uploadImage) {
        editorConfig.MENU_CONF.uploadImage.server = uploadAction.value
        editorConfig.MENU_CONF.uploadImage.headers = uploadHeaders.value
    }
})

// 组件销毁时，也及时销毁编辑器
onBeforeUnmount(() => {
    const editor = editorRef.value
    if (editor == null) return
    editor.destroy()
})

const handleCreated = (editor) => {
    editorRef.value = editor
}

const handleChange = (editor) => {
    emit('update:modelValue', editor.getHtml())
    emit('change', editor.getHtml())
}

const openMediaPicker = () => {
  mediaPickerRef.value?.openPicker?.()
}

const escapeAttr = (value) =>
  String(value || '')
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

const handleMediaSelect = ({ url, alt }) => {
  const editor = editorRef.value
  if (!editor) return
  if (!url) {
    ElMessage.warning(t('attachment.editor_public_required'))
    return
  }
  const safeUrl = escapeAttr(url)
  const safeAlt = escapeAttr(alt || 'image')
  editor.dangerouslyInsertHtml(`<img src="${safeUrl}" alt="${safeAlt}" />`)
}
</script>

<style scoped>
.wang-editor-wrapper {
  border: 1px solid #ccc;
  border-radius: 4px;
  overflow: hidden;
  transition: border-color 0.3s;
}

.wang-editor-media-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid #ccc;
  background: #fafafa;
}

.wang-editor-wrapper :deep(.w-e-toolbar) {
  border-bottom: 1px solid #ccc;
  transition: background-color 0.3s, border-color 0.3s;
}

.wang-editor-wrapper :deep(.w-e-text-container) {
  transition: background-color 0.3s, color 0.3s;
}

/* ========== 暗黑模式 ========== */

/* --- 外框 --- */
.dark-mode {
  border-color: var(--el-border-color);
}

.dark-mode .wang-editor-media-bar {
  background-color: var(--el-bg-color-overlay);
  border-bottom-color: var(--el-border-color);
}

/* --- 工具栏 --- */
.dark-mode :deep(.w-e-toolbar) {
  background-color: var(--el-bg-color-overlay);
  border-bottom-color: var(--el-border-color);
}

.dark-mode :deep(.w-e-bar-item button) {
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.w-e-bar-item button:hover) {
  background-color: var(--el-fill-color-light);
}

.dark-mode :deep(.w-e-bar-item .active) {
  background-color: var(--el-fill-color);
  color: var(--el-text-color-primary);
}

/* --- 工具栏下拉菜单 / 子菜单 --- */
.dark-mode :deep(.w-e-bar-item-menus-container) {
  background-color: var(--el-bg-color-overlay);
  border-color: var(--el-border-color);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.dark-mode :deep(.w-e-bar-item-menus-container .w-e-bar-item button) {
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.w-e-bar-item-menus-container .w-e-bar-item button:hover) {
  background-color: var(--el-fill-color-light);
}

/* --- Select 下拉列表 (标题、字体等) --- */
.dark-mode :deep(.w-e-drop-down-menu) {
  background-color: var(--el-bg-color-overlay);
  border-color: var(--el-border-color);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.dark-mode :deep(.w-e-drop-down-menu li) {
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.w-e-drop-down-menu li:hover) {
  background-color: var(--el-fill-color-light);
}

/* --- 颜色选择面板 --- */
.dark-mode :deep(.w-e-panel-content-color) {
  background-color: var(--el-bg-color-overlay);
  border-color: var(--el-border-color);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.dark-mode :deep(.w-e-panel-content-color li .color-block) {
  border-color: var(--el-border-color-lighter);
}

.dark-mode :deep(.w-e-panel-content-color .clear-color-btn) {
  color: var(--el-text-color-regular);
  border-color: var(--el-border-color);
}

.dark-mode :deep(.w-e-panel-content-color .clear-color-btn:hover) {
  background-color: var(--el-fill-color-light);
}

/* --- 编辑区域 --- */
.dark-mode :deep(.w-e-text-container) {
  background-color: var(--el-fill-color-blank);
  color: var(--el-text-color-regular);
  border-color: var(--el-border-color);
}

.dark-mode :deep(.w-e-text-container [data-slate-editor]) {
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.w-e-text-container [data-slate-editor] p),
.dark-mode :deep(.w-e-text-container [data-slate-editor] div),
.dark-mode :deep(.w-e-text-container [data-slate-editor] span),
.dark-mode :deep(.w-e-text-container [data-slate-editor] h1),
.dark-mode :deep(.w-e-text-container [data-slate-editor] h2),
.dark-mode :deep(.w-e-text-container [data-slate-editor] h3),
.dark-mode :deep(.w-e-text-container [data-slate-editor] h4),
.dark-mode :deep(.w-e-text-container [data-slate-editor] h5),
.dark-mode :deep(.w-e-text-container [data-slate-editor] li),
.dark-mode :deep(.w-e-text-container [data-slate-editor] td),
.dark-mode :deep(.w-e-text-container [data-slate-editor] th) {
  color: var(--el-text-color-regular);
}

/* --- 占位符 --- */
.dark-mode :deep(.w-e-text-placeholder) {
  color: var(--el-text-color-placeholder);
}

/* --- 链接 --- */
.dark-mode :deep(.w-e-text-container a) {
  color: var(--el-color-primary);
}

/* --- 引用块 --- */
.dark-mode :deep(.w-e-text-container blockquote) {
  border-left-color: var(--el-border-color);
  color: var(--el-text-color-secondary);
  background-color: var(--el-fill-color-light);
}

/* --- 代码 --- */
.dark-mode :deep(.w-e-text-container code) {
  background-color: var(--el-fill-color);
  color: var(--el-color-danger);
}

.dark-mode :deep(.w-e-text-container pre) {
  background-color: var(--el-fill-color-darker);
  border-color: var(--el-border-color);
}

.dark-mode :deep(.w-e-text-container pre code) {
  background-color: transparent;
  color: var(--el-text-color-regular);
}

/* --- 表格 --- */
.dark-mode :deep(.w-e-text-container table) {
  border-color: var(--el-border-color);
}

.dark-mode :deep(.w-e-text-container td),
.dark-mode :deep(.w-e-text-container th) {
  border-color: var(--el-border-color);
}

.dark-mode :deep(.w-e-text-container th) {
  background-color: var(--el-fill-color);
}

/* --- 分割线 --- */
.dark-mode :deep(.w-e-text-container hr) {
  border-color: var(--el-border-color);
}

/* --- 模态框 (插入链接 / 图片 / 表格等) --- */
.dark-mode :deep(.w-e-modal) {
  background-color: var(--el-bg-color-overlay);
  border-color: var(--el-border-color);
  color: var(--el-text-color-regular);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.5);
}

.dark-mode :deep(.w-e-modal .btn-close svg) {
  fill: var(--el-text-color-regular);
}

.dark-mode :deep(.w-e-modal label) {
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.w-e-modal input[type="text"]),
.dark-mode :deep(.w-e-modal input[type="number"]),
.dark-mode :deep(.w-e-modal input[type="url"]),
.dark-mode :deep(.w-e-modal textarea),
.dark-mode :deep(.w-e-modal select) {
  background-color: var(--el-fill-color-blank);
  border-color: var(--el-border-color);
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.w-e-modal input:focus),
.dark-mode :deep(.w-e-modal textarea:focus),
.dark-mode :deep(.w-e-modal select:focus) {
  border-color: var(--el-color-primary);
}

.dark-mode :deep(.w-e-modal button) {
  background-color: var(--el-fill-color);
  color: var(--el-text-color-regular);
  border-color: var(--el-border-color);
}

.dark-mode :deep(.w-e-modal button:hover) {
  background-color: var(--el-fill-color-light);
}

/* --- Hover bar (选中文字时的浮动工具条) --- */
.dark-mode :deep(.w-e-hover-bar) {
  background-color: var(--el-bg-color-overlay);
  border-color: var(--el-border-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
}

.dark-mode :deep(.w-e-hover-bar .w-e-bar-item button) {
  color: var(--el-text-color-regular);
}

.dark-mode :deep(.w-e-hover-bar .w-e-bar-item button:hover) {
  background-color: var(--el-fill-color-light);
}

/* --- 全屏模式 --- */
.dark-mode :deep(.w-e-full-screen-container) {
  background-color: var(--el-bg-color-overlay);
}
</style>
