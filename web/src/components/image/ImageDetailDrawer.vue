<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Download } from '@element-plus/icons-vue'

import { albumApi } from '@/api/album'
import { imageApi } from '@/api/image'
import { tagApi } from '@/api/tag'
import { useDevice } from '@/composables/useDevice'
import type { Album, Image, Tag } from '@/types/api'
import { formatBytes, formatDate } from '@/utils/format'
import ImageLinkList from './ImageLinkList.vue'
import RefererList from './RefererList.vue'

const props = defineProps<{
  modelValue: boolean
  image: Image | null
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'updated', image: Image): void
  (event: 'removed', id: number): void
}>()

const { isMobile } = useDevice()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const albums = ref<Album[]>([])
const tags = ref<Tag[]>([])
const activeTab = ref('links')
const saving = ref(false)
const optionsLoaded = ref(false)

const form = reactive({
  display_name: '',
  album_id: undefined as number | undefined,
  tags: [] as string[],
  is_public: true,
})

watch(
  () => props.image,
  (image) => {
    if (!image) {
      return
    }
    form.display_name = image.name || image.origin_name
    form.album_id = image.album_id ?? undefined
    form.tags = (image.tags ?? []).map((tag) => tag.name)
    form.is_public = image.is_public
    activeTab.value = 'links'
  },
  { immediate: true },
)

watch(visible, async (open) => {
  if (!open || optionsLoaded.value) {
    return
  }
  optionsLoaded.value = true
  const [albumList, tagList] = await Promise.all([
    albumApi.list().catch(() => [] as Album[]),
    tagApi.list().catch(() => [] as Tag[]),
  ])
  albums.value = albumList
  tags.value = tagList
})

async function save(): Promise<void> {
  if (!props.image) {
    return
  }
  saving.value = true
  try {
    const updated = await imageApi.update(props.image.id, {
      display_name: form.display_name.trim() || props.image.origin_name,
      album_id: form.album_id,
      clear_album: form.album_id === undefined,
      tags: form.tags,
      is_public: form.is_public,
    })
    emit('updated', updated)
    ElMessage.success('已保存')
  } finally {
    saving.value = false
  }
}

async function remove(): Promise<void> {
  if (!props.image) {
    return
  }
  try {
    await ElMessageBox.confirm('图片将被移入回收站，可在回收站中恢复。', '移入回收站', { type: 'warning' })
  } catch {
    return
  }
  await imageApi.remove(props.image.id)
  ElMessage.success('已移入回收站')
  emit('removed', props.image.id)
  visible.value = false
}

function download(): void {
  if (!props.image) {
    return
  }
  window.open(props.image.url, '_blank', 'noopener')
}
</script>

<template>
  <el-drawer v-model="visible" :size="isMobile ? '100%' : 520" title="图片详情">
    <template v-if="image">
      <div class="preview">
        <el-image :src="image.url" :preview-src-list="[image.url]" fit="contain" class="preview-image" />
      </div>

      <div class="meta-grid">
        <span class="key">尺寸</span>
        <span>{{ image.width }}×{{ image.height }}</span>
        <span class="key">体积</span>
        <span>{{ formatBytes(image.size) }}</span>
        <span class="key">格式</span>
        <span>{{ image.ext.toUpperCase() }}</span>
        <span class="key">上传时间</span>
        <span>{{ formatDate(image.created_at) }}</span>
        <span class="key">引用次数</span>
        <span>{{ image.ref_count }}</span>
      </div>

      <el-tabs v-model="activeTab" class="tabs">
        <el-tab-pane label="链接" name="links">
          <ImageLinkList :image="image" />
        </el-tab-pane>

        <el-tab-pane label="编辑" name="edit">
          <el-form label-width="72px" label-position="left">
            <el-form-item label="名称">
              <el-input v-model="form.display_name" maxlength="80" show-word-limit />
            </el-form-item>
            <el-form-item label="相册">
              <el-select v-model="form.album_id" placeholder="不加入相册" clearable class="w-full">
                <el-option v-for="album in albums" :key="album.id" :label="album.name" :value="album.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="标签">
              <el-select v-model="form.tags" multiple filterable allow-create default-first-option class="w-full" placeholder="选择或输入标签">
                <el-option v-for="tag in tags" :key="tag.id" :label="tag.name" :value="tag.name" />
              </el-select>
            </el-form-item>
            <el-form-item label="公开">
              <el-switch v-model="form.is_public" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="`引用来源 (${image.ref_count})`" name="referers">
          <RefererList :image-id="image.id" />
        </el-tab-pane>
      </el-tabs>
    </template>

    <template #footer>
      <template v-if="image">
        <slot name="extra" />
        <el-button :icon="Download" @click="download">下载原图</el-button>
        <el-button type="danger" plain :icon="Delete" @click="remove">移入回收站</el-button>
        <el-button v-if="activeTab === 'edit'" type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </template>
  </el-drawer>
</template>

<style scoped lang="scss">
.preview {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 260px;
  border-radius: var(--ph-radius-lg);
  background: var(--ph-fill);
  overflow: hidden;
}

.preview-image {
  max-width: 100%;
  max-height: 100%;
}

.meta-grid {
  display: grid;
  grid-template-columns: 80px 1fr 80px 1fr;
  gap: var(--ph-space-2) var(--ph-space-3);
  margin: var(--ph-space-4) 0;
  font-size: 13px;

  .key {
    color: var(--ph-text-secondary);
  }
}

.tabs {
  margin-top: var(--ph-space-4);
}
</style>
