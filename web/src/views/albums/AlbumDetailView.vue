<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Edit, Share, Upload } from '@element-plus/icons-vue'

import { albumApi, type AlbumUpdatePayload } from '@/api/album'
import { imageApi } from '@/api/image'
import EmptyState from '@/components/common/EmptyState.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import ImageDetailDrawer from '@/components/image/ImageDetailDrawer.vue'
import ImageGrid from '@/components/image/ImageGrid.vue'
import ShareDialog from '@/components/share/ShareDialog.vue'
import type { Album, Image } from '@/types/api'
import { fromNow } from '@/utils/format'

const route = useRoute()
const router = useRouter()

const albumId = computed(() => Number(route.params.id))
const album = ref<Album | null>(null)
const images = ref<Image[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(24)
const loading = ref(false)
const detailOpen = ref(false)
const detailImage = ref<Image | null>(null)
const shareOpen = ref(false)
const editOpen = ref(false)
const editName = ref('')
const editDescription = ref('')
const editPublic = ref(true)
const saving = ref(false)

async function loadAlbum(): Promise<void> {
  if (!albumId.value) {
    return
  }
  try {
    album.value = await albumApi.get(albumId.value)
  } catch {
    ElMessage.error('相册不存在或无权访问')
    void router.replace('/albums')
  }
}

async function loadImages(): Promise<void> {
  if (!albumId.value) {
    return
  }
  loading.value = true
  try {
    const data = await imageApi.list({ album_id: albumId.value, page: page.value, size: size.value })
    images.value = data.list ?? []
    total.value = data.total ?? 0
  } finally {
    loading.value = false
  }
}

function openImage(image: Image): void {
  detailImage.value = image
  detailOpen.value = true
}

function onUpdated(image: Image): void {
  const index = images.value.findIndex((item) => item.id === image.id)
  if (index >= 0) {
    images.value[index] = image
  }
  detailImage.value = image
}

function onRemoved(id: number): void {
  images.value = images.value.filter((item) => item.id !== id)
  total.value = Math.max(0, total.value - 1)
  detailOpen.value = false
}

async function removeFromAlbum(image: Image): Promise<void> {
  await imageApi.update(image.id, { clear_album: true })
  images.value = images.value.filter((item) => item.id !== image.id)
  total.value = Math.max(0, total.value - 1)
  detailOpen.value = false
  ElMessage.success('已移出相册')
}

function openEdit(): void {
  if (!album.value) {
    return
  }
  editName.value = album.value.name
  editDescription.value = album.value.description
  editPublic.value = album.value.is_public
  editOpen.value = true
}

async function saveEdit(): Promise<void> {
  if (!album.value || !editName.value.trim()) {
    ElMessage.warning('请填写相册名称')
    return
  }
  saving.value = true
  try {
    const payload: AlbumUpdatePayload = {
      name: editName.value.trim(),
      description: editDescription.value,
      is_public: editPublic.value,
    }
    album.value = await albumApi.update(album.value.id, payload)
    editOpen.value = false
    ElMessage.success('相册已更新')
  } finally {
    saving.value = false
  }
}

async function removeAlbum(): Promise<void> {
  if (!album.value) {
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除相册「${album.value.name}」吗？相册内的图片不会被删除。`, '删除相册', {
      type: 'warning',
    })
  } catch {
    return
  }
  await albumApi.remove(album.value.id)
  ElMessage.success('相册已删除')
  void router.replace('/albums')
}

function handlePageChange(value: number): void {
  page.value = value
  void loadImages()
}

watch(albumId, () => {
  page.value = 1
  void loadAlbum()
  void loadImages()
})

onMounted(() => {
  void loadAlbum()
  void loadImages()
})
</script>

<template>
  <div>
    <PageHeader
      :title="album?.name || '相册详情'"
      :description="album ? `${album.image_count} 张图片 · ${album.is_public ? '公开' : '私有'} · ${fromNow(album.updated_at)}更新` : ''"
    >
      <template #actions>
        <el-button :icon="ArrowLeft" @click="router.push('/albums')">返回相册</el-button>
        <el-button :icon="Edit" @click="openEdit">编辑</el-button>
        <el-button :icon="Share" :disabled="!album" @click="shareOpen = true">分享</el-button>
        <el-button type="primary" :icon="Upload" @click="router.push({ path: '/', query: { album: albumId } })">
          上传到相册
        </el-button>
      </template>
    </PageHeader>

    <div v-if="album?.description" class="description">{{ album.description }}</div>

    <ImageGrid v-if="images.length" :images="images" @open="openImage" />

    <el-skeleton v-else-if="loading" animated :rows="4" />

    <EmptyState v-else description="这个相册还没有图片">
      <template #actions>
        <el-button type="primary" @click="router.push({ path: '/', query: { album: albumId } })">上传图片</el-button>
      </template>
    </EmptyState>

    <div v-if="total > size" class="pager">
      <el-pagination
        layout="prev, pager, next"
        :current-page="page"
        :page-size="size"
        :total="total"
        background
        @current-change="handlePageChange"
      />
    </div>

    <div class="danger-zone">
      <span>删除相册不会删除其中的图片</span>
      <el-button text type="danger" @click="removeAlbum">删除相册</el-button>
    </div>

    <ImageDetailDrawer v-model="detailOpen" :image="detailImage" @updated="onUpdated" @removed="onRemoved">
      <template #extra>
        <el-button v-if="detailImage" text type="danger" @click="removeFromAlbum(detailImage)">移出相册</el-button>
      </template>
    </ImageDetailDrawer>

    <ShareDialog v-model="shareOpen" :album="album" />

    <el-dialog v-model="editOpen" title="编辑相册" width="440px">
      <el-form label-position="top">
        <el-form-item label="相册名称" required>
          <el-input v-model="editName" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editDescription" type="textarea" :rows="3" maxlength="255" show-word-limit />
        </el-form-item>
        <el-form-item label="可见性">
          <el-switch v-model="editPublic" active-text="公开" inactive-text="私有" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editOpen = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.description {
  margin-bottom: var(--ph-space-4);
  padding: var(--ph-space-3);
  font-size: 13px;
  color: var(--ph-text-secondary);
  background: var(--ph-fill);
  border-radius: var(--ph-radius-md);
}

.pager {
  display: flex;
  justify-content: center;
  margin-top: var(--ph-space-6);
}

.danger-zone {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: var(--ph-space-8);
  padding: var(--ph-space-3) var(--ph-space-4);
  font-size: 12px;
  color: var(--ph-text-secondary);
  background: var(--ph-fill);
  border-radius: var(--ph-radius-md);
}
</style>
