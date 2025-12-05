<template>
  <div class="project-detail-content">
    <a-spin :spinning="loading">
      <!-- 项目基本信息 -->
      <a-card title="基本信息" :bordered="false" style="margin-bottom: 16px">
        <a-descriptions :column="2" bordered>
          <a-descriptions-item label="项目名称">{{ project?.name }}</a-descriptions-item>
          <a-descriptions-item label="项目编码">{{ project?.code }}</a-descriptions-item>
          <a-descriptions-item label="开始日期">{{ project?.start_date || '-' }}</a-descriptions-item>
          <a-descriptions-item label="结束日期">{{ project?.end_date || '-' }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="project?.status === 'doing' || project?.status === 'wait' ? 'green' : 'red'">
              {{ project?.status === 'doing' || project?.status === 'wait' ? '正常' : '禁用' }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="负责人">
            {{ getProjectOwner() || '-' }}
          </a-descriptions-item>
          <a-descriptions-item label="成员数">{{ statistics?.total_members || 0 }} 人</a-descriptions-item>
          <a-descriptions-item label="描述" :span="2">
            <div v-if="project?.description" class="description-content">
              <MarkdownEditor
                :model-value="project.description"
                :readonly="true"
              />
            </div>
            <span v-else>-</span>
          </a-descriptions-item>
        </a-descriptions>
      </a-card>

      <!-- 项目描述 -->
      <a-card title="项目描述" :bordered="false" style="margin-bottom: 16px" v-if="project?.description">
        <div class="markdown-content">
          <MarkdownEditor
            :model-value="project.description"
            :readonly="true"
          />
        </div>
      </a-card>

      <!-- 历史记录 -->
      <a-card :bordered="false" style="margin-bottom: 16px">
        <template #title>
          <span>历史记录</span>
          <a-button 
            type="link" 
            size="small"
            @click.stop="handleAddNote" 
            :disabled="historyLoading"
            style="margin-left: 8px; padding: 0"
          >
            添加备注
          </a-button>
        </template>
        <a-spin :spinning="historyLoading" :style="{ minHeight: '100px' }">
          <a-timeline v-if="historyList.length > 0">
            <a-timeline-item
              v-for="(action, index) in historyList"
              :key="action.id"
            >
              <template #dot>
                <span style="font-weight: bold; color: #1890ff">{{ historyList.length - index }}</span>
              </template>
              <div>
                <div style="margin-bottom: 8px">
                  <span style="color: #666; margin-right: 8px">{{ formatDateTime(action.date) }}</span>
                  <span>{{ getActionDescription(action) }}</span>
                  <a-button
                    v-if="hasHistoryDetails(action)"
                    type="link"
                    size="small"
                    @click="toggleHistoryDetail(action.id)"
                    style="padding: 0; height: auto; margin-left: 8px"
                  >
                    {{ expandedHistoryIds.has(action.id) ? '收起' : '展开' }}
                  </a-button>
                </div>
                <!-- 字段变更详情和备注内容（可折叠） -->
                <div
                  v-show="expandedHistoryIds.has(action.id)"
                  style="margin-left: 24px; margin-top: 8px"
                >
                  <!-- 字段变更详情 -->
                  <div v-if="action.histories && action.histories.length > 0">
                    <div
                      v-for="history in action.histories"
                      :key="history.id"
                      style="margin-bottom: 8px; color: #666"
                    >
                      <div>修改了{{ getFieldDisplayName(history.field) }}</div>
                      <div style="margin-left: 16px; margin-top: 4px;">
                        <div>旧值："{{ history.old_value || history.old || '-' }}"</div>
                        <div>新值："{{ history.new_value || history.new || '-' }}"</div>
                      </div>
                    </div>
                  </div>
                  <!-- 备注内容 -->
                  <div v-if="action.comment" style="margin-top: 8px; color: #666">
                    {{ action.comment }}
                  </div>
                </div>
              </div>
            </a-timeline-item>
          </a-timeline>
          <a-empty v-else description="暂无历史记录" />
        </a-spin>
      </a-card>

      <!-- 统计概览 -->
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :span="6">
          <a-card :bordered="false">
            <a-statistic
              title="总任务数"
              :value="statistics?.total_tasks || 0"
              :value-style="{ color: '#1890ff' }"
            />
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card :bordered="false">
            <a-statistic
              title="总Bug数"
              :value="statistics?.total_bugs || 0"
              :value-style="{ color: '#ff4d4f' }"
            />
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card :bordered="false">
            <a-statistic
              title="总需求数"
              :value="statistics?.total_requirements || 0"
              :value-style="{ color: '#52c41a' }"
            />
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card :bordered="false">
            <a-statistic
              title="项目成员"
              :value="statistics?.total_members || 0"
              suffix="人"
              :value-style="{ color: '#722ed1' }"
            />
          </a-card>
        </a-col>
      </a-row>

      <!-- 任务统计 -->
      <a-card title="任务统计" :bordered="false" style="margin-bottom: 16px">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-card class="stat-card" @click="handleGoToTasks('wait')">
              <a-statistic
                title="待办"
                :value="statistics?.todo_tasks || 0"
                :value-style="{ color: '#1890ff' }"
              />
            </a-card>
          </a-col>
          <a-col :span="8">
            <a-card class="stat-card" @click="handleGoToTasks('doing')">
              <a-statistic
                title="进行中"
                :value="statistics?.in_progress_tasks || 0"
                :value-style="{ color: '#faad14' }"
              />
            </a-card>
          </a-col>
          <a-col :span="8">
            <a-card class="stat-card" @click="handleGoToTasks('done')">
              <a-statistic
                title="已完成"
                :value="statistics?.done_tasks || 0"
                :value-style="{ color: '#52c41a' }"
              />
            </a-card>
          </a-col>
        </a-row>
      </a-card>

      <!-- Bug统计 -->
      <a-card title="Bug统计" :bordered="false" style="margin-bottom: 16px">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-card class="stat-card" @click="handleGoToBugs('open')">
              <a-statistic
                title="待处理"
                :value="statistics?.open_bugs || 0"
                :value-style="{ color: '#ff4d4f' }"
              />
            </a-card>
          </a-col>
          <a-col :span="8">
            <a-card class="stat-card" @click="handleGoToBugs('in_progress')">
              <a-statistic
                title="处理中"
                :value="statistics?.in_progress_bugs || 0"
                :value-style="{ color: '#faad14' }"
              />
            </a-card>
          </a-col>
          <a-col :span="8">
            <a-card class="stat-card" @click="handleGoToBugs('resolved')">
              <a-statistic
                title="已解决"
                :value="statistics?.resolved_bugs || 0"
                :value-style="{ color: '#52c41a' }"
              />
            </a-card>
          </a-col>
        </a-row>
      </a-card>

      <!-- 需求统计 -->
      <a-card title="需求统计" :bordered="false" style="margin-bottom: 16px">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-card class="stat-card" @click="handleGoToRequirements('in_progress')">
              <a-statistic
                title="进行中"
                :value="statistics?.in_progress_requirements || 0"
                :value-style="{ color: '#faad14' }"
              />
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card class="stat-card" @click="handleGoToRequirements('completed')">
              <a-statistic
                title="已完成"
                :value="statistics?.completed_requirements || 0"
                :value-style="{ color: '#52c41a' }"
              />
            </a-card>
          </a-col>
        </a-row>
      </a-card>

      <!-- 项目成员 -->
      <a-card title="项目成员" :bordered="false">
        <template #extra>
          <a-button type="link" @click="handleManageMembers">成员管理</a-button>
        </template>
        <a-list
          :data-source="project?.members || []"
          :loading="loading"
        >
          <template #renderItem="{ item }">
            <a-list-item>
              <a-list-item-meta>
                <template #avatar>
                  <a-avatar :src="item.user?.avatar">
                    {{ (item.user?.nickname || item.user?.username)?.charAt(0).toUpperCase() }}
                  </a-avatar>
                </template>
                <template #title>
                  {{ item.user?.username }}{{ item.user?.nickname ? `(${item.user.nickname})` : '' }}
                </template>
                <template #description>
                  <a-tag>{{ item.role }}</a-tag>
                  <span v-if="item.user?.department" style="margin-left: 8px; color: #999">
                    {{ item.user.department.name }}
                  </span>
                </template>
              </a-list-item-meta>
            </a-list-item>
          </template>
        </a-list>
      </a-card>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { formatDateTime } from '@/utils/date'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import type { Project, ProjectStatistics, Action } from '@/api/project'

interface Props {
  project?: Project
  statistics?: ProjectStatistics
  loading?: boolean
  historyList?: Action[]
  historyLoading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  historyList: () => [],
  historyLoading: false
})

const emit = defineEmits<{
  addNote: []
  goToTasks: [status: string]
  goToBugs: [status: string]
  goToRequirements: [status: string]
  manageMembers: []
}>()

const expandedHistoryIds = ref<Set<number>>(new Set())

// 获取项目负责人
const getProjectOwner = (): string => {
  if (!props.project?.members) return ''
  const owner = props.project.members.find(m => m.role === 'owner')
  if (!owner?.user) return ''
  return `${owner.user.username}${owner.user.nickname ? `(${owner.user.nickname})` : ''}`
}

// 获取字段显示名称
const getFieldDisplayName = (fieldName: string): string => {
  const fieldNames: Record<string, string> = {
    name: '项目名称',
    code: '项目编码',
    description: '项目描述',
    status: '状态',
    start_date: '开始日期',
    end_date: '结束日期',
    tag_ids: '标签'
  }
  return fieldNames[fieldName] || fieldName
}

// 获取操作描述
const getActionDescription = (action: Action): string => {
  const actorName = action.actor
    ? `${action.actor.username}${action.actor.nickname ? `(${action.actor.nickname})` : ''}`
    : '系统'

  switch (action.action) {
    case 'created':
      return `由 ${actorName} 创建。`
    case 'edited':
      return `由 ${actorName} 编辑。`
    case 'commented':
      return `由 ${actorName} 添加了备注：${action.comment || ''}`
    default:
      return `由 ${actorName} 执行了 ${action.action} 操作。`
  }
}

// 判断历史记录是否有详情
const hasHistoryDetails = (action: Action): boolean => {
  return !!(action.histories && action.histories.length > 0) || !!action.comment
}

// 切换历史记录详情展开/收起
const toggleHistoryDetail = (actionId: number) => {
  const newSet = new Set(expandedHistoryIds.value)
  if (newSet.has(actionId)) {
    newSet.delete(actionId)
  } else {
    newSet.add(actionId)
  }
  expandedHistoryIds.value = newSet
}

// 事件处理
const handleAddNote = () => {
  emit('addNote')
}

const handleGoToTasks = (status: string) => {
  emit('goToTasks', status)
}

const handleGoToBugs = (status: string) => {
  emit('goToBugs', status)
}

const handleGoToRequirements = (status: string) => {
  emit('goToRequirements', status)
}

const handleManageMembers = () => {
  emit('manageMembers')
}
</script>

<style scoped>
.project-detail-content {
  width: 100%;
}

.description-content {
  max-height: 300px;
  overflow-y: auto;
}

.markdown-content {
  max-height: 500px;
  overflow-y: auto;
}

.stat-card {
  cursor: pointer;
  transition: all 0.3s;
}

.stat-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}
</style>

