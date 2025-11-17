import { checkInitStatus } from '@/api/init'
import router from '@/router'

// 检查系统初始化状态
export const checkSystemInit = async () => {
  try {
    const status = await checkInitStatus()
    if (!status.initialized) {
      // 如果未初始化，跳转到初始化页面
      if (router.currentRoute.value.name !== 'Init') {
        router.push('/init')
      }
      return false
    }
    return true
  } catch (error) {
    console.error('检查初始化状态失败:', error)
    // 如果检查失败，也跳转到初始化页面
    if (router.currentRoute.value.name !== 'Init') {
      router.push('/init')
    }
    return false
  }
}

