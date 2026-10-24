/** 复制文本到剪贴板，返回是否成功。 */
export async function copyText(text: string): Promise<boolean> {
  if (!text) {
    return false
  }
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    /* 非安全上下文或权限被拒绝时走兼容方案 */
  }
  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.top = '-9999px'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(textarea)
    return ok
  } catch {
    return false
  }
}

function triggerDownload(url: string, filename?: string): void {
  const link = document.createElement('a')
  link.href = url
  if (filename) {
    link.download = filename
  }
  link.rel = 'noopener'
  link.style.display = 'none'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

/** 直接下载外链资源。跨域时可加 download 参数由后端返回附件。 */
export function downloadURL(url: string, filename?: string): void {
  triggerDownload(url, filename)
}

/** 生成并下载文本文件。 */
export function downloadText(content: string, filename: string, type = 'text/plain;charset=utf-8'): void {
  const blob = new Blob([content], { type })
  const url = URL.createObjectURL(blob)
  triggerDownload(url, filename)
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

export function buildMarkdown(url: string, alt = ''): string {
  return `![${alt}](${url})`
}

export function buildHTML(url: string, alt = ''): string {
  return `<img src="${url}" alt="${alt}" />`
}

export function buildBBCode(url: string): string {
  return `[img]${url}[/img]`
}
