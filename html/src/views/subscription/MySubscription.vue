<template>
  <div v-loading="loading" class="my-subscription">
    <h3 class="page-title">{{ $t('menu.subscription') }}</h3>

    <el-empty v-if="!loading && (!data || data.enabled === false)" :description="$t('subscription.not_available')" />

    <template v-else-if="data">
      <el-alert
        v-if="expiringSoon"
        class="block"
        type="warning"
        show-icon
        :closable="false"
        :title="remainingDays < 0 ? $t('subscription.expired_hint') : $t('subscription.expiring_hint', { days: remainingDays })"
      />

      <el-card shadow="never" class="block">
        <template #header>{{ $t('subscription.current_plan') }}</template>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('subscription.plan_name')">
            {{ data.plan?.name || '—' }}
            <code v-if="data.plan?.code" class="code">{{ data.plan.code }}</code>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('subscription.status')">
            <el-tag v-if="sub?.status" size="small" :type="sub.status === 'active' ? 'success' : 'info'">{{ sub.status }}</el-tag>
            <span v-else>—</span>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('subscription.starts_at')">{{ formatDate(sub?.starts_at) }}</el-descriptions-item>
          <el-descriptions-item :label="$t('subscription.ends_at')">
            {{ sub?.ends_at ? formatDate(sub.ends_at) : $t('subscription.no_expiry') }}
          </el-descriptions-item>
          <el-descriptions-item v-if="sub?.trial_ends_at" :label="$t('subscription.trial_ends_at')">
            {{ formatDate(sub.trial_ends_at) }}
          </el-descriptions-item>
          <el-descriptions-item v-if="data.plan?.description" :label="$t('subscription.description')" :span="2">
            {{ data.plan.description }}
          </el-descriptions-item>
        </el-descriptions>
        <p class="hint">{{ $t('subscription.upgrade_hint') }}</p>
      </el-card>

      <el-card v-if="limits.length" shadow="never" class="block">
        <template #header>{{ $t('subscription.quota_title') }}</template>
        <div v-for="l in limits" :key="l.key" class="quota-row">
          <div class="quota-head">
            <span>{{ l.name }}</span>
            <span class="muted">
              {{ l.unlimited ? $t('subscription.quota_unlimited', { used: l.used }) : $t('subscription.quota_usage', { used: l.used, limit: l.limit }) }}
            </span>
          </div>
          <el-progress
            v-if="!l.unlimited"
            :percentage="quotaPercent(l)"
            :status="quotaPercent(l) >= 100 ? 'exception' : quotaPercent(l) >= 80 ? 'warning' : undefined"
          />
        </div>
      </el-card>

      <el-card shadow="never" class="block">
        <template #header>{{ $t('subscription.features_title') }}</template>
        <el-table :data="features" row-key="key" size="small" border>
          <template #empty>{{ $t('subscription.features_empty') }}</template>
          <el-table-column prop="name" :label="$t('subscription.feature')" min-width="200" />
          <el-table-column :label="$t('subscription.kind')" width="120">
            <template #default="{ row }">
              <el-tag size="small" type="info">{{ kindLabel(row.kind) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="$t('subscription.availability')" width="160">
            <template #default="{ row }">
              <el-tag v-if="row.enabled" size="small" type="success">{{ $t('subscription.enabled') }}</el-tag>
              <el-tag v-else size="small" type="warning">{{ $t('subscription.locked') }}</el-tag>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { getMyPlan } from '@/api/subscription'

const { t } = useI18n()
const loading = ref(true)
const data = ref(null)

const features = computed(() => data.value?.features || [])
const limits = computed(() => data.value?.limits || [])
const sub = computed(() => data.value?.subscription || null)

const formatDate = (value) => {
  if (!value) return '—'
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? String(value) : d.toLocaleString()
}

const remainingDays = computed(() => {
  const end = sub.value?.ends_at ? new Date(sub.value.ends_at).getTime() : NaN
  if (Number.isNaN(end)) return null
  return Math.ceil((end - Date.now()) / 86400000)
})
const expiringSoon = computed(() => remainingDays.value !== null && remainingDays.value <= 7)

const quotaPercent = (l) => {
  if (l.unlimited) return 0
  // limit 0 (not unlimited) means no quota at all: show as full
  if (l.limit <= 0) return 100
  return Math.min(100, Math.round((l.used / l.limit) * 100))
}

const kindLabel = (kind) => {
  if (kind === 'capability') return t('subscription.kind_capability')
  if (kind === 'module') return t('subscription.kind_module')
  return t('subscription.kind_other')
}

onMounted(async () => {
  try {
    const res = await getMyPlan()
    data.value = res?.data || null
  } catch (e) {
    if (!e?.__handled) ElMessage.error(e?.message || t('common.operation_failed'))
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.my-subscription {
  padding: 16px;
}
.page-title {
  margin: 0 0 16px;
}
.block {
  margin-bottom: 16px;
}
.code {
  margin-left: 8px;
  font-size: 12px;
}
.hint {
  margin: 12px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.quota-row {
  margin-bottom: 12px;
}
.quota-head {
  display: flex;
  gap: 12px;
  margin-bottom: 4px;
}
.muted {
  color: var(--el-text-color-secondary);
}
</style>
