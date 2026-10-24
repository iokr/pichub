<script setup lang="ts">
import { computed } from 'vue'
import { Close, Delete, RefreshRight } from '@element-plus/icons-vue'

import { useCopy } from '@/composables/useCopy'
import type { UploadTask } from '@/stores/upload'
import { formatBytes } from '@/utils/format'

const props = defineProps<{ task: UploadTask }>()

const emit = defineEmits<{
  (event: 'cancel', id: string): void
  (event: 'retry', id: string): void
  (event: 'remove', id: string): void
}>()

const { copy } = useCopy()

const active = computed(() => ['waiting', 'hashing', 'uploading'].includes(props.task.status))
const done = computed(() => ['success', 'error', 'canceled'].includes(props.task.status))

const statusText = computed(() => {
  switch (props.task.status) {
    case 'waiting':
      return '等待上传'
    case 'hashing':
      return '校验文件'
    case 'uploading':
      return `上传中 ${props.task.progress}%`
    case 'success':
      return props.task.dedup ? '秒传完成' : '上传完成'
    case 'error':
      return '上传失败'
    case 'canceled':
      return '已取消'
    default:
      return ''
  }
})
</script>

<template>
  <div class="upload-card" :class="task.status">
    <div class="thumb-wrap">
      <img :src="task.preview" alt="" class="thumb" />
      <span v-if="task.status === 'success'" class="thumb-badge success">✓</span>
      <span v-else-if="task.status === 'error'" class="thumb-badge error">!</span>
    </div>

    <div class="info">
      <div class="name text-ellipsis" :title="task.name">{{ task.name }}</div>
      <div class="meta">
        <span>{{ formatBytes(task.size) }}</span>
        <span v-if="task.result">{{ task.result.width }}×{{ task.result.height }}</span>
        <el-tag v-if="task.dedup" size="small" type="success" effect="plain">秒传</el-tag>
        <span class="status">{{ statusText }}</span>
      </div>

      <el-progress
        v-if="active"
        :percentage="task.progress"
        :stroke-width="4"
        :show-text="false"
        class="progress"
      />

      <div v-if="task.status === 'error'" class="error text-ellipsis" :title="task.error">{{ task.error }}</div>

      <div v-if="task.status === 'success' && task.result" class="links">
        <el-button size="small" text type="primary" @click="copy(task.result.url, '图片链接已复制')">URL</el-button>
        <el-button size="small" text type="primary" @click="copy(task.result.markdown, 'Markdown 已复制')">
          Markdown
        </el-button>
        <el-button size="small" text type="primary" @click="copy(task.result.html, 'HTML 已复制')">HTML</el-button>
        <el-button size="small" text type="primary" @click="copy(task.result.bbcode, 'BBCode 已复制')">
          BBCode
        </el-button>
      </div>
    </div>

    <div class="actions">
      <el-button v-if="active" text circle aria-label="取消上传" @click="emit('cancel', task.id)">
        <el-icon><Close /></el-icon>
      </el-button>
      <el-button v-if="task.status === 'error' || task.status === 'canceled'" text circle aria-label="重试" @click="emit('retry', task.id)">
        <el-icon><RefreshRight /></el-icon>
      </el-button>
      <el-button v-if="done" text circle aria-label="移除任务" @click="emit('remove', task.id)">
        <el-icon><Delete /></el-icon>
      </el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.upload-card {
  display: flex;
  gap: var(--ph-space-3);
  padding: var(--ph-space-3);
  border: 1px solid var(--ph-border-light);
  border-radius: var(--ph-radius-lg);
  background: var(--ph-fill-card);
  transition: border-color 150ms ease-out;

  &.success {
    border-color: var(--ph-primary-light-7);
  }

  &.error {
    border-color: #f7c6c6;
  }
}

.thumb-wrap {
  position: relative;
  flex: 0 0 64px;
  width: 64px;
  height: 64px;
}

.thumb {
  width: 100%;
  height: 100%;
  border-radius: var(--ph-radius-md);
  object-fit: cover;
  background: var(--ph-fill);
}

.thumb-badge {
  position: absolute;
  right: -4px;
  bottom: -4px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  color: #fff;
  font-size: 11px;
  font-weight: 700;

  &.success {
    background: var(--ph-success);
  }

  &.error {
    background: var(--ph-danger);
  }
}

.info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.name {
  font-size: 14px;
  color: var(--ph-text-primary);
}

.meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--ph-space-2);
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.status {
  color: var(--ph-text-secondary);
}

.progress {
  margin-top: 2px;
}

.error {
  font-size: 12px;
  color: var(--ph-danger);
}

.links {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
  margin-top: 2px;

  :deep(.el-button) {
    padding: 2px 4px;
    font-size: 12px;
  }
}

.actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;

  :deep(.el-button) {
    margin-left: 0;
    color: var(--ph-text-secondary);
  }
}
</style>
