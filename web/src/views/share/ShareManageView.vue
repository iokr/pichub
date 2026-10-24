<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Link, Plus, Search } from '@element-plus/icons-vue'

import { albumApi } from '@/api/album'
import { imageApi } from '@/api/image'
import { shareApi } from '@/api/share'
import PageHeader from '@/components/common/PageHeader.vue'
import CopyButton from '@/components/common/CopyButton.vue'
import type { Album, Image, Share } from '@/types/api'
import { formatDate, fromNow } from '@/utils/format'

const shares = ref<Share[]>([])
const albums = ref<Album[]>([])
const imageOptions = ref<Image[]>([])
const loading = ref(false)
const keyword = ref('')

const dialogOpen = ref(false)
const submitting = ref(false)
const searching = ref(false)
const form = reactive({
  target_type: 'image' as 'image' | 'album',
  target_id: undefined as number | undefined,
  password: '',
  expires_in_hours: 0,
  max_views: 0,
})

const filtered = computed(() => {
  const value = keyword.value.trim().toLowerCase()
  if (!value) {
    return shares.value
  }
  return shares.value.filter(
    (share) => share.slug.toLowerCase().includes(value) || share.url.toLowerCase().includes(value),
  )
})

async function load(): Promise<void> {
  loading.value = true
  try {
    shares.value = await shareApi.list()
  } finally {
    loading.value = false
  }
}

function statusText(share: Share): { text: string; type: 'success' | 'info' | 'warning' | 'danger' } {
  if (share.status !== 1) {
    return { text: '已禁用', type: 'info' }
  }
  if (share.expires_at && new Date(share.expires_at).getTime() < Date.now()) {
    return { text: '已过期', type: 'warning' }
  }
  if (share.max_views > 0 && share.views >= share.max_views) {
    return { text: '已达上限', type: 'danger' }
  }
  return { text: '生效中', type: 'success' }
}

function targetLabel(share: Share): string {
  return `${share.target_type === 'album' ? '相册' : '图片'} #${share.target_id}`
}

async function openDialog(): Promise<void> {
  form.target_type = 'image'
  form.target_id = undefined
  form.password = ''
  form.expires_in_hours = 0
  form.max_views = 0
  dialogOpen.value = true
  if (!albums.value.length) {
    albums.value = await albumApi.list().catch(() => [] as Album[])
  }
  await searchImages('')
}

async function searchImages(query: string): Promise<void> {
  searching.value = true
  try {
    const data = await imageApi.list({ keyword: query || undefined, page: 1, size: 20 })
    imageOptions.value = data.list ?? []
  } finally {
    searching.value = false
  }
}

async function submit(): Promise<void> {
  if (!form.target_id) {
    ElMessage.warning(form.target_type === 'image' ? '请选择图片' : '请选择相册')
    return
  }
  submitting.value = true
  try {
    const share = await shareApi.create({
      target_type: form.target_type,
      target_id: form.target_id,
      password: form.password.trim() || undefined,
      expires_in_hours: form.expires_in_hours || undefined,
      max_views: form.max_views || undefined,
    })
    shares.value.unshift(share)
    dialogOpen.value = false
    ElMessage.success('分享链接已创建')
  } finally {
    submitting.value = false
  }
}

async function remove(share: Share): Promise<void> {
  try {
    await ElMessageBox.confirm('删除后分享链接将立即失效，确定删除吗？', '删除分享', { type: 'warning' })
  } catch {
    return
  }
  await shareApi.remove(share.id)
  shares.value = shares.value.filter((item) => item.id !== share.id)
  ElMessage.success('分享已删除')
}

function openShare(share: Share): void {
  window.open(share.url, '_blank', 'noopener')
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="分享管理" description="把图片或相册以链接形式分享给他人，可设置密码、有效期与访问上限">
      <template #actions>
        <el-button type="primary" :icon="Plus" @click="openDialog">创建分享</el-button>
      </template>
    </PageHeader>

    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索链接或标识" :prefix-icon="Search" clearable class="search" />
    </div>

    <el-table :data="filtered" v-loading="loading" row-key="id">
      <el-table-column label="分享链接" min-width="260">
        <template #default="{ row }">
          <div class="link-cell">
            <el-icon class="link-icon"><Link /></el-icon>
            <code class="mono" @click="openShare(row)">{{ row.url }}</code>
            <CopyButton :text="row.url" />
          </div>
        </template>
      </el-table-column>
      <el-table-column label="内容" width="120">
        <template #default="{ row }">{{ targetLabel(row) }}</template>
      </el-table-column>
      <el-table-column label="密码" width="80">
        <template #default="{ row }">
          <el-tag v-if="row.has_password" size="small" type="warning">有密码</el-tag>
          <span v-else class="muted">无</span>
        </template>
      </el-table-column>
      <el-table-column label="访问量" width="100">
        <template #default="{ row }">
          {{ row.views }}<span v-if="row.max_views" class="muted"> / {{ row.max_views }}</span>
        </template>
      </el-table-column>
      <el-table-column label="有效期" width="160">
        <template #default="{ row }">{{ row.expires_at ? formatDate(row.expires_at) : '永久' }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusText(row).type" size="small">{{ statusText(row).text }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="120">
        <template #default="{ row }">{{ fromNow(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button text size="small" @click="openShare(row)">打开</el-button>
          <el-button text size="small" type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="还没有创建分享" :image-size="72" />
      </template>
    </el-table>

    <el-dialog v-model="dialogOpen" title="创建分享链接" width="520px">
      <el-form label-position="top">
        <el-form-item label="分享内容">
          <el-radio-group v-model="form.target_type" @change="form.target_id = undefined">
            <el-radio-button value="image">图片</el-radio-button>
            <el-radio-button value="album">相册</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="form.target_type === 'image'" label="选择图片" required>
          <el-select
            v-model="form.target_id"
            filterable
            remote
            clearable
            :remote-method="searchImages"
            :loading="searching"
            placeholder="输入关键字搜索图片"
            class="w-full"
          >
            <el-option v-for="image in imageOptions" :key="image.id" :label="image.name" :value="image.id" />
          </el-select>
        </el-form-item>

        <el-form-item v-else label="选择相册" required>
          <el-select v-model="form.target_id" filterable clearable placeholder="选择相册" class="w-full">
            <el-option v-for="album in albums" :key="album.id" :label="album.name" :value="album.id" />
          </el-select>
        </el-form-item>

        <el-form-item label="访问密码（可选）">
          <el-input v-model="form.password" placeholder="留空表示无需密码" maxlength="32" show-password />
        </el-form-item>

        <el-form-item label="有效期">
          <el-select v-model="form.expires_in_hours" class="w-full">
            <el-option label="永久有效" :value="0" />
            <el-option label="1 小时" :value="1" />
            <el-option label="1 天" :value="24" />
            <el-option label="7 天" :value="168" />
            <el-option label="30 天" :value="720" />
          </el-select>
        </el-form-item>

        <el-form-item label="最大访问次数（可选）">
          <el-input-number v-model="form.max_views" :min="0" :max="1_000_000" class="w-full" />
          <p class="hint">0 表示不限制访问次数</p>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogOpen = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">创建分享</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.toolbar {
  margin-bottom: var(--ph-space-4);
}

.search {
  width: 240px;
}

.mono {
  font-family: var(--ph-font-mono);
  font-size: 12px;
}

.link-cell {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
  min-width: 0;

  code {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: pointer;

    &:hover {
      color: var(--ph-primary);
    }
  }
}

.link-icon {
  color: var(--ph-text-tertiary);
}

.muted {
  font-size: 12px;
  color: var(--ph-text-tertiary);
}

.hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--ph-text-tertiary);
}

@media (max-width: 767.98px) {
  .search {
    width: 100%;
  }
}
</style>
