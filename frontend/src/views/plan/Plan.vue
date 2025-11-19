<template>
  <div class="plan-management">
    <a-layout>
      <AppHeader />
      <a-layout-content class="content">
        <div class="content-inner">
          <a-page-header title="计划管理">
            <template #extra>
              <a-button type="primary" @click="handleCreate">
                <template #icon><PlusOutlined /></template>
                新增计划
              </a-button>
            </template>
          </a-page-header>

          <a-card :bordered="false" style="margin-bottom: 16px">
            <a-form layout="inline" :model="searchForm">
              <a-form-item label="关键词">
                <a-input
                  v-model:value="searchForm.keyword"
                  placeholder="计划名称/描述"
                  allow-clear
                  style="width: 200px"
                />
              </a-form-item>
              <a-form-item label="状态">
                <a-select
                  v-model:value="searchForm.status"
                  placeholder="选择状态"
                  allow-clear
                  style="width: 120px"
                >
                  <a-select-option value="draft">草稿</a-select-option>
                  <a-select-option value="active">进行中</a-select-option>
                  <a-select-option value="completed">已完成</a-select-option>
                  <a-select-option value="cancelled">已取消</a-select-option>
                </a-select>
              </a-form-item>
              <a-form-item label="项目">
                <a-select
                  v-model:value="searchForm.project_id"
                  placeholder="选择项目"
                  allow-clear
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

          <a-card :bordered="false">
            <a-table
              :columns="columns"
              :data-source="plans"
              :loading="loading"
              :pagination="pagination"
              row-key="id"
              @change="handleTableChange"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'type'">
                  <a-tag color="green">
                    项目计划
                  </a-tag>
                </template>
                <template v-else-if="column.key === 'status'">
                  <a-tag :color="getStatusColor(record.status)">
                    {{ getStatusText(record.status) }}
                  </a-tag>
                </template>
                <template v-else-if="column.key === 'project'">
                  {{ record.project?.name || '-' }}
                </template>
                <template v-else-if="column.key === 'progress'">
                  <a-progress :percent="record.progress || 0" :status="record.status === 'completed' ? 'success' : 'active'" />
                </template>
                <template v-else-if="column.key === 'executions'">
                  <a-badge :count="record.executions?.length || 0" :number-style="{ backgroundColor: '#52c41a' }">
                    <a-button type="link" size="small" @click="handleViewExecutions(record)">
                      执行
                    </a-button>
                  </a-badge>
                </template>
                <template v-else-if="column.key === 'created_at'">
                  {{ formatDateTime(record.created_at) }}
                </template>
                <template v-else-if="column.key === 'action'">
                  <a-space>
                    <a-button type="link" size="small" @click="handleView(record)">
                      详情
                    </a-button>
                    <a-button type="link" size="small" @click="handleEdit(record)">
                      编辑
                    </a-button>
                    <a-button type="link" size="small" @click="handleManageExecutions(record)">
                      管理执行
                    </a-button>
                    <a-dropdown>
                      <a-button type="link" size="small">
                        状态 <DownOutlined />
                      </a-button>
                      <template #overlay>
                        <a-menu @click="(e: any) => handleStatusChange(record.id, e.key as string)">
                          <a-menu-item key="draft">草稿</a-menu-item>
                          <a-menu-item key="active">进行中</a-menu-item>
                          <a-menu-item key="completed">已完成</a-menu-item>
                          <a-menu-item key="cancelled">已取消</a-menu-item>
                        </a-menu>
                      </template>
                    </a-dropdown>
                    <a-popconfirm
                      title="确定要删除这个计划吗？"
                      @confirm="handleDelete(record.id)"
                    >
                      <a-button type="link" size="small" danger>删除</a-button>
                    </a-popconfirm>
                  </a-space>
                </template>
              </template>
            </a-table>
          </a-card>
        </div>
      </a-layout-content>
    </a-layout>

    <!-- 计划编辑/创建模态框 -->
    <a-modal
      v-model:open="modalVisible"
      :title="modalTitle"
      :width="900"
      @ok="handleSubmit"
      @cancel="handleCancel"
    >
      <a-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="计划名称" name="name">
          <a-input v-model:value="formData.name" placeholder="请输入计划名称" />
        </a-form-item>
        <a-form-item label="计划描述" name="description">
          <MarkdownEditor
            v-model="formData.description"
            placeholder="请输入计划描述（支持Markdown）"
            :rows="8"
          />
        </a-form-item>
        <a-form-item label="项目" name="project_id">
          <a-select
            v-model:value="formData.project_id"
            placeholder="选择项目"
            show-search
            :filter-option="filterProjectOption"
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
        <a-form-item label="状态" name="status">
          <a-select v-model:value="formData.status">
            <a-select-option value="draft">草稿</a-select-option>
            <a-select-option value="active">进行中</a-select-option>
            <a-select-option value="completed">已完成</a-select-option>
            <a-select-option value="cancelled">已取消</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="开始日期" name="start_date">
          <a-date-picker
            v-model:value="formData.start_date"
            placeholder="选择开始日期"
            style="width: 100%"
            format="YYYY-MM-DD"
          />
        </a-form-item>
        <a-form-item label="结束日期" name="end_date">
          <a-date-picker
            v-model:value="formData.end_date"
            placeholder="选择结束日期"
            style="width: 100%"
            format="YYYY-MM-DD"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 执行管理模态框 -->
    <a-modal
      v-model:open="executionModalVisible"
      title="管理计划执行"
      :width="1000"
      @ok="handleExecutionModalOk"
      @cancel="handleExecutionModalCancel"
    >
      <div v-if="selectedPlan">
        <a-space style="margin-bottom: 16px">
          <a-button type="primary" @click="handleCreateExecution">
            <template #icon><PlusOutlined /></template>
            新增执行
          </a-button>
        </a-space>
        <a-table
          :columns="executionColumns"
          :data-source="executions"
          :loading="executionLoading"
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
            <template v-else-if="column.key === 'action'">
              <a-space>
                <a-button type="link" size="small" @click="handleEditExecution(record)">
                  编辑
                </a-button>
                <a-button type="link" size="small" @click="handleUpdateExecutionProgress(record)">
                  进度
                </a-button>
                <a-dropdown>
                  <a-button type="link" size="small">
                    状态 <DownOutlined />
                  </a-button>
                  <template #overlay>
                    <a-menu @click="(e: any) => handleExecutionStatusChange(record.id, e.key as string)">
                      <a-menu-item key="pending">待处理</a-menu-item>
                      <a-menu-item key="in_progress">进行中</a-menu-item>
                      <a-menu-item key="completed">已完成</a-menu-item>
                      <a-menu-item key="cancelled">已取消</a-menu-item>
                    </a-menu>
                  </template>
                </a-dropdown>
                <a-popconfirm
                  title="确定要删除这个执行吗？"
                  @confirm="handleDeleteExecution(record.id)"
                >
                  <a-button type="link" size="small" danger>删除</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </template>
        </a-table>
      </div>
    </a-modal>

    <!-- 执行编辑/创建模态框 -->
    <a-modal
      v-model:open="executionEditModalVisible"
      :title="executionModalTitle"
      :width="800"
      @ok="handleExecutionSubmit"
      @cancel="handleExecutionCancel"
    >
      <a-form
        ref="executionFormRef"
        :model="executionFormData"
        :rules="executionFormRules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="执行名称" name="name">
          <a-input v-model:value="executionFormData.name" placeholder="请输入执行名称" />
        </a-form-item>
        <a-form-item label="执行描述" name="description">
          <a-textarea v-model:value="executionFormData.description" placeholder="请输入执行描述" :rows="4" />
        </a-form-item>
        <a-form-item label="状态" name="status">
          <a-select v-model:value="executionFormData.status">
            <a-select-option value="pending">待处理</a-select-option>
            <a-select-option value="in_progress">进行中</a-select-option>
            <a-select-option value="completed">已完成</a-select-option>
            <a-select-option value="cancelled">已取消</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="负责人" name="assignee_id">
          <a-select
            v-model:value="executionFormData.assignee_id"
            placeholder="选择负责人（可选）"
            allow-clear
            show-search
            :filter-option="filterUserOption"
          >
            <a-select-option
              v-for="user in users"
              :key="user.id"
              :value="user.id"
            >
              {{ user.username }}{{ user.nickname ? `(${user.nickname})` : '' }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="关联任务" name="task_id">
          <a-select
            v-model:value="executionFormData.task_id"
            placeholder="选择任务（可选）"
            allow-clear
            show-search
            :filter-option="filterTaskOption"
            :loading="taskLoading"
            @focus="loadTasksForProject"
          >
            <a-select-option
              v-for="task in availableTasks"
              :key="task.id"
              :value="task.id"
            >
              {{ task.title }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="开始日期" name="start_date">
          <a-date-picker
            v-model:value="executionFormData.start_date"
            placeholder="选择开始日期"
            style="width: 100%"
            format="YYYY-MM-DD"
          />
        </a-form-item>
        <a-form-item label="结束日期" name="end_date">
          <a-date-picker
            v-model:value="executionFormData.end_date"
            placeholder="选择结束日期"
            style="width: 100%"
            format="YYYY-MM-DD"
          />
        </a-form-item>
        <a-form-item label="进度" name="progress">
          <a-slider
            v-model:value="executionFormData.progress"
            :min="0"
            :max="100"
            :marks="{ 0: '0%', 50: '50%', 100: '100%' }"
          />
          <span style="margin-left: 8px">{{ executionFormData.progress }}%</span>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 更新执行进度模态框 -->
    <a-modal
      v-model:open="executionProgressModalVisible"
      title="更新执行进度"
      @ok="handleExecutionProgressSubmit"
      @cancel="handleExecutionProgressCancel"
    >
      <a-form
        ref="executionProgressFormRef"
        :model="executionProgressFormData"
        :rules="executionProgressFormRules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="进度" name="progress">
          <a-slider
            v-model:value="executionProgressFormData.progress"
            :min="0"
            :max="100"
            :marks="{ 0: '0%', 50: '50%', 100: '100%' }"
          />
          <span style="margin-left: 8px">{{ executionProgressFormData.progress }}%</span>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { PlusOutlined, DownOutlined } from '@ant-design/icons-vue'
import dayjs, { type Dayjs } from 'dayjs'
import { formatDateTime } from '@/utils/date'
import AppHeader from '@/components/AppHeader.vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import {
  getPlans,
  createPlan,
  updatePlan,
  deletePlan,
  updatePlanStatus,
  getPlanExecutions,
  createPlanExecution,
  updatePlanExecution,
  deletePlanExecution,
  updatePlanExecutionStatus,
  updatePlanExecutionProgress,
  type Plan,
  type PlanExecution,
  type CreatePlanRequest
} from '@/api/plan'
import { getProjects, type Project } from '@/api/project'
import { getUsers, type User } from '@/api/user'
import { getTasks, type Task } from '@/api/task'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const plans = ref<Plan[]>([])
const projects = ref<Project[]>([])
const users = ref<User[]>([])
const availableTasks = ref<Task[]>([])
const taskLoading = ref(false)

const searchForm = reactive({
  keyword: '',
  status: undefined as string | undefined,
  project_id: undefined as number | undefined
})

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showTotal: (total: number) => `共 ${total} 条`,
  showSizeChanger: true,
  showQuickJumper: true
})

const columns = [
  { title: '计划名称', dataIndex: 'name', key: 'name', ellipsis: true },
  { title: '类型', key: 'type', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '项目', key: 'project', width: 120 },
  { title: '进度', key: 'progress', width: 150 },
  { title: '执行', key: 'executions', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', key: 'action', width: 400, fixed: 'right' as const }
]

const modalVisible = ref(false)
const modalTitle = ref('新增计划')
const formRef = ref()
const formData = reactive<CreatePlanRequest & { id?: number; start_date?: Dayjs; end_date?: Dayjs }>({
  name: '',
  description: '',
  type: 'project_plan',
  status: 'draft',
  project_id: undefined,
  start_date: undefined,
  end_date: undefined
})

const formRules = {
  name: [{ required: true, message: '请输入计划名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择计划类型', trigger: 'change' }],
  project_id: [{ required: true, message: '请选择项目', trigger: 'change' }]
}

// 执行管理相关
const executionModalVisible = ref(false)
const selectedPlan = ref<Plan | null>(null)
const executions = ref<PlanExecution[]>([])
const executionLoading = ref(false)
const executionColumns = [
  { title: '执行名称', dataIndex: 'name', key: 'name' },
  { title: '状态', key: 'status', width: 100 },
  { title: '负责人', key: 'assignee', width: 150 },
  { title: '关联任务', key: 'task', width: 200 },
  { title: '进度', key: 'progress', width: 150 },
  { title: '开始日期', dataIndex: 'start_date', key: 'start_date', width: 120 },
  { title: '结束日期', dataIndex: 'end_date', key: 'end_date', width: 120 },
  { title: '操作', key: 'action', width: 300 }
]

const executionEditModalVisible = ref(false)
const executionModalTitle = ref('新增执行')
const executionFormRef = ref()
const executionFormData = reactive<{
  id?: number
  name: string
  description: string
  status: string
  progress: number
  assignee_id?: number
  task_id?: number
  start_date?: Dayjs
  end_date?: Dayjs
}>({
  name: '',
  description: '',
  status: 'pending',
  progress: 0,
  assignee_id: undefined,
  task_id: undefined,
  start_date: undefined,
  end_date: undefined
})

const executionFormRules = {
  name: [{ required: true, message: '请输入执行名称', trigger: 'blur' }]
}

const executionProgressModalVisible = ref(false)
const executionProgressFormRef = ref()
const executionProgressFormData = reactive({
  execution_id: 0,
  progress: 0
})

const executionProgressFormRules = {
  progress: [{ required: true, message: '请设置进度', trigger: 'change' }]
}

// 加载计划列表
const loadPlans = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.current,
      page_size: pagination.pageSize
    }
    if (searchForm.keyword) {
      params.keyword = searchForm.keyword
    }
    if (searchForm.status) {
      params.status = searchForm.status
    }
    if (searchForm.project_id) {
      params.project_id = searchForm.project_id
    }
    const response = await getPlans(params)
    plans.value = response.list
    pagination.total = response.total
  } catch (error: any) {
    message.error(error.message || '加载计划列表失败')
  } finally {
    loading.value = false
  }
}

// 加载项目列表
const loadProjects = async () => {
  try {
    const response = await getProjects()
    projects.value = response.list || []
  } catch (error: any) {
    console.error('加载项目列表失败:', error)
  }
}

// 加载用户列表
const loadUsers = async () => {
  try {
    const response = await getUsers()
    users.value = response.list || []
  } catch (error: any) {
    console.error('加载用户列表失败:', error)
  }
}

// 加载任务列表（用于执行关联任务）
const loadTasksForProject = async () => {
  if (!selectedPlan.value) {
    availableTasks.value = []
    return
  }
  const projectId = selectedPlan.value.project_id
  if (!projectId) {
    availableTasks.value = []
    return
  }
  taskLoading.value = true
  try {
    const response = await getTasks({ project_id: projectId })
    availableTasks.value = response.list
  } catch (error: any) {
    console.error('加载任务列表失败:', error)
  } finally {
    taskLoading.value = false
  }
}


// 搜索
const handleSearch = () => {
  pagination.current = 1
  loadPlans()
}

// 重置
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.status = undefined
  searchForm.project_id = undefined
  pagination.current = 1
  loadPlans()
}

// 表格变化
const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadPlans()
}

// 创建
const handleCreate = () => {
  modalTitle.value = '新增计划'
  formData.id = undefined
  formData.name = ''
  formData.description = ''
  formData.type = 'project_plan'
  formData.status = 'draft'
  formData.project_id = undefined
  formData.start_date = undefined
  formData.end_date = undefined
  modalVisible.value = true
}

// 编辑
const handleEdit = (record: Plan) => {
  modalTitle.value = '编辑计划'
  formData.id = record.id
  formData.name = record.name
  formData.description = record.description || ''
  formData.type = record.type
  formData.status = record.status
  formData.project_id = record.project_id
  formData.start_date = record.start_date ? dayjs(record.start_date) : undefined
  formData.end_date = record.end_date ? dayjs(record.end_date) : undefined
  modalVisible.value = true
}

// 查看详情
const handleView = (record: Plan) => {
  router.push(`/plan/${record.id}`)
}

// 查看执行
const handleViewExecutions = (record: Plan) => {
  handleManageExecutions(record)
}

// 管理执行
const handleManageExecutions = async (record: Plan) => {
  selectedPlan.value = record
  executionModalVisible.value = true
  await loadExecutions(record.id)
  if (record.project_id) {
    loadTasksForProject()
  }
}

// 加载执行列表
const loadExecutions = async (planId: number) => {
  executionLoading.value = true
  try {
    executions.value = await getPlanExecutions(planId)
  } catch (error: any) {
    message.error(error.message || '加载执行列表失败')
  } finally {
    executionLoading.value = false
  }
}

// 提交
const handleSubmit = async () => {
  try {
    await formRef.value.validate()
    const data: CreatePlanRequest = {
      name: formData.name,
      description: formData.description,
      type: formData.type,
      status: formData.status,
      project_id: formData.project_id,
      start_date: formData.start_date ? formData.start_date.format('YYYY-MM-DD') : undefined,
      end_date: formData.end_date ? formData.end_date.format('YYYY-MM-DD') : undefined
    }
    if (formData.id) {
      await updatePlan(formData.id, data)
      message.success('更新成功')
    } else {
      await createPlan(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadPlans()
  } catch (error: any) {
    if (error.errorFields) {
      return
    }
    message.error(error.message || '操作失败')
  }
}

// 取消
const handleCancel = () => {
  formRef.value?.resetFields()
}

// 删除
const handleDelete = async (id: number) => {
  try {
    await deletePlan(id)
    message.success('删除成功')
    loadPlans()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

// 状态变更
const handleStatusChange = async (id: number, status: string) => {
  try {
    await updatePlanStatus(id, { status: status as any })
    message.success('状态更新成功')
    loadPlans()
  } catch (error: any) {
    message.error(error.message || '状态更新失败')
  }
}

// 创建执行
const handleCreateExecution = () => {
  if (!selectedPlan.value) return
  executionModalTitle.value = '新增执行'
  executionFormData.id = undefined
  executionFormData.name = ''
  executionFormData.description = ''
  executionFormData.status = 'pending'
  executionFormData.progress = 0
  executionFormData.assignee_id = undefined
  executionFormData.task_id = undefined
  executionFormData.start_date = undefined
  executionFormData.end_date = undefined
  executionEditModalVisible.value = true
}

// 编辑执行
const handleEditExecution = (record: PlanExecution) => {
  executionModalTitle.value = '编辑执行'
  executionFormData.id = record.id
  executionFormData.name = record.name
  executionFormData.description = record.description || ''
  executionFormData.status = record.status
  executionFormData.progress = record.progress
  executionFormData.assignee_id = record.assignee_id
  executionFormData.task_id = record.task_id
  executionFormData.start_date = record.start_date ? dayjs(record.start_date) : undefined
  executionFormData.end_date = record.end_date ? dayjs(record.end_date) : undefined
  executionEditModalVisible.value = true
}

// 执行提交
const handleExecutionSubmit = async () => {
  if (!selectedPlan.value) return
  try {
    await executionFormRef.value.validate()
    const data: any = {
      name: executionFormData.name,
      description: executionFormData.description,
      status: executionFormData.status,
      progress: executionFormData.progress,
      assignee_id: executionFormData.assignee_id,
      task_id: executionFormData.task_id,
      start_date: executionFormData.start_date ? executionFormData.start_date.format('YYYY-MM-DD') : undefined,
      end_date: executionFormData.end_date ? executionFormData.end_date.format('YYYY-MM-DD') : undefined
    }
    if (executionFormData.id) {
      await updatePlanExecution(selectedPlan.value.id, executionFormData.id, data)
      message.success('更新成功')
    } else {
      await createPlanExecution(selectedPlan.value.id, data)
      message.success('创建成功')
    }
    executionEditModalVisible.value = false
    await loadExecutions(selectedPlan.value.id)
  } catch (error: any) {
    if (error.errorFields) {
      return
    }
    message.error(error.message || '操作失败')
  }
}

// 执行取消
const handleExecutionCancel = () => {
  executionFormRef.value?.resetFields()
}

// 执行模态框确定
const handleExecutionModalOk = () => {
  executionModalVisible.value = false
}

// 执行模态框取消
const handleExecutionModalCancel = () => {
  selectedPlan.value = null
  executions.value = []
}

// 删除执行
const handleDeleteExecution = async (executionId: number) => {
  if (!selectedPlan.value) return
  try {
    await deletePlanExecution(selectedPlan.value.id, executionId)
    message.success('删除成功')
    await loadExecutions(selectedPlan.value.id)
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

// 执行状态变更
const handleExecutionStatusChange = async (executionId: number, status: string) => {
  if (!selectedPlan.value) return
  try {
    await updatePlanExecutionStatus(selectedPlan.value.id, executionId, { status: status as any })
    message.success('状态更新成功')
    await loadExecutions(selectedPlan.value.id)
  } catch (error: any) {
    message.error(error.message || '状态更新失败')
  }
}

// 更新执行进度
const handleUpdateExecutionProgress = (record: PlanExecution) => {
  executionProgressFormData.execution_id = record.id
  executionProgressFormData.progress = record.progress
  executionProgressModalVisible.value = true
}

// 执行进度提交
const handleExecutionProgressSubmit = async () => {
  if (!selectedPlan.value) return
  try {
    await executionProgressFormRef.value.validate()
    await updatePlanExecutionProgress(
      selectedPlan.value.id,
      executionProgressFormData.execution_id,
      { progress: executionProgressFormData.progress }
    )
    message.success('进度更新成功')
    executionProgressModalVisible.value = false
    await loadExecutions(selectedPlan.value.id)
  } catch (error: any) {
    if (error.errorFields) {
      return
    }
    message.error(error.message || '进度更新失败')
  }
}

// 执行进度取消
const handleExecutionProgressCancel = () => {
  executionProgressFormRef.value?.resetFields()
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

// 项目筛选
const filterProjectOption = (input: string, option: any) => {
  const project = projects.value.find(p => p.id === option.value)
  if (!project) return false
  const searchText = input.toLowerCase()
  return (
    project.name.toLowerCase().includes(searchText) ||
    (project.code && project.code.toLowerCase().includes(searchText))
  )
}

// 任务筛选
const filterTaskOption = (input: string, option: any) => {
  const task = availableTasks.value.find(t => t.id === option.value)
  if (!task) return false
  const searchText = input.toLowerCase()
  return task.title.toLowerCase().includes(searchText)
}

// 用户筛选
const filterUserOption = (input: string, option: any) => {
  const user = users.value.find(u => u.id === option.value)
  if (!user) return false
  const searchText = input.toLowerCase()
  return (
    user.username.toLowerCase().includes(searchText) ||
    (user.nickname && user.nickname.toLowerCase().includes(searchText))
  )
}

onMounted(() => {
  loadPlans()
  loadProjects()
  loadUsers()
})
</script>

<style scoped>
.plan-management {
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
</style>

