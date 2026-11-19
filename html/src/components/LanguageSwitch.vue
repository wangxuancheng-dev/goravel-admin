<template>
  <el-dropdown @command="handleCommand" trigger="click">
    <el-button type="text" class="language-switch topbar-icon-btn">
      <el-icon class="language-icon topbar-icon topbar-icon--no-rotate">
        <TextIcon />
      </el-icon>
    </el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          command="zh-CN"
          :class="{ 'is-active': currentLanguage === 'zh-CN' }"
        >
          <span class="language-option">
            <span class="language-flag">🇨🇳</span>
            <span>{{ $t('common.language_zh') }}</span>
          </span>
        </el-dropdown-item>
        <el-dropdown-item
          command="en-US"
          :class="{ 'is-active': currentLanguage === 'en-US' }"
        >
          <span class="language-option">
            <span class="language-flag">🇺🇸</span>
            <span>{{ $t('common.language_en') }}</span>
          </span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup>
import { computed, defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import Storage from '../utils/storage'

const { t } = useI18n()

// 自定义文字图标组件
const TextIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      width: '1.2em',
      height: '1.2em',
      style: { verticalAlign: 'middle' }
    }, [
      h('path', {
        fill: 'currentColor',
        d: 'm18.5 10l4.4 11h-2.155l-1.201-3h-4.09l-1.199 3h-2.154L16.5 10h2zM10 2v2h6v2h-1.968a18.222 18.222 0 0 1-3.62 6.301a14.864 14.864 0 0 0 2.336 1.707l-.751 1.878A17.015 17.015 0 0 1 9 13.725a16.676 16.676 0 0 1-6.201 3.548l-.536-1.929a14.7 14.7 0 0 0 5.327-3.042A18.078 18.078 0 0 1 4.767 8h2.24A16.032 16.032 0 0 0 9 10.877a16.165 16.165 0 0 0 2.91-4.876L2 6V4h6V2h2zm7.5 10.885L16.253 16h2.492L17.5 12.885z'
      })
    ])
  }
})

const { locale } = useI18n()

const currentLanguage = computed(() => locale.value)

const currentLanguageText = computed(() => {
  return currentLanguage.value === 'zh-CN' ? t('common.language_zh') : t('common.language_en')
})

const handleCommand = (command) => {
  locale.value = command
  Storage.setItem('language', command)
  // 语言切换会自动通过 ElConfigProvider 更新，无需刷新页面
}
</script>

<style scoped>
.language-switch {
  color: var(--text-color-regular);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  margin: 0;
}

.language-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

.language-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.language-flag {
  font-size: 16px;
}

.is-active {
  color: var(--el-color-primary);
  font-weight: bold;
}
</style>

