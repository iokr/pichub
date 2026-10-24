<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

import { shareApi } from '@/api/share'
import CopyButton from '@/components/common/CopyButton.vue'
import type { Album, Image, Share } from '@/types/api'
import { fromNow } from '@/utils/format'

const props = defineProps<{
  image?: Image | null
  album?: Album | null
}>()

const open = defineModel<boolean>({ default: false })

const submitting = ref(false)
const created = ref<Share | null>(null)
const form = reactive({
  password: '',
  expires_in_hours: 0,
  max_views: 0,
})

const targetName = computed(() => props.image?.name || props.album?.name || '')
const targetType = computed<'image' | 'album'>(() => (props.image ? 'image' : 'album'))
const targetId = computed(() => props.image?.id || props.album?.id || 0)

watch(open, (value) => {
  if (value) {
    created.value = null
    form.password = ''
    form.expires_in_hours = 0
    form.max_views = 0
  }
})

async function submit(): Promise<void> {
  if (!targetId.value) {
    ElMessage.warning('缺少分享目标')
    return
  }
  submitting.value = true
  try {
    created.value = await shareApi.create({
      target_type: targetType.value,
      target_id: targetId.value,
      password: form.password.trim() || undefined,
      expires_in_hours: form.expires_in_hours || undefined,
      max_views: form.max_views || undefined,
    })
    ElMessage.success('分享链接已创建')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <el-dialog v-model="open" title="创建分享链接" width="480px">
    <template v-if="!created">
      <p class="target">
        分享{{ targetType === 'image' ? '图片' : '相册' }}：<strong>{{ targetName }}</strong>
      </p>
      <el-form label-position="top">
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
    </template>

    <div v-else class="result">
      <el-result icon="success" title="分享创建成功" :sub-title="created.has_password ? '访问者需要输入密码' : '可直接分享给他人'">
        <template #extra>
          <div class="link-row">
            <el-input :model-value="created.url" readonly />
            <CopyButton :text="created.url" />
          </div>
          <p class="expire">
            <span v-if="created.expires_at">{{ fromNow(created.expires_at) }}过期</span>
            <span v-else>永久有效</span>
            <span v-if="created.max_views"> · 最多访问 {{ created.max_views }} 次</span>
          </p>
        </template>
      </el-result>
    </div>

    <template #footer>
      <template v-if="!created">
        <el-button @click="open = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">创建分享</el-button>
      </template>
      <template v-else>
        <el-button @click="open = false">关闭</el-button>
        <el-button type="primary" @click="created = null">再创建一个</el-button>
      </template>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.target {
  margin-bottom: var(--ph-space-3);
  font-size: 13px;
  color: var(--ph-text-secondary);
}

.hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--ph-text-tertiary);
}

.link-row {
  display: flex;
  gap: var(--ph-space-2);
  width: 100%;
}

.expire {
  margin-top: var(--ph-space-2);
  font-size: 12px;
  color: var(--ph-text-secondary);
}
</style>
