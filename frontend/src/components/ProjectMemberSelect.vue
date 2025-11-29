<template>
  <div>
    <a-select
      :model-value="modelValue"
      :mode="multiple ? 'multiple' : undefined"
      :placeholder="placeholder"
      :allow-clear="allowClear"
      :show-search="showSearch"
      :filter-option="filterOption"
      :loading="loading"
      :disabled="disabled || !projectId"
      :style="style"
      @update:model-value="handleChange"
    >
      <a-select-option
        v-for="member in members"
        :key="member.user_id"
        :value="member.user_id"
      >
        {{ member.user?.username || '' }}{{ member.user?.nickname ? `(${member.user.nickname})` : '' }}
        <span v-if="showRole && member.role" style="color: #999; margin-left: 4px">
          ({{ getRoleText(member.role) }})
        </span>
      </a-select-option>
    </a-select>
    <div v-if="!projectId && showHint" style="color: #999; margin-top: 4px; font-size: 12px">
      {{ hintText || '请先选择项目' }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { getProjectMembers, type ProjectMember } from '@/api/project'

interface Props {
  modelValue?: number | number[] | undefined
  projectId?: number | undefined
  multiple?: boolean
  placeholder?: string
  allowClear?: boolean
  showSearch?: boolean
  disabled?: boolean
  style?: Record<string, any>
  showRole?: boolean
  showHint?: boolean
  hintText?: string
}

const props = withDefaults(defineProps<Props>(), {
  multiple: false,
  placeholder: '选择项目成员',
  allowClear: true,
  showSearch: true,
  disabled: false,
  style: () => ({}),
  showRole: true,
  showHint: true,
  hintText: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: number | number[] | undefined]
  'change': [value: number | number[] | undefined]
}>()

const members = ref<ProjectMember[]>([])
const loading = ref(false)

// 加载项目成员列表
const loadMembers = async (projectId: number | undefined) => {
  if (!projectId) {
    members.value = []
    loading.value = false
    return
  }
  
  loading.value = true
  try {
    members.value = await getProjectMembers(projectId)
  } catch (error: any) {
    console.error('加载项目成员失败:', error)
    members.value = []
  } finally {
    loading.value = false
  }
}

// 筛选函数
const filterOption = (input: string, option: any) => {
  const member = members.value.find(m => m.user_id === option.value)
  if (!member || !member.user) return false
  const searchText = input.toLowerCase()
  return (
    member.user.username.toLowerCase().includes(searchText) ||
    (member.user.nickname && member.user.nickname.toLowerCase().includes(searchText))
  )
}

// 获取角色文本
const getRoleText = (role: string): string => {
  const roleMap: Record<string, string> = {
    owner: '负责人',
    member: '成员',
    viewer: '查看者'
  }
  return roleMap[role] || role
}

// 处理值变化
const handleChange = (value: number | number[] | undefined) => {
  emit('update:modelValue', value)
  emit('change', value)
}

// 监听项目ID变化
watch(() => props.projectId, (newProjectId) => {
  loadMembers(newProjectId)
}, { immediate: true })

// 组件挂载时加载
onMounted(() => {
  if (props.projectId) {
    loadMembers(props.projectId)
  }
})
</script>

<style scoped>
/* 组件样式 */
</style>

