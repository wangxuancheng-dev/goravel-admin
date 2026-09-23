<template>
  <div class="platform-tenant-page">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      class="ops-banner"
      :title="$t('platform.cli_ops_title')"
      :description="healthDesc"
    />
    <el-alert
      type="warning"
      :closable="false"
      show-icon
      class="ops-banner"
      :title="$t('platform.migrate_current_binary_title')"
      :description="$t('platform.migrate_current_binary_desc')"
    />
    <div class="list-mode-bar">
      <el-radio-group :model-value="isRecycleView ? 'recycle' : 'active'" size="default" @change="onListModeChange">
        <el-radio-button label="active">{{ $t('tenant.list_active') }}</el-radio-button>
        <el-radio-button label="recycle">
          {{ $t('tenant.recycle_bin') }}
          <el-badge v-if="(opsSummary?.deleted || 0) > 0" :value="opsSummary.deleted" class="recycle-badge" />
        </el-radio-button>
      </el-radio-group>
      <span v-if="opsSummary" class="ops-inline">
        {{ $t('tenant.ops_summary', { failed: opsSummary.failed_provision || 0, busy: opsSummary.busy || 0, total: opsSummary.total || 0 }) }}
        <template v-if="(opsSummary.schema_behind || 0) > 0 || (opsSummary.schema_failed || 0) > 0">
          · {{ $t('tenant.schema_summary', {
            aligned: opsSummary.schema_aligned || 0,
            behind: opsSummary.schema_behind || 0,
            failed: opsSummary.schema_failed || 0,
            unknown: opsSummary.schema_unknown || 0
          }) }}
        </template>
        <template v-if="(opsSummary.failed_purge || 0) > 0">
          · {{ $t('tenant.failed_purge_count', { n: opsSummary.failed_purge }) }}
        </template>
        <template v-if="(opsSummary.failed_seed || 0) > 0">
          · {{ $t('tenant.failed_seed_count', { n: opsSummary.failed_seed }) }}
        </template>
        <template v-if="(opsSummary.maintenance || 0) > 0">
          · {{ $t('tenant.ops_maintenance') }} {{ opsSummary.maintenance }}
        </template>
        <template v-if="opsSummary.domain_unbound != null">
          · {{ $t('tenant.domain_summary', {
            unbound: opsSummary.domain_unbound || 0,
            pending: opsSummary.domain_pending || 0,
            active: opsSummary.domain_active || 0,
            failed: opsSummary.domain_verify_failed || 0
          }) }}
        </template>
        <template v-if="(opsSummary.health_fail || 0) > 0 || (opsSummary.health_warn || 0) > 0">
          · {{ $t('tenant.health_summary', {
            ok: opsSummary.health_ok || 0,
            warn: opsSummary.health_warn || 0,
            fail: opsSummary.health_fail || 0
          }) }}
        </template>
      </span>
      <template v-if="!isRecycleView && isOwner">
        <el-button
          size="small"
          :loading="batchLoading"
          :disabled="!(opsSummary?.failed_provision > 0)"
          @click="retryFailedMigrates"
        >
          {{ $t('tenant.retry_failed_migrate') }}
        </el-button>
        <el-button
          size="small"
          :loading="batchLoading"
          :disabled="!(opsSummary?.failed_seed > 0)"
          @click="retryFailedSeeds"
        >
          {{ $t('tenant.retry_failed_seed') }}
        </el-button>
        <el-button
          size="small"
          type="primary"
          :loading="batchLoading"
          :disabled="selectedRows.length < 1"
          @click="batchMigrateSelected"
        >
          {{ $t('tenant.batch_migrate') }}
        </el-button>
        <el-button
          size="small"
          :loading="batchLoading"
          :disabled="selectedRows.length < 1"
          @click="batchSeedSelected"
        >
          {{ $t('tenant.batch_seed') }}
        </el-button>
        <el-button
          size="small"
          :loading="batchLoading"
          :disabled="selectedRows.length < 1"
          @click="batchBackupSelected"
        >
          {{ $t('tenant.batch_backup') }}
        </el-button>
      </template>
      <template v-if="!isRecycleView">
        <el-button size="small" @click="exportCsv">{{ $t('tenant.export_csv') }}</el-button>
      </template>
      <span v-if="queueInfo" class="queue-meta">{{ $t('tenant.queue_status', { conn: queueInfo.connection || '-', pending: queueInfo.pending ?? '-', msg: queueInfo.message || '' }) }}</span>
    </div>

    <el-alert
      v-if="isRecycleView"
      type="warning"
      :closable="false"
      show-icon
      class="recycle-banner"
      :title="$t('tenant.recycle_banner_title')"
      :description="recycleBannerDesc"
    />

    <ListPage
    ref="listPageRef"
    page-class="platform-tenant"
    :title="isRecycleView ? $t('tenant.recycle_bin') : $t('menu.tenant')"
    :add-button-text="$t('tenant.add')"
    :show-add-button="!isRecycleView && isOwner"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="initialSearchForm"
    i18n-prefix="tenant"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :pagination="pagination"
    show-toolbar
    @add="openCreate"
    @search="handleSearch"
    @reset="onListReset"
    @refresh="loadData"
    @page-change="loadData"
    @sort-change="handleSortChange"
    @selection-change="handleSelectionChange"
  >
    <template #provision_status="{ row }">
      <el-tooltip :content="row.last_migrate_error || row.last_op_message || ''" :disabled="!(row.last_migrate_error || row.last_op_message)">
        <el-tag :type="provisionTagType(row.provision_status)" size="small">
          {{ provisionLabel(row.provision_status) }}
        </el-tag>
      </el-tooltip>
    </template>
    <template #schema_status="{ row }">
      <el-tooltip
        :content="schemaStatusTip(row)"
        :disabled="!schemaStatusTip(row)"
      >
        <el-tag :type="schemaTagType(row.schema_status)" size="small" effect="plain">
          {{ schemaLabel(row.schema_status) }}
        </el-tag>
      </el-tooltip>
    </template>
    <template #domain_status="{ row }">
      <el-tooltip :content="row.domain_primary_host || ''" :disabled="!row.domain_primary_host">
        <el-tag :type="domainTagType(row.domain_status)" size="small" effect="plain">
          {{ domainLabel(row.domain_status) }}
        </el-tag>
      </el-tooltip>
    </template>
    <template #health_status="{ row }">
      <el-tooltip :content="(row.health_issues || []).join(', ')" :disabled="!(row.health_issues || []).length">
        <el-tag :type="healthTagType(row.health_status)" size="small">
          {{ healthLabel(row.health_status) }}
          <span v-if="row.last_ping_ms != null && row.health_status"> · {{ row.last_ping_ms }}ms</span>
        </el-tag>
      </el-tooltip>
    </template>
    <template #last_op="{ row }">
      <el-tooltip :content="row.last_op_message || row.last_op_at || ''" :disabled="!(row.last_op_message || row.last_op_at)">
        <el-tag v-if="recycleOpTag(row)" :type="recycleOpTag(row).type" size="small" effect="plain">
          {{ recycleOpTag(row).label }}
        </el-tag>
        <span v-else class="op-meta">{{ row.last_op || '—' }} / {{ row.last_op_status || '—' }}</span>
      </el-tooltip>
    </template>
    <template #last_backup_path="{ row }">
      <div v-if="row.last_backup_path" class="backup-cell">
        <el-tooltip :content="row.last_backup_path" placement="top">
          <span class="backup-path">{{ shortPath(row.last_backup_path) }}</span>
        </el-tooltip>
        <el-button link type="primary" @click="copyText(row.last_backup_path)">{{ $t('tenant.backup_copy_path') }}</el-button>
      </div>
      <span v-else class="op-meta">—</span>
    </template>
    <template #status="{ row }">
      <el-switch
        :model-value="Number(row.status) === 1"
        :disabled="!isOwner"
        @change="(val) => onToggleStatus(row, val)"
      />
    </template>
    <template #maintenance="{ row }">
      <el-switch
        :model-value="!!row.maintenance"
        :disabled="!isOwner"
        :title="$t('tenant.maintenance_hint')"
        @change="(val) => onToggleMaintenance(row, val)"
      />
    </template>
    <template #actions="{ row }">
      <template v-if="isRecycleView">
        <template v-if="isOwner">
          <el-button type="primary" link :disabled="isBusy(row)" @click="onUndelete(row)">{{ $t('tenant.op_undelete') }}</el-button>
          <el-button type="danger" link :disabled="isBusy(row)" @click="openForceDelete(row)">{{ $t('tenant.op_force_delete_short') }}</el-button>
        </template>
        <el-dropdown trigger="click" @command="(cmd) => onRecycleMore(cmd, row)">
          <el-button link type="primary">{{ $t('tenant.op_more') }}</el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="timeline">{{ $t('tenant.op_timeline') }}</el-dropdown-item>
              <el-dropdown-item
                v-if="isOwner && row.last_op === 'purge' && row.last_op_status === 'failed'"
                command="purge"
                :disabled="isBusy(row)"
              >
                {{ $t('tenant.op_purge_retry') }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </template>
      <template v-else>
        <el-button link type="primary" @click="openDetail(row)">{{ $t('tenant.op_detail') }}</el-button>
        <el-button v-if="isOwner" link type="primary" :disabled="isBusy(row)" @click="openEdit(row)">{{ $t('common.edit') }}</el-button>
        <el-button link type="primary" @click="onPing(row)">{{ $t('tenant.op_ping') }}</el-button>
        <el-button link type="primary" @click="openSupport(row)">{{ $t('tenant.support') }}</el-button>
        <el-button v-if="isOwner" link type="primary" :disabled="isBusy(row)" @click="openMigrate(row)">{{ $t('tenant.op_migrate') }}</el-button>
        <el-dropdown trigger="click" @command="(cmd) => onMoreCommand(cmd, row)">
          <el-button link type="primary">{{ $t('tenant.op_more') }}</el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-if="isOwner" command="seed" :disabled="isBusy(row)">{{ $t('tenant.op_seed') }}</el-dropdown-item>
              <el-dropdown-item v-if="isOwner" command="backup" :disabled="isBusy(row)">{{ $t('tenant.op_backup') }}</el-dropdown-item>
              <el-dropdown-item command="backups">{{ $t('tenant.op_backups') }}</el-dropdown-item>
              <el-dropdown-item command="overview">{{ $t('tenant.op_overview') }}</el-dropdown-item>
              <el-dropdown-item command="timeline">{{ $t('tenant.op_timeline') }}</el-dropdown-item>
              <el-dropdown-item command="login">{{ $t('tenant.op_login_link') }}</el-dropdown-item>
              <el-dropdown-item v-if="isOwner" command="delete" divided>{{ $t('tenant.op_delete') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </template>
    </template>

    <template #form>
      <el-dialog
        v-model="dialogVisible"
        :title="editingId ? $t('tenant.edit') : $t('tenant.add')"
        width="640px"
        destroy-on-close
        @closed="resetForm"
      >
        <el-form ref="formRef" :model="form" :rules="formRules" label-width="110px" autocomplete="off">
          <el-form-item v-if="!editingId" :label="$t('tenant.code')" prop="code">
            <el-input v-model="form.code" :placeholder="$t('tenant.code_placeholder')" />
          </el-form-item>
          <el-form-item :label="$t('tenant.name')" prop="name">
            <el-input v-model="form.name" />
          </el-form-item>
          <template v-if="!editingId">
            <el-form-item :label="$t('tenant.driver')" prop="driver">
              <el-select v-model="form.driver" style="width: 100%">
                <el-option label="mysql" value="mysql" />
                <el-option label="postgres" value="postgres" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('tenant.isolation')" prop="isolation">
              <el-select v-model="form.isolation" style="width: 100%">
                <el-option label="database" value="database" />
                <el-option label="schema" value="schema" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('tenant.database')">
              <el-input v-model="form.database" :placeholder="$t('tenant.database_placeholder')" />
            </el-form-item>
            <el-form-item :label="$t('tenant.schema')">
              <el-input v-model="form.schema" :placeholder="$t('tenant.schema_placeholder')" />
            </el-form-item>
          </template>
          <el-divider content-position="left">{{ $t('tenant.remote_connection') }}</el-divider>
          <el-form-item :label="$t('tenant.host')">
            <el-input v-model="form.host" :placeholder="$t('tenant.host_placeholder')" />
          </el-form-item>
          <el-form-item :label="$t('tenant.port')">
            <el-input-number v-model="form.port" :min="0" :max="65535" controls-position="right" style="width: 100%" />
          </el-form-item>
          <el-form-item :label="$t('tenant.username')">
            <el-input
              v-model="form.username"
              name="tenant_db_username"
              autocomplete="off"
              readonly
              :placeholder="$t('tenant.username_placeholder')"
              @focus="unlockCredentialAutofill"
            />
          </el-form-item>
          <el-form-item :label="$t('tenant.password')">
            <el-input
              v-model="form.password"
              type="password"
              name="tenant_db_password"
              autocomplete="new-password"
              show-password
              readonly
              :placeholder="editingId && form.has_password ? $t('tenant.password_keep') : $t('tenant.password_placeholder')"
              @focus="unlockCredentialAutofill"
              @input="passwordTouched = true"
            />
          </el-form-item>
          <el-form-item v-if="editingId" :label="$t('tenant.database')">
            <el-input v-model="form.database" />
          </el-form-item>
          <el-form-item v-if="editingId" :label="$t('tenant.schema')">
            <el-input v-model="form.schema" />
          </el-form-item>
          <template v-if="!editingId">
            <el-divider content-position="left">{{ $t('tenant.onboard_section') }}</el-divider>
            <el-form-item :label="$t('tenant.onboard_migrate')">
              <el-switch v-model="form.with_migrate" />
            </el-form-item>
            <el-form-item :label="$t('tenant.onboard_seed')">
              <el-switch v-model="form.with_seed" :disabled="!form.with_migrate" />
            </el-form-item>
            <el-form-item :label="$t('tenant.onboard_domain')">
              <el-input v-model="form.domain_host" placeholder="crm.customer.com" clearable />
            </el-form-item>
            <el-form-item v-if="form.domain_host" :label="$t('tenant.domain_ssl_edge')">
              <el-select v-model="form.domain_ssl_mode" style="width: 100%">
                <el-option :label="$t('tenant.domain_ssl_edge')" value="edge" />
                <el-option :label="$t('tenant.domain_ssl_cdn')" value="customer_cdn" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('tenant.skip_create')">
              <el-switch v-model="form.skip_create" />
              <div class="form-tip">{{ $t('tenant.skip_create_tip') }}</div>
            </el-form-item>
          </template>
          <el-divider content-position="left">{{ $t('tenant.quota_section') }}</el-divider>
          <el-form-item :label="$t('tenant.storage_limit_mb')">
            <el-input-number v-model="form.storage_limit_mb" :min="0" :max="1048576" controls-position="right" style="width: 100%" />
            <div class="form-tip">{{ $t('tenant.quota_zero_unlimited') }}</div>
          </el-form-item>
          <el-divider content-position="left">{{ $t('tenant.storage_byob_section') }}</el-divider>
          <el-form-item :label="$t('tenant.storage_mode')">
            <el-select v-model="form.storage_mode" style="width: 100%">
              <el-option :label="$t('tenant.storage_mode_shared')" value="shared" />
              <el-option :label="$t('tenant.storage_mode_custom')" value="custom" />
            </el-select>
            <div class="form-tip">{{ $t('tenant.storage_mode_tip') }}</div>
          </el-form-item>
          <template v-if="form.storage_mode === 'custom'">
            <el-form-item :label="$t('tenant.storage_driver')">
              <el-select v-model="form.storage_driver" style="width: 100%">
                <el-option label="S3" value="s3" />
                <el-option label="OSS" value="oss" />
                <el-option label="COS" value="cos" />
                <el-option label="MinIO" value="minio" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('tenant.storage_bucket')">
              <el-input v-model="form.storage_bucket" />
            </el-form-item>
            <el-form-item :label="$t('tenant.storage_key')">
              <el-input v-model="form.storage_key" autocomplete="off" />
            </el-form-item>
            <el-form-item :label="$t('tenant.storage_secret')">
              <el-input
                v-model="form.storage_secret"
                type="password"
                show-password
                autocomplete="new-password"
                :placeholder="form.storage_has_secret ? $t('tenant.storage_secret_keep') : $t('tenant.storage_secret_placeholder')"
                @input="storageSecretTouched = true"
              />
            </el-form-item>
            <el-form-item :label="$t('tenant.storage_region')">
              <el-input v-model="form.storage_region" />
            </el-form-item>
            <el-form-item :label="$t('tenant.storage_endpoint')">
              <el-input v-model="form.storage_endpoint" />
            </el-form-item>
            <el-form-item :label="$t('tenant.storage_url')">
              <el-input v-model="form.storage_url" />
            </el-form-item>
            <el-form-item v-if="form.storage_driver === 's3'" :label="$t('tenant.storage_use_path_style')">
              <el-switch v-model="form.storage_use_path_style" />
            </el-form-item>
            <el-form-item v-if="form.storage_driver === 'minio'" :label="$t('tenant.storage_ssl')">
              <el-switch v-model="form.storage_ssl" />
            </el-form-item>
          </template>
        </el-form>
        <template #footer>
          <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" :loading="saving" @click="submitForm">{{ $t('common.confirm') }}</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="onboardResultVisible" :title="$t('tenant.onboard_done_title')" width="520px" destroy-on-close>
        <p>{{ $t('tenant.onboard_done_desc') }}</p>
        <el-input :model-value="onboardLoginUrl" readonly>
          <template #append>
            <el-button @click="copyText(onboardLoginUrl)">{{ $t('tenant.backup_copy_path') }}</el-button>
          </template>
        </el-input>
        <template #footer>
          <el-button type="primary" @click="onboardResultVisible = false">{{ $t('common.confirm') }}</el-button>
        </template>
      </el-dialog>
    </template>
  </ListPage>

  <el-dialog v-model="migrateVisible" :title="$t('tenant.op_migrate')" width="480px" destroy-on-close>
    <p>{{ $t('tenant.op_migrate_confirm') }}</p>
    <div class="seed-row">
      <el-switch v-model="withSeed" />
      <span>{{ $t('tenant.op_with_seed') }}</span>
    </div>
    <el-alert
      v-if="migrateRow?.last_migrate_error"
      type="error"
      :closable="false"
      show-icon
      class="migrate-tip"
      :title="$t('tenant.migrate_error')"
      :description="migrateRow.last_migrate_error"
    />
    <template #footer>
      <el-button @click="migrateVisible = false">{{ $t('common.cancel') }}</el-button>
      <el-button type="primary" @click="submitMigrate">{{ $t('common.confirm') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog
    v-model="detailVisible"
    :title="$t('tenant.detail_title')"
    width="800px"
    align-center
    destroy-on-close
    class="tenant-detail-dialog"
  >
    <template v-if="detailRow">
      <div class="dialog-scroll-body">
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item :label="$t('tenant.code')">{{ detailRow.code }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.name')">{{ detailRow.name }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.driver')">{{ detailRow.driver }} / {{ detailRow.isolation }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.database')">{{ detailRow.database }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.schema')">{{ detailRow.schema || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.host')">{{ detailRow.host || '—' }}:{{ detailRow.port || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.connection_name')">{{ detailRow.connection_name || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.provision_status')">{{ provisionLabel(detailRow.provision_status) }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.maintenance')">
          {{ detailRow.maintenance ? $t('tenant.maintenance_on') : $t('tenant.maintenance_off') }}
          <span v-if="detailRow.maintenance_message"> · {{ detailRow.maintenance_message }}</span>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.last_op')">{{ detailRow.last_op || '—' }} / {{ detailRow.last_op_status || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.last_op_at')">{{ detailRow.last_op_at || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.op_message')">{{ detailRow.last_op_message || detailRow.last_migrate_error || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.migrated_at')">{{ detailRow.migrated_at || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.backup_dir')">
          <div class="backup-cell">
            <span>{{ detailRow.backup_dir || backupDirOf(detailRow) }}</span>
            <el-button link type="primary" @click="copyText(detailRow.backup_dir || backupDirOf(detailRow))">{{ $t('tenant.backup_copy_path') }}</el-button>
          </div>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.backup_path')">
          <div v-if="detailRow.last_backup_path" class="backup-cell">
            <span>{{ detailRow.last_backup_path }}</span>
            <el-button link type="primary" @click="copyText(detailRow.last_backup_path)">{{ $t('tenant.backup_copy_path') }}</el-button>
          </div>
          <span v-else>—</span>
        </el-descriptions-item>
      </el-descriptions>
      <div class="domain-panel">
        <div class="domain-title">{{ $t('tenant.domains_title') }}</div>
        <p class="domain-hint">{{ $t('tenant.domains_hint') }}</p>
        <div v-if="isOwner" class="domain-add">
          <el-input v-model="domainHost" placeholder="crm.customer.com" clearable style="width: 200px" />
          <el-select v-model="domainSslMode" style="width: 150px">
            <el-option :label="$t('tenant.domain_ssl_edge')" value="edge" />
            <el-option :label="$t('tenant.domain_ssl_cdn')" value="customer_cdn" />
          </el-select>
          <el-button type="primary" :loading="domainBusy" @click="addDomain">{{ $t('tenant.domain_add') }}</el-button>
        </div>
        <div v-for="d in detailDomains" :key="d.id" class="domain-item">
          <div>
            <strong>{{ d.host }}</strong>
            <el-tag size="small" class="ml4">{{ d.status }}</el-tag>
            <el-tag size="small" class="ml4">{{ d.ssl_mode }}</el-tag>
            <el-tag v-if="d.is_primary" size="small" type="primary" class="ml4">{{ $t('tenant.domain_primary') }}</el-tag>
          </div>
          <div v-if="d.dns_guide?.txt_name" class="domain-dns">
            <div class="domain-dns-title">{{ $t('tenant.domain_dns_required') }}</div>
            <p v-if="d.ssl_mode === 'customer_cdn'" class="domain-dns-tip">{{ $t('tenant.domain_dns_cf_hint') }}</p>
            <p v-else-if="d.ssl_mode === 'edge'" class="domain-dns-tip">{{ $t('tenant.domain_dns_edge_hint') }}</p>
            <div class="domain-dns-row domain-dns-row--txt">
              <div class="domain-dns-type">TXT</div>
              <div class="domain-dns-body">
                <div class="domain-dns-line">
                  <span class="domain-dns-label">{{ $t('tenant.domain_dns_name') }}</span>
                  <code>{{ d.dns_guide.txt_name }}</code>
                  <el-button link type="primary" @click="copyText(d.dns_guide.txt_name)">{{ $t('common.copy') }}</el-button>
                </div>
                <div class="domain-dns-line">
                  <span class="domain-dns-label">{{ $t('tenant.domain_dns_value') }}</span>
                  <code>{{ d.dns_guide.txt_value }}</code>
                  <el-button link type="primary" @click="copyText(d.dns_guide.txt_value)">{{ $t('common.copy') }}</el-button>
                </div>
              </div>
            </div>
            <div v-if="d.dns_guide.domain_target && d.ssl_mode === 'edge'" class="domain-dns-row domain-dns-row--cname">
              <div class="domain-dns-type">CNAME</div>
              <div class="domain-dns-body">
                <div class="domain-dns-line">
                  <span class="domain-dns-label">{{ $t('tenant.domain_dns_name') }}</span>
                  <code>{{ d.host }}</code>
                  <el-button link type="primary" @click="copyText(d.host)">{{ $t('common.copy') }}</el-button>
                </div>
                <div class="domain-dns-line">
                  <span class="domain-dns-label">{{ $t('tenant.domain_dns_target') }}</span>
                  <code>{{ d.dns_guide.domain_target }}</code>
                  <el-button link type="primary" @click="copyText(d.dns_guide.domain_target)">{{ $t('common.copy') }}</el-button>
                </div>
              </div>
            </div>
            <p v-if="d.ssl_mode === 'customer_cdn' && d.dns_guide.domain_target" class="domain-dns-tip domain-dns-tip--muted">
              {{ $t('tenant.domain_dns_cname_optional', { target: d.dns_guide.domain_target }) }}
            </p>
          </div>
          <div v-if="d.last_check_error" class="domain-err">{{ d.last_check_error }}</div>
          <div v-if="isOwner" class="domain-actions">
            <el-button v-if="d.status !== 'active'" link type="primary" :loading="domainBusy" @click="verifyDomain(d)">{{ $t('tenant.domain_verify') }}</el-button>
            <el-button v-else link type="primary" :loading="domainBusy" @click="setPrimaryDomain(d)">{{ $t('tenant.domain_set_primary') }}</el-button>
            <el-button v-if="d.status !== 'disabled'" link :loading="domainBusy" @click="disableDomain(d)">{{ $t('tenant.domain_disable') }}</el-button>
            <el-button link type="danger" :loading="domainBusy" @click="removeDomain(d)">{{ $t('common.delete') }}</el-button>
          </div>
        </div>
        <el-empty v-if="!detailDomains.length" :description="$t('common.no_data')" :image-size="48" />
      </div>
      </div>
      <div class="drawer-actions">
        <el-button type="primary" @click="openBackups(detailRow)">{{ $t('tenant.op_backups') }}</el-button>
        <el-button @click="openOverview(detailRow)">{{ $t('tenant.op_overview') }}</el-button>
        <el-button @click="openTimeline(detailRow)">{{ $t('tenant.op_timeline') }}</el-button>
        <el-button @click="goTenantSystemLogs(detailRow)">{{ $t('tenant.op_system_logs') }}</el-button>
        <el-button @click="copyLoginLink(detailRow)">{{ $t('tenant.op_login_link') }}</el-button>
      </div>
      <div v-loading="systemLogSummaryLoading" class="system-log-summary">
        <div class="system-log-summary__title">{{ $t('tenant.system_log_summary') }}</div>
        <div v-if="systemLogSummary" class="system-log-summary__meta">
          <el-tag type="danger" size="small">{{ $t('tenant.system_log_errors_24h', { n: systemLogSummary.error_count_24h || 0 }) }}</el-tag>
          <el-tag type="warning" size="small">{{ $t('tenant.system_log_warnings_24h', { n: systemLogSummary.warning_count_24h || 0 }) }}</el-tag>
        </div>
        <ul v-if="systemLogSummary?.recent?.length" class="system-log-summary__list">
          <li v-for="item in systemLogSummary.recent" :key="item.id">
            <span class="system-log-summary__time">{{ item.created_at }}</span>
            <span>{{ item.module }} · {{ item.message }}</span>
          </li>
        </ul>
        <div v-else-if="!systemLogSummaryLoading" class="system-log-summary__empty">{{ $t('tenant.system_log_recent_empty') }}</div>
        <el-button link type="primary" @click="goTenantSystemLogs(detailRow)">{{ $t('tenant.system_log_view_all') }}</el-button>
      </div>
    </template>
  </el-dialog>

  <el-dialog v-model="supportVisible" :title="$t('tenant.support')" width="820px" destroy-on-close>
    <template v-if="supportRow">
      <el-descriptions :column="3" border size="small" class="mb-12">
        <el-descriptions-item :label="$t('tenant.login_success_24h')">{{ supportAudit?.login_success_24h ?? 0 }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.login_failed_24h')">{{ supportAudit?.login_failed_24h ?? 0 }}</el-descriptions-item>
        <el-descriptions-item :label="$t('tenant.operation_count_24h')">{{ supportAudit?.operation_count_24h ?? 0 }}</el-descriptions-item>
      </el-descriptions>
      <el-table v-loading="supportLoading" :data="supportAdmins" border size="small">
        <el-table-column prop="id" :label="$t('table.id')" width="70" />
        <el-table-column prop="username" :label="$t('admin.username')" width="120" />
        <el-table-column prop="nickname" :label="$t('admin.nickname')" width="120" />
        <el-table-column prop="is_2fa_bound" :label="$t('admin.is_2fa_bound')" width="90">
          <template #default="{ row }">{{ row.is_2fa_bound ? $t('common.yes') : $t('common.no') }}</template>
        </el-table-column>
        <el-table-column :label="$t('common.operation')" width="280">
          <template #default="{ row }">
            <template v-if="isOwner">
              <el-button link type="primary" @click="onResetTenantAdminPwd(row)">{{ $t('admin.reset_password') }}</el-button>
              <el-button link type="primary" @click="onUnlockTenantAdmin(row)">{{ $t('tenant.unlock_admin') }}</el-button>
              <el-button v-if="row.is_2fa_bound" link type="danger" @click="onResetTenantAdmin2FA(row)">{{ $t('admin.reset_2fa') }}</el-button>
            </template>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>
    </template>
  </el-dialog>

  <el-dialog v-model="backupsVisible" :title="$t('tenant.backup_list_title')" width="780px" destroy-on-close>
    <el-alert type="info" :closable="false" show-icon class="migrate-tip" :title="$t('tenant.backup_hint')" />
    <div v-if="backupsMeta.dir" class="backup-dir-row">
      <span>{{ $t('tenant.backup_dir') }}: {{ backupsMeta.dir }}</span>
      <el-button link type="primary" @click="copyText(backupsMeta.dir)">{{ $t('tenant.backup_copy_path') }}</el-button>
      <span class="queue-meta">{{ $t('tenant.backup_keep_label', { n: backupsMeta.keep || settingsKeep || '-' }) }}</span>
      <el-button v-if="isOwner" link type="warning" :loading="pruneLoading" @click="pruneBackups">{{ $t('tenant.backup_prune') }}</el-button>
    </div>
    <el-table v-loading="backupsLoading" :data="backupsList" size="small" empty-text="">
      <el-table-column prop="name" :label="$t('tenant.backup_name')" min-width="180" />
      <el-table-column prop="size" :label="$t('tenant.backup_size')" width="100">
        <template #default="{ row }">{{ formatSize(row.size) }}</template>
      </el-table-column>
      <el-table-column prop="mod_time" :label="$t('tenant.backup_time')" width="170" />
      <el-table-column :label="$t('common.operation')" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="copyText(row.path)">{{ $t('tenant.backup_copy_path') }}</el-button>
          <el-button link type="primary" :loading="downloadingName === row.name" @click="downloadBackup(row)">{{ $t('tenant.backup_download') }}</el-button>
          <el-button v-if="isOwner" link type="danger" :disabled="isBusy(backupsRow)" @click="restoreBackup(row)">{{ $t('tenant.op_restore') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!backupsLoading && backupsList.length === 0" :description="$t('tenant.backup_empty')" />
  </el-dialog>

  <el-dialog v-model="overviewVisible" :title="$t('tenant.overview_title')" width="520px" align-center destroy-on-close>
    <el-descriptions v-if="overviewData" :column="1" border size="small">
      <el-descriptions-item :label="$t('tenant.database')">{{ overviewData.database }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.driver')">{{ overviewData.driver }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.overview_ping')">{{ overviewData.ping_ok ? 'OK' : 'FAIL' }} / {{ overviewData.ping_ms }}ms</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.overview_tables')">{{ overviewData.table_count }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.overview_size')">{{ formatSize(overviewData.database_bytes) }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.overview_admins')">{{ overviewData.admins_count }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.overview_migrations')">{{ overviewData.migrations_count }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.storage_used')">{{ formatQuota(quotaData?.storage_used_bytes, quotaData?.storage_limit_bytes) }}</el-descriptions-item>
      <el-descriptions-item :label="$t('tenant.storage_mode')">{{ storageModeLabel(overviewTenant) }}</el-descriptions-item>
      <el-descriptions-item v-if="overviewTenant?.storage_mode === 'custom'" :label="$t('tenant.storage_bucket')">{{ overviewTenant.storage_bucket || '-' }}</el-descriptions-item>
      <el-descriptions-item v-if="overviewData.error" :label="$t('tenant.op_message')">{{ overviewData.error }}</el-descriptions-item>
    </el-descriptions>
  </el-dialog>

  <el-dialog
    v-model="timelineVisible"
    :title="timelineTitle"
    width="720px"
    align-center
    destroy-on-close
  >
    <div v-loading="timelineLoading" class="timeline-scroll-body">
      <el-timeline v-if="opLogs.length">
        <el-timeline-item v-for="item in opLogs" :key="item.id" :timestamp="item.finished_at || item.started_at || item.created_at" placement="top">
          <div>{{ item.op }} / {{ item.status }}</div>
          <div v-if="item.operator_name" class="op-meta">{{ $t('tenant_op_log.operator') }}: {{ item.operator_name }}</div>
          <div v-if="item.batch_id" class="op-meta">{{ $t('tenant_op_log.batch_id') }}: {{ item.batch_id }}</div>
          <div class="op-meta">{{ item.message || '—' }}</div>
        </el-timeline-item>
      </el-timeline>
      <el-empty v-else-if="!timelineLoading" :description="$t('tenant.timeline_empty')" />
    </div>
    <template #footer>
      <el-button @click="goTenantOpLogs">{{ $t('tenant.timeline_view_all') }}</el-button>
      <el-button type="primary" @click="timelineVisible = false">{{ $t('common.close') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="deleteVisible" :title="$t('tenant.op_delete')" width="480px" destroy-on-close>
    <p>{{ $t('tenant.delete_confirm_hint', { code: deleteRow?.code || '' }) }}</p>
    <el-form label-width="120px">
      <el-form-item :label="$t('tenant.confirm_code')">
        <el-input v-model="deleteConfirm" :placeholder="deleteRow?.code || ''" />
      </el-form-item>
      <el-form-item :label="$t('tenant.drop_database')">
        <el-switch v-model="deleteDropDb" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="deleteVisible = false">{{ $t('common.cancel') }}</el-button>
      <el-button type="danger" :loading="deleteLoading" @click="submitDelete">{{ $t('common.confirm') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="purgeVisible" :title="$t('tenant.op_purge_retry')" width="480px" destroy-on-close>
    <p>{{ $t('tenant.purge_retry_hint', { code: purgeRow?.code || '' }) }}</p>
    <el-alert
      v-if="purgeRow?.last_op === 'purge'"
      :type="purgeRow.last_op_status === 'failed' ? 'error' : 'info'"
      :closable="false"
      show-icon
      class="migrate-tip"
      :title="`${purgeRow.last_op} / ${purgeRow.last_op_status || '—'}`"
      :description="purgeRow.last_op_message || ''"
    />
    <el-form label-width="120px">
      <el-form-item :label="$t('tenant.purge_objects')">
        <el-switch v-model="purgeObjects" />
      </el-form-item>
      <el-form-item :label="$t('tenant.purge_backups')">
        <el-switch v-model="purgeBackups" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="purgeVisible = false">{{ $t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="purgeLoading" @click="submitPurge">{{ $t('common.confirm') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="forceDeleteVisible" :title="$t('tenant.op_force_delete')" width="480px" destroy-on-close>
    <p>{{ $t('tenant.force_delete_hint', { code: forceDeleteRow?.code || '' }) }}</p>
    <el-form label-width="120px">
      <el-form-item :label="$t('tenant.confirm_code')">
        <el-input v-model="forceDeleteConfirm" :placeholder="forceDeleteRow?.code || ''" />
      </el-form-item>
      <el-form-item :label="$t('tenant.purge_objects')">
        <el-switch v-model="forcePurgeObjects" />
        <div class="form-tip">{{ $t('tenant.purge_objects_tip') }}</div>
      </el-form-item>
      <el-form-item :label="$t('tenant.purge_backups')">
        <el-switch v-model="forcePurgeBackups" />
        <div class="form-tip">{{ $t('tenant.purge_backups_tip') }}</div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="forceDeleteVisible = false">{{ $t('common.cancel') }}</el-button>
      <el-button type="danger" :loading="forceDeleteLoading" @click="submitForceDelete">{{ $t('common.confirm') }}</el-button>
    </template>
  </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import {
  backupPlatformTenant,
  onboardPlatformTenant,
  deletePlatformTenant,
  forceDeletePlatformTenant,
  downloadPlatformTenantBackup,
  exportPlatformTenants,
  getPlatformTenantList,
  getPlatformTenantLoginLinks,
  getPlatformTenantDomains,
  createPlatformTenantDomain,
  verifyPlatformTenantDomain,
  setPrimaryPlatformTenantDomain,
  disablePlatformTenantDomain,
  deletePlatformTenantDomain,
  getPlatformTenantOpLogs,
  getPlatformTenantSystemLogSummary,
  getPlatformTenantOpsSummary,
  getPlatformTenantOverview,
  getPlatformTenantSettings,
  getPlatformTenantAdmins,
  getPlatformTenantAuditSummary,
  resetPlatformTenantAdminPassword,
  unlockPlatformTenantAdmin,
  resetPlatformTenantAdmin2FA,
  listPlatformTenantBackups,
  migratePlatformTenant,
  migratePlatformTenantBatch,
  opsPlatformTenantBatch,
  pingPlatformTenant,
  platformHealth,
  prunePlatformTenantBackups,
  purgePlatformTenant,
  restorePlatformTenant,
  seedPlatformTenant,
  undeletePlatformTenant,
  updatePlatformTenant,
  updatePlatformTenantMaintenance,
  updatePlatformTenantStatus
} from '@/api/platform'
import { isPlatformOwner } from '@/utils/platformRequest'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const isOwner = computed(() => isPlatformOwner())
const listPageRef = ref(null)
const formRef = ref(null)
const saving = ref(false)
const editingId = ref(null)
const passwordTouched = ref(false)

// Browsers treat username/password fields as login forms; readonly until focus blocks autofill.
const unlockCredentialAutofill = (e) => {
  const el = e?.target
  if (el?.hasAttribute?.('readonly')) {
    el.removeAttribute('readonly')
  }
}
const health = ref(null)
const migrateVisible = ref(false)
const migrateRow = ref(null)
const withSeed = ref(true)
const opsSummary = ref(null)
const batchLoading = ref(false)
const selectedRows = ref([])
const detailVisible = ref(false)
const detailRow = ref(null)
const systemLogSummary = ref(null)
const systemLogSummaryLoading = ref(false)
const detailDomains = ref([])
const domainHost = ref('')
const domainSslMode = ref('edge')
const domainBusy = ref(false)
const backupsVisible = ref(false)
const backupsRow = ref(null)
const backupsList = ref([])
const backupsMeta = reactive({ dir: '', last: '', keep: 0 })
const backupsLoading = ref(false)
const downloadingName = ref('')
const pruneLoading = ref(false)
const queueInfo = ref(null)
const settingsKeep = ref(0)
const overviewVisible = ref(false)
const overviewData = ref(null)
const overviewTenant = ref(null)
const quotaData = ref(null)
const timelineVisible = ref(false)
const timelineRow = ref(null)
const timelineLoading = ref(false)
const opLogs = ref([])
const timelineTitle = computed(() => {
  const code = timelineRow.value?.code || timelineRow.value?.name || ''
  return code ? `${t('tenant.timeline_title')} · ${code}` : t('tenant.timeline_title')
})
const deleteVisible = ref(false)
const deleteRow = ref(null)
const deleteConfirm = ref('')
const deleteDropDb = ref(false)
const deleteLoading = ref(false)
const purgeVisible = ref(false)
const purgeRow = ref(null)
const purgeObjects = ref(true)
const purgeBackups = ref(true)
const purgeLoading = ref(false)
const forceDeleteVisible = ref(false)
const forceDeleteRow = ref(null)
const forceDeleteConfirm = ref('')
const forcePurgeObjects = ref(true)
const forcePurgeBackups = ref(true)
const forceDeleteLoading = ref(false)
const retentionDays = ref(30)
let pollTimer = null

const healthDesc = computed(() => {
  const h = health.value
  const cli = t('platform.cli_ops_hint')
  if (!h) return cli
  const db = h.database_ok ? t('platform.health_db_ok') : t('platform.health_db_bad')
  const tenants = t('platform.health_tenants', {
    active: h.tenants?.active ?? 0,
    total: h.tenants?.total ?? 0
  })
  const q = h.queue?.connection ? ` · queue=${h.queue.connection}` : ''
  return `${t('platform.health_driver')}: ${h.driver} · ${db} · ${tenants}${q} · ${cli}`
})

const refreshOpsSummary = async () => {
  try {
    const res = await getPlatformTenantOpsSummary()
    opsSummary.value = res?.data?.summary || null
  } catch {
    // ignore
  }
}

const refreshSettings = async () => {
  try {
    const res = await getPlatformTenantSettings()
    settingsKeep.value = res?.data?.backup_keep || 0
    retentionDays.value = res?.data?.deleted_retention_days ?? 30
    queueInfo.value = res?.data?.queue || null
  } catch {
    // ignore
  }
}

const initialSearchForm = {
  code: '',
  name: '',
  status: '',
  provision_status: '',
  schema_status: '',
  last_op: '',
  last_op_status: '',
  maintenance: '',
  domain_host: '',
  domain_status: '',
  health_status: '',
  trashed: ''
}
const onboardResultVisible = ref(false)
const onboardLoginUrl = ref('')

const {
  pagination,
  tableData,
  loading,
  searchForm,
  dialogVisible,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange
} = useStandardListPage({
  fetchApi: getPlatformTenantList,
  initialSearchForm,
  defaultSort: 'id:desc'
})

onMounted(async () => {
  const schemaFromQuery = String(route.query.schema_status || '').trim()
  if (schemaFromQuery) {
    searchForm.schema_status = schemaFromQuery
  }
  const maintenanceFromQuery = String(route.query.maintenance || '').trim()
  if (maintenanceFromQuery) {
    searchForm.maintenance = maintenanceFromQuery
  }
  const domainFromQuery = String(route.query.domain_status || '').trim()
  if (domainFromQuery) {
    searchForm.domain_status = domainFromQuery
  }
  const healthFromQuery = String(route.query.health_status || '').trim()
  if (healthFromQuery) {
    searchForm.health_status = healthFromQuery
  }
  try {
    const res = await platformHealth()
    health.value = res?.data || null
    if (res?.data?.queue) queueInfo.value = res.data.queue
    if (res?.data?.backup_keep) settingsKeep.value = res.data.backup_keep
  } catch {
    // ignore — list still usable
  }
  await refreshOpsSummary()
  await refreshSettings()
  if (schemaFromQuery || maintenanceFromQuery || domainFromQuery || healthFromQuery) {
    await loadData()
  }
})

const isRecycleView = computed(() => searchForm.trashed === 'only')

const recycleBannerDesc = computed(() => {
  const base = t('tenant.recycle_banner_desc')
  if (retentionDays.value > 0) {
    return `${base} ${t('tenant.retention_days', { n: retentionDays.value })}`
  }
  return base
})

const setListMode = (mode) => {
  const next = mode === 'recycle' ? 'only' : ''
  if (searchForm.trashed === next) {
    loadData()
    return
  }
  searchForm.trashed = next
  handleSearch()
}

const onListModeChange = (mode) => {
  setListMode(mode)
}

const onListReset = () => {
  const keep = searchForm.trashed
  handleReset()
  searchForm.trashed = keep
  handleSearch()
}

const recycleOpTag = (row) => {
  if (!isRecycleView.value) return null
  const op = row?.last_op
  const st = row?.last_op_status
  if (op === 'purge') {
    if (st === 'queued' || st === 'running') return { type: 'warning', label: t('tenant.purge_status_running') }
    if (st === 'failed') return { type: 'danger', label: t('tenant.purge_status_failed') }
    if (st === 'success') return { type: 'success', label: t('tenant.purge_status_done') }
  }
  if (st === 'queued' || st === 'running') return { type: 'warning', label: t('tenant.op_busy') }
  return null
}

const onRecycleMore = (cmd, row) => {
  if (cmd === 'timeline') openTimeline(row)
  if (cmd === 'purge') openPurge(row)
}

const isBusy = (row) => {
  if (row?.provision_status === 'migrating') return true
  return row?.last_op_status === 'queued' || row?.last_op_status === 'running'
}

watch(
  tableData,
  (rows) => {
    const busy = Array.isArray(rows) && rows.some(isBusy)
    if (busy && !pollTimer) {
      pollTimer = window.setInterval(() => {
        loadData()
        refreshOpsSummary()
      }, 3000)
    } else if (!busy && pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  },
  { deep: true }
)

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

const provisionLabel = (status) => {
  switch (status) {
    case 'ready':
      return t('tenant.provision_ready')
    case 'migrating':
      return t('tenant.provision_migrating')
    case 'failed':
      return t('tenant.provision_failed')
    default:
      return t('tenant.provision_pending')
  }
}

const provisionTagType = (status) => {
  switch (status) {
    case 'ready':
      return 'success'
    case 'migrating':
      return 'warning'
    case 'failed':
      return 'danger'
    default:
      return 'info'
  }
}

const schemaLabel = (status) => {
  switch (status) {
    case 'aligned':
      return t('tenant.schema_aligned')
    case 'behind':
      return t('tenant.schema_behind')
    case 'failed':
      return t('tenant.schema_failed')
    case 'running':
      return t('tenant.schema_running')
    case 'unknown':
      return t('tenant.schema_unknown')
    default:
      return status || '—'
  }
}

const schemaTagType = (status) => {
  switch (status) {
    case 'aligned':
      return 'success'
    case 'behind':
      return 'warning'
    case 'failed':
      return 'danger'
    case 'running':
      return 'warning'
    default:
      return 'info'
  }
}

const domainLabel = (status) => {
  switch (status) {
    case 'unbound':
      return t('tenant.domain_unbound')
    case 'pending':
      return t('tenant.domain_pending')
    case 'active':
      return t('tenant.domain_active')
    case 'verify_failed':
      return t('tenant.domain_verify_failed')
    case 'disabled':
      return t('tenant.domain_disabled')
    default:
      return status || t('tenant.domain_unbound')
  }
}

const domainTagType = (status) => {
  switch (status) {
    case 'active':
      return 'success'
    case 'pending':
      return 'warning'
    case 'verify_failed':
      return 'danger'
    default:
      return 'info'
  }
}

const healthLabel = (status) => {
  switch (status) {
    case 'ok':
      return t('tenant.health_ok')
    case 'warn':
      return t('tenant.health_warn')
    case 'fail':
      return t('tenant.health_fail')
    default:
      return t('tenant.health_unknown')
  }
}

const healthTagType = (status) => {
  switch (status) {
    case 'ok':
      return 'success'
    case 'warn':
      return 'warning'
    case 'fail':
      return 'danger'
    default:
      return 'info'
  }
}

const schemaStatusTip = (row) => {
  if (!row) return ''
  const parts = []
  if (row.schema_migration_count != null || row.expected_migration_count != null) {
    parts.push(`${row.schema_migration_count ?? 0}/${row.expected_migration_count ?? '?'}`)
  }
  if (row.last_migrate_error) parts.push(row.last_migrate_error)
  return parts.join(' · ')
}

const searchFields = computed(() => {
  const base = [
    { prop: 'code', label: t('tenant.code'), type: 'input', width: '180px' },
    { prop: 'name', label: t('tenant.name'), type: 'input', width: '180px' }
  ]
  if (isRecycleView.value) return base
  return [
    ...base,
    {
      prop: 'schema_status',
      label: t('tenant.schema_status'),
      type: 'select',
      width: '160px',
      options: [
        { label: t('tenant.schema_aligned'), value: 'aligned' },
        { label: t('tenant.schema_behind'), value: 'behind' },
        { label: t('tenant.schema_failed'), value: 'failed' },
        { label: t('tenant.schema_running'), value: 'running' },
        { label: t('tenant.schema_unknown'), value: 'unknown' }
      ]
    },
    {
      prop: 'provision_status',
      label: t('tenant.provision_status_filter'),
      type: 'select',
      width: '160px',
      options: [
        { label: t('tenant.provision_pending'), value: 'pending' },
        { label: t('tenant.provision_migrating'), value: 'migrating' },
        { label: t('tenant.provision_ready'), value: 'ready' },
        { label: t('tenant.provision_failed'), value: 'failed' }
      ]
    },
    {
      prop: 'last_op',
      label: t('tenant.last_op_filter'),
      type: 'select',
      width: '140px',
      options: [
        { label: t('tenant.op_migrate'), value: 'migrate' },
        { label: t('tenant.op_seed'), value: 'seed' },
        { label: t('tenant.op_backup'), value: 'backup' },
        { label: t('tenant.op_restore'), value: 'restore' },
        { label: t('tenant.op_purge'), value: 'purge' }
      ]
    },
    {
      prop: 'last_op_status',
      label: t('tenant.last_op_status_filter'),
      type: 'select',
      width: '140px',
      options: [
        { label: t('tenant.op_status_failed'), value: 'failed' },
        { label: t('tenant.op_status_success'), value: 'success' },
        { label: t('tenant.op_status_running'), value: 'running' },
        { label: t('tenant.op_status_queued'), value: 'queued' },
        { label: t('tenant.op_status_idle'), value: 'idle' }
      ]
    },
    {
      prop: 'status',
      label: t('common.status'),
      type: 'select',
      width: '140px',
      options: [
        { label: t('common.enabled'), value: '1' },
        { label: t('common.disabled'), value: '0' }
      ]
    },
    {
      prop: 'maintenance',
      label: t('tenant.maintenance'),
      type: 'select',
      width: '140px',
      options: [
        { label: t('tenant.maintenance_on'), value: '1' },
        { label: t('tenant.maintenance_off'), value: '0' }
      ]
    },
    { prop: 'domain_host', label: t('tenant.domain_host'), type: 'input', width: '180px' },
    {
      prop: 'domain_status',
      label: t('tenant.domain_status'),
      type: 'select',
      width: '150px',
      options: [
        { label: t('tenant.domain_unbound'), value: 'unbound' },
        { label: t('tenant.domain_pending'), value: 'pending' },
        { label: t('tenant.domain_active'), value: 'active' },
        { label: t('tenant.domain_verify_failed'), value: 'verify_failed' }
      ]
    },
    {
      prop: 'health_status',
      label: t('tenant.health_status'),
      type: 'select',
      width: '140px',
      options: [
        { label: t('tenant.health_ok'), value: 'ok' },
        { label: t('tenant.health_warn'), value: 'warn' },
        { label: t('tenant.health_fail'), value: 'fail' },
        { label: t('tenant.health_unknown'), value: 'unknown' }
      ]
    }
  ]
})

const retryFailedMigrates = async () => {
  try {
    await ElMessageBox.confirm(t('tenant.retry_failed_migrate_confirm'), t('tenant.retry_failed_migrate'), {
      type: 'warning'
    })
  } catch {
    return
  }
  batchLoading.value = true
  try {
    const res = await migratePlatformTenantBatch({ provision_status: 'failed', with_seed: false })
    const n = res?.data?.queued_count ?? 0
    const batch = res?.data?.batch_id ? t('tenant.batch_id_suffix', { id: res.data.batch_id }) : ''
    ElMessage.success(t('tenant.batch_queued', { n, batch }))
    await loadData()
    await refreshOpsSummary()
  } catch (e) {
    console.error(e)
  } finally {
    batchLoading.value = false
  }
}

const retryFailedSeeds = async () => {
  try {
    await ElMessageBox.confirm(t('tenant.retry_failed_seed_confirm'), t('tenant.retry_failed_seed'), {
      type: 'warning'
    })
  } catch {
    return
  }
  batchLoading.value = true
  try {
    const res = await opsPlatformTenantBatch({
      op: 'seed',
      last_op: 'seed',
      last_op_status: 'failed'
    })
    const n = res?.data?.queued_count ?? 0
    const batch = res?.data?.batch_id ? t('tenant.batch_id_suffix', { id: res.data.batch_id }) : ''
    ElMessage.success(t('tenant.batch_queued', { n, batch }))
    await loadData()
    await refreshOpsSummary()
  } catch (e) {
    console.error(e)
  } finally {
    batchLoading.value = false
  }
}

const tableColumns = computed(() => {
  const cols = []
  if (!isRecycleView.value) {
    cols.push({ type: 'checkbox', width: 52, fixed: 'left', key: 'checkbox' })
  }
  cols.push(
    { field: 'id', title: t('table.id'), width: 70, sortable: true, key: 'id' },
    { field: 'code', title: t('tenant.code'), width: 110, key: 'code' },
    { field: 'name', title: t('tenant.name'), width: 120, key: 'name' },
    { field: 'driver', title: t('tenant.driver'), width: 90, key: 'driver' },
    { field: 'database', title: t('tenant.database'), width: 140, key: 'database' },
    { field: 'provision_status', title: t('tenant.provision_status'), width: 110, slot: 'provision_status', key: 'provision_status' },
    { field: 'schema_status', title: t('tenant.schema_status'), width: 110, slot: 'schema_status', key: 'schema_status' },
    { field: 'domain_status', title: t('tenant.domain_status'), width: 110, slot: 'domain_status', key: 'domain_status' },
    { field: 'health_status', title: t('tenant.health_status'), width: 120, slot: 'health_status', key: 'health_status' },
    { field: 'last_op', title: t('tenant.last_op'), width: 160, slot: 'last_op', key: 'last_op' }
  )
  if (isRecycleView.value) {
    cols.push({ field: 'deleted_at', title: t('tenant.deleted_at'), width: 170, key: 'deleted_at' })
  } else {
    cols.push(
      { field: 'last_backup_path', title: t('tenant.backup_path'), minWidth: 180, slot: 'last_backup_path', key: 'last_backup_path' },
      { field: 'status', title: t('common.status'), width: 90, slot: 'status', key: 'status' },
      { field: 'maintenance', title: t('tenant.maintenance'), width: 100, slot: 'maintenance', key: 'maintenance' },
      { field: 'created_at', title: t('table.created_at'), key: 'created_at' }
    )
  }
  cols.push({ field: 'actions', title: t('common.operation'), width: isRecycleView.value ? 240 : 320, slot: 'actions', key: 'actions' })
  return cols
})

const handleSelectionChange = (rows) => {
  selectedRows.value = Array.isArray(rows) ? rows : []
}

const form = reactive({
  code: '',
  name: '',
  driver: 'mysql',
  isolation: 'database',
  database: '',
  schema: '',
  host: '',
  port: 0,
  username: '',
  password: '',
  has_password: false,
  skip_create: false,
  with_migrate: true,
  with_seed: true,
  domain_host: '',
  domain_ssl_mode: 'edge',
  storage_limit_mb: 0,
  storage_mode: 'shared',
  storage_driver: 's3',
  storage_key: '',
  storage_secret: '',
  storage_has_secret: false,
  storage_region: '',
  storage_bucket: '',
  storage_url: '',
  storage_endpoint: '',
  storage_use_path_style: false,
  storage_ssl: true
})

const storageSecretTouched = ref(false)

const MB = 1024 * 1024
const mbToBytes = (m) => Math.round((Number(m) || 0) * MB)
const bytesToMb = (b) => {
  const n = Number(b) || 0
  if (n <= 0) return 0
  return Math.round(n / MB)
}
const formatQuota = (used, limit) => {
  const u = formatSize(used)
  const lim = Number(limit) || 0
  if (lim <= 0) return `${u} / ∞`
  return `${u} / ${formatSize(lim)}`
}
const storageModeLabel = (row) => {
  if (!row) return '-'
  return row.storage_mode === 'custom' ? t('tenant.storage_mode_custom') : t('tenant.storage_mode_shared')
}
const applyStorageFormFromRow = (row) => {
  form.storage_mode = row?.storage_mode === 'custom' ? 'custom' : 'shared'
  form.storage_driver = row?.storage_driver || 's3'
  form.storage_key = row?.storage_key || ''
  form.storage_secret = ''
  form.storage_has_secret = !!row?.storage_has_secret
  form.storage_region = row?.storage_region || ''
  form.storage_bucket = row?.storage_bucket || ''
  form.storage_url = row?.storage_url || ''
  form.storage_endpoint = row?.storage_endpoint || ''
  form.storage_use_path_style = !!row?.storage_use_path_style
  form.storage_ssl = row?.storage_ssl !== false
  storageSecretTouched.value = false
}
const buildStoragePayload = () => {
  const payload = {
    storage_mode: form.storage_mode || 'shared',
    storage_driver: form.storage_driver || '',
    storage_key: form.storage_key || '',
    storage_region: form.storage_region || '',
    storage_bucket: form.storage_bucket || '',
    storage_url: form.storage_url || '',
    storage_endpoint: form.storage_endpoint || '',
    storage_use_path_style: !!form.storage_use_path_style,
    storage_ssl: !!form.storage_ssl
  }
  if (storageSecretTouched.value && form.storage_secret) {
    payload.storage_secret = form.storage_secret
  }
  return payload
}

const formRules = computed(() => {
  const rules = {
    name: [{ required: true, message: t('tenant.name_required'), trigger: 'blur' }]
  }
  if (!editingId.value) {
    rules.code = [{ required: true, message: t('tenant.code_required'), trigger: 'blur' }]
  }
  return rules
})

const openCreate = () => {
  editingId.value = null
  passwordTouched.value = false
  storageSecretTouched.value = false
  dialogVisible.value = true
}

const openEdit = (row) => {
  editingId.value = row.id
  passwordTouched.value = false
  form.name = row.name || ''
  form.host = row.host || ''
  form.port = Number(row.port) || 0
  form.username = row.username || ''
  form.password = ''
  form.database = row.database || ''
  form.schema = row.schema || ''
  form.has_password = !!row.has_password
  form.storage_limit_mb = bytesToMb(row.storage_limit_bytes)
  applyStorageFormFromRow(row)
  dialogVisible.value = true
}

const openMigrate = (row) => {
  migrateRow.value = row
  withSeed.value = true
  migrateVisible.value = true
}

const resetForm = () => {
  editingId.value = null
  passwordTouched.value = false
  form.code = ''
  form.name = ''
  form.driver = 'mysql'
  form.isolation = 'database'
  form.database = ''
  form.schema = ''
  form.host = ''
  form.port = 0
  form.username = ''
  form.password = ''
  form.has_password = false
  form.skip_create = false
  form.with_migrate = true
  form.with_seed = true
  form.domain_host = ''
  form.domain_ssl_mode = 'edge'
  form.storage_limit_mb = 0
  applyStorageFormFromRow(null)
}

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (editingId.value) {
        const payload = {
          name: form.name,
          host: form.host,
          port: form.port || 0,
          username: form.username,
          database: form.database,
          schema: form.schema,
          storage_limit_bytes: mbToBytes(form.storage_limit_mb),
          ...buildStoragePayload()
        }
        // Ignore browser-autofilled password unless the user edited the field.
        if (passwordTouched.value && form.password) payload.password = form.password
        await updatePlatformTenant(editingId.value, payload)
        ElMessage.success(t('common.update_success'))
      } else {
        const res = await onboardPlatformTenant({
          code: form.code,
          name: form.name,
          driver: form.driver,
          isolation: form.isolation,
          database: form.database || undefined,
          schema: form.schema || undefined,
          host: form.host || undefined,
          port: form.port || undefined,
          username: form.username || undefined,
          password: passwordTouched.value && form.password ? form.password : undefined,
          skip_create: form.skip_create,
          with_migrate: !!form.with_migrate,
          with_seed: !!form.with_migrate && !!form.with_seed,
          domain_host: form.domain_host || undefined,
          domain_ssl_mode: form.domain_ssl_mode || 'edge',
          storage_limit_bytes: mbToBytes(form.storage_limit_mb) || undefined,
          ...buildStoragePayload()
        })
        const links = res?.data?.login_links || {}
        onboardLoginUrl.value =
          links.primary_custom_url || links.subdomain_url || links.query_url || ''
        if (res?.data?.domain_error) {
          ElMessage.warning(t('tenant.onboard_domain_failed', { msg: res.data.domain_error }))
        } else {
          ElMessage.success(t('tenant.onboard_success'))
        }
        if (onboardLoginUrl.value) {
          onboardResultVisible.value = true
        }
      }
      dialogVisible.value = false
      loadData()
      refreshOpsSummary()
    } catch (error) {
      if (!error?.__handled) {
        ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
      }
    } finally {
      saving.value = false
    }
  })
}

const onToggleStatus = async (row, enabled) => {
  const status = enabled ? 1 : 0
  try {
    await updatePlatformTenantStatus(row.id, status)
    row.status = status
    ElMessage.success(t('common.update_success'))
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
    loadData()
  }
}

const onToggleMaintenance = async (row, enabled) => {
  try {
    await ElMessageBox.confirm(
      enabled ? t('tenant.maintenance_confirm_on') : t('tenant.maintenance_confirm_off'),
      t('tenant.maintenance'),
      { type: 'warning' }
    )
  } catch {
    loadData()
    return
  }
  try {
    await updatePlatformTenantMaintenance(row.id, { maintenance: !!enabled })
    row.maintenance = !!enabled
    if (!enabled) row.maintenance_message = ''
    ElMessage.success(t('common.update_success'))
    refreshOpsSummary()
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
    loadData()
  }
}

const onPing = async (row) => {
  try {
    const res = await pingPlatformTenant(row.id)
    const ping = res?.data?.ping
    if (ping?.ok) {
      ElMessage.success(t('tenant.ping_ok_detail', { ms: ping.latency_ms ?? '-', host: ping.host || '-', db: ping.database || '-' }))
    } else {
      ElMessage.error(ping?.error || t('tenant.ping_fail'))
    }
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const onMoreCommand = (cmd, row) => {
  switch (cmd) {
    case 'seed':
      onSeed(row)
      break
    case 'backup':
      onBackup(row)
      break
    case 'backups':
      openBackups(row)
      break
    case 'overview':
      openOverview(row)
      break
    case 'timeline':
      openTimeline(row)
      break
    case 'login':
      copyLoginLink(row)
      break
    case 'delete':
      openDelete(row)
      break
    default:
      break
  }
}

const batchMigrateSelected = async () => {
  const ids = selectedRows.value.map((r) => r.id).filter(Boolean)
  if (!ids.length) {
    ElMessage.warning(t('tenant.batch_need_selection'))
    return
  }
  try {
    await ElMessageBox.confirm(t('tenant.batch_migrate_confirm', { n: ids.length }), {
      type: 'warning',
      title: t('tenant.batch_migrate')
    })
  } catch {
    return
  }
  batchLoading.value = true
  try {
    const res = await opsPlatformTenantBatch({ op: 'migrate', ids, with_seed: false })
    const n = res?.data?.queued_count ?? 0
    const batch = res?.data?.batch_id ? t('tenant.batch_id_suffix', { id: res.data.batch_id }) : ''
    ElMessage.success(t('tenant.batch_queued', { n, batch }))
    selectedRows.value = []
    await loadData()
    await refreshOpsSummary()
  } catch (e) {
    console.error(e)
  } finally {
    batchLoading.value = false
  }
}

const batchSeedSelected = async () => {
  const ids = selectedRows.value.map((r) => r.id).filter(Boolean)
  if (!ids.length) {
    ElMessage.warning(t('tenant.batch_need_selection'))
    return
  }
  try {
    await ElMessageBox.confirm(t('tenant.batch_seed_confirm', { n: ids.length }), { type: 'warning' })
  } catch {
    return
  }
  batchLoading.value = true
  try {
    const res = await opsPlatformTenantBatch({ op: 'seed', ids })
    const n = res?.data?.queued_count ?? 0
    const batch = res?.data?.batch_id ? t('tenant.batch_id_suffix', { id: res.data.batch_id }) : ''
    ElMessage.success(t('tenant.batch_queued', { n, batch }))
    selectedRows.value = []
    await loadData()
    await refreshOpsSummary()
  } catch (e) {
    console.error(e)
  } finally {
    batchLoading.value = false
  }
}

const batchBackupSelected = async () => {
  const ids = selectedRows.value.map((r) => r.id).filter(Boolean)
  if (!ids.length) {
    ElMessage.warning(t('tenant.batch_need_selection'))
    return
  }
  try {
    await ElMessageBox.confirm(t('tenant.batch_backup_confirm', { n: ids.length }), { type: 'warning' })
  } catch {
    return
  }
  batchLoading.value = true
  try {
    const res = await opsPlatformTenantBatch({ op: 'backup', ids })
    const n = res?.data?.queued_count ?? 0
    const batch = res?.data?.batch_id ? t('tenant.batch_id_suffix', { id: res.data.batch_id }) : ''
    ElMessage.success(t('tenant.batch_queued', { n, batch }))
    selectedRows.value = []
    await loadData()
    await refreshOpsSummary()
  } catch (e) {
    console.error(e)
  } finally {
    batchLoading.value = false
  }
}

const exportCsv = async () => {
  try {
    const blob = await exportPlatformTenants({ ...searchForm })
    const url = window.URL.createObjectURL(blob instanceof Blob ? blob : new Blob([blob]))
    const a = document.createElement('a')
    a.href = url
    a.download = `tenants_${Date.now()}.csv`
    a.click()
    window.URL.revokeObjectURL(url)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const openOverview = async (row) => {
  overviewVisible.value = true
  overviewData.value = null
  overviewTenant.value = row || null
  quotaData.value = null
  try {
    const res = await getPlatformTenantOverview(row.id)
    overviewData.value = res?.data?.overview || null
    quotaData.value = res?.data?.quota || null
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const openTimeline = async (row) => {
  timelineRow.value = row
  timelineVisible.value = true
  opLogs.value = []
  timelineLoading.value = true
  try {
    const res = await getPlatformTenantOpLogs(row.id, { limit: 20 })
    opLogs.value = res?.data?.list || []
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    timelineLoading.value = false
  }
}

const goTenantOpLogs = () => {
  const code = timelineRow.value?.code
  timelineVisible.value = false
  router.push(code ? { path: '/platform/tenant-op-logs', query: { code } } : '/platform/tenant-op-logs')
}

const copyLoginLink = async (row) => {
  try {
    const res = await getPlatformTenantLoginLinks(row.id)
    const links = res?.data?.links
    const text =
      links?.primary_custom_url || links?.subdomain_url || links?.query_url || `/?tenant_code=${row.code}`
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('tenant.login_link_copied', { hint: links?.hint || '' }))
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const loadDetailDomains = async (row) => {
  detailDomains.value = []
  if (!row?.id) return
  try {
    const res = await getPlatformTenantDomains(row.id)
    detailDomains.value = res?.data?.list || []
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const addDomain = async () => {
  if (!detailRow.value || !domainHost.value.trim()) return
  domainBusy.value = true
  try {
    await createPlatformTenantDomain(detailRow.value.id, {
      host: domainHost.value.trim(),
      ssl_mode: domainSslMode.value
    })
    domainHost.value = ''
    ElMessage.success(t('tenant.domain_add_success'))
    await loadDetailDomains(detailRow.value)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    domainBusy.value = false
  }
}

const verifyDomain = async (d) => {
  if (!detailRow.value) return
  domainBusy.value = true
  try {
    await verifyPlatformTenantDomain(detailRow.value.id, d.id)
    ElMessage.success(t('tenant.domain_verify_success'))
    await loadDetailDomains(detailRow.value)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    domainBusy.value = false
  }
}

const setPrimaryDomain = async (d) => {
  if (!detailRow.value) return
  domainBusy.value = true
  try {
    await setPrimaryPlatformTenantDomain(detailRow.value.id, d.id)
    ElMessage.success(t('common.success'))
    await loadDetailDomains(detailRow.value)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    domainBusy.value = false
  }
}

const disableDomain = async (d) => {
  if (!detailRow.value) return
  domainBusy.value = true
  try {
    await disablePlatformTenantDomain(detailRow.value.id, d.id)
    ElMessage.success(t('common.success'))
    await loadDetailDomains(detailRow.value)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    domainBusy.value = false
  }
}

const removeDomain = async (d) => {
  if (!detailRow.value) return
  domainBusy.value = true
  try {
    await deletePlatformTenantDomain(detailRow.value.id, d.id)
    ElMessage.success(t('common.success'))
    await loadDetailDomains(detailRow.value)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    domainBusy.value = false
  }
}

const openDelete = (row) => {
  deleteRow.value = row
  deleteConfirm.value = ''
  deleteDropDb.value = false
  deleteVisible.value = true
}

const submitDelete = async () => {
  if (!deleteRow.value) return
  deleteLoading.value = true
  try {
    await deletePlatformTenant(deleteRow.value.id, {
      confirm_code: deleteConfirm.value,
      drop_database: deleteDropDb.value
    })
    ElMessage.success(t('tenant.delete_to_recycle'))
    deleteVisible.value = false
    setListMode('recycle')
    refreshOpsSummary()
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    deleteLoading.value = false
  }
}

const onUndelete = async (row) => {
  try {
    await ElMessageBox.confirm(t('tenant.undelete_confirm', { code: row.code }), { type: 'warning' })
    await undeletePlatformTenant(row.id)
    ElMessage.success(t('tenant.undelete_success'))
    setListMode('active')
    await refreshOpsSummary()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const openPurge = (row) => {
  purgeRow.value = row
  purgeObjects.value = true
  purgeBackups.value = true
  purgeVisible.value = true
}

const submitPurge = async () => {
  if (!purgeRow.value) return
  purgeLoading.value = true
  try {
    await purgePlatformTenant(purgeRow.value.id, {
      purge_objects: purgeObjects.value,
      purge_backups: purgeBackups.value
    })
    ElMessage.success(t('tenant.purge_queued'))
    purgeVisible.value = false
    await loadData()
    await refreshOpsSummary()
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    purgeLoading.value = false
  }
}

const openForceDelete = (row) => {
  forceDeleteRow.value = row
  forceDeleteConfirm.value = ''
  forcePurgeObjects.value = true
  forcePurgeBackups.value = true
  forceDeleteVisible.value = true
}

const submitForceDelete = async () => {
  if (!forceDeleteRow.value) return
  forceDeleteLoading.value = true
  try {
    const res = await forceDeletePlatformTenant(forceDeleteRow.value.id, {
      confirm_code: forceDeleteConfirm.value,
      purge_objects: forcePurgeObjects.value,
      purge_backups: forcePurgeBackups.value
    })
    if (res?.data?.force_delete_queued || res?.data?.purge_queued) {
      ElMessage.success(t('tenant.force_delete_queued'))
    } else {
      ElMessage.success(t('tenant.force_delete_success'))
    }
    forceDeleteVisible.value = false
    await loadData()
    await refreshOpsSummary()
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    forceDeleteLoading.value = false
  }
}

const restoreBackup = async (file) => {
  if (!backupsRow.value || !file?.name) return
  try {
    await ElMessageBox.confirm(t('tenant.op_restore_confirm', { name: file.name }), { type: 'warning' })
    await restorePlatformTenant(backupsRow.value.id, { backup_name: file.name })
    ElMessage.success(t('tenant.op_queued'))
    loadData()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const pruneBackups = async () => {
  if (!backupsRow.value) return
  const keep = backupsMeta.keep || settingsKeep.value || 10
  try {
    await ElMessageBox.confirm(t('tenant.backup_prune_confirm', { n: keep }), { type: 'warning' })
  } catch {
    return
  }
  pruneLoading.value = true
  try {
    const res = await prunePlatformTenantBackups(backupsRow.value.id, { keep })
    ElMessage.success(t('tenant.backup_prune_done', { n: res?.data?.removed ?? 0 }))
    await openBackups(backupsRow.value)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    pruneLoading.value = false
  }
}

const submitMigrate = async () => {
  if (!migrateRow.value) return
  try {
    await migratePlatformTenant(migrateRow.value.id, { with_seed: withSeed.value })
    ElMessage.success(t('tenant.op_queued'))
    migrateVisible.value = false
    loadData()
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const onSeed = async (row) => {
  try {
    await ElMessageBox.confirm(t('tenant.op_seed_confirm'), { type: 'warning' })
    await seedPlatformTenant(row.id)
    ElMessage.success(t('tenant.op_queued'))
    loadData()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const onBackup = async (row) => {
  try {
    await ElMessageBox.confirm(t('tenant.op_backup_confirm'), { type: 'warning' })
    await backupPlatformTenant(row.id)
    ElMessage.success(t('tenant.op_queued'))
    loadData()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  }
}

const shortPath = (path) => {
  const s = String(path || '')
  if (s.length <= 28) return s
  return `…${s.slice(-26)}`
}

const backupDirOf = (row) => {
  if (row?.backup_dir) return row.backup_dir
  if (row?.code) return `storage/backups/tenants/${row.code}`
  return ''
}

const formatSize = (n) => {
  const size = Number(n) || 0
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

const copyText = async (text) => {
  const value = String(text || '')
  if (!value) return
  try {
    await navigator.clipboard.writeText(value)
    ElMessage.success(t('tenant.backup_copied'))
  } catch {
    ElMessage.error(t('common.operation_failed'))
  }
}

const openDetail = (row) => {
  detailRow.value = row
  detailVisible.value = true
  domainHost.value = ''
  domainSslMode.value = 'edge'
  systemLogSummary.value = null
  void loadDetailDomains(row)
  void loadSystemLogSummary(row)
}

const supportVisible = ref(false)
const supportLoading = ref(false)
const supportRow = ref(null)
const supportAdmins = ref([])
const supportAudit = ref(null)

const openSupport = async (row) => {
  supportRow.value = row
  supportVisible.value = true
  supportLoading.value = true
  supportAdmins.value = []
  supportAudit.value = null
  try {
    const [adminsRes, auditRes] = await Promise.all([
      getPlatformTenantAdmins(row.id),
      getPlatformTenantAuditSummary(row.id)
    ])
    supportAdmins.value = adminsRes?.data?.list || []
    supportAudit.value = auditRes?.data?.audit || null
  } catch (e) {
    console.error(e)
    ElMessage.error(t('common.operation_failed'))
  } finally {
    supportLoading.value = false
  }
}

const onResetTenantAdminPwd = async (adminRow) => {
  try {
    const { value } = await ElMessageBox.prompt(t('platform.new_password'), t('tenant.reset_admin_password'), {
      inputType: 'password'
    })
    await resetPlatformTenantAdminPassword(supportRow.value.id, adminRow.id, {
      password: value,
      confirm_password: value
    })
    ElMessage.success(t('admin.reset_password_success'))
  } catch (e) {
    if (e === 'cancel') return
    ElMessage.error(e?.message || t('common.operation_failed'))
  }
}

const onUnlockTenantAdmin = async (adminRow) => {
  try {
    await unlockPlatformTenantAdmin(supportRow.value.id, { username: adminRow.username })
    ElMessage.success(t('tenant.unlock_success'))
  } catch (e) {
    ElMessage.error(e?.message || t('common.operation_failed'))
  }
}

const onResetTenantAdmin2FA = async (adminRow) => {
  try {
    await ElMessageBox.confirm(t('tenant.reset_admin_2fa'), t('common.warning'), { type: 'warning' })
    await resetPlatformTenantAdmin2FA(supportRow.value.id, adminRow.id)
    ElMessage.success(t('admin.unbind_success'))
    void openSupport(supportRow.value)
  } catch (e) {
    if (e === 'cancel') return
    ElMessage.error(e?.message || t('common.operation_failed'))
  }
}

const loadSystemLogSummary = async (row) => {
  if (!row?.id) return
  systemLogSummaryLoading.value = true
  try {
    const res = await getPlatformTenantSystemLogSummary(row.id)
    systemLogSummary.value = res?.data || null
  } catch (e) {
    console.error(e)
    systemLogSummary.value = null
  } finally {
    systemLogSummaryLoading.value = false
  }
}

const goTenantSystemLogs = (row) => {
  const code = row?.code
  detailVisible.value = false
  router.push(code ? { path: '/platform/system-logs', query: { code } } : '/platform/system-logs')
}

const openBackups = async (row) => {
  backupsRow.value = row
  backupsVisible.value = true
  backupsList.value = []
  backupsMeta.dir = row.backup_dir || backupDirOf(row)
  backupsMeta.last = row.last_backup_path || ''
  backupsLoading.value = true
  try {
    const res = await listPlatformTenantBackups(row.id)
    backupsList.value = res?.data?.list || []
    backupsMeta.dir = res?.data?.backup_dir || backupsMeta.dir
    backupsMeta.last = res?.data?.last_backup_path || backupsMeta.last
    backupsMeta.keep = res?.data?.backup_keep || settingsKeep.value || 0
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    backupsLoading.value = false
  }
}

const downloadBackup = async (file) => {
  if (!backupsRow.value || !file?.name) return
  downloadingName.value = file.name
  try {
    const blob = await downloadPlatformTenantBackup(backupsRow.value.id, file.name)
    const url = window.URL.createObjectURL(blob instanceof Blob ? blob : new Blob([blob]))
    const a = document.createElement('a')
    a.href = url
    a.download = file.name
    a.click()
    window.URL.revokeObjectURL(url)
  } catch (error) {
    if (!error?.__handled) {
      ElMessage.error(error?.translatedMessage || error?.message || t('common.operation_failed'))
    }
  } finally {
    downloadingName.value = ''
  }
}
</script>

<style scoped>
.ops-banner {
  margin-bottom: 12px;
}
.form-tip {
  margin-top: 4px;
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.4;
}
.migrate-tip {
  margin-bottom: 12px;
}
.op-meta {
  font-size: 12px;
  color: #64748b;
}
.seed-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 12px 0;
}
.backup-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.backup-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: #475569;
}
.backup-dir-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 13px;
  color: #64748b;
}
.drawer-actions {
  margin-top: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.system-log-summary {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid #e2e8f0;
}
.system-log-summary__title {
  font-weight: 600;
  margin-bottom: 8px;
}
.system-log-summary__meta {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
.system-log-summary__list {
  margin: 0 0 8px;
  padding-left: 18px;
  font-size: 13px;
  color: #334155;
}
.system-log-summary__time {
  color: #94a3b8;
  margin-right: 8px;
}
.system-log-summary__empty {
  font-size: 13px;
  color: #94a3b8;
  margin-bottom: 8px;
}
.dialog-scroll-body {
  max-height: 60vh;
  overflow-y: auto;
  padding-right: 4px;
}
.timeline-scroll-body {
  max-height: 60vh;
  overflow-y: auto;
  min-height: 120px;
  padding-right: 4px;
}
.domain-panel {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid #e2e8f0;
}
.domain-title {
  font-weight: 600;
  margin-bottom: 4px;
}
.domain-hint {
  margin: 0 0 10px;
  font-size: 12px;
  color: #64748b;
}
.domain-add {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}
.domain-item {
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 8px;
  margin-bottom: 8px;
}
.domain-dns {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.domain-dns-title {
  font-size: 12px;
  font-weight: 600;
  color: #334155;
}
.domain-dns-tip {
  margin: 0;
  padding: 8px 10px;
  border-radius: 6px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  color: #1e3a8a;
  font-size: 12px;
  line-height: 1.5;
}
.domain-dns-tip--muted {
  background: #f8fafc;
  border-color: #e2e8f0;
  color: #64748b;
}
.domain-dns-row {
  display: flex;
  gap: 10px;
  align-items: stretch;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
  overflow: hidden;
}
.domain-dns-row--txt {
  border-left: 3px solid #2563eb;
}
.domain-dns-row--cname {
  border-left: 3px solid #059669;
}
.domain-dns-type {
  flex: 0 0 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: #0f172a;
  background: #e2e8f0;
}
.domain-dns-row--txt .domain-dns-type {
  background: #dbeafe;
  color: #1d4ed8;
}
.domain-dns-row--cname .domain-dns-type {
  background: #d1fae5;
  color: #047857;
}
.domain-dns-body {
  flex: 1;
  min-width: 0;
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.domain-dns-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #475569;
}
.domain-dns-label {
  flex: 0 0 auto;
  min-width: 36px;
  color: #64748b;
}
.domain-dns-line code {
  flex: 1 1 auto;
  min-width: 0;
  padding: 2px 6px;
  border-radius: 4px;
  background: #fff;
  border: 1px solid #e2e8f0;
  color: #0f172a;
  word-break: break-all;
}
.domain-err {
  margin-top: 4px;
  font-size: 12px;
  color: #dc2626;
}
.domain-actions {
  margin-top: 6px;
}
.ml4 {
  margin-left: 4px;
}
.ops-summary-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 13px;
  color: #475569;
}
.list-mode-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 12px;
  margin-bottom: 12px;
}
.ops-inline {
  font-size: 13px;
  color: #475569;
}
.recycle-banner {
  margin-bottom: 12px;
}
.recycle-badge {
  margin-left: 6px;
  vertical-align: middle;
}
.queue-meta {
  font-size: 12px;
  color: #64748b;
}
html.dark .domain-dns-title {
  color: rgba(255, 255, 255, 0.85);
}
html.dark .domain-dns-tip {
  background: rgba(37, 99, 235, 0.18);
  border-color: rgba(96, 165, 250, 0.35);
  color: #bfdbfe;
}
html.dark .domain-dns-tip--muted {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.12);
  color: rgba(255, 255, 255, 0.55);
}
html.dark .domain-dns-row {
  border-color: rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.04);
}
html.dark .domain-dns-type {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.85);
}
html.dark .domain-dns-row--txt .domain-dns-type {
  background: rgba(37, 99, 235, 0.28);
  color: #93c5fd;
}
html.dark .domain-dns-row--cname .domain-dns-type {
  background: rgba(5, 150, 105, 0.28);
  color: #6ee7b7;
}
html.dark .domain-dns-line {
  color: rgba(255, 255, 255, 0.65);
}
html.dark .domain-dns-label {
  color: rgba(255, 255, 255, 0.45);
}
html.dark .domain-dns-line code {
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(255, 255, 255, 0.12);
  color: rgba(255, 255, 255, 0.85);
}
html.dark .ops-summary-row,
html.dark .ops-inline {
  color: rgba(255, 255, 255, 0.65);
}
html.dark .queue-meta {
  color: rgba(255, 255, 255, 0.45);
}
</style>
