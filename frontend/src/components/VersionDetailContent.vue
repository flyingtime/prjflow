<template>
  <div class="version-detail-content">
    <a-spin :spinning="loading">
      <!-- 基本信息 -->
      <a-card title="基本信息" :bordered="false" style="margin-bottom: 16px">
        <a-descriptions :column="2" bordered>
          <a-descriptions-item label="版本号">{{ version?.version_number }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="getStatusColor(version?.status || '')">
              {{ getStatusText(version?.status || '') }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="项目">
            <a-button v-if="version?.project" type="link" @click="$router.push(`/project/${version.project.id}`)">
              {{ version.project.name }}
            </a-button>
            <span v-else>-</span>
          </a-descriptions-item>
          <a-descriptions-item label="发布日期">
            {{ formatDateTime(version?.release_date) }}
          </a-descriptions-item>
          <a-descriptions-item label="创建时间">
            {{ formatDateTime(version?.created_at) }}
          </a-descriptions-item>
          <a-descriptions-item label="更新时间">
            {{ formatDateTime(version?.updated_at) }}
          </a-descriptions-item>
        </a-descriptions>
      </a-card>

      <!-- 发布说明 -->
      <a-card title="发布说明" :bordered="false" style="margin-bottom: 16px">
        <div v-if="version?.release_notes" class="markdown-content">
          <MarkdownEditor
            :model-value="version.release_notes"
            :readonly="true"
          />
        </div>
        <a-empty v-else description="暂无发布说明" />
      </a-card>

      <!-- 关联需求 -->
      <a-card title="关联需求" :bordered="false" style="margin-bottom: 16px">
        <a-list
          v-if="version?.requirements && version.requirements.length > 0"
          :data-source="version.requirements"
          :pagination="false"
        >
          <template #renderItem="{ item }">
            <a-list-item>
              <a-list-item-meta>
                <template #title>
                  <a-button type="link" @click="$router.push(`/requirement/${item.id}`)">
                    {{ item.title }}
                  </a-button>
                </template>
                <template #description>
                  <a-space>
                    <a-tag :color="getRequirementStatusColor(item.status)">
                      {{ getRequirementStatusText(item.status) }}
                    </a-tag>
                    <a-tag :color="getPriorityColor(item.priority)">
                      {{ getPriorityText(item.priority) }}
                    </a-tag>
                  </a-space>
                </template>
              </a-list-item-meta>
            </a-list-item>
          </template>
        </a-list>
        <a-empty v-else description="暂无关联需求" />
      </a-card>

      <!-- 关联Bug -->
      <a-card title="关联Bug" :bordered="false" style="margin-bottom: 16px">
        <a-list
          v-if="version?.bugs && version.bugs.length > 0"
          :data-source="version.bugs"
          :pagination="false"
        >
          <template #renderItem="{ item }">
            <a-list-item>
              <a-list-item-meta>
                <template #title>
                  <a-button type="link" @click="$router.push(`/bug/${item.id}`)">
                    {{ item.title }}
                  </a-button>
                </template>
                <template #description>
                  <a-space>
                    <a-tag :color="getBugStatusColor(item.status)">
                      {{ getBugStatusText(item.status) }}
                    </a-tag>
                    <a-tag :color="getPriorityColor(item.priority)">
                      {{ getPriorityText(item.priority) }}
                    </a-tag>
                    <a-tag :color="getSeverityColor(item.severity)">
                      {{ getSeverityText(item.severity) }}
                    </a-tag>
                  </a-space>
                </template>
              </a-list-item-meta>
            </a-list-item>
          </template>
        </a-list>
        <a-empty v-else description="暂无关联Bug" />
      </a-card>

      <!-- 附件 -->
      <a-card title="附件" :bordered="false" style="margin-bottom: 16px">
        <div v-if="version?.attachments && version.attachments.length > 0" class="attachment-list">
          <div
            v-for="attachment in version.attachments"
            :key="attachment.id"
            class="attachment-item"
          >
            <div class="attachment-info">
              <PaperClipOutlined class="attachment-icon" />
              <span class="attachment-name" :title="attachment.file_name">{{ attachment.file_name }}</span>
              <span class="attachment-size">{{ formatFileSize(attachment.file_size) }}</span>
            </div>
            <div class="attachment-actions">
              <a-button
                type="link"
                size="small"
                @click="handleDownloadAttachment(attachment)"
              >
                <template #icon><DownloadOutlined /></template>
                下载
              </a-button>
            </div>
          </div>
        </div>
        <a-empty v-else description="暂无附件" />
      </a-card>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { PaperClipOutlined, DownloadOutlined } from '@ant-design/icons-vue'
import { formatDateTime } from '@/utils/date'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import { downloadFile, type Attachment } from '@/api/attachment'
import type { Version } from '@/api/version'

interface Props {
  version?: Version | null
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  version: null,
  loading: false
})

const router = useRouter()

// 格式化文件大小
const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

// 下载附件
const handleDownloadAttachment = async (attachment: Attachment) => {
  try {
    await downloadFile(attachment.id, attachment.file_name)
  } catch (error: any) {
    message.error(error.message || '下载失败')
  }
}

// 状态颜色
const getStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    wait: 'orange',
    normal: 'green',
    fail: 'red',
    terminate: 'default'
  }
  return colors[status] || 'default'
}

// 状态文本
const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    wait: '未开始',
    normal: '已发布',
    fail: '发布失败',
    terminate: '停止维护'
  }
  return texts[status] || status
}

// 需求状态颜色
const getRequirementStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    draft: 'default',
    reviewing: 'blue',
    active: 'green',
    changing: 'orange',
    closed: 'default'
  }
  return colors[status] || 'default'
}

// 需求状态文本
const getRequirementStatusText = (status: string) => {
  const texts: Record<string, string> = {
    draft: '草稿',
    reviewing: '评审中',
    active: '激活',
    changing: '变更中',
    closed: '已关闭'
  }
  return texts[status] || status
}

// Bug状态颜色
const getBugStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    active: 'orange',
    resolved: 'green',
    closed: 'default',
    reopened: 'red'
  }
  return colors[status] || 'default'
}

// Bug状态文本
const getBugStatusText = (status: string) => {
  const texts: Record<string, string> = {
    active: '激活',
    resolved: '已解决',
    closed: '已关闭',
    reopened: '重新打开'
  }
  return texts[status] || status
}

// 优先级颜色
const getPriorityColor = (priority: string) => {
  const colors: Record<string, string> = {
    low: 'default',
    normal: 'blue',
    high: 'orange',
    urgent: 'red'
  }
  return colors[priority] || 'default'
}

// 优先级文本
const getPriorityText = (priority: string) => {
  const texts: Record<string, string> = {
    low: '低',
    normal: '普通',
    high: '高',
    urgent: '紧急'
  }
  return texts[priority] || priority
}

// 严重程度颜色
const getSeverityColor = (severity: string) => {
  const colors: Record<string, string> = {
    low: 'default',
    medium: 'blue',
    high: 'orange',
    critical: 'red'
  }
  return colors[severity] || 'default'
}

// 严重程度文本
const getSeverityText = (severity: string) => {
  const texts: Record<string, string> = {
    low: '低',
    medium: '中',
    high: '高',
    critical: '严重'
  }
  return texts[severity] || severity
}
</script>

<style scoped>
.version-detail-content {
  width: 100%;
}

.attachment-list {
  width: 100%;
}

.attachment-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  margin-bottom: 8px;
  transition: all 0.3s;
}

.attachment-item:hover {
  background-color: #fafafa;
  border-color: #d9d9d9;
}

.attachment-info {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.attachment-icon {
  font-size: 20px;
  color: #1890ff;
  margin-right: 12px;
}

.attachment-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 12px;
}

.attachment-size {
  color: #999;
  font-size: 12px;
}

.attachment-actions {
  flex-shrink: 0;
}
</style>

