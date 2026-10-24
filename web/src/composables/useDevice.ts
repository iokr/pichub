import { computed, onScopeDispose, ref } from 'vue'

const width = ref(typeof window === 'undefined' ? 1280 : window.innerWidth)
let listeners = 0

function update(): void {
  width.value = window.innerWidth
}

/** 基于窗口宽度的响应式设备判断，全局共享同一个 resize 监听。 */
export function useDevice() {
  if (typeof window !== 'undefined') {
    if (listeners === 0) {
      window.addEventListener('resize', update, { passive: true })
      window.addEventListener('orientationchange', update, { passive: true })
    }
    listeners += 1
    onScopeDispose(() => {
      listeners -= 1
      if (listeners === 0) {
        window.removeEventListener('resize', update)
        window.removeEventListener('orientationchange', update)
      }
    })
  }

  const isMobile = computed(() => width.value < 768)
  const isTablet = computed(() => width.value >= 768 && width.value < 1200)
  const isDesktop = computed(() => width.value >= 1200)

  return { width, isMobile, isTablet, isDesktop }
}
