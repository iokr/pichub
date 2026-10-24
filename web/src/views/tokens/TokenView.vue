<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Key, Plus } from '@element-plus/icons-vue'

import { s3CredentialApi } from '@/api/s3'
import { tokenApi, type TokenCreatePayload } from '@/api/token'
import PageHeader from '@/components/common/PageHeader.vue'
import CopyButton from '@/components/common/CopyButton.vue'
import { useSettingStore } from '@/stores/setting'
import type { S3Credential, S3CredentialCreateResult, Token, TokenCreateResult } from '@/types/api'
import { formatDate, fromNow } from '@/utils/format'

const setting = useSettingStore()

const tokens = ref<Token[]>([])
const credentials = ref<S3Credential[]>([])
const loading = ref(false)
const activeTab = ref<'tokens' | 's3'>('tokens')

const tokenDialog = ref(false)
const tokenForm = reactive<TokenCreatePayload>({ name: '', scopes: ['upload', 'read', 'write', 'delete', 'stats'], expires_in_days: 0 })
const createdToken = ref<TokenCreateResult | null>(null)
const creating = ref(false)

const s3Dialog = ref(false)
const s3Name = ref('')
const createdCred = ref<S3CredentialCreateResult | null>(null)
const creatingCred = ref(false)

const scopeOptions = [
  { value: 'upload', label: '上传', description: '通过 API 上传图片' },
  { value: 'read', label: '读取', description: '读取图片与相册信息' },
  { value: 'write', label: '修改', description: '修改图片、相册与标签' },
  { value: 'delete', label: '删除', description: '删除图片与回收站操作' },
  { value: 'stats', label: '统计', description: '读取用量与统计数据' },
]

const availableScopes = computed(() => scopeOptions.filter((item) => tokenForm.scopes.includes(item.value)))

async function load(): Promise<void> {
  loading.value = true
  try {
    const [tokenList, credList] = await Promise.all([
      tokenApi.list().catch(() => [] as Token[]),
      setting.config.s3_gateway_enabled ? s3CredentialApi.list().catch(() => [] as S3Credential[]) : Promise.resolve([] as S3Credential[]),
    ])
    tokens.value = tokenList
    credentials.value = credList
  } finally {
    loading.value = false
  }
}

function openTokenDialog(): void {
  createdToken.value = null
  tokenForm.name = ''
  tokenForm.scopes = [...scopeOptions.map((item) => item.value)]
  tokenForm.expires_in_days = 0
  tokenDialog.value = true
}

async function createToken(): Promise<void> {
  if (!tokenForm.name.trim()) {
    ElMessage.warning('请填写令牌名称')
    return
  }
  if (!tokenForm.scopes.length) {
    ElMessage.warning('至少选择一个权限范围')
    return
  }
  creating.value = true
  try {
    createdToken.value = await tokenApi.create({
      name: tokenForm.name.trim(),
      scopes: tokenForm.scopes,
      expires_in_days: tokenForm.expires_in_days || undefined,
    })
    ElMessage.success('令牌已创建')
    await load()
  } finally {
    creating.value = false
  }
}

async function toggleToken(token: Token): Promise<void> {
  const next = token.status === 1 ? 0 : 1
  await tokenApi.setStatus(token.id, next)
  token.status = next
  ElMessage.success(next === 1 ? '令牌已启用' : '令牌已禁用')
}

async function removeToken(token: Token): Promise<void> {
  try {
    await ElMessageBox.confirm(`删除后使用该令牌的程序将立即失效，确定删除「${token.name}」吗？`, '删除令牌', {
      type: 'warning',
    })
  } catch {
    return
  }
  await tokenApi.remove(token.id)
  tokens.value = tokens.value.filter((item) => item.id !== token.id)
  ElMessage.success('令牌已删除')
}

function openS3Dialog(): void {
  createdCred.value = null
  s3Name.value = ''
  s3Dialog.value = true
}

async function createCredential(): Promise<void> {
  if (!s3Name.value.trim()) {
    ElMessage.warning('请填写凭据名称')
    return
  }
  creatingCred.value = true
  try {
    createdCred.value = await s3CredentialApi.create(s3Name.value.trim())
    ElMessage.success('凭据已创建')
    credentials.value = await s3CredentialApi.list()
  } finally {
    creatingCred.value = false
  }
}

async function removeCredential(cred: S3Credential): Promise<void> {
  try {
    await ElMessageBox.confirm(`确定删除凭据「${cred.name}」吗？使用该凭据的客户端将无法访问。`, '删除凭据', {
      type: 'warning',
    })
  } catch {
    return
  }
  await s3CredentialApi.remove(cred.id)
  credentials.value = credentials.value.filter((item) => item.id !== cred.id)
  ElMessage.success('凭据已删除')
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="API 令牌" description="使用令牌通过 HTTP API 或 S3 协议管理你的图片">
      <template #actions>
        <el-button v-if="activeTab === 'tokens'" type="primary" :icon="Plus" @click="openTokenDialog">创建令牌</el-button>
        <el-button v-else type="primary" :icon="Plus" @click="openS3Dialog">创建凭据</el-button>
      </template>
    </PageHeader>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="API 令牌" name="tokens">
        <el-table :data="tokens" v-loading="loading" row-key="id">
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column label="前缀" width="140">
            <template #default="{ row }">
              <code class="mono">{{ row.prefix }}…</code>
            </template>
          </el-table-column>
          <el-table-column label="权限" min-width="200">
            <template #default="{ row }">
              <el-tag v-for="scope in row.scopes" :key="scope" size="small" class="scope-tag">{{ scope }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="调用次数" width="100" prop="call_count" />
          <el-table-column label="最后使用" width="140">
            <template #default="{ row }">{{ row.last_used_at ? fromNow(row.last_used_at) : '从未使用' }}</template>
          </el-table-column>
          <el-table-column label="过期时间" width="160">
            <template #default="{ row }">{{ row.expires_at ? formatDate(row.expires_at) : '永不过期' }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
                {{ row.status === 1 ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button text size="small" @click="toggleToken(row)">{{ row.status === 1 ? '禁用' : '启用' }}</el-button>
              <el-button text size="small" type="danger" @click="removeToken(row)">删除</el-button>
            </template>
          </el-table-column>
          <template #empty>
            <el-empty description="还没有 API 令牌" :image-size="72" />
          </template>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="S3 凭据" name="s3" :disabled="!setting.config.s3_gateway_enabled">
        <el-alert
          v-if="!setting.config.s3_gateway_enabled"
          title="S3 网关未启用"
          description="管理员开启 S3 兼容网关后即可在此创建访问凭据。"
          type="info"
          :closable="false"
        />
        <template v-else>
          <el-alert
            class="s3-hint"
            type="info"
            :closable="false"
            :title="`服务地址：${setting.config.site_url}${setting.config.s3_gateway_prefix || '/s3'}`"
            description="任何兼容 S3 协议的客户端均可使用这里的 AccessKey / SecretKey 上传图片。"
          />
          <el-table :data="credentials" v-loading="loading" row-key="id">
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column label="AccessKey" min-width="200">
              <template #default="{ row }">
                <code class="mono">{{ row.access_key }}</code>
                <CopyButton :text="row.access_key" />
              </template>
            </el-table-column>
            <el-table-column label="调用次数" width="100" prop="call_count" />
            <el-table-column label="最后使用" width="140">
              <template #default="{ row }">{{ row.last_used_at ? fromNow(row.last_used_at) : '从未使用' }}</template>
            </el-table-column>
            <el-table-column label="创建时间" width="170">
              <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="90" fixed="right">
              <template #default="{ row }">
                <el-button text size="small" type="danger" @click="removeCredential(row)">删除</el-button>
              </template>
            </el-table-column>
            <template #empty>
              <el-empty description="还没有 S3 凭据" :image-size="72" />
            </template>
          </el-table>
        </template>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="tokenDialog" title="创建 API 令牌" width="520px">
      <template v-if="!createdToken">
        <el-form label-position="top">
          <el-form-item label="令牌名称" required>
            <el-input v-model="tokenForm.name" placeholder="例如：PicGo / 博客同步" maxlength="64" show-word-limit />
          </el-form-item>
          <el-form-item label="权限范围" required>
            <el-checkbox-group v-model="tokenForm.scopes" class="scope-group">
              <el-checkbox v-for="scope in scopeOptions" :key="scope.value" :value="scope.value">
                {{ scope.label }}
                <span class="scope-desc">{{ scope.description }}</span>
              </el-checkbox>
            </el-checkbox-group>
          </el-form-item>
          <el-form-item label="有效期">
            <el-select v-model="tokenForm.expires_in_days" class="w-full">
              <el-option label="永不过期" :value="0" />
              <el-option label="30 天" :value="30" />
              <el-option label="90 天" :value="90" />
              <el-option label="180 天" :value="180" />
              <el-option label="365 天" :value="365" />
            </el-select>
          </el-form-item>
        </el-form>
        <p class="notice">令牌仅在创建时显示一次，请妥善保存。</p>
      </template>

      <div v-else class="created">
        <el-result icon="success" title="令牌创建成功" sub-title="请立即复制并妥善保存，关闭后将无法再次查看。">
          <template #extra>
            <div class="secret">
              <code class="mono">{{ createdToken.token }}</code>
              <CopyButton :text="createdToken.token" />
            </div>
            <p class="scope-summary">权限：{{ availableScopes.map((item) => item.label).join('、') }}</p>
          </template>
        </el-result>
      </div>

      <template #footer>
        <template v-if="!createdToken">
          <el-button @click="tokenDialog = false">取消</el-button>
          <el-button type="primary" :loading="creating" @click="createToken">创建</el-button>
        </template>
        <el-button v-else type="primary" @click="tokenDialog = false">我已保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="s3Dialog" title="创建 S3 凭据" width="520px">
      <template v-if="!createdCred">
        <el-form label-position="top">
          <el-form-item label="凭据名称" required>
            <el-input v-model="s3Name" placeholder="例如：PicGo S3" maxlength="64" show-word-limit />
          </el-form-item>
        </el-form>
        <p class="notice">
          服务地址：<code class="mono">{{ setting.config.site_url }}{{ setting.config.s3_gateway_prefix || '/s3' }}</code>，
          Region 可填写 <code class="mono">us-east-1</code>。
        </p>
      </template>

      <div v-else class="created">
        <el-result icon="success" title="凭据创建成功" sub-title="SecretKey 仅显示一次，请立即保存。">
          <template #extra>
            <div class="secret-grid">
              <span class="key">AccessKey</span>
              <div class="secret">
                <code class="mono">{{ createdCred.access_key }}</code>
                <CopyButton :text="createdCred.access_key" />
              </div>
              <span class="key">SecretKey</span>
              <div class="secret">
                <code class="mono">{{ createdCred.secret_key }}</code>
                <CopyButton :text="createdCred.secret_key" />
              </div>
            </div>
          </template>
        </el-result>
      </div>

      <template #footer>
        <template v-if="!createdCred">
          <el-button @click="s3Dialog = false">取消</el-button>
          <el-button type="primary" :loading="creatingCred" @click="createCredential">创建</el-button>
        </template>
        <el-button v-else type="primary" @click="s3Dialog = false">我已保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.mono {
  font-family: var(--ph-font-mono);
  font-size: 12px;
}

.scope-tag {
  margin-right: 4px;
}

.scope-group {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-1);
}

.scope-desc {
  margin-left: var(--ph-space-2);
  font-size: 12px;
  color: var(--ph-text-tertiary);
}

.s3-hint {
  margin-bottom: var(--ph-space-4);
}

.notice {
  margin-top: var(--ph-space-2);
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.secret {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
  padding: var(--ph-space-2) var(--ph-space-3);
  background: var(--ph-fill);
  border-radius: var(--ph-radius-md);
  word-break: break-all;
}

.secret-grid {
  display: grid;
  grid-template-columns: 90px 1fr;
  align-items: center;
  gap: var(--ph-space-2) var(--ph-space-3);
  text-align: left;

  .key {
    font-size: 12px;
    color: var(--ph-text-secondary);
  }
}

.scope-summary {
  margin-top: var(--ph-space-2);
  font-size: 12px;
  color: var(--ph-text-secondary);
}
</style>
