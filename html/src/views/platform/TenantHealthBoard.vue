<template>
  <el-card shadow="never" class="health-board" v-loading="loading && !board">
    <template #header>
      <div class="health-board-header">
        <span>{{ $t('tenant.health_board_title') }}</span>
        <div class="health-board-meta">
          <span class="health-board-last">
            {{
              board?.last_checked_at
                ? $t('tenant.health_board_last_checked', { t: formatDateTimeDisplay(board.last_checked_at) })
                : $t('tenant.health_board_never')
            }}
          </span>
          <el-button link type="primary" @click="load">{{ $t('common.refresh') }}</el-button>
        </div>
      </div>
    </template>

    <el-alert
      v-if="(board?.stale || 0) > 0"
      type="warning"
      show-icon
      :closable="false"
      class="health-board-block"
      :title="$t('tenant.health_board_stale', { n: board.stale, h: staleHours })"
    />

    <div v-if="issueChips.length" class="health-board-chips health-board-block">
      <el-check-tag :checked="!activeFilter" @change="issueFilter = ''">
        {{ $t('tenant.health_board_all_issues') }} ({{ board?.abnormal ?? 0 }})
      </el-check-tag>
      <el-check-tag
        v-for="chip in issueChips"
        :key="chip.code"
        :checked="activeFilter === chip.code"
        @change="(checked) => (issueFilter = checked ? chip.code : '')"
      >
        {{ issueLabel(chip.code) }} ({{ chip.count }})
      </el-check-tag>
    </div>

    <el-empty
      v-if="board && board.abnormal === 0"
      :image-size="60"
      :description="$t('tenant.health_board_no_issue')"
    />
    <template v-else-if="board">
      <el-table :data="pagedRows" size="small" row-key="id" v-loading="loading">
        <el-table-column :label="$t('tenant.health_status')" width="90">
          <template #default="{ row }">
            <el-tag :type="row.health_status === 'fail' ? 'danger' : 'warning'" size="small">
              {{ row.health_status === 'fail' ? $t('tenant.health_fail') : $t('tenant.health_warn') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('menu.tenant')" min-width="140">
          <template #default="{ row }">
            <div class="tenant-code">
              {{ row.code }}
              <el-tag v-if="row.maintenance" size="small" type="info" class="maintenance-tag">
                {{ $t('tenant.maintenance_on') }}
              </el-tag>
            </div>
            <div v-if="row.name && row.name !== row.code" class="tenant-name">{{ row.name }}</div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('tenant.health_board_issues')" min-width="220">
          <template #default="{ row }">
            <div class="issue-tags">
              <el-tag
                v-for="code in row.issues"
                :key="code"
                size="small"
                :type="FAIL_ISSUES.has(code) ? 'danger' : 'warning'"
                effect="plain"
              >
                {{ issueLabel(code) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('tenant.health_board_ping')" width="100">
          <template #default="{ row }">
            <span v-if="row.last_ping_ok">{{ row.last_ping_ms }} ms</span>
            <span v-else class="text-danger">—</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('tenant.health_board_checked_at')" width="170">
          <template #default="{ row }">
            <span :class="{ 'text-warning': row.stale }">
              {{ row.health_checked_at ? formatDateTimeDisplay(row.health_checked_at) : $t('tenant.health_board_never') }}
            </span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('common.actions')" width="110">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="viewTenant(row)">
              {{ $t('tenant.health_board_view') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-if="rows.length > PAGE_SIZE"
        v-model:current-page="page"
        class="health-board-pager"
        size="small"
        layout="prev, pager, next"
        :page-size="PAGE_SIZE"
        :total="rows.length"
      />
    </template>

    <div v-if="board?.truncated" class="health-board-note">
      {{ $t('tenant.health_board_truncated', { n: board.items.length }) }}
    </div>
  </el-card>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { getPlatformTenantHealthBoard } from '@/api/platform'
import { formatDateTimeDisplay } from '@/utils/dateUtils'

const props = defineProps({
  /** Bump to reload, e.g. after a manual inspect. */
  reloadKey: { type: Number, default: 0 },
})

// Fail-level issues first so the chip row reads by severity.
const ISSUE_ORDER = [
  'ping_fail',
  'provision_failed',
  'migrate_failed',
  'schema_failed',
  'storage_quota_full',
  'schema_behind',
  'storage_quota_high',
  'domain_verify_failed',
]
const FAIL_ISSUES = new Set(['ping_fail', 'provision_failed', 'migrate_failed', 'schema_failed', 'storage_quota_full'])
const PAGE_SIZE = 10

const { t, te } = useI18n()
const router = useRouter()
const loading = ref(false)
const board = ref(null)
const issueFilter = ref('')
const page = ref(1)

const issueLabel = (code) => {
  const key = `tenant.health_issue_${code}`
  return te(key) ? t(key) : code
}

const staleHours = computed(() => Math.round((board.value?.stale_after_sec || 7200) / 3600))

const issueChips = computed(() => {
  const byIssue = board.value?.by_issue || {}
  const rank = (code) => {
    const i = ISSUE_ORDER.indexOf(code)
    return i === -1 ? 99 : i
  }
  return Object.keys(byIssue)
    .filter((code) => (byIssue[code] || 0) > 0)
    .sort((a, b) => rank(a) - rank(b) || a.localeCompare(b))
    .map((code) => ({ code, count: byIssue[code] }))
})

// Drop a selected issue once a reload no longer reports it, so the table never sticks empty.
const activeFilter = computed(() =>
  issueFilter.value && (board.value?.by_issue?.[issueFilter.value] || 0) > 0 ? issueFilter.value : '',
)

const rows = computed(() => {
  const items = board.value?.items || []
  return activeFilter.value ? items.filter((it) => (it.issues || []).includes(activeFilter.value)) : items
})

const pagedRows = computed(() => rows.value.slice((page.value - 1) * PAGE_SIZE, page.value * PAGE_SIZE))

watch(activeFilter, () => {
  page.value = 1
})

const load = async () => {
  loading.value = true
  try {
    const res = await getPlatformTenantHealthBoard()
    board.value = res?.data?.board || null
    page.value = 1
  } catch {
    board.value = null
  } finally {
    loading.value = false
  }
}

const viewTenant = (row) => {
  router.push({ path: '/platform/tenants', query: { code: row.code } })
}

watch(() => props.reloadKey, load)
onMounted(load)
</script>

<style scoped>
.health-board {
  margin-bottom: 16px;
}
.health-board-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.health-board-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}
.health-board-last {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.health-board-block {
  margin-bottom: 12px;
}
.health-board-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.tenant-code {
  font-weight: 600;
}
.maintenance-tag {
  margin-left: 6px;
  font-weight: normal;
}
.tenant-name {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.issue-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.text-danger {
  color: var(--el-color-danger);
}
.text-warning {
  color: var(--el-color-warning);
}
.health-board-pager {
  margin-top: 12px;
  justify-content: flex-end;
}
.health-board-note {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
