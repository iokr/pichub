<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import { tabNav } from './menu'

const route = useRoute()

function isActive(path: string): boolean {
  if (path === '/') {
    return route.path === '/'
  }
  return route.path === path || route.path.startsWith(`${path}/`)
}

const activePath = computed(() => tabNav.find((item) => isActive(item.path))?.path ?? '')
</script>

<template>
  <nav class="app-tabbar">
    <router-link
      v-for="item in tabNav"
      :key="item.path"
      :to="item.path"
      class="tab-item"
      :class="{ active: activePath === item.path }"
    >
      <el-icon :size="20"><component :is="item.icon" /></el-icon>
      <span>{{ item.label }}</span>
    </router-link>
  </nav>
</template>

<style scoped lang="scss">
.app-tabbar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 100;
  display: flex;
  height: var(--ph-tabbar-height);
  padding-bottom: env(safe-area-inset-bottom);
  border-top: 1px solid var(--ph-border-light);
  background: var(--ph-fill-card);
}

.tab-item {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  color: var(--ph-text-secondary);
  font-size: 11px;
  transition: color 150ms ease-out;

  &.active {
    color: var(--ph-primary);
  }
}
</style>
