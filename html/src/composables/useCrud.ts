import { ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { compact, map } from 'lodash-es'
import logger from '../utils/logger'
import { promptSensitiveConfirm } from '../utils/sensitiveConfirm'

/**
 * CRUD 配置选项
 */
export interface UseCrudOptions {
  /** 单个删除 API 函数 */
  deleteApi?: (id: number | string, data?: Record<string, unknown>) => Promise<any>
  /** 批量删除 API 函数 */
  batchDeleteApi?: (ids: (number | string)[]) => Promise<any>
  /** 删除确认提示的 i18n key */
  deleteConfirmKey?: string
  /** 删除成功提示的 i18n key */
  deleteSuccessKey?: string
  /** 批量删除确认提示的 i18n key */
  batchDeleteConfirmKey?: string
  /** 提示框标题的 i18n key */
  tipKey?: string
  /** 删除前二次确认（TOTP / 当前密码 → confirm_code） */
  requireSensitiveConfirm?: boolean
  /** 删除成功回调 */
  onDeleteSuccess?: (deletedItems: any, deletedIds: number | string | (number | string)[]) => void
  /** 删除失败回调 */
  onDeleteError?: (error: any, failedItems: any) => void
  /** 删除前置处理函数，返回 false 取消删除 */
  beforeDelete?: (item: any, id: number | string) => Promise<boolean | void> | boolean | void
  /** 删除后置处理函数 */
  afterDelete?: (item: any, id: number | string) => void
}

/**
 * CRUD 返回值类型
 */
export interface UseCrudReturn {
  /** 对话框显示状态 */
  dialogVisible: Ref<boolean>
  /** 当前编辑的 ID */
  editId: Ref<number | string | null>
  /** 打开添加对话框 */
  handleAdd: () => void
  /** 打开编辑对话框 */
  handleEdit: (rowOrId: { id?: number | string; ID?: number | string } | number | string) => void
  /** 关闭对话框 */
  handleClose: () => void
  /** 表单提交成功后处理 */
  handleFormSuccess: (reloadData?: () => void) => void
  /** 处理单个删除 */
  handleDelete: (
    rowOrId: { id?: number | string; ID?: number | string } | number | string,
    reloadData?: () => void
  ) => Promise<void>
  /** 处理批量删除 */
  handleBatchDelete: (
    rows: Array<{ id?: number | string; ID?: number | string }>,
    reloadData?: () => void
  ) => Promise<void>
}

/**
 * CRUD 操作通用 composable
 *
 * @description 提供通用的增删改查操作，支持按需使用
 *
 * @example
 * // 1. 只需要添加/编辑功能（不需要删除）
 * const { dialogVisible, editId, handleAdd, handleEdit, handleClose, handleFormSuccess } = useCrud()
 *
 * // 2. 需要删除功能
 * const { handleDelete } = useCrud({ deleteApi: deleteXxx })
 *
 * // 3. 需要批量删除功能
 * const { handleBatchDelete } = useCrud({ batchDeleteApi: batchDeleteXxx })
 *
 * // 4. 完整功能
 * const { handleDelete, handleBatchDelete } = useCrud({ deleteApi, batchDeleteApi })
 */
export function useCrud(options: UseCrudOptions = {}): UseCrudReturn {
  const {
    deleteApi = null,
    batchDeleteApi = null,
    deleteConfirmKey = '',
    deleteSuccessKey = '',
    batchDeleteConfirmKey = '',
    tipKey = 'form.tip',
    requireSensitiveConfirm = false,
    onDeleteSuccess = null,
    onDeleteError = null,
    beforeDelete = null,
    afterDelete = null
  } = options

  const { t } = useI18n()

  // 对话框显示状态
  const dialogVisible = ref(false)
  // 编辑ID
  const editId = ref<number | string | null>(null)

  /**
   * 打开添加对话框
   */
  const handleAdd = (): void => {
    editId.value = null
    dialogVisible.value = true
  }

  /**
   * 打开编辑对话框
   */
  const handleEdit = (
    rowOrId: { id?: number | string; ID?: number | string } | number | string
  ): void => {
    if (typeof rowOrId === 'object' && rowOrId !== null) {
      editId.value = rowOrId.id ?? rowOrId.ID ?? null
    } else {
      editId.value = rowOrId as number | string
    }
    dialogVisible.value = true
  }

  /**
   * 关闭对话框
   */
  const handleClose = (): void => {
    dialogVisible.value = false
    editId.value = null
  }

  /**
   * 表单提交成功后的处理
   */
  const handleFormSuccess = (reloadData?: () => void): void => {
    handleClose()
    if (reloadData && typeof reloadData === 'function') {
      reloadData()
    }
  }

  /**
   * 删除操作
   */
  const handleDelete = async (
    rowOrId: { id?: number | string; ID?: number | string } | number | string,
    reloadData?: () => void
  ): Promise<void> => {
    try {
      // 获取ID
      let id: number | string | undefined
      if (typeof rowOrId === 'object' && rowOrId !== null) {
        id = rowOrId.id ?? rowOrId.ID
      } else {
        id = rowOrId as number | string
      }

      if (!id) {
        logger.error('useCrud: delete id is required')
        return
      }

      // 删除前的钩子
      if (beforeDelete) {
        const result = await beforeDelete(rowOrId, id as number | string)
        if (result === false) {
          return // 取消删除
        }
      }

      // 确认删除
      const confirmKey = deleteConfirmKey || 'common.delete_confirm'
      await ElMessageBox.confirm(t(confirmKey), t(tipKey), {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      })

      // 执行删除
      if (!deleteApi) {
        logger.error('useCrud: deleteApi is required')
        return
      }

      if (requireSensitiveConfirm) {
        const confirmCode = await promptSensitiveConfirm(t)
        await deleteApi(id, { confirm_code: confirmCode })
      } else {
        await deleteApi(id)
      }

      // 删除成功提示
      const successKey = deleteSuccessKey || 'common.delete_success'
      ElMessage.success(t(successKey))

      // 删除成功回调
      if (onDeleteSuccess) {
        onDeleteSuccess(rowOrId, id as number | string)
      }

      // 删除后钩子
      if (afterDelete) {
        afterDelete(rowOrId, id as number | string)
      }

      // 重新加载数据
      if (reloadData && typeof reloadData === 'function') {
        reloadData()
      }
    } catch (error) {
      if (error === 'cancel') {
        return // 用户取消
      }

      logger.error('Delete error:', error)

      // 删除失败回调
      if (onDeleteError) {
        onDeleteError(error, rowOrId)
      }
    }
  }

  /**
   * 批量删除
   */
  const handleBatchDelete = async (
    rows: Array<{ id?: number | string; ID?: number | string }>,
    reloadData?: () => void
  ): Promise<void> => {
    if (!rows || rows.length === 0) {
      ElMessage.warning(t('common.please_select_items'))
      return
    }

    try {
      const confirmKey = batchDeleteConfirmKey || 'common.batch_delete_confirm'
      const count = rows.length

      await ElMessageBox.confirm(t(confirmKey, { count }), t(tipKey), {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      })

      const ids = compact(map(rows, (row) => row.id ?? row.ID)) as (number | string)[]
      if (ids.length === 0) {
        return
      }

      // 优先使用批量删除 API，否则循环调用单个删除 API
      if (batchDeleteApi) {
        await batchDeleteApi(ids)
      } else if (deleteApi) {
        for (const id of ids) {
          await deleteApi(id)
        }
      } else {
        logger.error('useCrud: deleteApi or batchDeleteApi is required for batch delete')
        return
      }

      const successKey = deleteSuccessKey || 'common.delete_success'
      ElMessage.success(t(successKey))

      if (onDeleteSuccess) {
        onDeleteSuccess(rows, ids)
      }

      if (reloadData && typeof reloadData === 'function') {
        reloadData()
      }
    } catch (error) {
      if (error === 'cancel') {
        return
      }

      logger.error('Batch delete error:', error)

      if (onDeleteError) {
        onDeleteError(error, rows)
      }
    }
  }

  return {
    // 状态
    dialogVisible,
    editId,

    // 方法
    handleAdd,
    handleEdit,
    handleClose,
    handleFormSuccess,
    handleDelete,
    handleBatchDelete
  }
}

