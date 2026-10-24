<script setup lang="ts">
import { computed } from 'vue'

import { useDevice } from '@/composables/useDevice'
import { useUploadStore } from '@/stores/upload'
import UploadCard from './UploadCard.vue'

const store = useUploadStore()
const { isMobile } = useDevice()

const orderedTasks = computed(() => [...store.tasks].reverse())
const title = computed(() => (store.activeCount ? `上传队列（${store.activeCount}）` : '上传队列'))
</script>

<template>
  <el-drawer
    v-model="store.drawerOpen"
    :size="isMobile ? '100%' : 420"
    :title="title"
    class="upload-drawer"
  >
    <div v-if="!store.tasks.length" class="empty">暂无上传任务</div>

    <div v-else class="task-list">
      <UploadCard
        v-for="task in orderedTasks"
        :key="task.id"
        :task="task"
        @cancel="store.cancel"
        @retry="store.retry"
        @remove="store.removeTask"
      />
    </div>

    <template #footer>
      <div class="drawer-footer">
        <el-button text :disabled="!store.successCount && !store.failedCount" @click="store.clearFinished()">
          清除已完成
        </el-button>
        <el-button type="primary" @click="store.toggleDrawer(false)">关闭</el-button>
      </div>
    </template>
  </el-drawer>
</template>

<style scoped lang="scss">
.task-list {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-3);
}

.empty {
  padding: var(--ph-space-16) 0;
  text-align: center;
  color: var(--ph-text-secondary);
}

.drawer-footer {
  display: flex;
  justify-content: space-between;
}
</style>
