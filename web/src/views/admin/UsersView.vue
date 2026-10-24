<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'

import { adminApi, type AdminUsersParams } from '@/api/admin'
import PageHeader from '@/components/common/PageHeader.vue'
import type { AdminUser, Group } from '@/types/api'
import { formatBytes, formatDate, percent } from '@/utils/format'

const users = ref<AdminUser[]>([])
const groups = ref<Group[]>([])
const total = ref(0)
const loading = ref(false)

const query = reactive<AdminUsersParams>({
  keyword: '',
  status: undefined,
  role: undefined,
  group_id: undefined,
  page: 1,
  size: 20,
})

const groupDialog = ref(false)
const targetUser = ref<AdminUser | null>(null)
const groupForm = reactive({
  group_id: undefined as number | undefined,
  quota_mb: 0,
  max_file_mb: 0,
  daily_limit: 0,
})
const savingGroup = ref(false)

const passwordDialog = ref(false)
const newPassword = ref('')
const savingPassword = ref(false)

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await adminApi.users({
      ...query,
      keyword: query.keyword?.trim() || undefined,
    })
    users.value = data.list ?? []
    total.value = data.total ?? 0
  } finally {
    loading.value = false
  }
}

function search(): void {
  query.page = 1
  void load()
}

function reset(): void {
  query.keyword = ''
  query.status = undefined
  query.role = undefined
  query.group_id = undefined
  query.page = 1
  void load()
}

function groupName(id: number): string {
  return groups.value.find((group) => group.id === id)?.name || '—'
}

function statusTag(status: number): { text: string; type: 'success' | 'danger' | 'warning' } {
  if (status === 1) {
    return { text: '正常', type: 'success' }
  }
  if (status === 2) {
    return { text: '禁用', type: 'danger' }
  }
  return { text: '待激活', type: 'warning' }
}

async function toggleStatus(user: AdminUser): Promise<void> {
  const next = user.status === 1 ? 2 : 1
  const action = next === 1 ? '启用' : '禁用'
  let reason = ''
  if (next === 2) {
    try {
      const result = await ElMessageBox.prompt(`禁用后该用户将无法登录，可填写原因：`, `禁用用户 ${user.username}`, {
        inputPlaceholder: '可选',
        confirmButtonText: '禁用',
        type: 'warning',
      })
      reason = result.value || ''
    } catch {
      return
    }
  } else {
    try {
      await ElMessageBox.confirm(`确定启用用户 ${user.username} 吗？`, '启用用户', { type: 'info' })
    } catch {
      return
    }
  }
  await adminApi.updateUserStatus(user.id, next, reason || undefined)
  user.status = next
  ElMessage.success(`已${action}用户`)
}

function openGroupDialog(user: AdminUser): void {
  targetUser.value = user
  groupForm.group_id = user.group_id
  groupForm.quota_mb = Math.round((user.quota_bytes / (1024 * 1024)) * 100) / 100
  groupForm.max_file_mb = Math.round((user.max_file_size / (1024 * 1024)) * 100) / 100
  groupForm.daily_limit = user.daily_limit
  groupDialog.value = true
}

async function saveGroup(): Promise<void> {
  if (!targetUser.value || !groupForm.group_id) {
    ElMessage.warning('请选择用户组')
    return
  }
  savingGroup.value = true
  try {
    await adminApi.updateUserGroup(targetUser.value.id, {
      group_id: groupForm.group_id,
      quota_bytes: Math.round(groupForm.quota_mb * 1024 * 1024),
      max_file_size: Math.round(groupForm.max_file_mb * 1024 * 1024),
      daily_limit: groupForm.daily_limit,
    })
    ElMessage.success('用户组与额度已更新')
    groupDialog.value = false
    await load()
  } finally {
    savingGroup.value = false
  }
}

function openPasswordDialog(user: AdminUser): void {
  targetUser.value = user
  newPassword.value = ''
  passwordDialog.value = true
}

async function savePassword(): Promise<void> {
  if (!targetUser.value) {
    return
  }
  if (newPassword.value.length < 8) {
    ElMessage.warning('密码至少 8 位')
    return
  }
  savingPassword.value = true
  try {
    await adminApi.resetUserPassword(targetUser.value.id, newPassword.value)
    ElMessage.success('密码已重置')
    passwordDialog.value = false
  } finally {
    savingPassword.value = false
  }
}

function handlePageChange(value: number): void {
  query.page = value
  void load()
}

onMounted(async () => {
  groups.value = await adminApi.groups().catch(() => [] as Group[])
  await load()
})
</script>

<template>
  <div>
    <PageHeader title="用户管理" :description="`共 ${total} 位用户`" />

    <div class="toolbar">
      <el-input
        v-model="query.keyword"
        placeholder="搜索用户名 / 邮箱"
        :prefix-icon="Search"
        clearable
        class="search"
        @keyup.enter="search"
        @clear="search"
      />
      <el-select v-model="query.status" placeholder="全部状态" clearable class="filter" @change="search">
        <el-option label="正常" :value="1" />
        <el-option label="禁用" :value="2" />
      </el-select>
      <el-select v-model="query.role" placeholder="全部角色" clearable class="filter" @change="search">
        <el-option label="管理员" :value="1" />
        <el-option label="普通用户" :value="0" />
      </el-select>
      <el-select v-model="query.group_id" placeholder="全部用户组" clearable class="filter" @change="search">
        <el-option v-for="group in groups" :key="group.id" :label="group.name" :value="group.id" />
      </el-select>
      <el-button @click="search">查询</el-button>
      <el-button text @click="reset">重置</el-button>
    </div>

    <el-table :data="users" v-loading="loading" row-key="id">
      <el-table-column label="用户" min-width="200">
        <template #default="{ row }">
          <div class="user-cell">
            <el-avatar :size="32" :src="row.avatar_url">{{ row.username.slice(0, 1).toUpperCase() }}</el-avatar>
            <div class="user-info">
              <span class="name">{{ row.nickname || row.username }}</span>
              <span class="email">{{ row.email }}</span>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="用户组" width="120">
        <template #default="{ row }">{{ groupName(row.group_id) }}</template>
      </el-table-column>
      <el-table-column label="角色" width="90">
        <template #default="{ row }">
          <el-tag :type="row.role === 1 ? 'warning' : 'info'" size="small">
            {{ row.role === 1 ? '管理员' : '用户' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="配额" min-width="160">
        <template #default="{ row }">
          <el-progress
            :percentage="percent(row.used_bytes, row.quota_bytes)"
            :stroke-width="6"
            :show-text="false"
          />
          <span class="quota-text">{{ formatBytes(row.used_bytes) }} / {{ formatBytes(row.quota_bytes) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="statusTag(row.status).type" size="small">{{ statusTag(row.status).text }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="最近登录" width="150">
        <template #default="{ row }">{{ row.last_login_at ? formatDate(row.last_login_at) : '从未登录' }}</template>
      </el-table-column>
      <el-table-column label="注册时间" width="150">
        <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button text size="small" @click="openGroupDialog(row)">调整额度</el-button>
          <el-button text size="small" @click="openPasswordDialog(row)">重置密码</el-button>
          <el-button text size="small" :type="row.status === 1 ? 'danger' : 'success'" @click="toggleStatus(row)">
            {{ row.status === 1 ? '禁用' : '启用' }}
          </el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="没有匹配的用户" :image-size="72" />
      </template>
    </el-table>

    <div class="pager">
      <el-pagination
        layout="total, prev, pager, next, sizes"
        :current-page="query.page"
        :page-size="query.size"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        background
        @current-change="handlePageChange"
        @size-change="
          (value: number) => {
            query.size = value
            query.page = 1
            load()
          }
        "
      />
    </div>

    <el-dialog v-model="groupDialog" title="调整用户组与额度" width="460px">
      <p v-if="targetUser" class="dialog-target">用户：{{ targetUser.username }}</p>
      <el-form label-position="top">
        <el-form-item label="用户组">
          <el-select v-model="groupForm.group_id" placeholder="选择用户组" class="w-full">
            <el-option v-for="group in groups" :key="group.id" :label="group.name" :value="group.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="存储配额（MB）">
          <el-input-number v-model="groupForm.quota_mb" :min="0" :step="100" class="w-full" />
        </el-form-item>
        <el-form-item label="单张上限（MB）">
          <el-input-number v-model="groupForm.max_file_mb" :min="0" :step="1" class="w-full" />
        </el-form-item>
        <el-form-item label="每日上传上限（0 为不限）">
          <el-input-number v-model="groupForm.daily_limit" :min="0" :step="10" class="w-full" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupDialog = false">取消</el-button>
        <el-button type="primary" :loading="savingGroup" @click="saveGroup">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="passwordDialog" title="重置密码" width="420px">
      <p v-if="targetUser" class="dialog-target">用户：{{ targetUser.username }}</p>
      <el-input v-model="newPassword" type="password" show-password placeholder="输入新密码（至少 8 位）" />
      <template #footer>
        <el-button @click="passwordDialog = false">取消</el-button>
        <el-button type="primary" :loading="savingPassword" @click="savePassword">重置密码</el-button>
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
  width: 220px;
}

.filter {
  width: 140px;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
}

.user-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.name {
  font-size: 13px;
  font-weight: 500;
  color: var(--ph-text-primary);
}

.email {
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.quota-text {
  font-size: 11px;
  color: var(--ph-text-tertiary);
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--ph-space-4);
}

.dialog-target {
  margin-bottom: var(--ph-space-3);
  font-size: 13px;
  color: var(--ph-text-secondary);
}

@media (max-width: 767.98px) {
  .search,
  .filter {
    width: calc(50% - var(--ph-space-1));
  }
}
</style>
