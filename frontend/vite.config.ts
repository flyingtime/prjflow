import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    port: 3000,
    host: '0.0.0.0', // 允许外部访问
    allowedHosts: [
      'project.smartxy.com.cn',
      'ungeneralising-harlow-orthogonally.ngrok-free.dev',
      'localhost',
      '127.0.0.1'
    ],
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
        // 不再去掉 /api 前缀，因为后端路由现在统一使用 /api 前缀
      },
      '/uploads': {
        target: 'http://localhost:8080',
        changeOrigin: true
        // 代理上传文件的静态服务
      }
    }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          // 将 node_modules 中的依赖分离
          if (id.includes('node_modules')) {
            // Vue 核心库 - 进一步拆分
            if (id.includes('vue-router')) {
              return 'vue-router'
            }
            if (id.includes('pinia')) {
              return 'pinia'
            }
            // Vue 核心 - 分离运行时和编译器
            if (id.includes('vue') && !id.includes('vue-echarts') && !id.includes('vue-router')) {
              // 如果是 Vue 的编译器部分，单独分离
              if (id.includes('compiler')) {
                return 'vue-compiler'
              }
              return 'vue-core'
            }
            // Ant Design Vue - 可能很大，需要单独分离
            if (id.includes('ant-design-vue')) {
              return 'ant-design-vue'
            }
            // Ant Design Icons
            if (id.includes('@ant-design/icons-vue')) {
              return 'ant-design-icons'
            }
            // ECharts 相关
            if (id.includes('echarts') || id.includes('vue-echarts')) {
              return 'echarts'
            }
            // Markdown 相关 - 进一步拆分
            // highlight.js 很大，包含所有语言，单独分离
            if (id.includes('highlight.js')) {
              return 'highlight'
            }
            if (id.includes('marked')) {
              return 'marked'
            }
            // 日期处理
            if (id.includes('dayjs')) {
              return 'dayjs'
            }
            // Axios
            if (id.includes('axios')) {
              return 'axios'
            }
            // QRCode
            if (id.includes('qrcode')) {
              return 'qrcode'
            }
            // 其他大型库单独分离
            // 如果 vendor 仍然很大，可以进一步拆分
            // 这里将所有其他依赖归为 vendor
            return 'vendor'
          }
        }
      }
    },
    // 调整 chunk 大小警告限制为 2000KB
    // 由于某些库（如 Vue 3 运行时、highlight.js、echarts）本身很大，设置为 2000KB 更合理
    // Vue 3 运行时约 1.8MB，但 gzip 压缩后只有 444KB，实际传输大小很小
    chunkSizeWarningLimit: 2000
  }
})
