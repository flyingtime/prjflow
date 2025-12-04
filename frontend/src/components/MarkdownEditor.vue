<template>
  <div class="markdown-editor">
    <a-tabs v-model:activeKey="activeTab" v-if="!readonly">
      <a-tab-pane key="edit" tab="编辑">
        <div ref="editorContainerRef" class="editor-container">
          <a-textarea
            ref="textareaRef"
            :value="modelValue || ''"
            @update:value="handleInput"
            :placeholder="placeholder"
            :rows="rows"
            style="font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace"
          />
        </div>
      </a-tab-pane>
      <a-tab-pane key="preview" tab="预览">
        <div 
          ref="previewContainerRef" 
          class="markdown-preview" 
          v-html="renderedMarkdown"
        ></div>
      </a-tab-pane>
    </a-tabs>
    <div 
      v-else 
      ref="previewContainerRef" 
      class="markdown-preview" 
      v-html="renderedMarkdown"
    ></div>
    
    <!-- 图片预览模态框 -->
    <a-modal
      v-model:open="imagePreviewVisible"
      :footer="null"
      :width="'90%'"
      :style="{ top: '20px' }"
      :mask-closable="true"
      @cancel="closeImagePreview"
      class="image-preview-modal"
    >
      <div class="image-preview-container" @wheel.prevent="handleImageWheel">
        <img
          ref="previewImageRef"
          :src="previewImageSrc"
          :style="{
            maxWidth: '100%',
            maxHeight: '90vh',
            width: imageScale + '%',
            height: 'auto',
            cursor: 'zoom-in',
            transition: 'width 0.1s ease-out',
            display: 'block',
            margin: '0 auto'
          }"
          @click="closeImagePreview"
          alt="预览图片"
        />
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/github.css'

interface Props {
  modelValue?: string
  placeholder?: string
  rows?: number
  readonly?: boolean
  projectId?: number // 项目ID，用于上传图片
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  placeholder: '请输入Markdown内容...',
  rows: 8,
  readonly: false,
  projectId: 0
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'image-uploaded': [oldUrl: string, newUrl: string] // 图片上传完成事件
}>()

const activeTab = ref('edit')
const textareaRef = ref<any>(null)
const editorContainerRef = ref<HTMLElement | null>(null)
const previewContainerRef = ref<HTMLElement | null>(null)
const previewImageRef = ref<HTMLImageElement | null>(null)

// 图片预览相关
const imagePreviewVisible = ref(false)
const previewImageSrc = ref('')
const imageScale = ref(100) // 图片缩放比例，默认100%

// 存储本地预览的图片映射（blob URL -> File）
const localImages = new Map<string, File>()

// 配置marked
marked.setOptions({
  highlight: function(code: string, lang: string) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return hljs.highlight(code, { language: lang }).value
      } catch (err) {
        console.error('Highlight error:', err)
      }
    }
    return hljs.highlightAuto(code).value
  },
  breaks: true,
  gfm: true
} as any)

// 渲染Markdown
const renderedMarkdown = computed(() => {
  if (!props.modelValue || props.modelValue.trim() === '') {
    return '<p class="empty-text">暂无内容</p>'
  }
  let html = marked.parse(props.modelValue) as string
  
  // 为所有图片添加点击事件
  html = html.replace(/<img([^>]*)\ssrc=["']([^"']+)["']([^>]*)>/gi, (match, before, src, after) => {
    // 添加点击事件和样式
    return `<img${before} src="${src}"${after} class="markdown-image-clickable" data-image-src="${src}" style="cursor: pointer;">`
  })
  
  // 处理链接，防止内部路由链接被Vue Router拦截
  // 将看起来像内部路由的链接转换为外部链接，或者阻止默认行为
  html = html.replace(/<a([^>]*)\shref=["']([^"']+)["']([^>]*)>/gi, (match, before, href, after) => {
    // 如果是内部路由路径（以 / 开头但不是 http/https），添加特殊处理
    if (href.startsWith('/') && !href.startsWith('//') && !href.startsWith('/uploads/')) {
      // 添加 data-href 属性，并阻止默认行为
      return `<a${before} href="javascript:void(0)" data-href="${href}"${after} class="markdown-link-internal">`
    }
    // 其他链接保持原样
    return match
  })
  
  // 调试：在只读模式下检查图片URL
  if (props.readonly) {
    const imgRegex = /!\[([^\]]*)\]\(([^)]+)\)/g
    const matches = Array.from(props.modelValue.matchAll(imgRegex))
    if (matches.length > 0) {
      console.log('Markdown中的图片URL:', matches.map(m => m[2]))
    }
  }
  return html
})

// 处理输入
const handleInput = (value: string) => {
  emit('update:modelValue', value)
}

// 处理粘贴事件
const handlePaste = async (e: ClipboardEvent) => {
  if (props.readonly) return

  const items = e.clipboardData?.items
  if (!items) return

  // 查找图片项
  for (let i = 0; i < items.length; i++) {
    const item = items[i]
    if (!item) continue
    if (item.type.indexOf('image') !== -1) {
      e.preventDefault()
      e.stopPropagation()

      const file = item.getAsFile()
      if (!file) continue

      // 创建 blob URL 用于本地预览
      const blobUrl = URL.createObjectURL(file)
      localImages.set(blobUrl, file)

      // 获取 textarea 元素
      const textarea = textareaRef.value?.$el?.querySelector('textarea') as HTMLTextAreaElement | null
      if (!textarea) {
        // 如果找不到 textarea，尝试直接使用 ref
        const directTextarea = textareaRef.value?.$el as HTMLTextAreaElement
        if (directTextarea && directTextarea.tagName === 'TEXTAREA') {
          insertImageAtCursor(directTextarea, file, blobUrl)
        }
        break
      }

      insertImageAtCursor(textarea, file, blobUrl)
      break
    }
  }
}

// 在光标位置插入图片
const insertImageAtCursor = (textarea: HTMLTextAreaElement, file: File, blobUrl: string) => {
  const start = textarea.selectionStart || 0
  const end = textarea.selectionEnd || 0
  const currentValue = props.modelValue || ''

  // 插入图片 Markdown 语法
  const imageMarkdown = `![${file.name}](${blobUrl})\n`
  const newValue = currentValue.slice(0, start) + imageMarkdown + currentValue.slice(end)

  emit('update:modelValue', newValue)

  // 设置光标位置
  setTimeout(() => {
    textarea.focus()
    const newPosition = start + imageMarkdown.length
    textarea.setSelectionRange(newPosition, newPosition)
  }, 0)
}

// 上传本地预览的图片并替换URL
// 这个方法由父组件在提交表单时调用
const uploadLocalImages = async (uploadFn: (file: File, projectId: number) => Promise<{ file_path: string }>): Promise<string> => {
  let content = props.modelValue || ''

  // 查找所有本地预览的图片URL（blob URL）
  const blobUrlRegex = /!\[([^\]]*)\]\((blob:[^)]+)\)/g
  const matches = Array.from(content.matchAll(blobUrlRegex))

  for (const match of matches) {
    const blobUrl = match[2]
    if (!blobUrl) continue
    const file = localImages.get(blobUrl)

    if (file && props.projectId) {
      try {
        // 上传图片
        const attachment = await uploadFn(file, props.projectId)
        
        // 构建服务器URL（使用 /uploads/ 前缀）
        // file_path 格式：YYYY/MM/DD/filename.ext
        const serverUrl = `/uploads/${attachment.file_path}`
        
        console.log('上传图片成功:', {
          blobUrl,
          serverUrl,
          file_path: attachment.file_path
        })
        
        // 替换URL（替换所有匹配的blob URL）
        const escapedBlobUrl = blobUrl.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
        content = content.replace(new RegExp(escapedBlobUrl, 'g'), serverUrl)
        
        // 清理 blob URL
        URL.revokeObjectURL(blobUrl)
        localImages.delete(blobUrl)

        emit('image-uploaded', blobUrl, serverUrl)
      } catch (error) {
        console.error('上传图片失败:', error)
        // 上传失败时，可以选择保留blob URL或移除图片标记
        // 这里选择保留，让用户知道上传失败
      }
    }
  }

  return content
}

// 暴露方法给父组件
defineExpose({
  uploadLocalImages,
  getCurrentContent: () => props.modelValue || '' // 获取当前编辑器内容
})

// 打开图片预览
const openImagePreview = (imageSrc: string) => {
  previewImageSrc.value = imageSrc
  imageScale.value = 100 // 重置缩放
  imagePreviewVisible.value = true
}

// 关闭图片预览
const closeImagePreview = () => {
  imagePreviewVisible.value = false
  imageScale.value = 100 // 重置缩放
}

// 处理图片滚轮缩放
const handleImageWheel = (e: WheelEvent) => {
  e.preventDefault()
  const delta = e.deltaY > 0 ? -10 : 10 // 向下滚动缩小，向上滚动放大
  const newScale = Math.max(50, Math.min(500, imageScale.value + delta)) // 限制在50%-500%之间
  imageScale.value = newScale
}

// 处理图片点击事件
const handleImageClick = (e: Event) => {
  const target = e.target as HTMLElement
  if (target.tagName === 'IMG' && target.classList.contains('markdown-image-clickable')) {
    e.preventDefault()
    e.stopPropagation()
    const imageSrc = target.getAttribute('data-image-src') || target.getAttribute('src') || ''
    if (imageSrc) {
      openImagePreview(imageSrc)
    }
  }
}

// 处理链接点击事件，防止内部路由链接被Vue Router拦截
const handleLinkClick = (e: Event) => {
  const target = e.target as HTMLElement
  if (target.tagName === 'A' && target.classList.contains('markdown-link-internal')) {
    e.preventDefault()
    e.stopPropagation()
    const href = target.getAttribute('data-href')
    if (href) {
      // 如果是内部路由，可以选择打开新窗口或者提示用户
      console.warn('Markdown中的内部链接被阻止:', href)
      // 或者可以选择在新窗口打开
      // window.open(href, '_blank')
    }
  }
}

// 绑定图片和链接点击事件
const bindImageClickHandler = () => {
  if (previewContainerRef.value) {
    // 移除旧的事件监听器（如果存在）
    previewContainerRef.value.removeEventListener('click', handleImageClick)
    previewContainerRef.value.removeEventListener('click', handleLinkClick)
    // 添加新的事件监听器
    previewContainerRef.value.addEventListener('click', handleImageClick)
    previewContainerRef.value.addEventListener('click', handleLinkClick)
  }
}

// 挂载时添加粘贴事件监听和图片点击事件监听
onMounted(() => {
  if (!props.readonly && editorContainerRef.value) {
    editorContainerRef.value.addEventListener('paste', handlePaste as unknown as EventListener)
  }
  
  // 使用事件委托，在预览容器上监听所有图片点击事件
  // 这样即使内容动态更新，也不需要重新绑定事件
  // 延迟绑定，确保DOM已渲染
  setTimeout(bindImageClickHandler, 100)
})

// 监听activeTab变化，当切换到预览标签页时重新绑定事件
watch(() => activeTab.value, (newTab) => {
  if (newTab === 'preview') {
    // 切换到预览标签页时，延迟绑定事件以确保DOM已渲染
    setTimeout(bindImageClickHandler, 100)
  }
})

// 监听renderedMarkdown变化，确保新渲染的图片也能点击
watch(() => renderedMarkdown.value, () => {
  // 由于使用了事件委托，不需要重新绑定事件
  // 但如果预览容器还没有绑定事件，则绑定一次
  if (previewContainerRef.value) {
    setTimeout(bindImageClickHandler, 100)
  }
})

// 组件卸载时清理 blob URL和事件监听
onUnmounted(() => {
  if (editorContainerRef.value) {
    editorContainerRef.value.removeEventListener('paste', handlePaste as unknown as EventListener)
  }
  if (previewContainerRef.value) {
    previewContainerRef.value.removeEventListener('click', handleImageClick)
    previewContainerRef.value.removeEventListener('click', handleLinkClick)
  }
  localImages.forEach((_, blobUrl) => {
    URL.revokeObjectURL(blobUrl)
  })
  localImages.clear()
})

// 监听modelValue变化，自动切换到预览
watch(() => props.modelValue, () => {
  if (activeTab.value === 'preview' && props.modelValue) {
    // 如果正在预览且有内容，保持预览状态
  }
})
</script>

<style scoped>
.markdown-editor {
  width: 100%;
}

.editor-container {
  width: 100%;
}

.markdown-preview {
  min-height: 200px;
  padding: 12px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  overflow-y: auto;
  max-height: 600px;
}

.markdown-preview :deep(h1),
.markdown-preview :deep(h2),
.markdown-preview :deep(h3),
.markdown-preview :deep(h4),
.markdown-preview :deep(h5),
.markdown-preview :deep(h6) {
  margin-top: 16px;
  margin-bottom: 8px;
  font-weight: 600;
  line-height: 1.25;
}

.markdown-preview :deep(h1) {
  font-size: 2em;
  border-bottom: 1px solid #eaecef;
  padding-bottom: 0.3em;
}

.markdown-preview :deep(h2) {
  font-size: 1.5em;
  border-bottom: 1px solid #eaecef;
  padding-bottom: 0.3em;
}

.markdown-preview :deep(h3) {
  font-size: 1.25em;
}

.markdown-preview :deep(p) {
  margin-bottom: 16px;
  line-height: 1.6;
}

.markdown-preview :deep(ul),
.markdown-preview :deep(ol) {
  margin-bottom: 16px;
  padding-left: 2em;
}

.markdown-preview :deep(li) {
  margin-bottom: 4px;
}

.markdown-preview :deep(blockquote) {
  padding: 0 1em;
  color: #6a737d;
  border-left: 0.25em solid #dfe2e5;
  margin-bottom: 16px;
}

.markdown-preview :deep(code) {
  padding: 0.2em 0.4em;
  margin: 0;
  font-size: 85%;
  background-color: rgba(27, 31, 35, 0.05);
  border-radius: 3px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.markdown-preview :deep(pre) {
  padding: 16px;
  overflow: auto;
  font-size: 85%;
  line-height: 1.45;
  background-color: #f6f8fa;
  border-radius: 6px;
  margin-bottom: 16px;
}

.markdown-preview :deep(pre code) {
  display: inline;
  max-width: auto;
  padding: 0;
  margin: 0;
  overflow: visible;
  line-height: inherit;
  word-wrap: normal;
  background-color: transparent;
  border: 0;
}

.markdown-preview :deep(table) {
  border-collapse: collapse;
  margin-bottom: 16px;
  width: 100%;
}

.markdown-preview :deep(table th),
.markdown-preview :deep(table td) {
  padding: 6px 13px;
  border: 1px solid #dfe2e5;
}

.markdown-preview :deep(table th) {
  font-weight: 600;
  background-color: #f6f8fa;
}

.markdown-preview :deep(table tr:nth-child(2n)) {
  background-color: #f6f8fa;
}

.markdown-preview :deep(a) {
  color: #0366d6;
  text-decoration: none;
}

.markdown-preview :deep(a:hover) {
  text-decoration: underline;
}

.markdown-preview :deep(img) {
  max-width: 100%;
  box-sizing: content-box;
  background-color: #fff;
  display: block;
  margin: 16px 0;
  border-radius: 4px;
}

.markdown-preview :deep(img.markdown-image-clickable) {
  cursor: pointer;
  transition: opacity 0.2s;
}

.markdown-preview :deep(img.markdown-image-clickable:hover) {
  opacity: 0.8;
}

/* 图片预览模态框样式 */
.image-preview-modal :deep(.ant-modal-body) {
  padding: 0;
  text-align: center;
  background-color: rgba(0, 0, 0, 0.85);
}

.image-preview-container {
  width: 100%;
  height: 90vh;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: auto;
  padding: 20px;
}

.markdown-preview :deep(hr) {
  height: 0.25em;
  padding: 0;
  margin: 24px 0;
  background-color: #e1e4e8;
  border: 0;
}

.empty-text {
  color: #999;
  font-style: italic;
}
</style>

