<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'

import { adminApi, type GroupPayload } from '@/api/admin'
import PageHeader from '@/components/common/PageHeader.vue'
import type { Group } from '@/types/api'
import { formatBytes, formatNumber } from '@/utils/format'

const groups = ref<Group[]>([])
const loading = ref(false)
const dialogOpen = ref(false)
const editing = ref<Group | null>(null)
const submitting = ref(false)

const form = reactive({
  name: '',
  description: '',
  quota_mb: 1024,
  max_file_mb: 10,
  daily_limit: 100,
  max_albums: 20,
  max_tokens: 5,
  max_shares: 20,
  allow_register: true,
  allow_custom_storage: false,
  is_default: false,
  status: 1,
})

async function load(): Promise<void> {
  loading.value = true
  try {
    groups.value = await adminApi.groups()
  } finally {
    loading.value = false
  }
}

function openCreate(): void {
  editing.value = null
  Object.assign(form, {
    name: '',
    description: '',
    quota_mb: 1024,
    max_file_mb: 10,
    daily_limit: 100,
    max_albums: 20,
    max_tokens: 5,
    max_shares: 20,
    allow_register: true,
    allow_custom_storage: false,
    is_default: false,
    status: 1,
  })
  dialogOpen.value = true
}

function openEdit(group: Group): void {
  editing.value = group
  Object.assign(form, {
    name: group.name,
    description: group.description,
    quota_mb: Math.round((group.quota_bytes / (1024 * 1024)) * 100) / 100,
    max_file_mb: Math.round((group.max_file_size / (1024 * 1024)) * 100) / 100,
    daily_limit: group.daily_limit,
    max_albums: group.max_albums,
    max_tokens: group.max_tokens,
    max_shares: group.max_shares,
    allow_register: group.allow_register,
    allow_custom_storage: group.allow_custom_storage,
    is_default: group.is_default,
    status: group.status,
  })
  dialogOpen.value = true
}

async function submit(): Promise<void> {
  if (!form.name.trim()) {
    ElMessage.warning('请填写用户组名称')
    return
  }
  const payload: GroupPayload = {
    name: form.name.trim(),
    description: form.description,
    quota_bytes: Math.round(form.quota_mb * 1024 * 1024),
    max_file_size: Math.round(form.max_file_mb * 1024 * 1024),
    daily_limit: form.daily_limit,
    max_albums: form.max_albums,
    max_tokens: form.max_tokens,
    max_shares: form.max_shares,
    allow_register: form.allow_register,
    allow_custom_storage: form.allow_custom_storage,
    is_default: form.is_default,
    status: form.status,
  }
  submitting.value = true
  try {
    if (editing.value) {
      await adminApi.updateGroup(editing.value.id, payload)
      ElMessage.success('用户组已更新')
    } else {
      await adminApi.createGroup(payload)
      ElMessage.success('用户组已创建')
    }
    dialogOpen.value = false
    await load()
  } finally {
    submitting.value = false
  }
}

async function remove(group: Group): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `确定删除用户组「${group.name}」吗？该组下的 ${group.user_count} 位用户需要先迁移到其他用户组。`,
      '删除用户组',
      { type: 'warning' },
    )
  } catch {
    return
  }
  await adminApi.deleteGroup(group.id)
  ElMessage.success('用户组已删除')
  await load()
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="用户组" description="为不同用户群体配置存储配额、上传限制与功能权限">
      <template #actions>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建用户组</el-button>
      </template>
    </PageHeader>

    <el-table :data="groups" v-loading="loading" row-key="id">
      <el-table-column label="名称" min-width="160">
        <template #default="{ row }">
          <div class="name-cell">
            <span class="name">{{ row.name }}</span>
            <el-tag v-if="row.is_default" type="success" size="small">默认</el-tag>
            <el-tag v-if="row.status !== 1" type="info" size="small">已停用</el-tag>
          </div>
          <p class="description">{{ row.description || '—' }}</p>
        </template>
      </el-table-column>
      <el-table-column label="用户数" width="90">
        <template #default="{ row }">{{ formatNumber(row.user_count) }}</template>
      </el-table-column>
      <el-table-column label="存储配额" width="110">
        <template #default="{ row }">{{ formatBytes(row.quota_bytes) }}</template>
      </el-table-column>
      <el-table-column label="单张上限" width="100">
        <template #default="{ row }">{{ formatBytes(row.max_file_size) }}</template>
      </el-table-column>
      <el-table-column label="每日上传" width="100">
        <template #default="{ row }">{{ row.daily_limit || '不限' }}</template>
      </el-table-column>
      <el-table-column label="相册 / 令牌 / 分享" width="150">
        <template #default="{ row }">{{ row.max_albums }} / {{ row.max_tokens }} / {{ row.max_shares }}</template>
      </el-table-column>
      <el-table-column label="注册" width="80">
        <template #default="{ row }">
          <el-tag :type="row.allow_register ? 'success' : 'info'" size="small">
            {{ row.allow_register ? '允许' : '禁止' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="自定义存储" width="110">
        <template #default="{ row }">
          <el-tag :type="row.allow_custom_storage ? 'success' : 'info'" size="small">
            {{ row.allow_custom_storage ? '允许' : '禁止' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button text size="small" @click="openEdit(row)">编辑</el-button>
          <el-button text size="small" type="danger" :disabled="row.is_default" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="还没有用户组" :image-size="72" />
      </template>
    </el-table>

    <el-dialog v-model="dialogOpen" :title="editing ? '编辑用户组' : '新建用户组'" width="560px">
      <el-form label-position="top">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" maxlength="64" show-word-limit placeholder="例如：VIP 用户" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" maxlength="255" show-word-limit />
        </el-form-item>

        <div class="form-grid">
          <el-form-item label="存储配额（MB）">
            <el-input-number v-model="form.quota_mb" :min="0" :step="100" class="w-full" />
          </el-form-item>
          <el-form-item label="单张上限（MB）">
            <el-input-number v-model="form.max_file_mb" :min="0" class="w-full" />
          </el-form-item>
          <el-form-item label="每日上传（0 不限）">
            <el-input-number v-model="form.daily_limit" :min="0" class="w-full" />
          </el-form-item>
          <el-form-item label="相册数量上限">
            <el-input-number v-model="form.max_albums" :min="0" class="w-full" />
          </el-form-item>
          <el-form-item label="令牌数量上限">
            <el-input-number v-model="form.max_tokens" :min="0" class="w-full" />
          </el-form-item>
          <el-form-item label="分享数量上限">
            <el-input-number v-model="form.max_shares" :min="0" class="w-full" />
          </el-form-item>
        </div>

        <el-form-item label="权限">
          <div class="switches">
            <el-switch v-model="form.allow_register" active-text="允许注册" />
            <el-switch v-model="form.allow_custom_storage" active-text="允许自定义存储" />
            <el-switch v-model="form.is_default" active-text="设为默认组" />
            <el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="启用" />
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogOpen = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.name-cell {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
}

.name {
  font-size: 13px;
  font-weight: 500;
}

.description {
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 var(--ph-space-4);
}

.switches {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ph-space-3);
}

@media (max-width: 767.98px) {
  .form-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
