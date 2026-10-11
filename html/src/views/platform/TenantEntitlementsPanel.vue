<template>
  <div v-loading="loading" class="tenant-entitlements">
    <p class="hint">{{ $t('entitlement.tenant_hint') }}</p>

    <el-row :gutter="12" class="summary-row">
      <el-col :xs="24" :sm="12" :md="8">
        <el-card shadow="never" class="summary-card">
          <div class="stat-label">{{ $t('entitlement.summary_plan') }}</div>
          <div class="stat-value">{{ currentPlanLabel }}</div>
          <div v-if="summary?.ends_at" class="stat-sub">
            {{ $t('entitlement.summary_ends_at') }}: {{ summary.ends_at }}
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="8">
        <el-card shadow="never" class="summary-card">
          <div class="stat-label">{{ $t('entitlement.summary_features') }}</div>
          <div class="stat-value">{{ summary?.features_enabled ?? 0 }} / {{ summary?.features_total ?? 0 }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="8">
        <el-card shadow="never" class="summary-card">
          <div class="stat-label">{{ $t('entitlement.summary_limits') }}</div>
          <div v-if="!(summary?.limits || []).length" class="stat-value muted">—</div>
          <div v-else class="limit-list">
            <div v-for="lim in summary.limits" :key="lim.key" class="limit-item">
              <div class="limit-name">{{ lim.name || lim.key }}</div>
              <div class="stat-value small">{{ formatLimit(lim) }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <div class="toolbar">
      <el-select
        v-model="planCode"
        :placeholder="$t('entitlement.select_plan')"
        :disabled="isViewer"
        style="min-width: 200px"
      >
        <el-option v-for="p in plans" :key="p.code" :label="`${p.name} (${p.code})`" :value="p.code" />
      </el-select>
      <el-button type="primary" :disabled="isViewer || !planCode" :loading="loading" @click="onAssignPlan">
        {{ $t('entitlement.assign_plan') }}
      </el-button>
      <el-button :disabled="isViewer" :loading="loading" @click="onRecompute">
        {{ $t('entitlement.recompute') }}
      </el-button>
      <span class="hint">{{ $t('entitlement.version') }}: {{ data?.effective?.version ?? '—' }}</span>
    </div>
    <el-table :data="data?.catalog || []" row-key="key" size="small" border>
      <template #empty>{{ $t('entitlement.empty_catalog') }}</template>
      <el-table-column :label="$t('entitlement.feature')" min-width="240">
        <template #default="{ row }">
          <span>{{ row.name }}</span>
          <code class="key">{{ row.key }}</code>
          <el-tag v-if="row.type === 'limit'" size="small">{{ $t('entitlement.type_limit') }}</el-tag>
          <el-tag v-if="row.always_on" type="success" size="small">{{ $t('entitlement.always_on') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('entitlement.enabled')" width="140">
        <template #default="{ row }">
          <span v-if="row.type === 'limit'">{{ formatLimit(row) }}</span>
          <el-switch
            v-else
            :model-value="!!row.effective"
            :disabled="isViewer || row.type !== 'boolean' || !!row.always_on"
            :loading="savingKey === row.key"
            @change="(val) => onToggle(row, !!val)"
          />
        </template>
      </el-table-column>
      <el-table-column :label="$t('entitlement.override')" width="120">
        <template #default="{ row }">
          <el-button
            v-if="row.has_override"
            link
            type="primary"
            :disabled="isViewer"
            :loading="savingKey === row.key"
            @click="onClearOverride(row)"
          >
            {{ $t('entitlement.clear_override') }}
          </el-button>
          <span v-else>—</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  assignPlatformTenantPlan,
  deletePlatformTenantEntitlement,
  getPlatformPlans,
  getPlatformTenantEntitlements,
  recomputePlatformTenantEntitlements,
  setPlatformTenantEntitlement
} from '@/api/platform'
import { getPlatformAdmin } from '@/utils/platformRequest'

const props = defineProps({
  tenantId: { type: Number, required: true }
})

const { t } = useI18n()
const isViewer = computed(() => getPlatformAdmin()?.role === 'viewer')

const loading = ref(false)
const data = ref(null)
const plans = ref([])
const planCode = ref()
const savingKey = ref('')

const reportError = (e) => {
  if (!e?.__handled) ElMessage.error(e?.message || t('common.operation_failed'))
}

const load = async () => {
  loading.value = true
  try {
    const [entRes, planRes] = await Promise.all([
      getPlatformTenantEntitlements(props.tenantId),
      getPlatformPlans()
    ])
    const ent = entRes?.data || null
    data.value = ent
    planCode.value = ent?.plan?.code || ent?.summary?.plan_code || ent?.effective?.plan_code || undefined
    plans.value = planRes?.data?.list || planRes?.list || []
  } catch (e) {
    reportError(e)
  } finally {
    loading.value = false
  }
}

const summary = computed(() => data.value?.summary || null)

const currentPlanLabel = computed(() => {
  const code = summary.value?.plan_code || data.value?.plan?.code || data.value?.effective?.plan_code
  if (!code) return t('tenant.plan_none')
  const name = summary.value?.plan_name || data.value?.plan?.name
  return name ? `${name} (${code})` : code
})

const formatLimit = (row) => {
  if (row?.unlimited) return `${row.used ?? 0} / ∞`
  return `${row?.used ?? 0} / ${row?.limit ?? 0}`
}

const planLabel = (code) => {
  if (!code) return t('entitlement.assign_none')
  const p = plans.value.find((x) => x.code === code)
  return p ? `${p.name} (${p.code})` : code
}

// Assigning cancels the active subscription and applies the new plan immediately: confirm first.
const onAssignPlan = async () => {
  if (!planCode.value) return
  const currentCode =
    data.value?.plan?.code || data.value?.summary?.plan_code || data.value?.effective?.plan_code || undefined
  const to = planLabel(planCode.value)
  const message =
    currentCode && currentCode === planCode.value
      ? t('entitlement.assign_confirm_same', { to })
      : t('entitlement.assign_confirm_change', { from: planLabel(currentCode), to })
  try {
    await ElMessageBox.confirm(message, t('entitlement.assign_confirm_title'), { type: 'warning' })
  } catch {
    return
  }
  loading.value = true
  try {
    await assignPlatformTenantPlan(props.tenantId, { plan_code: planCode.value })
    ElMessage.success(t('entitlement.plan_assigned'))
    await load()
  } catch (e) {
    reportError(e)
  } finally {
    loading.value = false
  }
}

const onToggle = async (row, enabled) => {
  savingKey.value = row.key
  try {
    await setPlatformTenantEntitlement(props.tenantId, row.key, {
      enabled,
      reason: 'platform_ui'
    })
    ElMessage.success(t('entitlement.override_saved'))
    await load()
  } catch (e) {
    reportError(e)
  } finally {
    savingKey.value = ''
  }
}

const onClearOverride = async (row) => {
  savingKey.value = row.key
  try {
    await deletePlatformTenantEntitlement(props.tenantId, row.key)
    ElMessage.success(t('entitlement.override_cleared'))
    await load()
  } catch (e) {
    reportError(e)
  } finally {
    savingKey.value = ''
  }
}

const onRecompute = async () => {
  loading.value = true
  try {
    await recomputePlatformTenantEntitlements(props.tenantId)
    ElMessage.success(t('entitlement.recomputed'))
    await load()
  } catch (e) {
    reportError(e)
  } finally {
    loading.value = false
  }
}

watch(() => props.tenantId, load)
onMounted(load)
</script>

<style scoped>
.hint {
  margin: 0 0 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.summary-row {
  margin-bottom: 12px;
}
.summary-card {
  margin-bottom: 8px;
}
.stat-label {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.stat-value {
  margin-top: 4px;
  font-size: 18px;
  font-weight: 600;
}
.stat-value.small {
  font-size: 15px;
}
.stat-value.muted {
  color: var(--el-text-color-secondary);
  font-weight: 400;
}
.stat-sub {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.limit-list {
  margin-top: 4px;
}
.limit-item + .limit-item {
  margin-top: 6px;
}
.limit-name {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.key {
  margin: 0 8px;
  font-size: 12px;
}
</style>
