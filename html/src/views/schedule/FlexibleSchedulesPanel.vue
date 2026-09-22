<template>
  <el-card shadow="never" class="flexible-card">
    <template #header>
      <div class="flexible-card__header">
        <div>
          <h3 class="flexible-card__title">{{ t('schedule.flexible_title') }}</h3>
          <p class="flexible-card__hint">{{ t('schedule.flexible_hint') }}</p>
        </div>
        <div class="flexible-card__actions">
          <el-button :loading="loading" @click="loadData">{{ t('common.refresh') }}</el-button>
          <el-button
            type="primary"
            :disabled="getButtonState('flexible_schedule.create').disabled"
            @click="openCreate"
          >
            {{ t('common.add') }}
          </el-button>
        </div>
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
      <el-table-column :label="t('schedule.flexible_tenant')" width="120">
        <template #default="{ row }">
          {{ row.tenant_id > 0 ? (row.tenant_code || row.tenant_id) : t('schedule.flexible_tenant_all') }}
        </template>
      </el-table-column>
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
      <el-table-column :label="t('schedule.flexible_next_runs')" min-width="170">
        <template #default="{ row }">
          <div v-if="row.next_runs?.length" class="muted">
            <div v-for="r in row.next_runs.slice(0, 2)" :key="r">{{ formatDateTimeDisplay(r) }}</div>
          </div>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.operation')" width="220" fixed="right" align="center">
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
            type="danger"
            link
            :disabled="getButtonState('flexible_schedule.delete').disabled"
            @click="onDelete(row)"
          >
            {{ t('common.delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="formVisible" :title="editing ? t('common.edit') : t('common.add')" width="560px">
      <el-form :model="form" label-width="110px">
        <el-form-item :label="t('schedule.flexible_name')" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="t('schedule.flexible_handler')" required>
          <el-select v-model="form.handler" style="width: 100%">
            <el-option
              v-for="h in handlers"
              :key="h.key"
              :label="`${h.name} (${h.key})`"
              :value="h.key"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('schedule.cron')" required>
          <el-input v-model="form.cron_expr" placeholder="0 20 * * *" />
          <div class="muted">{{ t('schedule.flexible_cron_tip') }}</div>
        </el-form-item>
        <el-form-item :label="t('schedule.flexible_timezone')" required>
          <el-input v-model="form.timezone" placeholder="UTC" />
        </el-form-item>
        <el-form-item :label="t('schedule.flexible_tenant')">
          <el-input-number v-model="form.tenant_id" :min="0" style="width: 100%" />
          <div class="muted">{{ t('schedule.flexible_tenant_tip') }}</div>
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
  </el-card>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  createFlexibleSchedule,
  deleteFlexibleSchedule,
  getFlexibleScheduleHandlers,
  getFlexibleScheduleList,
  previewFlexibleSchedule,
  runFlexibleSchedule,
  updateFlexibleSchedule,
} from '@/api/schedule'
import { usePermission } from '@/composables/usePermission'
import { describeCron } from '@/utils/cronLabel'
import { formatDateTimeDisplay } from '@/utils/dateUtils'
import { logger } from '@/utils/logger'

const { t } = useI18n()
const { getButtonState } = usePermission()

const loading = ref(false)
const saving = ref(false)
const rows = ref([])
const handlers = ref([])
const runningId = ref(null)
const formVisible = ref(false)
const editing = ref(null)
const previewRuns = ref([])
const form = reactive({
  name: '',
  handler: 'schedule_test_log',
  cron_expr: '*/5 * * * *',
  timezone: 'UTC',
  tenant_id: 0,
  enabled: true,
})

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

async function loadData() {
  loading.value = true
  try {
    const [listRes, handlerRes] = await Promise.all([
      getFlexibleScheduleList(),
      getFlexibleScheduleHandlers(),
    ])
    rows.value = listRes.data?.list || []
    handlers.value = handlerRes.data?.list || []
  } catch (error) {
    logger.error('Failed to load flexible schedules:', error)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, {
    name: '',
    handler: handlers.value[0]?.key || 'schedule_test_log',
    cron_expr: '*/5 * * * *',
    timezone: 'UTC',
    tenant_id: 0,
    enabled: true,
  })
  previewRuns.value = []
  formVisible.value = true
}

function openEdit(row) {
  editing.value = row
  Object.assign(form, {
    name: row.name,
    handler: row.handler,
    cron_expr: row.cron_expr,
    timezone: row.timezone || 'UTC',
    tenant_id: row.tenant_id || 0,
    enabled: !!row.enabled,
  })
  previewRuns.value = row.next_runs || []
  formVisible.value = true
}

async function onPreview() {
  try {
    const res = await previewFlexibleSchedule({
      cron_expr: form.cron_expr,
      timezone: form.timezone || 'UTC',
      count: 5,
    })
    previewRuns.value = res.data?.next_runs || []
  } catch (error) {
    logger.error('preview failed', error)
  }
}

async function onSave() {
  saving.value = true
  try {
    const payload = {
      name: form.name,
      handler: form.handler,
      cron_expr: form.cron_expr,
      timezone: form.timezone || 'UTC',
      tenant_id: Number(form.tenant_id) || 0,
      enabled: !!form.enabled,
    }
    if (editing.value) {
      await updateFlexibleSchedule(editing.value.id, payload)
      ElMessage.success(t('common.update_success'))
    } else {
      await createFlexibleSchedule(payload)
      ElMessage.success(t('common.create_success'))
    }
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
  } catch (error) {
    logger.error('run failed', error)
  } finally {
    runningId.value = null
  }
}

async function onDelete(row) {
  try {
    await ElMessageBox.confirm(
      t('schedule.flexible_delete_confirm', { name: row.name || row.handler }),
      t('common.delete'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await deleteFlexibleSchedule(row.id)
    ElMessage.success(t('common.delete_success'))
    await loadData()
  } catch (error) {
    logger.error('delete failed', error)
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
  align-items: flex-start;
}
.flexible-card__title {
  margin: 0;
  font-size: 16px;
}
.flexible-card__hint,
.muted {
  margin: 4px 0 0;
  color: var(--text-color-secondary, var(--el-text-color-secondary));
  font-size: 12px;
  line-height: 1.4;
}
.flexible-card__actions {
  display: flex;
  gap: 8px;
}
</style>
