import { ref, computed, watch, watchEffect, nextTick } from 'vue'
import { getOptions } from '../../api/option'
import { buildAdminAuthHeaders } from '../../utils/authHeaders'

export function useTreeSelect({ field, modelValue, onUpdate }) {
  const popoverVisible = ref(false)
  const filterText = ref('')
  const treeData = ref([])
  const fieldOptionsCache = ref({})
  const selectedLabel = ref('') // 保存选中节点的标签

  // 获取树形选择器显示值
  const getTreeSelectDisplayValue = (fieldObj, value) => {
    if (!value) return ''
    
    // 获取树形数据
    let dataSource = []
    if (typeof fieldObj.treeData === 'function') {
      try {
        dataSource = fieldObj.treeData() || []
      } catch (e) {
        console.error('Error getting treeData for display:', e)
        dataSource = []
      }
    } else if (Array.isArray(fieldObj.treeData)) {
      dataSource = fieldObj.treeData
    } else {
      dataSource = treeData.value || []
    }
    
    if (!Array.isArray(dataSource) || dataSource.length === 0) {
      return ''
    }
    
    const findNode = (data, targetId) => {
      for (const node of data) {
        const nodeId = node[fieldObj.treeProps?.value || 'id']
        if (nodeId == targetId) {
          return node[fieldObj.treeProps?.label || 'name']
        }
        if (node[fieldObj.treeProps?.children || 'children']) {
          const found = findNode(node[fieldObj.treeProps?.children || 'children'], targetId)
          if (found) return found
        }
      }
      return ''
    }
    
    return findNode(dataSource, value) || ''
  }

  // 计算输入框显示值（用于显示选中值的标签）
  const inputValue = computed(() => {
    const selectedValue = modelValue.value
    if (selectedValue !== null && selectedValue !== undefined) {
      // 特殊处理：id=0 表示顶级节点
      if (selectedValue === 0 || selectedValue === '0') {
        if (field.topNodeLabel) {
          return typeof field.topNodeLabel === 'function' 
            ? field.topNodeLabel() 
            : field.topNodeLabel
        }
        return ''
      }
      
      // 优先使用保存的标签，如果不存在则通过查找获取
      if (selectedLabel.value) {
        return selectedLabel.value
      }
      
      // 尝试从 treeData 或 field.treeData 获取标签
      const label = getTreeSelectDisplayValue(field, selectedValue)
      if (label) {
        return label
      }
      
      // 如果还是找不到，尝试从 field.treeData 直接获取
      let dataSource = []
      if (typeof field.treeData === 'function') {
        try {
          dataSource = field.treeData() || []
        } catch (e) {
          console.error('Error getting treeData in inputValue:', e)
        }
      } else if (Array.isArray(field.treeData)) {
        dataSource = field.treeData
      }
      
      if (dataSource.length > 0) {
        const idKey = field.treeProps?.value || 'id'
        const childrenKey = field.treeProps?.children || 'children'
        const node = findNodeById(dataSource, selectedValue, idKey, childrenKey)
        if (node) {
          const labelKey = field.treeProps?.label || 'label'
          const nameKey = field.treeProps?.name || 'name'
          return node[labelKey] || node[nameKey] || ''
        }
      }
    }
    return ''
  })

  // 更新弹窗显示状态
  const updatePopoverVisible = (visible) => {
    popoverVisible.value = visible
    // 关闭弹窗时，如果有选中值，清空搜索文本（恢复显示选中值）
    // 但只有在没有输入文本时才清空，避免清空用户正在输入的内容
    if (!visible && modelValue.value && !filterText.value) {
      filterText.value = ''
    }
    // 关闭弹窗时，如果没有选中值且没有输入文本，也清空搜索文本
    if (!visible && !modelValue.value && !filterText.value) {
      filterText.value = ''
    }
  }

  // 切换弹窗显示状态
  const togglePopover = () => {
    popoverVisible.value = !popoverVisible.value
    // 关闭弹窗时，如果有选中值，清空搜索文本（恢复显示选中值）
    // 但只有在没有输入文本时才清空，避免清空用户正在输入的内容
    if (!popoverVisible.value && modelValue.value && !filterText.value) {
      filterText.value = ''
    }
    // 关闭弹窗时，如果没有选中值且没有输入文本，也清空搜索文本
    if (!popoverVisible.value && !modelValue.value && !filterText.value) {
      filterText.value = ''
    }
  }

  // 处理节点点击
  const handleNodeClick = (data) => {
    const valueKey = field.treeProps?.value || 'id'
    const value = data[valueKey]
    // 保存选中节点的标签
    const labelKey = field.treeProps?.label || 'label'
    const nameKey = field.treeProps?.name || 'name'
    selectedLabel.value = data[labelKey] || data[nameKey] || ''
    onUpdate(value)
    filterText.value = ''
  }

  // 处理清除
  const handleClear = () => {
    onUpdate(null)
    filterText.value = ''
    selectedLabel.value = '' // 清空保存的标签
  }

  // 处理输入
  const handleInput = (val) => {
    const inputVal = val || ''
    filterText.value = inputVal
    // 如果开始输入，清空选中值（允许重新选择）
    if (inputVal && modelValue.value) {
      // 如果输入的内容与选中值的显示文本不同，清空选中值
      const displayValue = getTreeSelectDisplayValue(field, modelValue.value)
      if (inputVal !== displayValue) {
        onUpdate(null)
      }
    }
    // 输入时自动打开弹窗
    if (inputVal && !popoverVisible.value) {
      togglePopover()
    }
  }

  // 获取过滤后的树形数据
  const getFilteredTreeData = (data) => {
    if (!filterText.value || filterText.value === '') {
      return data || []
    }
    
    const labelKey = field.treeProps?.label || 'name'
    const childrenKey = field.treeProps?.children || 'children'
    
    const filterNode = (node) => {
      const label = node[labelKey] || ''
      const matches = label.toLowerCase().includes(filterText.value.toLowerCase())
      
      if (node[childrenKey] && Array.isArray(node[childrenKey])) {
        const filteredChildren = node[childrenKey].map(child => filterNode(child)).filter(Boolean)
        if (matches || filteredChildren.length > 0) {
          return {
            ...node,
            [childrenKey]: filteredChildren
          }
        }
      } else if (matches) {
        return node
      }
      
      return null
    }
    
    return (data || []).map(node => filterNode(node)).filter(Boolean)
  }

  function findNodeById(list, id, idKey = 'id', childrenKey = 'children') {
    // 统一转换为字符串，避免数字/字符串类型不匹配
    const targetId = String(id);
    for (const node of list || []) {
      const nodeId = String(node[idKey]);
      if (nodeId === targetId) return node;
      const children = node[childrenKey];
      if (children?.length) {
        const found = findNodeById(children, id, idKey, childrenKey);
        if (found) return found;
      }
    }
    return null;
  }

  function selectById(id) {
    if (id === null || id === undefined) return
    
    // 特殊处理：id=0 表示顶级/根节点，不在树中
    if (id === 0 || id === '0') {
      // 对于顶级节点，使用 topNodeLabel 或默认文本
      if (field.topNodeLabel) {
        selectedLabel.value = typeof field.topNodeLabel === 'function' 
          ? field.topNodeLabel() 
          : field.topNodeLabel
      } else {
        selectedLabel.value = ''
      }
      return
    }
    
    // 获取数据源
    let dataSource = []
    if (treeData.value?.length) {
      dataSource = treeData.value
    } else {
      // 如果树形数据未加载，尝试从 field.treeData 获取
      if (typeof field.treeData === 'function') {
        try {
          dataSource = field.treeData() || []
        } catch (e) {
          console.error('Error getting treeData in selectById:', e)
        }
      } else if (Array.isArray(field.treeData)) {
        dataSource = field.treeData
      }
    }
    
    if (dataSource.length === 0) return
    
    const idKey = field.treeProps?.value || 'id'
    const childrenKey = field.treeProps?.children || 'children'
    const labelKey = field.treeProps?.label || 'label'
    const nameKey = field.treeProps?.name || 'name'
    
    const node = findNodeById(dataSource, id, idKey, childrenKey)
    if (node) {
      const label = node[labelKey] || node[nameKey] || ''
      selectedLabel.value = label
    }
  }

  // 加载树形数据
  const loadData = async () => {
    if (!field.apiUrl) {
      // 处理 treeData（可能是函数或数组）
      let data = null
      if (typeof field.treeData === 'function') {
        try {
          data = field.treeData()
        } catch (e) {
          console.error('Error getting treeData:', e)
          data = []
        }
      } else {
        data = field.treeData
      }
      
      if (data && Array.isArray(data)) {
        treeData.value = getFilteredTreeData(data)
      } else {
        treeData.value = []
      }
      return
    }
    
    const cacheKey = field.apiUrl
    if (fieldOptionsCache.value[cacheKey] !== undefined) {
      const data = fieldOptionsCache.value[cacheKey]
      if (Array.isArray(data)) {
        treeData.value = getFilteredTreeData(data)
      }
      return
    }
    
    try {
      fieldOptionsCache.value[cacheKey] = null
      
      if (field.apiUrl.startsWith('/options')) {
        const url = new URL(field.apiUrl, window.location.origin)
        const type = url.searchParams.get('type')
        const params = {}
        for (const [key, value] of url.searchParams) {
          if (key !== 'type') {
            params[key] = value
          }
        }
        const res = await getOptions(type, params)
        if (res.data) {
          if (res.data.options && Array.isArray(res.data.options)) {
            fieldOptionsCache.value[cacheKey] = res.data.options
            treeData.value = getFilteredTreeData(res.data.options)
          } else if (res.data.list && Array.isArray(res.data.list)) {
            fieldOptionsCache.value[cacheKey] = res.data.list
            treeData.value = getFilteredTreeData(res.data.list)
          }
        }
      } else {
        const res = await fetch(field.apiUrl, {
          method: 'GET',
          headers: {
            ...buildAdminAuthHeaders(),
            'Content-Type': 'application/json'
          }
        })
        if (res.ok) {
          const data = await res.json()
          let options = []
          if (data.data && data.data.options) {
            options = data.data.options
          } else if (data.data && data.data.list) {
            options = data.data.list
          } else if (data.options) {
            options = data.options
          } else if (Array.isArray(data.data)) {
            options = data.data
          } else if (Array.isArray(data)) {
            options = data
          }
          fieldOptionsCache.value[cacheKey] = options
          treeData.value = getFilteredTreeData(options)
        }
      }
    } catch (error) {
      console.error('Load tree select data error:', error)
      fieldOptionsCache.value[cacheKey] = []
      treeData.value = []
    }
  }

  // 监听过滤文本变化，更新树形数据
  watch(filterText, () => {
    const currentTreeData = typeof field.treeData === 'function' ? field.treeData() : field.treeData
    if (currentTreeData && Array.isArray(currentTreeData)) {
      treeData.value = getFilteredTreeData(currentTreeData)
    } else if (field.apiUrl) {
      const cacheKey = field.apiUrl
      if (fieldOptionsCache.value[cacheKey]) {
        treeData.value = getFilteredTreeData(fieldOptionsCache.value[cacheKey])
      }
    }
  })

  watch(
    [() => treeData.value, () => modelValue.value],
    async ([newTreeData, newValue], [oldTreeData, oldValue]) => {
      // 确保值有效（排除 null/undefined）
      if (newValue !== null && newValue !== undefined) {
        // 等待 DOM 更新完成后再执行选中逻辑
        await nextTick();
        
        // 无论树形数据是否加载，都尝试设置标签
        // selectById 函数内部会处理数据源的选择
        selectById(newValue);
      } else if (newValue === null || newValue === undefined) {
        // 如果值被清空，也清空标签
        selectedLabel.value = ''
      }
    },
    { immediate: true, deep: true }
  );

  // 更新树形数据的函数
  const updateTreeData = (newTreeData) => {
    if (newTreeData && Array.isArray(newTreeData)) {
      if (newTreeData.length > 0) {
        treeData.value = getFilteredTreeData(newTreeData)
      } else {
        treeData.value = []
      }
    } else {
      treeData.value = []
    }
  }
  
  // 监听 field.treeData 变化，更新树形数据
  // 如果 treeData 是函数，使用 watchEffect 来确保能够追踪到所有响应式依赖
  if (typeof field.treeData === 'function') {
    // 使用 watchEffect 自动追踪函数内部访问的所有响应式依赖
    watchEffect(() => {
      try {
        const result = field.treeData()
        const data = Array.isArray(result) ? result : []
        updateTreeData(data)
        
        // 数据更新后，如果 modelValue 有值，尝试设置标签
        if (modelValue.value !== null && modelValue.value !== undefined) {
          nextTick(() => {
            selectById(modelValue.value)
          })
        }
      } catch (e) {
        console.error('Error getting treeData:', e)
        updateTreeData([])
      }
    }, { flush: 'post' })
  } else if (Array.isArray(field.treeData)) {
    // 如果 treeData 是数组，直接监听
    watch(
      () => field.treeData,
      (newData) => {
        updateTreeData(newData)
        // 数据更新后，如果 modelValue 有值，尝试设置标签
        if (modelValue.value !== null && modelValue.value !== undefined) {
          nextTick(() => {
            selectById(modelValue.value)
          })
        }
      },
      { deep: true, immediate: true, flush: 'post' }
    )
  }

  // 初始化加载数据
  // 注意：如果 treeData 是函数，watch 会在 immediate: true 时自动执行，这里不需要手动初始化
  // 但如果 treeData 是数组或使用 apiUrl，需要手动初始化
  if (typeof field.treeData === 'function') {
    // 函数形式由 watch 处理，但为了确保初始数据正确，也在这里初始化一次
    try {
      const initialData = field.treeData()
      if (initialData && Array.isArray(initialData) && initialData.length > 0) {
        treeData.value = getFilteredTreeData(initialData)
      }
    } catch (e) {
      console.error('Error initializing treeData:', e)
    }
  } else if (Array.isArray(field.treeData) && field.treeData.length > 0) {
    treeData.value = getFilteredTreeData(field.treeData)
  } else if (field.apiUrl) {
    loadData()
  }

  return {
    popoverVisible,
    filterText,
    treeData,
    inputValue,
    updatePopoverVisible,
    togglePopover,
    handleNodeClick,
    handleClear,
    handleInput,
    loadData,
    selectById
  }
}

