<template>
  <div class="platform-overview" v-loading="loading">
    <el-alert
      type="warning"
      :closable="false"
      show-icon
      class="deploy-alert"
      :title="$t('platform.deploy_tip_title')"
      :description="$t('platform.deploy_tip_desc')"
    />

    <el-row :gutter="16" class="stat-row">
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-label">{{ $t('platform.app_version') }}</div>
          <div class="stat-value">{{ overview?.app_version || '—' }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-label">{{ $t('tenant.schema_expected') }}</div>
          <div class="stat-value">{{ overview?.expected_migration_count ?? summary?.expected_migration_count ?? '—' }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-label">{{ $t('tenant.ops_queue_pending') }}</div>
          <div class="stat-value">{{ queue?.pending ?? '—' }}</div>
          <div class="stat-sub">{{ queue?.connection || '—' }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-label">{{ $t('tenant.ops_total') }}</div>
          <div class="stat-value">{{ summary?.total ?? 0 }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="stat-row">
      <el-col :xs="12" :sm="8" :md="4" v-for="item in schemaCards" :key="item.key">
        <el-card shadow="hover" class="stat-card clickable" @click="goTenants(item.key)">
          <div class="stat-label">{{ item.label }}</div>
          <div class="stat-value" :class="item.tone">{{ item.value }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="action-card">
      <template #header>
        <span>{{ $t('platform.overview_actions') }}</span>
        <el-button link type="primary" @click="load">{{ $t('common.refresh') }}</el-button>
      </template>
      <el-space wrap>
        <el-button type="primary" @click="goTenants('behind')">{{ $t('tenant.filter_schema_behind') }}</el-button>
        <el-button type="danger" plain @click="goTenants('failed')">{{ $t('tenant.filter_schema_failed') }}</el-button>
        <el-button @click="goTenants('unknown')">{{ $t('tenant.filter_schema_unknown') }}</el-button>
        <el-button @click="$router.push('/platform/tenants')">{{ $t('menu.tenant') }}</el-button>
        <el-button @click="$router.push('/platform/tenant-op-logs')">{{ $t('menu.tenant_op_log') }}</el-button>
      </el-space>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { getPlatformOpsOverview } from '@/api/platform'

const { t } = useI18n()
const router = useRouter()
const loading = ref(false)
const overview = ref(null)

const summary = computed(() => overview.value?.summary || null)
const queue = computed(() => overview.value?.queue || null)

const schemaCards = computed(() => {
  const s = summary.value || {}
  return [
    { key: 'aligned', label: t('tenant.schema_aligned'), value: s.schema_aligned || 0, tone: 'ok' },
    { key: 'behind', label: t('tenant.schema_behind'), value: s.schema_behind || 0, tone: 'warn' },
    { key: 'failed', label: t('tenant.schema_failed'), value: s.schema_failed || 0, tone: 'danger' },
    { key: 'running', label: t('tenant.schema_running'), value: s.schema_running || 0, tone: 'warn' },
    { key: 'unknown', label: t('tenant.schema_unknown'), value: s.schema_unknown || 0, tone: '' },
  ]
})

const load = async () => {
  loading.value = true
  try {
    const res = await getPlatformOpsOverview()
    overview.value = res?.data || null
  } catch {
    overview.value = null
  } finally {
    loading.value = false
  }
}

const goTenants = (schemaStatus) => {
  router.push({ path: '/platform/tenants', query: schemaStatus ? { schema_status: schemaStatus } : {} })
}

onMounted(load)
</script>

<style scoped>
.platform-overview {
  max-width: 1100px;
}
.deploy-alert {
  margin-bottom: 16px;
}
.stat-row {
  margin-bottom: 16px;
}
.stat-card {
  margin-bottom: 12px;
}
.stat-card.clickable {
  cursor: pointer;
}
.stat-label {
  color: #64748b;
  font-size: 13px;
  margin-bottom: 6px;
}
.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #0f172a;
}
.stat-value.ok {
  color: #16a34a;
}
.stat-value.warn {
  color: #d97706;
}
.stat-value.danger {
  color: #dc2626;
}
.stat-sub {
  margin-top: 4px;
  font-size: 12px;
  color: #94a3b8;
}
.action-card :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
