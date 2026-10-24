<script setup lang="ts">
import { computed } from 'vue'
import { Check } from '@element-plus/icons-vue'

import type { Image } from '@/types/api'
import { formatBytes } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    image: Image
    selectable?: boolean
    selected?: boolean
  }>(),
  {
    selectable: false,
    selected: false,
  },
)

const emit = defineEmits<{
  (event: 'open', image: Image): void
  (event: 'select', id: number): void
}>()

const displayName = computed(() => props.image.name || props.image.origin_name || props.image.hash)
</script>

<template>
  <div class="image-card" :class="{ selected }" @click="emit('open', image)">
    <div class="thumb">
      <img :src="image.thumb_url || image.url" :alt="displayName" loading="lazy" />

      <span v-if="selectable" class="checkbox" @click.stop="emit('select', image.id)">
        <el-icon v-if="selected" :size="14"><Check /></el-icon>
      </span>

      <span v-if="!image.is_public" class="privacy">私有</span>

      <div class="overlay">
        <span class="name text-ellipsis">{{ displayName }}</span>
        <span class="meta">
          {{ image.width }}×{{ image.height }} · {{ formatBytes(image.size) }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.image-card {
  border-radius: var(--ph-radius-lg);
  overflow: hidden;
  background: var(--ph-fill-card);
  border: 1px solid var(--ph-border-light);
  cursor: pointer;
  transition: transform 200ms cubic-bezier(0.4, 0, 0.2, 1), box-shadow 200ms cubic-bezier(0.4, 0, 0.2, 1),
    border-color 150ms ease-out;

  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--ph-shadow-md);

    .overlay {
      opacity: 1;
    }
  }

  &.selected {
    border-color: var(--ph-primary);
    box-shadow: 0 0 0 2px var(--ph-primary-light-7);
  }
}

.thumb {
  position: relative;
  aspect-ratio: 1 / 1;
  background: var(--ph-fill);
  overflow: hidden;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
}

.checkbox {
  position: absolute;
  top: 8px;
  left: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: 2px solid #fff;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.25);
  color: #fff;
  transition: background-color 150ms ease-out;
}

.selected .checkbox {
  background: var(--ph-primary);
  border-color: var(--ph-primary);
}

.privacy {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 0 6px;
  border-radius: var(--ph-radius-sm);
  background: rgba(0, 0, 0, 0.45);
  color: #fff;
  font-size: 11px;
  line-height: 20px;
}

.overlay {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--ph-space-6) var(--ph-space-3) var(--ph-space-2);
  background: linear-gradient(to top, rgba(0, 0, 0, 0.65), rgba(0, 0, 0, 0));
  color: #fff;
  opacity: 0;
  transition: opacity 200ms ease-out;
}

.name {
  font-size: 13px;
}

.meta {
  font-size: 11px;
  opacity: 0.85;
}

@media (max-width: 767.98px) {
  .overlay {
    opacity: 1;
    padding: var(--ph-space-6) var(--ph-space-2) var(--ph-space-2);
  }
}
</style>
