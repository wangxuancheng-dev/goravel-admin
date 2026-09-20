<template>
  <el-dialog
    v-model="dialogVisible"
    :title="editId ? $t('demo_activity.edit') : $t('demo_activity.add')"
    width="720px"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="formData" :rules="rules" label-width="130px">
      <el-form-item :label="$t('demo_activity.title')" prop="title">
        <el-input v-model="formData.title" />
      </el-form-item>
      <el-form-item :label="$t('demo_activity.schedule_type')" prop="schedule_type">
        <el-radio-group v-model="formData.schedule_type">
          <el-radio-button value="once">{{ $t('demo_activity.type_once') }}</el-radio-button>
          <el-radio-button value="daily">{{ $t('demo_activity.type_daily') }}</el-radio-button>
          <el-radio-button value="weekly">{{ $t('demo_activity.type_weekly') }}</el-radio-button>
          <el-radio-button value="monthly">{{ $t('demo_activity.type_monthly') }}</el-radio-button>
          <el-radio-button value="yearly">{{ $t('demo_activity.type_yearly') }}</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item :label="$t('demo_activity.timezone')" prop="timezone">
        <el-select v-model="formData.timezone" style="width: 100%">
          <el-option label="Asia/Shanghai" value="Asia/Shanghai" />
          <el-option label="UTC" value="UTC" />
        </el-select>
      </el-form-item>

      <template v-if="formData.schedule_type === 'once'">
        <el-form-item :label="$t('demo_activity.start_at')" prop="start_at">
          <el-date-picker v-model="formData.start_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" style="width: 100%" />
        </el-form-item>
        <el-form-item :label="$t('demo_activity.end_at')" prop="end_at">
          <el-date-picker v-model="formData.end_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" style="width: 100%" />
        </el-form-item>
      </template>

      <template v-else-if="formData.schedule_type === 'daily'">
        <el-form-item :label="$t('demo_activity.daily_start')" prop="daily_start">
          <el-input v-model="formData.daily_start" placeholder="09:00" />
        </el-form-item>
        <el-form-item :label="$t('demo_activity.daily_end')" prop="daily_end">
          <el-input v-model="formData.daily_end" placeholder="18:00" />
        </el-form-item>
      </template>

      <template v-else-if="formData.schedule_type === 'weekly'">
        <el-form-item :label="$t('demo_activity.weekdays')" prop="weekdays">
          <el-checkbox-group v-model="formData.weekdays">
            <el-checkbox label="1">{{ $t('demo_activity.weekday_1') }}</el-checkbox>
            <el-checkbox label="2">{{ $t('demo_activity.weekday_2') }}</el-checkbox>
            <el-checkbox label="3">{{ $t('demo_activity.weekday_3') }}</el-checkbox>
            <el-checkbox label="4">{{ $t('demo_activity.weekday_4') }}</el-checkbox>
            <el-checkbox label="5">{{ $t('demo_activity.weekday_5') }}</el-checkbox>
            <el-checkbox label="6">{{ $t('demo_activity.weekday_6') }}</el-checkbox>
            <el-checkbox label="7">{{ $t('demo_activity.weekday_7') }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item :label="$t('demo_activity.daily_start')" prop="daily_start">
          <el-input v-model="formData.daily_start" placeholder="09:00" />
        </el-form-item>
        <el-form-item :label="$t('demo_activity.daily_end')" prop="daily_end">
          <el-input v-model="formData.daily_end" placeholder="18:00" />
        </el-form-item>
      </template>

      <template v-else-if="formData.schedule_type === 'monthly'">
        <el-form-item :label="$t('demo_activity.month_mode')">
          <el-radio-group v-model="formData.month_mode">
            <el-radio-button value="days">{{ $t('demo_activity.month_mode_days') }}</el-radio-button>
            <el-radio-button value="range">{{ $t('demo_activity.month_mode_range') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="formData.month_mode === 'days'" :label="$t('demo_activity.month_days')" prop="month_days">
          <el-input v-model="formData.month_days" placeholder="1,15,28" />
        </el-form-item>
        <template v-else>
          <el-form-item :label="$t('demo_activity.month_day_start')" prop="month_day_start">
            <el-input-number v-model="formData.month_day_start" :min="1" :max="31" />
          </el-form-item>
          <el-form-item :label="$t('demo_activity.month_day_end')" prop="month_day_end">
            <el-input-number v-model="formData.month_day_end" :min="1" :max="31" />
          </el-form-item>
        </template>
        <el-form-item :label="$t('demo_activity.daily_start')">
          <el-input v-model="formData.daily_start" placeholder="09:00" />
        </el-form-item>
        <el-form-item :label="$t('demo_activity.daily_end')">
          <el-input v-model="formData.daily_end" placeholder="18:00" />
        </el-form-item>
      </template>

      <template v-else-if="formData.schedule_type === 'yearly'">
        <el-form-item :label="$t('demo_activity.year_start')" prop="year_start">
          <el-input v-model="formData.year_start" placeholder="03-01" />
        </el-form-item>
        <el-form-item :label="$t('demo_activity.year_end')" prop="year_end">
          <el-input v-model="formData.year_end" placeholder="03-15" />
        </el-form-item>
        <el-form-item :label="$t('demo_activity.daily_start')">
          <el-input v-model="formData.daily_start" placeholder="09:00" />
        </el-form-item>
        <el-form-item :label="$t('demo_activity.daily_end')">
          <el-input v-model="formData.daily_end" placeholder="18:00" />
        </el-form-item>
      </template>

      <el-form-item :label="$t('common.enabled')" prop="enabled">
        <el-switch v-model="formData.enabled" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        {{ $t('common.confirm') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import {
  createDemoActivity,
  updateDemoActivity,
  getDemoActivityDetail
} from '@/api/demoActivity'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  editId: { type: [Number, String], default: null }
})
const emit = defineEmits(['update:modelValue', 'success'])
const { t } = useI18n()
const formRef = ref(null)
const submitting = ref(false)

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const formData = reactive({
  title: '',
  schedule_type: 'once',
  timezone: 'Asia/Shanghai',
  enabled: true,
  start_at: '',
  end_at: '',
  daily_start: '09:00',
  daily_end: '18:00',
  weekdays: ['1', '2', '3', '4', '5'],
  month_mode: 'days',
  month_days: '1,15',
  month_day_start: 1,
  month_day_end: 5,
  year_start: '03-01',
  year_end: '03-15'
})

const rules = computed(() => {
  const base = {
    title: [{ required: true, message: t('form.required'), trigger: 'blur' }],
    schedule_type: [{ required: true, message: t('form.required'), trigger: 'change' }]
  }
  const type = formData.schedule_type
  if (type === 'once') {
    base.start_at = [{ required: true, message: t('form.required'), trigger: 'change' }]
    base.end_at = [{ required: true, message: t('form.required'), trigger: 'change' }]
  } else if (type === 'daily' || type === 'weekly') {
    base.daily_start = [{ required: true, message: t('form.required'), trigger: 'blur' }]
    base.daily_end = [{ required: true, message: t('form.required'), trigger: 'blur' }]
    if (type === 'weekly') {
      base.weekdays = [{ required: true, message: t('form.required'), trigger: 'change' }]
    }
  } else if (type === 'monthly' && formData.month_mode === 'days') {
    base.month_days = [{ required: true, message: t('form.required'), trigger: 'blur' }]
  } else if (type === 'yearly') {
    base.year_start = [{ required: true, message: t('form.required'), trigger: 'blur' }]
    base.year_end = [{ required: true, message: t('form.required'), trigger: 'blur' }]
  }
  return base
})

const resetForm = () => {
  formData.title = ''
  formData.schedule_type = 'once'
  formData.timezone = 'Asia/Shanghai'
  formData.enabled = true
  formData.start_at = ''
  formData.end_at = ''
  formData.daily_start = '09:00'
  formData.daily_end = '18:00'
  formData.weekdays = ['1', '2', '3', '4', '5']
  formData.month_mode = 'days'
  formData.month_days = '1,15'
  formData.month_day_start = 1
  formData.month_day_end = 5
  formData.year_start = '03-01'
  formData.year_end = '03-15'
  formRef.value?.clearValidate?.()
}

const loadDetail = async () => {
  if (!props.editId) {
    resetForm()
    return
  }
  const res = await getDemoActivityDetail(props.editId)
  const row = res?.data?.demo_activity || res?.data?.activity || res?.data || {}
  formData.title = row.title || ''
  formData.schedule_type = row.schedule_type || 'once'
  formData.timezone = row.timezone || 'Asia/Shanghai'
  formData.enabled = row.enabled !== false
  formData.start_at = row.start_at || ''
  formData.end_at = row.end_at || ''
  formData.daily_start = row.daily_start || '09:00'
  formData.daily_end = row.daily_end || '18:00'
  formData.weekdays = String(row.weekdays || '')
    .split(',')
    .filter(Boolean)
  formData.month_mode = row.month_days ? 'days' : 'range'
  formData.month_days = row.month_days || '1,15'
  formData.month_day_start = row.month_day_start || 1
  formData.month_day_end = row.month_day_end || 5
  formData.year_start = row.year_start || '03-01'
  formData.year_end = row.year_end || '03-15'
}

watch(
  () => props.modelValue,
  (v) => {
    if (v) loadDetail()
  }
)

const handleClose = () => {
  resetForm()
}

const handleSubmit = async () => {
  await formRef.value?.validate?.()
  submitting.value = true
  try {
    const type = formData.schedule_type
    const payload = {
      title: formData.title,
      schedule_type: type,
      timezone: formData.timezone,
      enabled: formData.enabled
    }
    if (type === 'once') {
      payload.start_at = formData.start_at
      payload.end_at = formData.end_at
    } else if (type === 'daily') {
      payload.daily_start = formData.daily_start
      payload.daily_end = formData.daily_end
    } else if (type === 'weekly') {
      payload.weekdays = (formData.weekdays || []).join(',')
      payload.daily_start = formData.daily_start
      payload.daily_end = formData.daily_end
    } else if (type === 'monthly') {
      if (formData.month_mode === 'range') {
        payload.month_day_start = formData.month_day_start
        payload.month_day_end = formData.month_day_end
        payload.month_days = ''
      } else {
        payload.month_days = formData.month_days
        payload.month_day_start = 0
        payload.month_day_end = 0
      }
      payload.daily_start = formData.daily_start || ''
      payload.daily_end = formData.daily_end || ''
    } else if (type === 'yearly') {
      payload.year_start = formData.year_start
      payload.year_end = formData.year_end
      payload.daily_start = formData.daily_start || ''
      payload.daily_end = formData.daily_end || ''
    }
    if (props.editId) {
      await updateDemoActivity(props.editId, payload)
      ElMessage.success(t('common.update_success'))
    } else {
      await createDemoActivity(payload)
      ElMessage.success(t('common.create_success'))
    }
    dialogVisible.value = false
    emit('success')
  } finally {
    submitting.value = false
  }
}
</script>
