<template>
  <div class="login-container">
    <a-card class="login-card" :bordered="false">
      <template #title>
        <div class="login-title">
          <img src="/prjflow512x512.png" alt="PrjFlow" class="login-logo" />
          <h2>项目管理系统</h2>
        </div>
      </template>
      <div class="login-content">
        <a-tabs v-model:activeKey="loginType" centered>
          <a-tab-pane key="password" tab="账号登录">
            <a-form
              :model="loginForm"
              :rules="loginRules"
              @finish="handlePasswordLogin"
              layout="vertical"
            >
              <a-form-item name="username" label="用户名">
                <a-input v-model:value="loginForm.username" placeholder="请输入用户名" size="large" />
              </a-form-item>
              <a-form-item name="password" label="密码">
                <a-input-password v-model:value="loginForm.password" placeholder="请输入密码" size="large" />
              </a-form-item>
              <a-form-item>
                <a-button type="primary" html-type="submit" block size="large" :loading="loading">
                  登录
                </a-button>
              </a-form-item>
            </a-form>
          </a-tab-pane>
          <a-tab-pane key="wechat" tab="微信登录">
            <WeChatQRCode
              :fetchQRCode="getWeChatQRCode"
              initial-status-text="请使用微信扫码"
              hint="扫码后会在微信内打开授权页面"
              :show-auth-url="false"
              @success="handleLoginSuccess"
            />
          </a-tab-pane>
        </a-tabs>
      </div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { getWeChatQRCode, passwordLogin } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { usePermissionStore } from '@/stores/permission'
import WeChatQRCode from '@/components/WeChatQRCode.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const permissionStore = usePermissionStore()

const loginType = ref('password')
const loading = ref(false)
const loginForm = ref({
  username: '',
  password: ''
})

const loginRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

// 用户名密码登录
const handlePasswordLogin = async () => {
  try {
    loading.value = true
    const response = await passwordLogin({
      username: loginForm.value.username,
      password: loginForm.value.password
    })
    handleLoginSuccess(response)
  } catch (error: any) {
    // 响应拦截器已经显示了错误消息，这里不再重复显示
    // 但如果响应拦截器没有显示（比如网络错误），这里作为后备显示
    if (!error.response) {
      message.error(error.message || '登录失败')
    }
  } finally {
    loading.value = false
  }
}

// 处理登录成功
const handleLoginSuccess = async (data: any) => {
  if (data.token && data.user) {
    // 保存token和refresh token
    if (data.refresh_token) {
      authStore.setTokens(data.token, data.refresh_token)
    } else {
      authStore.setToken(data.token)
    }
    // 将 is_first_login 添加到 user 对象中，以便 setUser 可以正确设置状态
    const userData = { ...data.user, is_first_login: data.is_first_login }
    authStore.setUser(userData)
    
    // 加载用户权限
    try {
      await permissionStore.loadPermissions()
    } catch (error) {
      console.error('加载权限失败:', error)
    }
    
    message.success('登录成功！')
    
    // 如果是首次登录，跳转到修改密码页面
    if (data.is_first_login) {
      setTimeout(() => {
        router.push('/auth/change-password')
      }, 1000)
    } else {
      // 否则跳转到指定页面（如果有 redirect 参数）或工作台
      setTimeout(() => {
        const redirect = route.query.redirect as string | undefined
        if (redirect && redirect !== '/login') {
          router.push(redirect)
        } else {
          router.push('/dashboard')
        }
      }, 1000)
    }
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 400px;
  text-align: center;
}

.login-title {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.login-logo {
  height: 40px;
  width: 40px;
  object-fit: contain;
}

.login-title h2 {
  margin: 0;
}

.login-content {
  padding: 20px 0;
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

.hint-small {
  color: #999;
  font-size: 12px;
  margin: 0;
}

.auth-url-container {
  width: 100%;
  max-width: 400px;
  margin: 10px 0;
  padding: 10px;
  background: #f5f5f5;
  border-radius: 4px;
  text-align: left;
}

.auth-url-label {
  font-size: 12px;
  color: #666;
  margin-bottom: 5px;
}

.auth-url-text {
  word-break: break-all;
  font-size: 11px;
  color: #333;
  margin: 0;
}
</style>

