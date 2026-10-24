<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  Connection,
  Files,
  Folder,
  Link,
  Picture,
  Postcard,
  Tickets,
  User as UserIcon,
} from '@element-plus/icons-vue'

import { adminApi } from '@/api/admin'
import PageHeader from '@/components/common/PageHeader.vue'
import type { Dashboard } from '@/types/api'
import { formatBytes, formatNumber } from '@/utils/format'

const router = useRouter()
const data = ref<Dashboard | null>(null)
const loading = ref(true)

const mainCards = computed(() => [
  {
    key: 'users',
    label: '用户总数',
    value: formatNumber(data.value?.user_count),
    icon: UserIcon,
    tone: 'blue',
    to: '/admin/users',
  },
  {
    key: 'images',
    label: '图片总数',
    value: formatNumber(data.value?.image_count),
    icon: Picture,
    tone: 'green',
    to: '/gallery',
  },
  {
    key: 'storage',
    label: '存储占用',
    value: formatBytes(data.value?.storage_bytes),
    icon: Files,
    tone: 'purple',
    to: '/admin/storages',
  },
  {
    key: 'today',
    label: '今日上传',
    value: formatNumber(data.value?.today_uploads),
    icon: Postcard,
    tone: 'orange',
    to: '/stats',
  },
  {
    key: 'hits',
    label: '引用访问',
    value: formatNumber(data.value?.referer_hits),
    icon: Link,
    tone: 'cyan',
    to: '/stats',
  },
])

const secondary = computed(() => [
  { label: '相册', value: formatNumber(data.value?.album_count), to: '/albums' },
  { label: '分享', value: formatNumber(data.value?.share_count), to: '/shares' },
  { label: '回收站', value: formatNumber(data.value?.trash_count), to: '/trash' },
  { label: '用户组', value: formatNumber(data.value?.group_count), to: '/admin/groups' },
  { label: '邀请码', value: formatNumber(data.value?.invite_count), to: '/admin/invites' },
])

const queue = computed(() => [
  { label: '待处理', value: data.value?.queue_ready ?? 0, type: 'info' as const },
  { label: '延迟任务', value: data.value?.queue_delayed ?? 0, type: 'warning' as const },
  { label: '死信任务', value: data.value?.queue_dead ?? 0, type: 'danger' as const },
])

const shortcuts = [
  { label: '用户管理', description: '查看、封禁与调整用户配额', icon: UserIcon, to: '/admin/users' },
  { label: '用户组', description: '配置不同用户组的额度与权限', icon: Tickets, to: '/admin/groups' },
  { label: '存储策略', description: '管理本地与对象存储后端', icon: Folder, to: '/admin/storages' },
  { label: '系统设置', description: '站点信息、邮箱与上传限制', icon: Connection, to: '/admin/settings' },
]

async function load(): Promise<void> {
  loading.value = true
  try {
    data.value = await adminApi.dashboard()
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="管理仪表盘" description="掌握站点用户、内容与队列的整体运行情况">
      <template #actions>
        <el-button @click="load">刷新数据</el-button>
      </template>
    </PageHeader>

    <div class="main-grid" v-loading="loading">
      <div v-for="card in mainCards" :key="card.key" class="stat-card" :class="card.tone" @click="router.push(card.to)">
        <div class="icon-wrap">
          <el-icon :size="22"><component :is="card.icon" /></el-icon>
        </div>
        <div>
          <p class="value">{{ card.value }}</p>
          <p class="label">{{ card.label }}</p>
        </div>
      </div>
    </div>

    <div class="secondary-grid" v-loading="loading">
      <router-link v-for="item in secondary" :key="item.label" :to="item.to" class="mini-card">
        <span class="mini-value">{{ item.value }}</span>
        <span class="mini-label">{{ item.label }}</span>
      </router-link>
    </div>

    <div class="grid">
      <section class="panel">
        <div class="panel-header">
          <h3>任务队列</h3>
          <span class="muted">异步任务执行情况</span>
        </div>
        <div class="queue">
          <div v-for="item in queue" :key="item.label" class="queue-item">
            <el-tag :type="item.type" size="small">{{ item.label }}</el-tag>
            <span class="queue-value">{{ formatNumber(item.value) }}</span>
          </div>
        </div>
      </section>

      <section class="panel">
        <div class="panel-header">
          <h3>快捷入口</h3>
        </div>
        <div class="shortcuts">
          <router-link v-for="item in shortcuts" :key="item.to" :to="item.to" class="shortcut">
            <el-icon :size="18"><component :is="item.icon" /></el-icon>
            <div>
              <p class="shortcut-label">{{ item.label }}</p>
              <p class="shortcut-desc">{{ item.description }}</p>
            </div>
          </router-link>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped lang="scss">
.main-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: var(--ph-space-3);
  margin-bottom: var(--ph-space-3);
}

.stat-card {
  display: flex;
  align-items: center;
  gap: var(--ph-space-3);
  padding: var(--ph-space-4);
  background: var(--ph-fill-card);
  border: 1px solid var(--ph-border-light);
  border-radius: var(--ph-radius-lg);
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--ph-shadow-md);
  }

  .icon-wrap {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border-radius: var(--ph-radius-md);
  }

  &.blue .icon-wrap {
    color: var(--ph-primary);
    background: var(--ph-primary-light-9);
  }

  &.green .icon-wrap {
    color: var(--ph-success);
    background: rgb(0 180 42 / 10%);
  }

  &.purple .icon-wrap {
    color: #722ed1;
    background: rgb(114 46 209 / 10%);
  }

  &.orange .icon-wrap {
    color: var(--ph-warning);
    background: rgb(255 125 0 / 10%);
  }

  &.cyan .icon-wrap {
    color: #0fc6c2;
    background: rgb(15 198 194 / 10%);
  }
}

.value {
  font-size: 22px;
  font-weight: 600;
  color: var(--ph-text-primary);
}

.label {
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.secondary-grid {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ph-space-2);
  margin-bottom: var(--ph-space-4);
}

.mini-card {
  display: flex;
  align-items: baseline;
  gap: var(--ph-space-2);
  padding: var(--ph-space-2) var(--ph-space-4);
  font-size: 12px;
  color: var(--ph-text-secondary);
  background: var(--ph-fill-card);
  border: 1px solid var(--ph-border-light);
  border-radius: var(--ph-radius-md);

  &:hover {
    color: var(--ph-primary);
  }
}

.mini-value {
  font-size: 16px;
  font-weight: 600;
  color: var(--ph-text-primary);
}

.grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr);
  gap: var(--ph-space-4);
}

.panel {
  padding: var(--ph-space-5);
  background: var(--ph-fill-card);
  border: 1px solid var(--ph-border-light);
  border-radius: var(--ph-radius-lg);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--ph-space-4);

  h3 {
    font-size: 15px;
  }
}

.muted {
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.queue {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-3);
}

.queue-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--ph-space-3);
  background: var(--ph-fill);
  border-radius: var(--ph-radius-md);
}

.queue-value {
  font-size: 18px;
  font-weight: 600;
}

.shortcuts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--ph-space-2);
}

.shortcut {
  display: flex;
  align-items: flex-start;
  gap: var(--ph-space-2);
  padding: var(--ph-space-3);
  color: var(--ph-text-primary);
  background: var(--ph-fill);
  border-radius: var(--ph-radius-md);
  transition: background 0.2s ease;

  &:hover {
    background: var(--ph-primary-light-9);
    color: var(--ph-primary);
  }
}

.shortcut-label {
  font-size: 13px;
  font-weight: 500;
}

.shortcut-desc {
  margin-top: 2px;
  font-size: 11px;
  color: var(--ph-text-secondary);
}

@media (max-width: 1199px) {
  .main-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 767.98px) {
  .main-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--ph-space-2);
  }

  .stat-card {
    padding: var(--ph-space-3);
  }

  .shortcuts {
    grid-template-columns: minmax(0, 1fr);
  }

  .panel {
    padding: var(--ph-space-4);
  }
}
</style>
