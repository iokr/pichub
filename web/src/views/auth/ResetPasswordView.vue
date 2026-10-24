<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Lock } from '@element-plus/icons-vue'

import { authApi } from '@/api/auth'
import AuthShell from '@/components/auth/AuthShell.vue'

const route = useRoute()
const router = useRouter()

const token = computed(() => (typeof route.query.token === 'string' ? route.query.token : ''))
const formRef = ref<FormInstance>()
const submitting = ref(false)
const form = reactive({ password: '', confirm: '' })

const rules: FormRules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, max: 64, message: '密码长度为 8-64 个字符', trigger: 'blur' },
  ],
  confirm: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (_rule, value, callback) => {
        if (value !== form.password) {
          callback(new Error('两次输入的密码不一致'))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
}

async function submit(): Promise<void> {
  if (!token.value) {
    ElMessage.error('缺少重置令牌，请从邮件中的链接重新进入')
    return
  }
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  submitting.value = true
  try {
    await authApi.resetPassword(token.value, form.password)
    ElMessage.success('密码已重置，请使用新密码登录')
    await router.replace('/login')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AuthShell>
    <h2>重置密码</h2>
    <p class="subtitle">为账号设置一个新密码</p>

    <el-alert
      v-if="!token"
      type="warning"
      :closable="false"
      show-icon
      title="链接无效"
      description="当前地址缺少重置令牌，请从邮件中的链接重新进入。"
      class="alert"
    />

    <el-form ref="formRef" :model="form" :rules="rules" size="large" class="auth-form" @keyup.enter="submit">
      <el-form-item prop="password">
        <el-input
          v-model="form.password"
          type="password"
          show-password
          placeholder="新密码（至少 8 位）"
          :prefix-icon="Lock"
          autocomplete="new-password"
        />
      </el-form-item>
      <el-form-item prop="confirm">
        <el-input
          v-model="form.confirm"
          type="password"
          show-password
          placeholder="确认新密码"
          :prefix-icon="Lock"
          autocomplete="new-password"
        />
      </el-form-item>
      <el-button type="primary" class="submit" :loading="submitting" :disabled="!token" @click="submit">
        重置密码
      </el-button>
    </el-form>

    <p class="links">
      <router-link to="/login">返回登录</router-link>
    </p>
  </AuthShell>
</template>

<style scoped lang="scss">
h2 {
  font-size: 26px;
}

.subtitle {
  margin: var(--ph-space-2) 0 var(--ph-space-5);
  font-size: 13px;
  color: var(--ph-text-secondary);
}

.alert {
  margin-bottom: var(--ph-space-4);
}

.auth-form {
  :deep(.el-input__wrapper) {
    min-height: 44px;
  }

  :deep(.el-form-item) {
    margin-bottom: var(--ph-space-4);
  }
}

.submit {
  width: 100%;
  height: 44px;
}

.links {
  margin-top: var(--ph-space-4);
  font-size: 13px;
}
</style>
