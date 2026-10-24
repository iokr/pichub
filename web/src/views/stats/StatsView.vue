<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Collection, Document, Files, Link, Picture, View } from '@element-plus/icons-vue'

import { userApi } from '@/api/user'
import PageHeader from '@/components/common/PageHeader.vue'
import TrendChart from '@/components/charts/TrendChart.vue'
import type { UserStats } from '@/types/api'
import { formatBytes, formatNumber, percent } from '@/utils/format'

const router = useRouter()
const stats = ref<UserStats | null>(null)
const loading = ref(true)

const quotaPercent = computed(() => percent(stats.value?.used_bytes, stats.value?.quota_bytes))

const cards = computed(() => {
  const data = stats.value
  return [
    { key: 'images', label: '图片总数', value: formatNumber(data?.image_count), icon: Picture, tone: 'blue', to: '/gallery' },
    { key: 'albums', label: '相册数量', value: formatNumber(data?.album_count), icon: Collection, tone: 'purple', to: '/albums' },
    { key: 'shares', label: '分享链接', value: formatNumber(data?.share_count), icon: Link, tone: 'green', to: '/shares' },
    { key: 'trash', label: '回收站', value: formatNumber(data?.trash_count), icon: Files, tone: 'orange', to: '/trash' },
    { key: 'today', label: '今日上传', value: formatNumber(data?.today_uploads), icon: Document, tone: 'cyan', to: '/' },
    { key: 'hits', label: '引用访问', value: formatNumber(data?.referer_hits), icon: View, tone: 'red', to: '/stats' },
  ]
})

async function load(): Promise<void> {
  loading.value = true
  try {
    stats.value = await userApi.stats()
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="数据统计" description="查看图片、存储与引用来源的使用情况" />

    <div class="card-grid" v-loading="loading">
      <div v-for="card in cards" :key="card.key" class="stat-card" :class="card.tone" @click="router.push(card.to)">
        <div class="icon-wrap">
          <el-icon :size="20"><component :is="card.icon" /></el-icon>
        </div>
        <div class="content">
          <p class="value">{{ card.value }}</p>
          <p class="label">{{ card.label }}</p>
        </div>
      </div>
    </div>

    <div class="grid">
      <section class="panel">
        <div class="panel-header">
          <h3>引用趋势</h3>
          <span class="muted">最近 30 天</span>
        </div>
        <TrendChart v-if="stats?.trend?.length" :data="stats.trend" height="300px" />
        <el-empty v-else description="暂无引用数据" :image-size="72" />
      </section>

      <section class="panel">
        <div class="panel-header">
          <h3>存储用量</h3>
          <router-link to="/settings" class="more">配额详情</router-link>
        </div>
        <div class="storage">
          <div class="storage-value">
            <span class="big">{{ formatBytes(stats?.used_bytes) }}</span>
            <span class="muted">/ {{ formatBytes(stats?.quota_bytes) }}</span>
          </div>
          <el-progress :percentage="quotaPercent" :stroke-width="10" :show-text="false" />
          <div class="storage-meta">
            <div>
              <p class="num">{{ formatNumber(stats?.image_count) }}</p>
              <p class="label">图片</p>
            </div>
            <div>
              <p class="num">{{ formatNumber(stats?.today_uploads) }}</p>
              <p class="label">今日上传</p>
            </div>
            <div>
              <p class="num">{{ stats?.daily_limit || '∞' }}</p>
              <p class="label">每日上限</p>
            </div>
          </div>
        </div>
      </section>
    </div>

    <section class="panel hosts">
      <div class="panel-header">
        <h3>引用来源 Top 域名</h3>
        <span class="muted">按累计引用次数排序</span>
      </div>
      <el-table v-if="stats?.top_hosts?.length" :data="stats.top_hosts" size="large">
        <el-table-column type="index" label="#" width="60" />
        <el-table-column prop="host" label="域名" min-width="220">
          <template #default="{ row }">
            <span class="host">{{ row.host }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="images" label="引用图片数" width="140" />
        <el-table-column label="引用次数" width="200">
          <template #default="{ row }">
            <div class="hit-cell">
              <el-progress
                :percentage="percent(row.hits, stats?.top_hosts?.[0]?.hits)"
                :stroke-width="6"
                :show-text="false"
                class="hit-bar"
              />
              <span>{{ formatNumber(row.hits) }}</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-else description="暂无引用来源记录" :image-size="72" />
    </section>
  </div>
</template>

<style scoped lang="scss">
.card-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: var(--ph-space-3);
  margin-bottom: var(--ph-space-4);
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
  transition: box-shadow 0.2s ease, transform 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--ph-shadow-md);
  }

  .icon-wrap {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    border-radius: var(--ph-radius-md);
  }

  &.blue .icon-wrap {
    color: var(--ph-primary);
    background: var(--ph-primary-light-9);
  }

  &.purple .icon-wrap {
    color: #722ed1;
    background: rgb(114 46 209 / 10%);
  }

  &.green .icon-wrap {
    color: var(--ph-success);
    background: rgb(0 180 42 / 10%);
  }

  &.orange .icon-wrap {
    color: var(--ph-warning);
    background: rgb(255 125 0 / 10%);
  }

  &.cyan .icon-wrap {
    color: #0fc6c2;
    background: rgb(15 198 194 / 10%);
  }

  &.red .icon-wrap {
    color: var(--ph-danger);
    background: rgb(245 63 63 / 10%);
  }
}

.value {
  font-size: 20px;
  font-weight: 600;
  color: var(--ph-text-primary);
}

.label {
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
  gap: var(--ph-space-4);
  margin-bottom: var(--ph-space-4);
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
  margin-bottom: var(--ph-space-3);

  h3 {
    font-size: 15px;
  }
}

.muted {
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.more {
  font-size: 12px;
}

.storage-value {
  display: flex;
  align-items: baseline;
  gap: var(--ph-space-1);
  margin-bottom: var(--ph-space-3);

  .big {
    font-size: 24px;
    font-weight: 600;
  }
}

.storage-meta {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--ph-space-2);
  margin-top: var(--ph-space-4);
  text-align: center;

  .num {
    font-size: 18px;
    font-weight: 600;
  }
}

.hosts {
  margin-bottom: var(--ph-space-4);
}

.host {
  font-family: var(--ph-font-mono);
  font-size: 12px;
}

.hit-cell {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);

  .hit-bar {
    flex: 1;
  }
}

@media (max-width: 1199px) {
  .card-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 767.98px) {
  .card-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--ph-space-2);
  }

  .stat-card {
    padding: var(--ph-space-3);
  }

  .panel {
    padding: var(--ph-space-4);
  }
}
</style>
