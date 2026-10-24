import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'

import { uploadApi } from '@/api/upload'
import type { UploadOptions, UploadResult } from '@/types/api'

export type UploadStatus = 'waiting' | 'hashing' | 'uploading' | 'success' | 'error' | 'canceled'

export interface UploadTask {
  id: string
  file: File
  name: string
  size: number
  progress: number
  status: UploadStatus
  error: string
  dedup: boolean
  preview: string
  options: UploadOptions
  controller: AbortController | null
  result?: UploadResult
  createdAt: number
}

const MAX_CONCURRENT = 3
/** 超过该体积跳过秒传检测，避免前端哈希耗时过长。 */
const HASH_LIMIT = 64 * 1024 * 1024
const SETTINGS_KEY = 'pichub.upload_options'

function loadSavedOptions(): UploadOptions {
  try {
    const raw = window.localStorage.getItem(SETTINGS_KEY)
    if (raw) {
      return JSON.parse(raw) as UploadOptions
    }
  } catch {
    /* 忽略损坏的本地配置 */
  }
  return {}
}

function persistOptions(options: UploadOptions): void {
  try {
    window.localStorage.setItem(SETTINGS_KEY, JSON.stringify(options))
  } catch {
    /* 忽略存储失败 */
  }
}

async function sha256(file: File): Promise<string> {
  if (!window.crypto?.subtle) {
    return ''
  }
  try {
    const buffer = await file.arrayBuffer()
    const digest = await window.crypto.subtle.digest('SHA-256', buffer)
    return Array.from(new Uint8Array(digest))
      .map((byte) => byte.toString(16).padStart(2, '0'))
      .join('')
  } catch {
    return ''
  }
}

function extensionOf(name: string): string {
  const index = name.lastIndexOf('.')
  return index >= 0 ? name.slice(index + 1).toLowerCase() : ''
}

export const useUploadStore = defineStore('upload', () => {
  const tasks = ref<UploadTask[]>([])
  const options = reactive<UploadOptions>(loadSavedOptions())
  const completedTick = ref(0)
  const drawerOpen = ref(false)

  let running = 0
  let seq = 0

  const activeTasks = computed(() =>
    tasks.value.filter((task) => task.status === 'waiting' || task.status === 'hashing' || task.status === 'uploading'),
  )
  const activeCount = computed(() => activeTasks.value.length)
  const successCount = computed(() => tasks.value.filter((task) => task.status === 'success').length)
  const failedCount = computed(() => tasks.value.filter((task) => task.status === 'error').length)
  const overallProgress = computed(() => {
    const active = activeTasks.value
    if (!active.length) {
      return 100
    }
    return Math.round(active.reduce((sum, task) => sum + task.progress, 0) / active.length)
  })

  function saveOptions(patch: Partial<UploadOptions> = {}): void {
    Object.assign(options, patch)
    persistOptions({ ...options })
  }

  function enqueue(files: File[] | FileList, extra: UploadOptions = {}): UploadTask[] {
    const list = Array.from(files)
    if (!list.length) {
      return []
    }
    const created: UploadTask[] = []
    for (const file of list) {
      seq += 1
      const task: UploadTask = {
        id: `task-${Date.now()}-${seq}`,
        file,
        name: file.name,
        size: file.size,
        progress: 0,
        status: 'waiting',
        error: '',
        dedup: false,
        preview: URL.createObjectURL(file),
        options: { ...options, ...extra },
        controller: null,
        createdAt: Date.now(),
      }
      tasks.value.push(task)
      created.push(task)
    }
    drawerOpen.value = true
    pump()
    return created
  }

  function pump(): void {
    if (running >= MAX_CONCURRENT) {
      return
    }
    const next = tasks.value.find((task) => task.status === 'waiting')
    if (!next) {
      return
    }
    running += 1
    void run(next)
    pump()
  }

  async function run(task: UploadTask): Promise<void> {
    task.status = 'hashing'
    task.error = ''
    task.progress = 0
    task.dedup = false
    task.controller = new AbortController()
    try {
      let hash = ''
      if (task.file.size > 0 && task.file.size <= HASH_LIMIT) {
        hash = await sha256(task.file)
      }
      if (hash) {
        try {
          const checked = await uploadApi.check({
            hash,
            size: task.file.size,
            name: task.file.name,
            ext: extensionOf(task.file.name),
          })
          if (checked.exists && checked.image) {
            task.result = checked.image
            task.dedup = true
            task.progress = 100
            task.status = 'success'
            completedTick.value += 1
            return
          }
        } catch {
          /* 秒传检测失败时继续走普通上传 */
        }
      }
      task.status = 'uploading'
      const result = await uploadApi.upload(
        task.file,
        task.options,
        (percent) => {
          task.progress = percent
        },
        task.controller.signal,
      )
      task.result = result
      task.progress = 100
      task.status = 'success'
      completedTick.value += 1
    } catch (error) {
      if (task.controller?.signal.aborted) {
        task.status = 'canceled'
      } else {
        task.status = 'error'
        task.error = error instanceof Error ? error.message : '上传失败'
      }
    } finally {
      task.controller = null
      running = Math.max(0, running - 1)
      pump()
    }
  }

  function cancel(id: string): void {
    const task = tasks.value.find((item) => item.id === id)
    if (!task) {
      return
    }
    if (task.status === 'waiting') {
      task.status = 'canceled'
      return
    }
    task.controller?.abort()
  }

  function retry(id: string): void {
    const task = tasks.value.find((item) => item.id === id)
    if (!task || task.status === 'uploading' || task.status === 'hashing') {
      return
    }
    task.status = 'waiting'
    task.progress = 0
    task.error = ''
    pump()
  }

  function clearFinished(): void {
    tasks.value = tasks.value.filter((task) => {
      const done = task.status === 'success' || task.status === 'error' || task.status === 'canceled'
      if (done) {
        URL.revokeObjectURL(task.preview)
      }
      return !done
    })
  }

  function removeTask(id: string): void {
    const index = tasks.value.findIndex((task) => task.id === id)
    if (index < 0) {
      return
    }
    const task = tasks.value[index]
    task.controller?.abort()
    URL.revokeObjectURL(task.preview)
    tasks.value.splice(index, 1)
  }

  function toggleDrawer(open?: boolean): void {
    drawerOpen.value = open ?? !drawerOpen.value
  }

  return {
    tasks,
    options,
    drawerOpen,
    activeTasks,
    activeCount,
    successCount,
    failedCount,
    overallProgress,
    completedTick,
    enqueue,
    saveOptions,
    cancel,
    retry,
    clearFinished,
    removeTask,
    toggleDrawer,
  }
})
