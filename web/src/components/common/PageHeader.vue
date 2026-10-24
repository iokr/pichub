<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <div class="page-header">
    <div class="heading">
      <h2>{{ title }}</h2>
      <p v-if="description" class="description">{{ description }}</p>
    </div>
    <div class="actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.page-header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--ph-space-3);
  margin-bottom: var(--ph-space-6);
}

.heading {
  min-width: 0;
}

h2 {
  font-size: 24px;
}

.description {
  margin-top: var(--ph-space-1);
  font-size: 13px;
  color: var(--ph-text-secondary);
}

.actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--ph-space-2);
}

@media (max-width: 767.98px) {
  h2 {
    font-size: 20px;
  }

  .page-header {
    margin-bottom: var(--ph-space-4);
  }
}
</style>
