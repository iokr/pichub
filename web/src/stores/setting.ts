import { ref } from 'vue'
import { defineStore } from 'pinia'

import { siteApi } from '@/api/site'
import type { SiteConfig } from '@/types/api'

const DEFAULT_CONFIG: SiteConfig = {
  name: 'PicHub',
  description: '简洁高效的图床系统',
  logo_url: '',
  icp: '',
  site_url: '',
  register_enabled: true,
  register_mode: 'open',
  email_verify_level: 'off',
  captcha_enabled: false,
  default_visibility: 'public',
  max_file_size: 10 * 1024 * 1024,
  allowed_extensions: ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg', 'avif'],
  thumbnail_enabled: true,
  thumbnail_sizes: ['small', 'medium', 'large'],
  s3_gateway_enabled: false,
  s3_gateway_prefix: '',
}

export const useSettingStore = defineStore('setting', () => {
  const config = ref<SiteConfig>({ ...DEFAULT_CONFIG })
  const loaded = ref(false)
  const loading = ref(false)

  async function load(force = false): Promise<SiteConfig> {
    if (loaded.value && !force) {
      return config.value
    }
    if (loading.value) {
      return config.value
    }
    loading.value = true
    try {
      const data = await siteApi.config()
      if (data) {
        config.value = { ...DEFAULT_CONFIG, ...data }
      }
      loaded.value = true
    } catch {
      /* 站点配置获取失败时使用默认值，不阻塞页面 */
    } finally {
      loading.value = false
    }
    return config.value
  }

  function setTitle(title?: string): void {
    document.title = title ? `${title} - ${config.value.name}` : config.value.name
  }

  return { config, loaded, loading, load, setTitle }
})
