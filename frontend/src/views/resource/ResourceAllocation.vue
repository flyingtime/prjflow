<template>
  <div class="resource-allocation">
    <a-layout>
      <AppHeader />
      <a-layout-content class="content">
        <div class="content-inner">
          <a-page-header title="资源分配">
            <template #extra>
              <a-space>
                <a-button @click="handlePeriodChange('week')" :type="period === 'week' ? 'primary' : 'default'">
                  本周
                </a-button>
                <a-button @click="handlePeriodChange('month')" :type="period === 'month' ? 'primary' : 'default'">
                  本月
                </a-button>
                <a-button @click="handlePeriodChange('all')" :type="period === 'all' ? 'primary' : 'default'">
                  全部
                </a-button>
              </a-space>
            </template>
          </a-page-header>

          <a-card :bordered="false" style="margin-bottom: 16px">
            <template #title>
              <a-space>
                <span>搜索条件</span>
                <a-button type="text" size="small" @click="toggleSearchForm">
                  <template #icon>
                    <UpOutlined v-if="searchFormVisible" />
                    <DownOutlined v-else />
                  </template>
                  {{ searchFormVisible ? '收起' : '展开' }}
                </a-button>
              </a-space>
            </template>
            <a-form v-show="searchFormVisible" layout="inline" :model="searchForm">
              <a-form-item label="用户">
                <a-select
                  v-model:value="searchForm.user_id"
                  placeholder="选择用户"
                  allow-clear
                  show-search
                  :filter-option="filterUserOption"
                  style="width: 150px"
                >
                  <a-select-option
                    v-for="user in users"
                    :key="user.id"
                    :value="user.id"
                  >
                    {{ user.nickname || user.username }}({{ user.username }})
                  </a-select-option>
                </a-select>
              </a-form-item>
              <a-form-item label="项目">
                <a-select
                  v-model:value="searchForm.project_id"
                  placeholder="选择项目"
                  allow-clear
                  show-search
                  :filter-option="filterProjectOption"
                  style="width: 150px"
                >
                  <a-select-option
                    v-for="project in projects"
                    :key="project.id"
                    :value="project.id"
                  >
                    {{ project.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
              <a-form-item>
                <a-button type="primary" @click="handleSearch">查询</a-button>
                <a-button style="margin-left: 8px" @click="handleReset">重置</a-button>
              </a-form-item>
            </a-form>
          </a-card>

          <a-card :bordered="false" class="table-card">
            <a-table
              :scroll="{ x: 'max-content', y: tableScrollHeight }"
              :columns="columns"
              :data-source="allocations"
              :loading="loading"
              :pagination="pagination"
              row-key="id"
              @change="handleTableChange"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'date'">
                  {{ formatDate(record.date) }}
                </template>
                <template v-else-if="column.key === 'user'">
                  {{ record.resource?.user?.nickname || record.resource?.user?.username || '-' }}
                  <span v-if="record.resource?.user?.username" style="color: #999">
                    ({{ record.resource.user.username }})
                  </span>
                </template>
                <template v-else-if="column.key === 'project'">
                  <span v-if="record.project">{{ record.project.name }}</span>
                  <span v-else-if="record.resource?.project">{{ record.resource.project.name }}</span>
                  <span v-else>-</span>
                </template>
                <template v-else-if="column.key === 'task'">
                  <span v-if="record.task">{{ record.task.name }}</span>
                  <span v-else>-</span>
                </template>
                <template v-else-if="column.key === 'bug'">
                  <span v-if="record.bug">{{ record.bug.title }}</span>
                  <span v-else>-</span>
                </template>
                <template v-else-if="column.key === 'hours'">
                  {{ record.hours }} 小时
                </template>
                <template v-else-if="column.key === 'description'">
                  <span v-if="record.description">{{ record.description }}</span>
                  <span v-else>-</span>
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
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import AppHeader from '@/components/AppHeader.vue'
import { DownOutlined, UpOutlined } from '@ant-design/icons-vue'
import {
  getResourceAllocations,
  type ResourceAllocation
} from '@/api/resource'
import { getUsers } from '@/api/user'
import { getProjects } from '@/api/project'
import type { User } from '@/api/user'
import type { Project } from '@/api/project'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const loading = ref(false)
const searchFormVisible = ref(false)
const users = ref<User[]>([])
const projects = ref<Project[]>([])
const allocations = ref<ResourceAllocation[]>([])

// 从路由参数获取周期
const period = ref<'week' | 'month' | 'all'>(
  (route.query.period as 'week' | 'month' | 'all') || 'week'
)

const searchForm = reactive({
  user_id: undefined as number | undefined,
  project_id: undefined as number | undefined
})

// 计算表格滚动高度
const tableScrollHeight = computed(() => {
  return 'calc(100vh - 500px)'
})

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showTotal: (total: number) => `共 ${total} 条`,
  showSizeChanger: true,
  showQuickJumper: true
})

const columns = [
  { title: '日期', key: 'date', width: 120, fixed: 'left' },
  { title: '用户', key: 'user', width: 150 },
  { title: '项目', key: 'project', width: 200 },
  { title: '工时', key: 'hours', width: 100 },
  { title: '关联任务', key: 'task', width: 200 },
  { title: '关联Bug', key: 'bug', width: 200 },
  { title: '工作描述', key: 'description', ellipsis: true }
]

const filterUserOption = (input: string, option: any) => {
  const user = users.value.find(u => u.id === option.value)
  if (!user) return false
  const nickname = user.nickname || ''
  const username = user.username || ''
  return nickname.toLowerCase().includes(input.toLowerCase()) ||
    username.toLowerCase().includes(input.toLowerCase())
}

const filterProjectOption = (input: string, option: any) => {
  const project = projects.value.find(p => p.id === option.value)
  if (!project) return false
  const searchText = input.toLowerCase()
  return (
    project.name.toLowerCase().includes(searchText) ||
    (project.code && project.code.toLowerCase().includes(searchText))
  )
}

// 计算日期范围
const getDateRange = () => {
  const now = dayjs()
  if (period.value === 'week') {
    const weekStart = now.startOf('week').add(1, 'day') // 周一开始
    const weekEnd = weekStart.add(6, 'days')
    return {
      start_date: weekStart.format('YYYY-MM-DD'),
      end_date: weekEnd.format('YYYY-MM-DD')
    }
  } else if (period.value === 'month') {
    return {
      start_date: now.startOf('month').format('YYYY-MM-DD'),
      end_date: now.endOf('month').format('YYYY-MM-DD')
    }
  } else {
    // 全部：不设置日期范围
    return {}
  }
}

const loadAllocations = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.current,
      size: pagination.pageSize
    }

    // 添加日期范围
    const dateRange = getDateRange()
    if (dateRange.start_date) {
      params.start_date = dateRange.start_date
    }
    if (dateRange.end_date) {
      params.end_date = dateRange.end_date
    }

    // 添加搜索条件
    if (searchForm.user_id) {
      params.user_id = searchForm.user_id
    }
    if (searchForm.project_id) {
      params.project_id = searchForm.project_id
    }

    // 如果是当前用户，默认只显示自己的
    const userId = authStore.user?.id
    if (userId && !searchForm.user_id) {
      params.user_id = userId
    }

    const res = await getResourceAllocations(params)
    allocations.value = res.list || []
    pagination.total = res.total || 0
  } catch (error: any) {
    message.error('加载资源分配列表失败: ' + (error.response?.data?.message || error.message))
  } finally {
    loading.value = false
  }
}

const loadUsers = async () => {
  try {
    const res = await getUsers({ size: 1000 })
    users.value = res.list || []
  } catch (error: any) {
    message.error('加载用户列表失败: ' + (error.response?.data?.message || error.message))
  }
}

const loadProjects = async () => {
  try {
    const res = await getProjects({ size: 1000 })
    projects.value = res.list || []
  } catch (error: any) {
    message.error('加载项目列表失败: ' + (error.response?.data?.message || error.message))
  }
}

const toggleSearchForm = () => {
  searchFormVisible.value = !searchFormVisible.value
}

const handleSearch = () => {
  pagination.current = 1
  loadAllocations()
}

const handleReset = () => {
  searchForm.user_id = undefined
  searchForm.project_id = undefined
  pagination.current = 1
  loadAllocations()
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadAllocations()
}

const handlePeriodChange = (p: 'week' | 'month' | 'all') => {
  period.value = p
  router.push({
    path: '/resource-allocation',
    query: { period: p }
  })
  pagination.current = 1
  loadAllocations()
}

const formatDate = (dateStr: string) => {
  return dayjs(dateStr).format('YYYY-MM-DD')
}

// 监听路由参数变化
watch(() => route.query.period, (newPeriod) => {
  if (newPeriod && (newPeriod === 'week' || newPeriod === 'month' || newPeriod === 'all')) {
    period.value = newPeriod
    pagination.current = 1
    loadAllocations()
  }
}, { immediate: true })

onMounted(() => {
  loadUsers()
  loadProjects()
  loadAllocations()
})
</script>

<style scoped>
.resource-allocation {
  min-height: 100vh;
}

.content {
  padding: 24px;
  background: #f0f2f5;
}

.content-inner {
  max-width: 100%;
  width: 100%;
  margin: 0 auto;
  background: #fff;
  padding: 24px;
  border-radius: 8px;
  overflow-y: auto;
}

.table-card {
  margin-bottom: 16px;
}
</style>

