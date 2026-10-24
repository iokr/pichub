<script setup lang="ts">
import { useTheme } from '@/composables/useTheme'

const { isDark, toggle } = useTheme()
</script>

<template>
  <el-tooltip :content="isDark ? '切换到亮色模式' : '切换到暗色模式'" placement="bottom">
    <el-button text circle class="theme-toggle" aria-label="切换主题" @click="toggle">
      <el-icon :size="18">
        <Sunny v-if="isDark" />
        <Moon v-else />
      </el-icon>
    </el-button>
  </el-tooltip>
</template>

<style scoped>
.theme-toggle {
  color: var(--ph-text-regular);
}
</style>
