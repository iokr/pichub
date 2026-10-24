<script setup lang="ts">
withDefaults(
  defineProps<{
    description?: string
    size?: 'small' | 'default' | 'large'
  }>(),
  {
    description: '暂无数据',
    size: 'default',
  },
)
</script>

<template>
  <div class="empty-state">
    <el-empty :description="description" :image-size="size === 'small' ? 64 : 96">
      <slot name="actions" />
    </el-empty>
  </div>
</template>

<style scoped>
.empty-state {
  padding: var(--ph-space-8) 0;
}
</style>
