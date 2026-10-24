<script setup lang="ts">
import { computed } from 'vue'

import type { Album, StorageStrategy, Tag, UploadOptions } from '@/types/api'

const props = defineProps<{
  modelValue: UploadOptions
  albums: Album[]
  tagOptions?: Tag[]
  storages?: StorageStrategy[]
  showStorage?: boolean
}>()

const emit = defineEmits<{ (event: 'update:modelValue', value: UploadOptions): void }>()

const tagList = computed(() =>
  (props.modelValue.tags || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean),
)

function patch(key: keyof UploadOptions, value: unknown): void {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function updateTags(values: string[]): void {
  patch('tags', values.join(','))
}

function updateVisibility(value: string | number | boolean | undefined): void {
  patch('is_public', value === 'public')
}
</script>

<template>
  <div class="upload-params">
    <div class="field">
      <span class="label">相册</span>
      <el-select
        :model-value="modelValue.album_id"
        placeholder="不加入相册"
        clearable
        @update:model-value="patch('album_id', $event)"
      >
        <el-option v-for="album in albums" :key="album.id" :label="album.name" :value="album.id" />
      </el-select>
    </div>

    <div class="field">
      <span class="label">标签</span>
      <el-select
        :model-value="tagList"
        multiple
        filterable
        allow-create
        default-first-option
        placeholder="选择或输入标签"
        @update:model-value="updateTags"
      >
        <el-option v-for="tag in tagOptions || []" :key="tag.id" :label="tag.name" :value="tag.name" />
      </el-select>
    </div>

    <div class="field">
      <span class="label">可见性</span>
      <el-radio-group :model-value="modelValue.is_public === false ? 'private' : 'public'" @update:model-value="updateVisibility">
        <el-radio-button value="public">公开</el-radio-button>
        <el-radio-button value="private">私有</el-radio-button>
      </el-radio-group>
    </div>

    <div v-if="showStorage" class="field">
      <span class="label">存储策略</span>
      <el-select
        :model-value="modelValue.storage_id"
        placeholder="使用默认存储"
        clearable
        @update:model-value="patch('storage_id', $event)"
      >
        <el-option v-for="item in storages || []" :key="item.id" :label="item.name" :value="item.id" />
      </el-select>
    </div>
  </div>
</template>

<style scoped lang="scss">
.upload-params {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-3);
}

.field {
  display: flex;
  align-items: center;
  gap: var(--ph-space-3);

  .label {
    flex: 0 0 64px;
    font-size: 13px;
    color: var(--ph-text-secondary);
  }

  :deep(.el-select) {
    flex: 1;
  }
}
</style>
