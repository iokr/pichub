import { computed, ref, watchEffect } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto'

const STORAGE_KEY = 'pichub.theme'

function readMode(): ThemeMode {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (raw === 'light' || raw === 'dark' || raw === 'auto') {
      return raw
    }
  } catch {
    /* 忽略存储异常 */
  }
  return 'auto'
}

const mode = ref<ThemeMode>(readMode())
const systemDark = ref(false)

let initialized = false

function init(): void {
  if (initialized || typeof window === 'undefined') {
    return
  }
  initialized = true
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  systemDark.value = media.matches
  media.addEventListener('change', (event) => {
    systemDark.value = event.matches
  })
  watchEffect(() => {
    const dark = mode.value === 'dark' || (mode.value === 'auto' && systemDark.value)
    document.documentElement.classList.toggle('dark', dark)
    document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
  })
}

init()

/** 亮色 / 暗色 / 跟随系统。 */
export function useTheme() {
  const isDark = computed(() => mode.value === 'dark' || (mode.value === 'auto' && systemDark.value))

  function setMode(next: ThemeMode): void {
    mode.value = next
    try {
      window.localStorage.setItem(STORAGE_KEY, next)
    } catch {
      /* 忽略存储异常 */
    }
  }

  function toggle(): void {
    setMode(isDark.value ? 'light' : 'dark')
  }

  return { mode, isDark, setMode, toggle }
}
