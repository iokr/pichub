<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Setting } from '@element-plus/icons-vue'

import { albumApi } from '@/api/album'
import { imageApi } from '@/api/image'
import { storageApi } from '@/api/storage'
import { tagApi } from '@/api/tag'
import { userApi } from '@/api/user'
import PageHeader from '@/components/common/PageHeader.vue'
import ImageGrid from '@/components/image/ImageGrid.vue'
import ImageDetailDrawer from '@/components/image/ImageDetailDrawer.vue'
import UploadCard from '@/components/upload/UploadCard.vue'
import UploadDropzone from '@/components/upload/UploadDropzone.vue'
import UploadParams from '@/components/upload/UploadParams.vue'
import { useSettingStore } from '@/stores/setting'
import { useUploadStore } from '@/stores/upload'
import { useUserStore } from '@/stores/user'
import type { Album, Image, StorageStrategy, Tag, UploadOptions } from '@/types/api'
import { formatBytes, percent } from '@/utils/format'

const setting = useSettingStore()
const user = useUserStore()
const upload = useUploadStore()
const route = useRoute()

const albums = ref<Album[]>([])
const tags = ref<Tag[]>([])
const storages = ref<StorageStrategy[]>([])
const recent = ref<Image[]>([])
const todayUploads = ref(0)
const showParams = ref(false)
const detailOpen = ref(false)
const detailImage = ref<Image | null>(null)

const options = computed<UploadOptions>({
  get: () => upload.options,
  set: (value) => upload.saveOptions(value),
})

const maxSize = computed(() => user.user?.max_file_size || setting.config.max_file_size || 10 * 1024 * 1024)
const usedPercent = computed(() => percent(user.user?.used_bytes, user.user?.quota_bytes))
const hint = computed(
  () =>
    `支持 ${setting.config.allowed_extensions.map((ext) => ext.toUpperCase()).join(' / ')}，单张最大 ${formatBytes(
      maxSize.value,
    )}`,
)

const orderedTasks = computed(() => [...upload.tasks].reverse())

function handleFiles(files: File[]): void {
  const valid: File[] = []
  const allowed = setting.config.allowed_extensions.map((ext) => ext.toLowerCase())
  for (const file of files) {
    if (file.size > maxSize.value) {
      ElMessage.warning(`${file.name} 超过大小限制 ${formatBytes(maxSize.value)}`)
      continue
    }
    const ext = file.name.includes('.') ? file.name.split('.').pop()!.toLowerCase() : ''
    if (allowed.length && !allowed.includes(ext)) {
      ElMessage.warning(`${file.name} 的格式不受支持`)
      continue
    }
    valid.push(file)
  }
  if (valid.length) {
    upload.enqueue(valid)
  }
}

async function loadRecent(): Promise<void> {
  try {
    const page = await imageApi.list({ page: 1, size: 8 })
    recent.value = page.list ?? []
  } catch {
    /* 最近上传加载失败不阻塞工作台 */
  }
}

async function loadStats(): Promise<void> {
  try {
    const stats = await userApi.stats()
    todayUploads.value = stats.today_uploads
  } catch {
    /* 统计加载失败不阻塞工作台 */
  }
}

function openImage(image: Image): void {
  detailImage.value = image
  detailOpen.value = true
}

watch(
  () => upload.completedTick,
  () => {
    void loadRecent()
    void loadStats()
    void user.fetchProfile().catch(() => undefined)
  },
)

onMounted(async () => {
  const albumId = Number(route.query.album)
  if (albumId) {
    upload.saveOptions({ ...upload.options, album_id: albumId })
  }
  const [albumList, tagList, storageList] = await Promise.all([
    albumApi.list().catch(() => [] as Album[]),
    tagApi.list().catch(() => [] as Tag[]),
    storageApi.list().catch(() => [] as StorageStrategy[]),
  ])
  albums.value = albumList
  tags.value = tagList
  storages.value = storageList
  void loadRecent()
  void loadStats()
})
</script>

<template>
  <div>
    <PageHeader title="上传工作台" description="拖拽、粘贴或点击选择图片，上传完成后可一键复制多种格式链接">
      <template #actions>
        <el-button :icon="Setting" @click="showParams = !showParams">
          {{ showParams ? '收起参数' : '上传参数' }}
        </el-button>
      </template>
    </PageHeader>

    <div class="workspace">
      <section class="main-column">
        <div class="panel">
          <UploadDropzone :hint="hint" @files="handleFiles" />

          <el-collapse-transition>
            <div v-show="showParams" class="params">
              <UploadParams
                v-model="options"
                :albums="albums"
                :tag-options="tags"
                :storages="storages"
                :show-storage="user.isAdmin || storages.length > 0"
              />
            </div>
          </el-collapse-transition>
        </div>

        <div v-if="upload.tasks.length" class="panel">
          <div class="panel-header">
            <h3>上传队列（{{ upload.activeCount || upload.tasks.length }}）</h3>
            <el-button text size="small" @click="upload.clearFinished()">清除已完成</el-button>
          </div>
          <div class="task-list">
            <UploadCard
              v-for="task in orderedTasks"
              :key="task.id"
              :task="task"
              @cancel="upload.cancel"
              @retry="upload.retry"
              @remove="upload.removeTask"
            />
          </div>
        </div>
      </section>

      <aside class="side-column">
        <div class="panel quota">
          <h3>存储用量</h3>
          <el-progress :percentage="usedPercent" :stroke-width="8" :show-text="false" />
          <div class="quota-text">
            <span>{{ formatBytes(user.user?.used_bytes) }}</span>
            <span class="muted">/ {{ formatBytes(user.user?.quota_bytes) }}</span>
          </div>
          <div class="quota-meta">
            <span>今日上传 {{ todayUploads }}</span>
            <span>单张上限 {{ formatBytes(maxSize) }}</span>
          </div>
        </div>

        <div class="panel">
          <div class="panel-header">
            <h3>最近上传</h3>
            <router-link to="/gallery" class="more">查看全部</router-link>
          </div>
          <ImageGrid v-if="recent.length" :images="recent" @open="openImage" />
          <el-empty v-else description="还没有上传记录" :image-size="72" />
        </div>
      </aside>
    </div>

    <ImageDetailDrawer v-model="detailOpen" :image="detailImage" @updated="loadRecent" @removed="loadRecent" />
  </div>
</template>

<style scoped lang="scss">
.workspace {
  display: grid;
  grid-template-columns: minmax(0, 2.2fr) minmax(0, 1fr);
  gap: var(--ph-space-6);
  align-items: start;
}

.main-column,
.side-column {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-4);
  min-width: 0;
}

.panel {
  padding: var(--ph-space-5);
  background: var(--ph-fill-card);
  border: 1px solid var(--ph-border-light);
  border-radius: var(--ph-radius-lg);
}

.params {
  margin-top: var(--ph-space-4);
  padding-top: var(--ph-space-4);
  border-top: 1px dashed var(--ph-border);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ph-space-3);

  h3 {
    font-size: 15px;
  }
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-2);
}

.quota {
  h3 {
    margin-bottom: var(--ph-space-3);
    font-size: 15px;
  }
}

.quota-text {
  display: flex;
  align-items: baseline;
  gap: var(--ph-space-1);
  margin-top: var(--ph-space-2);
  font-size: 20px;
  font-weight: 600;
  color: var(--ph-text-primary);
}

.quota-meta {
  display: flex;
  justify-content: space-between;
  margin-top: var(--ph-space-2);
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.muted {
  font-size: 12px;
  font-weight: 400;
  color: var(--ph-text-secondary);
}

.more {
  font-size: 12px;
}

:deep(.image-grid) {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--ph-space-2);
}

@media (max-width: 1199px) {
  .workspace {
    grid-template-columns: minmax(0, 1fr);
  }

  :deep(.image-grid) {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 767.98px) {
  :deep(.image-grid) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .panel {
    padding: var(--ph-space-4);
  }
}
</style>
