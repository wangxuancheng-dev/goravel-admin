<template>
  <el-select
    ref="timezoneSelectRef"
    class="timezone-switch"
    v-model="selectedTimezone"
    size="small"
    filterable
    placement="bottom-end"
    :offset="8"
    autocomplete="new-password"
    name="timezone-input"
    :aria-label="$t('header.timezone')"
    popper-class="timezone-select-popper"
    :teleported="true"
    :placeholder="$t('header.timezone')"
    @visible-change="handleVisibleChange"
  >
    <el-option
      v-for="option in timezoneOptions"
      :key="option.value"
      :label="option.label"
      :value="option.value"
    />
  </el-select>
</template>

<script setup>
import { computed, ref, onMounted, nextTick } from 'vue'
import { useAppStore } from '../store/app'
import { timezoneSelectOptions } from '@/utils/timezoneOptions'

const appStore = useAppStore()
const timezoneSelectRef = ref(null)

const timezoneOptions = computed(() => timezoneSelectOptions(appStore.timezone))

const selectedTimezone = computed({
  get: () => appStore.timezone,
  set: (val) => appStore.setTimezone(val),
})

const applyInputAntiAutofill = () => {
  const inputEl = timezoneSelectRef.value?.$el?.querySelector?.('input.el-select__input')
  if (!inputEl) return

  inputEl.setAttribute('autocomplete', 'off')
  inputEl.setAttribute('autocorrect', 'off')
  inputEl.setAttribute('autocapitalize', 'off')
  inputEl.setAttribute('spellcheck', 'false')
  inputEl.setAttribute('data-form-type', 'other')
  inputEl.setAttribute('data-lpignore', 'true')
  inputEl.setAttribute('data-1p-ignore', 'true')
  inputEl.setAttribute('name', 'timezone-filter-input')

  inputEl.setAttribute('readonly', 'readonly')
  const unlockReadonly = () => {
    inputEl.removeAttribute('readonly')
    inputEl.removeEventListener('focus', unlockReadonly)
    inputEl.removeEventListener('pointerdown', unlockReadonly)
  }
  inputEl.addEventListener('focus', unlockReadonly, { once: true })
  inputEl.addEventListener('pointerdown', unlockReadonly, { once: true })
}

const handleVisibleChange = async () => {
  await nextTick()
  applyInputAntiAutofill()
}

onMounted(() => {
  applyInputAntiAutofill()
})
</script>

<style scoped>
.timezone-switch {
  width: min(240px, 100%);
  min-width: 0;
}

.timezone-switch :deep(.el-select__selection) {
  overflow: hidden;
}

.timezone-switch :deep(.el-select__selected-item) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.timezone-switch :deep(.el-input__wrapper) {
  border-radius: 10px;
  transition: box-shadow 0.2s ease, border-color 0.2s ease;
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--border-color-light) 75%, transparent) inset;
}

.timezone-switch :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 35%, transparent) inset;
}

.timezone-switch :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--el-color-primary) inset, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 18%, transparent);
}
</style>

<style>
.el-popper.timezone-select-popper {
  border-radius: 12px !important;
  border: 1px solid color-mix(in srgb, var(--border-color-light) 70%, transparent) !important;
  background: color-mix(in srgb, var(--card-bg, #fff) 98%, transparent) !important;
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.14), 0 2px 8px rgba(15, 23, 42, 0.08) !important;
  padding: 6px 0 !important;
  overflow: hidden;
  backdrop-filter: blur(8px);
  min-width: 280px !important;
  max-width: min(420px, 92vw);
}

.el-popper.timezone-select-popper .el-select-dropdown__wrap {
  max-height: min(380px, 52vh);
}

.el-popper.timezone-select-popper .el-select-dropdown__list {
  padding: 4px 6px;
}

.el-popper.timezone-select-popper .el-select-dropdown__item {
  border-radius: 8px;
  margin: 2px 0;
  padding: 0 12px;
  min-height: 36px;
  line-height: 36px;
  font-size: 13px;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.el-popper.timezone-select-popper .el-select-dropdown__item.is-hovering,
.el-popper.timezone-select-popper .el-select-dropdown__item:hover {
  background-color: color-mix(in srgb, var(--el-color-primary) 10%, transparent) !important;
}

.el-popper.timezone-select-popper .el-select-dropdown__item.is-selected {
  font-weight: 600;
  color: var(--el-color-primary) !important;
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent) !important;
}

.el-popper.timezone-select-popper .el-select-dropdown__empty {
  padding: 12px 16px;
  font-size: 13px;
}

.el-popper.timezone-select-popper .el-scrollbar__bar {
  z-index: 2;
}

html.dark .el-popper.timezone-select-popper {
  border-color: rgba(255, 255, 255, 0.12) !important;
  background: color-mix(in srgb, var(--card-bg, #1d1e1f) 96%, transparent) !important;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.5), 0 2px 8px rgba(0, 0, 0, 0.35) !important;
}

html.dark .el-popper.timezone-select-popper .el-select-dropdown__item.is-hovering,
html.dark .el-popper.timezone-select-popper .el-select-dropdown__item:hover {
  background: rgba(255, 255, 255, 0.08) !important;
}

html.dark .el-popper.timezone-select-popper .el-select-dropdown__item.is-selected {
  background: color-mix(in srgb, var(--el-color-primary) 22%, rgba(255, 255, 255, 0.06)) !important;
}
</style>
