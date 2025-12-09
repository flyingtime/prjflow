<template>
  <div class="attachment-upload">
    <!-- 上传按钮 -->
    <a-upload
      v-if="!readonly"
      v-permission="'attachment:upload'"
      :file-list="fileList"
      :before-upload="beforeUpload"
      :custom-request="handleUpload"
      :show-upload-list="false"
      multiple
    >
      <a-button>
        <template #icon><UploadOutlined /></template>
        选择文件
      </a-button>
    </a-upload>

    <!-- 文件列表（只显示新上传的文件，排除已在 existingAttachments 中的） -->
    <div v-if="newUploadedFiles.length > 0" class="file-list">
      <div
        v-for="(file, index) in newUploadedFiles"
        :key="file.uid || index"
        class="file-item"
      >
        <div class="file-info">
          <PaperClipOutlined class="file-icon" />
          <span class="file-name" :title="file.name">{{ file.name }}</span>
          <span class="file-size">{{ formatFileSize(file.size || 0) }}</span>
        </div>
        <div class="file-actions">
          <!-- 上传进度 -->
          <a-progress
            v-if="file.status === 'uploading'"
            :percent="file.percent || 0"
            size="small"
            style="width: 100px; margin-right: 8px"
          />
          <!-- 下载按钮（已上传） -->
          <a-button
            v-if="file.status === 'done' && file.id"
            type="link"
            size="small"
            @click="handleDownload(file)"
          >
            <template #icon><DownloadOutlined /></template>
          </a-button>
          <!-- 删除按钮 -->
          <a-button
            v-if="!readonly"
            v-permission="'attachment:delete'"
            type="link"
            size="small"
            danger
            @click="handleRemove(file, index)"
          >
            <template #icon><DeleteOutlined /></template>
          </a-button>
        </div>
      </div>
    </div>

    <!-- 已存在的附件列表 -->
    <div v-if="existingAttachments.length > 0" class="file-list">
      <div
        v-for="attachment in existingAttachments"
        :key="attachment.id"
        class="file-item"
      >
        <div class="file-info">
          <PaperClipOutlined class="file-icon" />
          <span class="file-name" :title="attachment.file_name">{{ attachment.file_name }}</span>
          <span class="file-size">{{ formatFileSize(attachment.file_size) }}</span>
        </div>
        <div class="file-actions">
          <a-button
            type="link"
            size="small"
            @click="handleDownloadAttachment(attachment)"
          >
            <template #icon><DownloadOutlined /></template>
          </a-button>
          <!-- 删除按钮（非只读模式） -->
          <a-button
            v-if="!readonly"
            v-permission="'attachment:delete'"
            type="link"
            size="small"
            danger
            @click="handleRemoveExistingAttachment(attachment)"
          >
            <template #icon><DeleteOutlined /></template>
          </a-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { message } from 'ant-design-vue'
import {
  UploadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  PaperClipOutlined
} from '@ant-design/icons-vue'
import type { UploadFile } from 'ant-design-vue'
import type { UploadRequestOption } from 'ant-design-vue/es/vc-upload/interface'
import { uploadFile as uploadFileAPI, deleteAttachment, downloadFile, type Attachment } from '@/api/attachment'

interface Props {
  projectId: number
  modelValue?: number[] // 已上传的附件ID列表
  readonly?: boolean
  existingAttachments?: Attachment[] // 已存在的附件列表（只读模式）
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => [],
  readonly: false,
  existingAttachments: () => []
})

const emit = defineEmits<{
  'update:modelValue': [value: number[]]
  'attachment-deleted': [attachmentId: number] // 附件删除事件
}>()

interface ExtendedUploadFile extends UploadFile {
  id?: number
}

const fileList = ref<ExtendedUploadFile[]>([])

// 计算属性：只显示新上传的文件，排除已在 existingAttachments 中的文件
const newUploadedFiles = computed(() => {
  const existingIds = props.existingAttachments?.map(a => a.id) || []
  return fileList.value.filter(file => {
    // 如果文件已上传完成且有ID，且不在 existingAttachments 中，则显示
    if (file.status === 'done' && file.id) {
      return !existingIds.includes(file.id)
    }
    // 正在上传的文件总是显示
    return true
  })
})

// 格式化文件大小
const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

// 上传前验证
const beforeUpload = (file: File): boolean => {
  // 文件大小限制：100MB
  const maxSize = 100 * 1024 * 1024
  if (file.size > maxSize) {
    message.error('文件大小不能超过 100MB')
    return false
  }
  return true
}

// 自定义上传
const handleUpload = async (options: UploadRequestOption) => {
  const { file, onProgress, onSuccess, onError } = options

  if (!(file instanceof File)) {
    onError?.(new Error('无效的文件'))
    return
  }

  // 添加到文件列表
  const uploadFileItem: ExtendedUploadFile = {
    uid: `${Date.now()}-${Math.random()}`,
    name: file.name,
    size: file.size,
    status: 'uploading',
    percent: 0
  }
  fileList.value.push(uploadFileItem)

  try {
    // 上传文件
    const attachment = await uploadFileAPI(
      file,
      props.projectId,
      (progress) => {
        uploadFileItem.percent = progress
        onProgress?.({ percent: progress })
      }
    )

    // 更新文件状态
    uploadFileItem.status = 'done'
    uploadFileItem.percent = 100
    uploadFileItem.id = attachment.id
    uploadFileItem.response = attachment

    // 更新已上传的附件ID列表
    const currentIds = props.modelValue ? [...props.modelValue] : []
    currentIds.push(attachment.id)
    emit('update:modelValue', currentIds)

    onSuccess?.(attachment)
    message.success('上传成功')
  } catch (error: any) {
    uploadFileItem.status = 'error'
    onError?.(error)
    message.error(error.message || '上传失败')
  }
}

// 删除文件
const handleRemove = async (file: ExtendedUploadFile, index: number) => {
  // 如果已上传，需要调用删除API
  if (file.status === 'done' && file.id) {
    try {
      await deleteAttachment(file.id)
      // 从已上传的附件ID列表中移除
      const currentIds = (props.modelValue || []).filter(id => id !== file.id)
      emit('update:modelValue', currentIds)
      message.success('删除成功')
    } catch (error: any) {
      message.error(error.message || '删除失败')
      return
    }
  }

  // 从文件列表中移除（使用文件对象查找实际索引）
  const actualIndex = fileList.value.findIndex(f => f.uid === file.uid || (f.id && f.id === file.id))
  if (actualIndex !== -1) {
    fileList.value.splice(actualIndex, 1)
  }
}

// 下载文件
const handleDownload = async (file: ExtendedUploadFile) => {
  if (!file.id || !file.name) return

  try {
    await downloadFile(file.id, file.name)
  } catch (error: any) {
    message.error(error.message || '下载失败')
  }
}

// 下载已存在的附件
const handleDownloadAttachment = async (attachment: Attachment) => {
  try {
    await downloadFile(attachment.id, attachment.file_name)
  } catch (error: any) {
    message.error(error.message || '下载失败')
  }
}

// 删除已存在的附件
const handleRemoveExistingAttachment = async (attachment: Attachment) => {
  try {
    await deleteAttachment(attachment.id)
    // 从已上传的附件ID列表中移除
    const currentIds = (props.modelValue || []).filter(id => id !== attachment.id)
    emit('update:modelValue', currentIds)
    // 通知父组件附件已删除
    emit('attachment-deleted', attachment.id)
    message.success('删除成功')
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

// 监听 existingAttachments 和 modelValue 的变化，确保 modelValue 包含所有已存在的附件ID
watch([() => props.existingAttachments, () => props.modelValue], ([newAttachments, currentIds]) => {
  // 如果 existingAttachments 为空，不处理（但保留已有的 modelValue）
  if (!newAttachments || newAttachments.length === 0) {
    // 但如果 modelValue 是 undefined，需要初始化为空数组
    if (currentIds === undefined) {
      emit('update:modelValue', [])
    }
    return
  }
  
  const existingIds = newAttachments.map(a => a.id)
  const currentIdsList = currentIds || []
  
  // 检查是否所有已存在的附件ID都在modelValue中
  const missingIds = existingIds.filter(id => !currentIdsList.includes(id))
  
  // 只有当有缺失的ID时才更新，避免覆盖已设置的值（包括新上传的附件）
  if (missingIds.length > 0) {
    // 合并已存在的附件ID和新上传的附件ID
    const allIds = [...new Set([...currentIdsList, ...missingIds])]
    emit('update:modelValue', allIds)
  } else {
    // 即使没有缺失的ID，也要确保 modelValue 不是 undefined
    if (currentIds === undefined && existingIds.length > 0) {
      emit('update:modelValue', existingIds)
    }
  }
}, { deep: true, immediate: true })

// 监听 modelValue 变化，清理已删除的附件
watch(() => props.modelValue, (newIds) => {
  // 移除不在新列表中的已上传文件
  fileList.value = fileList.value.filter(file => {
    if (file.status === 'done' && file.id) {
      return newIds.includes(file.id)
    }
    return true
  })
}, { deep: true })
</script>

<style scoped>
.attachment-upload {
  width: 100%;
}

.file-list {
  margin-top: 16px;
}

.file-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  margin-bottom: 8px;
  background: #fafafa;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
}

.file-info {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.file-icon {
  margin-right: 8px;
  color: #1890ff;
}

.file-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 8px;
}

.file-size {
  color: #999;
  font-size: 12px;
  white-space: nowrap;
}

.file-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>

