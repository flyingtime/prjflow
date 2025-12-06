<template>
  <div class="audit-log">
    <a-layout>
      <AppHeader />
      <a-layout-content class="content">
        <div class="content-inner">
          <a-page-header title="审计日志" />

          <!-- 搜索栏 -->
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
              <a-form-item label="关键词">
                <a-input
                  v-model:value="searchForm.keyword"
                  placeholder="用户名/资源类型/操作类型"
                  allow-clear
                  style="width: 200px"
                />
              </a-form-item>
              <a-form-item label="操作类型">
                <a-select
                  v-model:value="searchForm.action_type"
                  placeholder="选择操作类型"
                  allow-clear
                  style="width: 150px"
                >
                  <a-select-option value="login">登录</a-select-option>
                  <a-select-option value="logout">登出</a-select-option>
                  <a-select-option value="create">创建</a-select-option>
                  <a-select-option value="update">更新</a-select-option>
                  <a-select-option value="delete">删除</a-select-option>
                </a-select>
              </a-form-item>
              <a-form-item label="资源类型">
                <a-select
                  v-model:value="searchForm.resource_type"
                  placeholder="选择资源类型"
                  allow-clear
                  style="width: 150px"
                >
                  <a-select-option value="user">用户</a-select-option>
                  <a-select-option value="project">项目</a-select-option>
                  <a-select-option value="permission">权限</a-select-option>
                  <a-select-option value="role">角色</a-select-option>
                  <a-select-option value="system">系统</a-select-option>
                </a-select>
              </a-form-item>
              <a-form-item label="操作结果">
                <a-select
                  v-model:value="searchForm.success"
                  placeholder="选择操作结果"
                  allow-clear
                  style="width: 120px"
                >
                  <a-select-option :value="true">成功</a-select-option>
                  <a-select-option :value="false">失败</a-select-option>
                </a-select>
              </a-form-item>
              <a-form-item label="时间范围">
                <a-range-picker
                  v-model:value="dateRange"
                  format="YYYY-MM-DD"
                  style="width: 240px"
                />
              </a-form-item>
              <a-form-item>
                <a-button type="primary" @click="handleSearch">查询</a-button>
                <a-button style="margin-left: 8px" @click="handleReset">重置</a-button>
              </a-form-item>
            </a-form>
          </a-card>

          <!-- 审计日志列表 -->
          <a-card :bordered="false" class="table-card">
            <a-table
              :columns="columns"
              :data-source="auditLogs"
              :loading="loading"
              :scroll="{ x: 'max-content', y: tableScrollHeight }"
              :pagination="pagination"
              @change="handleTableChange"
              row-key="id"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'action_type'">
                  <a-tag :color="getActionTypeColor(record.action_type)">
                    {{ getActionTypeName(record.action_type) }}
                  </a-tag>
                </template>
                <template v-else-if="column.key === 'success'">
                  <a-tag :color="record.success ? 'green' : 'red'">
                    {{ record.success ? '成功' : '失败' }}
                  </a-tag>
                </template>
                <template v-else-if="column.key === 'created_at'">
                  {{ formatDateTime(record.created_at) }}
                </template>
                <template v-else-if="column.key === 'action'">
                  <a-button type="link" size="small" @click="handleViewDetail(record)">
                    查看详情
                  </a-button>
                </template>
              </template>
            </a-table>
          </a-card>

          <!-- 详情对话框 -->
          <a-modal
            v-model:open="detailVisible"
            title="审计日志详情"
            width="800px"
            :footer="null"
          >
            <a-descriptions :column="2" bordered v-if="currentRecord">
              <a-descriptions-item label="ID">{{ currentRecord.id }}</a-descriptions-item>
              <a-descriptions-item label="操作时间">
                {{ formatDateTime(currentRecord.created_at) }}
              </a-descriptions-item>
              <a-descriptions-item label="用户">
                {{ currentRecord.username }} (ID: {{ currentRecord.user_id }})
              </a-descriptions-item>
              <a-descriptions-item label="操作类型">
                <a-tag :color="getActionTypeColor(currentRecord.action_type)">
                  {{ getActionTypeName(currentRecord.action_type) }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="资源类型">
                {{ currentRecord.resource_type || '-' }}
              </a-descriptions-item>
              <a-descriptions-item label="资源ID">
                {{ currentRecord.resource_id || '-' }}
              </a-descriptions-item>
              <a-descriptions-item label="操作结果">
                <a-tag :color="currentRecord.success ? 'green' : 'red'">
                  {{ currentRecord.success ? '成功' : '失败' }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="IP地址">
                {{ currentRecord.ip_address || '-' }}
              </a-descriptions-item>
              <a-descriptions-item label="请求路径" :span="2">
                {{ currentRecord.path || '-' }}
              </a-descriptions-item>
              <a-descriptions-item label="请求方法">
                {{ currentRecord.method || '-' }}
              </a-descriptions-item>
              <a-descriptions-item label="请求参数" :span="2">
                <pre v-if="currentRecord.params" style="max-height: 200px; overflow: auto; background: #f5f5f5; padding: 8px; border-radius: 4px;">
{{ formatJSON(currentRecord.params) }}
                </pre>
                <span v-else>-</span>
              </a-descriptions-item>
              <a-descriptions-item label="错误信息" :span="2" v-if="currentRecord.error_msg">
                <a-alert type="error" :message="currentRecord.error_msg" />
              </a-descriptions-item>
              <a-descriptions-item label="备注" :span="2" v-if="currentRecord.comment">
                {{ currentRecord.comment }}
              </a-descriptions-item>
            </a-descriptions>
          </a-modal>
        </div>
      </a-layout-content>
    </a-layout>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import { UpOutlined, DownOutlined } from '@ant-design/icons-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { type Dayjs } from 'dayjs'
import { getAuditLogs, type AuditLog, type AuditLogListParams } from '@/api/auditLog'
import { formatDateTime } from '@/utils/date'
import AppHeader from '@/components/AppHeader.vue'

const searchFormVisible = ref(true)
const loading = ref(false)
const auditLogs = ref<AuditLog[]>([])
const total = ref(0)
const dateRange = ref<[Dayjs, Dayjs] | null>(null)
const detailVisible = ref(false)
const currentRecord = ref<AuditLog | null>(null)

const searchForm = reactive<AuditLogListParams>({
  keyword: undefined,
  action_type: undefined,
  resource_type: undefined,
  success: undefined,
  start_date: undefined,
  end_date: undefined,
  page: 1,
  size: 20
})

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条记录`,
  pageSizeOptions: ['10', '20', '50', '100']
})

const tableScrollHeight = computed(() => {
  return window.innerHeight - 400
})

const columns: TableColumnsType = [
  {
    title: '时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: 180,
    fixed: 'left'
  },
  {
    title: '用户',
    dataIndex: 'username',
    key: 'username',
    width: 120
  },
  {
    title: '操作类型',
    dataIndex: 'action_type',
    key: 'action_type',
    width: 100
  },
  {
    title: '资源类型',
    dataIndex: 'resource_type',
    key: 'resource_type',
    width: 100
  },
  {
    title: '资源ID',
    dataIndex: 'resource_id',
    key: 'resource_id',
    width: 100
  },
  {
    title: 'IP地址',
    dataIndex: 'ip_address',
    key: 'ip_address',
    width: 140
  },
  {
    title: '请求路径',
    dataIndex: 'path',
    key: 'path',
    width: 200,
    ellipsis: true
  },
  {
    title: '操作结果',
    dataIndex: 'success',
    key: 'success',
    width: 100
  },
  {
    title: '操作',
    key: 'action',
    width: 100,
    fixed: 'right'
  }
]

const toggleSearchForm = () => {
  searchFormVisible.value = !searchFormVisible.value
}

const handleSearch = () => {
  // 处理日期范围
  if (dateRange.value && dateRange.value.length === 2) {
    searchForm.start_date = dateRange.value[0].format('YYYY-MM-DD')
    searchForm.end_date = dateRange.value[1].format('YYYY-MM-DD')
  } else {
    searchForm.start_date = undefined
    searchForm.end_date = undefined
  }

  pagination.current = 1
  loadAuditLogs()
}

const handleReset = () => {
  searchForm.keyword = undefined
  searchForm.action_type = undefined
  searchForm.resource_type = undefined
  searchForm.success = undefined
  searchForm.start_date = undefined
  searchForm.end_date = undefined
  dateRange.value = null
  pagination.current = 1
  loadAuditLogs()
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadAuditLogs()
}

const loadAuditLogs = async () => {
  loading.value = true
  try {
    const params: AuditLogListParams = {
      ...searchForm,
      page: pagination.current,
      size: pagination.pageSize
    }

    const response = await getAuditLogs(params)
    auditLogs.value = response.list
    total.value = response.total
    pagination.total = response.total
  } catch (error: any) {
    message.error(error.message || '加载审计日志失败')
  } finally {
    loading.value = false
  }
}

const handleViewDetail = (record: AuditLog) => {
  currentRecord.value = record
  detailVisible.value = true
}

const getActionTypeName = (actionType: string) => {
  const map: Record<string, string> = {
    login: '登录',
    logout: '登出',
    create: '创建',
    update: '更新',
    delete: '删除'
  }
  return map[actionType] || actionType
}

const getActionTypeColor = (actionType: string) => {
  const map: Record<string, string> = {
    login: 'blue',
    logout: 'default',
    create: 'green',
    update: 'orange',
    delete: 'red'
  }
  return map[actionType] || 'default'
}

const formatJSON = (jsonStr: string) => {
  try {
    const obj = JSON.parse(jsonStr)
    return JSON.stringify(obj, null, 2)
  } catch {
    return jsonStr
  }
}

onMounted(() => {
  loadAuditLogs()
})
</script>

<style scoped>
.audit-log {
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

.table-card {
  margin-top: 16px;
}
</style>

