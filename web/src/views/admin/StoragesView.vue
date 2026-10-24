<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'

import { adminStorageApi, type StoragePayload, type StorageUpdatePayload } from '@/api/storage'
import PageHeader from '@/components/common/PageHeader.vue'
import type { StorageStrategy } from '@/types/api'
import { formatDate } from '@/utils/format'

const storages = ref<StorageStrategy[]>([])
const loading = ref(false)
const testing = ref<number | null>(null)
const dialogOpen = ref(false)
const editing = ref<StorageStrategy | null>(null)
const submitting = ref(false)

const driverOptions = [
  { value: 'local', label: '本地存储' },
  { value: 's3', label: 'S3 / 兼容对象存储' },
  { value: 'oss', label: '阿里云 OSS' },
  { value: 'cos', label: '腾讯云 COS' },
  { value: 'webdav', label: 'WebDAV' },
]

const form = reactive({
  name: '',
  driver: 'local',
  domain: '',
  path_prefix: '',
  is_default: false,
  status: 1,
  root_path: './data/uploads',
  path_template: '{Y}/{m}/{d}/{hash}{ext}',
  endpoint: '',
  region: '',
  bucket: '',
  path_style: false,
  use_ssl: true,
  public_url: '',
  url: '',
  access_key: '',
  secret_key: '',
  username: '',
  password: '',
})

const isObjectStorage = computed(() => ['s3', 'oss', 'cos'].includes(form.driver))

function driverLabel(driver: string): string {
  return driverOptions.find((item) => item.value === driver)?.label || driver
}

async function load(): Promise<void> {
  loading.value = true
  try {
    storages.value = await adminStorageApi.list()
  } finally {
    loading.value = false
  }
}

function openCreate(): void {
  editing.value = null
  Object.assign(form, {
    name: '',
    driver: 'local',
    domain: '',
    path_prefix: '',
    is_default: false,
    status: 1,
    root_path: './data/uploads',
    path_template: '{Y}/{m}/{d}/{hash}{ext}',
    endpoint: '',
    region: '',
    bucket: '',
    path_style: false,
    use_ssl: true,
    public_url: '',
    url: '',
    access_key: '',
    secret_key: '',
    username: '',
    password: '',
  })
  dialogOpen.value = true
}

function openEdit(storage: StorageStrategy): void {
  editing.value = storage
  Object.assign(form, {
    name: storage.name,
    driver: storage.driver,
    domain: storage.domain,
    path_prefix: storage.path_prefix,
    is_default: storage.is_default,
    status: storage.status,
    root_path: storage.root_path || '',
    path_template: storage.path_template || '',
    endpoint: storage.endpoint || '',
    region: storage.region || '',
    bucket: storage.bucket || '',
    path_style: storage.path_style,
    use_ssl: storage.use_ssl !== false,
    public_url: storage.public_url || '',
    url: storage.url || '',
    access_key: '',
    secret_key: '',
    username: '',
    password: '',
  })
  dialogOpen.value = true
}

async function submit(): Promise<void> {
  if (!form.name.trim()) {
    ElMessage.warning('请填写策略名称')
    return
  }
  const payload: StoragePayload = {
    name: form.name.trim(),
    driver: form.driver as StoragePayload['driver'],
    domain: form.domain.trim() || undefined,
    path_prefix: form.path_prefix.trim() || undefined,
    is_default: form.is_default,
    status: form.status,
    root_path: form.driver === 'local' ? form.root_path : undefined,
    path_template: form.path_template || undefined,
    endpoint: isObjectStorage.value ? form.endpoint : undefined,
    region: isObjectStorage.value ? form.region : undefined,
    bucket: isObjectStorage.value ? form.bucket : undefined,
    path_style: isObjectStorage.value ? form.path_style : undefined,
    use_ssl: isObjectStorage.value ? form.use_ssl : undefined,
    public_url: form.public_url || undefined,
    url: form.driver === 'webdav' ? form.url : undefined,
    access_key: form.driver !== 'local' && form.driver !== 'webdav' ? form.access_key || undefined : undefined,
    secret_key: form.driver !== 'local' && form.driver !== 'webdav' ? form.secret_key || undefined : undefined,
    username: form.driver === 'webdav' ? form.username || undefined : undefined,
    password: form.driver === 'webdav' ? form.password || undefined : undefined,
  }
  submitting.value = true
  try {
    if (editing.value) {
      const update: StorageUpdatePayload = { ...payload }
      delete (update as Record<string, unknown>).driver
      await adminStorageApi.update(editing.value.id, update)
      ElMessage.success('存储策略已更新')
    } else {
      await adminStorageApi.create(payload)
      ElMessage.success('存储策略已创建')
    }
    dialogOpen.value = false
    await load()
  } finally {
    submitting.value = false
  }
}

async function test(storage: StorageStrategy): Promise<void> {
  testing.value = storage.id
  try {
    const result = await adminStorageApi.test(storage.id)
    if (result.ok) {
      ElMessage.success(result.message || '连接成功')
    } else {
      ElMessage.error(result.message || '连接失败')
    }
    await load()
  } finally {
    testing.value = null
  }
}

async function remove(storage: StorageStrategy): Promise<void> {
  try {
    await ElMessageBox.confirm(`确定删除存储策略「${storage.name}」吗？已上传的文件不会被删除。`, '删除存储策略', {
      type: 'warning',
    })
  } catch {
    return
  }
  await adminStorageApi.remove(storage.id)
  ElMessage.success('存储策略已删除')
  await load()
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="存储策略" description="配置本地磁盘与对象存储后端，支持设置默认策略与连通性测试">
      <template #actions>
        <el-button type="primary" :icon="Plus" @click="openCreate">新建策略</el-button>
      </template>
    </PageHeader>

    <el-table :data="storages" v-loading="loading" row-key="id">
      <el-table-column label="名称" min-width="160">
        <template #default="{ row }">
          <div class="name-cell">
            <span class="name">{{ row.name }}</span>
            <el-tag v-if="row.is_default" type="success" size="small">默认</el-tag>
            <el-tag v-if="row.status !== 1" type="info" size="small">已停用</el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="驱动" width="160">
        <template #default="{ row }">
          <el-tag size="small" effect="plain">{{ driverLabel(row.driver) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="访问域名" min-width="180">
        <template #default="{ row }">
          <span class="mono">{{ row.domain || '跟随站点' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="路径" min-width="180">
        <template #default="{ row }">
          <span class="mono">{{ row.driver === 'local' ? row.root_path : row.bucket || row.url }}</span>
        </template>
      </el-table-column>
      <el-table-column label="密钥" width="90">
        <template #default="{ row }">
          <el-tag :type="row.has_secret ? 'success' : 'info'" size="small">
            {{ row.has_secret ? '已配置' : '未配置' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="连通性" width="150">
        <template #default="{ row }">
          <span v-if="row.last_test_at" class="test-cell">
            <el-tag :type="row.last_test_ok ? 'success' : 'danger'" size="small">
              {{ row.last_test_ok ? '正常' : '失败' }}
            </el-tag>
            <span class="muted">{{ formatDate(row.last_test_at) }}</span>
          </span>
          <span v-else class="muted">未测试</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button text size="small" :loading="testing === row.id" @click="test(row)">测试</el-button>
          <el-button text size="small" @click="openEdit(row)">编辑</el-button>
          <el-button text size="small" type="danger" :disabled="row.is_default" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="还没有存储策略" :image-size="72" />
      </template>
    </el-table>

    <el-dialog v-model="dialogOpen" :title="editing ? '编辑存储策略' : '新建存储策略'" width="600px">
      <el-form label-position="top">
        <div class="form-grid">
          <el-form-item label="策略名称" required>
            <el-input v-model="form.name" maxlength="64" show-word-limit placeholder="例如：本地磁盘" />
          </el-form-item>
          <el-form-item label="存储驱动" required>
            <el-select v-model="form.driver" :disabled="!!editing" class="w-full">
              <el-option v-for="item in driverOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="访问域名">
            <el-input v-model="form.domain" placeholder="https://img.example.com，留空跟随站点" />
          </el-form-item>
          <el-form-item label="路径前缀">
            <el-input v-model="form.path_prefix" placeholder="例如：images" />
          </el-form-item>
        </div>

        <template v-if="form.driver === 'local'">
          <div class="form-grid">
            <el-form-item label="存储根目录">
              <el-input v-model="form.root_path" placeholder="./data/uploads" />
            </el-form-item>
            <el-form-item label="路径模板">
              <el-input v-model="form.path_template" placeholder="{Y}/{m}/{d}/{hash}{ext}" />
            </el-form-item>
          </div>
        </template>

        <template v-else-if="isObjectStorage">
          <div class="form-grid">
            <el-form-item label="Endpoint">
              <el-input v-model="form.endpoint" placeholder="s3.amazonaws.com 或 oss-cn-hangzhou.aliyuncs.com" />
            </el-form-item>
            <el-form-item label="Region">
              <el-input v-model="form.region" placeholder="us-east-1" />
            </el-form-item>
            <el-form-item label="Bucket">
              <el-input v-model="form.bucket" placeholder="my-bucket" />
            </el-form-item>
            <el-form-item label="公开访问地址">
              <el-input v-model="form.public_url" placeholder="CDN 或自定义域名，可留空" />
            </el-form-item>
            <el-form-item label="AccessKey">
              <el-input v-model="form.access_key" :placeholder="editing?.has_secret ? '留空表示不修改' : '访问密钥'" />
            </el-form-item>
            <el-form-item label="SecretKey">
              <el-input
                v-model="form.secret_key"
                type="password"
                show-password
                :placeholder="editing?.has_secret ? '留空表示不修改' : '访问密钥'"
              />
            </el-form-item>
          </div>
          <el-form-item label="选项">
            <div class="switches">
              <el-switch v-model="form.path_style" active-text="使用 Path Style" />
              <el-switch v-model="form.use_ssl" active-text="使用 HTTPS" />
            </div>
          </el-form-item>
        </template>

        <template v-else-if="form.driver === 'webdav'">
          <div class="form-grid">
            <el-form-item label="WebDAV 地址">
              <el-input v-model="form.url" placeholder="https://dav.example.com/remote.php/dav" />
            </el-form-item>
            <el-form-item label="公开访问地址">
              <el-input v-model="form.public_url" placeholder="可选" />
            </el-form-item>
            <el-form-item label="用户名">
              <el-input v-model="form.username" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input
                v-model="form.password"
                type="password"
                show-password
                :placeholder="editing?.has_secret ? '留空表示不修改' : ''"
              />
            </el-form-item>
          </div>
        </template>

        <el-form-item label="状态">
          <div class="switches">
            <el-switch v-model="form.is_default" active-text="设为默认存储" />
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

.mono {
  font-family: var(--ph-font-mono);
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.test-cell {
  display: inline-flex;
  align-items: center;
  gap: var(--ph-space-1);
}

.muted {
  font-size: 11px;
  color: var(--ph-text-tertiary);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 var(--ph-space-4);
}

.switches {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ph-space-4);
}

@media (max-width: 767.98px) {
  .form-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
