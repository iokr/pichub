<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, RefreshLeft } from '@element-plus/icons-vue'

import { imageApi } from '@/api/image'
import EmptyState from '@/components/common/EmptyState.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import type { Image } from '@/types/api'
import { formatBytes, formatDate } from '@/utils/format'

const images = ref<Image[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(24)
const loading = ref(false)
const selection = ref<number[]>([])

const selectedCount = computed(() => selection.value.length)
const allSelected = computed(() => images.value.length > 0 && selection.value.length === images.value.length)

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await imageApi.trash({ page: page.value, size: size.value })
    images.value = data.list ?? []
    total.value = data.total ?? 0
    selection.value = []
  } finally {
    loading.value = false
  }
}

function toggleSelect(id: number): void {
  const index = selection.value.indexOf(id)
  if (index >= 0) {
    selection.value.splice(index, 1)
  } else {
    selection.value.push(id)
  }
}

function toggleSelectAll(): void {
  selection.value = allSelected.value ? [] : images.value.map((image) => image.id)
}

async function restoreOne(image: Image): Promise<void> {
  await imageApi.restore(image.id)
  ElMessage.success(`已恢复 ${image.name}`)
  await load()
}

async function restoreSelected(): Promise<void> {
  if (!selectedCount.value) {
    ElMessage.warning('请先选择图片')
    return
  }
  const result = await imageApi.batch(selection.value, 'restore')
  ElMessage.success(`已恢复 ${result.affected} 张图片`)
  await load()
}

async function removeOne(image: Image): Promise<void> {
  try {
    await ElMessageBox.confirm(`永久删除后不可找回，确定删除 ${image.name} 吗？`, '永久删除', {
      type: 'warning',
      confirmButtonText: '永久删除',
    })
  } catch {
    return
  }
  await imageApi.hardDelete(image.id)
  ElMessage.success('已永久删除')
  await load()
}

async function removeSelected(): Promise<void> {
  if (!selectedCount.value) {
    ElMessage.warning('请先选择图片')
    return
  }
  try {
    await ElMessageBox.confirm(`确定永久删除选中的 ${selectedCount.value} 张图片吗？该操作不可恢复。`, '永久删除', {
      type: 'warning',
      confirmButtonText: '永久删除',
    })
  } catch {
    return
  }
  const ids = [...selection.value]
  const results = await Promise.allSettled(ids.map((id) => imageApi.hardDelete(id)))
  const failed = results.filter((item) => item.status === 'rejected').length
  if (failed) {
    ElMessage.warning(`${ids.length - failed} 张删除成功，${failed} 张删除失败`)
  } else {
    ElMessage.success(`已永久删除 ${ids.length} 张图片`)
  }
  await load()
}

async function clearTrash(): Promise<void> {
  try {
    await ElMessageBox.confirm('确定清空回收站吗？所有图片将被永久删除且无法恢复。', '清空回收站', {
      type: 'warning',
      confirmButtonText: '清空',
    })
  } catch {
    return
  }
  const result = await imageApi.clearTrash()
  ElMessage.success(`已清空 ${result.affected} 张图片`)
  page.value = 1
  await load()
}

function handlePageChange(value: number): void {
  page.value = value
  void load()
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="回收站" :description="`回收站内共 ${total} 张图片，仅保留最近删除的内容`">
      <template #actions>
        <el-button :disabled="!selectedCount" @click="restoreSelected">
          恢复选中{{ selectedCount ? `（${selectedCount}）` : '' }}
        </el-button>
        <el-button :disabled="!selectedCount" type="danger" plain @click="removeSelected">永久删除</el-button>
        <el-button type="danger" plain :icon="Delete" :disabled="!total" @click="clearTrash">清空回收站</el-button>
      </template>
    </PageHeader>

    <div v-if="images.length" class="select-bar">
      <el-checkbox :model-value="allSelected" @change="toggleSelectAll">全选本页</el-checkbox>
      <span class="muted">已选 {{ selectedCount }} 张</span>
    </div>

    <div v-if="images.length" class="trash-grid">
      <div
        v-for="image in images"
        :key="image.id"
        class="trash-card"
        :class="{ selected: selection.includes(image.id) }"
      >
        <div class="thumb" @click="toggleSelect(image.id)">
          <img :src="image.thumb_url || image.url" :alt="image.name" loading="lazy" />
          <el-checkbox
            class="check"
            :model-value="selection.includes(image.id)"
            @click.stop
            @change="toggleSelect(image.id)"
          />
        </div>
        <div class="info">
          <p class="name" :title="image.name">{{ image.name }}</p>
          <p class="meta">
            {{ formatBytes(image.size) }}
            <span v-if="image.deleted_at">· {{ formatDate(image.deleted_at) }} 删除</span>
          </p>
          <div class="actions">
            <el-button text size="small" :icon="RefreshLeft" @click="restoreOne(image)">恢复</el-button>
            <el-button text size="small" type="danger" @click="removeOne(image)">删除</el-button>
          </div>
        </div>
      </div>
    </div>

    <el-skeleton v-else-if="loading" animated :rows="4" />

    <EmptyState v-else description="回收站是空的" />

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
  </div>
</template>

<style scoped lang="scss">
.select-bar {
  display: flex;
  align-items: center;
  gap: var(--ph-space-3);
  margin-bottom: var(--ph-space-3);
}

.muted {
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.trash-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--ph-space-4);
}

.trash-card {
  overflow: hidden;
  background: var(--ph-fill-card);
  border: 1px solid var(--ph-border-light);
  border-radius: var(--ph-radius-lg);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    box-shadow: var(--ph-shadow-md);
  }

  &.selected {
    border-color: var(--ph-primary);
  }
}

.thumb {
  position: relative;
  aspect-ratio: 4 / 3;
  background: var(--ph-fill);
  cursor: pointer;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .check {
    position: absolute;
    top: 8px;
    left: 8px;
    padding: 2px 6px;
    background: var(--ph-fill-card);
    border-radius: var(--ph-radius-sm);
  }
}

.info {
  padding: var(--ph-space-3);
}

.name {
  font-size: 13px;
  font-weight: 500;
  color: var(--ph-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta {
  margin-top: 2px;
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.actions {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--ph-space-1);
}

.pager {
  display: flex;
  justify-content: center;
  margin-top: var(--ph-space-6);
}

@media (max-width: 767.98px) {
  .trash-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--ph-space-2);
  }
}
</style>
