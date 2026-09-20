<!--
  通用表格操作按钮组件
  
  使用示例：
  
  1. 简单用法（只有主要操作）：
  <TableActionButtons
    :row="row"
    :primary-actions="[
      {
        key: 'edit',
        label: $t('common.edit'),
        type: 'primary',
        permission: 'admin.update',
        handler: handleEdit
      },
      {
        key: 'delete',
        label: $t('common.delete'),
        type: 'danger',
        permission: 'admin.destroy',
        handler: handleDelete
      }
    ]"
    :get-button-state="getButtonState"
  />
  
  2. 带下拉菜单的用法：
  <TableActionButtons
    :row="row"
    :primary-actions="getPrimaryActions(row)"
    :more-actions="getMoreActions(row)"
    :get-button-state="getButtonState"
    @action="handleAction"
  />
  
  操作配置说明：
  - key: 操作的唯一标识
  - label: 按钮显示的文本
  - type: 按钮类型（primary/danger/warning/info/success）
  - permission: 权限标识，用于权限检查
  - handler: 点击处理函数，接收 row 作为参数
  - show: 是否显示该操作（函数或布尔值）
  - disabled: 是否禁用该操作（函数或布尔值）
  - command: 下拉菜单命令（用于 moreActions）
  - divided: 是否显示分割线（仅用于下拉菜单）
-->
<template>
  <div class="table-action-buttons">
    <!-- 主要操作按钮 -->
    <template v-for="(action, index) in primaryActions" :key="index">
      <el-button
        v-if="shouldShowAction(action, row)"
        :type="action.type || 'primary'"
        link
        :disabled="isActionDisabled(action, row)"
        @click="handleAction(action, row)"
        :title="action.title || (action.iconOnly ? action.label : '')"
      >
        <el-icon v-if="action.icon" :class="{ 'el-icon--left': !!action.label && !action.iconOnly }">
          <component :is="action.icon" />
        </el-icon>
        <template v-if="!action.iconOnly">{{ action.label }}</template>
      </el-button>
    </template>
    
    <!-- 更多操作：使用下拉菜单 -->
    <el-dropdown
      v-if="hasMoreActions(row)"
      trigger="click"
      :teleported="false"
      @command="(command) => handleDropdownCommand(command, row)"
    >
      <el-button
        type="info"
        link
        :disabled="isMoreActionsDisabled(row)"
      >
        {{ t('common.more') }}
        <el-icon class="el-icon--right"><ArrowDownIcon /></el-icon>
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <template v-for="(action, index) in moreActions" :key="index">
            <el-dropdown-item
              v-if="shouldShowAction(action, row)"
              :command="action.command || action.key"
              :disabled="isActionDisabled(action, row)"
              :divided="action.divided"
              :icon="action.icon"
            >
              {{ action.label }}
            </el-dropdown-item>
          </template>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowDown } from '@element-plus/icons-vue'
import { markRaw } from 'vue'

const { t } = useI18n()
const ArrowDownIcon = markRaw(ArrowDown)

const props = defineProps({
  // 行数据
  row: {
    type: Object,
    required: true
  },
  // 主要操作按钮配置
  primaryActions: {
    type: Array,
    default: () => []
  },
  // 更多操作按钮配置（下拉菜单）
  moreActions: {
    type: Array,
    default: () => []
  },
  // 权限检查函数
  getButtonState: {
    type: Function,
    default: () => ({ disabled: false })
  }
})

const emit = defineEmits(['action'])

// 判断是否显示操作
const shouldShowAction = (action, row) => {
  if (action.show === false) return false
  if (typeof action.show === 'function' && !action.show(row)) {
    return false
  }
  if (action.permission && props.getButtonState) {
    if (!props.getButtonState(action.permission).show) {
      return false
    }
  }
  return true
}

// 判断操作是否禁用
const isActionDisabled = (action, row) => {
  if (action.disabled === true) return true
  if (typeof action.disabled === 'function') {
    return action.disabled(row)
  }
  if (action.permission && props.getButtonState) {
    return props.getButtonState(action.permission).disabled
  }
  return false
}

// 处理操作点击
const handleAction = (action, row) => {
  if (action.handler) {
    action.handler(row)
  } else {
    emit('action', action.command || action.key, row)
  }
}

// 处理下拉菜单命令
const handleDropdownCommand = (command, row) => {
  const action = props.moreActions.find(a => (a.command || a.key) === command)
  if (action) {
    handleAction(action, row)
  } else {
    emit('action', command, row)
  }
}

// 判断是否有更多操作
const hasMoreActions = (row) => {
  return props.moreActions.some(action => shouldShowAction(action, row))
}

// 判断更多操作是否全部禁用
const isMoreActionsDisabled = (row) => {
  const visibleActions = props.moreActions.filter(action => shouldShowAction(action, row))
  if (visibleActions.length === 0) return true
  return visibleActions.every(action => isActionDisabled(action, row))
}
</script>

<style scoped>
.table-action-buttons {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: nowrap;
  white-space: nowrap;
}

.table-action-buttons .el-button {
  margin: 0;
  padding: 0;
  white-space: nowrap;
}

.table-action-buttons .el-dropdown {
  margin-left: 0;
}

.table-action-buttons .el-icon--right {
  margin-left: 2px;
}
</style>

