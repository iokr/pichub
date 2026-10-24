<script setup lang="ts">
import { useSettingStore } from '@/stores/setting'

const setting = useSettingStore()
</script>

<template>
  <div class="auth-shell">
    <aside class="brand-panel">
      <div class="brand-inner">
        <router-link to="/" class="brand">
          <span class="brand-mark">P</span>
          <span class="brand-name">{{ setting.config.name }}</span>
        </router-link>
        <h1 class="slogan">{{ setting.config.description || '简洁高效的图床系统' }}</h1>
        <ul class="features">
          <li>拖拽即传，秒传去重</li>
          <li>多存储后端，自由切换</li>
          <li>引用统计与防盗链保护</li>
        </ul>
      </div>
    </aside>

    <main class="form-panel">
      <div class="form-inner">
        <slot />
      </div>
    </main>
  </div>
</template>

<style scoped lang="scss">
.auth-shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  min-height: 100vh;
  background: var(--ph-fill-card);
}

.brand-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--ph-space-12);
  background: var(--ph-gradient-brand);
  color: #fff;
}

.brand-inner {
  max-width: 420px;
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: var(--ph-space-3);
  color: #fff;
  font-size: 20px;
  font-weight: 600;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: var(--ph-radius-lg);
  background: rgba(255, 255, 255, 0.2);
  font-size: 20px;
  font-weight: 700;
}

.slogan {
  margin: var(--ph-space-10) 0 var(--ph-space-6);
  color: #fff;
  font-size: 32px;
  line-height: 1.3;
  font-weight: 600;
}

.features {
  display: flex;
  flex-direction: column;
  gap: var(--ph-space-3);
  margin: 0;
  padding: 0;
  list-style: none;

  li {
    position: relative;
    padding-left: var(--ph-space-5);
    font-size: 15px;
    opacity: 0.9;

    &::before {
      content: '';
      position: absolute;
      top: 8px;
      left: 0;
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: rgba(255, 255, 255, 0.8);
    }
  }
}

.form-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--ph-space-8) var(--ph-space-6);
  background: var(--ph-fill-card);
}

.form-inner {
  width: 100%;
  max-width: 380px;
}

@media (max-width: 991px) {
  .auth-shell {
    grid-template-columns: 1fr;
  }

  .brand-panel {
    padding: var(--ph-space-8) var(--ph-space-6) var(--ph-space-6);
  }

  .slogan {
    margin: var(--ph-space-4) 0 0;
    font-size: 22px;
  }

  .features {
    display: none;
  }

  .form-panel {
    align-items: flex-start;
    padding: var(--ph-space-8) var(--ph-space-6);
  }
}
</style>
