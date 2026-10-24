<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import { useUserStore } from '@/stores/user'
import { adminNav, mainNav, matchActivePath } from './menu'

const route = useRoute()
const user = useUserStore()

const activeIndex = computed(() => matchActivePath(route.path))
</script>

<template>
  <aside class="app-sidebar">
    <el-menu :default-active="activeIndex" router class="sidebar-menu">
      <el-menu-item v-for="item in mainNav" :key="item.path" :index="item.path">
        <el-icon><component :is="item.icon" /></el-icon>
        <span>{{ item.label }}</span>
      </el-menu-item>
    </el-menu>

    <template v-if="user.isAdmin">
      <div class="sidebar-section">管理后台</div>
      <el-menu :default-active="activeIndex" router class="sidebar-menu">
        <el-menu-item v-for="item in adminNav" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </el-menu-item>
      </el-menu>
    </template>
  </aside>
</template>

<style scoped lang="scss">
.app-sidebar {
  width: var(--ph-sidebar-width);
  flex: 0 0 var(--ph-sidebar-width);
  padding: var(--ph-space-4) var(--ph-space-3);
  border-right: 1px solid var(--ph-border-light);
  background: var(--ph-fill-card);
  overflow-y: auto;
  transition: width 200ms cubic-bezier(0.4, 0, 0.2, 1);
}

.sidebar-menu {
  background: transparent;

  :deep(.el-menu-item) {
    height: 42px;
    margin-bottom: 4px;
    border-radius: var(--ph-radius-md);
    color: var(--ph-text-regular);

    &:hover {
      background: var(--ph-fill-hover);
    }

    &.is-active {
      background: var(--ph-primary-light-9);
      color: var(--ph-primary);
      font-weight: 500;
    }
  }
}

.sidebar-section {
  margin: var(--ph-space-4) var(--ph-space-3) var(--ph-space-2);
  font-size: 12px;
  letter-spacing: 0.08em;
  color: var(--ph-text-secondary);
}

@media (max-width: 1199px) {
  .app-sidebar {
    width: 64px;
    flex-basis: 64px;
    padding: var(--ph-space-4) var(--ph-space-2);

    :deep(.el-menu-item span) {
      display: none;
    }

    :deep(.el-menu-item) {
      justify-content: center;
      padding: 0 !important;
    }
  }

  .sidebar-section {
    display: none;
  }
}
</style>
