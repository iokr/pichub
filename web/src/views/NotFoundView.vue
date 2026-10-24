<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="not-found">
    <div class="code">404</div>
    <h1>页面不存在</h1>
    <p class="hint">你访问的页面可能已被移除或地址有误。</p>
    <div class="actions">
      <el-button @click="router.back()">返回上一页</el-button>
      <el-button type="primary" @click="router.push('/')">回到首页</el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.not-found {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: var(--ph-space-8);
  text-align: center;
}

.code {
  font-size: 96px;
  font-weight: 700;
  line-height: 1;
  background: var(--ph-gradient-brand);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

h1 {
  margin-top: var(--ph-space-4);
  font-size: 24px;
}

.hint {
  margin-top: var(--ph-space-2);
  color: var(--ph-text-secondary);
}

.actions {
  display: flex;
  gap: var(--ph-space-3);
  margin-top: var(--ph-space-6);
}
</style>
