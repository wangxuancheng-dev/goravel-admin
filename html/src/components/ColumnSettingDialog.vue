<template>
  <el-popover
    v-model:visible="visible"
    placement="bottom-end"
    :width="popoverWidth"
    trigger="click"
    popper-class="vxe-table-custom-popover"
    :teleported="true"
    @hide="handleClose"
  >
    <template #reference>
      <slot name="reference">
        <el-button 
          :icon="SettingIcon"
          circle
          :title="$t('common.column_setting')"
        />
      </slot>
    </template>
    
    <div class="vxe-table-custom--panel">
      <div class="vxe-table-custom--panel-header">
        <el-checkbox
          v-model="selectAll"
          :indeterminate="isIndeterminate"
          @change="handleSelectAll"
        >
          {{ $t('vxe.toolbar.customAll') || $t('common.select_all') }}
        </el-checkbox>
      </div>
      
      <div class="vxe-table-custom--panel-body">
        <ul class="vxe-table-custom--panel-list" ref="listRef">
          <li
            v-for="(column, index) in sortedColumns"
            :key="column.key"
            class="vxe-table-custom--option level--1"
            :class="{
              'dragging': dragIndex === index,
              'drag-over': dragOverIndex === index
            }"
            :data-index="index"
            :data-key="column.key"
            draggable="true"
            @dragstart="handleDragStart($event, index)"
            @dragover="handleDragOver($event, index)"
            @drop="handleDrop($event, index)"
            @dragend="handleDragEnd($event)"
          >
            <div class="vxe-table-custom--checkbox-option" :class="{ 'is--checked': isColumnVisible(column.key) }">
              <el-checkbox
                :model-value="isColumnVisible(column.key)"
                :disabled="column.required"
                @change="(val) => handleColumnToggle(column.key, val)"
              />
            </div>
            
            <div class="vxe-table-custom--name-option">
              <div class="vxe-table-custom--sort-option">
                <span class="vxe-table-custom--sort-btn" :title="$t('vxe.custom.setting.sortHelpTip') || $t('common.drag_to_sort')">
                  <i class="vxe-table-icon-drag-handle"></i>
                </span>
              </div>
              <div class="vxe-table-custom--checkbox-label">{{ column.title }}</div>
            </div>
            
            <div class="vxe-table-custom--fixed-option">
              <el-button
                v-if="localFixedColumns[column.key] === 'left'"
                class="vxe-button type--text theme--primary"
                :title="$t('vxe.toolbar.cancelFixed') || $t('common.unfreeze')"
                @click="handleFixedChange(column.key, null)"
              >
                <i class="vxe-button--item vxe-button--prefix-icon vxe-table-icon-fixed-left-fill"></i>
              </el-button>
              <el-button
                v-else-if="localFixedColumns[column.key] === 'right'"
                class="vxe-button type--text theme--primary"
                :title="$t('vxe.toolbar.cancelFixed') || $t('common.unfreeze')"
                @click="handleFixedChange(column.key, null)"
              >
                <i class="vxe-button--item vxe-button--prefix-icon vxe-table-icon-fixed-right-fill"></i>
              </el-button>
              <template v-else>
                <el-button
                  class="vxe-button type--text"
                  :title="$t('vxe.toolbar.fixedLeft') || $t('common.freeze_left')"
                  @click="handleFixedChange(column.key, 'left')"
                >
                  <i class="vxe-button--item vxe-button--prefix-icon vxe-table-icon-fixed-left"></i>
                </el-button>
                <el-button
                  class="vxe-button type--text"
                  :title="$t('vxe.toolbar.fixedRight') || $t('common.freeze_right')"
                  @click="handleFixedChange(column.key, 'right')"
                >
                  <i class="vxe-button--item vxe-button--prefix-icon vxe-table-icon-fixed-right"></i>
                </el-button>
              </template>
            </div>
          </li>
        </ul>
      </div>
      
      <div class="vxe-table-custom--panel-footer">
        <el-button size="small" @click="handleReset">{{ $t('vxe.table.customReset') || $t('common.reset') }}</el-button>
        <el-button size="small" @click="handleClose">{{ $t('vxe.table.customCancel') || $t('common.cancel') }}</el-button>
        <el-button size="small" type="primary" @click="handleConfirm">{{ $t('vxe.table.customConfirm') || $t('common.confirm') }}</el-button>
      </div>
    </div>
  </el-popover>
</template>

<script setup>
import { ref, watch, computed, markRaw, onBeforeUnmount, nextTick } from 'vue'
import { Setting } from '@element-plus/icons-vue'

const SettingIcon = markRaw(Setting)

const listRef = ref(null)
const dragIndex = ref(-1)
const dragOverIndex = ref(-1)

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  visibleColumns: {
    type: Array,
    required: true
  },
  allColumns: {
    type: Array,
    required: true
  },
  defaultVisibleColumns: {
    type: Array,
    default: () => []
  },
  columnOrder: {
    type: Array,
    default: () => []
  },
  fixedColumns: {
    type: Object,
    default: () => ({})
  },
  popoverWidth: {
    type: [String, Number],
    default: 360
  }
})

const emit = defineEmits(['update:modelValue', 'confirm'])

const visible = ref(props.modelValue)
const localVisibleColumns = ref(Array.isArray(props.visibleColumns) ? [...props.visibleColumns] : [])
const localColumnOrder = ref(Array.isArray(props.columnOrder) ? [...props.columnOrder] : [])
const localFixedColumns = ref(props.fixedColumns && typeof props.fixedColumns === 'object' ? { ...props.fixedColumns } : {})

// 初始化列顺序和冻结状态
const initLocalState = () => {
  const columns = props.allColumns || []
  
  // 初始化列顺序：优先使用传入的 columnOrder，否则使用默认顺序
  // 注意：不要覆盖已有的 localColumnOrder，因为它可能已经从 props 中设置了
  if (localColumnOrder.value.length === 0 && columns.length > 0) {
    // 如果 props 中有 columnOrder，使用它；否则使用默认顺序
    if (Array.isArray(props.columnOrder) && props.columnOrder.length > 0) {
      // 过滤掉 checkbox 和 operation，只保留可配置的列
      const filteredOrder = props.columnOrder.filter(
        key => key !== 'checkbox' && key !== 'operation'
      )
      
      // 构建完整的顺序：checkbox + 可配置列 + operation
      const newOrder = []
      if (props.columnOrder.includes('checkbox')) {
        newOrder.push('checkbox')
      }
      newOrder.push(...filteredOrder)
      if (props.columnOrder.includes('operation')) {
        newOrder.push('operation')
      }
      
      localColumnOrder.value = newOrder.length > 0 ? newOrder : columns.map(col => col.key).filter(Boolean)
    } else {
      // 默认顺序：checkbox + 可配置列 + operation
      const configurableKeys = columns
        .map(col => col.key)
        .filter(key => key && key !== 'checkbox' && key !== 'operation')
      
      const newOrder = []
      if (columns.some(col => col.key === 'checkbox')) {
        newOrder.push('checkbox')
      }
      newOrder.push(...configurableKeys)
      if (columns.some(col => col.key === 'operation')) {
        newOrder.push('operation')
      }
      
      localColumnOrder.value = newOrder
    }
  }
  
  // 确保列顺序包含所有列（如果有新列添加）
  // 只处理可配置的列，checkbox 和 operation 保持固定位置
  const configurableKeys = columns
    .map(col => col.key)
    .filter(key => key && key !== 'checkbox' && key !== 'operation')
  
  const currentConfigurableOrder = localColumnOrder.value.filter(
    key => key !== 'checkbox' && key !== 'operation'
  )
  
  const missingKeys = configurableKeys.filter(key => !currentConfigurableOrder.includes(key))
  if (missingKeys.length > 0) {
    // 在可配置列的最后添加新列
    const newOrder = []
    if (localColumnOrder.value.includes('checkbox')) {
      newOrder.push('checkbox')
    }
    newOrder.push(...currentConfigurableOrder, ...missingKeys)
    if (localColumnOrder.value.includes('operation')) {
      newOrder.push('operation')
    }
    localColumnOrder.value = newOrder
  }
  
  // 合并保存的冻结状态和列的默认冻结状态
  const fixedMap = { ...localFixedColumns.value }
  columns.forEach(col => {
    // 如果列有默认的 fixed 属性，且当前没有设置，则使用默认值
    if (col.fixed && !fixedMap[col.key]) {
      fixedMap[col.key] = col.fixed
    }
  })
  localFixedColumns.value = fixedMap
}

// 排序后的列（根据 localColumnOrder）
const sortedColumns = computed(() => {
  const columns = props.allColumns || []
  
  // 额外过滤：确保排除 checkbox 和 operation 列（双重保险）
  const filteredColumns = columns.filter(col => {
    const key = col.key || ''
    return key !== 'checkbox' && key !== 'operation'
  })
  
  // 如果没有列顺序，使用默认顺序
  if (localColumnOrder.value.length === 0) {
    return filteredColumns.map(col => ({
      ...col,
      fixed: localFixedColumns.value[col.key] || col.fixed
    }))
  }
  
  // 按保存的顺序排序
  const result = []
  const columnMap = {}
  filteredColumns.forEach(col => {
    columnMap[col.key] = col
  })
  
  // 按顺序添加列
  localColumnOrder.value.forEach(key => {
    // 跳过 checkbox 和 operation
    if (key === 'checkbox' || key === 'operation') return
    
    const col = columnMap[key]
    if (col) {
      result.push({
        ...col,
        fixed: localFixedColumns.value[key] || col.fixed
      })
    }
  })
  
  // 添加新列（不在顺序中的）
  filteredColumns.forEach(col => {
    if (!localColumnOrder.value.includes(col.key)) {
      result.push({
        ...col,
        fixed: localFixedColumns.value[col.key] || col.fixed
      })
    }
  })
  
  return result
})

// 检查列是否可见
const isColumnVisible = (key) => {
  return localVisibleColumns.value.includes(key)
}

// 全选状态
const selectAll = computed({
  get() {
    const visibleCols = sortedColumns.value.filter(col => !col.required)
    return visibleCols.length > 0 && visibleCols.every(col => isColumnVisible(col.key))
  },
  set(val) {
    const visibleCols = sortedColumns.value.filter(col => !col.required)
    if (val) {
      visibleCols.forEach(col => {
        if (!localVisibleColumns.value.includes(col.key)) {
          localVisibleColumns.value.push(col.key)
        }
      })
    } else {
      visibleCols.forEach(col => {
        const index = localVisibleColumns.value.indexOf(col.key)
        if (index > -1) {
          localVisibleColumns.value.splice(index, 1)
        }
      })
    }
  }
})

// 半选状态
const isIndeterminate = computed(() => {
  const visibleCols = sortedColumns.value.filter(col => !col.required)
  const checkedCount = visibleCols.filter(col => isColumnVisible(col.key)).length
  return checkedCount > 0 && checkedCount < visibleCols.length
})

// 切换列显示/隐藏
const handleColumnToggle = (key, val) => {
  if (val) {
    if (!localVisibleColumns.value.includes(key)) {
      localVisibleColumns.value.push(key)
    }
  } else {
    const index = localVisibleColumns.value.indexOf(key)
    if (index > -1) {
      localVisibleColumns.value.splice(index, 1)
    }
  }
}

// 全选/取消全选
const handleSelectAll = (val) => {
  selectAll.value = val
}

// 冻结列变化
const handleFixedChange = (key, fixed) => {
  if (fixed) {
    localFixedColumns.value[key] = fixed
  } else {
    delete localFixedColumns.value[key]
  }
}

// 拖拽开始
const handleDragStart = (e, index) => {
  dragIndex.value = index
  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('text/plain', index.toString())
  // 设置拖拽图像
  const dragImage = e.target.cloneNode(true)
  dragImage.style.opacity = '0.5'
  document.body.appendChild(dragImage)
  e.dataTransfer.setDragImage(dragImage, 0, 0)
  setTimeout(() => {
    document.body.removeChild(dragImage)
  }, 0)
}

// 拖拽经过
const handleDragOver = (e, index) => {
  e.preventDefault()
  e.stopPropagation()
  e.dataTransfer.dropEffect = 'move'
  
  if (dragIndex.value === index || dragIndex.value === -1) return
  
  dragOverIndex.value = index
}

// 放置
const handleDrop = (e, dropIndex) => {
  e.preventDefault()
  e.stopPropagation()
  
  if (dragIndex.value === -1) return
  
  const fromIndex = dragIndex.value
  const toIndex = parseInt(e.currentTarget.dataset.index)
  
  if (fromIndex !== toIndex && fromIndex >= 0 && toIndex >= 0) {
    // 获取当前排序后的列（已过滤掉 checkbox 和 operation）
    const currentSortedColumns = sortedColumns.value
    
    // 确保索引有效
    if (fromIndex < currentSortedColumns.length && toIndex < currentSortedColumns.length) {
      // 获取要移动的列的 key
      const fromKey = currentSortedColumns[fromIndex].key
      const toKey = currentSortedColumns[toIndex].key
      
      if (!fromKey || !toKey) return
      
      // 从 localColumnOrder 中过滤掉 checkbox 和 operation，只保留可配置的列
      const configurableOrder = localColumnOrder.value.filter(
        key => key !== 'checkbox' && key !== 'operation'
      )
      
      // 找到 fromKey 和 toKey 在 configurableOrder 中的位置
      const fromOrderIndex = configurableOrder.indexOf(fromKey)
      const toOrderIndex = configurableOrder.indexOf(toKey)
      
      if (fromOrderIndex !== -1 && toOrderIndex !== -1) {
        // 重新排序
        const [removed] = configurableOrder.splice(fromOrderIndex, 1)
        configurableOrder.splice(toOrderIndex, 0, removed)
        
        // 更新 localColumnOrder（保持 checkbox 和 operation 的位置）
        const newOrder = []
        
        // 如果原始列中有 checkbox，保持在最前
        const originalColumns = props.allColumns || []
        if (originalColumns.some(col => col.key === 'checkbox')) {
          newOrder.push('checkbox')
        }
        
        // 添加重新排序后的可配置列
        newOrder.push(...configurableOrder)
        
        // 如果原始列中有 operation，保持在最后
        if (originalColumns.some(col => col.key === 'operation')) {
          newOrder.push('operation')
        }
        
        // 创建新数组引用以触发响应式更新
        localColumnOrder.value = newOrder
      }
    }
  }
  
  dragIndex.value = -1
  dragOverIndex.value = -1
}

// 拖拽结束
const handleDragEnd = (e) => {
  dragIndex.value = -1
  dragOverIndex.value = -1
}

// 监听外部 visible 变化
watch(() => props.modelValue, (newVal) => {
  visible.value = newVal
  if (newVal) {
    // 打开对话框时，重置为当前的 visibleColumns、columnOrder 和 fixedColumns
    localVisibleColumns.value = Array.isArray(props.visibleColumns) ? [...props.visibleColumns] : []
    localColumnOrder.value = Array.isArray(props.columnOrder) ? [...props.columnOrder] : []
    localFixedColumns.value = props.fixedColumns && typeof props.fixedColumns === 'object' ? { ...props.fixedColumns } : {}
    initLocalState()
  }
})

// 监听内部 visible 变化
watch(visible, (newVal) => {
  emit('update:modelValue', newVal)
})

// 监听 visibleColumns 变化
watch(() => props.visibleColumns, (newVal) => {
  if (!visible.value) {
    localVisibleColumns.value = Array.isArray(newVal) ? [...newVal] : []
  }
}, { deep: true })

// 监听 allColumns、columnOrder、fixedColumns 变化
watch(() => [props.allColumns, props.columnOrder, props.fixedColumns], () => {
  if (visible.value) {
    localColumnOrder.value = Array.isArray(props.columnOrder) ? [...props.columnOrder] : []
    localFixedColumns.value = props.fixedColumns && typeof props.fixedColumns === 'object' ? { ...props.fixedColumns } : {}
    initLocalState()
  }
}, { deep: true })

// 重置
const handleReset = () => {
  localVisibleColumns.value = Array.isArray(props.defaultVisibleColumns) ? [...props.defaultVisibleColumns] : []
  // 重置列顺序为默认顺序
  const columns = props.allColumns || []
  localColumnOrder.value = columns.map(col => col.key).filter(Boolean)
  // 重置冻结状态
  localFixedColumns.value = {}
  initLocalState()
}

const handleClose = () => {
  visible.value = false
  // 关闭时恢复为原始值
  localVisibleColumns.value = Array.isArray(props.visibleColumns) ? [...props.visibleColumns] : []
  localColumnOrder.value = Array.isArray(props.columnOrder) ? [...props.columnOrder] : []
  localFixedColumns.value = props.fixedColumns && typeof props.fixedColumns === 'object' ? { ...props.fixedColumns } : {}
  initLocalState()
}

const handleConfirm = () => {
  const confirmData = {
    visibleColumns: Array.isArray(localVisibleColumns.value) ? [...localVisibleColumns.value] : [],
    fixedColumns: { ...localFixedColumns.value },
    columnOrder: [...localColumnOrder.value]
  }
  
  emit('confirm', confirmData)
  visible.value = false
}

// 初始化
initLocalState()

// 组件卸载前确保 popover 关闭
onBeforeUnmount(() => {
  visible.value = false
})
</script>

<style scoped>
/* vxe-table 列设置面板样式 */
.vxe-table-custom--panel {
  min-width: 320px;
  max-width: 400px;
  width: 100%;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow: hidden;
  border-radius: 12px;
}

.vxe-table-custom--panel-header {
  padding: 10px 12px;
  border-bottom: 1px solid color-mix(in srgb, var(--border-color-light) 72%, transparent);
  margin-bottom: 8px;
}

.vxe-table-custom--panel-header :deep(.el-checkbox) {
  margin: 0;
}

.vxe-table-custom--panel-body {
  flex: 1;
  max-height: 400px;
  overflow-y: auto;
  margin-bottom: 8px;
  min-height: 0;
}

.vxe-table-custom--panel-list {
  list-style: none;
  margin: 0;
  padding: 4px 0;
}

.vxe-table-custom--option {
  display: flex;
  align-items: center;
  padding: 9px 12px;
  margin-bottom: 2px;
  border-radius: 8px;
  transition: background-color 0.2s, opacity 0.2s, border-color 0.2s;
  cursor: move;
  width: 100%;
  box-sizing: border-box;
  position: relative;
  border: 1px solid transparent;
}

.vxe-table-custom--option:hover {
  background-color: color-mix(in srgb, var(--el-color-primary) 8%, transparent);
  border-color: color-mix(in srgb, var(--el-color-primary) 26%, transparent);
}

.vxe-table-custom--option.dragging {
  opacity: 0.5;
  background-color: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
}

.vxe-table-custom--option.drag-over {
  border-top: 2px solid var(--el-color-primary);
  background-color: color-mix(in srgb, var(--el-color-primary) 15%, transparent);
}

.vxe-table-custom--option[draggable="true"] {
  user-select: none;
  -webkit-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
}

.vxe-table-custom--checkbox-option {
  display: flex;
  align-items: center;
  margin-right: 12px;
  flex-shrink: 0;
  width: auto;
}

.vxe-table-custom--checkbox-option :deep(.el-checkbox) {
  margin: 0;
}

/* 避免 checkbox 容器 hover 背景覆盖整行 hover 背景 */
.vxe-table-custom--panel-list .vxe-table-custom--checkbox-option:hover {
  background-color: transparent !important;
}

.vxe-table-custom--checkbox-option.is--checked :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background-color: var(--el-color-primary);
  border-color: var(--el-color-primary);
}

.vxe-table-custom--name-option {
  flex: 1;
  display: flex;
  align-items: center;
  min-width: 0;
  overflow: hidden;
}

.vxe-table-custom--sort-option {
  margin-right: 8px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.vxe-table-custom--sort-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: move;
  color: #909399;
  width: 18px;
  height: 18px;
}

.vxe-table-custom--sort-btn:hover {
  color: var(--el-color-primary);
}

.vxe-table-icon-drag-handle {
  display: inline-block;
  width: 16px;
  height: 16px;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 16 16'%3E%3Cpath fill='%23909399' d='M8 1l1.8 1.8h-1.2v3.4h3.4V5L13.8 6.8 12 8.6V7.4H8.6v3.4h1.2L8 12.6l-1.8-1.8h1.2V7.4H4V8.6L2.2 6.8 4 5v1.2h3.4V2.8H6.2z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: center;
  cursor: move;
}

.vxe-table-custom--checkbox-label {
  flex: 1;
  font-size: 13px;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.5;
}

.vxe-table-custom--fixed-option {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-left: 8px;
  flex-shrink: 0;
}

.vxe-button {
  padding: 4px 6px;
  min-width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.vxe-button.type--text {
  color: #909399;
}

.vxe-button.type--text:hover {
  color: var(--el-color-primary);
  background: #ecf5ff;
}

.vxe-button.theme--primary {
  color: var(--el-color-primary);
}

.vxe-button--prefix-icon {
  font-size: 14px;
  line-height: 1;
}

.vxe-table-icon-fixed-left,
.vxe-table-icon-fixed-right,
.vxe-table-icon-fixed-left-fill,
.vxe-table-icon-fixed-right-fill {
  display: inline-block;
  width: 14px;
  height: 14px;
  background-repeat: no-repeat;
  background-position: center;
}

.vxe-table-icon-fixed-left {
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 14 14'%3E%3Cpath fill='%23909399' d='M2 2h2v10H2V2zm3 4.2h4.2V4.8L12 7 9.2 9.2V7.8H5V6.2z'/%3E%3C/svg%3E");
}

.vxe-table-icon-fixed-right {
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 14 14'%3E%3Cpath fill='%23909399' d='M10 2h2v10h-2V2zM9 7.8H4.8v1.4L2 7l2.8-2.2v1.4H9v1.6z'/%3E%3C/svg%3E");
}

.vxe-table-icon-fixed-left-fill {
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 14 14'%3E%3Cpath fill='%23409eff' d='M2 2h2v10H2V2zm3 4.2h4.2V4.8L12 7 9.2 9.2V7.8H5V6.2z'/%3E%3C/svg%3E");
}

.vxe-table-icon-fixed-right-fill {
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 14 14'%3E%3Cpath fill='%23409eff' d='M10 2h2v10h-2V2zM9 7.8H4.8v1.4L2 7l2.8-2.2v1.4H9v1.6z'/%3E%3C/svg%3E");
}

.vxe-table-custom--panel-footer {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  padding: 10px 4px 2px 4px;
  margin-top: 8px;
  border-top: 1px solid color-mix(in srgb, var(--border-color-light) 72%, transparent);
  flex-shrink: 0;
  width: 100%;
  box-sizing: border-box;
  overflow: hidden;
}

.vxe-table-custom--panel-footer .el-button {
  flex-shrink: 0;
  white-space: nowrap;
  min-width: auto;
  padding: 5px 15px;
}

/* 滚动条样式 */
.vxe-table-custom--panel-body::-webkit-scrollbar {
  width: 6px;
}

.vxe-table-custom--panel-body::-webkit-scrollbar-track {
  background: var(--el-fill-color-lighter);
  border-radius: 3px;
}

.vxe-table-custom--panel-body::-webkit-scrollbar-thumb {
  background: var(--el-border-color-darker);
  border-radius: 3px;
}

.vxe-table-custom--panel-body::-webkit-scrollbar-thumb:hover {
  background: var(--el-text-color-placeholder);
}
</style>

<style>
/* 全局样式，用于覆盖 el-popover */
.vxe-table-custom-popover {
  padding: 8px !important;
  overflow: visible !important;
  border-radius: 14px !important;
  border: 1px solid color-mix(in srgb, var(--border-color-light) 70%, transparent) !important;
  background: color-mix(in srgb, var(--card-bg, #fff) 96%, transparent) !important;
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.14), 0 2px 8px rgba(15, 23, 42, 0.08) !important;
  backdrop-filter: blur(8px);
}

.vxe-table-custom-popover .el-popover__content {
  overflow: visible !important;
}

html.dark .vxe-table-custom-popover {
  border-color: rgba(255, 255, 255, 0.12) !important;
  background: color-mix(in srgb, var(--card-bg, #1d1e1f) 92%, transparent) !important;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.5), 0 2px 8px rgba(0, 0, 0, 0.35) !important;
}
</style>

