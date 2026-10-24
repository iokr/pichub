<script setup lang="ts">
import type { Image } from '@/types/api'
import ImageCard from './ImageCard.vue'

withDefaults(
  defineProps<{
    images: Image[]
    selectable?: boolean
    selection?: number[]
  }>(),
  {
    selectable: false,
    selection: () => [],
  },
)

const emit = defineEmits<{
  (event: 'open', image: Image): void
  (event: 'select', id: number): void
}>()
</script>

<template>
  <div class="image-grid">
    <ImageCard
      v-for="image in images"
      :key="image.id"
      :image="image"
      :selectable="selectable"
      :selected="selection.includes(image.id)"
      @open="emit('open', $event)"
      @select="emit('select', $event)"
    />
  </div>
</template>

<style scoped lang="scss">
.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--ph-space-4);
}

@media (max-width: 767.98px) {
  .image-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--ph-space-2);
  }
}
</style>
