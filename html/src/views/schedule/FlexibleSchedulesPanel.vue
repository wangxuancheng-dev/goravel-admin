<template>
  <el-card shadow="never" class="flexible-card">
    <template #header>
      <div class="flexible-card__header">
        <h3 class="flexible-card__title">{{ t('schedule.flexible_title') }}</h3>
        <el-button :loading="loading" @click="loadData">{{ t('common.refresh') }}</el-button>
      </div>
    </template>

    <el-table v-loading="loading" :data="rows" row-key="id" border>
      <el-table-column prop="name" :label="t('schedule.flexible_name')" min-width="120" />
      <el-table-column :label="t('schedule.flexible_handler')" min-width="160">
        <template #default="{ row }">
          <div>{{ row.handler_name || row.handler }}</div>
          <code class="muted">{{ row.handler }}</code>
        </template>
      </el-table-column>
      <el-table-column :label="t('schedule.cron')" min-width="200">
        <template #default="{ row }">
          <code>{{ row.cron_expr || '-' }}</code>
          <div v-if="row.cron_expr" class="muted">{{ describeCron(row.cron_expr, t) }}</div>
        </template>
      </el-table-column>
      <el-table-column prop="timezone" :label="t('schedule.flexible_timezone')" width="120" />
      <el-table-column :label="t('common.status')" width="90" align="center">
        <template #default="{ row }">
          <el-switch
            :model-value="row.enabled"
            :disabled="getButtonState('flexible_schedule.update').disabled"
            @change="(v) => onToggle(row, v)"
          />
        </template>
      </el-table-column>
      <el-table-column :label="t('schedule.last_status')" width="100" align="center">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.last_status)" size="small">
            {{ statusLabel(row.last_status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="last_run_at" :label="t('schedule.last_run_at')" min-width="150">
        <template #default="{ row }">{{ formatDateTimeDisplay(row.last_run_at) }}</template>
      </el-table-column>
      <el-table-column :label="t('schedule.last_duration')" width="100" align="right">
        <template #default="{ row }">{{ formatDuration(row.last_duration_ms) }}</template>
      </el-table-column>
      <el-table-column :label="t('schedule.flexible_next_runs')" min-width="170">
        <template #default="{ row }">
          <div v-if="row.next_runs?.length" class="muted">
            <div v-for="r in row.next_runs.slice(0, 2)" :key="r">{{ formatDateTimeDisplay(r) }}</div>
          </div>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.operation')" width="240" fixed="right" align="center">
        <template #default="{ row }">
          <el-button
            type="primary"
            link
            :disabled="getButtonState('flexible_schedule.update').disabled"
            @click="openEdit(row)"
          >
            {{ t('common.edit') }}
          </el-button>
          <el-button
            type="primary"
            link
            :loading="runningId === row.id"
            :disabled="getButtonState('flexible_schedule.run').disabled"
            @click="onRun(row)"
          >
            {{ t('schedule.run_now') }}
          </el-button>
          <el-button
            link
            :disabled="!row.last_status || row.last_status === 'never'"
            @click="openResult(row)"
          >
            {{ t('schedule.view_result') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="formVisible" :title="t('common.edit')" width="520px">
      <el-form :model="form" label-width="100px">
        <el-form-item :label="t('schedule.cron')" required>
          <el-input v-model="form.cron_expr" placeholder="*/5 * * * *" />
          <div class="muted">{{ t('schedule.flexible_cron_tip') }}</div>
        </el-form-item>
        <el-form-item :label="t('schedule.flexible_timezone')" required>
          <el-select
            v-model="form.timezone"
            filterable
            style="width: 100%"
            :placeholder="DEFAULT_TIMEZONE"
          >
            <el-option
              v-for="opt in timezoneOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('common.status')">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item>
          <el-button @click="onPreview">{{ t('schedule.flexible_preview') }}</el-button>
        </el-form-item>
        <el-form-item v-if="previewRuns.length" :label="t('schedule.flexible_next_runs')">
          <div class="muted">
            <div v-for="r in previewRuns" :key="r">{{ formatDateTimeDisplay(r) }}</div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="resultVisible" :title="t('schedule.result_title')" width="720px">
      <el-descriptions :column="1" border>
        <el-descriptions-item :label="t('schedule.flexible_name')">
          {{ resultData.name || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('schedule.flexible_handler')">
          {{ resultData.handler_name || resultData.handler || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('schedule.last_status')">
          <el-tag :type="statusTagType(resultData.last_status)" size="small">
            {{ statusLabel(resultData.last_status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="t('schedule.last_run_at')">
          {{ resultData.last_run_at || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('schedule.last_duration')">
          {{ formatDuration(resultData.last_duration_ms) }}
        </el-descriptions-item>
        <el-descriptions-item v-if="resultData.last_error" :label="t('schedule.error')">
          <pre class="schedule-result-pre schedule-result-pre--error">{{ resultData.last_error }}</pre>
        </el-descriptions-item>
        <el-descriptions-item :label="t('schedule.output')">
          <pre class="schedule-result-pre">{{ resultData.last_output || t('schedule.output_empty') }}</pre>
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button type="primary" @click="resultVisible = false">{{ t('common.close') }}</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  getFlexibleScheduleList,
  previewFlexibleSchedule,
  runFlexibleSchedule,
  updateFlexibleSchedule,
} from '@/api/schedule'
import { usePermission } from '@/composables/usePermission'
import { describeCron } from '@/utils/cronLabel'
import { formatDateTimeDisplay } from '@/utils/dateUtils'
import { logger } from '@/utils/logger'
import { DEFAULT_TIMEZONE, timezoneSelectOptions } from '@/utils/timezoneOptions'

const { t } = useI18n()
const { getButtonState } = usePermission()

const loading = ref(false)
const saving = ref(false)
const rows = ref([])
const runningId = ref(null)
const formVisible = ref(false)
const editing = ref(null)
const previewRuns = ref([])
const resultVisible = ref(false)
const resultData = reactive({
  name: '',
  handler: '',
  handler_name: '',
  last_status: '',
  last_run_at: '',
  last_duration_ms: undefined,
  last_error: '',
  last_output: '',
})
const form = reactive({
  cron_expr: '*/5 * * * *',
  timezone: DEFAULT_TIMEZONE,
  enabled: true,
})
const timezoneOptions = computed(() => timezoneSelectOptions(form.timezone))

function statusLabel(status) {
  if (status === 'success') return t('schedule.status_success')
  if (status === 'failed') return t('schedule.status_failed')
  if (status === 'skipped') return t('schedule.flexible_status_skipped')
  return t('schedule.status_never')
}

function statusTagType(status) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'skipped') return 'warning'
  return 'info'
}

function formatDuration(ms) {
  if (ms === undefined || ms === null || ms === '') return '-'
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(2)} s`
}

function openResult(row = {}) {
  resultData.name = row.name || ''
  resultData.handler = row.handler || ''
  resultData.handler_name = row.handler_name || ''
  resultData.last_status = row.last_status || ''
  resultData.last_run_at = row.last_run_at || ''
  resultData.last_duration_ms = row.last_duration_ms
  resultData.last_error = row.last_error || ''
  resultData.last_output = row.last_output || ''
  resultVisible.value = true
}

async function loadData() {
  loading.value = true
  try {
    const listRes = await getFlexibleScheduleList()
    rows.value = listRes.data?.list || []
  } catch (error) {
    logger.error('Failed to load flexible schedules:', error)
  } finally {
    loading.value = false
  }
}

function openEdit(row) {
  editing.value = row
  Object.assign(form, {
    cron_expr: row.cron_expr,
    timezone: row.timezone || DEFAULT_TIMEZONE,
    enabled: !!row.enabled,
  })
  previewRuns.value = row.next_runs || []
  formVisible.value = true
}

async function onPreview() {
  try {
    const res = await previewFlexibleSchedule({
      cron_expr: form.cron_expr,
      timezone: form.timezone || DEFAULT_TIMEZONE,
      count: 5,
    })
    previewRuns.value = res.data?.next_runs || []
  } catch (error) {
    logger.error('preview failed', error)
  }
}

async function onSave() {
  if (!editing.value) return
  saving.value = true
  try {
    await updateFlexibleSchedule(editing.value.id, {
      cron_expr: form.cron_expr,
      timezone: form.timezone || DEFAULT_TIMEZONE,
      enabled: !!form.enabled,
    })
    ElMessage.success(t('common.update_success'))
    formVisible.value = false
    await loadData()
  } catch (error) {
    logger.error('save flexible schedule failed', error)
  } finally {
    saving.value = false
  }
}

async function onToggle(row, enabled) {
  try {
    await updateFlexibleSchedule(row.id, { enabled })
    ElMessage.success(t('common.update_success'))
    await loadData()
  } catch (error) {
    logger.error('toggle failed', error)
  }
}

async function onRun(row) {
  try {
    await ElMessageBox.confirm(
      t('schedule.flexible_run_confirm', { name: row.name || row.handler }),
      t('schedule.run_now'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  runningId.value = row.id
  try {
    const res = await runFlexibleSchedule(row.id, !row.enabled)
    const updated = res.data?.flexible_schedule
    if (updated?.last_status === 'failed') {
      ElMessage.warning(t('schedule.run_failed'))
    } else if (updated?.last_status === 'skipped') {
      ElMessage.warning(t('schedule.flexible_status_skipped'))
    } else {
      ElMessage.success(t('schedule.run_success'))
    }
    await loadData()
    if (updated) openResult(updated)
  } catch (error) {
    logger.error('run failed', error)
  } finally {
    runningId.value = null
  }
}

onMounted(loadData)
</script>

<style scoped>
.flexible-card {
  margin-top: 16px;
}
.flexible-card__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}
.flexible-card__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.muted {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}
.schedule-result-pre {
  margin: 0;
  max-height: 320px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.6;
  font-family: Consolas, 'Courier New', ui-monospace, SFMono-Regular, Menlo, Monaco, monospace;
}
.schedule-result-pre--error {
  max-height: 160px;
  color: var(--el-color-danger);
}
</style>
