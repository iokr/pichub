import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'

import { imageApi, type ImageBatchAction } from '@/api/image'
import type { Image, ImageListQuery, ImageUpdatePayload } from '@/types/api'

export const useImageStore = defineStore('image', () => {
  const items = ref<Image[]>([])
  const total = ref(0)
  const loading = ref(false)
  const finished = ref(false)
  const query = reactive<ImageListQuery>({ page: 1, size: 24 })
  const selection = ref<number[]>([])

  const selectedImages = computed(() => items.value.filter((item) => selection.value.includes(item.id)))
  const hasSelection = computed(() => selection.value.length > 0)
  const isEmpty = computed(() => !loading.value && items.value.length === 0)

  async function fetchPage(reset: boolean): Promise<void> {
    if (loading.value) {
      return
    }
    loading.value = true
    try {
      const params: ImageListQuery = { ...query }
      if (reset) {
        params.page = 1
      }
      const page = await imageApi.list(params)
      const list = page.list ?? []
      if (reset) {
        items.value = list
      } else {
        const existing = new Set(items.value.map((item) => item.id))
        items.value = [...items.value, ...list.filter((item) => !existing.has(item.id))]
      }
      total.value = page.total ?? items.value.length
      const currentPage = params.page ?? 1
      const size = params.size ?? 24
      finished.value = items.value.length >= total.value || list.length < size || (params.cursor ?? 0) > 0
      query.page = currentPage + 1
    } finally {
      loading.value = false
    }
  }

  /** 重新加载第一页，保留当前筛选条件。 */
  async function reload(): Promise<void> {
    query.page = 1
    finished.value = false
    await fetchPage(true)
  }

  /** 加载下一页，用于无限滚动。 */
  async function loadMore(): Promise<void> {
    if (finished.value || loading.value) {
      return
    }
    await fetchPage(false)
  }

  /** 修改筛选条件并重新加载。 */
  async function applyFilter(patch: Partial<ImageListQuery>): Promise<void> {
    Object.assign(query, patch)
    await reload()
  }

  function resetQuery(): void {
    for (const key of Object.keys(query) as (keyof ImageListQuery)[]) {
      if (key === 'size') {
        continue
      }
      delete query[key]
    }
    query.page = 1
    query.size = 24
  }

  function toggleSelect(id: number): void {
    const index = selection.value.indexOf(id)
    if (index >= 0) {
      selection.value.splice(index, 1)
    } else {
      selection.value.push(id)
    }
  }

  function selectAll(): void {
    selection.value = items.value.map((item) => item.id)
  }

  function clearSelection(): void {
    selection.value = []
  }

  function patchImage(image: Image): void {
    const index = items.value.findIndex((item) => item.id === image.id)
    if (index >= 0) {
      items.value[index] = image
    }
  }

  function removeLocal(ids: number[]): void {
    const removing = new Set(ids)
    items.value = items.value.filter((item) => !removing.has(item.id))
    total.value = Math.max(0, total.value - removing.size)
    selection.value = selection.value.filter((id) => !removing.has(id))
  }

  async function update(id: number, payload: ImageUpdatePayload): Promise<Image> {
    const image = await imageApi.update(id, payload)
    patchImage(image)
    return image
  }

  async function batch(action: ImageBatchAction, albumId?: number): Promise<number> {
    if (!selection.value.length) {
      return 0
    }
    const result = await imageApi.batch([...selection.value], action, albumId)
    if (action === 'delete' || action === 'move') {
      removeLocal(selection.value)
    } else if (action === 'public' || action === 'private') {
      const target = action === 'public'
      items.value = items.value.map((item) =>
        selection.value.includes(item.id) ? { ...item, is_public: target } : item,
      )
    }
    if (action !== 'move') {
      clearSelection()
    }
    return result?.affected ?? 0
  }

  async function remove(id: number): Promise<void> {
    await imageApi.remove(id)
    removeLocal([id])
  }

  return {
    items,
    total,
    loading,
    finished,
    query,
    selection,
    selectedImages,
    hasSelection,
    isEmpty,
    reload,
    loadMore,
    applyFilter,
    resetQuery,
    toggleSelect,
    selectAll,
    clearSelection,
    patchImage,
    removeLocal,
    update,
    batch,
    remove,
  }
})
