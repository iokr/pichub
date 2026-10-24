<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Message } from '@element-plus/icons-vue'

import { authApi } from '@/api/auth'
import AuthShell from '@/components/auth/AuthShell.vue'
import CaptchaField from '@/components/common/CaptchaField.vue'
import { useSettingStore } from '@/stores/setting'

const setting = useSettingStore()

const formRef = ref<FormInstance>()
const captchaRef = ref<InstanceType<typeof CaptchaField> | null>(null)
const captchaId = ref('')
const sending = ref(false)
const sent = ref(false)

const form = reactive({ email: '', captcha_code: '' })
const captchaEnabled = computed(() => setting.config.captcha_enabled)

const rules: FormRules = {
  email: [
    { required: true, message: '请输入注册邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' },
  ],
}

async function submit(): Promise<void> {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  sending.value = true
  try {
    await authApi.forgotPassword(form.email.trim(), {
      captcha_id: captchaEnabled.value ? captchaId.value : undefined,
      captcha_code: captchaEnabled.value ? form.captcha_code : undefined,
    })
    sent.value = true
    ElMessage.success('如果邮箱已注册，重置邮件已发送')
  } catch {
    if (captchaEnabled.value) {
      captchaRef.value?.refresh()
    }
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <AuthShell>
    <h2>找回密码</h2>
    <p class="subtitle">输入注册邮箱，我们会发送一封重置密码的邮件</p>

    <el-alert
      v-if="sent"
      type="success"
      :closable="false"
      show-icon
      title="重置邮件已发送"
      description="请查收邮件并点击其中的链接完成密码重置，链接有效期有限。"
      class="alert"
    />

    <el-form ref="formRef" :model="form" :rules="rules" size="large" class="auth-form" @keyup.enter="submit">
      <el-form-item prop="email">
        <el-input v-model="form.email" placeholder="注册邮箱" :prefix-icon="Message" autocomplete="email" />
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
      <el-button type="primary" class="submit" :loading="sending" @click="submit">发送重置邮件</el-button>
    </el-form>

    <div class="links">
      <router-link to="/login">返回登录</router-link>
      <router-link to="/register">注册账号</router-link>
    </div>
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
  display: flex;
  justify-content: space-between;
  margin-top: var(--ph-space-4);
  font-size: 13px;
}
</style>
