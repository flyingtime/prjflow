<template>
  <div class="init-container">
    <a-card class="init-card" :bordered="false">
      <template #title>
        <h2>系统初始化</h2>
        <p class="subtitle">欢迎使用项目管理系统，请完成初始配置</p>
      </template>
      
      <!-- 第一步：配置微信 -->
      <div v-if="step === 1">
        <a-form
          :model="wechatConfig"
          :rules="wechatRules"
          @finish="handleSaveWeChatConfig"
          layout="vertical"
        >
          <a-divider orientation="left">微信配置</a-divider>
          
          <a-form-item label="微信AppID" name="wechat_app_id">
            <a-input
              v-model:value="wechatConfig.wechat_app_id"
              placeholder="请输入微信开放平台AppID"
              size="large"
            />
          </a-form-item>

          <a-form-item label="微信AppSecret" name="wechat_app_secret">
            <a-input-password
              v-model:value="wechatConfig.wechat_app_secret"
              placeholder="请输入微信开放平台AppSecret"
              size="large"
            />
          </a-form-item>

          <a-form-item>
            <a-button
              type="primary"
              html-type="submit"
              size="large"
              :loading="loading"
              block
            >
              保存配置并继续
            </a-button>
          </a-form-item>
        </a-form>
      </div>

      <!-- 第二步：扫码登录创建管理员 -->
      <div v-else-if="step === 2">
        <a-spin :spinning="loading">
          <div class="qr-section">
            <a-divider orientation="left">管理员登录</a-divider>
            <p class="hint">请使用微信扫码登录，系统将自动创建管理员账号</p>
            
            <div v-if="!qrCodeUrl" class="qr-placeholder">
              <a-button type="primary" @click="getQRCode" :loading="qrLoading">
                获取二维码
              </a-button>
            </div>
            
            <div v-else class="qr-container">
              <img :src="qrCodeUrl" alt="微信登录二维码" />
              <p>请使用微信扫码登录</p>
              <a-button @click="getQRCode" :loading="qrLoading">刷新二维码</a-button>
            </div>
          </div>
        </a-spin>
      </div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { 
  checkInitStatus, 
  saveWeChatConfig, 
  getInitQRCode, 
  initSystem,
  type WeChatConfigRequest 
} from '@/api/init'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const step = ref(1) // 1: 配置微信, 2: 扫码登录
const loading = ref(false)
const qrLoading = ref(false)
const qrCodeUrl = ref('')
const ticket = ref('')
let pollTimer: number | null = null

const wechatConfig = ref<WeChatConfigRequest>({
  wechat_app_id: '',
  wechat_app_secret: ''
})

const wechatRules = {
  wechat_app_id: [
    { required: true, message: '请输入微信AppID', trigger: 'blur' }
  ],
  wechat_app_secret: [
    { required: true, message: '请输入微信AppSecret', trigger: 'blur' }
  ]
}

// 保存微信配置
const handleSaveWeChatConfig = async () => {
  loading.value = true
  try {
    await saveWeChatConfig(wechatConfig.value)
    message.success('微信配置保存成功')
    step.value = 2
    // 自动获取二维码
    await getQRCode()
  } catch (error: any) {
    message.error(error.message || '保存配置失败')
  } finally {
    loading.value = false
  }
}

// 获取二维码
const getQRCode = async () => {
  qrLoading.value = true
  try {
    const data = await getInitQRCode()
    qrCodeUrl.value = data.qrCodeUrl
    ticket.value = data.ticket
    startPolling()
  } catch (error: any) {
    message.error(error.message || '获取二维码失败')
  } finally {
    qrLoading.value = false
  }
}

// 开始轮询检查登录状态
const startPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
  }
  
  // 注意：这里需要根据实际的微信登录流程来实现
  // 微信开放平台的扫码登录通常需要：
  // 1. 用户扫码后，微信会回调到配置的回调地址
  // 2. 或者使用轮询方式检查二维码状态
  // 这里暂时使用模拟方式，实际需要根据微信开放平台的API实现
  
  pollTimer = window.setInterval(async () => {
    // 这里应该调用检查二维码状态的API
    // 如果用户已扫码，获取code后调用initSystem
    // 暂时注释，需要根据实际微信API实现
  }, 2000)
}

// 处理微信登录回调（当用户扫码后，微信会返回code）
const handleWeChatLogin = async (code: string) => {
  loading.value = true
  try {
    const response = await initSystem({ code })
    
    // 保存token和用户信息
    authStore.setToken(response.token)
    authStore.setUser(response.user)
    
    message.success('系统初始化成功！')
    
    // 停止轮询
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
    
    // 跳转到工作台
    setTimeout(() => {
      router.push('/dashboard')
    }, 1000)
  } catch (error: any) {
    message.error(error.message || '初始化失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  // 检查是否已经初始化
  try {
    const status = await checkInitStatus()
    if (status.initialized) {
      // 如果已初始化，跳转到登录页
      router.push('/login')
    } else {
      // 检查是否已配置微信
      // 如果已配置，直接进入第二步
      // 这里可以添加检查逻辑
    }
  } catch (error) {
    console.error('检查初始化状态失败:', error)
  }
})

onUnmounted(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
  }
})
</script>

<style scoped>
.init-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.init-card {
  width: 100%;
  max-width: 600px;
}

.subtitle {
  color: #666;
  font-size: 14px;
  margin-top: 8px;
  margin-bottom: 0;
}

.hint {
  color: #666;
  margin-bottom: 20px;
}

.qr-section {
  text-align: center;
}

.qr-placeholder {
  padding: 40px;
}

.qr-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.qr-container img {
  width: 200px;
  height: 200px;
}

:deep(.ant-divider) {
  margin: 24px 0 16px 0;
}
</style>
