<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Search } from '@element-plus/icons-vue'

import { adminApi } from '@/api/admin'
import PageHeader from '@/components/common/PageHeader.vue'
import type { EmailLog, OperationLog } from '@/types/api'
import { formatDate } from '@/utils/format'

const activeTab = ref<'operation' | 'email'>('operation')

const logs = ref<OperationLog[]>([])
const logTotal = ref(0)
const logLoading = ref(false)
const logQuery = reactive({ user_id: undefined as number | undefined, action: '', page: 1, size: 20 })

const emails = ref<EmailLog[]>([])
const emailTotal = ref(0)
const emailLoading = ref(false)
const emailQuery = reactive({ user_id: undefined as number | undefined, status: undefined as number | undefined, page: 1, size: 20 })

async function loadLogs(): Promise<void> {
  logLoading.value = true
  try {
    const data = await adminApi.logs({
      ...logQuery,
      action: logQuery.action.trim() || undefined,
    })
    logs.value = data.list ?? []
    logTotal.value = data.total ?? 0
  } finally {
    logLoading.value = false
  }
}

async function loadEmails(): Promise<void> {
  emailLoading.value = true
  try {
    const data = await adminApi.emailLogs(emailQuery)
    emails.value = data.list ?? []
    emailTotal.value = data.total ?? 0
  } finally {
    emailLoading.value = false
  }
}

function searchLogs(): void {
  logQuery.page = 1
  void loadLogs()
}

function searchEmails(): void {
  emailQuery.page = 1
  void loadEmails()
}

function logPageChange(value: number): void {
  logQuery.page = value
  void loadLogs()
}

function emailPageChange(value: number): void {
  emailQuery.page = value
  void loadEmails()
}

function emailStatusTag(status: number): { text: string; type: 'success' | 'danger' | 'warning' } {
  if (status === 1) {
    return { text: '发送成功', type: 'success' }
  }
  if (status === 2) {
    return { text: '发送失败', type: 'danger' }
  }
  return { text: '待发送', type: 'warning' }
}

function detailText(detail: unknown): string {
  if (!detail) {
    return '—'
  }
  try {
    return JSON.stringify(detail)
  } catch {
    return String(detail)
  }
}

onMounted(() => {
  void loadLogs()
  void loadEmails()
})
</script>

<template>
  <div>
    <PageHeader title="日志中心" description="追踪后台操作与邮件投递记录" />

    <el-tabs v-model="activeTab">
      <el-tab-pane label="操作日志" name="operation">
        <div class="toolbar">
          <el-input
            v-model="logQuery.action"
            placeholder="筛选操作类型，例如 image.delete"
            :prefix-icon="Search"
            clearable
            class="search"
            @keyup.enter="searchLogs"
            @clear="searchLogs"
          />
          <el-input-number
            v-model="logQuery.user_id"
            :min="0"
            :controls="false"
            placeholder="用户 ID"
            class="id-input"
          />
          <el-button @click="searchLogs">查询</el-button>
        </div>

        <el-table :data="logs" v-loading="logLoading" row-key="id">
          <el-table-column prop="action" label="操作" width="170">
            <template #default="{ row }">
              <code class="mono">{{ row.action }}</code>
            </template>
          </el-table-column>
          <el-table-column label="对象" width="160">
            <template #default="{ row }">
              <span v-if="row.target_type">{{ row.target_type }} #{{ row.target_id }}</span>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
          <el-table-column prop="user_id" label="用户 ID" width="100" />
          <el-table-column prop="ip" label="IP" width="140">
            <template #default="{ row }">
              <code class="mono">{{ row.ip || '—' }}</code>
            </template>
          </el-table-column>
          <el-table-column label="详情" min-width="220">
            <template #default="{ row }">
              <el-tooltip :content="detailText(row.detail)" placement="top">
                <span class="detail">{{ detailText(row.detail) }}</span>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column label="时间" width="160">
            <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
          </el-table-column>
          <template #empty>
            <el-empty description="暂无操作日志" :image-size="72" />
          </template>
        </el-table>

        <div class="pager">
          <el-pagination
            layout="total, prev, pager, next, sizes"
            :current-page="logQuery.page"
            :page-size="logQuery.size"
            :page-sizes="[20, 50, 100]"
            :total="logTotal"
            background
            @current-change="logPageChange"
            @size-change="
              (value: number) => {
                logQuery.size = value
                logQuery.page = 1
                loadLogs()
              }
            "
          />
        </div>
      </el-tab-pane>

      <el-tab-pane label="邮件日志" name="email">
        <div class="toolbar">
          <el-input-number
            v-model="emailQuery.user_id"
            :min="0"
            :controls="false"
            placeholder="用户 ID"
            class="id-input"
          />
          <el-select v-model="emailQuery.status" placeholder="全部状态" clearable class="filter" @change="searchEmails">
            <el-option label="发送成功" :value="1" />
            <el-option label="发送失败" :value="2" />
          </el-select>
          <el-button @click="searchEmails">查询</el-button>
        </div>

        <el-table :data="emails" v-loading="emailLoading" row-key="id">
          <el-table-column prop="to_email" label="收件人" min-width="200" />
          <el-table-column prop="subject" label="主题" min-width="220" />
          <el-table-column label="模板" width="140">
            <template #default="{ row }">
              <code class="mono">{{ row.template || '—' }}</code>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="emailStatusTag(row.status).type" size="small">
                {{ emailStatusTag(row.status).text }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="错误信息" min-width="200">
            <template #default="{ row }">
              <span v-if="row.error" class="error">{{ row.error }}</span>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
          <el-table-column label="发送时间" width="160">
            <template #default="{ row }">{{ row.sent_at ? formatDate(row.sent_at) : '—' }}</template>
          </el-table-column>
          <el-table-column label="创建时间" width="160">
            <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
          </el-table-column>
          <template #empty>
            <el-empty description="暂无邮件日志" :image-size="72" />
          </template>
        </el-table>

        <div class="pager">
          <el-pagination
            layout="total, prev, pager, next, sizes"
            :current-page="emailQuery.page"
            :page-size="emailQuery.size"
            :page-sizes="[20, 50, 100]"
            :total="emailTotal"
            background
            @current-change="emailPageChange"
            @size-change="
              (value: number) => {
                emailQuery.size = value
                emailQuery.page = 1
                loadEmails()
              }
            "
          />
        </div>
      </el-tab-pane>
    </el-tabs>
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
  width: 260px;
}

.filter {
  width: 140px;
}

.id-input {
  width: 120px;
}

.mono {
  font-family: var(--ph-font-mono);
  font-size: 12px;
}

.muted {
  font-size: 12px;
  color: var(--ph-text-tertiary);
}

.detail {
  display: inline-block;
  max-width: 100%;
  font-size: 12px;
  color: var(--ph-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.error {
  font-size: 12px;
  color: var(--ph-danger);
  word-break: break-all;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--ph-space-4);
}

@media (max-width: 767.98px) {
  .search,
  .filter,
  .id-input {
    width: 100%;
  }
}
</style>
