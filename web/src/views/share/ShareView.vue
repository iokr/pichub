<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { Download, Lock } from '@element-plus/icons-vue'

import { ApiError } from '@/api/client'
import { shareApi } from '@/api/share'
import { useSettingStore } from '@/stores/setting'
import type { ShareAccess } from '@/types/api'
import { formatDate } from '@/utils/format'

const route = useRoute()
const setting = useSettingStore()

const slug = computed(() => String(route.params.slug ?? ''))
const loading = ref(true)
const errorMessage = ref('')
const needPassword = ref(false)
const password = ref('')
const submitting = ref(false)
const access = ref<ShareAccess | null>(null)

const images = computed(() => access.value?.images ?? [])
const previewList = computed(() => images.value.map((image) => image.url))
const isAlbum = computed(() => access.value?.share.target_type === 'album')

async function load(pwd?: string): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    access.value = await shareApi.access(slug.value, pwd)
    needPassword.value = false
  } catch (error) {
    if (error instanceof ApiError && error.status === 403) {
      needPassword.value = true
      errorMessage.value = pwd ? error.message : ''
    } else {
      needPassword.value = false
      errorMessage.value = error instanceof Error ? error.message : '分享不存在或已失效'
    }
  } finally {
    loading.value = false
  }
}

async function submitPassword(): Promise<void> {
  if (!password.value.trim()) {
    return
  }
  submitting.value = true
  try {
    await load(password.value.trim())
  } finally {
    submitting.value = false
  }
}

function downloadImage(url: string): void {
  window.open(url, '_blank', 'noopener')
}

onMounted(() => load())
</script>

<template>
  <div class="share-page">
    <header class="share-header">
      <router-link to="/" class="brand">{{ setting.config.name || 'PicHub' }}</router-link>
      <span class="tagline">{{ isAlbum ? '相册分享' : '图片分享' }}</span>
    </header>

    <main class="share-main">
      <el-skeleton v-if="loading" animated :rows="6" class="skeleton" />

      <div v-else-if="needPassword" class="panel password-panel">
        <el-icon :size="40" class="lock"><Lock /></el-icon>
        <h2>需要访问密码</h2>
        <p class="hint">该分享已设置密码，请输入后查看内容。</p>
        <el-input
          v-model="password"
          type="password"
          show-password
          placeholder="请输入访问密码"
          class="password-input"
          @keyup.enter="submitPassword"
        />
        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
        <el-button type="primary" :loading="submitting" class="submit" @click="submitPassword">确认</el-button>
      </div>

      <div v-else-if="errorMessage" class="panel">
        <el-result icon="warning" title="无法访问" :sub-title="errorMessage">
          <template #extra>
            <el-button @click="load()">重试</el-button>
          </template>
        </el-result>
      </div>

      <template v-else-if="access">
        <div class="share-meta">
          <h1>{{ isAlbum ? access.album?.name || '相册分享' : images[0]?.name }}</h1>
          <p class="meta">
            <span>{{ images.length }} 张图片</span>
            <span v-if="access.share.expires_at">· {{ formatDate(access.share.expires_at) }} 过期</span>
            <span v-else>· 永久有效</span>
            <span v-if="access.share.has_password">· 密码保护</span>
          </p>
        </div>

        <div v-if="isAlbum" class="album-grid">
          <div v-for="(image, index) in images" :key="image.id" class="album-item">
            <el-image
              :src="image.thumb_url || image.url"
              :preview-src-list="previewList"
              :initial-index="index"
              fit="cover"
              loading="lazy"
              preview-teleported
              class="album-image"
            />
            <div class="item-overlay">
              <span class="name">{{ image.name }}</span>
              <el-button circle size="small" :icon="Download" @click.stop="downloadImage(image.url)" />
            </div>
          </div>
        </div>

        <div v-else-if="images[0]" class="single">
          <el-image
            :src="images[0].url"
            :preview-src-list="previewList"
            fit="contain"
            preview-teleported
            class="single-image"
          />
          <div class="single-actions">
            <span class="single-meta">{{ images[0].width }}×{{ images[0].height }}</span>
            <el-button :icon="Download" @click="downloadImage(images[0].url)">下载原图</el-button>
          </div>
        </div>

        <el-empty v-else description="分享内容为空" />
      </template>
    </main>

    <footer class="share-footer">
      <span>由 {{ setting.config.name || 'PicHub' }} 提供图床服务</span>
      <span v-if="setting.config.icp">{{ setting.config.icp }}</span>
    </footer>
  </div>
</template>

<style scoped lang="scss">
.share-page {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--ph-fill);
}

.share-header {
  display: flex;
  align-items: center;
  gap: var(--ph-space-3);
  height: var(--ph-topbar-height);
  padding: 0 var(--ph-space-6);
  background: var(--ph-fill-card);
  border-bottom: 1px solid var(--ph-border-light);
}

.brand {
  font-size: 18px;
  font-weight: 700;
  color: var(--ph-text-primary);
}

.tagline {
  padding: 2px 8px;
  font-size: 12px;
  color: var(--ph-text-secondary);
  background: var(--ph-fill);
  border-radius: 999px;
}

.share-main {
  flex: 1;
  width: 100%;
  max-width: 1100px;
  margin: 0 auto;
  padding: var(--ph-space-8) var(--ph-space-4);
}

.skeleton {
  max-width: 720px;
  margin: 0 auto;
}

.panel {
  max-width: 420px;
  margin: 0 auto;
  padding: var(--ph-space-6);
  text-align: center;
  background: var(--ph-fill-card);
  border: 1px solid var(--ph-border-light);
  border-radius: var(--ph-radius-lg);
}

.password-panel {
  .lock {
    color: var(--ph-primary);
  }

  h2 {
    margin-top: var(--ph-space-3);
    font-size: 20px;
  }
}

.hint {
  margin-top: var(--ph-space-1);
  font-size: 13px;
  color: var(--ph-text-secondary);
}

.password-input {
  margin-top: var(--ph-space-4);
}

.error {
  margin-top: var(--ph-space-2);
  font-size: 12px;
  color: var(--ph-danger);
}

.submit {
  width: 100%;
  margin-top: var(--ph-space-3);
}

.share-meta {
  margin-bottom: var(--ph-space-5);
  text-align: center;

  h1 {
    font-size: 24px;
    word-break: break-all;
  }
}

.meta {
  margin-top: var(--ph-space-1);
  font-size: 13px;
  color: var(--ph-text-secondary);

  span + span {
    margin-left: 4px;
  }
}

.album-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--ph-space-3);
}

.album-item {
  position: relative;
  overflow: hidden;
  aspect-ratio: 4 / 3;
  background: var(--ph-fill-card);
  border-radius: var(--ph-radius-md);
  cursor: zoom-in;

  &:hover .item-overlay {
    opacity: 1;
  }
}

.album-image {
  width: 100%;
  height: 100%;
}

.item-overlay {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ph-space-2);
  padding: var(--ph-space-2) var(--ph-space-3);
  color: #fff;
  background: linear-gradient(transparent, rgb(0 0 0 / 60%));
  opacity: 0;
  transition: opacity 0.2s ease;

  .name {
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.single {
  max-width: 900px;
  margin: 0 auto;
  text-align: center;
}

.single-image {
  max-height: 70vh;
  border-radius: var(--ph-radius-lg);
  background: var(--ph-fill-card);
}

.single-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--ph-space-3);
  margin-top: var(--ph-space-4);
}

.single-meta {
  font-size: 12px;
  color: var(--ph-text-secondary);
}

.share-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--ph-space-3);
  padding: var(--ph-space-5);
  font-size: 12px;
  color: var(--ph-text-tertiary);
}

@media (max-width: 767.98px) {
  .share-header {
    padding: 0 var(--ph-space-4);
  }

  .share-main {
    padding: var(--ph-space-4) var(--ph-space-3);
  }

  .album-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--ph-space-2);
  }

  .item-overlay {
    opacity: 1;
  }

  .share-meta h1 {
    font-size: 18px;
  }
}
</style>
