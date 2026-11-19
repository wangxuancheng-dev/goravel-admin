<template>
  <div class="monitor-page">
    <!-- 系统告警提示 -->
    <el-alert
      v-if="systemInfo.alerts && systemInfo.alerts.length > 0"
      v-for="(alert, index) in systemInfo.alerts"
      :key="index"
      :title="getAlertMessage(alert)"
      :type="alert.level === 'high' ? 'error' : 'warning'"
      :closable="false"
      show-icon
      style="margin-bottom: 20px"
    />
    
    <!-- 核心资源监控：CPU、内存、磁盘 -->
    <el-row :gutter="20">
      <!-- CPU信息 -->
      <el-col :span="8">
        <el-card class="monitor-card cpu-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon cpu-icon"><Cpu /></el-icon>
                <span>{{ $t('monitor.cpu') }}</span>
              </div>
              <el-button size="small" :icon="Refresh" circle :disabled="refreshing" :loading="refreshing" @click="loadData" />
            </div>
          </template>
          <div class="monitor-content">
            <div class="monitor-item usage-item">
              <div class="usage-header">
                <span class="label">
                  {{ $t('monitor.cpu_usage') }}
                  <span class="usage-scope-hint">
                    · {{ $t('monitor.cpu_usage_system', { cores: systemInfo.cpu?.cores || 0 }) }}
                  </span>
                </span>
                <span class="percent-value">{{ formatPercent(systemInfo.cpu?.percent || 0) }}</span>
              </div>
              <el-progress
                :percentage="formatPercentForProgress(systemInfo.cpu?.percent || 0)"
                :color="getProgressColor(systemInfo.cpu?.percent || 0)"
                :stroke-width="20"
                :show-text="false"
                class="usage-progress"
              />
            </div>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.cpu_model') }}</span>
                <span class="info-value" style="font-size: 12px">{{ systemInfo.cpu?.model || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.cpu_logical_cores') }}</span>
                <span class="info-value highlight">{{ systemInfo.cpu?.cores || 0 }}</span>
              </div>
              <div class="info-item" v-if="systemInfo.cpu?.physical_cores">
                <span class="info-label">{{ $t('monitor.cpu_physical_cores') }}</span>
                <span class="info-value">{{ systemInfo.cpu?.physical_cores || 0 }}</span>
              </div>
            </div>
            <!-- CPU使用率走势图 -->
            <div ref="cpuChartRef" style="height: 180px; margin-top: 15px;"></div>
          </div>
        </el-card>
      </el-col>

      <!-- 内存信息 -->
      <el-col :span="8">
        <el-card class="monitor-card memory-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon memory-icon"><DataBoard /></el-icon>
                <span>{{ $t('monitor.memory') }}</span>
              </div>
              <el-button size="small" :icon="Refresh" circle :disabled="refreshing" :loading="refreshing" @click="loadData" />
            </div>
          </template>
          <div class="monitor-content">
            <div class="monitor-item usage-item">
              <div class="usage-header">
                <span class="label">{{ $t('monitor.memory_usage') }}</span>
                <span class="percent-value">{{ formatPercent(systemInfo.memory?.percent || 0) }}</span>
              </div>
              <el-progress
                :percentage="formatPercentForProgress(systemInfo.memory?.percent || 0)"
                :color="getProgressColor(systemInfo.memory?.percent || 0)"
                :stroke-width="20"
                :show-text="false"
                class="usage-progress"
              />
            </div>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.memory_total') }}</span>
                <span class="info-value">{{ formatBytes(systemInfo.memory?.total || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.memory_used') }}</span>
                <span class="info-value highlight">{{ formatBytes(systemInfo.memory?.used || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.memory_available') }}</span>
                <span class="info-value">{{ formatBytes(systemInfo.memory?.available || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.memory_free') }}</span>
                <span class="info-value">{{ formatBytes(systemInfo.memory?.free || 0) }}</span>
              </div>
            </div>
            <!-- 内存使用率走势图 -->
            <div ref="memoryChartRef" style="height: 180px; margin-top: 15px;"></div>
          </div>
        </el-card>
      </el-col>

      <!-- 磁盘信息 -->
      <el-col :span="8">
        <el-card class="monitor-card disk-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon disk-icon"><FolderOpened /></el-icon>
                <span>{{ $t('monitor.disk') }}</span>
              </div>
              <el-button size="small" :icon="Refresh" circle :disabled="refreshing" :loading="refreshing" @click="loadData" />
            </div>
          </template>
          <div class="monitor-content">
            <div class="monitor-item usage-item">
              <div class="usage-header">
                <span class="label">{{ $t('monitor.disk_usage') }}</span>
                <span class="percent-value">{{ formatPercent(systemInfo.disk?.percent || 0) }}</span>
              </div>
              <el-progress
                :percentage="formatPercentForProgress(systemInfo.disk?.percent || 0)"
                :color="getProgressColor(systemInfo.disk?.percent || 0)"
                :stroke-width="20"
                :show-text="false"
                class="usage-progress"
              />
            </div>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.disk_total') }}</span>
                <span class="info-value">{{ formatBytes(systemInfo.disk?.total || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.disk_used') }}</span>
                <span class="info-value highlight">{{ formatBytes(systemInfo.disk?.used || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.disk_free') }}</span>
                <span class="info-value">{{ formatBytes(systemInfo.disk?.free || 0) }}</span>
              </div>
              <div class="info-item full-width">
                <span class="info-label">{{ $t('monitor.disk_path') }}</span>
                <span class="info-value" style="font-size: 12px">{{ systemInfo.disk?.path || '-' }}</span>
              </div>
            </div>
            <!-- 磁盘使用率走势图 -->
            <div ref="diskChartRef" style="height: 180px; margin-top: 15px;"></div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 资源使用率统计图 -->
    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="12">
        <el-card class="monitor-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon"><TrendCharts /></el-icon>
                <span>{{ $t('monitor.resource_usage_chart') }}</span>
              </div>
            </div>
          </template>
          <div class="monitor-content">
            <div ref="resourcePieChartRef" style="height: 300px;"></div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card class="monitor-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon"><Connection /></el-icon>
                <span>{{ $t('monitor.network_speed_chart') }}</span>
              </div>
            </div>
          </template>
          <div class="monitor-content">
            <div ref="networkChartRef" style="height: 300px;"></div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 系统负载与FD（仅 Linux） -->
    <el-row v-if="isLinux" :gutter="20" style="margin-top: 20px">
      <!-- 系统负载（仅Linux） -->
      <el-col :span="12">
        <el-card class="monitor-card load-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon load-icon"><TrendCharts /></el-icon>
                <span>{{ $t('monitor.load') }}</span>
              </div>
              <el-button size="small" :icon="Refresh" circle :disabled="refreshing" :loading="refreshing" @click="loadData" />
            </div>
          </template>
          <div class="monitor-content">
            <div class="load-display">
              <div class="load-value">
                <span class="load-number">{{ formatLoad(systemInfo.load?.load1 || 0) }}</span>
                <span class="load-percent">({{ formatPercent(systemInfo.load?.load1_percent || 0) }})</span>
              </div>
              <div class="load-label">{{ $t('monitor.load_current') }}</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 文件描述符信息（仅Linux） -->
      <el-col :span="12">
        <el-card class="monitor-card fd-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon fd-icon"><Document /></el-icon>
                <span>{{ $t('monitor.file_descriptors') }}</span>
              </div>
              <el-button size="small" :icon="Refresh" circle :disabled="refreshing" :loading="refreshing" @click="loadData" />
            </div>
          </template>
          <div class="monitor-content">
            <div class="monitor-item usage-item">
              <div class="usage-header">
                <span class="label">{{ $t('monitor.fd_usage') }}</span>
                <span class="percent-value">{{ formatPercent(systemInfo.file_descriptors?.percent || 0) }}</span>
              </div>
              <el-progress
                :percentage="formatPercentForProgress(systemInfo.file_descriptors?.percent || 0)"
                :color="getProgressColor(systemInfo.file_descriptors?.percent || 0)"
                :stroke-width="20"
                :show-text="false"
                class="usage-progress"
              />
            </div>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.fd_used') }}</span>
                <span class="info-value highlight">{{ formatNumber(systemInfo.file_descriptors?.used || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.fd_free') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.file_descriptors?.free || 0) }}</span>
              </div>
              <div class="info-item full-width">
                <span class="info-label">{{ $t('monitor.fd_max') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.file_descriptors?.max || 0) }}</span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 运行时和系统信息 -->
    <el-row :gutter="20" style="margin-top: 20px">
      <!-- 运行时信息 -->
      <el-col :span="12">
        <el-card class="monitor-card runtime-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon runtime-icon"><Operation /></el-icon>
                <span>{{ $t('monitor.runtime') }}</span>
              </div>
              <el-button size="small" :icon="Refresh" circle :disabled="refreshing" :loading="refreshing" @click="loadData" />
            </div>
          </template>
          <div class="monitor-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.goroutines') }}</span>
                <span class="info-value highlight">{{ formatNumber(systemInfo.runtime?.goroutines || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.total_processes') }}</span>
                <span class="info-value highlight">{{ formatNumber(systemInfo.runtime?.total_processes || 0) }}</span>
              </div>
              <div class="info-item" v-if="systemInfo.runtime?.num_cpu">
                <span class="info-label">{{ $t('monitor.num_cpu') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.runtime.num_cpu) }}</span>
              </div>
              <div class="info-item" v-if="systemInfo.runtime?.gomaxprocs">
                <span class="info-label">{{ $t('monitor.gomaxprocs') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.runtime.gomaxprocs) }}</span>
              </div>
            </div>
            <div v-if="systemInfo.runtime?.memory" style="margin-top: 15px; padding-top: 15px; border-top: 1px solid #ebeef5">
              <div class="info-label" style="margin-bottom: 10px; font-weight: 600">{{ $t('monitor.memory_stats') }}:</div>
              <div class="info-grid">
                <div class="info-item">
                  <span class="info-label">{{ $t('monitor.mem_alloc') }}</span>
                  <span class="info-value">{{ formatBytes(systemInfo.runtime.memory.alloc || 0) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ $t('monitor.mem_total_alloc') }}</span>
                  <span class="info-value">{{ formatBytes(systemInfo.runtime.memory.total_alloc || 0) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ $t('monitor.mem_sys') }}</span>
                  <span class="info-value">{{ formatBytes(systemInfo.runtime.memory.sys || 0) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ $t('monitor.mem_heap_alloc') }}</span>
                  <span class="info-value highlight">{{ formatBytes(systemInfo.runtime.memory.heap_alloc || 0) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ $t('monitor.mem_heap_sys') }}</span>
                  <span class="info-value">{{ formatBytes(systemInfo.runtime.memory.heap_sys || 0) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ $t('monitor.mem_heap_objects') }}</span>
                  <span class="info-value">{{ formatNumber(systemInfo.runtime.memory.heap_objects || 0) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ $t('monitor.mem_num_gc') }}</span>
                  <span class="info-value">{{ formatNumber(systemInfo.runtime.memory.num_gc || 0) }}</span>
                </div>
                <div class="info-item" v-if="systemInfo.runtime.memory.pause_total_ns">
                  <span class="info-label">{{ $t('monitor.mem_pause_total') }}</span>
                  <span class="info-value">{{ formatDuration(systemInfo.runtime.memory.pause_total_ns / 1000000) }}</span>
                </div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 系统信息 -->
      <el-col :span="12">
        <el-card class="monitor-card system-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon system-icon"><Monitor /></el-icon>
                <span>{{ $t('monitor.system_info') }}</span>
              </div>
              <el-button size="small" :icon="Refresh" circle :disabled="refreshing" :loading="refreshing" @click="loadData" />
            </div>
          </template>
          <div class="monitor-content">
            <div class="info-grid">
              <div class="info-item full-width">
                <span class="info-label">{{ $t('monitor.hostname') }}</span>
                <span class="info-value highlight">{{ systemInfo.system?.hostname || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.os') }}</span>
                <span class="info-value highlight">{{ systemInfo.system?.os || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.arch') }}</span>
                <span class="info-value highlight">{{ systemInfo.system?.arch || '-' }}</span>
              </div>
              <div class="info-item full-width">
                <span class="info-label">{{ $t('monitor.go_version') }}</span>
                <span class="info-value">{{ systemInfo.system?.go_version || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor_runtime.health_status') }}</span>
                <span class="info-value highlight">{{ systemInfo.health?.status || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor_runtime.alert_count') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.health?.alert_count || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor_runtime.app_env') }}</span>
                <span class="info-value highlight">{{ systemInfo.app?.env || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor_runtime.app_debug') }}</span>
                <span class="info-value">{{ formatBool(systemInfo.app?.debug) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor_runtime.queue_connection') }}</span>
                <span class="info-value">{{ systemInfo.app?.queue_connection || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor_runtime.cache_store') }}</span>
                <span class="info-value">{{ systemInfo.app?.cache_store || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor_runtime.ws_online_admins') }}</span>
                <span class="info-value highlight">{{ systemInfo.websocket?.online_admins ?? 0 }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor_runtime.ws_connections') }}</span>
                <span class="info-value highlight">{{ systemInfo.websocket?.connections ?? 0 }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor_runtime.mem_alloc') }}</span>
                <span class="info-value">{{ formatBytes(systemInfo.runtime?.memory?.alloc || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor_runtime.mem_heap_alloc') }}</span>
                <span class="info-value highlight">{{ formatBytes(systemInfo.runtime?.memory?.heap_alloc || 0) }}</span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 进程监控信息 -->
    <el-row v-if="systemInfo.processes" :gutter="20" style="margin-top: 20px">
      <el-col :span="24">
        <el-card class="monitor-card processes-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon processes-icon"><Operation /></el-icon>
                <span>{{ $t('monitor.processes') }}</span>
              </div>
              <el-button size="small" :icon="Refresh" circle :disabled="refreshing" :loading="refreshing" @click="loadData" />
            </div>
          </template>
          <div class="monitor-content">
            <el-row :gutter="20">
              <!-- MySQL 进程 -->
              <el-col :span="6" v-if="systemInfo.processes.mysql">
                <div class="process-card">
                  <div class="process-header">
                    <el-tag :type="getProcessStatusType(systemInfo.processes.mysql.status)" size="large">
                      {{ $t('monitor.process_mysql') }}
                    </el-tag>
                    <el-tag v-if="systemInfo.processes.mysql.type" :type="systemInfo.processes.mysql.type === 'remote' ? 'warning' : 'success'" size="small">
                      {{ systemInfo.processes.mysql.type === 'remote' ? $t('monitor.process_remote') : $t('monitor.process_local') }}
                    </el-tag>
                  </div>
                  <div class="process-content">
                    <div class="process-item">
                      <span class="process-label">{{ $t('monitor.process_status') }}:</span>
                      <span class="process-value" :class="getProcessStatusType(systemInfo.processes.mysql.status) === 'success' ? 'highlight' : ''">
                        {{ systemInfo.processes.mysql.status === 'running' ? $t('monitor.process_running') : 
                           systemInfo.processes.mysql.status === 'sleep' ? $t('monitor.process_running') :
                           systemInfo.processes.mysql.status === 'connected' ? $t('monitor.process_connected') :
                           systemInfo.processes.mysql.status === 'not_found' ? $t('monitor.process_not_found') :
                           systemInfo.processes.mysql.status === 'disconnected' ? $t('monitor.process_disconnected') :
                           systemInfo.processes.mysql.status }}
                      </span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.pid && systemInfo.processes.mysql.pid > 0">
                      <span class="process-label">{{ $t('monitor.process_pid') }}:</span>
                      <span class="process-value">{{ systemInfo.processes.mysql.pid }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.cpu !== undefined && systemInfo.processes.mysql.cpu > 0">
                      <span class="process-label">{{ $t('monitor.process_cpu') }}:</span>
                      <span class="process-value highlight">{{ formatPercent(systemInfo.processes.mysql.cpu) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.memory && systemInfo.processes.mysql.memory > 0">
                      <span class="process-label">{{ $t('monitor.process_memory') }}:</span>
                      <span class="process-value highlight">{{ formatBytes(systemInfo.processes.mysql.memory) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.version">
                      <span class="process-label">{{ $t('monitor.process_version') }}:</span>
                      <span class="process-value">{{ systemInfo.processes.mysql.version }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.host">
                      <span class="process-label">{{ $t('monitor.process_host') }}:</span>
                      <span class="process-value">{{ systemInfo.processes.mysql.host }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.threads !== undefined">
                      <span class="process-label">{{ $t('monitor.process_threads') }}:</span>
                      <span class="process-value">{{ formatNumber(systemInfo.processes.mysql.threads || 0) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.connections !== undefined">
                      <span class="process-label">{{ $t('monitor.process_connections') }}:</span>
                      <span class="process-value highlight">
                        {{ formatNumber(systemInfo.processes.mysql.connections || 0) }}
                        <span v-if="systemInfo.processes.mysql.max_connections" class="connections-info">
                          / {{ formatNumber(systemInfo.processes.mysql.max_connections) }}
                          <span class="connections-percent" :class="getConnectionPercentClass(systemInfo.processes.mysql.connections, systemInfo.processes.mysql.max_connections)">
                            ({{ formatPercent((systemInfo.processes.mysql.connections || 0) / systemInfo.processes.mysql.max_connections * 100) }})
                          </span>
                        </span>
                      </span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.queries !== undefined && systemInfo.processes.mysql.queries > 0">
                      <span class="process-label">{{ $t('monitor.process_queries') }}:</span>
                      <span class="process-value">{{ formatNumber(systemInfo.processes.mysql.queries) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.uptime !== undefined && systemInfo.processes.mysql.uptime > 0">
                      <span class="process-label">{{ $t('monitor.process_uptime') }}:</span>
                      <span class="process-value">{{ formatUptime(systemInfo.processes.mysql.uptime) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.slow_queries !== undefined && systemInfo.processes.mysql.slow_queries > 0">
                      <span class="process-label">{{ $t('monitor.process_slow_queries') }}:</span>
                      <span class="process-value warning">{{ formatNumber(systemInfo.processes.mysql.slow_queries) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.table_locks_waited !== undefined && systemInfo.processes.mysql.table_locks_waited > 0">
                      <span class="process-label">{{ $t('monitor.process_table_locks') }}:</span>
                      <span class="process-value warning">{{ formatNumber(systemInfo.processes.mysql.table_locks_waited) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.innodb_row_lock_waits !== undefined && systemInfo.processes.mysql.innodb_row_lock_waits > 0">
                      <span class="process-label">{{ $t('monitor.process_row_locks') }}:</span>
                      <span class="process-value warning">{{ formatNumber(systemInfo.processes.mysql.innodb_row_lock_waits) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.threads_running !== undefined">
                      <span class="process-label">{{ $t('monitor.process_threads_running') }}:</span>
                      <span class="process-value">{{ formatNumber(systemInfo.processes.mysql.threads_running) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.mysql.buffer_pool_size !== undefined && systemInfo.processes.mysql.buffer_pool_size > 0">
                      <span class="process-label">{{ $t('monitor.process_buffer_pool') }}:</span>
                      <span class="process-value">{{ formatBytes(systemInfo.processes.mysql.buffer_pool_size) }}</span>
                    </div>
                  </div>
                </div>
              </el-col>

              <!-- PostgreSQL 进程 -->
              <el-col :span="6" v-if="systemInfo.processes.postgresql">
                <div class="process-card">
                  <div class="process-header">
                    <el-tag :type="getProcessStatusType(systemInfo.processes.postgresql.status)" size="large">
                      {{ $t('monitor.process_postgresql') }}
                    </el-tag>
                    <el-tag v-if="systemInfo.processes.postgresql.type" :type="systemInfo.processes.postgresql.type === 'remote' ? 'warning' : 'success'" size="small">
                      {{ systemInfo.processes.postgresql.type === 'remote' ? $t('monitor.process_remote') : $t('monitor.process_local') }}
                    </el-tag>
                  </div>
                  <div class="process-content">
                    <div class="process-item">
                      <span class="process-label">{{ $t('monitor.process_status') }}:</span>
                      <span class="process-value" :class="getProcessStatusType(systemInfo.processes.postgresql.status) === 'success' ? 'highlight' : ''">
                        {{ systemInfo.processes.postgresql.status === 'running' ? $t('monitor.process_running') : 
                           systemInfo.processes.postgresql.status === 'sleep' ? $t('monitor.process_running') :
                           systemInfo.processes.postgresql.status === 'connected' ? $t('monitor.process_connected') :
                           systemInfo.processes.postgresql.status === 'not_found' ? $t('monitor.process_not_found') :
                           systemInfo.processes.postgresql.status === 'disconnected' ? $t('monitor.process_disconnected') :
                           systemInfo.processes.postgresql.status }}
                      </span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.pid && systemInfo.processes.postgresql.pid > 0">
                      <span class="process-label">{{ $t('monitor.process_pid') }}:</span>
                      <span class="process-value">{{ systemInfo.processes.postgresql.pid }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.cpu !== undefined && systemInfo.processes.postgresql.cpu > 0">
                      <span class="process-label">{{ $t('monitor.process_cpu') }}:</span>
                      <span class="process-value highlight">{{ formatPercent(systemInfo.processes.postgresql.cpu) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.memory && systemInfo.processes.postgresql.memory > 0">
                      <span class="process-label">{{ $t('monitor.process_memory') }}:</span>
                      <span class="process-value highlight">{{ formatBytes(systemInfo.processes.postgresql.memory) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.version">
                      <span class="process-label">{{ $t('monitor.process_version') }}:</span>
                      <span class="process-value">{{ systemInfo.processes.postgresql.version }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.host">
                      <span class="process-label">{{ $t('monitor.process_host') }}:</span>
                      <span class="process-value">{{ systemInfo.processes.postgresql.host }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.connections !== undefined">
                      <span class="process-label">{{ $t('monitor.process_connections') }}:</span>
                      <span class="process-value highlight">
                        {{ formatNumber(systemInfo.processes.postgresql.connections || 0) }}
                        <span v-if="systemInfo.processes.postgresql.max_connections" class="connections-info">
                          / {{ formatNumber(systemInfo.processes.postgresql.max_connections) }}
                          <span class="connections-percent" :class="getConnectionPercentClass(systemInfo.processes.postgresql.connections, systemInfo.processes.postgresql.max_connections)">
                            ({{ formatPercent((systemInfo.processes.postgresql.connections || 0) / systemInfo.processes.postgresql.max_connections * 100) }})
                          </span>
                        </span>
                      </span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.active_connections !== undefined">
                      <span class="process-label">{{ $t('monitor.postgresql_active_connections') }}:</span>
                      <span class="process-value highlight">{{ formatNumber(systemInfo.processes.postgresql.active_connections || 0) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.idle_connections !== undefined">
                      <span class="process-label">{{ $t('monitor.postgresql_idle_connections') }}:</span>
                      <span class="process-value">{{ formatNumber(systemInfo.processes.postgresql.idle_connections || 0) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.queries !== undefined && systemInfo.processes.postgresql.queries > 0">
                      <span class="process-label">{{ $t('monitor.process_queries') }}:</span>
                      <span class="process-value">{{ formatNumber(systemInfo.processes.postgresql.queries) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.uptime !== undefined && systemInfo.processes.postgresql.uptime > 0">
                      <span class="process-label">{{ $t('monitor.process_uptime') }}:</span>
                      <span class="process-value">{{ formatUptime(systemInfo.processes.postgresql.uptime) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.postgresql.database_size !== undefined && systemInfo.processes.postgresql.database_size > 0">
                      <span class="process-label">{{ $t('monitor.postgresql_database_size') }}:</span>
                      <span class="process-value highlight">{{ formatBytes(systemInfo.processes.postgresql.database_size) }}</span>
                    </div>
                  </div>
                </div>
              </el-col>

              <!-- Redis 进程 -->
              <el-col :span="6" v-if="systemInfo.processes.redis">
                <div class="process-card">
                  <div class="process-header">
                    <el-tag :type="getProcessStatusType(systemInfo.processes.redis.status)" size="large">
                      {{ $t('monitor.process_redis') }}
                    </el-tag>
                    <el-tag v-if="systemInfo.processes.redis.type" :type="systemInfo.processes.redis.type === 'remote' ? 'warning' : 'success'" size="small">
                      {{ systemInfo.processes.redis.type === 'remote' ? $t('monitor.process_remote') : $t('monitor.process_local') }}
                    </el-tag>
                  </div>
                  <div class="process-content">
                    <div class="process-item">
                      <span class="process-label">{{ $t('monitor.process_status') }}:</span>
                      <span class="process-value" :class="getProcessStatusType(systemInfo.processes.redis.status) === 'success' ? 'highlight' : ''">
                        {{ systemInfo.processes.redis.status === 'running' ? $t('monitor.process_running') : 
                           systemInfo.processes.redis.status === 'sleep' ? $t('monitor.process_running') :
                           systemInfo.processes.redis.status === 'connected' ? $t('monitor.process_connected') :
                           systemInfo.processes.redis.status === 'not_found' ? $t('monitor.process_not_found') :
                           systemInfo.processes.redis.status === 'disconnected' ? $t('monitor.process_disconnected') :
                           systemInfo.processes.redis.status }}
                      </span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.pid && systemInfo.processes.redis.pid > 0">
                      <span class="process-label">{{ $t('monitor.process_pid') }}:</span>
                      <span class="process-value">{{ systemInfo.processes.redis.pid }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.cpu !== undefined && systemInfo.processes.redis.cpu > 0">
                      <span class="process-label">{{ $t('monitor.process_cpu') }}:</span>
                      <span class="process-value highlight">{{ formatPercent(systemInfo.processes.redis.cpu) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.memory && systemInfo.processes.redis.memory > 0">
                      <span class="process-label">{{ $t('monitor.process_memory') }}:</span>
                      <span class="process-value highlight">{{ formatBytes(systemInfo.processes.redis.memory) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.version">
                      <span class="process-label">{{ $t('monitor.process_version') }}:</span>
                      <span class="process-value">{{ systemInfo.processes.redis.version }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.host">
                      <span class="process-label">{{ $t('monitor.process_host') }}:</span>
                      <span class="process-value">{{ systemInfo.processes.redis.host }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.connected_clients !== undefined">
                      <span class="process-label">{{ $t('monitor.redis_connected_clients') }}:</span>
                      <span class="process-value highlight">{{ formatNumber(systemInfo.processes.redis.connected_clients || 0) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.total_commands_processed !== undefined && systemInfo.processes.redis.total_commands_processed > 0">
                      <span class="process-label">{{ $t('monitor.redis_total_commands') }}:</span>
                      <span class="process-value">{{ formatNumber(systemInfo.processes.redis.total_commands_processed) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.instantaneous_ops_per_sec !== undefined && systemInfo.processes.redis.instantaneous_ops_per_sec > 0">
                      <span class="process-label">{{ $t('monitor.redis_ops_per_sec') }}:</span>
                      <span class="process-value highlight">{{ formatNumber(systemInfo.processes.redis.instantaneous_ops_per_sec) }}/s</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.keyspace_hits !== undefined || systemInfo.processes.redis.keyspace_misses !== undefined">
                      <span class="process-label">{{ $t('monitor.redis_keyspace') }}:</span>
                      <span class="process-value">
                        {{ formatNumber(systemInfo.processes.redis.keyspace_hits || 0) }} / {{ formatNumber(systemInfo.processes.redis.keyspace_misses || 0) }}
                        <span v-if="systemInfo.processes.redis.keyspace_hit_rate !== undefined" class="connections-info">
                          ({{ formatPercent(systemInfo.processes.redis.keyspace_hit_rate) }})
                        </span>
                      </span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.used_memory_peak !== undefined && systemInfo.processes.redis.used_memory_peak > 0">
                      <span class="process-label">{{ $t('monitor.redis_memory_peak') }}:</span>
                      <span class="process-value">{{ formatBytes(systemInfo.processes.redis.used_memory_peak) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.uptime !== undefined && systemInfo.processes.redis.uptime > 0">
                      <span class="process-label">{{ $t('monitor.process_uptime') }}:</span>
                      <span class="process-value">{{ formatUptime(systemInfo.processes.redis.uptime) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.redis.db_size !== undefined && systemInfo.processes.redis.db_size > 0">
                      <span class="process-label">{{ $t('monitor.redis_db_size') }}:</span>
                      <span class="process-value highlight">{{ formatNumber(systemInfo.processes.redis.db_size) }}</span>
                    </div>
                  </div>
                </div>
              </el-col>

              <!-- 应用进程 -->
              <el-col :span="6" v-if="systemInfo.processes.app">
                <div class="process-card">
                  <div class="process-header">
                    <el-tag :type="getProcessStatusType(systemInfo.processes.app.status)" size="large">
                      {{ $t('monitor.process_app') }}
                    </el-tag>
                    <el-tag type="info" size="small">{{ $t('monitor.process_local') }}</el-tag>
                  </div>
                  <div class="process-content">
                    <div class="process-item">
                      <span class="process-label">{{ $t('monitor.process_status') }}:</span>
                      <span class="process-value" :class="getProcessStatusType(systemInfo.processes.app.status) === 'success' ? 'highlight' : ''">
                        {{ systemInfo.processes.app.status === 'running' ? $t('monitor.process_running') : 
                           systemInfo.processes.app.status === 'sleep' ? $t('monitor.process_running') :
                           systemInfo.processes.app.status === 'connected' ? $t('monitor.process_connected') :
                           systemInfo.processes.app.status === 'not_found' ? $t('monitor.process_not_found') :
                           systemInfo.processes.app.status }}
                      </span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.app.pid && systemInfo.processes.app.pid > 0">
                      <span class="process-label">{{ $t('monitor.process_pid') }}:</span>
                      <span class="process-value highlight">{{ systemInfo.processes.app.pid }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.app.cpu !== undefined && systemInfo.processes.app.cpu >= 0">
                      <span class="process-label">{{ $t('monitor.process_cpu') }}:</span>
                      <span class="process-value highlight">{{ formatPercent(systemInfo.processes.app.cpu) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.app.memory && systemInfo.processes.app.memory > 0">
                      <span class="process-label">{{ $t('monitor.process_memory') }}:</span>
                      <span class="process-value highlight">{{ formatBytes(systemInfo.processes.app.memory) }}</span>
                    </div>
                    <div class="process-item" v-if="systemInfo.processes.app.process_name">
                      <span class="process-label">{{ $t('monitor.process_name') }}:</span>
                      <span class="process-value">{{ systemInfo.processes.app.process_name }}</span>
                    </div>
                  </div>
                </div>
              </el-col>
            </el-row>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 进程占用排行 -->
    <el-row
      v-if="systemInfo.process_top && ((systemInfo.process_top.by_cpu && systemInfo.process_top.by_cpu.length) || (systemInfo.process_top.by_memory && systemInfo.process_top.by_memory.length))"
      :gutter="20"
      style="margin-top: 20px"
    >
      <el-col :span="24">
        <el-card class="monitor-card process-top-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon"><TrendCharts /></el-icon>
                <span>{{ $t('monitor.process_ranking') }}</span>
              </div>
              <span class="process-top-hint">{{ $t('monitor.process_ranking_hint') }}</span>
            </div>
          </template>
          <div class="monitor-content process-top-body">
            <el-row :gutter="16">
              <el-col :xs="24" :sm="24" :md="12">
                <div class="process-top-subtitle">{{ $t('monitor.process_top_by_cpu') }}</div>
                <div class="process-top-cpu-hint">{{ $t('monitor.process_top_cpu_hint') }}</div>
                <el-table
                  :data="systemInfo.process_top.by_cpu"
                  stripe
                  border
                  size="small"
                  class="process-top-table"
                  max-height="380"
                >
                  <el-table-column type="index" :label="'#'" width="48" />
                  <el-table-column prop="pid" :label="$t('monitor.process_pid')" width="88" />
                  <el-table-column prop="name" :label="$t('monitor.process_name')" min-width="120" show-overflow-tooltip />
                  <el-table-column v-if="hasProcessTopUser(systemInfo.process_top.by_cpu)" prop="user" :label="$t('monitor.process_top_user')" width="100" show-overflow-tooltip />
                  <el-table-column :label="$t('monitor.process_cpu')" width="100" align="right">
                    <template #default="{ row }">
                      {{ formatPercent(row.cpu_percent || 0) }}
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('monitor.memory_usage')" width="100" align="right">
                    <template #default="{ row }">
                      {{ formatPercent(row.memory_percent || 0) }}
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('monitor.process_memory')" min-width="110" align="right">
                    <template #default="{ row }">
                      {{ formatBytes(row.memory_bytes || 0) }}
                    </template>
                  </el-table-column>
                </el-table>
              </el-col>
              <el-col :xs="24" :sm="24" :md="12" class="process-top-col-second">
                <div class="process-top-subtitle">{{ $t('monitor.process_top_by_memory') }}</div>
                <el-table
                  :data="systemInfo.process_top.by_memory"
                  stripe
                  border
                  size="small"
                  class="process-top-table"
                  max-height="380"
                >
                  <el-table-column type="index" :label="'#'" width="48" />
                  <el-table-column prop="pid" :label="$t('monitor.process_pid')" width="88" />
                  <el-table-column prop="name" :label="$t('monitor.process_name')" min-width="120" show-overflow-tooltip />
                  <el-table-column v-if="hasProcessTopUser(systemInfo.process_top.by_memory)" prop="user" :label="$t('monitor.process_top_user')" width="100" show-overflow-tooltip />
                  <el-table-column :label="$t('monitor.process_cpu')" width="100" align="right">
                    <template #default="{ row }">
                      {{ formatPercent(row.cpu_percent || 0) }}
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('monitor.memory_usage')" width="100" align="right">
                    <template #default="{ row }">
                      {{ formatPercent(row.memory_percent || 0) }}
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('monitor.process_memory')" min-width="110" align="right">
                    <template #default="{ row }">
                      {{ formatBytes(row.memory_bytes || 0) }}
                    </template>
                  </el-table-column>
                </el-table>
              </el-col>
            </el-row>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 详细监控信息：TCP连接、磁盘IO、磁盘分区、网卡详情 -->
    <el-row v-if="showDetailedStatsRow" :gutter="20" style="margin-top: 20px">
      <!-- TCP连接统计 -->
      <el-col :span="detailStatsColSpan" v-if="hasTcpStats">
        <el-card class="monitor-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon"><Connection /></el-icon>
                <span>{{ $t('monitor.tcp_connections') }}</span>
              </div>
            </div>
          </template>
          <div class="monitor-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.tcp_total') }}</span>
                <span class="info-value highlight">{{ formatNumber(systemInfo.tcp_connections.total || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.tcp_established') }}</span>
                <span class="info-value highlight">{{ formatNumber(systemInfo.tcp_connections.established || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.tcp_listen') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.tcp_connections.listen || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.tcp_time_wait') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.tcp_connections.time_wait || 0) }}</span>
              </div>
            </div>
            <div v-if="systemInfo.tcp_connections.listening_ports && systemInfo.tcp_connections.listening_ports.length > 0" style="margin-top: 15px">
              <div class="info-label" style="margin-bottom: 8px; font-size: 12px">{{ $t('monitor.tcp_listening_ports') }}:</div>
              <el-tag v-for="port in systemInfo.tcp_connections.listening_ports.slice(0, 15)" :key="port" size="small" style="margin-right: 6px; margin-bottom: 6px">
                {{ port }}
              </el-tag>
              <span v-if="systemInfo.tcp_connections.listening_ports.length > 15" class="info-value" style="font-size: 11px; color: #909399">
                +{{ systemInfo.tcp_connections.listening_ports.length - 15 }}
              </span>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 磁盘IO统计 -->
      <el-col :span="detailStatsColSpan" v-if="hasDiskIOStats">
        <el-card class="monitor-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon"><TrendCharts /></el-icon>
                <span>{{ $t('monitor.disk_io') }}</span>
              </div>
            </div>
          </template>
          <div class="monitor-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.disk_read_bytes') }}</span>
                <span class="info-value">{{ formatBytes(systemInfo.disk_io.total_read_bytes || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.disk_write_bytes') }}</span>
                <span class="info-value">{{ formatBytes(systemInfo.disk_io.total_write_bytes || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.disk_read_count') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.disk_io.total_read_count || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.disk_write_count') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.disk_io.total_write_count || 0) }}</span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 磁盘分区信息（如果分区数量较少，显示在卡片中） -->
      <el-col :span="detailStatsColSpan" v-if="hasSmallDiskPartitions">
        <el-card class="monitor-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon"><FolderOpened /></el-icon>
                <span>{{ $t('monitor.disk_partitions') }}</span>
              </div>
            </div>
          </template>
          <div class="monitor-content">
            <div v-for="(part, index) in systemInfo.disk_partitions" :key="index" style="margin-bottom: 12px; padding-bottom: 12px; border-bottom: 1px solid #ebeef5" :class="{ 'no-border': index === systemInfo.disk_partitions.length - 1 }">
              <div style="font-weight: 600; margin-bottom: 6px; font-size: 13px">{{ part.mountpoint || part.device }}</div>
              <div class="info-grid" style="grid-template-columns: 1fr 1fr;">
                <div class="info-item">
                  <span class="info-label" style="font-size: 11px">{{ $t('monitor.partition_total') }}</span>
                  <span class="info-value" style="font-size: 12px">{{ formatBytes(part.total || 0) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label" style="font-size: 11px">{{ $t('monitor.partition_used') }}</span>
                  <span class="info-value highlight" style="font-size: 12px">{{ formatBytes(part.used || 0) }}</span>
                </div>
              </div>
              <el-progress
                :percentage="formatPercentForProgress(part.percent || 0)"
                :color="getProgressColor(part.percent || 0)"
                :stroke-width="6"
                style="margin-top: 6px"
              />
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 磁盘分区表格（分区数量较多时） -->
    <el-row v-if="systemInfo.disk_partitions && systemInfo.disk_partitions.length > 3" :gutter="20" style="margin-top: 20px">
      <el-col :span="24">
        <el-card class="monitor-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon"><FolderOpened /></el-icon>
                <span>{{ $t('monitor.disk_partitions') }}</span>
              </div>
            </div>
          </template>
          <div class="monitor-content">
            <el-table :data="systemInfo.disk_partitions" style="width: 100%" stripe>
              <el-table-column :label="$t('monitor.partition_device')" prop="device" width="150" />
              <el-table-column :label="$t('monitor.partition_mountpoint')" prop="mountpoint" />
              <el-table-column :label="$t('monitor.partition_fstype')" prop="fstype" width="120" />
              <el-table-column :label="$t('monitor.partition_total')" width="120">
                <template #default="{ row }">
                  {{ formatBytes(row.total || 0) }}
                </template>
              </el-table-column>
              <el-table-column :label="$t('monitor.partition_used')" width="120">
                <template #default="{ row }">
                  {{ formatBytes(row.used || 0) }}
                </template>
              </el-table-column>
              <el-table-column :label="$t('monitor.partition_free')" width="120">
                <template #default="{ row }">
                  {{ formatBytes(row.free || 0) }}
                </template>
              </el-table-column>
              <el-table-column :label="$t('monitor.partition_percent')" width="150">
                <template #default="{ row }">
                  <el-progress
                    :percentage="formatPercentForProgress(row.percent || 0)"
                    :color="getProgressColor(row.percent || 0)"
                    :stroke-width="8"
                  />
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 网络汇总（与网卡流量放在同一块） -->
    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="24">
        <el-card class="monitor-card network-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon network-icon"><Connection /></el-icon>
                <span>{{ $t('monitor.network') }}</span>
              </div>
              <el-button size="small" :icon="Refresh" circle :disabled="refreshing" :loading="refreshing" @click="loadData" />
            </div>
          </template>
          <div class="monitor-content">
            <div v-if="systemInfo.net?.speed_total_mbps !== undefined" class="monitor-item" style="margin-bottom: 15px; padding: 12px; background: var(--el-fill-color-light); border-radius: 8px">
              <div style="font-size: 13px; color: var(--el-text-color-regular); margin-bottom: 8px; font-weight: 600">{{ $t('monitor.bandwidth_speed') }}</div>
              <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px">
                <span style="font-size: 12px; color: #909399">{{ $t('monitor.current_speed') }}:</span>
                <span style="font-size: 16px; font-weight: 600; color: var(--el-color-primary)">
                  {{ formatNumber(systemInfo.net?.speed_total_mbps || 0, 2) }} Mbps
                </span>
              </div>
              <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px">
                <span style="font-size: 12px; color: #909399">{{ $t('monitor.peak_speed') }}:</span>
                <span style="font-size: 16px; font-weight: 600; color: #e6a23c">
                  {{ formatNumber(systemInfo.net?.peak_total_mbps || 0, 2) }} Mbps
                </span>
              </div>
              <div style="font-size: 11px; color: #909399; margin-top: 8px; padding-top: 8px; border-top: 1px solid #e4e7ed">
                <span>↑{{ formatNumber(systemInfo.net?.speed_sent_mbps || 0, 2) }} Mbps</span>
                <span style="margin: 0 8px">|</span>
                <span>↓{{ formatNumber(systemInfo.net?.speed_recv_mbps || 0, 2) }} Mbps</span>
                <span style="margin-left: 12px; color: #c0c4cc">
                  (峰值: ↑{{ formatNumber(systemInfo.net?.peak_sent_mbps || 0, 2) }} ↓{{ formatNumber(systemInfo.net?.peak_recv_mbps || 0, 2) }})
                </span>
              </div>
            </div>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.net_bytes_sent') }}</span>
                <span class="info-value highlight">{{ formatBytes(systemInfo.net?.bytes_sent || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.net_bytes_recv') }}</span>
                <span class="info-value highlight">{{ formatBytes(systemInfo.net?.bytes_recv || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.net_packets_sent') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.net?.packets_sent || 0) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">{{ $t('monitor.net_packets_recv') }}</span>
                <span class="info-value">{{ formatNumber(systemInfo.net?.packets_recv || 0) }}</span>
              </div>
              <div class="info-item" v-if="(systemInfo.net?.errin || 0) > 0 || (systemInfo.net?.errout || 0) > 0 || (systemInfo.net?.dropin || 0) > 0 || (systemInfo.net?.dropout || 0) > 0">
                <span class="info-label" style="color: #f56c6c">{{ $t('monitor.net_errin') }}</span>
                <span class="info-value error">{{ formatNumber(systemInfo.net?.errin || 0) }}</span>
              </div>
              <div class="info-item" v-if="(systemInfo.net?.errin || 0) > 0 || (systemInfo.net?.errout || 0) > 0 || (systemInfo.net?.dropin || 0) > 0 || (systemInfo.net?.dropout || 0) > 0">
                <span class="info-label" style="color: #f56c6c">{{ $t('monitor.net_dropin') }}</span>
                <span class="info-value error">{{ formatNumber(systemInfo.net?.dropin || 0) }}</span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 网卡流量详情 -->
    <el-row v-if="systemInfo.net?.interfaces && systemInfo.net.interfaces.length > 0" :gutter="20" style="margin-top: 20px">
      <el-col :span="24">
        <el-card class="monitor-card interfaces-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon class="card-icon interfaces-icon"><Monitor /></el-icon>
                <span>{{ $t('monitor.network_interfaces') }}</span>
              </div>
              <el-button size="small" :icon="Refresh" circle :disabled="refreshing" :loading="refreshing" @click="loadData" />
            </div>
          </template>
          <div class="monitor-content">
            <el-table 
              :data="systemInfo.net.interfaces" 
              border 
              style="width: 100%"
              class="interfaces-table"
              stripe
            >
              <el-table-column :label="$t('monitor.interface_name')" prop="name" width="150" />
              <el-table-column :label="$t('monitor.interface_bytes_sent')" width="150">
                <template #default="{ row }">
                  {{ formatBytes(row.bytes_sent || 0) }}
                </template>
              </el-table-column>
              <el-table-column :label="$t('monitor.interface_bytes_recv')" width="150">
                <template #default="{ row }">
                  {{ formatBytes(row.bytes_recv || 0) }}
                </template>
              </el-table-column>
              <el-table-column :label="$t('monitor.interface_packets_sent')" width="150">
                <template #default="{ row }">
                  {{ formatNumber(row.packets_sent || 0) }}
                </template>
              </el-table-column>
              <el-table-column :label="$t('monitor.interface_packets_recv')" width="150">
                <template #default="{ row }">
                  {{ formatNumber(row.packets_recv || 0) }}
                </template>
              </el-table-column>
              <el-table-column :label="$t('monitor.interface_errin')" width="120">
                <template #default="{ row }">
                  <span :style="{ color: (row.errin || 0) > 0 ? '#f56c6c' : '' }">{{ formatNumber(row.errin || 0) }}</span>
                </template>
              </el-table-column>
              <el-table-column :label="$t('monitor.interface_errout')" width="120">
                <template #default="{ row }">
                  <span :style="{ color: (row.errout || 0) > 0 ? '#f56c6c' : '' }">{{ formatNumber(row.errout || 0) }}</span>
                </template>
              </el-table-column>
              <el-table-column :label="$t('monitor.interface_dropin')" width="120">
                <template #default="{ row }">
                  <span :style="{ color: (row.dropin || 0) > 0 ? '#f56c6c' : '' }">{{ formatNumber(row.dropin || 0) }}</span>
                </template>
              </el-table-column>
              <el-table-column :label="$t('monitor.interface_dropout')" width="120">
                <template #default="{ row }">
                  <span :style="{ color: (row.dropout || 0) > 0 ? '#f56c6c' : '' }">{{ formatNumber(row.dropout || 0) }}</span>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, onActivated, onDeactivated, computed, nextTick, watch } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useAppStore, THEME_COLORS } from '../../store/app'
import * as echarts from 'echarts'
import { 
  Cpu, 
  DataBoard, 
  FolderOpened, 
  Connection, 
  Monitor, 
  TrendCharts, 
  Document, 
  Refresh,
  Operation
} from '@element-plus/icons-vue'
import { getSystemInfo, createSystemInfoSSE } from '../../api/monitor'
import { createSSEConnection, closeSSEConnection } from '../../utils/sse'

const { t } = useI18n()
const appStore = useAppStore()

// 主题相关计算属性
const isDark = computed(() => appStore.darkMode)
const textColor = computed(() => isDark.value ? '#cfd3dc' : '#303133')
const secondaryTextColor = computed(() => isDark.value ? '#909399' : '#606266')
const primaryColor = computed(() => {
  const preset = THEME_COLORS.find((t) => t.key === appStore.themeColor) || THEME_COLORS[0]
  return preset.color
})

const systemInfo = ref({
  os: 'linux',
  cpu: {},
  memory: {},
  disk: {},
  net: {},
  load: {},
  file_descriptors: {},
  runtime: {},
  system: {},
  processes: {},
  alerts: [],
  tcp_connections: {},
  disk_io: {},
  disk_partitions: [],
  process_top: {
    by_cpu: [],
    by_memory: [],
    limit: 20
  }
})
// 判断是否为Linux系统
const isLinux = computed(() => {
  return systemInfo.value.os === 'linux'
})

const hasTcpStats = computed(() => !!systemInfo.value.tcp_connections)
const hasDiskIOStats = computed(() => (systemInfo.value.disk_io?.total_read_bytes || 0) > 0)
const hasSmallDiskPartitions = computed(() => {
  const partitions = systemInfo.value.disk_partitions || []
  return partitions.length > 0 && partitions.length <= 3
})
const showDetailedStatsRow = computed(() => hasTcpStats.value || hasDiskIOStats.value || hasSmallDiskPartitions.value)
const detailStatsCardCount = computed(() => {
  let count = 0
  if (hasTcpStats.value) count++
  if (hasDiskIOStats.value) count++
  if (hasSmallDiskPartitions.value) count++
  return count
})
const detailStatsColSpan = computed(() => {
  if (detailStatsCardCount.value <= 1) return 24
  if (detailStatsCardCount.value === 2) return 12
  return 8
})

const loading = ref(false)
const refreshing = ref(false) // 刷新按钮状态，防止重复点击
let eventSource = null
let refreshTimer = null
let errorCount = 0 // SSE 错误计数
let lastErrorTime = 0 // 上次错误时间
const MAX_ERROR_COUNT = 5 // 最大错误次数，超过后降级到轮询
const ERROR_WINDOW = 10000 // 错误时间窗口（10秒）
const POLL_INTERVAL_MS = 30000 // SSE 失败后的 HTTP 轮询间隔

// 刷新节流控制
let lastRefreshTime = 0 // 上次刷新时间
const MIN_REFRESH_INTERVAL = 2000 // 最小刷新间隔（毫秒），2秒

// 图表引用
const cpuChartRef = ref(null)
const memoryChartRef = ref(null)
const diskChartRef = ref(null)
const networkChartRef = ref(null)
const resourcePieChartRef = ref(null)

// 图表实例
let cpuChartInstance = null
let memoryChartInstance = null
let diskChartInstance = null
let networkChartInstance = null
let resourcePieChartInstance = null

// 历史数据（保留最近60个数据点，约2分钟，每2秒一个点）
const historyData = ref({
  cpu: [],
  memory: [],
  disk: [],
  network: {
    sent: [],
    recv: [],
    total: []
  },
  timestamps: []
})

// 最大历史数据点数
const MAX_HISTORY_POINTS = 60

// 使用 SSE 实时推送
const startSSEStream = () => {
  try {
    // 重置错误计数
    errorCount = 0
    lastErrorTime = 0
    
    const url = createSystemInfoSSE({ interval: 2 })
    eventSource = createSSEConnection(url, {
      onMessage: (data) => {
        // 收到消息时重置错误计数
        errorCount = 0
        lastErrorTime = 0
        
        if (data.type === 'connected') {
          // SSE 连接成功消息（前端根据当前语言翻译）
          if (data.message_key) {
            const message = t(`messages.${data.message_key}`)
            console.log('SSE connected:', message)
          }
        } else if (data.type === 'system_info') {
          systemInfo.value = data.data || {}
          loading.value = false
          // 更新历史数据和图表
          updateHistoryData()
          updateCharts()
        }
      },
      onError: (error, source) => {
        const now = Date.now()
        
        // 如果距离上次错误时间超过窗口期，重置计数
        if (now - lastErrorTime > ERROR_WINDOW) {
          errorCount = 0
        }
        
        errorCount++
        lastErrorTime = now
        
        // 只有在错误次数过多或连接真正关闭时才降级到轮询
        // EventSource 会自动重连，所以大多数错误是正常的
        if (source && source.readyState === EventSource.CLOSED) {
          // 连接已关闭，无法重连
          console.warn('SSE connection closed, switching to polling mode')
          ElMessage.warning(t('monitor.sse_connection_failed') || '实时推送连接失败，已切换到定时刷新模式')
          closeSSEConnection(eventSource)
          eventSource = null
          startPolling()
        } else if (errorCount >= MAX_ERROR_COUNT) {
          // 错误次数过多，可能网络不稳定，降级到轮询
          console.warn(`SSE connection errors exceeded ${MAX_ERROR_COUNT}, switching to polling mode`)
          ElMessage.warning(t('monitor.sse_connection_failed') || '实时推送连接不稳定，已切换到定时刷新模式')
          closeSSEConnection(eventSource)
          eventSource = null
          startPolling()
        } else {
          // EventSource 会自动重连，这是正常行为，只记录调试信息
          // 不打印错误日志，避免控制台噪音
          // console.debug('SSE connection error (will auto-reconnect):', error)
        }
      },
      onOpen: () => {
        // 连接成功时重置错误计数
        errorCount = 0
        lastErrorTime = 0
        console.log('SSE connected for system info')
        loading.value = false
      }
    })
  } catch (error) {
    console.error('Failed to start SSE stream:', error)
    ElMessage.warning(t('monitor.sse_init_failed') || '无法启动实时推送，已切换到定时刷新模式')
    startPolling()
  }
}

// SSE 失败后的定时刷新降级方案
const startPolling = () => {
  if (refreshTimer) {
    return
  }
  loadData({ silent: true })
  refreshTimer = setInterval(() => {
    loadData({ silent: true })
  }, POLL_INTERVAL_MS)
}

// 更新历史数据
const updateHistoryData = () => {
  const now = new Date()
  const timeStr = now.toLocaleTimeString()
  
  // 添加新数据点
  historyData.value.cpu.push(systemInfo.value.cpu?.percent || 0)
  historyData.value.memory.push(systemInfo.value.memory?.percent || 0)
  historyData.value.disk.push(systemInfo.value.disk?.percent || 0)
  historyData.value.network.sent.push(systemInfo.value.net?.speed_sent_mbps || 0)
  historyData.value.network.recv.push(systemInfo.value.net?.speed_recv_mbps || 0)
  historyData.value.network.total.push(systemInfo.value.net?.speed_total_mbps || 0)
  historyData.value.timestamps.push(timeStr)
  
  // 限制数据点数量
  if (historyData.value.cpu.length > MAX_HISTORY_POINTS) {
    historyData.value.cpu.shift()
    historyData.value.memory.shift()
    historyData.value.disk.shift()
    historyData.value.network.sent.shift()
    historyData.value.network.recv.shift()
    historyData.value.network.total.shift()
    historyData.value.timestamps.shift()
  }
}

// 初始化图表
const initCharts = () => {
  nextTick(() => {
    // CPU使用率走势图
    if (cpuChartRef.value && !cpuChartInstance) {
      cpuChartInstance = echarts.init(cpuChartRef.value)
      cpuChartInstance.setOption({
        backgroundColor: 'transparent',
        grid: { top: 10, left: 30, right: 10, bottom: 30 },
        xAxis: {
          type: 'category',
          data: [],
          axisLabel: { fontSize: 10, rotate: 45, color: textColor.value },
          axisLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#dcdfe6' } },
          splitLine: { show: false }
        },
        yAxis: {
          type: 'value',
          max: 100,
          axisLabel: { fontSize: 10, formatter: '{value}%', color: textColor.value },
          axisLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#dcdfe6' } },
          splitLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#ebeef5' } }
        },
        series: [{
          data: [],
          type: 'line',
          smooth: true,
          areaStyle: { opacity: 0.3 },
          lineStyle: { color: '#F56C6C', width: 2 },
          itemStyle: { color: '#F56C6C' }
        }],
        tooltip: { 
          trigger: 'axis', 
          axisPointer: { type: 'cross' },
          backgroundColor: isDark.value ? '#2d2d30' : '#fff',
          borderColor: isDark.value ? '#3d3e40' : '#e4e7ed',
          textStyle: { color: textColor.value },
          formatter: (params) => {
            if (Array.isArray(params) && params.length > 0) {
              const time = params[0].axisValue
              let result = time + '<br/>'
              params.forEach(item => {
                const value = typeof item.value === 'number' ? item.value.toFixed(1) : item.value
                result += `${item.marker}${t('monitor.cpu')}: ${value}%<br/>`
              })
              return result
            }
            return ''
          }
        }
      })
    }
    
    // 内存使用率走势图
    if (memoryChartRef.value && !memoryChartInstance) {
      memoryChartInstance = echarts.init(memoryChartRef.value)
      memoryChartInstance.setOption({
        backgroundColor: 'transparent',
        grid: { top: 10, left: 30, right: 10, bottom: 30 },
        xAxis: {
          type: 'category',
          data: [],
          axisLabel: { fontSize: 10, rotate: 45, color: textColor.value },
          axisLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#dcdfe6' } },
          splitLine: { show: false }
        },
        yAxis: {
          type: 'value',
          max: 100,
          axisLabel: { fontSize: 10, formatter: '{value}%', color: textColor.value },
          axisLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#dcdfe6' } },
          splitLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#ebeef5' } }
        },
        series: [{
          data: [],
          type: 'line',
          smooth: true,
          areaStyle: { opacity: 0.3 },
          lineStyle: { color: primaryColor.value, width: 2 },
          itemStyle: { color: primaryColor.value }
        }],
        tooltip: { 
          trigger: 'axis', 
          axisPointer: { type: 'cross' },
          backgroundColor: isDark.value ? '#2d2d30' : '#fff',
          borderColor: isDark.value ? '#3d3e40' : '#e4e7ed',
          textStyle: { color: textColor.value },
          formatter: (params) => {
            if (Array.isArray(params) && params.length > 0) {
              const time = params[0].axisValue
              let result = time + '<br/>'
              params.forEach(item => {
                const value = typeof item.value === 'number' ? item.value.toFixed(1) : item.value
                result += `${item.marker}${t('monitor.memory')}: ${value}%<br/>`
              })
              return result
            }
            return ''
          }
        }
      })
    }
    
    // 磁盘使用率走势图
    if (diskChartRef.value && !diskChartInstance) {
      diskChartInstance = echarts.init(diskChartRef.value)
      diskChartInstance.setOption({
        backgroundColor: 'transparent',
        grid: { top: 10, left: 30, right: 10, bottom: 30 },
        xAxis: {
          type: 'category',
          data: [],
          axisLabel: { fontSize: 10, rotate: 45, color: textColor.value },
          axisLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#dcdfe6' } },
          splitLine: { show: false }
        },
        yAxis: {
          type: 'value',
          max: 100,
          axisLabel: { fontSize: 10, formatter: '{value}%', color: textColor.value },
          axisLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#dcdfe6' } },
          splitLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#ebeef5' } }
        },
        series: [{
          data: [],
          type: 'line',
          smooth: true,
          areaStyle: { opacity: 0.3 },
          lineStyle: { color: '#67C23A', width: 2 },
          itemStyle: { color: '#67C23A' }
        }],
        tooltip: { 
          trigger: 'axis', 
          axisPointer: { type: 'cross' },
          backgroundColor: isDark.value ? '#2d2d30' : '#fff',
          borderColor: isDark.value ? '#3d3e40' : '#e4e7ed',
          textStyle: { color: textColor.value },
          formatter: (params) => {
            if (Array.isArray(params) && params.length > 0) {
              const time = params[0].axisValue
              let result = time + '<br/>'
              params.forEach(item => {
                const value = typeof item.value === 'number' ? item.value.toFixed(2) : item.value
                result += `${item.marker}${item.seriesName}: ${value} Mbps<br/>`
              })
              return result
            }
            return ''
          }
        }
      })
    }
    
    // 网络带宽速度走势图
    if (networkChartRef.value && !networkChartInstance) {
      networkChartInstance = echarts.init(networkChartRef.value)
      networkChartInstance.setOption({
        backgroundColor: 'transparent',
        grid: { top: 20, left: 40, right: 20, bottom: 30 },
        legend: {
          data: [t('monitor.net_send'), t('monitor.net_receive')],
          top: 5,
          textStyle: { fontSize: 11, color: textColor.value }
        },
        xAxis: {
          type: 'category',
          data: [],
          axisLabel: { fontSize: 10, rotate: 45, color: textColor.value },
          axisLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#dcdfe6' } },
          splitLine: { show: false }
        },
        yAxis: {
          type: 'value',
          axisLabel: { fontSize: 10, formatter: '{value} Mbps', color: textColor.value },
          axisLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#dcdfe6' } },
          splitLine: { lineStyle: { color: isDark.value ? '#3d3e40' : '#ebeef5' } }
        },
        series: [
          {
            name: t('monitor.net_send'),
            data: [],
            type: 'line',
            smooth: true,
            lineStyle: { color: primaryColor.value, width: 2 },
            itemStyle: { color: primaryColor.value }
          },
          {
            name: t('monitor.net_receive'),
            data: [],
            type: 'line',
            smooth: true,
            lineStyle: { color: '#E6A23C', width: 2 },
            itemStyle: { color: '#E6A23C' }
          }
        ],
        tooltip: { 
          trigger: 'axis', 
          axisPointer: { type: 'cross' },
          backgroundColor: isDark.value ? '#2d2d30' : '#fff',
          borderColor: isDark.value ? '#3d3e40' : '#e4e7ed',
          textStyle: { color: textColor.value },
          formatter: (params) => {
            if (Array.isArray(params) && params.length > 0) {
              const time = params[0].axisValue
              let result = time + '<br/>'
              params.forEach(item => {
                const value = typeof item.value === 'number' ? item.value.toFixed(2) : item.value
                result += `${item.marker}${item.seriesName}: ${value} Mbps<br/>`
              })
              return result
            }
            return ''
          }
        }
      })
    }
    
    // 资源使用率饼图
    if (resourcePieChartRef.value && !resourcePieChartInstance) {
      resourcePieChartInstance = echarts.init(resourcePieChartRef.value)
      resourcePieChartInstance.setOption({
        backgroundColor: 'transparent',
        tooltip: {
          trigger: 'item',
          backgroundColor: isDark.value ? '#2d2d30' : '#fff',
          borderColor: isDark.value ? '#3d3e40' : '#e4e7ed',
          textStyle: { color: textColor.value },
          formatter: (params) => {
            const value = typeof params.value === 'number' ? params.value.toFixed(1) : params.value
            const percent = typeof params.percent === 'number' ? params.percent.toFixed(1) : params.percent
            return `${params.seriesName}<br/>${params.name}: ${value}% (${percent}%)`
          }
        },
        legend: {
          orient: 'vertical',
          left: 'left',
          top: 'middle',
          textStyle: { fontSize: 12, color: textColor.value }
        },
        series: [{
          name: t('monitor.resource_usage'),
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['60%', '50%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 10,
            borderColor: isDark.value ? '#3d3e40' : '#fff',
            borderWidth: 2
          },
          label: {
            show: true,
            color: textColor.value,
            formatter: (params) => {
              const value = typeof params.value === 'number' ? params.value.toFixed(1) : params.value
              return `${params.name}: ${value}%`
            }
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 14,
              fontWeight: 'bold'
            }
          },
          data: []
        }]
      })
    }
    
    // 窗口大小改变时调整图表
    window.addEventListener('resize', handleResize)
  })
}

// 更新图表数据
const updateCharts = () => {
  const timestamps = historyData.value.timestamps
  const cpuData = historyData.value.cpu
  const memoryData = historyData.value.memory
  const diskData = historyData.value.disk
  const networkSent = historyData.value.network.sent
  const networkRecv = historyData.value.network.recv
  
  // 更新CPU图表
  if (cpuChartInstance && cpuData.length > 0) {
    cpuChartInstance.setOption({
      xAxis: { data: timestamps },
      series: [{ data: cpuData }]
    })
  }
  
  // 更新内存图表
  if (memoryChartInstance && memoryData.length > 0) {
    memoryChartInstance.setOption({
      xAxis: { data: timestamps },
      series: [{ data: memoryData }]
    })
  }
  
  // 更新磁盘图表
  if (diskChartInstance && diskData.length > 0) {
    diskChartInstance.setOption({
      xAxis: { data: timestamps },
      series: [{ data: diskData }]
    })
  }
  
  // 更新网络图表
  if (networkChartInstance && networkSent.length > 0) {
    networkChartInstance.setOption({
      xAxis: { data: timestamps },
      series: [
        { data: networkSent },
        { data: networkRecv }
      ]
    })
  }
  
  // 更新资源饼图
  if (resourcePieChartInstance) {
    const cpuPercent = systemInfo.value.cpu?.percent || 0
    const memoryPercent = systemInfo.value.memory?.percent || 0
    const diskPercent = systemInfo.value.disk?.percent || 0
    
    resourcePieChartInstance.setOption({
      series: [{
        data: [
          { 
            value: cpuPercent, 
            name: t('monitor.cpu'),
            itemStyle: { color: '#F56C6C' }
          },
          { 
            value: memoryPercent, 
            name: t('monitor.memory'),
            itemStyle: { color: primaryColor.value }
          },
          { 
            value: diskPercent, 
            name: t('monitor.disk'),
            itemStyle: { color: '#67C23A' }
          }
        ]
      }]
    })
  }
}

// 处理窗口大小改变
const handleResize = () => {
  cpuChartInstance?.resize()
  memoryChartInstance?.resize()
  diskChartInstance?.resize()
  networkChartInstance?.resize()
  resourcePieChartInstance?.resize()
}

// 手动刷新（兼容原有功能）
const hasMonitorData = () => {
  const { cpu, memory } = systemInfo.value
  return Boolean(cpu?.model || memory?.total)
}

const loadData = async (options = {}) => {
  const { silent = false } = options

  // 如果正在刷新，直接返回，防止重复点击
  if (refreshing.value) {
    return
  }
  
  // 节流控制：检查距离上次刷新的时间间隔（手动刷新才限制）
  const now = Date.now()
  const timeSinceLastRefresh = now - lastRefreshTime
  
  if (!silent && timeSinceLastRefresh < MIN_REFRESH_INTERVAL) {
    // 如果距离上次刷新时间太短，提示用户并忽略本次点击
    const remainingTime = Math.ceil((MIN_REFRESH_INTERVAL - timeSinceLastRefresh) / 1000)
    ElMessage.warning(t('monitor.refresh_too_frequent', { seconds: remainingTime }))
    return
  }
  
  // 更新上次刷新时间
  lastRefreshTime = now
  
  refreshing.value = !silent
  if (!silent) {
    loading.value = true
  }
  try {
    const { data } = await getSystemInfo()
    systemInfo.value = data || {}
    updateHistoryData()
    updateCharts()
  } catch (error) {
    console.error('Load system info error:', error)
    if (!silent && !error.__handled) {
      const errorMessage = error.response?.data?.message || error.message || t('error.default')
      ElMessage.error(errorMessage)
    }
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

// 进程排行中是否展示「用户」列（任一进程有 user 则显示）
const hasProcessTopUser = (rows) => {
  if (!rows || !rows.length) return false
  return rows.some((r) => r && r.user)
}

const formatBytes = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const value = bytes / Math.pow(k, i)
  
  // 根据值的大小决定小数位数
  if (value >= 100) {
    return Math.round(value) + ' ' + sizes[i]
  } else if (value >= 10) {
    return Math.round(value * 10) / 10 + ' ' + sizes[i]
  } else {
    return Math.round(value * 100) / 100 + ' ' + sizes[i]
  }
}

const formatNumber = (num, decimals = 0) => {
  if (num === null || num === undefined) return '0'
  const value = Number(num)
  
  // 如果指定了小数位数
  if (decimals > 0) {
    // 对于小于1的值，保留指定的小数位数
    if (value < 1) {
      return value.toFixed(decimals).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
    }
    // 对于大于等于1的值，智能处理小数位数
    if (value >= 100) {
      // 大于等于100，不显示小数
      return Math.round(value).toLocaleString()
    } else if (value >= 10) {
      // 10-100之间，保留1位小数
      return (Math.round(value * 10) / 10).toFixed(1).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
    } else {
      // 1-10之间，保留指定的小数位数
      return value.toFixed(decimals).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
    }
  }
  
  // 没有指定小数位数，使用默认格式化
  return value.toLocaleString()
}

const formatBool = (value) => (value ? 'true' : 'false')

// 格式化百分比：根据值的大小决定保留的小数位数
// 优化显示：避免过多小数位
const formatPercent = (percent) => {
  if (percent === 0 || percent === null || percent === undefined) return '0%'
  
  // 如果百分比大于等于100，保留0位小数
  if (percent >= 100) {
    return Math.round(percent) + '%'
  }
  // 如果百分比大于等于1，保留1位小数
  if (percent >= 1) {
    return percent.toFixed(1) + '%'
  }
  // 如果百分比小于1，保留2位小数
  return percent.toFixed(2) + '%'
}

// 格式化百分比用于进度条（保留2位小数）
const formatPercentForProgress = (percent) => {
  return Math.round(percent * 100) / 100
}

// 格式化负载值
const formatLoad = (load) => {
  if (load === 0) return '0'
  const value = Number(load)
  // 负载值通常小于10，保留2位小数；大于等于10时保留1位小数
  if (value >= 10) {
    return value.toFixed(1)
  }
  return value.toFixed(2)
}

const getProgressColor = (percentage) => {
  if (percentage < 50) {
    return '#67c23a'
  } else if (percentage < 80) {
    return '#e6a23c'
  } else {
    return '#f56c6c'
  }
}

// 格式化运行时间（秒转换为友好格式）
const formatUptime = (seconds) => {
  if (!seconds || seconds <= 0) return '0秒'
  
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60
  
  const parts = []
  if (days > 0) parts.push(`${days}天`)
  if (hours > 0) parts.push(`${hours}小时`)
  if (minutes > 0) parts.push(`${minutes}分钟`)
  if (secs > 0 || parts.length === 0) parts.push(`${secs}秒`)
  
  return parts.join(' ')
}

// 格式化时长（毫秒转换为友好格式）
const formatDuration = (milliseconds) => {
  if (!milliseconds || milliseconds <= 0) return '0ms'
  
  if (milliseconds < 1000) {
    return `${Math.round(milliseconds)}ms`
  }
  
  const seconds = milliseconds / 1000
  if (seconds < 60) {
    return `${Math.round(seconds * 10) / 10}s`
  }
  
  const minutes = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${minutes}分${secs}秒`
}

// 获取连接数百分比样式类
const getConnectionPercentClass = (connections, maxConnections) => {
  if (!maxConnections || maxConnections <= 0) return ''
  const percent = (connections || 0) / maxConnections * 100
  if (percent >= 90) return 'connections-danger'
  if (percent >= 80) return 'connections-warning'
  return 'connections-safe'
}

// 获取进程状态类型
const getProcessStatusType = (status) => {
  // sleep 状态对于服务进程来说是正常的，显示为 success
  if (status === 'running' || status === 'connected' || status === 'sleep') {
    return 'success'
  } else if (status === 'not_found' || status === 'disconnected') {
    return 'danger'
  } else if (status === 'error' || status === 'zombie' || status === 'stopped') {
    return 'warning'
  }
  return 'info'
}

// 获取告警消息（根据 message_key 和 value 翻译）
const getAlertMessage = (alert) => {
  if (!alert) return ''
  
  // 如果已经有 message 字段（兼容旧数据），直接返回
  if (alert.message) {
    return alert.message
  }
  
  // 根据 message_key 和 value 翻译
  if (alert.message_key && alert.value !== undefined) {
    const messageKey = `messages.${alert.message_key}`
    // 使用 i18n 翻译，value 作为参数
    return t(messageKey, { value: alert.value.toFixed(2) })
  }
  
  return ''
}

// 清理函数：关闭 SSE 连接和定时器
const cleanup = () => {
  if (eventSource) {
    closeSSEConnection(eventSource)
    eventSource = null
  }
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

// 监听主题变化，重新初始化图表
watch(() => appStore.darkMode, () => {
  if (cpuChartInstance) {
    cpuChartInstance.dispose()
    cpuChartInstance = null
  }
  if (memoryChartInstance) {
    memoryChartInstance.dispose()
    memoryChartInstance = null
  }
  if (diskChartInstance) {
    diskChartInstance.dispose()
    diskChartInstance = null
  }
  if (networkChartInstance) {
    networkChartInstance.dispose()
    networkChartInstance = null
  }
  if (resourcePieChartInstance) {
    resourcePieChartInstance.dispose()
    resourcePieChartInstance = null
  }
  nextTick(() => {
    initCharts()
    updateCharts()
  })
})

onMounted(() => {
  initCharts()
  window.addEventListener('resize', handleResize)
  if (!hasMonitorData()) {
    loadData({ silent: true })
  }
  startSSEStream()
})

// 路由离开前清理（确保路由切换时断开连接）
onBeforeRouteLeave(() => {
  cleanup()
})

// 组件被缓存时清理（keep-alive场景）
onDeactivated(() => {
  cleanup()
  window.removeEventListener('resize', handleResize)
})

// keep-alive 下重新激活时恢复 SSE（不重复关闭/重建，避免打断连接）
onActivated(() => {
  window.addEventListener('resize', handleResize)
  if (!cpuChartInstance) {
    initCharts()
  } else {
    handleResize()
  }
  if (!hasMonitorData()) {
    loadData({ silent: true })
  }
  if (!eventSource) {
    startSSEStream()
  }
})

onUnmounted(() => {
  cleanup()
  // 移除窗口大小监听
  window.removeEventListener('resize', handleResize)
  // 销毁图表实例
  cpuChartInstance?.dispose()
  memoryChartInstance?.dispose()
  diskChartInstance?.dispose()
  networkChartInstance?.dispose()
  resourcePieChartInstance?.dispose()
})
</script>

<style scoped lang="scss">
.monitor-page {
  padding: var(--space-md);
  background: var(--bg-color-tertiary);
  min-height: calc(100vh - 60px);
  transition: background 0.3s ease;
}

/* 暗黑模式样式 */
html.dark .monitor-page {
  background: var(--el-bg-color-page) !important;
}

.monitor-card {
  border-radius: var(--border-radius-lg);
  border: 1px solid var(--border-color-light);
  transition: all 0.3s ease;
  overflow: hidden;
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1) !important;
  }

  :deep(.el-card__header) {
    background: var(--el-color-primary);
    color: #fff;
    padding: var(--space-md) 20px;
    border-bottom: none;
  }

  :deep(.el-card__body) {
    padding: var(--space-md);
    background: var(--card-bg);
    transition: background-color 0.3s ease;
  }
}

/* 暗黑模式样式 */
html.dark .monitor-card :deep(.el-card__body) {
  background: var(--el-bg-color) !important;
}

.cpu-card :deep(.el-card__header) {
  background: #F56C6C;
}

.memory-card :deep(.el-card__header) {
  background: var(--el-color-primary);
}

.disk-card :deep(.el-card__header) {
  background: #67C23A;
}

.network-card :deep(.el-card__header) {
  background: #E6A23C;
}

.interfaces-card :deep(.el-card__header) {
  background: #909399;
}

.load-card :deep(.el-card__header) {
  background: var(--el-color-primary);
}

.fd-card :deep(.el-card__header) {
  background: #909399;
}

.runtime-card :deep(.el-card__header) {
  background: var(--el-color-primary);
}

.system-card :deep(.el-card__header) {
  background: #909399;
}

.processes-card :deep(.el-card__header) {
  background: var(--el-color-primary);
}

.process-top-card :deep(.el-card__header) {
  background: var(--el-color-primary);
}

.process-top-hint {
  font-size: 12px;
  font-weight: 400;
  opacity: 0.92;
  max-width: 58%;
  text-align: right;
  line-height: 1.45;
}

.usage-scope-hint {
  font-size: 12px;
  font-weight: 400;
  opacity: 0.85;
}

.process-top-cpu-hint {
  font-size: 12px;
  font-weight: 400;
  color: var(--text-color-secondary);
  margin: -4px 0 8px;
  line-height: 1.5;
}

html.dark .process-top-cpu-hint {
  color: var(--el-text-color-secondary) !important;
}

.process-top-subtitle {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-color-regular);
  margin-bottom: var(--space-sm);
}

html.dark .process-top-subtitle {
  color: var(--el-text-color-regular) !important;
}

.process-top-body {
  padding-top: 4px;
}

.process-top-table {
  width: 100%;
}

@media (max-width: 991px) {
  .process-top-col-second {
    margin-top: 18px;
  }
}

.process-card {
  padding: var(--space-md);
  background: var(--card-bg);
  border-radius: var(--border-radius-lg);
  border: 1px solid var(--border-color-light);
  height: 100%;
  transition: all 0.3s ease;
  
  &:hover {
    background: var(--bg-color-tertiary);
    border-color: var(--el-color-primary);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(64, 158, 255, 0.15);
  }
}

/* 暗黑模式样式 */
html.dark .process-card {
  background: var(--el-bg-color) !important;
  border-color: var(--el-border-color) !important;
  
  &:hover {
    background: var(--el-bg-color-page) !important;
    border-color: var(--el-color-primary) !important;
  }
}

.process-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-md);
  padding-bottom: var(--space-sm);
  border-bottom: 2px solid var(--border-color-light);
}

/* 暗黑模式样式 */
html.dark .process-header {
  border-bottom-color: var(--el-border-color) !important;
}

.process-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.process-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-xs) 0;
}

.process-label {
  font-size: 13px;
  color: var(--text-color-secondary);
  font-weight: 500;
}

.process-value {
  font-size: 14px;
  color: var(--text-color-primary);
  font-weight: 600;
  
  &.highlight {
    color: var(--el-color-primary);
    font-size: 15px;
  }
  
  &.warning {
    color: #e6a23c;
  }
  
  &.error {
    color: #f56c6c;
  }
}

/* 暗黑模式样式 */
html.dark .process-label {
  color: var(--el-text-color-secondary) !important;
}

html.dark .process-value {
  color: var(--el-text-color-regular) !important;
}

.connections-info {
  font-size: 13px;
  color: var(--text-color-secondary);
  font-weight: 500;
  margin-left: 4px;
}

.connections-percent {
  font-size: 12px;
  margin-left: 4px;
  
  &.connections-safe {
    color: #67c23a;
  }
  
  &.connections-warning {
    color: #e6a23c;
  }
  
  &.connections-danger {
    color: #f56c6c;
    font-weight: 600;
  }
}

/* 暗黑模式样式 */
html.dark .connections-info {
  color: var(--el-text-color-secondary) !important;
}

.no-border {
  border-bottom: none !important;
  padding-bottom: 0 !important;
  margin-bottom: 0 !important;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
  font-size: 16px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.card-icon {
  font-size: 20px;
  opacity: 0.9;
}

.monitor-content {
  padding: 0;
}

/* 暗黑模式样式 - 确保图表容器背景透明 */
html.dark .monitor-content {
  background: transparent !important;
}

html.dark .monitor-content > div {
  background: transparent !important;
}

.usage-item {
  margin-bottom: var(--space-lg);
  padding-bottom: 20px;
  border-bottom: 2px solid var(--border-color-extra-light);
}

/* 暗黑模式样式 */
html.dark .usage-item {
  border-bottom-color: var(--el-border-color-lighter) !important;
}

.usage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-sm);
  
  .label {
    font-weight: 600;
    color: var(--text-color-primary);
    font-size: 14px;
  }
  
  .percent-value {
    font-weight: 700;
    font-size: 18px;
    color: var(--el-color-primary);
  }
}

/* 暗黑模式样式 */
html.dark .usage-header .label {
  color: var(--el-text-color-primary) !important;
}

.usage-progress {
  :deep(.el-progress-bar__outer) {
    border-radius: 12px;
    overflow: hidden;
  }
  
  :deep(.el-progress-bar__inner) {
    border-radius: 12px;
    transition: width 0.6s ease;
  }
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-md);
  
  .full-width {
    grid-column: 1 / -1;
  }
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: var(--space-sm);
  background: var(--card-bg);
  border-radius: var(--border-radius-lg);
  border: 1px solid var(--border-color-light);
  transition: all 0.3s ease;
  
  &:hover {
    background: var(--bg-color-tertiary);
    border-color: var(--el-color-primary);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(64, 158, 255, 0.15);
  }
}

/* 暗黑模式样式 */
html.dark .info-item {
  background: var(--el-bg-color-page) !important;
  border-color: var(--el-border-color) !important;
  
  &:hover {
    background: var(--el-fill-color-light) !important;
    border-color: var(--el-color-primary) !important;
  }
}

.info-label {
  font-size: 12px;
  color: var(--text-color-secondary);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-value {
  font-size: 15px;
  color: var(--text-color-primary);
  font-weight: 600;
  word-break: break-all;
  
  &.highlight {
    color: var(--el-color-primary);
    font-size: 16px;
  }
  
  &.error {
    color: #f56c6c;
  }
}

/* 暗黑模式样式 */
html.dark .info-label {
  color: var(--el-text-color-secondary) !important;
}

html.dark .info-value {
  color: var(--el-text-color-regular) !important;
}

.load-display {
  text-align: center;
  padding: 30px 20px;
  background: var(--card-bg);
  border-radius: var(--border-radius-lg);
  border: 1px solid var(--border-color-light);
}

/* 暗黑模式样式 */
html.dark .load-display {
  background: var(--el-bg-color) !important;
  border-color: var(--el-border-color) !important;
}

.load-value {
  margin-bottom: 12px;
}

.load-number {
  font-size: 48px;
  font-weight: 700;
  color: var(--el-color-primary);
  display: inline-block;
  margin-right: 8px;
}

.load-percent {
  font-size: 20px;
  color: var(--text-color-secondary);
  font-weight: 500;
}

.load-label {
  font-size: 14px;
  color: var(--text-color-regular);
  font-weight: 500;
}

/* 暗黑模式样式 */
html.dark .load-label {
  color: var(--el-text-color-regular) !important;
}

html.dark .load-percent {
  color: var(--el-text-color-secondary) !important;
}

.interfaces-table {
  :deep(.el-table__header) {
    th {
      background: var(--bg-color-tertiary);
      color: var(--text-color-primary);
      font-weight: 600;
      border-bottom: 1px solid var(--border-color-light);
    }
  }
  
  :deep(.el-table__row) {
    transition: all 0.3s ease;
    
    &:hover {
      background-color: #f0f9ff;
      transform: scale(1.01);
    }
  }
  
  :deep(.el-table__body) {
    tr {
      td {
        padding: 12px 0;
      }
    }
  }
}

/* 暗黑模式样式 - Element Plus Table会自动适配，这里只需要确保hover效果 */
html.dark .interfaces-table :deep(.el-table__row:hover) {
  background-color: var(--el-fill-color-light) !important;
}

// 响应式设计
@media (max-width: 1200px) {
  .info-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .monitor-page {
    padding: var(--space-sm);
  }
  
  .monitor-card {
    margin-bottom: var(--space-md);
  }
  
  .load-number {
    font-size: 36px;
  }

  /* 宽表格在窄屏可横向滚动，避免列被裁切 */
  .monitor-page :deep(.el-card__body) {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  .monitor-page :deep(.el-table) {
    min-width: 640px;
  }
}
</style>