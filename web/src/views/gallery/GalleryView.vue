<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Upload } from '@element-plus/icons-vue'

import { albumApi } from '@/api/album'
import { tagApi } from '@/api/tag'
import EmptyState from '@/components/common/EmptyState.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import ImageDetailDrawer from '@/components/image/ImageDetailDrawer.vue'
import ImageGrid from '@/components/image/ImageGrid.vue'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import { useImageStore } from '@/stores/image'
import type { Album, Image, Tag } from '@/types/api'

const store = useImageStore()
const route = useRoute()
const router = useRouter()

const albums = ref<Album[]>([])
const tags = ref<Tag[]>([])
const keyword = ref('')
const albumId = ref<number | undefined>()
const tagId = ref<number | undefined>()
const visibility = ref<'all' | 'public' | 'private'>('all')
const selectMode = ref(false)
const detailOpen = ref(false)
const detailImage = ref<Image | null>(null)
const moveDialog = ref(false)
const moveAlbumId = ref<number | undefined>()
const sentinel = ref<HTMLElement | null>(null)

const selectedCount = computed(() => store.selection.length)

useInfiniteScroll(sentinel, () => store.loadMore(), {
  disabled: () => store.loading || store.finished,
})

function currentQuery() {
  return {
    keyword: keyword.value.trim() || undefined,
    album_id: albumId.value,
    tag_id: tagId.value,
    is_public: visibility.value === 'all' ? undefined : visibility.value === 'public',
  }
}

async function applyFilters(): Promise<void> {
  store.clearSelection()
  await store.applyFilter(currentQuery())
}

async function search(): Promise<void> {
  const value = keyword.value.trim()
  await router.replace({ path: '/gallery', query: value ? { keyword: value } : {} })
  await applyFilters()
}

function resetFilters(): void {
  keyword.value = ''
  albumId.value = undefined
  tagId.value = undefined
  visibility.value = 'all'
  void applyFilters()
}

function toggleSelectMode(): void {
  selectMode.value = !selectMode.value
  if (!selectMode.value) {
    store.clearSelection()
  }
}

function openImage(image: Image): void {
  if (selectMode.value) {
    store.toggleSelect(image.id)
    return
  }
  detailImage.value = image
  detailOpen.value = true
}

async function batch(action: 'delete' | 'public' | 'private'): Promise<void> {
  if (!selectedCount.value) {
    ElMessage.warning('请先选择图片')
    return
  }
  if (action === 'delete') {
    try {
      await ElMessageBox.confirm(`确定将选中的 ${selectedCount.value} 张图片移入回收站吗？`, '移入回收站', {
        type: 'warning',
      })
    } catch {
      return
    }
  }
  const affected = await store.batch(action)
  ElMessage.success(`已处理 ${affected} 张图片`)
}

function openMoveDialog(): void {
  if (!selectedCount.value) {
    ElMessage.warning('请先选择图片')
    return
  }
  moveAlbumId.value = undefined
  moveDialog.value = true
}

async function confirmMove(): Promise<void> {
  if (!moveAlbumId.value) {
    ElMessage.warning('请选择目标相册')
    return
  }
  const affected = await store.batch('move', moveAlbumId.value)
  moveDialog.value = false
  ElMessage.success(`已移动 ${affected} 张图片`)
}

function onUpdated(image: Image): void {
  store.patchImage(image)
  detailImage.value = image
}

function onRemoved(id: number): void {
  store.removeLocal([id])
  detailOpen.value = false
}

onMounted(async () => {
  const queryKeyword = typeof route.query.keyword === 'string' ? route.query.keyword : ''
  keyword.value = queryKeyword
  const queryAlbum = Number(route.query.album_id)
  if (queryAlbum) {
    albumId.value = queryAlbum
  }
  const [albumList, tagList] = await Promise.all([
    albumApi.list().catch(() => [] as Album[]),
    tagApi.list().catch(() => [] as Tag[]),
  ])
  albums.value = albumList
  tags.value = tagList
  store.resetQuery()
  await store.applyFilter(currentQuery())
})
</script>

<template>
  <div>
    <PageHeader title="图库" :description="`共 ${store.total} 张图片`">
      <template #actions>
        <el-button :type="selectMode ? 'primary' : 'default'" @click="toggleSelectMode">
          {{ selectMode ? '退出多选' : '批量管理' }}
        </el-button>
        <el-button type="primary" :icon="Upload" @click="router.push('/')">上传图片</el-button>
      </template>
    </PageHeader>

    <div class="toolbar">
      <el-input
        v-model="keyword"
        placeholder="搜索图片名称"
        :prefix-icon="Search"
        clearable
        class="search"
        @keyup.enter="search"
        @clear="applyFilters"
      />
      <el-select v-model="albumId" placeholder="全部相册" clearable class="filter" @change="applyFilters">
        <el-option v-for="album in albums" :key="album.id" :label="album.name" :value="album.id" />
      </el-select>
      <el-select v-model="tagId" placeholder="全部标签" clearable class="filter" @change="applyFilters">
        <el-option v-for="tag in tags" :key="tag.id" :label="tag.name" :value="tag.id" />
      </el-select>
      <el-select v-model="visibility" class="filter" @change="applyFilters">
        <el-option label="全部可见性" value="all" />
        <el-option label="公开" value="public" />
        <el-option label="私有" value="private" />
      </el-select>
      <el-button text @click="resetFilters">重置</el-button>
    </div>

    <div v-if="selectMode" class="batch-bar">
      <span class="count">已选 {{ selectedCount }} 张</span>
      <div class="batch-actions">
        <el-button size="small" @click="store.selectAll">全选本页</el-button>
        <el-button size="small" @click="batch('public')">设为公开</el-button>
        <el-button size="small" @click="batch('private')">设为私有</el-button>
        <el-button size="small" @click="openMoveDialog">移动到相册</el-button>
        <el-button size="small" type="danger" plain @click="batch('delete')">移入回收站</el-button>
      </div>
    </div>

    <ImageGrid
      v-if="store.items.length"
      :images="store.items"
      :selectable="selectMode"
      :selection="store.selection"
      @open="openImage"
      @select="store.toggleSelect"
    />

    <EmptyState v-else-if="store.isEmpty" description="还没有图片，去上传第一张吧">
      <template #actions>
        <el-button type="primary" @click="router.push('/')">上传图片</el-button>
      </template>
    </EmptyState>

    <div v-if="store.loading" class="loading">
      <el-skeleton animated :rows="3" />
    </div>

    <div ref="sentinel" class="sentinel"></div>
    <p v-if="store.finished && store.items.length" class="finished">已经到底了</p>

    <ImageDetailDrawer v-model="detailOpen" :image="detailImage" @updated="onUpdated" @removed="onRemoved" />

    <el-dialog v-model="moveDialog" title="移动到相册" width="360px">
      <el-select v-model="moveAlbumId" placeholder="选择目标相册" class="w-full">
        <el-option v-for="album in albums" :key="album.id" :label="album.name" :value="album.id" />
      </el-select>
      <template #footer>
        <el-button @click="moveDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmMove">确定移动</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--ph-space-2);
  margin-bottom: var(--ph-space-4);
}

.search {
  width: 240px;
}

.filter {
  width: 140px;
}

.batch-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--ph-space-2);
  margin-bottom: var(--ph-space-4);
  padding: var(--ph-space-2) var(--ph-space-3);
  border: 1px solid var(--ph-primary-light-7);
  border-radius: var(--ph-radius-md);
  background: var(--ph-primary-light-9);

  .count {
    font-size: 13px;
    color: var(--ph-primary);
  }
}

.batch-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ph-space-1);
}

.loading {
  margin-top: var(--ph-space-4);
}

.sentinel {
  height: 1px;
}

.finished {
  margin-top: var(--ph-space-4);
  text-align: center;
  font-size: 12px;
  color: var(--ph-text-secondary);
}

@media (max-width: 767.98px) {
  .search,
  .filter {
    width: 100%;
  }
}
</style>
