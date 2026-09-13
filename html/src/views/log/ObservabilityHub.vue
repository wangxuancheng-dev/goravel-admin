<template>
  <div class="list-page observability-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ $t('menu.observability') }}</span>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane v-if="tabAccess.queue" :label="$t('observability.queue_tab')" name="queue">
          <el-alert type="info" :closable="false" class="queue-hint" :title="$t('observability.queue_dashboard_hint')" />
          <el-card shadow="never" class="queue-alert-card" style="margin-bottom: 12px">
            <template #header>{{ $t('observability.queue_alert_title') }}</template>
            <el-alert type="warning" :closable="false" :title="$t('observability.queue_alert_hint')" style="margin-bottom: 12px" />
            <div class="queue-alert-row" style="display:flex;flex-wrap:wrap;gap:8px;align-items:center">
              <el-tag :type="queueAlert.configured ? 'success' : 'info'">
                {{ queueAlert.configured
                  ? $t('observability.queue_alert_configured', { source: queueAlert.source || '-' })
                  : $t('observability.queue_alert_not_configured') }}
              </el-tag>
              <span v-if="queueAlert.configured" class="queue-alert-meta">
                {{ $t('observability.queue_alert_threshold', { n: queueAlert.threshold || 100 }) }}
                · {{ $t('observability.queue_alert_url', { url: queueAlert.url_masked || '***' }) }}
              </span>
              <el-button
                v-if="canTestQueueAlert"
                :loading="queueAlertTesting"
                :disabled="!queueAlert.configured"
                @click="onTestQueueAlert"
              >
                {{ $t('observability.queue_alert_test') }}
              </el-button>
            </div>
          </el-card>
          <div class="search-row queue-toolbar">
            <span class="queue-default-label">{{ $t('observability.queue_default') }}: <code>{{ queueDashboard.default_connection || '-' }}</code></span>
            <el-button type="primary" :loading="queueLoading" @click="loadQueue">{{ $t('common.refresh') }}</el-button>
          </div>

          <el-card v-if="currentQueuePanel" shadow="never" class="queue-panel-card">
            <template #header>
              <div class="queue-panel-header">
                <div>
                  <strong>{{ currentQueuePanel.connection }}</strong>
                  <el-tag class="queue-panel-kind" size="small" :type="queueKindTag(currentQueuePanel.kind)">
                    {{ $t(`observability.queue_kind_${currentQueuePanel.kind}`) }}
                  </el-tag>
                </div>
              </div>
            </template>

            <el-descriptions :column="4" border size="small" class="queue-panel-desc">
              <el-descriptions-item :label="$t('observability.queue_driver')">{{ currentQueuePanel.driver_raw || '-' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('observability.queue_redis_client')">{{ currentQueuePanel.redis_client || '-' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('observability.queue_consumer_group')">
                {{ currentQueuePanel.kind === 'redis_stream' ? (currentQueuePanel.consumer_group || '-') : '-' }}
              </el-descriptions-item>
              <el-descriptions-item :label="$t('observability.queue_default_queue')">{{ currentQueuePanel.default_queue || '-' }}</el-descriptions-item>
            </el-descriptions>

            <el-table v-if="currentQueueRows.length" :data="currentQueueRows" size="small" border>
              <el-table-column prop="name" :label="$t('observability.queue_name')" width="180">
                <template #default="{ row: q }">
                  <code>{{ q.name }}</code>
                </template>
              </el-table-column>
              <el-table-column prop="pending" :label="$t('observability.metric_pending')" width="100" />
              <el-table-column prop="reserved" :label="$t('observability.metric_reserved')" width="100" />
              <el-table-column prop="delayed" :label="$t('observability.metric_delayed')" width="100" />
              <el-table-column prop="failed" :label="$t('observability.metric_failed')" width="100" />
              <el-table-column prop="total" :label="$t('observability.metric_total')" width="100" />
              <el-table-column
                v-if="currentQueuePanel.kind === 'redis_stream'"
                prop="stream_total"
                :label="$t('observability.metric_stream_total')"
                width="150"
              >
                <template #default="{ row: q }">
                  <span>{{ q.stream_total == null ? '—' : q.stream_total }}</span>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-else :description="$t('observability.queue_no_data')" />
            <el-text v-if="currentQueuePanel.fetch_error" type="danger" class="queue-fetch-err">{{ currentQueuePanel.fetch_error }}</el-text>
          </el-card>
          <el-empty v-else :description="$t('observability.queue_panel_missing')" />
        </el-tab-pane>

        <el-tab-pane v-if="tabAccess.trace" :label="$t('observability.trace_tab')" name="trace">
          <div class="search-row">
            <el-input v-model="traceQuery.trace_id" :placeholder="$t('observability.trace_id_placeholder')" clearable />
            <el-button type="primary" :loading="traceLoading" @click="loadTrace">{{ $t('common.search') }}</el-button>
          </div>

          <el-empty v-if="!traceLoading && !traceData.trace_id" :description="$t('observability.trace_empty')" />
          <div v-else-if="traceData.trace_id">
            <el-descriptions border :column="2">
              <el-descriptions-item :label="$t('log.trace_id')">{{ traceData.trace_id }}</el-descriptions-item>
              <el-descriptions-item :label="$t('observability.trace_request')">{{ traceData.request?.method }} {{ traceData.request?.path }}</el-descriptions-item>
            </el-descriptions>

            <el-divider>{{ $t('observability.trace_operations') }}</el-divider>
            <el-table :data="traceData.operations || []" size="small" border>
              <el-table-column prop="created_at" :label="$t('log.time')" width="180" />
              <el-table-column prop="method" :label="$t('log.method')" width="90" />
              <el-table-column prop="path" :label="$t('log.path')" min-width="260" />
              <el-table-column prop="duration" :label="$t('observability.duration_ms')" width="120" />
              <el-table-column prop="title" :label="$t('log.title')" min-width="180" />
            </el-table>

            <el-divider>{{ $t('observability.trace_exceptions') }}</el-divider>
            <el-table :data="traceData.exceptions || []" size="small" border>
              <el-table-column prop="created_at" :label="$t('log.time')" width="180" />
              <el-table-column prop="level" :label="$t('log.level')" width="100" />
              <el-table-column prop="module" :label="$t('log.module')" width="140" />
              <el-table-column prop="message" :label="$t('log.message')" min-width="320" />
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="tabAccess.audit" :label="$t('observability.audit_tab')" name="audit">
          <div class="search-row">
            <el-input v-model="auditQuery.trace_id" :placeholder="$t('observability.trace_id_placeholder')" clearable />
            <el-input v-model="auditQuery.keyword" :placeholder="$t('observability.keyword_placeholder')" clearable />
            <el-button type="primary" :loading="auditLoading" @click="loadAudit">{{ $t('common.search') }}</el-button>
          </div>

          <el-table :data="auditData.list" size="small" border>
            <el-table-column prop="time" :label="$t('log.time')" width="180" />
            <el-table-column prop="type" :label="$t('observability.event_type')" width="110" />
            <el-table-column prop="trace_id" :label="$t('log.trace_id')" width="220" />
            <el-table-column prop="module" :label="$t('log.module')" width="130" />
            <el-table-column prop="message" :label="$t('log.message')" min-width="340" />
          </el-table>

          <div class="pager">
            <el-pagination
              v-model:current-page="auditQuery.page"
              v-model:page-size="auditQuery.page_size"
              :total="auditData.total"
              layout="total, prev, pager, next"
              @current-change="loadAudit"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="tabAccess.slowSql" :label="$t('observability.slow_sql_tab')" name="slowSql">
          <div class="search-row">
            <span class="field-label">{{ $t('observability.slow_sql_hours') }}</span>
            <el-input-number v-model="slowSqlQuery.hours" :min="1" :max="168" />
            <span class="field-label">{{ $t('observability.slow_sql_min_duration_ms') }}</span>
            <el-input-number v-model="slowSqlQuery.min_duration_ms" :min="1" :max="10000" />
            <span class="field-label">{{ $t('observability.slow_sql_limit') }}</span>
            <el-input-number v-model="slowSqlQuery.limit" :min="1" :max="100" />
            <el-button type="primary" :loading="slowSqlLoading" @click="loadSlowSql">{{ $t('common.search') }}</el-button>
          </div>

          <el-table :data="slowSqlData" size="small" border>
            <el-table-column prop="max_duration_ms" :label="$t('observability.max_ms')" width="120">
              <template #default="{ row }">
                <span>{{ Number(row.max_duration_ms || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="avg_duration_ms" :label="$t('observability.avg_ms')" width="120">
              <template #default="{ row }">
                <span>{{ Number(row.avg_duration_ms || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="count" :label="$t('common.count')" width="90" />
            <el-table-column prop="normalized_sql" :label="$t('observability.normalized_sql')" min-width="420">
              <template #default="{ row }">
                <el-tooltip
                  v-if="row.normalized_sql"
                  :content="row.normalized_sql"
                  placement="top-start"
                  effect="dark"
                  :show-after="200"
                >
                  <code class="sql-ellipsis">{{ row.normalized_sql }}</code>
                </el-tooltip>
                <code v-else>-</code>
              </template>
            </el-table-column>
            <el-table-column prop="last_seen_at" :label="$t('observability.last_seen')" width="180" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane v-if="tabAccess.apiPerformance" :label="$t('observability.api_performance_tab')" name="apiPerformance">
          <div class="search-row">
            <span class="field-label">{{ $t('observability.api_perf_hours') }}</span>
            <el-input-number v-model="apiPerfQuery.hours" :min="1" :max="168" />
            <span class="field-label">{{ $t('observability.api_perf_limit') }}</span>
            <el-input-number v-model="apiPerfQuery.limit" :min="1" :max="100" />
            <el-button type="primary" :loading="apiPerfLoading" @click="loadApiPerformance">{{ $t('common.search') }}</el-button>
          </div>

          <el-divider>{{ $t('observability.api_perf_slow_top') }}</el-divider>
          <el-table :data="apiPerfData.slow_top || []" size="small" border>
            <el-table-column prop="method" :label="$t('log.method')" width="90" />
            <el-table-column prop="route_template" :label="$t('observability.api_route_template')" min-width="280" />
            <el-table-column prop="p95_duration_ms" :label="$t('observability.p95_ms')" width="120">
              <template #default="{ row }">
                <span>{{ Number(row.p95_duration_ms || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="p99_duration_ms" :label="$t('observability.p99_ms')" width="120">
              <template #default="{ row }">
                <span>{{ Number(row.p99_duration_ms || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="avg_duration_ms" :label="$t('observability.avg_ms')" width="120">
              <template #default="{ row }">
                <span>{{ Number(row.avg_duration_ms || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="max_duration_ms" :label="$t('observability.max_ms')" width="120">
              <template #default="{ row }">
                <span>{{ Number(row.max_duration_ms || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="count" :label="$t('common.count')" width="100" />
            <el-table-column :label="$t('common.operation')" width="140">
              <template #default="{ row }">
                <el-button link type="primary" @click="loadApiPerformanceTraces(row)">{{ $t('observability.view_traces') }}</el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-divider>{{ $t('observability.api_perf_error_top') }}</el-divider>
          <el-table :data="apiPerfData.error_top || []" size="small" border>
            <el-table-column prop="method" :label="$t('log.method')" width="90" />
            <el-table-column prop="route_template" :label="$t('observability.api_route_template')" min-width="280" />
            <el-table-column prop="error_rate" :label="$t('observability.error_rate')" width="120">
              <template #default="{ row }">
                <span>{{ (Number(row.error_rate || 0) * 100).toFixed(2) }}%</span>
              </template>
            </el-table-column>
            <el-table-column prop="error_count" :label="$t('observability.error_count')" width="120" />
            <el-table-column prop="count" :label="$t('common.count')" width="100" />
            <el-table-column :label="$t('common.operation')" width="140">
              <template #default="{ row }">
                <el-button link type="primary" @click="loadApiPerformanceTraces(row)">{{ $t('observability.view_traces') }}</el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-divider>{{ $t('observability.api_perf_qps_top') }}</el-divider>
          <el-table :data="apiPerfData.qps_top || []" size="small" border>
            <el-table-column prop="method" :label="$t('log.method')" width="90" />
            <el-table-column prop="route_template" :label="$t('observability.api_route_template')" min-width="280" />
            <el-table-column prop="qps" :label="$t('observability.qps')" width="120">
              <template #default="{ row }">
                <span>{{ Number(row.qps || 0).toFixed(4) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="count" :label="$t('common.count')" width="100" />
            <el-table-column :label="$t('common.operation')" width="140">
              <template #default="{ row }">
                <el-button link type="primary" @click="loadApiPerformanceTraces(row)">{{ $t('observability.view_traces') }}</el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-divider>{{ $t('observability.api_perf_trace_drilldown') }}</el-divider>
          <el-alert
            v-if="selectedApiEndpoint.route_template"
            type="info"
            :closable="false"
            class="queue-hint"
            :title="`${selectedApiEndpoint.method} ${selectedApiEndpoint.route_template}`"
          />
          <el-table v-loading="apiPerfTraceLoading" :data="apiPerfTraceData" size="small" border>
            <el-table-column prop="occurred_at" :label="$t('log.time')" width="180" />
            <el-table-column prop="trace_id" :label="$t('log.trace_id')" min-width="250">
              <template #default="{ row }">
                <el-button link type="primary" @click="jumpToTrace(row.trace_id)">{{ row.trace_id || '-' }}</el-button>
              </template>
            </el-table-column>
            <el-table-column prop="status_code" :label="$t('log.status_code')" width="110" />
            <el-table-column prop="duration_ms" :label="$t('observability.duration_ms')" width="130" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane v-if="tabAccess.pprof && showPprofTab" :label="$t('observability.pprof_tab')" name="pprof">
          <el-alert v-if="pprofStatus.token_required" type="warning" :closable="false" class="queue-hint" :title="$t('observability.pprof_hint')" />
          <div v-if="pprofStatus.token_required" class="search-row pprof-row">
            <el-input
              v-model="pprofForm.token"
              :placeholder="$t('observability.pprof_token_placeholder')"
              type="password"
              clearable
            />
            <el-button type="primary" :loading="pprofVerifying" @click="handleVerifyPprofToken">
              {{ $t('observability.pprof_verify') }}
            </el-button>
            <el-tag :type="pprofVerified ? 'success' : 'info'">
              {{ pprofVerified ? $t('observability.pprof_verified') : $t('observability.pprof_unverified') }}
            </el-tag>
          </div>

          <div class="pprof-actions">
            <el-button type="primary" @click="openPprofPath('/debug/pprof/')">Index</el-button>
            <el-button @click="openPprofPath('/debug/pprof/heap')">Heap</el-button>
            <el-button @click="openPprofPath('/debug/pprof/goroutine')">Goroutine</el-button>
            <el-button @click="openPprofPath('/debug/pprof/allocs')">Allocs</el-button>
            <el-button @click="openPprofPath('/debug/pprof/profile')">CPU Profile</el-button>
            <el-button @click="openPprofPath('/debug/pprof/trace')">Trace</el-button>
            <el-button @click="openPprofPath('/debug/pprof/runtime')">Runtime</el-button>
          </div>

          <el-divider>{{ $t('observability.cpu_hotspots_title') }}</el-divider>
          <div class="search-row pprof-row">
            <span>{{ $t('observability.pprof_sample_seconds') }}</span>
            <el-input-number v-model="cpuHotspotForm.seconds" :min="1" :max="120" />
            <span>{{ $t('observability.pprof_top_n') }}</span>
            <el-input-number v-model="cpuHotspotForm.top_n" :min="1" :max="100" />
            <el-button type="primary" :loading="cpuHotspotLoading" :disabled="memoryHotspotLoading" @click="loadCpuHotspots">
              {{ $t('observability.pprof_start_sampling') }}
            </el-button>
            <el-tag v-if="cpuHotspotLoading" type="warning">
              {{ $t('observability.pprof_sampling_in_progress', { seconds: cpuSamplingRemaining }) }}
            </el-tag>
          </div>
          <el-table :data="cpuHotspotData" size="small" border>
            <el-table-column type="index" :label="$t('table.sort')" width="70" />
            <el-table-column prop="function" :label="$t('observability.pprof_function')" min-width="320" />
            <el-table-column prop="flat_ms" :label="$t('observability.pprof_cpu_self_ms')" width="150" />
            <el-table-column prop="flat_percent" :label="$t('observability.pprof_self_percent')" width="120" />
            <el-table-column prop="cum_ms" :label="$t('observability.pprof_cpu_cum_ms')" width="150" />
            <el-table-column prop="cum_percent" :label="$t('observability.pprof_cum_percent')" width="120" />
          </el-table>

          <el-divider>{{ $t('observability.memory_hotspots_title') }}</el-divider>
          <div class="search-row pprof-row">
            <span>{{ $t('observability.pprof_top_n') }}</span>
            <el-input-number v-model="memoryHotspotForm.top_n" :min="1" :max="100" />
            <el-button type="primary" :loading="memoryHotspotLoading" :disabled="cpuHotspotLoading" @click="loadMemoryHotspots">
              {{ $t('common.refresh') }}
            </el-button>
          </div>
          <el-table :data="memoryHotspotData" size="small" border>
            <el-table-column type="index" :label="$t('table.sort')" width="70" />
            <el-table-column prop="function" :label="$t('observability.pprof_function')" min-width="320" />
            <el-table-column prop="flat_bytes" :label="$t('observability.pprof_mem_flat_bytes')" width="170" />
            <el-table-column prop="flat_percent" :label="$t('observability.pprof_self_percent')" width="120" />
            <el-table-column prop="cum_bytes" :label="$t('observability.pprof_mem_cum_bytes')" width="170" />
            <el-table-column prop="cum_percent" :label="$t('observability.pprof_cum_percent')" width="120" />
          </el-table>
        </el-tab-pane>

      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../../store/user'
import { getApiPerformanceOverview, getApiPerformanceTraces, getAuditTimeline, getPprofCpuHotspots, getPprofMemoryHotspots, getPprofStatus, getQueueDashboard, testQueueAlert, getSlowSqlTop, getTraceAggregate, verifyPprofToken } from '../../api/observability'

const activeTab = ref('queue')
const userStore = useUserStore()
const { t } = useI18n()

const traceLoading = ref(false)
const traceQuery = reactive({ trace_id: '' })
const traceData = reactive({})

const slowSqlLoading = ref(false)
const slowSqlQuery = reactive({ hours: 24, min_duration_ms: 200, limit: 20 })
const slowSqlData = ref([])
const apiPerfLoading = ref(false)
const apiPerfTraceLoading = ref(false)
const apiPerfQuery = reactive({ hours: 24, limit: 20 })
const apiPerfData = reactive({ slow_top: [], error_top: [], qps_top: [] })
const apiPerfTraceData = ref([])
const selectedApiEndpoint = reactive({ method: '', route_template: '' })

const auditLoading = ref(false)
const auditQuery = reactive({ trace_id: '', keyword: '', page: 1, page_size: 20 })
const auditData = reactive({ list: [], total: 0 })

const queueLoading = ref(false)
const queueDashboard = reactive({ default_connection: '', connections: [] })
const queueAlert = reactive({ configured: false, source: '', url_masked: '', threshold: 100 })
const queueAlertTesting = ref(false)
const pprofStatus = reactive({ enabled: false, is_developer: false, token_required: false })
const pprofForm = reactive({ token: '' })
const pprofVerifying = ref(false)
const pprofVerified = ref(false)
const pprofStatusLoaded = ref(false)
const cpuHotspotForm = reactive({ seconds: 3, top_n: 15 })
const memoryHotspotForm = reactive({ top_n: 25 })
const cpuHotspotLoading = ref(false)
const memoryHotspotLoading = ref(false)
const cpuHotspotData = ref([])
const memoryHotspotData = ref([])
const cpuSamplingRemaining = ref(0)
let cpuSamplingTimer = null

const hasAnyPermission = (slugs = []) => {
  if (!Array.isArray(slugs) || slugs.length === 0) return false
  return slugs.some(slug => userStore.hasPermission(slug))
}

const canTestQueueAlert = computed(() => hasAnyPermission(['observability.queue_alert_test']))

const tabAccess = reactive({
  queue: hasAnyPermission(['observability.queue_dashboard']),
  trace: hasAnyPermission(['observability.trace']),
  audit: hasAnyPermission(['observability.audit_timeline']),
  slowSql: hasAnyPermission(['observability.slow_sql_top']),
  apiPerformance: hasAnyPermission([
    'observability.api_performance_overview',
    'observability.api_performance_traces',
    'observability.api_performance'
  ]),
  pprof: !!(userStore.config?.isDeveloperAdmin && userStore.config?.pprofEnabled)
})

const showPprofTab = computed(() => {
  const cfg = userStore.config || {}
  return pprofStatusLoaded.value &&
    pprofStatus.enabled &&
    pprofStatus.is_developer &&
    !!cfg.isDeveloperAdmin &&
    !!cfg.pprofEnabled
})

const visibleTabs = computed(() => {
  const tabs = []
  if (tabAccess.queue) tabs.push('queue')
  if (tabAccess.trace) tabs.push('trace')
  if (tabAccess.audit) tabs.push('audit')
  if (tabAccess.slowSql) tabs.push('slowSql')
  if (tabAccess.apiPerformance) tabs.push('apiPerformance')
  if (tabAccess.pprof && showPprofTab.value) tabs.push('pprof')
  return tabs
})

const currentQueuePanel = computed(() => {
  const rows = queueDashboard.connections || []
  if (!rows.length) return null
  return rows.find(item => item?.is_default) || rows.find(item => item?.connection === queueDashboard.default_connection) || rows[0]
})

const queuesToRows = (queues) => {
  if (!queues || typeof queues !== 'object') return []
  return Object.keys(queues).map((name) => {
    const s = queues[name] || {}
    return {
      name,
      pending: s.pending ?? 0,
      reserved: s.reserved ?? 0,
      delayed: s.delayed != null ? s.delayed : 0,
      failed: s.failed ?? 0,
      total: s.total ?? 0,
      stream_total: s.stream_total ?? null
    }
  })
}

const currentQueueRows = computed(() => {
  if (!currentQueuePanel.value?.queues) return []
  return queuesToRows(currentQueuePanel.value.queues)
})

const queueKindTag = (kind) => {
  const m = { database: 'success', redis_list: 'primary', redis_stream: 'warning', sync: 'info', other: 'danger' }
  return m[kind] || 'info'
}

const handleViewRequestError = (error) => {
  if (error?.__handled) return
  ElMessage.error(error?.response?.data?.message || error?.message || t('error.default'))
}

const loadQueue = async () => {
  queueLoading.value = true
  try {
    const res = await getQueueDashboard()
    queueDashboard.default_connection = res.data?.default_connection || ''
    queueDashboard.connections = res.data?.connections || []
    const alert = res.data?.queue_alert || {}
    queueAlert.configured = !!alert.configured
    queueAlert.source = alert.source || ''
    queueAlert.url_masked = alert.url_masked || ''
    queueAlert.threshold = alert.threshold || 100
  } catch (error) {
    handleViewRequestError(error)
  } finally {
    queueLoading.value = false
  }
}

const onTestQueueAlert = async () => {
  queueAlertTesting.value = true
  try {
    await testQueueAlert()
    ElMessage.success(t('observability.queue_alert_test_ok'))
    await loadQueue()
  } catch (error) {
    handleViewRequestError(error)
  } finally {
    queueAlertTesting.value = false
  }
}

const loadPprofStatus = async () => {
  try {
    const res = await getPprofStatus()
    pprofStatus.enabled = !!res.data?.enabled
    pprofStatus.is_developer = !!res.data?.is_developer
    pprofStatus.token_required = !!res.data?.token_required
    pprofStatusLoaded.value = true
  } catch (error) {
    if (error?.response?.status === 403) {
      tabAccess.pprof = false
      pprofStatusLoaded.value = true
      pprofStatus.enabled = false
      pprofStatus.is_developer = false
      pprofStatus.token_required = false
      return
    }
    pprofStatusLoaded.value = true
    pprofStatus.enabled = false
    pprofStatus.is_developer = false
    pprofStatus.token_required = false
    handleViewRequestError(error)
  }
}

const handleVerifyPprofToken = async () => {
  if (!showPprofTab.value) return
  if (pprofStatus.token_required && !pprofForm.token) {
    ElMessage.warning(t('observability.pprof_token_placeholder'))
    return
  }

  pprofVerifying.value = true
  try {
    await verifyPprofToken({ token: pprofForm.token || '' })
    pprofVerified.value = true
    ElMessage.success(t('observability.pprof_verified'))
  } catch (error) {
    pprofVerified.value = false
    const retryAfter = error?.response?.data?.retry_after
    if (error?.response?.status === 429 && retryAfter) {
      ElMessage.error(t('observability.pprof_rate_limited', { seconds: retryAfter }))
      return
    }
    ElMessage.error(error.response?.data?.message || error.message || t('observability.pprof_unverified'))
  } finally {
    pprofVerifying.value = false
  }
}

const ensurePprofSamplingReady = () => {
  if (!showPprofTab.value) return false
  if (pprofStatus.token_required) {
    if (!pprofForm.token) {
      ElMessage.warning(t('observability.pprof_token_placeholder'))
      return false
    }
    if (!pprofVerified.value) {
      ElMessage.warning(t('observability.pprof_verify'))
      return false
    }
  }
  return true
}

const handlePprofApiError = (error) => {
  const retryAfter = error?.response?.data?.retry_after
  if (error?.response?.status === 429 && retryAfter) {
    ElMessage.error(t('observability.pprof_rate_limited', { seconds: retryAfter }))
    return
  }
  ElMessage.error(error?.response?.data?.message || error?.message || t('observability.pprof_unverified'))
}

const toCpuHotspotRows = (rows) => {
  return (rows || []).map(item => ({
    function: item.function || '-',
    flat_ms: `${((item.flat_value || 0) / 1e6).toFixed(2)} ms`,
    flat_percent: `${Number(item.flat_percent || 0).toFixed(2)}%`,
    cum_ms: `${((item.cum_value || 0) / 1e6).toFixed(2)} ms`,
    cum_percent: `${Number(item.cum_percent || 0).toFixed(2)}%`
  }))
}

const toMemoryHotspotRows = (rows) => {
  const formatBytes = (bytes) => {
    const value = Number(bytes || 0)
    if (value >= 1024 * 1024 * 1024) return `${(value / (1024 * 1024 * 1024)).toFixed(2)} GB`
    if (value >= 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(2)} MB`
    if (value >= 1024) return `${(value / 1024).toFixed(2)} KB`
    return `${value} B`
  }
  return (rows || []).map(item => ({
    function: item.function || '-',
    flat_bytes: formatBytes(item.flat_value),
    flat_percent: `${Number(item.flat_percent || 0).toFixed(2)}%`,
    cum_bytes: formatBytes(item.cum_value),
    cum_percent: `${Number(item.cum_percent || 0).toFixed(2)}%`
  }))
}

const loadCpuHotspots = async () => {
  if (!ensurePprofSamplingReady()) return
  if (cpuHotspotLoading.value || memoryHotspotLoading.value) return
  cpuHotspotLoading.value = true
  cpuSamplingRemaining.value = Number(cpuHotspotForm.seconds || 0)
  if (cpuSamplingTimer) clearInterval(cpuSamplingTimer)
  cpuSamplingTimer = setInterval(() => {
    if (cpuSamplingRemaining.value > 0) {
      cpuSamplingRemaining.value -= 1
    } else {
      clearInterval(cpuSamplingTimer)
      cpuSamplingTimer = null
    }
  }, 1000)
  try {
    const res = await getPprofCpuHotspots({
      token: pprofForm.token || '',
      seconds: cpuHotspotForm.seconds,
      top_n: cpuHotspotForm.top_n
    })
    cpuHotspotData.value = toCpuHotspotRows(res.data?.list || [])
  } catch (error) {
    handlePprofApiError(error)
  } finally {
    cpuHotspotLoading.value = false
    cpuSamplingRemaining.value = 0
    if (cpuSamplingTimer) {
      clearInterval(cpuSamplingTimer)
      cpuSamplingTimer = null
    }
  }
}

const loadMemoryHotspots = async () => {
  if (!ensurePprofSamplingReady()) return
  if (cpuHotspotLoading.value || memoryHotspotLoading.value) return
  memoryHotspotLoading.value = true
  try {
    const res = await getPprofMemoryHotspots({
      token: pprofForm.token || '',
      top_n: memoryHotspotForm.top_n
    })
    memoryHotspotData.value = toMemoryHotspotRows(res.data?.list || [])
  } catch (error) {
    handlePprofApiError(error)
  } finally {
    memoryHotspotLoading.value = false
  }
}

const openPprofPath = (path) => {
  if (!showPprofTab.value) return

  if (pprofStatus.token_required) {
    if (!pprofForm.token) {
      ElMessage.warning(t('observability.pprof_token_placeholder'))
      return
    }
    if (!pprofVerified.value) {
      ElMessage.warning(t('observability.pprof_verify'))
      return
    }
  }

  const apiBaseURL = import.meta.env.VITE_API_BASE_URL
  const backendOrigin = apiBaseURL
    ? apiBaseURL.replace(/\/+$/, '')
    : window.location.origin
  const url = new URL(path, backendOrigin)
  if (pprofForm.token) {
    url.searchParams.set('token', pprofForm.token)
  }
  window.open(url.toString(), '_blank', 'noopener,noreferrer')
}

const loadTrace = async () => {
  if (!traceQuery.trace_id) {
    ElMessage.warning('trace_id is required')
    return
  }
  traceLoading.value = true
  try {
    const res = await getTraceAggregate(traceQuery)
    Object.keys(traceData).forEach(key => delete traceData[key])
    Object.assign(traceData, res.data || {})
  } catch (error) {
    handleViewRequestError(error)
  } finally {
    traceLoading.value = false
  }
}

const loadSlowSql = async () => {
  slowSqlLoading.value = true
  try {
    const res = await getSlowSqlTop(slowSqlQuery)
    slowSqlData.value = res.data?.list || []
  } catch (error) {
    handleViewRequestError(error)
  } finally {
    slowSqlLoading.value = false
  }
}

const loadApiPerformance = async () => {
  apiPerfLoading.value = true
  try {
    const res = await getApiPerformanceOverview(apiPerfQuery)
    apiPerfData.slow_top = res.data?.slow_top || []
    apiPerfData.error_top = res.data?.error_top || []
    apiPerfData.qps_top = res.data?.qps_top || []
  } catch (error) {
    if (error?.response?.status === 403) {
      tabAccess.apiPerformance = false
      return
    }
    handleViewRequestError(error)
  } finally {
    apiPerfLoading.value = false
  }
}

const loadApiPerformanceTraces = async (row) => {
  selectedApiEndpoint.method = row?.method || ''
  selectedApiEndpoint.route_template = row?.route_template || ''
  apiPerfTraceLoading.value = true
  try {
    const res = await getApiPerformanceTraces({
      method: selectedApiEndpoint.method,
      route_template: selectedApiEndpoint.route_template,
      hours: apiPerfQuery.hours,
      limit: apiPerfQuery.limit
    })
    apiPerfTraceData.value = res.data?.list || []
  } catch (error) {
    if (error?.response?.status === 403) {
      tabAccess.apiPerformance = false
      return
    }
    handleViewRequestError(error)
  } finally {
    apiPerfTraceLoading.value = false
  }
}

const jumpToTrace = (traceID) => {
  if (!traceID) return
  traceQuery.trace_id = traceID
  activeTab.value = 'trace'
  loadTrace()
}

const loadAudit = async () => {
  auditLoading.value = true
  try {
    const res = await getAuditTimeline(auditQuery)
    auditData.list = res.data?.list || []
    auditData.total = res.data?.total || 0
  } catch (error) {
    handleViewRequestError(error)
  } finally {
    auditLoading.value = false
  }
}

onMounted(() => {
  if (tabAccess.slowSql) loadSlowSql()
  if (tabAccess.apiPerformance) loadApiPerformance()
  if (tabAccess.audit) loadAudit()
  if (tabAccess.queue) loadQueue()
  if (tabAccess.pprof) loadPprofStatus()
})

watch(
  visibleTabs,
  (tabs) => {
    if (!tabs.length) return
    if (!tabs.includes(activeTab.value)) {
      activeTab.value = tabs[0]
    }
  },
  { immediate: true }
)

onUnmounted(() => {
  if (cpuSamplingTimer) {
    clearInterval(cpuSamplingTimer)
    cpuSamplingTimer = null
  }
})
</script>

<style scoped>
.search-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.field-label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.sql-ellipsis {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.queue-hint {
  margin-bottom: 12px;
}

.queue-toolbar {
  align-items: center;
}

.queue-default-label {
  flex: 1;
}

.queue-panel-card {
  border-radius: 8px;
}

.queue-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.queue-panel-kind {
  margin-left: 8px;
}

.queue-panel-desc {
  margin-bottom: 12px;
}

.queue-fetch-err {
  display: block;
  margin-top: 8px;
}

.pprof-row {
  align-items: center;
}

.pprof-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
</style>
