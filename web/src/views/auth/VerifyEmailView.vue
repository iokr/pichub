<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { CircleCheckFilled, CircleCloseFilled, Loading } from '@element-plus/icons-vue'

import { authApi } from '@/api/auth'
import AuthShell from '@/components/auth/AuthShell.vue'

const route = useRoute()
const router = useRouter()

const token = computed(() => (typeof route.query.token === 'string' ? route.query.token : ''))
const status = ref<'loading' | 'success' | 'error'>('loading')
const message = ref('正在验证邮箱…')

async function verify(): Promise<void> {
  if (!token.value) {
    status.value = 'error'
    message.value = '缺少验证令牌，请从邮件中的链接重新进入。'
    return
  }
  try {
    await authApi.verifyEmail(token.value)
    status.value = 'success'
    message.value = '邮箱验证成功，即将前往工作台。'
    window.setTimeout(() => {
      void router.replace('/')
    }, 1600)
  } catch (error) {
    status.value = 'error'
    message.value = error instanceof Error ? error.message : '验证失败，链接可能已过期。'
  }
}

onMounted(verify)
</script>

<template>
  <AuthShell>
    <div class="verify">
      <el-icon v-if="status === 'loading'" :size="56" class="icon loading"><Loading /></el-icon>
      <el-icon v-else-if="status === 'success'" :size="56" class="icon success"><CircleCheckFilled /></el-icon>
      <el-icon v-else :size="56" class="icon error"><CircleCloseFilled /></el-icon>

      <h2>{{ status === 'success' ? '验证成功' : status === 'error' ? '验证失败' : '验证中' }}</h2>
      <p class="message">{{ message }}</p>

      <div class="actions">
        <el-button v-if="status === 'error'" @click="verify">重新验证</el-button>
        <el-button type="primary" @click="router.replace('/login')">前往登录</el-button>
      </div>
    </div>
  </AuthShell>
</template>

<style scoped lang="scss">
.verify {
  text-align: center;
}

.icon {
  &.loading {
    color: var(--ph-primary);
    animation: spin 1s linear infinite;
  }

  &.success {
    color: var(--ph-success);
  }

  &.error {
    color: var(--ph-danger);
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

h2 {
  margin-top: var(--ph-space-4);
  font-size: 24px;
}

.message {
  margin-top: var(--ph-space-2);
  color: var(--ph-text-secondary);
}

.actions {
  display: flex;
  justify-content: center;
  gap: var(--ph-space-3);
  margin-top: var(--ph-space-6);
}
</style>
