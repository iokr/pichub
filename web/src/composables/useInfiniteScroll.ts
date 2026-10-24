import { onBeforeUnmount, watch, type Ref } from 'vue'

export interface InfiniteScrollOptions {
  rootMargin?: string
  disabled?: Ref<boolean> | (() => boolean)
}

/** 观察哨兵元素，进入视口时触发加载更多。 */
export function useInfiniteScroll(
  target: Ref<HTMLElement | null>,
  onLoad: () => void,
  options: InfiniteScrollOptions = {},
): void {
  let observer: IntersectionObserver | null = null

  function isDisabled(): boolean {
    const disabled = options.disabled
    if (!disabled) {
      return false
    }
    return typeof disabled === 'function' ? disabled() : disabled.value
  }

  function disconnect(): void {
    observer?.disconnect()
    observer = null
  }

  watch(
    target,
    (element) => {
      disconnect()
      if (!element || typeof IntersectionObserver === 'undefined') {
        return
      }
      observer = new IntersectionObserver(
        (entries) => {
          if (entries.some((entry) => entry.isIntersecting) && !isDisabled()) {
            onLoad()
          }
        },
        { rootMargin: options.rootMargin ?? '240px' },
      )
      observer.observe(element)
    },
    { immediate: true },
  )

  onBeforeUnmount(disconnect)
}
