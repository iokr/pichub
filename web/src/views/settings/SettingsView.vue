<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Bell, Lock, User } from '@element-plus/icons-vue'

import { authApi } from '@/api/auth'
import { storageApi } from '@/api/storage'
import { userApi } from '@/api/user'
import CaptchaField from '@/components/common/CaptchaField.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import { useSettingStore } from '@/stores/setting'
import { useUserStore } from '@/stores/user'
import type { StorageStrategy } from '@/types/api'
import { formatBytes, formatDate, percent } from '@/utils/format'

const setting = useSettingStore()
const user = useUserStore()

const activeTab = ref('profile')
const storages = ref<StorageStrategy[]>([])
const savingProfile = ref(false)
const savingPassword = ref(false)
const sendingVerify = ref(false)

const profile = reactive({
  nickname: '',
  avatar_url: '',
  default_visibility: 'public' as 'public' | 'private',
  referer_stat_enabled: true,
  storage_config_id: undefined as number | undefined,
})

const passwordForm = reactive({
  old_password: '',
  new_password: '',
  confirm: '',
  captcha_id: '',
  captcha_code: '',
})

const current = computed(() => user.user)
const quotaPercent = computed(() => percent(current.value?.used_bytes, current.value?.quota_bytes))
const roleText = computed(() => (current.value?.role === 1 ? '管理员' : '普通用户'))

watch(
  () => user.user,
  (value) => {
    if (!value) {
      return
    }
    profile.nickname = value.nickname
    profile.avatar_url = value.avatar_url
    profile.default_visibility = value.default_visibility || 'public'
    profile.referer_stat_enabled = value.referer_stat_enabled
    profile.storage_config_id = value.storage_config_id ?? undefined
  },
  { immediate: true },
)

async function saveProfile(): Promise<void> {
  savingProfile.value = true
  try {
    await user.updateProfile({
      nickname: profile.nickname.trim(),
      avatar_url: profile.avatar_url.trim(),
      default_visibility: profile.default_visibility,
      referer_stat_enabled: profile.referer_stat_enabled,
      storage_config_id: profile.storage_config_id,
    })
    ElMessage.success('资料已保存')
  } finally {
    savingProfile.value = false
  }
}

async function changePassword(): Promise<void> {
  if (!passwordForm.old_password || !passwordForm.new_password) {
    ElMessage.warning('请填写完整密码信息')
    return
  }
  if (passwordForm.new_password.length < 8) {
    ElMessage.warning('新密码至少 8 位')
    return
  }
  if (passwordForm.new_password !== passwordForm.confirm) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  savingPassword.value = true
  try {
    await userApi.changePassword({
      old_password: passwordForm.old_password,
      new_password: passwordForm.new_password,
      captcha_id: passwordForm.captcha_id || undefined,
      captcha_code: passwordForm.captcha_code || undefined,
    })
    ElMessage.success('密码已更新，请使用新密码重新登录')
    passwordForm.old_password = ''
    passwordForm.new_password = ''
    passwordForm.confirm = ''
    passwordForm.captcha_code = ''
  } finally {
    savingPassword.value = false
  }
}

async function resendVerify(): Promise<void> {
  if (!current.value?.email) {
    return
  }
  sendingVerify.value = true
  try {
    await authApi.resendVerify(current.value.email)
    ElMessage.success('验证邮件已发送，请查收')
  } finally {
    sendingVerify.value = false
  }
}

onMounted(async () => {
  if (!user.user) {
    await user.fetchProfile().catch(() => undefined)
  }
  storages.value = await storageApi.list().catch(() => [] as StorageStrategy[])
})
</script>

<template>
  <div>
    <PageHeader title="设置" description="管理个人资料、安全选项与上传偏好" />

    <el-tabs v-model="activeTab" class="settings-tabs">
      <el-tab-pane name="profile">
        <template #label>
          <span class="tab-label"><el-icon><User /></el-icon>个人资料</span>
        </template>

        <div class="panel">
          <div class="avatar-row">
            <el-avatar :size="72" :src="profile.avatar_url || current?.avatar_url">
              {{ (current?.nickname || current?.username || 'U').slice(0, 1).toUpperCase() }}
            </el-avatar>
            <div class="avatar-info">
              <p class="username">{{ current?.username }}</p>
              <p class="muted">{{ current?.email }}</p>
            </div>
          </div>

          <el-form label-position="top" class="form">
            <el-form-item label="昵称">
              <el-input v-model="profile.nickname" maxlength="32" show-word-limit placeholder="展示名称" />
            </el-form-item>
            <el-form-item label="头像地址">
              <el-input v-model="profile.avatar_url" placeholder="https://example.com/avatar.png" />
            </el-form-item>
            <el-form-item label="默认可见性">
              <el-radio-group v-model="profile.default_visibility">
                <el-radio-button value="public">公开</el-radio-button>
                <el-radio-button value="private">私有</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="storages.length" label="默认存储策略">
              <el-select v-model="profile.storage_config_id" clearable placeholder="跟随系统默认" class="w-full">
                <el-option v-for="item in storages" :key="item.id" :label="item.name" :value="item.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="引用统计分析">
              <el-switch v-model="profile.referer_stat_enabled" />
              <span class="muted inline">开启后会记录图片被引用的来源域名</span>
            </el-form-item>
            <el-button type="primary" :loading="savingProfile" @click="saveProfile">保存资料</el-button>
          </el-form>
        </div>
      </el-tab-pane>

      <el-tab-pane name="security">
        <template #label>
          <span class="tab-label"><el-icon><Lock /></el-icon>安全设置</span>
        </template>

        <div class="panel">
          <h3>修改密码</h3>
          <el-form label-position="top" class="form narrow">
            <el-form-item label="当前密码" required>
              <el-input v-model="passwordForm.old_password" type="password" show-password autocomplete="current-password" />
            </el-form-item>
            <el-form-item label="新密码" required>
              <el-input v-model="passwordForm.new_password" type="password" show-password autocomplete="new-password" />
            </el-form-item>
            <el-form-item label="确认新密码" required>
              <el-input v-model="passwordForm.confirm" type="password" show-password autocomplete="new-password" />
            </el-form-item>
            <el-form-item v-if="setting.config.captcha_enabled" label="验证码">
              <CaptchaField
                v-model="passwordForm.captcha_code"
                v-model:id="passwordForm.captcha_id"
                :enabled="setting.config.captcha_enabled"
              />
            </el-form-item>
            <el-button type="primary" :loading="savingPassword" @click="changePassword">更新密码</el-button>
          </el-form>
        </div>
      </el-tab-pane>

      <el-tab-pane name="account">
        <template #label>
          <span class="tab-label"><el-icon><Bell /></el-icon>账号信息</span>
        </template>

        <div class="panel">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="用户名">{{ current?.username }}</el-descriptions-item>
            <el-descriptions-item label="角色">{{ roleText }}</el-descriptions-item>
            <el-descriptions-item label="邮箱">
              {{ current?.email }}
              <el-tag v-if="current?.email_verified" type="success" size="small">已验证</el-tag>
              <el-tag v-else type="warning" size="small">未验证</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="注册时间">{{ formatDate(current?.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="存储用量">
              {{ formatBytes(current?.used_bytes) }} / {{ formatBytes(current?.quota_bytes) }}
            </el-descriptions-item>
            <el-descriptions-item label="单张上限">{{ formatBytes(current?.max_file_size) }}</el-descriptions-item>
            <el-descriptions-item label="每日上传上限">{{ current?.daily_limit || '不限' }}</el-descriptions-item>
            <el-descriptions-item label="最近登录">
              {{ current?.last_login_at ? formatDate(current.last_login_at) : '—' }}
            </el-descriptions-item>
          </el-descriptions>

          <div class="quota">
            <div class="quota-label">
              <span>配额使用</span>
              <span>{{ quotaPercent }}%</span>
            </div>
            <el-progress :percentage="quotaPercent" :stroke-width="8" :show-text="false" />
          </div>

          <div v-if="!current?.email_verified" class="verify">
            <span>邮箱尚未验证，部分功能可能受限。</span>
            <el-button size="small" :loading="sendingVerify" @click="resendVerify">重新发送验证邮件</el-button>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped lang="scss">
.settings-tabs {
  max-width: 760px;
}

.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
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

.avatar-row {
  display: flex;
  align-items: center;
  gap: var(--ph-space-4);
  margin-bottom: var(--ph-space-5);
}

.avatar-info {
  .username {
    font-size: 16px;
    font-weight: 600;
  }
}

.muted {
  font-size: 12px;
  color: var(--ph-text-secondary);

  &.inline {
    margin-left: var(--ph-space-2);
  }
}

.form {
  max-width: 420px;
}

.narrow {
  max-width: 360px;
}

.quota {
  margin-top: var(--ph-space-4);
}

.quota-label {
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--ph-space-2);
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.verify {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ph-space-3);
  margin-top: var(--ph-space-4);
  padding: var(--ph-space-3);
  font-size: 12px;
  color: var(--ph-warning);
  background: rgb(255 125 0 / 8%);
  border-radius: var(--ph-radius-md);
}

@media (max-width: 767.98px) {
  .panel {
    padding: var(--ph-space-4);
  }
}
</style>
