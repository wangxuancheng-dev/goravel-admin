<template>
  <ListPage
    page-class="platform-admin"
    :title="$t('menu.platform_admin')"
    :show-add-button="isOwner"
    :add-button-text="$t('common.add')"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="initialSearchForm"
    i18n-prefix="platform_admin"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :pagination="pagination"
    show-toolbar
    @add="openCreate"
    @search="handleSearch"
    @reset="handleReset"
    @refresh="loadData"
    @page-change="loadData"
    @sort-change="handleSortChange"
  >
    <template #role="{ row }">
      <el-tag :type="row.role === 'viewer' ? 'info' : 'success'" size="small">
        {{ row.role === 'viewer' ? $t('platform.role_viewer') : $t('platform.role_owner') }}
      </el-tag>
    </template>
    <template #status="{ row }">
      <el-tag :type="Number(row.status) === 1 ? 'success' : 'danger'" size="small">
        {{ Number(row.status) === 1 ? $t('common.enabled') : $t('common.disabled') }}
      </el-tag>
    </template>
    <template #operation="{ row }">
      <template v-if="isOwner">
        <el-button link type="primary" @click="openEdit(row)">{{ $t('common.edit') }}</el-button>
        <el-button link type="primary" @click="openReset(row)">{{ $t('platform_admin.reset_password') }}</el-button>
        <el-button
          link
          type="danger"
          :disabled="Number(row.id) === Number(currentId)"
          @click="handleDelete(row)"
        >
          {{ $t('common.delete') }}
        </el-button>
      </template>
      <span v-else class="muted">—</span>
    </template>
  </ListPage>

  <el-dialog
    v-model="formVisible"
    :title="editingId ? $t('platform_admin.edit') : $t('platform_admin.create')"
    width="480px"
    destroy-on-close
    @closed="clearForm"
  >
    <el-form ref="formRef" :model="form" :rules="formRules" label-width="100px">
      <el-form-item v-if="!editingId" :label="$t('platform_admin.username')" prop="username">
        <el-input v-model="form.username" autocomplete="off" />
      </el-form-item>
      <el-form-item v-if="!editingId" :label="$t('platform_admin.password')" prop="password">
        <el-input v-model="form.password" type="password" show-password autocomplete="new-password" />
      </el-form-item>
      <el-form-item :label="$t('platform_admin.name')" prop="name">
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item :label="$t('platform_admin.role')" prop="role">
        <el-select v-model="form.role" style="width: 100%" :disabled="isSelfEdit">
          <el-option :label="$t('platform.role_owner')" value="owner" />
          <el-option :label="$t('platform.role_viewer')" value="viewer" />
        </el-select>
      </el-form-item>
      <el-form-item v-if="editingId" :label="$t('table.status')" prop="status">
        <el-select v-model="form.status" style="width: 100%" :disabled="isSelfEdit">
          <el-option :label="$t('common.enabled')" :value="1" />
          <el-option :label="$t('common.disabled')" :value="0" />
        </el-select>
      </el-form-item>
      <el-alert v-if="isSelfEdit" type="info" :closable="false" show-icon :title="$t('platform_admin.self_edit_hint')" />
    </el-form>
    <template #footer>
      <el-button @click="formVisible = false">{{ $t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="saving" @click="submitForm">{{ $t('common.confirm') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="resetVisible" :title="$t('platform_admin.reset_password')" width="420px" destroy-on-close @closed="resetPwdForm">
    <el-form ref="resetFormRef" :model="resetForm" :rules="resetRules" label-width="110px">
      <el-form-item :label="$t('platform_admin.password')" prop="password">
        <el-input v-model="resetForm.password" type="password" show-password />
      </el-form-item>
      <el-form-item :label="$t('platform.confirm_password')" prop="confirm_password">
        <el-input v-model="resetForm.confirm_password" type="password" show-password />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="resetVisible = false">{{ $t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="resetSaving" @click="submitReset">{{ $t('common.confirm') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import {
  createPlatformAdmin,
  deletePlatformAdmin,
  getPlatformAdminList,
  resetPlatformAdminPassword,
  updatePlatformAdmin,
} from '@/api/platform'
import { getPlatformAdmin } from '@/utils/platformRequest'

const { t } = useI18n()
const platformAdmin = computed(() => getPlatformAdmin())
const isOwner = computed(() => platformAdmin.value?.role !== 'viewer')
const currentId = computed(() => platformAdmin.value?.id)

const initialSearchForm = { username: '', role: '', status: '' }

const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
} = useStandardListPage({
  fetchApi: getPlatformAdminList,
  initialSearchForm,
  defaultSort: 'id:desc',
})

const searchFields = computed(() => [
  { prop: 'username', label: t('platform_admin.username'), type: 'input', width: '140px' },
  {
    prop: 'role',
    label: t('platform_admin.role'),
    type: 'select',
    width: '120px',
    options: [
      { label: t('platform.role_owner'), value: 'owner' },
      { label: t('platform.role_viewer'), value: 'viewer' },
    ],
  },
  {
    prop: 'status',
    label: t('table.status'),
    type: 'select',
    width: '120px',
    options: [
      { label: t('common.enabled'), value: '1' },
      { label: t('common.disabled'), value: '0' },
    ],
  },
])

const tableColumns = computed(() => [
  { field: 'id', title: t('table.id'), width: 70, sortable: true, key: 'id' },
  { field: 'username', title: t('platform_admin.username'), width: 140, key: 'username' },
  { field: 'name', title: t('platform_admin.name'), width: 140, key: 'name' },
  { field: 'role', title: t('platform_admin.role'), width: 110, slot: 'role', key: 'role' },
  { field: 'status', title: t('table.status'), width: 100, slot: 'status', key: 'status' },
  { field: 'created_at', title: t('common.created_at'), width: 170, key: 'created_at' },
  { field: 'operation', title: t('table.operation'), width: 220, slot: 'operation', key: 'operation', fixed: 'right' },
])

const formVisible = ref(false)
const saving = ref(false)
const editingId = ref(null)
const formRef = ref()
const form = reactive({
  username: '',
  password: '',
  name: '',
  role: 'viewer',
  status: 1,
})
const isSelfEdit = computed(() => editingId.value && Number(editingId.value) === Number(currentId.value))

const formRules = computed(() => ({
  username: editingId.value
    ? []
    : [{ required: true, message: () => t('platform_admin.username'), trigger: 'blur' }],
  password: editingId.value
    ? []
    : [
        { required: true, message: () => t('platform_admin.password'), trigger: 'blur' },
        { min: 6, message: () => t('platform_admin.password_min'), trigger: 'blur' },
      ],
  role: [{ required: true, message: () => t('platform_admin.role'), trigger: 'change' }],
}))

const clearForm = () => {
  editingId.value = null
  form.username = ''
  form.password = ''
  form.name = ''
  form.role = 'viewer'
  form.status = 1
}

const openCreate = () => {
  clearForm()
  formVisible.value = true
}

const openEdit = (row) => {
  editingId.value = row.id
  form.username = row.username || ''
  form.password = ''
  form.name = row.name || ''
  form.role = row.role === 'viewer' ? 'viewer' : 'owner'
  form.status = Number(row.status) === 1 ? 1 : 0
  formVisible.value = true
}

const submitForm = async () => {
  await formRef.value?.validate()
  saving.value = true
  try {
    if (editingId.value) {
      const payload = { name: form.name }
      if (!isSelfEdit.value) {
        payload.role = form.role
        payload.status = form.status
      }
      await updatePlatformAdmin(editingId.value, payload)
    } else {
      await createPlatformAdmin({
        username: form.username,
        password: form.password,
        name: form.name,
        role: form.role,
      })
    }
    ElMessage.success(t('common.success'))
    formVisible.value = false
    loadData()
  } catch (e) {
    if (!e?.__handled) ElMessage.error(e?.message || t('common.operation_failed'))
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(t('platform_admin.delete_confirm'), t('common.warning'), { type: 'warning' })
    await deletePlatformAdmin(row.id)
    ElMessage.success(t('common.success'))
    loadData()
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    if (!e?.__handled) ElMessage.error(e?.message || t('common.operation_failed'))
  }
}

const resetVisible = ref(false)
const resetSaving = ref(false)
const resetTargetId = ref(null)
const resetFormRef = ref()
const resetForm = reactive({
  password: '',
  confirm_password: '',
})
const resetRules = {
  password: [
    { required: true, message: () => t('platform_admin.password'), trigger: 'blur' },
    { min: 6, message: () => t('platform_admin.password_min'), trigger: 'blur' },
  ],
  confirm_password: [
    { required: true, message: () => t('platform.confirm_password'), trigger: 'blur' },
    {
      validator: (_r, v, cb) => {
        if (v !== resetForm.password) cb(new Error(t('platform.confirm_password')))
        else cb()
      },
      trigger: 'blur',
    },
  ],
}

const resetPwdForm = () => {
  resetTargetId.value = null
  resetForm.password = ''
  resetForm.confirm_password = ''
}

const openReset = (row) => {
  resetPwdForm()
  resetTargetId.value = row.id
  resetVisible.value = true
}

const submitReset = async () => {
  await resetFormRef.value?.validate()
  resetSaving.value = true
  try {
    await resetPlatformAdminPassword(resetTargetId.value, {
      password: resetForm.password,
      confirm_password: resetForm.confirm_password,
    })
    ElMessage.success(t('common.success'))
    resetVisible.value = false
  } catch (e) {
    if (!e?.__handled) ElMessage.error(e?.message || t('common.operation_failed'))
  } finally {
    resetSaving.value = false
  }
}
</script>

<style scoped>
.muted {
  color: #94a3b8;
}
</style>
