<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { Refresh } from '@element-plus/icons-vue'

import { authApi } from '@/api/auth'

const props = defineProps<{ enabled: boolean }>()

const captchaId = defineModel<string>('id', { default: '' })
const code = defineModel<string>({ default: '' })

const image = ref('')
const loading = ref(false)

function normalize(base64: string): string {
  if (!base64) {
    return ''
  }
  return base64.startsWith('data:') ? base64 : `data:image/png;base64,${base64}`
}

async function refresh(): Promise<void> {
  if (!props.enabled) {
    return
  }
  loading.value = true
  try {
    const data = await authApi.captcha()
    captchaId.value = data.captcha_id
    image.value = normalize(data.image_base64)
    code.value = ''
  } catch {
    image.value = ''
  } finally {
    loading.value = false
  }
}

watch(
  () => props.enabled,
  (enabled) => {
    if (enabled) {
      void refresh()
    }
  },
)

onMounted(() => {
  if (props.enabled) {
    void refresh()
  }
})

defineExpose({ refresh })
</script>

<template>
  <div v-if="enabled" class="captcha-field">
    <el-input v-model="code" placeholder="图片验证码" maxlength="8" autocomplete="off" />
    <button type="button" class="captcha-image" aria-label="刷新验证码" @click="refresh">
      <img v-if="image" :src="image" alt="验证码" />
      <span v-else class="placeholder">
        <el-icon v-if="!loading"><Refresh /></el-icon>
        {{ loading ? '加载中' : '点击获取' }}
      </span>
    </button>
  </div>
</template>

<style scoped lang="scss">
.captcha-field {
  display: flex;
  gap: var(--ph-space-3);
}

.captcha-image {
  flex: 0 0 120px;
  height: 40px;
  padding: 0;
  border: 1px solid var(--ph-border);
  border-radius: var(--ph-radius-md);
  background: var(--ph-fill);
  cursor: pointer;
  overflow: hidden;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  height: 100%;
  font-size: 12px;
  color: var(--ph-text-secondary);
}
</style>
