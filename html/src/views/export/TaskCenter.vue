<template>
  <div class="list-page task-center">
    <div class="task-center-title">{{ $t('menu.export') }}</div>
    <el-tabs v-model="activeTab">
      <el-tab-pane :label="$t('export.tab_exports')" name="exports">
        <ExportList embedded />
      </el-tab-pane>
      <el-tab-pane :label="$t('export.tab_imports')" name="imports">
        <ImportList />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import ExportList from './ExportList.vue'
import ImportList from './ImportList.vue'

const activeTab = ref('exports')
</script>

<style scoped>
.task-center-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 12px;
}
</style>
