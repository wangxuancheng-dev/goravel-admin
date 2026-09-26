<template>
  <div>
    <div style="margin-bottom: 12px; display: flex; gap: 8px;">
      <el-button @click="loadSessions" :loading="loading">{{ $t('common.refresh') }}</el-button>
      <el-button type="danger" plain @click="revokeOthers">{{ $t('profile.session_revoke_others') }}</el-button>
    </div>
    <p style="color: var(--el-text-color-secondary); font-size: 13px; margin-bottom: 12px;">
      {{ $t('profile.session_hint') }}
    </p>
    <el-table :data="sessions" v-loading="loading" stripe>
      <el-table-column :label="$t('profile.session_device')" min-width="200">
        <template #default="{ row }">
          <span>{{ [row.browser, row.os].filter(Boolean).join(' / ') || row.name || '-' }}</span>
          <el-tag v-if="row.is_current" size="small" type="primary" style="margin-left: 8px">
            {{ $t('profile.session_current') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column prop="last_used_at" :label="$t('profile.session_last_used')" width="180" />
      <el-table-column prop="created_at" :label="$t('table.created_at')" width="180" />
      <el-table-column :label="$t('table.operation')" width="120" fixed="right">
        <template #default="{ row }">
          <el-button v-if="!row.is_current" type="danger" link @click="revokeOne(row)">
            {{ $t('profile.session_revoke') }}
          </el-button>
          <span v-else>-</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAuthTokens, revokeAuthToken, revokeOtherAuthTokens } from '../../api/auth'

const { t } = useI18n()
const loading = ref(false)
const sessions = ref([])

const loadSessions = async () => {
  loading.value = true
  try {
    const res = await getAuthTokens()
    sessions.value = res.data?.tokens || []
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

const revokeOne = async (row) => {
  try {
    await ElMessageBox.confirm(t('profile.session_revoke_confirm'), { type: 'warning' })
    await revokeAuthToken(row.id)
    ElMessage.success(t('common.operation_success'))
    await loadSessions()
  } catch (e) {
    if (e !== 'cancel') console.error(e)
  }
}

const revokeOthers = async () => {
  try {
    await ElMessageBox.confirm(t('profile.session_revoke_others_confirm'), { type: 'warning' })
    await revokeOtherAuthTokens()
    ElMessage.success(t('common.operation_success'))
    await loadSessions()
  } catch (e) {
    if (e !== 'cancel') console.error(e)
  }
}

onMounted(() => { loadSessions() })
</script>