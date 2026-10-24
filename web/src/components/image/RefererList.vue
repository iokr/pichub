<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import { imageApi } from '@/api/image'
import type { Referer } from '@/types/api'
import { formatDate } from '@/utils/format'

const props = defineProps<{ imageId: number }>()

const list = ref<Referer[]>([])
const total = ref(0)
const page = ref(1)
const size = 10
const loading = ref(false)

async function load(): Promise<void> {
  loading.value = true
  try {
    const resp = await imageApi.referers(props.imageId, { page: page.value, size })
    list.value = resp.list ?? []
    total.value = resp.total ?? 0
  } finally {
    loading.value = false
  }
}

async function remove(item: Referer): Promise<void> {
  try {
    await ElMessageBox.confirm(`确定删除来自 ${item.host} 的引用记录吗？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  await imageApi.deleteReferer(props.imageId, item.id)
  ElMessage.success('已删除')
  await load()
}

onMounted(load)

defineExpose({ reload: load })
</script>

<template>
  <div class="referer-list">
    <el-table v-loading="loading" :data="list" size="small">
      <el-table-column label="来源" min-width="180">
        <template #default="{ row }">
          <a :href="row.url" target="_blank" rel="noopener noreferrer" class="host">{{ row.host }}</a>
        </template>
      </el-table-column>
      <el-table-column prop="hits" label="命中" width="72" align="right" />
      <el-table-column label="最近出现" width="140">
        <template #default="{ row }">{{ formatDate(row.last_seen_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="64" align="right">
        <template #default="{ row }">
          <el-button link type="danger" size="small" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-if="total > size"
      class="pager"
      layout="prev, pager, next"
      :total="total"
      :page-size="size"
      :current-page="page"
      @current-change="(value: number) => { page = value; load() }"
    />
  </div>
</template>

<style scoped lang="scss">
.host {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pager {
  margin-top: var(--ph-space-3);
  justify-content: flex-end;
}
</style>
