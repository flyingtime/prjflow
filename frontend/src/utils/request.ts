import axios from 'axios'
import type { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse, AxiosRequestConfig } from 'axios'
import { message } from 'ant-design-vue'
import { useAuthStore } from '../stores/auth'
import { refreshToken as refreshTokenAPI } from '../api/auth'
import router from '../router'

// 使用相对路径，通过 Vite 代理转发到后端
const baseURL = import.meta.env.VITE_API_BASE_URL || '/api'

const axiosInstance: AxiosInstance = axios.create({
  baseURL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  },
  paramsSerializer: {
    indexes: null // 将数组序列化为 tags=value1&tags=value2 格式
  }
})

// 刷新 token 的 Promise，用于防止并发请求时多次刷新
let refreshTokenPromise: Promise<string | null> | null = null

// 防止重复跳转登录的标志位
let isRedirectingToLogin = false

// 跳转到登录页的辅助函数
const redirectToLogin = (authStore: ReturnType<typeof useAuthStore>) => {
  // 如果正在跳转，直接返回，避免重复调用
  if (isRedirectingToLogin) {
    return
  }
  
  isRedirectingToLogin = true
  const currentRoute = router.currentRoute.value
  
  if (currentRoute.name !== 'Login') {
    authStore.logout()
    
    const redirectPath = currentRoute.fullPath !== '/' ? currentRoute.fullPath : undefined
    message.error('登录已过期，请重新登录')
    
    router.push({
      name: 'Login',
      query: redirectPath ? { redirect: redirectPath } : {}
    }).finally(() => {
      // 跳转完成后重置标志位（延迟一点，确保所有并发请求都已处理）
      setTimeout(() => {
        isRedirectingToLogin = false
      }, 1000)
    })
  } else {
    authStore.logout()
    // 如果已经在登录页，立即重置标志位
    setTimeout(() => {
      isRedirectingToLogin = false
    }, 1000)
  }
}

// 刷新 token 的函数（支持并发请求，所有请求共享同一个刷新操作）
const refreshAccessToken = async (): Promise<string | null> => {
  const authStore = useAuthStore()
  
  // 如果没有 refresh token，直接返回 null
  if (!authStore.refreshToken) {
    return null
  }

  // 如果已经有正在进行的刷新请求，等待它完成（所有并发请求共享同一个Promise）
  if (refreshTokenPromise) {
    try {
      return await refreshTokenPromise
    } catch {
      // 如果之前的刷新失败，返回 null
      return null
    }
  }

  // 创建新的刷新请求（使用立即执行的异步函数确保原子性）
  refreshTokenPromise = (async () => {
    try {
      const response = await refreshTokenAPI({ refresh_token: authStore.refreshToken! })
      
      // 更新 token 和 refresh token
      authStore.setTokens(response.token, response.refresh_token)
      
      return response.token
    } catch (error: any) {
      // 刷新失败，返回 null（不在这里调用logout，由调用者统一处理）
      return null
    } finally {
      // 清除刷新 Promise，允许下次刷新（延迟清除，确保所有等待的请求都能获取结果）
      // 使用 setTimeout 确保在 Promise 完成后才清除
      setTimeout(() => {
        refreshTokenPromise = null
      }, 100)
    }
  })()

  // 等待刷新完成并返回结果
  try {
    return await refreshTokenPromise
  } catch {
    return null
  }
}

// 请求拦截器
axiosInstance.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const authStore = useAuthStore()
    if (authStore.token && config.headers) {
      config.headers.Authorization = `Bearer ${authStore.token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
axiosInstance.interceptors.response.use(
  async (response: AxiosResponse) => {
    const res = response.data
    
    // 如果响应格式是 { code, message, data }
    if (res.code !== undefined) {
      if (res.code === 200) {
        return res.data
      } else {
        // 处理业务错误码（如401未授权）
        // 注意：后端返回401时，HTTP状态码是200，但业务code是401
        if (res.code === 401) {
          const originalRequest = response.config as InternalAxiosRequestConfig & { _retry?: boolean }
          const authStore = useAuthStore()
          
          // 如果请求是刷新 token 的请求，或者已经重试过，直接跳转登录
          if (originalRequest.url?.includes('/auth/refresh') || originalRequest._retry) {
            redirectToLogin(authStore)
            return Promise.reject(new Error(res.message || '登录已过期'))
          }
          
          // 尝试刷新 token
          const newToken = await refreshAccessToken()
          
          if (newToken) {
            // 刷新成功，重试原始请求
            originalRequest._retry = true
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${newToken}`
            }
            return axiosInstance(originalRequest)
          } else {
            // 刷新失败，清除认证信息并跳转到登录页
            redirectToLogin(authStore)
            return Promise.reject(new Error(res.message || '登录已过期'))
          }
        }
        
        // 其他业务错误码，显示错误消息
        message.error(res.message || '请求失败')
        return Promise.reject(new Error(res.message || '请求失败'))
      }
    }
    
    return res
  },
  async (error) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }
    
    if (error.response) {
      const { status, data } = error.response
      
      switch (status) {
        case 401:
          // token 失效，尝试使用 refresh token 刷新
          const authStore = useAuthStore()
          
          // 如果请求是刷新 token 的请求，或者已经重试过，直接跳转登录
          if (originalRequest.url?.includes('/auth/refresh') || originalRequest._retry) {
            // refresh token 也失效了，清除认证信息并跳转到登录页
            redirectToLogin(authStore)
            return Promise.reject(error)
          }
          
          // 尝试刷新 token
          const newToken = await refreshAccessToken()
          
          if (newToken) {
            // 刷新成功，重试原始请求
            originalRequest._retry = true
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${newToken}`
            }
            return axiosInstance(originalRequest)
          } else {
            // 刷新失败，清除认证信息并跳转到登录页
            redirectToLogin(authStore)
          }
          break
        case 403:
          message.error('没有权限访问')
          break
        case 404:
          message.error('请求的资源不存在')
          break
        case 500:
          message.error('服务器错误')
          break
        default:
          message.error(data?.message || `请求失败: ${status}`)
      }
    } else {
      message.error('网络错误，请检查网络连接')
    }
    
    return Promise.reject(error)
  }
)

// 创建包装的 request 对象，确保类型正确
const request = {
  get: <T = any>(url: string, config?: AxiosRequestConfig): Promise<T> => {
    return axiosInstance.get(url, config).then(res => res as unknown as T)
  },
  post: <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> => {
    return axiosInstance.post(url, data, config).then(res => res as unknown as T)
  },
  put: <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> => {
    return axiosInstance.put(url, data, config).then(res => res as unknown as T)
  },
  delete: <T = any>(url: string, config?: AxiosRequestConfig): Promise<T> => {
    return axiosInstance.delete(url, config).then(res => res as unknown as T)
  },
  patch: <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> => {
    return axiosInstance.patch(url, data, config).then(res => res as unknown as T)
  }
}

export default request

