<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Lock, Message, User } from '@element-plus/icons-vue'

import { authApi } from '@/api/auth'
import AuthShell from '@/components/auth/AuthShell.vue'
import CaptchaField from '@/components/common/CaptchaField.vue'
import { useSettingStore } from '@/stores/setting'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const setting = useSettingStore()
const user = useUserStore()

const formRef = ref<FormInstance>()
const captchaRef = ref<InstanceType<typeof CaptchaField> | null>(null)
const captchaId = ref('')
const submitting = ref(false)

const form = reactive({
  username: '',
  email: '',
  password: '',
  confirm: '',
  captcha_code: '',
  invite_code: '',
})

const captchaEnabled = computed(() => setting.config.captcha_enabled)
const inviteRequired = computed(() => setting.config.register_mode === 'invite')

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名长度为 3-20 个字符', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_-]+$/, message: '仅支持字母、数字、下划线和短横线', trigger: 'blur' },
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, max: 64, message: '密码长度为 8-64 个字符', trigger: 'blur' },
  ],
  confirm: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
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
  invite_code: inviteRequired.value
    ? [{ required: true, message: '请输入邀请码', trigger: 'blur' }]
    : [],
}

async function submit(): Promise<void> {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  submitting.value = true
  try {
    await user.register({
      username: form.username.trim(),
      email: form.email.trim(),
      password: form.password,
      captcha_id: captchaEnabled.value ? captchaId.value : undefined,
      captcha_code: captchaEnabled.value ? form.captcha_code : undefined,
      invite_code: inviteRequired.value ? form.invite_code.trim() : undefined,
    })
    ElMessage.success('注册成功')
    await router.replace('/')
  } catch {
    if (captchaEnabled.value) {
      captchaRef.value?.refresh()
    }
  } finally {
    submitting.value = false
  }
}

async function checkUsername(): Promise<void> {
  const value = form.username.trim()
  if (!value) {
    return
  }
  const result = await authApi.checkUsername(value).catch(() => null)
  if (result && !result.available) {
    ElMessage.warning(result.message)
  }
}

async function checkEmail(): Promise<void> {
  const value = form.email.trim()
  if (!value) {
    return
  }
  const result = await authApi.checkEmail(value).catch(() => null)
  if (result && !result.available) {
    ElMessage.warning(result.message)
  }
}
</script>

<template>
  <AuthShell>
    <h2>创建账号</h2>
    <p class="subtitle">注册后即可上传、管理并分享图片</p>

    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
      size="large"
      class="auth-form"
      @keyup.enter="submit"
    >
      <el-form-item prop="username">
        <el-input
          v-model="form.username"
          placeholder="用户名"
          :prefix-icon="User"
          autocomplete="username"
          @blur="checkUsername"
        />
      </el-form-item>
      <el-form-item prop="email">
        <el-input
          v-model="form.email"
          placeholder="邮箱"
          :prefix-icon="Message"
          autocomplete="email"
          @blur="checkEmail"
        />
      </el-form-item>
      <el-form-item prop="password">
        <el-input
          v-model="form.password"
          type="password"
          show-password
          placeholder="密码（至少 8 位）"
          :prefix-icon="Lock"
          autocomplete="new-password"
        />
      </el-form-item>
      <el-form-item prop="confirm">
        <el-input
          v-model="form.confirm"
          type="password"
          show-password
          placeholder="确认密码"
          :prefix-icon="Lock"
          autocomplete="new-password"
        />
      </el-form-item>
      <el-form-item v-if="inviteRequired" prop="invite_code">
        <el-input v-model="form.invite_code" placeholder="邀请码" />
      </el-form-item>
      <el-form-item v-if="captchaEnabled" prop="captcha_code">
        <CaptchaField
          ref="captchaRef"
          v-model="form.captcha_code"
          v-model:id="captchaId"
          :enabled="captchaEnabled"
          class="w-full"
        />
      </el-form-item>

      <el-button type="primary" class="submit" :loading="submitting" @click="submit">注册</el-button>
    </el-form>

    <p class="links">
      已有账号？
      <router-link to="/login">直接登录</router-link>
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

.auth-form {
  :deep(.el-form-item) {
    margin-bottom: var(--ph-space-3);
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
  margin-top: var(--ph-space-4);
  font-size: 13px;
  color: var(--ph-text-secondary);
}
</style>
