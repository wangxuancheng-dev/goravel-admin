<template>
  <el-popover
    v-model:visible="visible"
    placement="bottom-end"
    :width="400"
    trigger="click"
    popper-class="menu-search-popover"
    :teleported="true"
  >
    <template #reference>
      <el-button
        type="text"
        class="header-btn topbar-icon-btn menu-search-trigger"
        :title="triggerTitle"
      >
        <el-icon class="header-icon-fixed topbar-icon"><Search /></el-icon>
        <kbd class="menu-search-kbd">{{ shortcutLabel }}</kbd>
      </el-button>
    </template>

      <div class="menu-search-content">
        <el-input
          v-model="searchKeyword"
          :placeholder="$t('header.menu_search_placeholder')"
          clearable
          @input="handleSearch"
          @keydown.enter="handleEnter"
          @keydown.down.prevent="handleArrowDown"
          @keydown.up.prevent="handleArrowUp"
          @keydown.esc.stop="visible = false"
          ref="inputRef"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>

        <div class="menu-search-results" v-if="filteredMenus.length > 0">
          <div
            v-for="(menu, index) in filteredMenus"
            :key="menu.id || menu.ID || index"
            class="menu-search-item"
            :class="{ 'is-active': index === selectedIndex }"
            @click="handleMenuClick(menu)"
            @mouseenter="selectedIndex = index"
          >
            <el-icon v-if="getIcon(menu.icon || menu.Icon)" class="menu-item-icon">
              <component :is="getIcon(menu.icon || menu.Icon)" />
            </el-icon>
            <span class="menu-item-title">{{ getMenuTitle(menu) }}</span>
            <span v-if="menu.path || menu.Path" class="menu-item-path">{{ menu.path || menu.Path }}</span>
          </div>
        </div>

        <div v-else class="menu-search-empty">
          {{ searchKeyword ? $t('header.menu_search_no_results') : $t('header.menu_search_hint') }}
        </div>

        <div class="menu-search-footer">
          {{ $t('header.menu_search_shortcut_hint', { shortcut: shortcutLabel }) }}
        </div>
      </div>
    </el-popover>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Search } from '@element-plus/icons-vue'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import { getMenuTitle as getMenuTitleUtil } from '../utils/menuTranslation'

const router = useRouter()
const { t, te } = useI18n()

const props = defineProps({
  menus: {
    type: Array,
    default: () => []
  }
})

const visible = ref(false)
const searchKeyword = ref('')
const selectedIndex = ref(-1)
const inputRef = ref(null)

const isApplePlatform = () => {
  if (typeof navigator === 'undefined') return false
  const platform = navigator.platform || ''
  const ua = navigator.userAgent || ''
  return /Mac|iPhone|iPad|iPod/i.test(platform) || /Mac OS X/i.test(ua)
}

const shortcutLabel = computed(() => (isApplePlatform() ? '\u2318K' : 'Ctrl+K'))
const triggerTitle = computed(() => `${t('header.menu_search')} (${shortcutLabel.value})`)

const onGlobalKeydown = (event) => {
  const key = String(event.key || '').toLowerCase()
  if (key === 'k' && (event.metaKey || event.ctrlKey) && !event.altKey) {
    event.preventDefault()
    visible.value = !visible.value
    return
  }
  if (key === 'escape' && visible.value) {
    visible.value = false
  }
}

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onGlobalKeydown)
})

// Flatten menu tree for search (type=2 leaf menus with path only).
const flattenMenus = (menus, result = []) => {
  menus.forEach((menu) => {
    const menuType = menu.type !== undefined ? menu.type : (menu.Type !== undefined ? menu.Type : 1)
    const path = (menu.path || menu.Path || '').trim()
    if (menuType === 2 && path !== '') {
      result.push(menu)
    }
    const children = menu.children || menu.Children
    if (children && children.length > 0) {
      flattenMenus(children, result)
    }
  })
  return result
}

// 所有可搜索的菜单
const allMenus = computed(() => {
  return flattenMenus(props.menus || [])
})

// 过滤后的菜单
const filteredMenus = computed(() => {
  if (!searchKeyword.value || searchKeyword.value.trim() === '') {
    return allMenus.value.slice(0, 10)
  }

  const keyword = searchKeyword.value.toLowerCase().trim()
  return allMenus.value.filter(menu => {
    const title = getMenuTitle(menu).toLowerCase()
    const path = (menu.path || menu.Path || '').toLowerCase()
    const slug = ((menu.slug || menu.Slug) || '').toLowerCase()

    return title.includes(keyword) ||
           path.includes(keyword) ||
           slug.includes(keyword)
  }).slice(0, 10)
})

// 获取菜单标题
const getMenuTitle = (menu) => {
  return getMenuTitleUtil(t, te, menu)
}

// 获取图标
const normalizeIconName = (iconName) => {
  if (!iconName) return ''
  const trimmed = iconName.trim()
  if (!trimmed) return ''
  if (ElementPlusIconsVue[trimmed]) return trimmed
  const pascalCase = trimmed.charAt(0).toUpperCase() + trimmed.slice(1)
  if (ElementPlusIconsVue[pascalCase]) return pascalCase
  return ''
}

const getIcon = (iconName) => {
  const normalized = normalizeIconName(iconName)
  return normalized ? ElementPlusIconsVue[normalized] : null
}

// 处理搜索
const handleSearch = () => {
  selectedIndex.value = -1
}

// 处理菜单点击
const handleMenuClick = (menu) => {
  // 检查菜单类型和路径
  const menuType = menu.type !== undefined ? menu.type : (menu.Type !== undefined ? menu.Type : 1)
  
  // 如果是目录类型（type === 1）或按钮类型（type === 3），不处理
  if (menuType === 1 || menuType === 3) {
    return
  }
  
  if (!menu.path && !menu.Path) {
    return
  }
  const path = (menu.path || menu.Path || '').trim()
  if (!path) {
    return
  }

  const linkType = menu.link_type !== undefined ? menu.link_type : (menu.LinkType !== undefined ? menu.LinkType : 1)
  const openType = menu.open_type !== undefined ? menu.open_type : (menu.OpenType !== undefined ? menu.OpenType : 1)

  if (linkType === 2) {
    if (openType === 1) {
      const title = getMenuTitle(menu)
      const iframePath = `/iframe?url=${encodeURIComponent(path)}&title=${encodeURIComponent(title)}`
      router.push(iframePath)
    } else if (openType === 2) {
      window.open(path, '_blank')
    }
  } else {
    router.push(path)
  }

  visible.value = false
  searchKeyword.value = ''
}

// 处理回车键
const handleEnter = () => {
  if (selectedIndex.value >= 0 && filteredMenus.value[selectedIndex.value]) {
    handleMenuClick(filteredMenus.value[selectedIndex.value])
  } else if (filteredMenus.value.length > 0) {
    handleMenuClick(filteredMenus.value[0])
  }
}

// 处理下箭头
const handleArrowDown = () => {
  if (selectedIndex.value < filteredMenus.value.length - 1) {
    selectedIndex.value++
  } else {
    selectedIndex.value = 0
  }
}

// 处理上箭头
const handleArrowUp = () => {
  if (selectedIndex.value > 0) {
    selectedIndex.value--
  } else {
    selectedIndex.value = filteredMenus.value.length - 1
  }
}

// 监听弹窗打开，自动聚焦输入框
watch(visible, (newVal) => {
  if (newVal) {
    nextTick(() => {
      if (inputRef.value && inputRef.value.focus) {
        inputRef.value.focus()
      }
    })
  } else {
    searchKeyword.value = ''
    selectedIndex.value = -1
  }
})
</script>

<style scoped>
.topbar-icon-btn {
  color: var(--text-color-regular);
  padding: 8px;
  border-radius: 10px;
  transition: all 0.25s ease;
}

.topbar-icon {
  color: currentColor;
  transition: transform 0.25s ease, color 0.25s ease;
}

.topbar-icon-btn:hover,
.topbar-icon-btn:focus-visible {
  color: var(--el-color-primary);
  background-color: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
}

.topbar-icon-btn:hover .topbar-icon,
.topbar-icon-btn:focus-visible .topbar-icon {
  transform: scale(1.08);
}

.topbar-icon-btn:active {
  transform: scale(0.96);
}

.menu-search-trigger {
  display: inline-flex !important;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 36px;
  padding: 0 10px !important;
  margin-inline-end: 4px;
  flex-shrink: 0;
}

.menu-search-kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  padding: 1px 5px;
  border: 1px solid var(--el-border-color, #dcdfe6);
  border-radius: 6px;
  background: var(--el-fill-color-light, #f5f7fa);
  color: var(--el-text-color-secondary, #909399);
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  line-height: 1.4;
  white-space: nowrap;
}

@media (max-width: 1280px) {
  .menu-search-kbd {
    display: none;
  }

  .menu-search-trigger {
    padding: 8px !important;
    margin-inline-end: 0;
  }
}

.menu-search-content {
  padding: 8px;
}

.header-icon-fixed {
  display: flex;
  align-items: center;
  justify-content: center;
}

.menu-search-results {
  margin-top: 12px;
  max-height: 300px;
  overflow-y: auto;
}

.menu-search-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  margin-bottom: 4px;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
  gap: 8px;
}

.menu-search-item:hover,
.menu-search-item.is-active {
  background-color: #f5f7fa;
}

.menu-item-icon {
  flex-shrink: 0;
  font-size: 16px;
  color: #606266;
}

.menu-item-title {
  flex: 1;
  font-size: 14px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-item-path {
  flex-shrink: 0;
  font-size: 12px;
  color: #909399;
  margin-left: 8px;
}

.menu-search-empty {
  margin-top: 12px;
  padding: 20px;
  text-align: center;
  color: #909399;
  font-size: 14px;
}

.menu-search-footer {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter, #ebeef5);
  font-size: 12px;
  color: var(--el-text-color-secondary, #909399);
}
</style>

<style>
.menu-search-popover {
  padding: 0 !important;
}

.menu-search-popover .el-popover__content {
  padding: 0 !important;
}
</style>
