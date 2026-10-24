<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Folder, Plus, Search, Share } from '@element-plus/icons-vue'

import { albumApi, type AlbumCreatePayload } from '@/api/album'
import EmptyState from '@/components/common/EmptyState.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import ShareDialog from '@/components/share/ShareDialog.vue'
import type { Album } from '@/types/api'
import { fromNow } from '@/utils/format'

const router = useRouter()

const albums = ref<Album[]>([])
const loading = ref(false)
const keyword = ref('')
const dialogOpen = ref(false)
const editing = ref<Album | null>(null)
const submitting = ref(false)
const shareOpen = ref(false)
const shareTarget = ref<Album | null>(null)
const form = reactive<AlbumCreatePayload>({ name: '', description: '', is_public: true })

const filtered = computed(() => {
  const value = keyword.value.trim().toLowerCase()
  if (!value) {
    return albums.value
  }
  return albums.value.filter((album) => album.name.toLowerCase().includes(value))
})

async function load(): Promise<void> {
  loading.value = true
  try {
    albums.value = await albumApi.list()
  } finally {
    loading.value = false
  }
}

function openCreate(): void {
  editing.value = null
  form.name = ''
  form.description = ''
  form.is_public = true
  dialogOpen.value = true
}

function openEdit(album: Album): void {
  editing.value = album
  form.name = album.name
  form.description = album.description
  form.is_public = album.is_public
  dialogOpen.value = true
}

async function submit(): Promise<void> {
  if (!form.name.trim()) {
    ElMessage.warning('请填写相册名称')
    return
  }
  submitting.value = true
  try {
    if (editing.value) {
      await albumApi.update(editing.value.id, {
        name: form.name.trim(),
        description: form.description,
        is_public: form.is_public,
      })
      ElMessage.success('相册已更新')
    } else {
      await albumApi.create({ ...form, name: form.name.trim() })
      ElMessage.success('相册已创建')
    }
    dialogOpen.value = false
    await load()
  } finally {
    submitting.value = false
  }
}

async function remove(album: Album): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `确定删除相册「${album.name}」吗？相册内的图片不会被删除，只会移出相册。`,
      '删除相册',
      { type: 'warning' },
    )
  } catch {
    return
  }
  await albumApi.remove(album.id)
  ElMessage.success('相册已删除')
  await load()
}

function openShare(album: Album): void {
  shareTarget.value = album
  shareOpen.value = true
}

function openAlbum(album: Album): void {
  void router.push(`/albums/${album.id}`)
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="相册" :description="`共 ${albums.length} 个相册`">
      <template #actions>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建相册</el-button>
      </template>
    </PageHeader>

    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索相册" :prefix-icon="Search" clearable class="search" />
    </div>

    <div v-if="filtered.length" class="album-grid">
      <div v-for="album in filtered" :key="album.id" class="album-card">
        <div class="cover" @click="openAlbum(album)">
          <img v-if="album.cover_url" :src="album.cover_url" :alt="album.name" loading="lazy" />
          <div v-else class="cover-placeholder">
            <el-icon :size="36"><Folder /></el-icon>
          </div>
          <span class="badge" :class="album.is_public ? 'public' : 'private'">
            {{ album.is_public ? '公开' : '私有' }}
          </span>
        </div>
        <div class="body">
          <div class="title-row">
            <h3 :title="album.name">{{ album.name }}</h3>
            <span class="count">{{ album.image_count }} 张</span>
          </div>
          <p class="description">{{ album.description || '暂无描述' }}</p>
          <div class="footer">
            <span class="time">{{ fromNow(album.updated_at || album.created_at) }}更新</span>
            <div class="ops">
              <el-button text size="small" :icon="Share" @click="openShare(album)">分享</el-button>
              <el-button text size="small" :icon="Edit" @click="openEdit(album)">编辑</el-button>
              <el-button text size="small" type="danger" @click="remove(album)">删除</el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <el-skeleton v-else-if="loading" animated :rows="4" />

    <EmptyState v-else description="还没有相册，创建一个来整理图片吧">
      <template #actions>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建相册</el-button>
      </template>
    </EmptyState>

    <el-dialog v-model="dialogOpen" :title="editing ? '编辑相册' : '新建相册'" width="440px">
      <el-form label-position="top">
        <el-form-item label="相册名称" required>
          <el-input v-model="form.name" maxlength="64" show-word-limit placeholder="例如：旅行照片" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" maxlength="255" show-word-limit />
        </el-form-item>
        <el-form-item label="可见性">
          <el-switch v-model="form.is_public" active-text="公开" inactive-text="私有" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogOpen = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">保存</el-button>
      </template>
    </el-dialog>

    <ShareDialog v-model="shareOpen" :album="shareTarget" />
  </div>
</template>

<style scoped lang="scss">
.toolbar {
  margin-bottom: var(--ph-space-4);
}

.search {
  width: 240px;
}

.album-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--ph-space-4);
}

.album-card {
  overflow: hidden;
  background: var(--ph-fill-card);
  border: 1px solid var(--ph-border-light);
  border-radius: var(--ph-radius-lg);
  transition: box-shadow 0.2s ease, transform 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--ph-shadow-md);
  }
}

.cover {
  position: relative;
  aspect-ratio: 16 / 10;
  background: var(--ph-fill);
  cursor: pointer;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.cover-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  color: var(--ph-text-tertiary);
}

.badge {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 2px 8px;
  font-size: 11px;
  border-radius: 999px;

  &.public {
    color: var(--ph-success);
    background: rgb(0 180 42 / 12%);
  }

  &.private {
    color: var(--ph-warning);
    background: rgb(255 125 0 / 12%);
  }
}

.body {
  padding: var(--ph-space-3) var(--ph-space-4) var(--ph-space-4);
}

.title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ph-space-2);

  h3 {
    font-size: 15px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.count {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.description {
  height: 20px;
  margin-top: var(--ph-space-1);
  font-size: 12px;
  color: var(--ph-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ph-space-2);
  margin-top: var(--ph-space-2);
}

.time {
  font-size: 12px;
  color: var(--ph-text-tertiary);
}

.ops {
  display: flex;
  gap: 0;
}

@media (max-width: 767.98px) {
  .search {
    width: 100%;
  }

  .album-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--ph-space-2);
  }

  .ops {
    flex-wrap: wrap;
  }
}
</style>
