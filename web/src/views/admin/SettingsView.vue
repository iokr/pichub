<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

import { adminApi } from '@/api/admin'
import PageHeader from '@/components/common/PageHeader.vue'
import { useSettingStore } from '@/stores/setting'
import type { SettingItem } from '@/types/api'

const setting = useSettingStore()

const items = ref<SettingItem[]>([])
const values = reactive<Record<string, string>>({})
const extensions = ref<string[]>([])
const loading = ref(false)
const saving = ref(false)

const groups = computed(() => {
  const map = new Map<string, SettingItem[]>()
  for (const item of items.value) {
    const group = groupLabel(item.key)
    if (!map.has(group)) {
      map.set(group, [])
    }
    map.get(group)!.push(item)
  }
  return Array.from(map.entries()).map(([label, list]) => ({ label, list }))
})

function groupLabel(key: string): string {
  if (key.startsWith('site.')) return '站点信息'
  if (key.startsWith('register.')) return '注册与邀请'
  if (key.startsWith('captcha.')) return '验证码'
  if (key.startsWith('upload.')) return '上传设置'
  if (key.startsWith('referer.')) return '引用统计'
  if (key.startsWith('s3_gateway.')) return 'S3 兼容网关'
  return '其他设置'
}

function isBool(item: SettingItem): boolean {
  return item.type === 'bool'
}

function boolValue(key: string): boolean {
  const value = values[key]
  return value === 'true' || value === '1'
}

function setBool(key: string, value: boolean): void {
  values[key] = value ? 'true' : 'false'
}

function isJson(item: SettingItem): boolean {
  return item.type === 'json'
}

async function load(): Promise<void> {
  loading.value = true
  try {
    items.value = await adminApi.settings()
    for (const item of items.value) {
      if (item.key === 'upload.allowed_extensions') {
        try {
          const parsed = JSON.parse(item.value)
          extensions.value = Array.isArray(parsed) ? parsed.map((ext: string) => ext.replace(/^\./, '')) : []
        } catch {
          extensions.value = item.value
            .replace(/[[\]"]/g, '')
            .split(',')
            .map((ext) => ext.trim())
            .filter(Boolean)
        }
        values[item.key] = JSON.stringify(extensions.value)
      } else {
        values[item.key] = item.value
      }
    }
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  if (!items.value.length) {
    return
  }
  const payload: Record<string, string> = {}
  for (const item of items.value) {
    payload[item.key] = item.key === 'upload.allowed_extensions' ? JSON.stringify(extensions.value) : values[item.key] ?? ''
  }
  saving.value = true
  try {
    await adminApi.updateSettings(payload)
    ElMessage.success('系统设置已保存')
    await setting.load(true)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="系统设置" description="修改后立即生效，密文项显示为掩码，留空保持不变">
      <template #actions>
        <el-button @click="load">重新加载</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存设置</el-button>
      </template>
    </PageHeader>

    <el-skeleton v-if="loading" animated :rows="8" />

    <div v-else class="groups">
      <section v-for="group in groups" :key="group.label" class="panel">
        <h3>{{ group.label }}</h3>
        <div class="fields">
          <div v-for="item in group.list" :key="item.key" class="field">
            <div class="field-label">
              <span class="label">{{ item.description || item.key }}</span>
              <code class="key">{{ item.key }}</code>
            </div>
            <div class="field-control">
              <el-switch
                v-if="isBool(item)"
                :model-value="boolValue(item.key)"
                @update:model-value="(value: string | number | boolean) => setBool(item.key, !!value)"
              />
              <el-select
                v-else-if="item.key === 'upload.allowed_extensions'"
                v-model="extensions"
                multiple
                filterable
                allow-create
                default-first-option
                placeholder="输入扩展名后回车添加"
                class="extensions"
              />
              <el-select
                v-else-if="item.key === 'register.mode'"
                v-model="values[item.key]"
                class="w-full"
              >
                <el-option label="开放注册" value="open" />
                <el-option label="仅邀请码" value="invite" />
              </el-select>
              <el-select
                v-else-if="item.key === 'register.require_email_verify'"
                v-model="values[item.key]"
                class="w-full"
              >
                <el-option label="关闭验证" value="off" />
                <el-option label="软性提醒" value="soft" />
                <el-option label="强制验证" value="force" />
              </el-select>
              <el-select
                v-else-if="item.key === 'upload.default_visibility'"
                v-model="values[item.key]"
                class="w-full"
              >
                <el-option label="公开" value="public" />
                <el-option label="私有" value="private" />
              </el-select>
              <el-input-number
                v-else-if="item.type === 'int'"
                :model-value="Number(values[item.key]) || 0"
                :min="0"
                class="w-full"
                @update:model-value="(value: number | undefined) => (values[item.key] = String(value ?? 0))"
              />
              <el-input
                v-else-if="item.is_secret"
                v-model="values[item.key]"
                type="password"
                show-password
                placeholder="留空或保持 ****** 表示不修改"
              />
              <el-input v-else v-model="values[item.key]" :placeholder="item.description" />
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped lang="scss">
.groups {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-4);
  max-width: 900px;
}

.panel {
  padding: var(--ph-space-5);
  background: var(--ph-fill-card);
  border: 1px solid var(--ph-border-light);
  border-radius: var(--ph-radius-lg);

  h3 {
    margin-bottom: var(--ph-space-4);
    font-size: 15px;
  }
}

.fields {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-4);
}

.field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.2fr);
  gap: var(--ph-space-4);
  align-items: start;
}

.field-label {
  display: flex;
  flex-direction: column;
  gap: 2px;

  .label {
    font-size: 13px;
    color: var(--ph-text-primary);
  }

  .key {
    font-family: var(--ph-font-mono);
    font-size: 11px;
    color: var(--ph-text-tertiary);
  }
}

.field-control {
  display: flex;
  justify-content: flex-end;

  > * {
    max-width: 320px;
  }
}

.extensions {
  width: 100%;
}

@media (max-width: 767.98px) {
  .panel {
    padding: var(--ph-space-4);
  }

  .field {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--ph-space-2);
  }

  .field-control > * {
    max-width: none;
  }
}
</style>
