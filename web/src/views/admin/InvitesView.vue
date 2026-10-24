<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'

import { adminApi } from '@/api/admin'
import CopyButton from '@/components/common/CopyButton.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import type { Group, Invite } from '@/types/api'
import { formatDate, fromNow } from '@/utils/format'

const invites = ref<Invite[]>([])
const groups = ref<Group[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(false)

const dialogOpen = ref(false)
const submitting = ref(false)
const created = ref<Invite[]>([])
const form = reactive({
  max_uses: 1,
  group_id: undefined as number | undefined,
  expires_in_hours: 0,
  count: 1,
})

function groupName(id: number): string {
  return groups.value.find((group) => group.id === id)?.name || `#${id}`
}

function statusTag(invite: Invite): { text: string; type: 'success' | 'info' | 'warning' | 'danger' } {
  if (invite.status !== 1) {
    return { text: '已失效', type: 'info' }
  }
  if (invite.expires_at && new Date(invite.expires_at).getTime() < Date.now()) {
    return { text: '已过期', type: 'warning' }
  }
  if (invite.max_uses > 0 && invite.used_count >= invite.max_uses) {
    return { text: '已用完', type: 'danger' }
  }
  return { text: '可用', type: 'success' }
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await adminApi.invites(page.value, size.value)
    invites.value = data.list ?? []
    total.value = data.total ?? 0
  } finally {
    loading.value = false
  }
}

function openDialog(): void {
  created.value = []
  form.max_uses = 1
  form.group_id = groups.value.find((group) => group.is_default)?.id || groups.value[0]?.id
  form.expires_in_hours = 0
  form.count = 1
  dialogOpen.value = true
}

async function submit(): Promise<void> {
  if (form.count < 1 || form.count > 50) {
    ElMessage.warning('单次最多生成 50 个邀请码')
    return
  }
  submitting.value = true
  try {
    created.value = await adminApi.createInvites({
      max_uses: form.max_uses,
      group_id: form.group_id,
      expires_in_hours: form.expires_in_hours || undefined,
      count: form.count,
    })
    ElMessage.success(`已生成 ${created.value.length} 个邀请码`)
    page.value = 1
    await load()
  } finally {
    submitting.value = false
  }
}

async function remove(invite: Invite): Promise<void> {
  try {
    await ElMessageBox.confirm(`删除后邀请码 ${invite.code} 将立即失效，确定删除吗？`, '删除邀请码', {
      type: 'warning',
    })
  } catch {
    return
  }
  await adminApi.deleteInvite(invite.id)
  ElMessage.success('邀请码已删除')
  await load()
}

function copyAll(): void {
  const text = created.value.map((invite) => invite.code).join('\n')
  void navigator.clipboard?.writeText(text)
  ElMessage.success('已复制全部邀请码')
}

function handlePageChange(value: number): void {
  page.value = value
  void load()
}

onMounted(async () => {
  groups.value = await adminApi.groups().catch(() => [] as Group[])
  await load()
})
</script>

<template>
  <div>
    <PageHeader title="邀请码" description="控制注册入口，可限制使用次数、有效期与注册后加入的用户组">
      <template #actions>
        <el-button type="primary" :icon="Plus" @click="openDialog">生成邀请码</el-button>
      </template>
    </PageHeader>

    <el-table :data="invites" v-loading="loading" row-key="id">
      <el-table-column label="邀请码" min-width="200">
        <template #default="{ row }">
          <div class="code-cell">
            <code class="mono">{{ row.code }}</code>
            <CopyButton :text="row.code" />
          </div>
        </template>
      </el-table-column>
      <el-table-column label="用户组" width="120">
        <template #default="{ row }">{{ groupName(row.group_id) }}</template>
      </el-table-column>
      <el-table-column label="使用情况" width="120">
        <template #default="{ row }">
          {{ row.used_count }} / {{ row.max_uses || '不限' }}
        </template>
      </el-table-column>
      <el-table-column label="有效期" width="160">
        <template #default="{ row }">{{ row.expires_at ? formatDate(row.expires_at) : '永久' }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTag(row).type" size="small">{{ statusTag(row).text }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="140">
        <template #default="{ row }">{{ fromNow(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="90" fixed="right">
        <template #default="{ row }">
          <el-button text size="small" type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="还没有邀请码" :image-size="72" />
      </template>
    </el-table>

    <div class="pager">
      <el-pagination
        layout="total, prev, pager, next, sizes"
        :current-page="page"
        :page-size="size"
        :page-sizes="[20, 50, 100]"
        :total="total"
        background
        @current-change="handlePageChange"
        @size-change="
          (value: number) => {
            size = value
            page = 1
            load()
          }
        "
      />
    </div>

    <el-dialog v-model="dialogOpen" title="生成邀请码" width="480px">
      <template v-if="!created.length">
        <el-form label-position="top">
          <el-form-item label="注册后加入用户组">
            <el-select v-model="form.group_id" placeholder="选择用户组" class="w-full">
              <el-option v-for="group in groups" :key="group.id" :label="group.name" :value="group.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="最大使用次数（0 为不限）">
            <el-input-number v-model="form.max_uses" :min="0" :max="10000" class="w-full" />
          </el-form-item>
          <el-form-item label="有效期">
            <el-select v-model="form.expires_in_hours" class="w-full">
              <el-option label="永久有效" :value="0" />
              <el-option label="1 天" :value="24" />
              <el-option label="7 天" :value="168" />
              <el-option label="30 天" :value="720" />
            </el-select>
          </el-form-item>
          <el-form-item label="生成数量">
            <el-input-number v-model="form.count" :min="1" :max="50" class="w-full" />
          </el-form-item>
        </el-form>
      </template>

      <div v-else class="created">
        <el-alert type="success" :closable="false" :title="`已生成 ${created.length} 个邀请码`" />
        <div class="code-list">
          <div v-for="invite in created" :key="invite.id" class="code-item">
            <code class="mono">{{ invite.code }}</code>
            <CopyButton :text="invite.code" />
          </div>
        </div>
      </div>

      <template #footer>
        <template v-if="!created.length">
          <el-button @click="dialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submit">生成</el-button>
        </template>
        <template v-else>
          <el-button @click="copyAll">复制全部</el-button>
          <el-button type="primary" @click="dialogOpen = false">完成</el-button>
        </template>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.code-cell {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
}

.mono {
  font-family: var(--ph-font-mono);
  font-size: 12px;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--ph-space-4);
}

.code-list {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-2);
  max-height: 320px;
  margin-top: var(--ph-space-3);
  overflow-y: auto;
}

.code-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ph-space-2);
  padding: var(--ph-space-2) var(--ph-space-3);
  background: var(--ph-fill);
  border-radius: var(--ph-radius-md);
}
</style>
