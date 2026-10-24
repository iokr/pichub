<script setup lang="ts">
import { useCopy } from '@/composables/useCopy'

const props = withDefaults(
  defineProps<{
    text: string
    label?: string
    size?: 'small' | 'default' | 'large'
    type?: 'primary' | 'default' | 'success' | 'warning' | 'info' | 'danger'
    link?: boolean
  }>(),
  {
    label: '复制',
    size: 'small',
    type: 'primary',
    link: true,
  },
)

const { copy } = useCopy()
</script>

<template>
  <el-button :size="size" :type="type" :link="link" @click.stop="copy(props.text)">
    {{ label }}
  </el-button>
</template>
