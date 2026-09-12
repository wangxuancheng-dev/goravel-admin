<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="900px"
    @close="handleDialogClose"
    @opened="handleDialogOpened"
  >
    <div v-loading="loading">
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="100px"
      >
        <el-form-item :label="$t('role.name')" prop="name">
          <el-input v-model="formData.name" :disabled="loading" />
        </el-form-item>
        <el-form-item :label="$t('role.slug')" prop="slug">
          <el-input 
            v-model="formData.slug" 
            :disabled="isProtectedRole(formData) || loading"
            :placeholder="isProtectedRole(formData) ? $t('role.protected_role_slug_disabled') : ''"
          />
        </el-form-item>
        <el-form-item :label="$t('common.description')">
          <el-input v-model="formData.description" type="textarea" :disabled="loading" />
        </el-form-item>
        <el-form-item :label="$t('role.data_scope')" prop="data_scope">
          <el-select
            v-model="formData.data_scope"
            :disabled="isProtectedRole(formData) || loading"
            style="width: 100%"
          >
            <el-option
              v-for="opt in dataScopeOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item
          v-if="formData.data_scope === 2 && !isProtectedRole(formData)"
          :label="$t('role.data_scope_depts')"
          prop="department_ids"
        >
          <el-tree-select
            v-model="formData.department_ids"
            :data="departmentTree"
            :props="{ label: 'label', value: 'value', children: 'children' }"
            multiple
            check-strictly
            filterable
            clearable
            :render-after-expand="false"
            :disabled="loading"
            style="width: 100%"
            :placeholder="$t('role.data_scope_depts_placeholder')"
          />
        </el-form-item>
        <el-form-item 
          v-if="!isProtectedRole(formData)" 
          :label="$t('role.menus_and_permissions')"
        >
          <div class="menu-permission-container">
            <div class="tree-wrapper">
              <el-tree
                :key="treeKey"
                ref="menuPermissionTreeRef"
                :data="menuPermissionTree"
                :props="{ children: 'children', label: 'label' }"
                show-checkbox
                node-key="id"
                :checked-keys="checkedKeys"
                class="menu-permission-tree"
                :expand-on-click-node="false"
                :default-expand-all="false"
                :disabled="loading"
                @check="handleTreeCheck"
              >
                <template #default="{ node, data }">
                  <span v-if="data.isMenu" class="menu-node">
                    <el-icon class="node-icon menu-icon"><FolderOpenedIcon /></el-icon>
                    <span class="menu-name">{{ data.name }}</span>
                    <el-tag v-if="data.type" size="small" :type="getMenuTypeTag(data.type)" class="menu-type-tag">
                      {{ getMenuTypeText(data.type) }}
                    </el-tag>
                  </span>
                  <span v-else class="permission-node">
                    <el-icon class="node-icon permission-icon"><KeyIcon /></el-icon>
                    <span class="permission-name">{{ data.displayDesc || data.name }}</span>
                    <span v-if="data.method" class="permission-method" :class="`method-${data.method.toLowerCase()}`">
                      {{ data.method }}
                    </span>
                    <el-tooltip v-if="data.path" :content="data.path" placement="top">
                      <el-icon class="permission-path-icon">
                        <InfoFilledIcon />
                      </el-icon>
                    </el-tooltip>
                  </span>
                </template>
              </el-tree>
            </div>
          </div>
        </el-form-item>
        <el-form-item v-if="isProtectedRole(formData)" :label="$t('role.menus_and_permissions')">
          <div class="protected-role-tip">
            <el-icon><LockIcon /></el-icon>
            <span>{{ $t('role.super_admin_has_all_permissions') }}</span>
          </div>
        </el-form-item>
        <el-form-item :label="$t('table.status')" prop="status">
          <el-radio-group v-model.number="formData.status" :disabled="loading">
            <el-radio :label="1">{{ $t('common.enabled') }}</el-radio>
            <el-radio :label="0" :disabled="isProtectedRole(formData)">{{ $t('common.disabled') }}</el-radio>
          </el-radio-group>
          <div v-if="isProtectedRole(formData)" class="protected-tip">
            <el-icon><LockIcon /></el-icon>
            <span>{{ $t('role.protected_cannot_disable') }}</span>
          </div>
        </el-form-item>
        <el-form-item :label="$t('common.sort')">
          <el-input-number v-model="formData.sort" :min="0" :disabled="loading" />
        </el-form-item>
      </el-form>
    </div>
    <template #footer>
      <el-button @click="handleCancel">{{ $t('common.cancel') }}</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $t('common.confirm') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, watch, nextTick, onMounted, markRaw } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { InfoFilled, FolderOpened, Key, Lock } from '@element-plus/icons-vue'
import { getRoleDetail, createRole, updateRole } from '../../api/role'
import { getPermissionList } from '../../api/permission'
import { getMenuTree } from '../../api/menu'
import { getOptions } from '../../api/option'
import { getMenuTranslation } from '../../utils/menuTranslation'
import { groupBy, map } from 'lodash-es'
import { mapTree } from '../../utils/tree'
import { mapFields } from '../../utils/normalizeFormData'

// 使用 markRaw 标记图标组件，避免被 Vue 做成响应式对象
const InfoFilledIcon = markRaw(InfoFilled)
const FolderOpenedIcon = markRaw(FolderOpened)
const KeyIcon = markRaw(Key)
const LockIcon = markRaw(Lock)

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  editId: {
    type: [Number, String],
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'success'])

const { t, te, tm } = useI18n()
const formRef = ref(null)
const menuPermissionTreeRef = ref(null)

const loading = ref(false)
const submitting = ref(false)

// 定义表单初始值的复用函数（返回新对象，避免引用问题）
const getFormInitialValue = () => ({
  id: null,
  name: '',
  slug: '',
  description: '',
  permission_ids: [],
  menu_ids: [],
  department_ids: [],
  data_scope: 1,
  status: 1,
  sort: 0
})

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const dialogTitle = computed(() => formData.id ? t('role.edit_role') : t('role.add_role'))

const menuPermissionTree = ref([])
const checkedKeys = ref([])
const treeKey = ref(0)
const protectedRoleSlugs = ref(['super-admin'])
const departmentTree = ref([])

const dataScopeOptions = computed(() => [
  { value: 1, label: t('role.data_scope_all') },
  { value: 2, label: t('role.data_scope_custom') },
  { value: 3, label: t('role.data_scope_dept') },
  { value: 4, label: t('role.data_scope_dept_and_child') },
  { value: 5, label: t('role.data_scope_self') }
])

const normalizeDepartmentTree = (nodes) => {
  if (!Array.isArray(nodes)) return []
  return nodes.map((node) => {
    const value = Number(node.value ?? node.id ?? node.ID ?? 0)
    const children = node.children || node.Children || []
    return {
      label: node.label || node.name || node.Name || String(value),
      value,
      children: children.length ? normalizeDepartmentTree(children) : undefined
    }
  }).filter((n) => n.value > 0)
}

const loadDepartmentTree = async () => {
  try {
    const res = await getOptions('department')
    const payload = res?.data ?? res
    const options = payload?.options || payload?.list || payload || []
    departmentTree.value = normalizeDepartmentTree(Array.isArray(options) ? options : [])
  } catch (e) {
    departmentTree.value = []
  }
}

const formData = reactive(getFormInitialValue())

const formRules = computed(() => ({
  name: [{ required: true, message: t('role.name_required'), trigger: 'blur' }],
  slug: [{ required: true, message: t('role.slug_required'), trigger: 'blur' }]
}))

// 获取菜单标题（优先使用 slug，如果没有则使用 path 和 title 映射，最后使用原始标题）
const getMenuTitle = (menu) => {
  if (!menu || typeof menu !== 'object') {
    return ''
  }
  
  // 优先使用 slug 作为翻译键标识
  const slug = menu.Slug || menu.slug || ''
  if (slug) {
    const translated = getMenuTranslation(t, te, slug)
    if (translated) {
      return translated
    }
  }
  
  // 回退到原始标题
  return menu.Title || menu.title || ''
}

// 获取权限名称（优先使用 slug，如果没有则使用 description 或 name）
const getPermissionName = (permission) => {
  if (!permission || typeof permission !== 'object') {
    return ''
  }
  
  // 优先使用 slug 作为翻译键标识
  const slug = permission.Slug || permission.slug || ''
  if (slug) {
    const slugKey = `permission.${slug}`

    if (typeof te === 'function' && te(slugKey)) {
      return t(slugKey)
    }

    const messages = typeof tm === 'function' ? tm('permission') : null
    if (messages && Object.prototype.hasOwnProperty.call(messages, slug)) {
      const value = messages[slug]
      if (typeof value === 'string') {
        return value
      }
    }
  }
  
  // 回退到 description 或 name
  return permission.Description || permission.description || permission.Name || permission.name || ''
}

const getModuleNameFromPath = (path) => {
  if (!path) return t('role.other_module')
  
  let cleanPath = path.split('?')[0].replace(/\*/g, '').replace(/\/$/, '')
  cleanPath = cleanPath.replace(/\/\d+(\/|$)/g, '/')
  cleanPath = cleanPath.replace(/\/$/, '')
  
  const parts = cleanPath.split('/').filter(p => p)
  if (parts.length >= 3) {
    const module = parts[parts.length - 1]
    const singular = module.replace(/s$/, '').replace(/-/g, '_')
    
    // 优先使用菜单翻译（统一使用 menu.*）
    const menuKey = `menu.${singular}`
    if (typeof te === 'function' && te(menuKey)) {
      return t(menuKey)
    }
    
    // 回退到 role.module_*（用于其他模块）
    const translationKey = `role.module_${singular}`
    const translated = t(translationKey)
    if (translated !== translationKey) {
      return translated
    }
    return module.charAt(0).toUpperCase() + module.slice(1).replace(/-/g, ' ')
  }
  
  return t('role.other_module')
}

const transformPermissionToTree = (permissions) => {
  if (!permissions || !Array.isArray(permissions)) return []
  
  // 使用 lodash-es 的 groupBy 按模块分组
  const groupedPermissions = groupBy(permissions, perm => {
    const path = perm.Path || perm.path || '/'
    return getModuleNameFromPath(path)
  })
  
  // 转换为树形结构
  const tree = map(groupedPermissions, (perms, moduleName) => {
    const children = map(perms, perm => {
      const path = perm.Path || perm.path || '/'
      const method = perm.Method || perm.method || ''
      const name = perm.Name || perm.name || ''
      const slug = perm.Slug || perm.slug || ''
      const description = perm.Description || perm.description || ''
      const id = perm.id || perm.ID
      const displayLabel = description || name
      
      return {
        id: id,
        name: name,
        slug: slug,
        method: method,
        path: path,
        description: description,
        label: displayLabel,
        displayName: name,
        displayDesc: description || name
      }
    })
    
    // 对子节点按方法排序
    children.sort((a, b) => {
      const methodOrder = { 'GET': 1, 'POST': 2, 'PUT': 3, 'PATCH': 4, 'DELETE': 5 }
      return (methodOrder[a.method] || 99) - (methodOrder[b.method] || 99)
    })
    
    return {
      id: `module_${moduleName}`,
      name: moduleName,
      label: moduleName,
      children: children
    }
  })
  
  // 按模块名排序
  tree.sort((a, b) => a.name.localeCompare(b.name))
  
  return tree
}

const transformMenuToTree = (menus) => {
  if (!menus || !Array.isArray(menus)) return []
  
  // 使用工具函数递归转换菜单树
  return mapTree(menus, (node) => {
    const type = node.Type !== undefined ? node.Type : (node.type !== undefined ? node.type : 1)
    const icon = node.Icon || node.icon || ''
    const path = node.Path || node.path || ''
    const slug = node.Slug || node.slug || ''
    
    // 使用多语言函数获取菜单标题
    const title = getMenuTitle(node)
    
    return {
      id: `menu_${node.id}`, // 添加前缀避免ID冲突
      rawId: node.id,
      name: title,
      label: title,
      slug: slug,
      type: type,
      icon: icon,
      path: path,
      component: node.Component || node.component || '',
      permission: node.Permission || node.permission || '',
      isMenu: true
    }
  }, 'children')
}

const attachPermissionsToMenus = (menuTree, permissions) => {
  if (!permissions || !Array.isArray(permissions)) return menuTree
  
  // 使用 lodash-es 的 groupBy 按菜单ID分组权限
  const permissionMap = groupBy(permissions, perm => {
    return perm.MenuID || perm.menu_id || 0
  })
  
  // 使用工具函数递归处理菜单树
  return mapTree(menuTree, (node) => {
    const result = { ...node }
    
    if (result.isMenu && result.rawId) {
      const menuId = result.rawId
      const matchedPermissions = permissionMap[menuId] || []
      
      if (matchedPermissions.length > 0) {
        if (!result.children) {
          result.children = []
        }
        
        matchedPermissions.forEach(perm => {
          const method = perm.Method || perm.method || ''
          const id = perm.id || perm.ID
          const slug = perm.Slug || perm.slug || ''
          
          // 使用多语言函数获取权限名称
          const permissionName = getPermissionName(perm)
          
          result.children.push({
            id: `perm_${id}`, // 添加前缀避免ID冲突
            rawId: id,
            name: permissionName,
            slug: slug,
            method: method,
            path: perm.Path || perm.path || '',
            description: perm.Description || perm.description || '',
            label: permissionName,
            displayDesc: permissionName,
            isMenu: false,
            isPermission: true
          })
        })
        
        result.children.sort((a, b) => {
          if (a.isMenu !== b.isMenu) {
            return a.isMenu ? -1 : 1
          }
          if (!a.isMenu && !b.isMenu) {
            const methodOrder = { 'GET': 1, 'POST': 2, 'PUT': 3, 'PATCH': 4, 'DELETE': 5 }
            return (methodOrder[a.method] || 99) - (methodOrder[b.method] || 99)
          }
          return 0
        })
      }
    }
    
    return result
  }, 'children')
}

const buildMenuPermissionTree = (menus, permissions) => {
  const menuTreeData = transformMenuToTree(menus)
  const treeWithPermissions = attachPermissionsToMenus(menuTreeData, permissions)
  
  const matchedPermissionIds = new Set()
  const collectPermissionIds = (nodes) => {
    nodes.forEach(node => {
      if (node.isPermission) {
        matchedPermissionIds.add(node.rawId)
      }
      if (node.children) {
        collectPermissionIds(node.children)
      }
    })
  }
  collectPermissionIds(treeWithPermissions)
  
  const unmatchedPermissions = permissions.filter(perm => {
    const id = perm.id || perm.ID
    return !matchedPermissionIds.has(id)
  })
  
  if (unmatchedPermissions.length > 0) {
    const otherPermissionsNode = {
      id: 'other_permissions',
      name: t('role.other_permissions'),
      label: t('role.other_permissions'),
      isMenu: true,
      type: 1,
      children: unmatchedPermissions.map(perm => {
        const method = perm.Method || perm.method || ''
        const id = perm.id || perm.ID
        const slug = perm.Slug || perm.slug || ''
        const permissionName = getPermissionName(perm)
        
        return {
          id: `perm_${id}`, // 添加前缀
          rawId: id,
          name: permissionName,
          slug: slug,
          method: method,
          path: perm.Path || perm.path || '',
          description: perm.Description || perm.description || '',
          label: permissionName,
          displayDesc: permissionName,
          isMenu: false,
          isPermission: true
        }
      })
    }
    
    treeWithPermissions.push(otherPermissionsNode)
  }
  
  return treeWithPermissions
}

const getMenuTypeTag = (type) => {
  const typeMap = {
    1: 'info',
    2: 'success',
    3: 'warning'
  }
  return typeMap[type] || 'info'
}

const getMenuTypeText = (type) => {
  const typeMap = {
    1: t('menu.type_directory'),
    2: t('menu.type_menu'),
    3: t('menu.type_button')
  }
  return typeMap[type] || ''
}

const loadMenuPermissionTree = async () => {
  try {
    const [menuRes, permissionRes] = await Promise.all([
      getMenuTree(),
      getPermissionList({ page_size: 1000 })
    ])
    
    const menus = menuRes.data?.menus || menuRes.data?.list || []
    const permissions = permissionRes.data?.list || []
    
    menuPermissionTree.value = buildMenuPermissionTree(menus, permissions)
  } catch (error) {
    console.error('Load menu permission tree error:', error)
  }
}

const isProtectedRole = (row) => {
  const slug = row.slug || row.Slug || ''
  return protectedRoleSlugs.value.includes(slug)
}

const resetForm = () => {
  loading.value = false
  Object.assign(formData, getFormInitialValue())
  checkedKeys.value = []
  treeKey.value++
  formRef.value?.resetFields()
}

// 设置树形组件的选中状态
const setTreeCheckedKeys = async () => {
  // 如果对话框未打开，不处理
  if (!dialogVisible.value) return
  
  // 如果是新增角色，清空选中状态
  if (!formData.id && !props.editId) {
    await nextTick()
    if (menuPermissionTreeRef.value) {
      menuPermissionTreeRef.value.setCheckedKeys([], false)
      checkedKeys.value = []
    }
    return
  }
  
  // 如果是编辑角色，设置选中状态
  // 确保树组件引用和树数据都已准备好
  if (!menuPermissionTreeRef.value || menuPermissionTree.value.length === 0) {
    return
  }
  
  // 确保不在加载中
  if (loading.value) {
    return
  }
  
  // 从 formData 中获取选中的权限 ID 和菜单 ID
  const permissionIds = (formData.permission_ids || []).map(id => Number(id))
  const menuIds = (formData.menu_ids || []).map(id => Number(id))
  
  // 如果权限ID和菜单ID都为空，清空所有选中状态
  if (permissionIds.length === 0 && menuIds.length === 0) {
    await nextTick()
    if (menuPermissionTreeRef.value) {
      menuPermissionTreeRef.value.setCheckedKeys([], false)
      checkedKeys.value = []
    }
    return
  }
  
  // 构建权限ID集合，用于快速查找
  const permissionIdSet = new Set(permissionIds)
  
  // 收集树中所有节点的映射关系
  const nodeMap = new Map() // key: rawId, value: node
  const menuNodeMap = new Map() // key: menu rawId, value: menu node
  const permissionNodeMap = new Map() // key: permission rawId, value: permission node
  
  const collectNodes = (nodes) => {
    nodes.forEach(node => {
      if (node.rawId !== undefined) {
        nodeMap.set(node.rawId, node)
        if (node.isMenu) {
          menuNodeMap.set(node.rawId, node)
        } else if (node.isPermission) {
          permissionNodeMap.set(node.rawId, node)
        }
      }
      if (node.children) {
        collectNodes(node.children)
      }
    })
  }
  collectNodes(menuPermissionTree.value)
  
  // 收集需要选中的节点ID
  const keysToCheck = new Set()
  
  // 添加所有权限节点ID
  permissionIds.forEach(id => {
    const node = permissionNodeMap.get(id)
    if (node && node.id) {
      keysToCheck.add(node.id)
    }
  })
  
  menuIds.forEach(menuId => {
    const menuNode = menuNodeMap.get(menuId)
    if (!menuNode || !menuNode.id) return
    
    // 检查该菜单下是否有权限
    const menuPermissionIds = []
    if (menuNode.children) {
      menuNode.children.forEach(child => {
        if (child.isPermission && child.rawId) {
          menuPermissionIds.push(child.rawId)
        }
      })
    }
    
    const hasPermissions = menuPermissionIds.length > 0
    
    if (!hasPermissions) {
      // 如果菜单下没有权限，直接选中菜单
      keysToCheck.add(menuNode.id)
    } else {
      // 如果菜单下有权限，检查权限的选中情况
      const selectedPermissionCount = menuPermissionIds.filter(permId => permissionIdSet.has(permId)).length
      
      if (selectedPermissionCount === menuPermissionIds.length) {
        // 所有权限都被选中，选中菜单（显示全选状态）
        keysToCheck.add(menuNode.id)
      }
    }
  })
  
  try {
    // 验证节点ID是否存在于树中
    const allNodeIds = new Set()
    const collectAllNodeIds = (nodes) => {
      nodes.forEach(node => {
        if (node.id) {
          allNodeIds.add(node.id)
        }
        if (node.children) {
          collectAllNodeIds(node.children)
        }
      })
    }
    collectAllNodeIds(menuPermissionTree.value)
    
    // 过滤出存在于树中的节点ID
    const validCheckedKeys = Array.from(keysToCheck).filter(id => allNodeIds.has(id))
    
    if (validCheckedKeys.length !== keysToCheck.size) {
      console.warn('Some node IDs not found in tree:', {
        requested: Array.from(keysToCheck),
        valid: validCheckedKeys,
        allInTree: Array.from(allNodeIds)
      })
    }
    
    // 等待 DOM 更新
    await nextTick()
    
    // 先清空所有选中状态，确保不会保留之前的状态
    menuPermissionTreeRef.value.setCheckedKeys([], false)
    await nextTick()
    
    // 设置选中的节点，第二个参数为 false 表示不自动选中父节点
    // 对于菜单节点，我们已经手动添加了应该被完全选中的菜单
    // 对于权限节点，我们添加了所有应该被选中的权限
    // Element Plus 会自动根据子节点的选中状态来显示父节点的半选状态
    menuPermissionTreeRef.value.setCheckedKeys(validCheckedKeys, false)
    
    // 等待一下让树组件更新
    await nextTick()
    
    // 获取实际选中的 keys
    const actualCheckedKeys = menuPermissionTreeRef.value.getCheckedKeys() || []
    checkedKeys.value = actualCheckedKeys
  } catch (error) {
    console.error('Set checked keys error:', error)
  }
}

// 监听 editId 变化，加载详情
watch(() => props.editId, async (newId) => {
  if (newId && dialogVisible.value) {
    await loadDetail(newId)
  } else if (!newId && dialogVisible.value) {
    resetForm()
  }
}, { immediate: true })

// 监听 dialogVisible 变化
watch(dialogVisible, async (visible) => {
  if (visible) {
    await loadDepartmentTree()
    if (props.editId) {
      await loadDetail(props.editId)
    } else {
      resetForm()
    }
  }
})

// 监听树形数据加载完成，自动设置选中状态（替代固定次数的重试机制）
watch(
  [() => menuPermissionTree.value, () => dialogVisible.value, () => formData.id, () => loading.value],
  async ([tree, visible, formId, isLoading]) => {
    // 当对话框打开、树数据已加载、不在加载中、且有表单数据时，设置选中状态
    if (visible && tree && tree.length > 0 && !isLoading && (formId || props.editId)) {
      await nextTick()
      await setTreeCheckedKeys()
    }
  },
  { deep: true }
)

const loadDetail = async (id) => {
  loading.value = true
  try {
    const res = await getRoleDetail(id)
    if (res.data && res.data.role) {
      const role = res.data.role
      const rolePermissions = role.Permissions || role.permissions || []
      const roleMenus = role.Menus || role.menus || []
      
      // 确保 ID 是数字类型，与树节点的 ID 类型一致
      const permissionIds = rolePermissions.map(p => {
        const id = p.id || p.ID
        return id ? Number(id) : null
      }).filter(id => id !== null)
      
      const menuIds = roleMenus.map(m => {
        const id = m.id || m.ID
        return id ? Number(id) : null
      }).filter(id => id !== null)
      
      const roleDepartments = role.Departments || role.departments || []
      const departmentIds = roleDepartments.map(d => {
        const id = d.id || d.ID
        return id ? Number(id) : null
      }).filter(id => id !== null)

      // 确保菜单权限树数据已加载
      if (menuPermissionTree.value.length === 0) {
        await loadMenuPermissionTree()
      }
      if (departmentTree.value.length === 0) {
        await loadDepartmentTree()
      }
      
      // 使用工具函数映射字段，自动处理 snake_case 和 PascalCase
      const mapped = mapFields(role, getFormInitialValue())
      Object.assign(formData, {
        ...mapped,
        permission_ids: permissionIds,
        menu_ids: menuIds,
        department_ids: departmentIds,
        data_scope: Number(mapped.data_scope || role.data_scope || role.DataScope || 1),
        status: Number(mapped.status)
      })
      if (isProtectedRole(formData)) {
        formData.data_scope = 1
        formData.department_ids = []
      }
      // 注意：不要在这里设置 checkedKeys，因为如果包含菜单ID，会导致菜单下的所有权限被选中
      // 只设置权限ID，让 handleDialogOpened 来处理
      checkedKeys.value = []
    }
  } catch (error) {
    console.error('Load role detail error:', error)
  } finally {
    loading.value = false
  }
}

const handleDialogClose = () => {
  checkedKeys.value = []
  treeKey.value++
  if (menuPermissionTreeRef.value) {
    menuPermissionTreeRef.value.setCheckedKeys([], false)
  }
  formRef.value?.resetFields()
}

const handleDialogOpened = async () => {
  if (!formData.id && !props.editId) {
    // 新增角色，清空选中状态
    await setTreeCheckedKeys()
  } else {
    // 编辑角色，确保数据已加载
    // 如果正在加载数据，等待加载完成
    if (loading.value) {
      // 等待 loading 完成
      while (loading.value) {
        await new Promise(resolve => setTimeout(resolve, 50))
      }
    }
    
    // 如果 editId 存在但 formData.id 不存在，说明数据还没加载，先加载数据
    if (props.editId && !formData.id) {
      await loadDetail(props.editId)
    }
    
    // 确保菜单权限树数据已加载
    if (menuPermissionTree.value.length === 0) {
      await loadMenuPermissionTree()
    }
    
    // 设置选中状态（如果树数据已准备好）
    await setTreeCheckedKeys()
  }
}

const handleTreeCheck = () => {
  if (menuPermissionTreeRef.value) {
    checkedKeys.value = menuPermissionTreeRef.value.getCheckedKeys() || []
  }
}

const handleCancel = () => {
  dialogVisible.value = false
}

const handleSubmit = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (valid) {
      if (formData.id && isProtectedRole(formData) && formData.status === 0) {
        ElMessage.warning(t('role.protected_cannot_disable'))
        return
      }
      
      submitting.value = true
      try {
        // 只处理需要转换的字段
        const data = {
          ...formData,
          status: formData.status !== undefined && formData.status !== null 
            ? Number(formData.status) 
            : 1, // 确保是数字
          sort: Number(formData.sort) || 0, // 确保是数字
          description: formData.description || '', // 空字符串处理
          data_scope: isProtectedRole(formData) ? 1 : (Number(formData.data_scope) || 1),
          department_ids: isProtectedRole(formData) || Number(formData.data_scope) !== 2
            ? []
            : (formData.department_ids || []).map((id) => Number(id)).filter((id) => id > 0)
        }
        // 删除前端使用的 id 字段（如果存在）
        delete data.id
        
        // super-admin 角色不需要设置菜单和权限，因为它拥有所有权限
        if (!isProtectedRole(formData)) {
          // 获取所有选中的节点（完全选中的）
          const allCheckedKeys = menuPermissionTreeRef.value?.getCheckedKeys() || []
          
          // 收集菜单ID和权限ID
          const menuIds = []
          const permissionIds = []
          
          // 递归收集所有选中的菜单和权限ID
          const collectIds = (nodes) => {
            nodes.forEach(node => {
              const nodeId = node.id.toString()
              const isChecked = allCheckedKeys.includes(node.id)
              
              // 如果是权限节点且被选中
              if (node.isPermission && isChecked) {
                // 提取权限ID (去除 perm_ 前缀)
                if (nodeId.startsWith('perm_')) {
                  permissionIds.push(Number(nodeId.replace('perm_', '')))
                } else {
                  permissionIds.push(Number(nodeId))
                }
              }
              
              // 如果是菜单节点，检查是否应该保存
              if (node.isMenu && nodeId !== 'other_permissions') {
                // 检查该菜单下是否有权限被选中（只检查直接子权限，不包括子菜单下的权限）
                const hasDirectPermission = node.children && node.children.some(child => 
                  child.isPermission && allCheckedKeys.includes(child.id)
                )
                
                // 检查该菜单是否被完全选中（包括所有直接子权限和子菜单）
                // 如果菜单被完全选中，或者有直接权限被选中，则保存该菜单
                if (isChecked || hasDirectPermission) {
                  // 提取菜单ID (去除 menu_ 前缀)
                  if (nodeId.startsWith('menu_')) {
                    menuIds.push(Number(nodeId.replace('menu_', '')))
                  } else {
                    menuIds.push(Number(nodeId))
                  }
                }
              }
              
              // 递归处理子节点
              if (node.children) {
                collectIds(node.children)
              }
            })
          }
          
          // 辅助函数：检查节点下是否有权限被选中
          const checkHasSelectedPermission = (node, checkedKeys) => {
            if (!node.children) return false
            for (const child of node.children) {
              if (child.isPermission && checkedKeys.includes(child.id)) {
                return true
              }
              if (child.children && checkHasSelectedPermission(child, checkedKeys)) {
                return true
              }
            }
            return false
          }
          
          collectIds(menuPermissionTree.value)
          
          // 去重
          data.permission_ids = [...new Set(permissionIds)]
          data.menu_ids = [...new Set(menuIds)]
        }
        
        if (formData.id) {
          await updateRole(formData.id, data)
          ElMessage.success(t('role.update_success'))
        } else {
          await createRole(data)
          ElMessage.success(t('role.create_success'))
        }
        dialogVisible.value = false
        emit('success')
      } catch (error) {
        console.error('Submit error:', error)
        // 如果错误已经在响应拦截器中处理过，就不再重复显示
        if (!error.__handled) {
          const errorMessage = error.response?.data?.message || error.message || t('common.operation_failed')
          ElMessage.error(errorMessage)
        }
      } finally {
        submitting.value = false
      }
    }
  })
}

// 组件挂载时加载菜单权限树
onMounted(() => {
  loadMenuPermissionTree()
})

// 暴露方法供父组件调用
defineExpose({
  resetForm,
  loadDetail
})
</script>

<style scoped>
.menu-permission-container {
  border: 1px solid var(--border-color-light);
  border-radius: var(--border-radius-lg);
  background: var(--bg-color-tertiary);
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.tree-wrapper {
  max-height: 500px;
  overflow-y: auto;
  padding: 12px;
  background: var(--card-bg, #fff);
  transition: background-color 0.3s ease;
  min-height: 200px;
}

.tree-wrapper::-webkit-scrollbar {
  width: 8px;
}

.tree-wrapper::-webkit-scrollbar-track {
  background: var(--bg-color-tertiary);
  border-radius: var(--border-radius-sm);
  transition: background-color 0.3s ease;
}

.tree-wrapper::-webkit-scrollbar-thumb {
  background: var(--border-color-base);
  border-radius: var(--border-radius-sm);
  transition: background-color 0.3s ease;
}

.tree-wrapper::-webkit-scrollbar-thumb:hover {
  background: var(--text-color-secondary, #a8a8a8);
}

.menu-permission-tree {
  font-size: 14px;
}

.menu-permission-tree :deep(.el-tree-node) {
  margin-bottom: 2px;
}

.menu-permission-tree :deep(.el-tree-node__content) {
  height: 32px;
  padding: 4px 6px;
  border-radius: var(--border-radius-sm);
  margin-bottom: 2px;
  transition: all 0.2s ease;
  border: 1px solid transparent;
}

.menu-permission-tree :deep(.el-tree-node__content:hover) {
  background-color: var(--el-color-primary-light-9) !important;
  border-color: var(--el-color-primary-light-5) !important;
  transition: all 0.2s ease;
}

.menu-permission-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: var(--el-color-primary-light-9);
  border-color: var(--sidebar-active, var(--el-color-primary));
}

.menu-permission-tree :deep(.el-tree-node__expand-icon) {
  color: var(--text-color-secondary, #909399);
  font-size: 14px;
  transition: color 0.2s ease;
}

.menu-permission-tree :deep(.el-tree-node__expand-icon:hover) {
  color: var(--sidebar-active, var(--el-color-primary));
}

.menu-permission-tree :deep(.el-checkbox) {
  margin-right: 8px;
}

.permission-node,
.menu-node {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  flex: 1;
  min-width: 0;
}

.node-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.menu-icon {
  color: var(--el-color-primary);
}

.permission-icon {
  color: var(--el-color-success);
}

.permission-name,
.menu-name {
  font-weight: 500;
  /* color: #303133; */
  flex: 1;
  min-width: 0;
  word-break: break-word;
  line-height: 1.5;
}

.menu-name {
  font-size: 14px;
}

.permission-name {
  font-size: 13px;
}

.menu-type-tag {
  margin-left: 4px;
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 10px;
  font-weight: 500;
}

.permission-method {
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  letter-spacing: 0.3px;
  flex-shrink: 0;
  line-height: 1.2;
}

.method-get {
  background: var(--el-color-success);
}

.method-post {
  background: var(--el-color-primary);
}

.method-put {
  background: var(--el-color-warning);
}

.method-patch {
  background: var(--el-color-info);
}

.method-delete {
  background: var(--el-color-danger);
}

.permission-path-icon {
  color: var(--text-color-secondary);
  font-size: 12px;
  cursor: help;
  margin-left: 2px;
  transition: color 0.2s;
  flex-shrink: 0;
}

.permission-path-icon:hover {
  color: var(--el-color-primary);
}

.protected-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  padding: 8px 12px;
  background: var(--el-color-warning-light-9);
  border: 1px solid var(--el-color-warning-light-5);
  border-radius: var(--border-radius-sm);
  color: var(--el-color-warning-dark-2);
  font-size: 12px;
}

.protected-tip .el-icon {
  font-size: 14px;
}

.protected-role-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-5);
  border-radius: var(--border-radius-sm);
  color: var(--el-color-primary);
  font-size: 13px;
}

.protected-role-tip .el-icon {
  font-size: 16px;
  color: var(--el-color-primary);
}
</style>
