<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { Menu as MenuIcon, Search, Upload } from '@element-plus/icons-vue'

import ThemeToggle from '@/components/common/ThemeToggle.vue'
import { useDevice } from '@/composables/useDevice'
import { useSettingStore } from '@/stores/setting'
import { useUserStore } from '@/stores/user'
import { adminNav, mainNav, matchActivePath } from './menu'

const router = useRouter()
const setting = useSettingStore()
const user = useUserStore()
const { isMobile } = useDevice()

const drawer = ref(false)
const keyword = ref('')

const activeIndex = computed(() => matchActivePath(router.currentRoute.value.path))
const initial = computed(() => (user.displayName || 'P').slice(0, 1).toUpperCase())

function search(): void {
  const value = keyword.value.trim()
  void router.push({ path: '/gallery', query: value ? { keyword: value } : {} })
}

async function handleCommand(command: string): Promise<void> {
  if (command === 'settings') {
    void router.push('/settings')
    return
  }
  if (command === 'admin') {
    void router.push('/admin')
    return
  }
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
        type: 'warning',
        confirmButtonText: '退出',
        cancelButtonText: '取消',
      })
    } catch {
      return
    }
    await user.logout()
    void router.replace('/login')
  }
}
</script>

<template>
  <header class="app-topbar">
    <div class="topbar-left">
      <el-button v-if="isMobile" text class="icon-button" aria-label="打开导航" @click="drawer = true">
        <el-icon :size="20"><MenuIcon /></el-icon>
      </el-button>
      <router-link to="/" class="brand">
        <img v-if="setting.config.logo_url" :src="setting.config.logo_url" alt="logo" class="brand-logo" />
        <span v-else class="brand-mark">P</span>
        <span class="brand-name">{{ setting.config.name }}</span>
      </router-link>
    </div>

    <div v-if="!isMobile" class="topbar-search">
      <el-input
        v-model="keyword"
        placeholder="搜索图片名称"
        :prefix-icon="Search"
        clearable
        @keyup.enter="search"
      />
    </div>

    <div class="topbar-actions">
      <el-button v-if="!isMobile" type="primary" :icon="Upload" @click="router.push('/')">上传</el-button>
      <ThemeToggle />
      <el-dropdown trigger="click" @command="handleCommand">
        <button class="user-trigger" type="button" aria-label="用户菜单">
          <el-avatar :size="32" :src="user.user?.avatar_url || undefined">{{ initial }}</el-avatar>
          <span v-if="!isMobile" class="user-name">{{ user.displayName || '未登录' }}</span>
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="settings">个人设置</el-dropdown-item>
            <el-dropdown-item v-if="user.isAdmin" command="admin">管理后台</el-dropdown-item>
            <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </header>

  <el-drawer v-model="drawer" direction="ltr" size="260px" :with-header="false" class="nav-drawer">
    <div class="drawer-brand">
      <span class="brand-mark">P</span>
      <span class="brand-name">{{ setting.config.name }}</span>
    </div>
    <el-menu :default-active="activeIndex" router @select="drawer = false">
      <el-menu-item v-for="item in mainNav" :key="item.path" :index="item.path">
        <el-icon><component :is="item.icon" /></el-icon>
        <span>{{ item.label }}</span>
      </el-menu-item>
      <template v-if="user.isAdmin">
        <el-divider />
        <el-menu-item v-for="item in adminNav" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </el-menu-item>
      </template>
    </el-menu>
  </el-drawer>
</template>

<style scoped lang="scss">
.app-topbar {
  position: sticky;
  top: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  gap: var(--ph-space-4);
  height: var(--ph-topbar-height);
  padding: 0 var(--ph-space-6);
  border-bottom: 1px solid var(--ph-border-light);
  background: var(--ph-fill-card);
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
  min-width: 180px;
}

.brand {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
  color: var(--ph-text-primary);
  font-weight: 600;
}

.brand-logo {
  width: 30px;
  height: 30px;
  border-radius: var(--ph-radius-md);
  object-fit: cover;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: var(--ph-radius-md);
  background: var(--ph-gradient-brand);
  color: #fff;
  font-weight: 700;
}

.brand-name {
  font-size: 16px;
}

.topbar-search {
  flex: 1;
  max-width: 420px;

  :deep(.el-input__wrapper) {
    background: var(--ph-fill);
    box-shadow: none;
  }
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
  margin-left: auto;
}

.icon-button {
  color: var(--ph-text-regular);
}

.user-trigger {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
  padding: 4px 6px;
  border: none;
  border-radius: var(--ph-radius-full);
  background: transparent;
  color: var(--ph-text-primary);
  cursor: pointer;
  transition: background-color 150ms ease-out;

  &:hover {
    background: var(--ph-fill-hover);
  }
}

.user-name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}

.drawer-brand {
  display: flex;
  align-items: center;
  gap: var(--ph-space-2);
  padding: var(--ph-space-2) var(--ph-space-3) var(--ph-space-4);
  font-weight: 600;
  color: var(--ph-text-primary);
}

@media (max-width: 767.98px) {
  .app-topbar {
    height: var(--ph-tabbar-height);
    padding: 0 var(--ph-space-4);
    gap: var(--ph-space-2);
  }

  .topbar-left {
    min-width: 0;
  }

  .brand-name {
    font-size: 15px;
  }
}
</style>
