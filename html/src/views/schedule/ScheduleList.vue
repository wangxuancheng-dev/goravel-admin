<template>
  <div class="schedule-page">
    <el-card shadow="never">
      <template #header>
        <div class="schedule-page__header">
          <div>
            <h2 class="schedule-page__title">{{ t('menu.schedule') }}</h2>
            <p class="schedule-page__hint">{{ t('schedule.hint') }}</p>
          </div>
          <el-button :loading="loading" @click="loadData">
            {{ t('common.refresh') }}
          </el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="tableData" row-key="id" border>
        <el-table-column prop="command" :label="t('schedule.command')" min-width="220" />
        <el-table-column :label="t('schedule.description')" min-width="180">
          <template #default="{ row }">
            {{ resolveDescription(row) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('schedule.cron')" min-width="220">
          <template #default="{ row }">
            <div class="schedule-cron">
              <code class="schedule-cron__expr">{{ row.cron || '-' }}</code>
              <div v-if="row.cron" class="schedule-cron__label">{{ describeCron(row.cron, t) }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('schedule.on_one_server')" width="110" align="center">
          <template #default="{ row }">
            <el-tag :type="row.on_one_server ? 'success' : 'info'" size="small">
              {{ row.on_one_server ? t('common.yes') : t('common.no') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('schedule.last_status')" width="110" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.last_status)" size="small">
              {{ statusLabel(row.last_status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_run_at" :label="t('schedule.last_run_at')" min-width="160">
          <template #default="{ row }">
            {{ row.last_run_at || '-' }}
          </template>
        </el-table-column>
        <el-table-column :label="t('schedule.last_duration')" width="110" align="right">
          <template #default="{ row }">
            {{ formatDuration(row.last_duration_ms) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="180" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              type="primary"
              link
              :loading="runningCommand === row.command"
              :disabled="getButtonState('schedule.run').disabled || !!runningCommand"
              @click="handleRun(row)"
            >
              {{ t('schedule.run_now') }}
            </el-button>
            <el-button
              link
              :disabled="row.last_status === 'never' || !row.last_status"
              @click="openResultFromRow(row)"
            >
              {{ t('schedule.view_result') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      v-model="resultVisible"
      :title="t('schedule.result_title')"
      width="720px"
      destroy-on-close
    >
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item :label="t('schedule.command')">
          {{ resultData.command || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('schedule.last_status')">
          <el-tag :type="statusTagType(resultData.status)" size="small">
            {{ statusLabel(resultData.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="t('schedule.last_run_at')">
          {{ resultData.run_at || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('schedule.last_duration')">
          {{ formatDuration(resultData.duration_ms) }}
        </el-descriptions-item>
        <el-descriptions-item v-if="resultData.error" :label="t('schedule.error')">
          <pre class="schedule-result-pre schedule-result-pre--error">{{ resultData.error }}</pre>
        </el-descriptions-item>
        <el-descriptions-item :label="t('schedule.output')">
          <pre class="schedule-result-pre">{{ resultData.output || t('schedule.output_empty') }}</pre>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { getScheduleList, runSchedule } from '@/api/schedule'
import { usePermission } from '@/composables/usePermission'
import { describeCron } from '@/utils/cronLabel'
import { logger } from '@/utils/logger'

const { t, te } = useI18n()
const { getButtonState } = usePermission()

const loading = ref(false)
const tableData = ref([])
const runningCommand = ref('')
const resultVisible = ref(false)
const resultData = reactive({
  command: '',
  status: '',
  error: '',
  output: '',
  duration_ms: undefined,
  run_at: '',
})

function resolveDescription(row) {
  const key = `schedule.commands.${String(row.command || '').replace(/:/g, '_')}`
  if (te(key)) return t(key)
  return row.description || row.command || '-'
}

function statusLabel(status) {
  if (status === 'success') return t('schedule.status_success')
  if (status === 'failed') return t('schedule.status_failed')
  return t('schedule.status_never')
}

function statusTagType(status) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  return 'info'
}

function formatDuration(ms) {
  if (ms === undefined || ms === null || ms === '') return '-'
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(2)} s`
}

function showResult(payload = {}) {
  resultData.command = payload.command || ''
  resultData.status = payload.status || ''
  resultData.error = payload.error || ''
  resultData.output = payload.output || ''
  resultData.duration_ms = payload.duration_ms
  resultData.run_at = payload.run_at || ''
  resultVisible.value = true
}

function openResultFromRow(row) {
  showResult({
    command: row.command,
    status: row.last_status,
    error: row.last_error,
    output: row.last_output,
    duration_ms: row.last_duration_ms,
    run_at: row.last_run_at,
  })
}

async function loadData() {
  loading.value = true
  try {
    const res = await getScheduleList()
    tableData.value = res.data?.list || []
  } catch (error) {
    logger.error('Failed to load schedules:', error)
  } finally {
    loading.value = false
  }
}

async function handleRun(row) {
  try {
    await ElMessageBox.confirm(
      t('schedule.run_confirm', { command: row.command }),
      t('schedule.run_now'),
      { type: 'warning' },
    )
  } catch {
    return
  }

  runningCommand.value = row.command
  try {
    const res = await runSchedule(row.command)
    const result = res.data?.result || {}
    if (result.status === 'failed') {
      ElMessage.warning(t('schedule.run_failed'))
    } else {
      ElMessage.success(t('schedule.run_success'))
    }
    showResult(result)
    await loadData()
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.message || t('common.operation_failed'))
    }
    await loadData()
  } finally {
    runningCommand.value = ''
  }
}

onMounted(loadData)
</script>

<style scoped>
.schedule-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.schedule-page__title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.schedule-page__hint {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.schedule-cron__expr {
  font-family: Consolas, 'Courier New', ui-monospace, monospace;
  font-size: 12px;
}

.schedule-cron__label {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}

.schedule-result-pre {
  margin: 0;
  max-height: 320px;
  overflow: auto;
  white-space: pre;
  word-break: normal;
  font-family: Consolas, 'Courier New', ui-monospace, SFMono-Regular, Menlo, Monaco, monospace;
  font-size: 12px;
  line-height: 1.6;
  tab-size: 2;
}

.schedule-result-pre--error {
  color: var(--el-color-danger);
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
