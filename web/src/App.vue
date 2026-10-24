<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import BlankLayout from '@/layouts/BlankLayout.vue'
import DefaultLayout from '@/layouts/DefaultLayout.vue'

const route = useRoute()
const layout = computed(() => (route.meta.layout === 'blank' ? BlankLayout : DefaultLayout))
</script>

<template>
  <component :is="layout">
    <router-view v-slot="{ Component }">
      <transition name="page" mode="out-in">
        <component :is="Component" />
      </transition>
    </router-view>
  </component>
</template>
