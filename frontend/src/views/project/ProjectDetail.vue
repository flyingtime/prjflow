<template>
  <div class="project-detail">
    <a-layout>
      <AppHeader />
      <a-layout-content class="content">
        <div class="content-inner">
          <a-page-header
            :title="project?.name || '项目详情'"
            :sub-title="project?.code"
            @back="() => router.push('/project')"
          >
            <template #extra>
              <a-space>
                <a-button @click="handleManageRequirements">需求管理</a-button>
                <a-button @click="handleManageTasks">任务管理</a-button>
                <a-button @click="handleManageBugs">Bug管理</a-button>
                <a-button @click="handleViewBoards">看板</a-button>
                <a-button @click="handleViewGantt">甘特图</a-button>
                <a-button @click="handleViewProgress">进度跟踪</a-button>
                <a-button @click="handleViewResourceStatistics">资源统计</a-button>
                <a-button @click="handleManageModules">功能模块</a-button>
                <a-button @click="handleEdit">编辑</a-button>
                <a-button @click="handleManageMembers">成员管理</a-button>
              </a-space>
            </template>
          </a-page-header>

          <ProjectDetailContent
            :project="project"
            :statistics="statistics"
            :loading="loading"
            :history-list="historyList"
            :history-loading="historyLoading"
            @add-note="handleAddNote"
            @go-to-tasks="goToTasks"
            @go-to-bugs="goToBugs"
            @go-to-requirements="goToRequirements"
            @manage-members="handleManageMembers"
          />
        </div>
      </a-layout-content>
    </a-layout>

    <!-- 项目成员管理对话框 -->
    <a-modal
      v-model:open="memberModalVisible"
      title="项目成员管理"
      :mask-closable="true"
      @cancel="handleCloseMemberModal"
      @ok="handleCloseMemberModal"
      ok-text="关闭"
      width="800px"
    >
      <a-spin :spinning="memberLoading">
        <div style="margin-bottom: 16px">
          <a-space>
            <a-select
              v-model:value="selectedUserIds"
              mode="multiple"
              placeholder="选择用户"
              style="width: 300px"
              show-search
              :filter-option="(input: string, option: any) => {
                const user = users.find(u => u.id === option.value)
                if (!user) return false
                const searchText = input.toLowerCase()
                return user.username.toLowerCase().includes(searchText) ||
                  (user.nickname && user.nickname.toLowerCase().includes(searchText))
              }"
            >
              <a-select-option
                v-for="user in users"
                :key="user.id"
                :value="user.id"
              >
                {{ user.username }}{{ user.nickname ? `(${user.nickname})` : '' }}
              </a-select-option>
            </a-select>
            <a-select
              v-model:value="memberRole"
              placeholder="选择角色"
              style="width: 150px"
            >
              <a-select-option value="owner">负责人</a-select-option>
              <a-select-option value="member">成员</a-select-option>
              <a-select-option value="viewer">查看者</a-select-option>
            </a-select>
            <a-button type="primary" @click="handleAddMembers">添加成员</a-button>
          </a-space>
        </div>
        <a-table
          :columns="memberColumns"
          :data-source="projectMembers"
          :scroll="{ x: 'max-content' }"
          row-key="id"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'user'">
              {{ record.user?.username || '-' }}{{ record.user?.nickname ? `(${record.user.nickname})` : '' }}
            </template>
            <template v-else-if="column.key === 'role'">
              <a-select
                :value="record.role"
                @change="(value: any) => handleUpdateMemberRole(record.id, value)"
                style="width: 120px"
              >
                <a-select-option value="owner">负责人</a-select-option>
                <a-select-option value="member">成员</a-select-option>
                <a-select-option value="viewer">查看者</a-select-option>
              </a-select>
            </template>
            <template v-else-if="column.key === 'action'">
              <a-popconfirm
                title="确定要移除这个成员吗？"
                @confirm="handleRemoveMember(record.id)"
              >
                <a-button type="link" size="small" danger>移除</a-button>
              </a-popconfirm>
            </template>
          </template>
        </a-table>
      </a-spin>
    </a-modal>

    <!-- 功能模块管理对话框 -->
    <a-modal
      v-model:open="moduleManageModalVisible"
      title="功能模块管理（系统资源）"
      :mask-closable="true"
      @cancel="handleCloseModuleModal"
      width="900px"
      :footer="null"
    >
      <ModuleManagement :show-card="false" />
    </a-modal>

    <!-- 项目编辑模态框 -->
    <a-modal
      v-model:open="editModalVisible"
      title="编辑项目"
      :width="800"
      :mask-closable="false"
      @ok="handleEditSubmit"
      @cancel="handleEditCancel"
    >
      <a-form
        ref="editFormRef"
        :model="editFormData"
        :rules="editFormRules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="项目名称" name="name">
          <a-input v-model:value="editFormData.name" placeholder="请输入项目名称" />
        </a-form-item>
        <a-form-item label="项目编码" name="code">
          <a-input v-model:value="editFormData.code" placeholder="请输入项目编码" />
        </a-form-item>
        <a-form-item label="负责人" name="owner_id">
          <ProjectMemberSelect
            v-model="editFormData.owner_id"
            :project-id="project?.id"
            :multiple="false"
            placeholder="选择负责人（可选）"
            :show-role="true"
            :show-hint="!project?.id"
          />
        </a-form-item>
        <a-form-item label="项目描述" name="description">
          <MarkdownEditor
            ref="editDescriptionEditorRef"
            v-model="editFormData.description"
            placeholder="请输入项目描述（支持Markdown）"
            :rows="8"
            :project-id="project?.id || 0"
          />
        </a-form-item>
        <a-form-item label="状态" name="status">
          <a-select v-model:value="editFormData.status">
            <a-select-option value="wait">等待</a-select-option>
            <a-select-option value="doing">进行中</a-select-option>
            <a-select-option value="suspended">已暂停</a-select-option>
            <a-select-option value="closed">已关闭</a-select-option>
            <a-select-option value="done">已完成</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="标签" name="tag_ids">
          <a-select
            v-model:value="editFormData.tag_ids"
            mode="multiple"
            placeholder="选择标签（支持多选）"
            allow-clear
            :options="tagOptions"
            :field-names="{ label: 'name', value: 'id' }"
          />
        </a-form-item>
        <a-form-item label="开始日期" name="start_date">
          <a-date-picker
            v-model:value="editFormData.start_date"
            placeholder="选择开始日期（可选）"
            style="width: 100%"
            format="YYYY-MM-DD"
          />
        </a-form-item>
        <a-form-item label="结束日期" name="end_date">
          <a-date-picker
            v-model:value="editFormData.end_date"
            placeholder="选择结束日期（可选）"
            style="width: 100%"
            format="YYYY-MM-DD"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 添加备注模态框 -->
    <a-modal
      v-model:open="noteModalVisible"
      title="添加备注"
      :mask-closable="true"
      @ok="handleNoteSubmit"
      @cancel="handleNoteCancel"
    >
      <a-form
        ref="noteFormRef"
        :model="noteFormData"
        :rules="noteFormRules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="备注" name="comment">
          <a-textarea
            v-model:value="noteFormData.comment"
            placeholder="请输入备注"
            :rows="4"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import dayjs, { type Dayjs } from 'dayjs'
import AppHeader from '@/components/AppHeader.vue'
import ModuleManagement from '@/components/ModuleManagement.vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import ProjectMemberSelect from '@/components/ProjectMemberSelect.vue'
import ProjectDetailContent from '@/components/ProjectDetailContent.vue'
import { 
  getProject, 
  updateProject,
  getProjectMembers,
  addProjectMembers,
  updateProjectMember,
  removeProjectMember,
  getProjectHistory,
  addProjectHistoryNote,
  type ProjectDetailResponse, 
  type Project,
  type ProjectMember,
  type CreateProjectRequest,
  type Action
} from '@/api/project'
import { getUsers, type User } from '@/api/user'
import { getTags, type Tag } from '@/api/tag'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const project = ref<Project>()
const statistics = ref<any>()

// 成员管理相关
const memberModalVisible = ref(false)
const memberLoading = ref(false)
const users = ref<User[]>([])
const projectMembers = ref<ProjectMember[]>([])
const selectedUserIds = ref<number[]>([])
const memberRole = ref('member')

// 功能模块管理相关
const moduleManageModalVisible = ref(false)

// 历史记录相关
const historyLoading = ref(false)
const historyList = ref<Action[]>([])
const noteModalVisible = ref(false)
const noteFormRef = ref()
const noteFormData = reactive({
  comment: ''
})
const noteFormRules = {
  comment: [{ required: true, message: '请输入备注', trigger: 'blur' }]
}

// 编辑模态框相关
const editModalVisible = ref(false)
const editFormRef = ref()
const editDescriptionEditorRef = ref<InstanceType<typeof MarkdownEditor> | null>(null)
const editFormData = reactive<Omit<CreateProjectRequest, 'start_date' | 'end_date'> & { 
  start_date?: Dayjs | undefined
  end_date?: Dayjs | undefined
  owner_id?: number | undefined
}>({
  name: '',
  code: '',
  description: '',
  status: 'wait',
  tag_ids: [],
  start_date: undefined,
  end_date: undefined,
  owner_id: undefined
})
const editFormRules = {
  name: [{ required: true, message: '请输入项目名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入项目编码', trigger: 'blur' }]
}
const tags = ref<Tag[]>([])
const tagOptions = ref<Array<{ id: number; name: string; color?: string }>>([])

const memberColumns = [
  { title: '用户', key: 'user', width: 150 },
  { title: '角色', key: 'role', width: 150 },
  { title: '操作', key: 'action', width: 100 }
]

// 加载项目详情
const loadProject = async () => {
  const projectId = Number(route.params.id)
  if (!projectId) {
    message.error('项目ID无效')
    router.push('/project')
    return
  }

  loading.value = true
  try {
    const data: ProjectDetailResponse = await getProject(projectId)
    project.value = data.project
    statistics.value = data.statistics
    await loadProjectHistory(projectId) // 加载历史记录
  } catch (error: any) {
    message.error(error.message || '加载项目详情失败')
    router.push('/project')
  } finally {
    loading.value = false
  }
}

// 查看看板
const handleViewBoards = () => {
  if (!project.value) return
  router.push(`/project/${project.value.id}/boards`)
}

// 查看甘特图
const handleViewGantt = () => {
  if (!project.value) return
  router.push(`/project/${project.value.id}/gantt`)
}

// 查看进度跟踪
const handleViewProgress = () => {
  if (!project.value) return
  router.push(`/project/${project.value.id}/progress`)
}

// 查看资源统计
const handleViewResourceStatistics = () => {
  if (!project.value) return
  router.push({
    path: '/resource/statistics',
    query: { project_id: project.value.id }
  })
}

// 编辑项目
const handleEdit = async () => {
  if (!project.value) return
  
  editFormData.name = project.value.name
  editFormData.code = project.value.code
  editFormData.description = project.value.description || ''
  editFormData.status = project.value.status
  editFormData.tag_ids = project.value.tags?.map(t => t.id) || []
  editFormData.start_date = project.value.start_date ? dayjs(project.value.start_date) : undefined
  editFormData.end_date = project.value.end_date ? dayjs(project.value.end_date) : undefined
  // 获取当前负责人ID
  const owner = project.value.members?.find(m => m.role === 'owner')
  editFormData.owner_id = owner?.user_id
  
  editModalVisible.value = true
  await loadTags()
}

// 编辑提交
const handleEditSubmit = async () => {
  if (!project.value) return
  
  try {
    await editFormRef.value.validate()
    
    // 获取最新的描述内容
    let description = editFormData.description || ''
    
    // 如果有项目ID，尝试上传本地图片（如果有的话）
    if (editDescriptionEditorRef.value && project.value.id) {
      try {
        const uploadedDescription = await editDescriptionEditorRef.value.uploadLocalImages(async (file: File, projectId: number) => {
          // TODO: 需要实现文件上传API
          const { uploadFile } = await import('@/api/attachment')
          const attachment = await uploadFile(file, projectId)
          return attachment
        })
        description = uploadedDescription
      } catch (error: any) {
        console.error('上传图片失败:', error)
        message.warning('部分图片上传失败，请检查')
        description = editFormData.description || ''
      }
    }
    
    const data: Partial<CreateProjectRequest> = {
      name: editFormData.name,
      code: editFormData.code,
      description: description || '',
      status: editFormData.status,
      tag_ids: editFormData.tag_ids,
      start_date: editFormData.start_date && typeof editFormData.start_date !== 'string' && 'isValid' in editFormData.start_date && (editFormData.start_date as Dayjs).isValid() ? (editFormData.start_date as Dayjs).format('YYYY-MM-DD') : (typeof editFormData.start_date === 'string' ? editFormData.start_date : undefined),
      end_date: editFormData.end_date && typeof editFormData.end_date !== 'string' && 'isValid' in editFormData.end_date && (editFormData.end_date as Dayjs).isValid() ? (editFormData.end_date as Dayjs).format('YYYY-MM-DD') : (typeof editFormData.end_date === 'string' ? editFormData.end_date : undefined)
    }
    
    await updateProject(project.value.id, data)
    
    // 处理负责人更新
    if (editFormData.owner_id !== undefined) {
      const oldOwner = project.value.members?.find(m => m.role === 'owner')
      const newOwnerId = editFormData.owner_id
      
      // 如果负责人改变了
      if (oldOwner?.user_id !== newOwnerId) {
        // 如果原来有负责人，将其角色改为member
        if (oldOwner) {
          await updateProjectMember(project.value.id, oldOwner.id, 'member')
        }
        
        // 如果选择了新的负责人，将其角色改为owner
        if (newOwnerId) {
          const newOwner = project.value.members?.find(m => m.user_id === newOwnerId)
          if (newOwner) {
            await updateProjectMember(project.value.id, newOwner.id, 'owner')
          }
        }
      }
    }
    
    message.success('更新成功')
    editModalVisible.value = false
    await loadProject() // 重新加载项目详情（会自动加载历史记录）
  } catch (error: any) {
    if (error.errorFields) {
      return
    }
    message.error(error.message || '更新失败')
  }
}

// 编辑取消
const handleEditCancel = () => {
  editFormRef.value?.resetFields()
}

// 加载标签
const loadTags = async () => {
  try {
    tags.value = await getTags()
    tagOptions.value = tags.value.map(t => ({ id: t.id, name: t.name, color: t.color }))
  } catch (error: any) {
    console.error('加载标签列表失败:', error)
  }
}

// 加载历史记录
const loadProjectHistory = async (projectId?: number) => {
  const id = projectId || Number(route.params.id)
  if (!id) return

  historyLoading.value = true
  try {
    const response = await getProjectHistory(id)
    historyList.value = response.list || []
  } catch (error: any) {
    console.error('加载历史记录失败:', error)
  } finally {
    historyLoading.value = false
  }
}

// 获取操作描述（用于备注模态框）
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

// 添加备注
const handleAddNote = () => {
  if (!project.value) {
    message.warning('项目信息未加载完成，请稍候再试')
    return
  }
  noteFormData.comment = ''
  noteModalVisible.value = true
}

// 提交备注
const handleNoteSubmit = async () => {
  if (!project.value) return
  try {
    await noteFormRef.value.validate()
    await addProjectHistoryNote(project.value.id, { comment: noteFormData.comment })
    message.success('添加备注成功')
    noteModalVisible.value = false
    await loadProjectHistory(project.value.id)
  } catch (error: any) {
    if (error.errorFields) {
      return
    }
    message.error(error.message || '添加备注失败')
  }
}

// 取消添加备注
const handleNoteCancel = () => {
  noteFormRef.value?.resetFields()
}

// 加载用户列表
const loadUsers = async () => {
  try {
    const response = await getUsers({ size: 1000 })
    users.value = response.list || []
  } catch (error: any) {
    console.error('加载用户列表失败:', error)
  }
}

// 加载项目成员
const loadProjectMembers = async (projectId: number) => {
  memberLoading.value = true
  try {
    projectMembers.value = await getProjectMembers(projectId)
  } catch (error: any) {
    message.error(error.message || '加载项目成员失败')
  } finally {
    memberLoading.value = false
  }
}

// 成员管理
const handleManageMembers = async () => {
  if (!project.value) return
  memberModalVisible.value = true
  selectedUserIds.value = []
  memberRole.value = 'member'
  await loadProjectMembers(project.value.id)
  if (users.value.length === 0) {
    await loadUsers()
  }
}

// 关闭成员管理对话框
const handleCloseMemberModal = async () => {
  memberModalVisible.value = false
  selectedUserIds.value = []
  memberRole.value = 'member'
  // 重新加载项目详情以更新成员列表
  if (project.value) {
    await loadProject()
  }
}

// 添加成员
const handleAddMembers = async () => {
  if (!project.value || selectedUserIds.value.length === 0) {
    message.warning('请选择用户')
    return
  }
  try {
    await addProjectMembers(project.value.id, {
      user_ids: selectedUserIds.value,
      role: memberRole.value
    })
    message.success('添加成功')
    selectedUserIds.value = []
    await loadProjectMembers(project.value.id)
    // 重新加载项目详情
    await loadProject()
  } catch (error: any) {
    message.error(error.message || '添加失败')
  }
}

// 更新成员角色
const handleUpdateMemberRole = async (memberId: number, role: string) => {
  if (!project.value) return
  try {
    await updateProjectMember(project.value.id, memberId, role)
    message.success('更新成功')
    await loadProjectMembers(project.value.id)
    // 重新加载项目详情
    await loadProject()
  } catch (error: any) {
    message.error(error.message || '更新失败')
  }
}

// 移除成员
const handleRemoveMember = async (memberId: number) => {
  if (!project.value) return
  try {
    await removeProjectMember(project.value.id, memberId)
    message.success('移除成功')
    await loadProjectMembers(project.value.id)
    // 重新加载项目详情
    await loadProject()
  } catch (error: any) {
    message.error(error.message || '移除失败')
  }
}

// 功能模块管理
const handleManageModules = () => {
  moduleManageModalVisible.value = true
}

// 关闭模块管理对话框
const handleCloseModuleModal = () => {
  moduleManageModalVisible.value = false
}

// 需求管理
const handleManageRequirements = () => {
  if (!project.value) return
  router.push({
    path: '/requirement',
    query: { project_id: project.value.id }
  })
}

// 任务管理
const handleManageTasks = () => {
  if (!project.value) return
  router.push({
    path: '/task',
    query: { project_id: project.value.id }
  })
}

// Bug管理
const handleManageBugs = () => {
  if (!project.value) return
  router.push({
    path: '/bug',
    query: { project_id: project.value.id }
  })
}

// 跳转到任务列表
const goToTasks = (status: string) => {
  if (!project.value) return
  router.push({
    path: '/task',
    query: { status, project_id: project.value.id }
  })
}

// 跳转到Bug列表
const goToBugs = (status: string) => {
  if (!project.value) return
  router.push({
    path: '/bug',
    query: { status, project_id: project.value.id }
  })
}

// 跳转到需求列表
const goToRequirements = (status: string) => {
  if (!project.value) return
  router.push({
    path: '/requirement',
    query: { status, project_id: project.value.id }
  })
}

onMounted(() => {
  loadProject()
})
</script>

<style scoped>
.project-detail {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.project-detail :deep(.ant-layout) {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.content {
  flex: 1;
  padding: 24px;
  background: #f0f2f5;
  overflow-y: auto;
  overflow-x: hidden;
}

.content-inner {
  background: white;
  padding: 24px;
  border-radius: 4px;
  min-height: fit-content;
}

.description-content {
  max-width: 100%;
}

.stat-card {
  cursor: pointer;
  transition: all 0.3s;
  text-align: center;
}

.stat-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}
</style>

