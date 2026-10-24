<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Lock, User } from '@element-plus/icons-vue'

import AuthShell from '@/components/auth/AuthShell.vue'
import CaptchaField from '@/components/common/CaptchaField.vue'
import { useSettingStore } from '@/stores/setting'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const setting = useSettingStore()
const user = useUserStore()

const formRef = ref<FormInstance>()
const captchaRef = ref<InstanceType<typeof CaptchaField> | null>(null)
const captchaId = ref('')
const form = reactive({ account: '', password: '', captcha_code: '' })

const captchaEnabled = computed(() => setting.config.captcha_enabled)

const rules: FormRules = {
  account: [{ required: true, message: '请输入用户名或邮箱', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function submit(): Promise<void> {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  try {
    await user.login({
      account: form.account.trim(),
      password: form.password,
      captcha_id: captchaEnabled.value ? captchaId.value : undefined,
      captcha_code: captchaEnabled.value ? form.captcha_code : undefined,
    })
    ElMessage.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect)
  } catch {
    if (captchaEnabled.value) {
      captchaRef.value?.refresh()
    }
  }
}
</script>

<template>
  <AuthShell>
    <h2>欢迎回来</h2>
    <p class="subtitle">登录 {{ setting.config.name }}，继续管理你的图片</p>

    <el-form ref="formRef" :model="form" :rules="rules" size="large" class="auth-form" @keyup.enter="submit">
      <el-form-item prop="account">
        <el-input v-model="form.account" placeholder="用户名或邮箱" :prefix-icon="User" autocomplete="username" />
      </el-form-item>
      <el-form-item prop="password">
        <el-input
          v-model="form.password"
          type="password"
          show-password
          placeholder="密码"
          :prefix-icon="Lock"
          autocomplete="current-password"
        />
      </el-form-item>
      <el-form-item v-if="captchaEnabled">
        <CaptchaField
          ref="captchaRef"
          v-model="form.captcha_code"
          v-model:id="captchaId"
          :enabled="captchaEnabled"
          class="w-full"
        />
      </el-form-item>
      <el-button type="primary" class="submit" :loading="user.loading" @click="submit">登录</el-button>
    </el-form>

    <div class="links">
      <router-link v-if="setting.config.register_enabled" to="/register">注册账号</router-link>
      <router-link to="/forgot-password">忘记密码</router-link>
    </div>
  </AuthShell>
</template>

<style scoped lang="scss">
h2 {
  font-size: 26px;
}

.subtitle {
  margin: var(--ph-space-2) 0 var(--ph-space-6);
  font-size: 13px;
  color: var(--ph-text-secondary);
}

.auth-form {
  :deep(.el-form-item) {
    margin-bottom: var(--ph-space-4);
  }

  :deep(.el-input__wrapper) {
    min-height: 44px;
  }
}

.submit {
  width: 100%;
  height: 44px;
  margin-top: var(--ph-space-2);
}

.links {
  display: flex;
  justify-content: space-between;
  margin-top: var(--ph-space-4);
  font-size: 13px;
}
</style>
