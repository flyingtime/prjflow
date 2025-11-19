<template>
  <div class="plan-detail">
    <a-layout>
      <AppHeader />
      <a-layout-content class="content">
        <div class="content-inner">
          <a-page-header
            :title="plan?.name || '计划详情'"
            @back="() => router.push('/plan')"
          >
            <template #extra>
              <a-space>
                <a-button @click="handleEdit">编辑</a-button>
                <a-button @click="handleManageExecutions">管理执行</a-button>
              </a-space>
            </template>
          </a-page-header>

          <a-card v-if="plan" :bordered="false" style="margin-bottom: 16px">
            <a-descriptions :column="2" bordered>
              <a-descriptions-item label="计划名称">{{ plan.name }}</a-descriptions-item>
              <a-descriptions-item label="计划类型">
                <a-tag :color="plan.type === 'product_plan' ? 'blue' : 'green'">
                  {{ plan.type === 'product_plan' ? '产品计划' : '项目计划' }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="状态">
                <a-tag :color="getStatusColor(plan.status)">
                  {{ getStatusText(plan.status) }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="进度">
                <a-progress :percent="plan.progress || 0" :status="plan.status === 'completed' ? 'success' : 'active'" />
              </a-descriptions-item>
              <a-descriptions-item label="产品" v-if="plan.product">
                {{ plan.product.name }}
              </a-descriptions-item>
              <a-descriptions-item label="项目" v-if="plan.project">
                <a-button type="link" @click="router.push(`/project/${plan.project_id}`)">
                  {{ plan.project.name }}
                </a-button>
              </a-descriptions-item>
              <a-descriptions-item label="创建人">
                {{ plan.creator ? `${plan.creator.username}${plan.creator.nickname ? `(${plan.creator.nickname})` : ''}` : '-' }}
              </a-descriptions-item>
              <a-descriptions-item label="创建时间">
                {{ formatDateTime(plan.created_at) }}
              </a-descriptions-item>
              <a-descriptions-item label="开始日期" v-if="plan.start_date">
                {{ dayjs(plan.start_date).format('YYYY-MM-DD') }}
              </a-descriptions-item>
              <a-descriptions-item label="结束日期" v-if="plan.end_date">
                {{ dayjs(plan.end_date).format('YYYY-MM-DD') }}
              </a-descriptions-item>
            </a-descriptions>
          </a-card>

          <a-card v-if="plan" title="计划描述" :bordered="false" style="margin-bottom: 16px">
            <div v-if="plan.description" class="markdown-content" v-html="markdownToHtml(plan.description)"></div>
            <a-empty v-else description="暂无描述" />
          </a-card>

          <a-card v-if="plan" title="计划执行" :bordered="false">
            <a-table
              :columns="executionColumns"
              :data-source="plan.executions || []"
              row-key="id"
              :pagination="false"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'status'">
                  <a-tag :color="getExecutionStatusColor(record.status)">
                    {{ getExecutionStatusText(record.status) }}
                  </a-tag>
                </template>
                <template v-else-if="column.key === 'assignee'">
                  {{ record.assignee ? `${record.assignee.username}${record.assignee.nickname ? `(${record.assignee.nickname})` : ''}` : '-' }}
                </template>
                <template v-else-if="column.key === 'task'">
                  <a-button v-if="record.task" type="link" @click="router.push(`/task/${record.task.id}`)">
                    {{ record.task.title }}
                  </a-button>
                  <span v-else>-</span>
                </template>
                <template v-else-if="column.key === 'progress'">
                  <a-progress :percent="record.progress" :status="record.status === 'completed' ? 'success' : 'active'" />
                </template>
              </template>
            </a-table>
          </a-card>
        </div>
      </a-layout-content>
    </a-layout>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import { marked } from 'marked'
import { formatDateTime } from '@/utils/date'
import AppHeader from '@/components/AppHeader.vue'
import { getPlan, type Plan } from '@/api/plan'

const route = useRoute()
const router = useRouter()
const plan = ref<Plan | null>(null)

const executionColumns = [
  { title: '执行名称', dataIndex: 'name', key: 'name' },
  { title: '状态', key: 'status', width: 100 },
  { title: '负责人', key: 'assignee', width: 150 },
  { title: '关联任务', key: 'task', width: 200 },
  { title: '进度', key: 'progress', width: 150 },
  { title: '开始日期', dataIndex: 'start_date', key: 'start_date', width: 120 },
  { title: '结束日期', dataIndex: 'end_date', key: 'end_date', width: 120 }
]

// 加载计划详情
const loadPlan = async () => {
  const id = Number(route.params.id)
  if (!id) {
    message.error('计划ID无效')
    router.push('/plan')
    return
  }
  try {
    plan.value = await getPlan(id)
  } catch (error: any) {
    message.error(error.message || '加载计划详情失败')
    router.push('/plan')
  }
}

// 编辑
const handleEdit = () => {
  if (plan.value) {
    router.push(`/plan?edit=${plan.value.id}`)
  }
}

// 管理执行
const handleManageExecutions = () => {
  if (plan.value) {
    router.push(`/plan?manage_executions=${plan.value.id}`)
  }
}

// 获取状态颜色
const getStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    draft: 'default',
    active: 'blue',
    completed: 'green',
    cancelled: 'red'
  }
  return colors[status] || 'default'
}

// 获取状态文本
const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    draft: '草稿',
    active: '进行中',
    completed: '已完成',
    cancelled: '已取消'
  }
  return texts[status] || status
}

// 获取执行状态颜色
const getExecutionStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    pending: 'orange',
    in_progress: 'blue',
    completed: 'green',
    cancelled: 'red'
  }
  return colors[status] || 'default'
}

// 获取执行状态文本
const getExecutionStatusText = (status: string) => {
  const texts: Record<string, string> = {
    pending: '待处理',
    in_progress: '进行中',
    completed: '已完成',
    cancelled: '已取消'
  }
  return texts[status] || status
}

// Markdown转HTML
const markdownToHtml = (markdown: string) => {
  return marked(markdown || '')
}

onMounted(() => {
  loadPlan()
})
</script>

<style scoped>
.plan-detail {
  min-height: 100vh;
}

.content {
  padding: 24px;
  background: #f0f2f5;
}

.content-inner {
  max-width: 1400px;
  margin: 0 auto;
}

.markdown-content {
  line-height: 1.8;
}
</style>

