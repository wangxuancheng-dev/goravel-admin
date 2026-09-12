import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import logger from '@/utils/logger'
import ErrorHandler from '@/utils/errorHandler'

/**
 * CSV file import via hidden file input.
 */
export function useCsvImport(options = {}) {
  const {
    importApi,
    accept = '.csv',
    onSuccess = null,
    invalidFileTypeKey = 'common.invalid_file_type',
    successKey = 'common.import_success',
    noDataKey = 'common.import_no_data'
  } = options

  const { t } = useI18n()
  const fileInputRef = ref(null)
  const isImporting = ref(false)

  const handleImport = () => {
    fileInputRef.value?.click()
  }

  const handleFileChange = async (event) => {
    const file = event.target.files?.[0]
    if (!file) {
      return
    }

    if (!file.name.toLowerCase().endsWith('.csv')) {
      ElMessage.error(t(invalidFileTypeKey) || 'Invalid file type, please upload a CSV file')
      if (fileInputRef.value) {
        fileInputRef.value.value = ''
      }
      return
    }

    if (isImporting.value) {
      return
    }

    isImporting.value = true
    try {
      const response = await importApi(file)
      const result = response.data?.data || response.data

      if (result?.async && result?.import_id) {
        ElMessage.success(t('export.task_submitted') || t('common.operation_success') || 'Queued')
        if (onSuccess) {
          await onSuccess(result)
        }
        return
      }

      if (result.success_count > 0) {
        ElMessage.success(
          t(successKey) ||
            `Import succeeded: ${result.success_count} ok, ${result.failed_count} failed`
        )

        if (result.failed_count > 0 && result.errors?.length) {
          const errorMsg = result.errors.slice(0, 10).join('\n')
          ElMessage.warning(
            result.errors.length > 10
              ? `Partial import failed, first 10 errors:\n${errorMsg}\n...`
              : `Partial import failed:\n${errorMsg}`
          )
        }

        if (onSuccess) {
          await onSuccess(result)
        }
      } else {
        ElMessage.warning(t(noDataKey) || 'No rows were imported')
        if (result.errors?.length) {
          ElMessage.error(`Import failed:\n${result.errors.slice(0, 10).join('\n')}`)
        }
      }
    } catch (error) {
      logger.error('CSV import error:', error)
      if (!error.__handled) {
        ErrorHandler.handle(error, { silent: true })
      }
    } finally {
      isImporting.value = false
      if (fileInputRef.value) {
        fileInputRef.value.value = ''
      }
    }
  }

  return {
    fileInputRef,
    isImporting,
    handleImport,
    handleFileChange,
    accept
  }
}
