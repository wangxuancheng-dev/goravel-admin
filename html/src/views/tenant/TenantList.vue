<template>
  <div class="legacy-tenant-redirect">
    <el-result
      icon="info"
      :title="$t('platform.moved_title')"
      :sub-title="$t('platform.moved_hint')"
    >
      <template #extra>
        <el-button type="primary" @click="$router.push('/platform/tenants')">
          {{ $t('platform.go_platform') }}
        </el-button>
      </template>
    </el-result>
  </div>
</template>

<script setup>
// 租户管理已迁至平台控制台；保留此组件避免旧菜单 component 路径 404
</script>
