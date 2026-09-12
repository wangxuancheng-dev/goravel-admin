# 权限按钮显示配置说明

## 功能说明

此配置用于控制操作按钮（如添加、编辑、删除）在没有权限时是否显示在页面上。

## 配置项

在 `.env` 文件中添加以下配置：

```env
# 是否显示无权限的操作按钮
# false: 不显示（默认）- 用户没有权限时，按钮完全隐藏
# true: 显示但禁用 - 用户没有权限时，按钮显示但处于禁用状态
ADMIN_SHOW_BUTTONS_WITHOUT_PERMISSION=false
```

## 配置说明

- **默认值**: `false`（不显示）
- **类型**: 布尔值（`true` 或 `false`）
- **作用范围**: 全局配置，影响所有页面的操作按钮

### 配置效果

#### `ADMIN_SHOW_BUTTONS_WITHOUT_PERMISSION=false`（默认）
- 用户没有权限时，按钮**完全隐藏**
- 页面更加简洁，用户看不到无法使用的功能

#### `ADMIN_SHOW_BUTTONS_WITHOUT_PERMISSION=true`
- 用户没有权限时，按钮**显示但禁用**
- 用户可以知道有哪些功能，但无法使用
- 适合需要让用户了解系统完整功能的场景

## 使用方法

### 在 Vue 组件中使用

#### 方法一：使用 `getButtonState`（推荐，一个方法搞定）

**方式一：同时控制显示和禁用（根据配置决定是否隐藏）**

```vue
<template>
  <div>
    <!-- 根据配置和权限决定是否显示按钮 -->
    <el-button 
      v-if="getButtonState('admin.store').show" 
      :disabled="getButtonState('admin.store').disabled"
      @click="handleAdd"
    >
      添加
    </el-button>
    
    <el-button 
      v-if="getButtonState('admin.update').show" 
      :disabled="getButtonState('admin.update').disabled"
      @click="handleEdit(row)"
    >
      编辑
    </el-button>
    
    <el-button 
      v-if="getButtonState('admin.destroy').show" 
      :disabled="getButtonState('admin.destroy').disabled"
      @click="handleDelete(row)"
    >
      删除
    </el-button>
  </div>
</template>

<script setup>
import { usePermission } from '@/composables/usePermission'

const { getButtonState } = usePermission()
</script>
```

**方式二：只控制禁用（按钮始终显示，无权限时禁用）**

```vue
<template>
  <div>
    <!-- 按钮始终显示，无权限时禁用 -->
    <el-button 
      :disabled="getButtonState('admin.store').disabled"
      @click="handleAdd"
    >
      添加
    </el-button>
    
    <el-button 
      :disabled="getButtonState('admin.update').disabled"
      @click="handleEdit(row)"
    >
      编辑
    </el-button>
    
    <el-button 
      :disabled="getButtonState('admin.destroy').disabled"
      @click="handleDelete(row)"
    >
      删除
    </el-button>
  </div>
</template>

<script setup>
import { usePermission } from '@/composables/usePermission'

const { getButtonState } = usePermission()
</script>
```

#### 方法二：使用两个方法（兼容旧代码）

```vue
<template>
  <div>
    <el-button 
      v-if="shouldShowButton('admin.store')" 
      :disabled="isButtonDisabled('admin.store')"
      @click="handleAdd"
    >
      添加
    </el-button>
  </div>
</template>

<script setup>
import { usePermission } from '@/composables/usePermission'

const { shouldShowButton, isButtonDisabled } = usePermission()
</script>
```

#### 方法三：直接使用 store

```vue
<template>
  <div>
    <el-button 
      v-if="userStore.shouldShowButton('admin.store')" 
      :disabled="!userStore.hasPermission('admin.store')"
      @click="handleAdd"
    >
      添加
    </el-button>
  </div>
</template>

<script setup>
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
</script>
```

## 权限标识说明

权限标识格式通常为：`模块.操作`

常见示例：
- `admin.store` - 添加管理员
- `admin.update` - 编辑管理员
- `admin.destroy` - 删除管理员
- `role.store` - 添加角色
- `role.update` - 编辑角色
- `permission.index` - 查看权限列表

## 注意事项

1. 配置修改后需要重启后端服务才能生效
2. 前端会在用户登录时自动获取配置，无需刷新页面
3. 如果用户有权限，按钮总是会显示（不受此配置影响）
4. 此配置只影响按钮的显示/隐藏，不影响后端权限验证

## 相关文件

- 后端配置：`config/admin.go`
- 后端 API：`app/http/controllers/admin/auth_controller.go`
- 前端 Store：`html/src/store/user.js`
- 前端工具：`html/src/composables/usePermission.js`

