<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { UploadFilled } from '@element-plus/icons-vue'

const props = withDefaults(
  defineProps<{
    accept?: string
    multiple?: boolean
    disabled?: boolean
    hint?: string
  }>(),
  {
    accept: 'image/*',
    multiple: true,
    disabled: false,
    hint: '',
  },
)

const emit = defineEmits<{ (event: 'files', files: File[]): void }>()

const inputRef = ref<HTMLInputElement | null>(null)
const dragging = ref(false)
let dragDepth = 0

function open(): void {
  if (props.disabled) {
    return
  }
  inputRef.value?.click()
}

function onChange(event: Event): void {
  const input = event.target as HTMLInputElement
  const files = input.files ? Array.from(input.files) : []
  if (files.length) {
    emit('files', files)
  }
  input.value = ''
}

function onDragEnter(event: DragEvent): void {
  event.preventDefault()
  dragDepth += 1
  dragging.value = true
}

function onDragOver(event: DragEvent): void {
  event.preventDefault()
}

function onDragLeave(event: DragEvent): void {
  event.preventDefault()
  dragDepth -= 1
  if (dragDepth <= 0) {
    dragDepth = 0
    dragging.value = false
  }
}

function onDrop(event: DragEvent): void {
  event.preventDefault()
  dragDepth = 0
  dragging.value = false
  if (props.disabled || !event.dataTransfer) {
    return
  }
  const files = Array.from(event.dataTransfer.files).filter(
    (file) => file.type.startsWith('image/') || file.type === '',
  )
  if (files.length) {
    emit('files', files)
  }
}

function onPaste(event: ClipboardEvent): void {
  const items = event.clipboardData?.items
  if (!items) {
    return
  }
  const files: File[] = []
  for (const item of Array.from(items)) {
    if (item.kind === 'file') {
      const file = item.getAsFile()
      if (file) {
        files.push(file)
      }
    }
  }
  if (files.length) {
    emit('files', files)
  }
}

onMounted(() => window.addEventListener('paste', onPaste))
onBeforeUnmount(() => window.removeEventListener('paste', onPaste))
</script>

<template>
  <div
    class="dropzone"
    :class="{ dragging, disabled }"
    role="button"
    tabindex="0"
    @click="open"
    @keydown.enter="open"
    @dragenter="onDragEnter"
    @dragover="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
  >
    <input
      ref="inputRef"
      type="file"
      :accept="accept"
      :multiple="multiple"
      :disabled="disabled"
      hidden
      @change="onChange"
    />
    <el-icon :size="56" class="icon"><UploadFilled /></el-icon>
    <div class="title">拖拽图片到这里，或点击上传</div>
    <div class="hint">{{ hint || '支持 JPG / PNG / GIF / WebP，也可直接粘贴截图' }}</div>
  </div>
</template>

<style scoped lang="scss">
.dropzone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--ph-space-2);
  min-height: 220px;
  padding: var(--ph-space-8);
  border: 2px dashed var(--ph-border);
  border-radius: var(--ph-radius-lg);
  background: var(--ph-fill);
  text-align: center;
  cursor: pointer;
  transition: border-color 150ms ease-out, background-color 150ms ease-out;

  &:hover,
  &.dragging {
    border-color: var(--ph-primary);
    background: var(--ph-primary-light-9);
  }

  &.disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }
}

.icon {
  color: var(--ph-primary);
}

.title {
  font-size: 16px;
  color: var(--ph-text-primary);
}

.hint {
  font-size: 12px;
  color: var(--ph-text-secondary);
}
</style>
