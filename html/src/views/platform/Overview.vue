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
        <el-card shadow="hover" class="stat-card clickable" @click="goMaintenance">
          <div class="stat-label">{{ $t('tenant.ops_maintenance') }}</div>
          <div class="stat-value" :class="(summary?.maintenance || 0) > 0 ? 'warn' : ''">{{ summary?.maintenance ?? 0 }}</div>
          <div class="stat-sub">{{ $t('tenant.ops_total') }} {{ summary?.total ?? 0 }}</div>
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

    <el-row :gutter="16" class="stat-row">
      <el-col :xs="12" :sm="8" :md="6" v-for="item in domainCards" :key="item.key">
        <el-card shadow="hover" class="stat-card clickable" @click="goQuery(item.query)">
          <div class="stat-label">{{ item.label }}</div>
          <div class="stat-value" :class="item.tone">{{ item.value }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="stat-row">
      <el-col :xs="12" :sm="8" :md="6" v-for="item in healthCards" :key="item.key">
        <el-card shadow="hover" class="stat-card clickable" @click="goQuery(item.query)">
          <div class="stat-label">{{ item.label }}</div>
          <div class="stat-value" :class="item.tone">{{ item.value }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="action-card" style="margin-bottom: 16px">
      <template #header>
        <span>{{ $t('platform.alerts_title') }}</span>
      </template>
      <el-space wrap>
        <el-tag :type="alerts?.tenant_ops?.configured ? 'success' : 'info'">
          {{ $t('tenant.alert_tenant_ops') }}:
          {{ alerts?.tenant_ops?.configured ? $t('tenant.alert_configured') : $t('tenant.alert_not_configured') }}
          <span v-if="alerts?.tenant_ops?.source"> ({{ alerts.tenant_ops.source }})</span>
        </el-tag>
        <el-tag :type="(alerts?.tenant_health?.webhook_configured || alerts?.tenant_health?.mail_configured) ? 'success' : 'info'">
          {{ $t('tenant.alert_tenant_health') }}:
          {{ (alerts?.tenant_health?.webhook_configured || alerts?.tenant_health?.mail_configured) ? $t('tenant.alert_configured') : $t('tenant.alert_not_configured') }}
        </el-tag>
        <el-tag :type="alerts?.queue?.configured ? 'success' : 'info'">
          {{ $t('tenant.alert_queue') }}:
          {{ alerts?.queue?.configured ? $t('tenant.alert_configured') : $t('tenant.alert_not_configured') }}
          <span v-if="alerts?.queue?.source"> ({{ alerts.queue.source }})</span>
        </el-tag>
      </el-space>
    </el-card>

    <el-card shadow="never" class="action-card">
      <template #header>
        <span>{{ $t('platform.overview_actions') }}</span>
        <el-button link type="primary" @click="load">{{ $t('common.refresh') }}</el-button>
      </template>
      <el-space wrap>
        <el-button type="primary" @click="goTenants('behind')">{{ $t('tenant.filter_schema_behind') }}</el-button>
        <el-button type="danger" plain @click="goTenants('failed')">{{ $t('tenant.filter_schema_failed') }}</el-button>
        <el-button @click="goQuery({ domain_status: 'verify_failed' })">{{ $t('tenant.filter_domain_failed') }}</el-button>
        <el-button type="danger" plain @click="goQuery({ health_status: 'fail' })">{{ $t('tenant.filter_health_fail') }}</el-button>
        <el-button :loading="inspectLoading" @click="runHealthInspect">{{ $t('tenant.health_inspect') }}</el-button>
        <el-button @click="goMaintenance">{{ $t('platform.filter_maintenance') }}</el-button>
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
import { ElMessage } from 'element-plus'
import { getPlatformOpsOverview, healthInspectPlatformTenants } from '@/api/platform'

const { t } = useI18n()
const router = useRouter()
const loading = ref(false)
const inspectLoading = ref(false)
const overview = ref(null)

const summary = computed(() => overview.value?.summary || null)
const queue = computed(() => overview.value?.queue || null)
const alerts = computed(() => overview.value?.alerts || null)

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

const domainCards = computed(() => {
  const s = summary.value || {}
  return [
    { key: 'unbound', label: t('tenant.domain_unbound'), value: s.domain_unbound || 0, tone: '', query: { domain_status: 'unbound' } },
    { key: 'pending', label: t('tenant.domain_pending'), value: s.domain_pending || 0, tone: 'warn', query: { domain_status: 'pending' } },
    { key: 'active', label: t('tenant.domain_active'), value: s.domain_active || 0, tone: 'ok', query: { domain_status: 'active' } },
    { key: 'verify_failed', label: t('tenant.domain_verify_failed'), value: s.domain_verify_failed || 0, tone: 'danger', query: { domain_status: 'verify_failed' } },
  ]
})

const healthCards = computed(() => {
  const s = summary.value || {}
  return [
    { key: 'ok', label: t('tenant.health_ok'), value: s.health_ok || 0, tone: 'ok', query: { health_status: 'ok' } },
    { key: 'warn', label: t('tenant.health_warn'), value: s.health_warn || 0, tone: 'warn', query: { health_status: 'warn' } },
    { key: 'fail', label: t('tenant.health_fail'), value: s.health_fail || 0, tone: 'danger', query: { health_status: 'fail' } },
    { key: 'unknown', label: t('tenant.health_unknown'), value: s.health_unknown || 0, tone: '', query: { health_status: 'unknown' } },
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

const goQuery = (query) => {
  router.push({ path: '/platform/tenants', query: query || {} })
}

const goMaintenance = () => {
  router.push({ path: '/platform/tenants', query: { maintenance: '1' } })
}

const runHealthInspect = async () => {
  inspectLoading.value = true
  try {
    const res = await healthInspectPlatformTenants({ alert: true })
    const report = res?.data?.report
    ElMessage.success(t('tenant.health_inspect_done', {
      ok: report?.ok ?? 0,
      warn: report?.warn ?? 0,
      fail: report?.fail ?? 0
    }))
    await load()
  } catch {
    // handled
  } finally {
    inspectLoading.value = false
  }
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
