<script setup lang="ts">
import { computed } from 'vue'

import CopyButton from '@/components/common/CopyButton.vue'
import type { Image } from '@/types/api'
import { buildBBCode, buildHTML, buildMarkdown } from '@/utils/download'

const props = defineProps<{ image: Pick<Image, 'url' | 'thumb_url' | 'name'> }>()

const rows = computed(() => {
  const value = [
    { label: 'URL', text: props.image.url },
    { label: 'Markdown', text: buildMarkdown(props.image.url, props.image.name) },
    { label: 'HTML', text: buildHTML(props.image.url, props.image.name) },
    { label: 'BBCode', text: buildBBCode(props.image.url) },
  ]
  if (props.image.thumb_url) {
    value.push({ label: '缩略图', text: props.image.thumb_url })
  }
  return value
})
</script>

<template>
  <div class="link-list">
    <div v-for="row in rows" :key="row.label" class="link-row">
      <span class="label">{{ row.label }}</span>
      <el-input :model-value="row.text" readonly size="small" class="input" />
      <CopyButton :text="row.text" label="复制" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.link-list {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-2);
}

.link-row {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
}

.label {
  flex: 0 0 72px;
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.input {
  flex: 1;
  min-width: 0;

  :deep(.el-input__inner) {
    font-family: var(--ph-font-mono);
    font-size: 12px;
  }
}
</style>
