import { ElMessage } from 'element-plus'

import { copyText } from '@/utils/download'

/** 复制并统一提示结果。 */
export function useCopy() {
  async function copy(text: string, message = '已复制到剪贴板'): Promise<boolean> {
    const ok = await copyText(text)
    if (ok) {
      ElMessage.success(message)
    } else {
      ElMessage.error('复制失败，请手动选择复制')
    }
    return ok
  }

  return { copy }
}
