<script setup lang="ts">
import AppSidebar from '@/components/layout/AppSidebar.vue'
import AppTabBar from '@/components/layout/AppTabBar.vue'
import AppTopBar from '@/components/layout/AppTopBar.vue'
import UploadDrawer from '@/components/upload/UploadDrawer.vue'
import { useDevice } from '@/composables/useDevice'
import { useUploadStore } from '@/stores/upload'

const { isMobile } = useDevice()
const upload = useUploadStore()
</script>

<template>
  <div class="app-layout">
    <AppTopBar />

    <div class="app-body">
      <AppSidebar v-if="!isMobile" />
      <main class="app-main">
        <slot />
      </main>
    </div>

    <AppTabBar v-if="isMobile" />

    <button
      v-if="upload.tasks.length"
      type="button"
      class="upload-fab"
      :class="{ mobile: isMobile }"
      @click="upload.toggleDrawer(true)"
    >
      <el-progress
        type="circle"
        :percentage="upload.overallProgress"
        :width="34"
        :stroke-width="3"
        :show-text="false"
        :color="upload.activeCount ? '#409EFF' : '#00B42A'"
      />
      <span class="fab-text">
        {{ upload.activeCount ? `上传中 ${upload.activeCount}` : `已完成 ${upload.successCount}` }}
      </span>
    </button>

    <UploadDrawer />
  </div>
</template>

<style scoped lang="scss">
.app-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--ph-fill);
}

.app-body {
  display: flex;
  flex: 1;
  min-height: 0;
}

.app-main {
  flex: 1;
  min-width: 0;
  padding: var(--ph-space-6) var(--ph-space-6) var(--ph-space-10);
}

.upload-fab {
  position: fixed;
  right: var(--ph-space-6);
  bottom: var(--ph-space-6);
  z-index: 90;
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
  padding: 8px 14px 8px 8px;
  border: 1px solid var(--ph-border-light);
  border-radius: var(--ph-radius-full);
  background: var(--ph-fill-card);
  box-shadow: var(--ph-shadow-md);
  color: var(--ph-text-primary);
  cursor: pointer;
  transition: transform 150ms ease-out, box-shadow 150ms ease-out;

  &:hover {
    transform: translateY(-1px);
    box-shadow: var(--ph-shadow-lg);
  }

  &.mobile {
    right: var(--ph-space-4);
    bottom: calc(var(--ph-tabbar-height) + var(--ph-space-4));
  }
}

.fab-text {
  font-size: 13px;
}

@media (max-width: 767.98px) {
  .app-main {
    padding: var(--ph-space-4) var(--ph-space-4) calc(var(--ph-tabbar-height) + var(--ph-space-6));
  }
}
</style>
