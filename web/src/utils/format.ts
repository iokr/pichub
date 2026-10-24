/** 把字节数格式化为人类可读的体积。 */
export function formatBytes(bytes?: number | null, digits = 1): string {
  if (!bytes || bytes <= 0) {
    return '0 B'
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let value = bytes
  let index = 0
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index += 1
  }
  return `${index === 0 ? Math.round(value) : value.toFixed(digits)} ${units[index]}`
}

/** 千分位数字。 */
export function formatNumber(value?: number | null): string {
  if (value === undefined || value === null || Number.isNaN(value)) {
    return '0'
  }
  return value.toLocaleString('zh-CN')
}

/** 把体积转换为 MB 数值，用于表单输入。 */
export function bytesToMB(bytes?: number | null): number {
  if (!bytes) {
    return 0
  }
  return Math.round((bytes / (1024 * 1024)) * 100) / 100
}

export function mbToBytes(mb?: number | null): number {
  if (!mb) {
    return 0
  }
  return Math.round(mb * 1024 * 1024)
}

function toDate(input?: string | number | Date | null): Date | null {
  if (input === undefined || input === null || input === '') {
    return null
  }
  const date = input instanceof Date ? input : new Date(input)
  return Number.isNaN(date.getTime()) ? null : date
}

const pad = (value: number): string => String(value).padStart(2, '0')

/** 格式化为 YYYY-MM-DD 或 YYYY-MM-DD HH:mm。 */
export function formatDate(input?: string | number | Date | null, withTime = true): string {
  const date = toDate(input)
  if (!date) {
    return '-'
  }
  const day = `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
  return withTime ? `${day} ${pad(date.getHours())}:${pad(date.getMinutes())}` : day
}

/** 相对时间，例如“3 分钟前”。 */
export function fromNow(input?: string | number | Date | null): string {
  const date = toDate(input)
  if (!date) {
    return '-'
  }
  const diff = Date.now() - date.getTime()
  const abs = Math.abs(diff)
  const minute = 60_000
  const hour = 60 * minute
  const day = 24 * hour
  if (abs < minute) {
    return '刚刚'
  }
  if (abs < hour) {
    return `${Math.floor(abs / minute)} 分钟前`
  }
  if (abs < day) {
    return `${Math.floor(abs / hour)} 小时前`
  }
  if (abs < 30 * day) {
    return `${Math.floor(abs / day)} 天前`
  }
  return formatDate(date, false)
}

/** 截断中间部分，保留头尾。 */
export function ellipsisMiddle(text: string, head = 8, tail = 6): string {
  if (!text || text.length <= head + tail + 3) {
    return text
  }
  return `${text.slice(0, head)}...${text.slice(-tail)}`
}

export function percent(part?: number | null, total?: number | null): number {
  if (!part || !total || total <= 0) {
    return 0
  }
  return Math.min(100, Math.round((part / total) * 100))
}

/** 解析后端返回的日期为本地时间字符串，用于日期选择器。 */
export function toLocalISO(input?: string | number | Date | null): string {
  const date = toDate(input)
  if (!date) {
    return ''
  }
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(
    date.getMinutes(),
  )}`
}
