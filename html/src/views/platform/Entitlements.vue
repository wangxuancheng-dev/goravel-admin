<template>
  <div class="platform-entitlements">
    <el-card shadow="never" class="block">
      <template #header>
        <div class="card-head">
          <span>{{ $t('entitlement.catalog_title') }}</span>
          <div class="card-head__actions">
            <el-button :disabled="isViewer" @click="capabilityVisible = true">
              {{ $t('entitlement.register_capability') }}
            </el-button>
            <el-button type="primary" :disabled="isViewer" @click="moduleVisible = true">
              {{ $t('entitlement.register_module') }}
            </el-button>
          </div>
        </div>
      </template>
      <p class="hint">{{ $t('entitlement.catalog_hint') }}</p>
      <p class="hint">{{ $t('entitlement.handwritten_hint') }}</p>
      <p class="hint">{{ $t('entitlement.capability_hint') }}</p>
      <el-table v-loading="loading" :data="features" row-key="key" border size="small">
        <el-table-column prop="name" :label="$t('entitlement.feature')" min-width="160" />
        <el-table-column :label="'Key'" min-width="200">
          <template #default="{ row }"><code>{{ row.key }}</code></template>
        </el-table-column>
        <el-table-column :label="$t('entitlement.kind')" width="120">
          <template #default="{ row }">
            <el-tag v-if="featureKind(row) === 'capability'" type="primary" size="small">{{ $t('entitlement.kind_capability') }}</el-tag>
            <el-tag v-else-if="featureKind(row) === 'quota'" type="info" size="small">{{ $t('entitlement.kind_quota') }}</el-tag>
            <el-tag v-else-if="featureKind(row) === 'module'" type="warning" size="small">{{ $t('entitlement.kind_module') }}</el-tag>
            <span v-else>—</span>
          </template>
        </el-table-column>
        <el-table-column prop="type" :label="$t('entitlement.type')" width="100" />
        <el-table-column :label="$t('entitlement.always_on')" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.always_on" type="success" size="small">{{ $t('common.yes') }}</el-tag>
            <span v-else>—</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" class="block">
      <template #header>
        <div class="card-head">
          <span>{{ $t('entitlement.plans_title') }}</span>
          <el-button type="primary" :disabled="isViewer" @click="planVisible = true">
            {{ $t('entitlement.add_plan') }}
          </el-button>
        </div>
      </template>
      <el-table v-loading="loading" :data="plans" row-key="code" border size="small">
        <el-table-column prop="name" :label="$t('entitlement.plan')" min-width="140" />
        <el-table-column :label="'Code'" min-width="120">
          <template #default="{ row }"><code>{{ row.code }}</code></template>
        </el-table-column>
        <el-table-column :label="$t('entitlement.default')" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.is_default" type="success" size="small">{{ $t('common.yes') }}</el-tag>
            <span v-else>—</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('common.actions')" width="160">
          <template #default="{ row }">
            <el-button link type="primary" :disabled="isViewer" @click="openPlanEntitlements(row)">
              {{ $t('entitlement.edit_plan_entitlements') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="moduleVisible" :title="$t('entitlement.register_module')" width="520px" destroy-on-close @closed="resetModuleForm">
      <p class="hint">{{ $t('entitlement.handwritten_hint') }}</p>
      <el-form ref="moduleFormRef" :model="moduleForm" :rules="moduleRules" label-width="110px">
        <el-form-item :label="$t('entitlement.module_name')" prop="module_name">
          <el-input v-model="moduleForm.module_name" placeholder="guestbook" />
        </el-form-item>
        <el-form-item :label="$t('entitlement.feature')" prop="display_name">
          <el-input v-model="moduleForm.display_name" />
        </el-form-item>
        <el-form-item :label="$t('entitlement.menu_slugs')" prop="menu_slug">
          <el-input v-model="moduleForm.menu_slug" placeholder="guestbook" />
        </el-form-item>
        <el-form-item :label="$t('entitlement.always_on')" prop="always_on">
          <el-switch v-model="moduleForm.always_on" />
        </el-form-item>
        <el-form-item :label="$t('entitlement.row_quota')" prop="row_quota">
          <el-switch v-model="moduleForm.row_quota" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="moduleVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitModule">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="capabilityVisible" :title="$t('entitlement.register_capability')" width="520px" destroy-on-close @closed="resetCapabilityForm">
      <p class="hint">{{ $t('entitlement.capability_hint') }}</p>
      <el-form ref="capabilityFormRef" :model="capabilityForm" :rules="capabilityRules" label-width="110px">
        <el-form-item :label="$t('entitlement.module_name')" prop="module_name">
          <el-autocomplete
            v-model="capabilityForm.module_name"
            :fetch-suggestions="queryModuleOptions"
            placeholder="member"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item :label="$t('entitlement.capability_name')" prop="capability">
          <el-input v-model="capabilityForm.capability" placeholder="export" />
          <div class="form-extra">{{ $t('entitlement.capability_key_preview') }}</div>
        </el-form-item>
        <el-form-item :label="$t('entitlement.feature')" prop="display_name">
          <el-input v-model="capabilityForm.display_name" placeholder="Member export" />
        </el-form-item>
        <el-form-item :label="$t('entitlement.always_on')" prop="always_on">
          <el-switch v-model="capabilityForm.always_on" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="capabilityVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitCapability">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="planVisible" :title="$t('entitlement.add_plan')" width="520px" destroy-on-close @closed="resetPlanForm">
      <el-form ref="planFormRef" :model="planForm" :rules="planRules" label-width="110px">
        <el-form-item label="Code" prop="code">
          <el-input v-model="planForm.code" placeholder="pro" />
        </el-form-item>
        <el-form-item :label="$t('entitlement.plan')" prop="name">
          <el-input v-model="planForm.name" />
        </el-form-item>
        <el-form-item :label="$t('entitlement.description')" prop="description">
          <el-input v-model="planForm.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item :label="$t('entitlement.default')" prop="is_default">
          <el-switch v-model="planForm.is_default" />
        </el-form-item>
        <el-form-item :label="$t('entitlement.price_monthly')" prop="price_monthly">
          <el-input-number v-model="planForm.price_monthly" :min="0" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="planVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitPlan">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="planEntVisible" :title="$t('entitlement.edit_plan_entitlements')" width="640px" destroy-on-close>
      <p class="hint">{{ $t('entitlement.plan_entitlements_hint') }}</p>
      <el-input
        v-model="planEntText"
        type="textarea"
        :rows="12"
        :placeholder="'module.guestbook=true\nquota.module.guestbook.rows=1000'"
      />
      <template #footer>
        <el-button @click="planEntVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitPlanEntitlements">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import {
  getPlatformFeatures,
  getPlatformPlanEntitlements,
  getPlatformPlans,
  registerPlatformCapability,
  registerPlatformModuleFeature,
  updatePlatformPlan,
  upsertPlatformPlan
} from '@/api/platform'
import { getPlatformAdmin } from '@/utils/platformRequest'

const { t } = useI18n()
const isViewer = computed(() => getPlatformAdmin()?.role === 'viewer')

const features = ref([])
const plans = ref([])
const loading = ref(false)
const saving = ref(false)

const moduleVisible = ref(false)
const capabilityVisible = ref(false)
const planVisible = ref(false)
const planEntVisible = ref(false)
const editingPlan = ref(null)
const planEntText = ref('')

const moduleFormRef = ref()
const capabilityFormRef = ref()
const planFormRef = ref()

const moduleForm = reactive({
  module_name: '',
  display_name: '',
  menu_slug: '',
  always_on: false,
  row_quota: false
})
const capabilityForm = reactive({
  module_name: '',
  capability: '',
  display_name: '',
  always_on: false
})
const planForm = reactive({
  code: '',
  name: '',
  description: '',
  is_default: false,
  price_monthly: 0
})

const resetModuleForm = () => {
  Object.assign(moduleForm, { module_name: '', display_name: '', menu_slug: '', always_on: false, row_quota: false })
}
const resetCapabilityForm = () => {
  Object.assign(capabilityForm, { module_name: '', capability: '', display_name: '', always_on: false })
}
const resetPlanForm = () => {
  Object.assign(planForm, { code: '', name: '', description: '', is_default: false, price_monthly: 0 })
}

// Keep module.{a}.{b} within the backend feature key length limit (63).
const segmentPattern = /^[A-Za-z][A-Za-z0-9_-]{0,23}$/
const segmentRule = {
  validator: (_r, v, cb) => {
    if (!v || segmentPattern.test(String(v).trim())) cb()
    else cb(new Error(t('entitlement.capability_name_invalid')))
  },
  trigger: 'blur'
}
const requiredRule = { required: true, message: () => t('common.required'), trigger: 'blur' }

const moduleRules = {
  module_name: [requiredRule]
}
const capabilityRules = {
  module_name: [requiredRule, segmentRule],
  capability: [requiredRule, segmentRule]
}
const planRules = {
  code: [requiredRule],
  name: [requiredRule]
}

const featureKind = (row) => {
  const key = String(row?.key || '')
  if (key.startsWith('quota.')) return 'quota'
  if (key.startsWith('module.') && key.split('.').length >= 3) return 'capability'
  if (key.startsWith('module.')) return 'module'
  return 'other'
}

const moduleOptions = computed(() =>
  features.value
    .filter((f) => featureKind(f) === 'module')
    .map((f) => {
      const name = String(f.key || '').replace(/^module\./, '')
      return { value: name, label: f.name ? `${f.name} (${name})` : name }
    })
)

const queryModuleOptions = (query, cb) => {
  const q = String(query || '').toLowerCase()
  cb(moduleOptions.value.filter((o) => !q || o.value.toLowerCase().includes(q) || o.label.toLowerCase().includes(q)))
}

const pickList = (res) => res?.data?.list || res?.list || []

const reportError = (e) => {
  if (!e?.__handled) ElMessage.error(e?.message || t('common.operation_failed'))
}

const load = async () => {
  loading.value = true
  try {
    const [fRes, pRes] = await Promise.all([getPlatformFeatures(), getPlatformPlans()])
    features.value = pickList(fRes)
    plans.value = pickList(pRes)
  } catch (e) {
    reportError(e)
  } finally {
    loading.value = false
  }
}

const submitModule = async () => {
  try {
    await moduleFormRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await registerPlatformModuleFeature({
      module_name: moduleForm.module_name,
      display_name: moduleForm.display_name,
      menu_slug: moduleForm.menu_slug,
      always_on: !!moduleForm.always_on,
      row_quota: !!moduleForm.row_quota
    })
    ElMessage.success(t('entitlement.module_registered'))
    moduleVisible.value = false
    await load()
  } catch (e) {
    reportError(e)
  } finally {
    saving.value = false
  }
}

const submitCapability = async () => {
  try {
    await capabilityFormRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await registerPlatformCapability({
      module_name: capabilityForm.module_name,
      capability: capabilityForm.capability,
      display_name: capabilityForm.display_name,
      always_on: !!capabilityForm.always_on
    })
    ElMessage.success(t('entitlement.capability_registered'))
    capabilityVisible.value = false
    await load()
  } catch (e) {
    reportError(e)
  } finally {
    saving.value = false
  }
}

const submitPlan = async () => {
  try {
    await planFormRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await upsertPlatformPlan({
      ...planForm,
      is_public: true,
      status: 1
    })
    ElMessage.success(t('common.save_success'))
    planVisible.value = false
    await load()
  } catch (e) {
    reportError(e)
  } finally {
    saving.value = false
  }
}

const openPlanEntitlements = async (plan) => {
  editingPlan.value = plan
  planEntVisible.value = true
  try {
    const res = await getPlatformPlanEntitlements(plan.code)
    const list = pickList(res)
    planEntText.value =
      list.length > 0
        ? list
            .map((row) => {
              const key = String(row.feature_key || '').trim()
              return key ? `${key}=${String(row.value ?? '')}` : ''
            })
            .filter(Boolean)
            .join('\n')
        : features.value
            .map((f) => {
              if (f.type === 'boolean') return `${f.key}=false`
              if (f.type === 'limit') return `${f.key}=0`
              return ''
            })
            .filter(Boolean)
            .join('\n')
  } catch (e) {
    reportError(e)
    planEntText.value = ''
  }
}

const submitPlanEntitlements = async () => {
  if (!editingPlan.value) return
  const entitlements = {}
  String(planEntText.value || '')
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .forEach((line) => {
      const idx = line.indexOf('=')
      if (idx <= 0) return
      const key = line.slice(0, idx).trim()
      const value = line.slice(idx + 1).trim()
      if (key) entitlements[key] = value
    })
  saving.value = true
  try {
    await updatePlatformPlan(editingPlan.value.code, {
      code: editingPlan.value.code,
      name: editingPlan.value.name,
      description: editingPlan.value.description,
      is_default: editingPlan.value.is_default,
      status: editingPlan.value.status || 1,
      entitlements
    })
    ElMessage.success(t('entitlement.plan_entitlements_saved'))
    planEntVisible.value = false
    await load()
  } catch (e) {
    reportError(e)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.block {
  margin-bottom: 16px;
}
.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.card-head__actions {
  display: flex;
  gap: 8px;
}
.hint {
  margin: 0 0 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.form-extra {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
