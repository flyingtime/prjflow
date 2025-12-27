<template>
  <div class="version-detail">
    <a-layout>
      <AppHeader />
      <a-layout-content class="content">
        <div class="content-inner">
          <a-page-header
            :title="version?.version_number || '版本详情'"
            @back="() => $router.push('/version')"
          >
            <template #extra>
              <a-space>
                <a-button @click="handleEdit">编辑</a-button>
                <a-button v-if="version?.status === 'wait'" type="primary" @click="handleRelease">
                  发布
                </a-button>
                <a-dropdown>
                  <a-button>
                    状态 <DownOutlined />
                  </a-button>
                  <template #overlay>
                    <a-menu @click="(e: any) => handleStatusChange(e.key as string)">
                      <a-menu-item key="wait">未开始</a-menu-item>
                      <a-menu-item key="normal">已发布</a-menu-item>
                      <a-menu-item key="fail">发布失败</a-menu-item>
                      <a-menu-item key="terminate">停止维护</a-menu-item>
                    </a-menu>
                  </template>
                </a-dropdown>
                <a-popconfirm
                  title="确定要删除这个版本吗？"
                  @confirm="handleDelete"
                >
                  <a-button danger>删除</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-page-header>

          <VersionDetailContent :version="version" :loading="loading" />
        </div>
      </a-layout-content>
    </a-layout>

    <!-- 版本编辑模态框 -->
    <a-modal
      v-model:open="editModalVisible"
      title="编辑版本"
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
        <a-form-item label="版本号" name="version_number">
          <a-input v-model:value="editFormData.version_number" placeholder="请输入版本号" />
        </a-form-item>
        <a-form-item label="项目" name="project_id">
          <a-select
            v-model:value="editFormData.project_id"
            placeholder="选择项目"
            show-search
            :filter-option="filterProjectOption"
            :disabled="true"
            @change="handleFormProjectChange"
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
          <a-select v-model:value="editFormData.status">
            <a-select-option value="wait">未开始</a-select-option>
            <a-select-option value="normal">已发布</a-select-option>
            <a-select-option value="fail">发布失败</a-select-option>
            <a-select-option value="terminate">停止维护</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="发布日期" name="release_date">
          <a-date-picker
            v-model:value="editFormData.release_date"
            placeholder="选择发布日期"
            style="width: 100%"
            format="YYYY-MM-DD"
          />
        </a-form-item>
        <a-form-item label="发布说明" name="release_notes">
          <MarkdownEditor
            ref="editReleaseNotesEditorRef"
            v-model="editFormData.release_notes"
            placeholder="请输入发布说明（支持Markdown）"
            :rows="8"
            :project-id="version?.project_id || 0"
          />
        </a-form-item>
        <a-form-item label="关联需求" name="requirement_ids">
          <a-select
            v-model:value="editFormData.requirement_ids"
            mode="multiple"
            placeholder="选择需求（可选）"
            show-search
            :filter-option="filterRequirementOption"
            style="width: 100%"
          >
            <a-select-option
              v-for="requirement in availableRequirements"
              :key="requirement.id"
              :value="requirement.id"
            >
              {{ requirement.title }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="关联Bug" name="bug_ids">
          <a-select
            v-model:value="editFormData.bug_ids"
            mode="multiple"
            placeholder="选择Bug（可选）"
            show-search
            :filter-option="filterBugOption"
            style="width: 100%"
          >
            <a-select-option
              v-for="bug in availableBugs"
              :key="bug.id"
              :value="bug.id"
            >
              {{ bug.title }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="附件">
          <AttachmentUpload
            v-if="editFormData.project_id && editFormData.project_id > 0"
            :project-id="editFormData.project_id"
            :model-value="editFormData.attachment_ids"
            :existing-attachments="versionAttachments"
            @update:modelValue="(value) => { editFormData.attachment_ids = value }"
            @attachment-deleted="handleAttachmentDeleted"
          />
          <span v-else style="color: #999;">请先选择项目后再上传附件</span>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { DownOutlined } from '@ant-design/icons-vue'
import dayjs, { type Dayjs } from 'dayjs'
import AppHeader from '@/components/AppHeader.vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import VersionDetailContent from '@/components/VersionDetailContent.vue'
import AttachmentUpload from '@/components/AttachmentUpload.vue'
import { getAttachments, type Attachment } from '@/api/attachment'
import {
  getVersion,
  updateVersion,
  updateVersionStatus,
  deleteVersion,
  releaseVersion,
  type Version,
  type UpdateVersionRequest
} from '@/api/version'
import { getProjects, type Project } from '@/api/project'
import { getRequirements, type Requirement } from '@/api/requirement'
import { getBugs, type Bug } from '@/api/bug'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const version = ref<Version | null>(null)
const projects = ref<Project[]>([])
const availableRequirements = ref<Requirement[]>([])
const availableBugs = ref<Bug[]>([])

// 编辑模态框相关
const editModalVisible = ref(false)
const editFormRef = ref()
const editReleaseNotesEditorRef = ref<InstanceType<typeof MarkdownEditor> | null>(null)
const editFormData = reactive<Omit<UpdateVersionRequest, 'release_date'> & { 
  release_date?: Dayjs | undefined
  requirement_ids?: number[]
  bug_ids?: number[]
  attachment_ids?: number[]
  project_id?: number  // 用于显示，不会提交
}>({
  version_number: '',
  release_notes: '',
  status: 'wait',
  release_date: undefined,
  requirement_ids: [],
  bug_ids: [],
  attachment_ids: [],
  project_id: undefined
})
const versionAttachments = ref<Attachment[]>([])
const editFormRules = {
  version_number: [{ required: true, message: '请输入版本号', trigger: 'blur' }]
}

// 加载版本详情
const loadVersion = async () => {
  const id = route.params.id as string
  if (!id) {
    message.error('版本ID不存在')
    router.push('/version')
    return
  }

  loading.value = true
  try {
    const res = await getVersion(Number(id))
    version.value = res
  } catch (error: any) {
    message.error(error.response?.data?.message || '加载失败')
    router.push('/version')
  } finally {
    loading.value = false
  }
}

// 编辑
const handleEdit = async () => {
  if (!version.value) return
  
  editFormData.version_number = version.value.version_number
  editFormData.release_notes = version.value.release_notes || ''
  // 直接使用后端状态值，不进行转换
  editFormData.status = version.value.status || 'wait'
  editFormData.project_id = version.value.project_id
  editFormData.release_date = version.value.release_date ? dayjs(version.value.release_date) : undefined
  editFormData.requirement_ids = version.value.requirements?.map((r: any) => r.id) || []
  editFormData.bug_ids = version.value.bugs?.map((b: any) => b.id) || []
  
  // 加载版本附件
  try {
    if (version.value.attachments && version.value.attachments.length > 0) {
      versionAttachments.value = version.value.attachments
      editFormData.attachment_ids = version.value.attachments.map((a: any) => a.id)
    } else {
      versionAttachments.value = await getAttachments({ version_id: version.value.id })
      editFormData.attachment_ids = versionAttachments.value.map(a => a.id)
    }
  } catch (error: any) {
    console.error('加载附件失败:', error)
    versionAttachments.value = []
    editFormData.attachment_ids = []
  }
  
  await nextTick() // 确保附件数据已加载
  editModalVisible.value = true
  if (projects.value.length === 0) {
    await loadProjects()
  }
  await loadAvailableRequirementsAndBugs()
}

// 编辑提交
const handleEditSubmit = async () => {
  if (!version.value) return
  
  try {
    await editFormRef.value.validate()
    
    // 获取最新的发布说明内容
    let releaseNotes = editFormData.release_notes || ''
    
    // 如果有项目ID，尝试上传本地图片（如果有的话）
    if (editReleaseNotesEditorRef.value && version.value.project_id) {
      try {
        const uploadedReleaseNotes = await editReleaseNotesEditorRef.value.uploadLocalImages(async (file: File, projectId: number) => {
          const { uploadFile } = await import('@/api/attachment')
          const attachment = await uploadFile(file, projectId)
          return attachment
        })
        releaseNotes = uploadedReleaseNotes
      } catch (error: any) {
        console.error('上传图片失败:', error)
        message.warning('部分图片上传失败，请检查')
        releaseNotes = editFormData.release_notes || ''
      }
    }
    
    const updateData: any = {
      version_number: editFormData.version_number,
      release_notes: releaseNotes,
      status: editFormData.status,
      release_date: editFormData.release_date && editFormData.release_date.isValid() ? editFormData.release_date.format('YYYY-MM-DD') : undefined,
      requirement_ids: editFormData.requirement_ids || [],
      bug_ids: editFormData.bug_ids || []
    }
    
    // 始终发送 attachment_ids，如果为 undefined 或 null，发送空数组
    const attachmentIdsValue = editFormData.attachment_ids
    if (attachmentIdsValue === undefined || attachmentIdsValue === null) {
      updateData.attachment_ids = []
    } else {
      updateData.attachment_ids = Array.isArray(attachmentIdsValue) ? attachmentIdsValue : []
    }
    
    await updateVersion(version.value.id, updateData)
    
    message.success('更新成功')
    editModalVisible.value = false
    await loadVersion() // 重新加载版本详情
  } catch (error: any) {
    if (error.errorFields) {
      return
    }
    message.error(error.response?.data?.message || '更新失败')
  }
}

// 编辑取消
const handleEditCancel = () => {
  editFormRef.value?.resetFields()
}

// 加载项目列表
const loadProjects = async () => {
  try {
    const res = await getProjects({ page: 1, size: 1000 })
    projects.value = res.list
  } catch (error: any) {
    console.error('加载项目列表失败:', error)
  }
}

// 加载可用的需求和Bug列表
const loadAvailableRequirementsAndBugs = async () => {
  if (!version.value?.project_id) return
  
  try {
    // 加载需求列表
    const requirementsRes = await getRequirements({ project_id: version.value.project_id, size: 1000 })
    availableRequirements.value = requirementsRes.list || []
    
    // 加载Bug列表
    const bugsRes = await getBugs({ project_id: version.value.project_id, size: 1000 })
    availableBugs.value = bugsRes.list || []
  } catch (error: any) {
    console.error('加载需求和Bug列表失败:', error)
  }
}

// 项目筛选
const filterProjectOption = (input: string, option: any) => {
  return option.children.toLowerCase().indexOf(input.toLowerCase()) >= 0
}

// 处理附件删除事件
const handleAttachmentDeleted = (attachmentId: number) => {
  // 从versionAttachments中移除已删除的附件
  versionAttachments.value = versionAttachments.value.filter(a => a.id !== attachmentId)
  // 从editFormData.attachment_ids中移除
  if (editFormData.attachment_ids) {
    editFormData.attachment_ids = editFormData.attachment_ids.filter(id => id !== attachmentId)
  }
}

// 需求筛选
const filterRequirementOption = (input: string, option: any) => {
  return option.children.toLowerCase().indexOf(input.toLowerCase()) >= 0
}

// Bug筛选
const filterBugOption = (input: string, option: any) => {
  return option.children.toLowerCase().indexOf(input.toLowerCase()) >= 0
}

// 表单项目选择改变
const handleFormProjectChange = () => {
  loadAvailableRequirementsAndBugs()
}

// 删除
const handleDelete = async () => {
  if (!version.value) return
  try {
    await deleteVersion(version.value.id)
    message.success('删除成功')
    router.push('/version')
  } catch (error: any) {
    message.error(error.response?.data?.message || '删除失败')
  }
}

// 状态变更
const handleStatusChange = async (status: string) => {
  if (!version.value) return
  try {
    await updateVersionStatus(version.value.id, status)
    message.success('状态更新成功')
    loadVersion()
  } catch (error: any) {
    message.error(error.response?.data?.message || '状态更新失败')
  }
}

// 发布版本
const handleRelease = async () => {
  if (!version.value) return
  try {
    await releaseVersion(version.value.id)
    message.success('发布成功')
    loadVersion()
  } catch (error: any) {
    message.error(error.response?.data?.message || '发布失败')
  }
}

onMounted(() => {
  loadVersion()
})
</script>

<style scoped>
.version-detail {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.version-detail :deep(.ant-layout) {
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
  max-width: 100%;
  width: 100%;
  margin: 0 auto;
  min-height: fit-content;
}

.markdown-content {
  padding: 16px 0;
}
</style>

