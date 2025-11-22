<template>
  <div class="report-management">
    <a-layout>
      <AppHeader />
      <a-layout-content class="content">
        <div class="content-inner">
          <a-page-header title="工作报告">
            <template #extra>
              <a-button type="primary" @click="handleCreate">
                <template #icon><PlusOutlined /></template>
                新增{{ activeTab === 'daily' ? '日报' : '周报' }}
              </a-button>
            </template>
          </a-page-header>

          <a-tabs v-model:activeKey="activeTab" @change="handleTabChange">
            <a-tab-pane key="daily" tab="日报">
              <a-card :bordered="false" style="margin-bottom: 16px">
                <a-form layout="inline" :model="dailySearchForm">
                  <a-form-item label="状态">
                    <a-select
                      v-model:value="dailySearchForm.status"
                      placeholder="选择状态"
                      allow-clear
                      style="width: 120px"
                    >
                      <a-select-option value="draft">草稿</a-select-option>
                      <a-select-option value="submitted">已提交</a-select-option>
                      <a-select-option value="approved">已审批</a-select-option>
                      <a-select-option value="rejected">已拒绝</a-select-option>
                    </a-select>
                  </a-form-item>
                  <a-form-item label="开始日期">
                    <a-date-picker
                      v-model:value="dailySearchForm.start_date"
                      placeholder="选择开始日期"
                      style="width: 150px"
                      format="YYYY-MM-DD"
                    />
                  </a-form-item>
                  <a-form-item label="结束日期">
                    <a-date-picker
                      v-model:value="dailySearchForm.end_date"
                      placeholder="选择结束日期"
                      style="width: 150px"
                      format="YYYY-MM-DD"
                    />
                  </a-form-item>
                  <a-form-item label="项目">
                    <a-select
                      v-model:value="dailySearchForm.project_id"
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
                    <a-button type="primary" @click="handleDailySearch">查询</a-button>
                    <a-button style="margin-left: 8px" @click="handleDailyReset">重置</a-button>
                  </a-form-item>
                </a-form>
              </a-card>

              <a-card :bordered="false">
                <a-table
                  :scroll="{ x: 'max-content' }"
                  :columns="dailyColumns"
                  :data-source="dailyReports"
                  :loading="dailyLoading"
                  :pagination="dailyPagination"
                  row-key="id"
                  @change="handleDailyTableChange"
                >
                  <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'status'">
                      <a-tag :color="getStatusColor(record.status)">
                        {{ getStatusText(record.status) }}
                      </a-tag>
                    </template>
                    <template v-else-if="column.key === 'approval_status'">
                      <div v-if="record.approvers && record.approvers.length > 0">
                        <a-space size="small" wrap>
                          <span v-for="approver in record.approvers" :key="approver.id">
                            <a-tag :color="getApproverStatusColor(record, approver.id)">
                              {{ approver.nickname || approver.username }}: {{ getApproverStatusText(record, approver.id) }}
                            </a-tag>
                          </span>
                        </a-space>
                      </div>
                      <span v-else>-</span>
                    </template>
                    <template v-else-if="column.key === 'date'">
                      {{ formatDate(record.date) }}
                    </template>
                    <template v-else-if="column.key === 'project'">
                      {{ record.project?.name || '-' }}
                    </template>
                    <template v-else-if="column.key === 'task'">
                      {{ record.tasks?.map((t: any) => t.title).join('、') || '-' }}
                    </template>
                    <template v-else-if="column.key === 'created_at'">
                      {{ formatDateTime(record.created_at) }}
                    </template>
                    <template v-else-if="column.key === 'action'">
                      <a-space>
                        <a-button type="link" size="small" @click="handleDailyEdit(record)">
                          编辑
                        </a-button>
                        <a-button
                          v-if="record.status === 'draft'"
                          type="link"
                          size="small"
                          @click="handleDailySubmit(record)"
                        >
                          提交
                        </a-button>
                        <a-button
                          v-if="canApproveDaily(record)"
                          type="link"
                          size="small"
                          style="color: #52c41a"
                          @click="handleDailyApproveClick(record)"
                        >
                          审批
                        </a-button>
                        <a-popconfirm
                          title="确定要删除这个日报吗？"
                          @confirm="handleDailyDelete(record.id)"
                        >
                          <a-button type="link" size="small" danger>删除</a-button>
                        </a-popconfirm>
                      </a-space>
                    </template>
                  </template>
                </a-table>
              </a-card>
            </a-tab-pane>

            <a-tab-pane key="weekly" tab="周报">
              <a-card :bordered="false" style="margin-bottom: 16px">
                <a-form layout="inline" :model="weeklySearchForm">
                  <a-form-item label="状态">
                    <a-select
                      v-model:value="weeklySearchForm.status"
                      placeholder="选择状态"
                      allow-clear
                      style="width: 120px"
                    >
                      <a-select-option value="draft">草稿</a-select-option>
                      <a-select-option value="submitted">已提交</a-select-option>
                      <a-select-option value="approved">已审批</a-select-option>
                      <a-select-option value="rejected">已拒绝</a-select-option>
                    </a-select>
                  </a-form-item>
                  <a-form-item label="开始日期">
                    <a-date-picker
                      v-model:value="weeklySearchForm.start_date"
                      placeholder="选择开始日期"
                      style="width: 150px"
                      format="YYYY-MM-DD"
                    />
                  </a-form-item>
                  <a-form-item label="结束日期">
                    <a-date-picker
                      v-model:value="weeklySearchForm.end_date"
                      placeholder="选择结束日期"
                      style="width: 150px"
                      format="YYYY-MM-DD"
                    />
                  </a-form-item>
                  <a-form-item label="项目">
                    <a-select
                      v-model:value="weeklySearchForm.project_id"
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
                    <a-button type="primary" @click="handleWeeklySearch">查询</a-button>
                    <a-button style="margin-left: 8px" @click="handleWeeklyReset">重置</a-button>
                  </a-form-item>
                </a-form>
              </a-card>

              <a-card :bordered="false">
                <a-table
                  :scroll="{ x: 'max-content' }"
                  :columns="weeklyColumns"
                  :data-source="weeklyReports"
                  :loading="weeklyLoading"
                  :pagination="weeklyPagination"
                  row-key="id"
                  @change="handleWeeklyTableChange"
                >
                  <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'status'">
                      <a-tag :color="getStatusColor(record.status)">
                        {{ getStatusText(record.status) }}
                      </a-tag>
                    </template>
                    <template v-else-if="column.key === 'approval_status'">
                      <div v-if="record.approvers && record.approvers.length > 0">
                        <a-space size="small" wrap>
                          <span v-for="approver in record.approvers" :key="approver.id">
                            <a-tag :color="getApproverStatusColor(record, approver.id)">
                              {{ approver.nickname || approver.username }}: {{ getApproverStatusText(record, approver.id) }}
                            </a-tag>
                          </span>
                        </a-space>
                      </div>
                      <span v-else>-</span>
                    </template>
                    <template v-else-if="column.key === 'week'">
                      {{ formatDate(record.week_start) }} ~ {{ formatDate(record.week_end) }}
                    </template>
                    <template v-else-if="column.key === 'project'">
                      {{ record.project?.name || '-' }}
                    </template>
                    <template v-else-if="column.key === 'task'">
                      {{ record.tasks?.map((t: any) => t.title).join('、') || '-' }}
                    </template>
                    <template v-else-if="column.key === 'created_at'">
                      {{ formatDateTime(record.created_at) }}
                    </template>
                    <template v-else-if="column.key === 'action'">
                      <a-space>
                        <a-button type="link" size="small" @click="handleWeeklyEdit(record)">
                          编辑
                        </a-button>
                        <a-button
                          v-if="record.status === 'draft'"
                          type="link"
                          size="small"
                          @click="handleWeeklySubmit(record)"
                        >
                          提交
                        </a-button>
                        <a-button
                          v-if="canApproveWeekly(record)"
                          type="link"
                          size="small"
                          style="color: #52c41a"
                          @click="handleWeeklyApproveClick(record)"
                        >
                          审批
                        </a-button>
                        <a-popconfirm
                          title="确定要删除这个周报吗？"
                          @confirm="handleWeeklyDelete(record.id)"
                        >
                          <a-button type="link" size="small" danger>删除</a-button>
                        </a-popconfirm>
                      </a-space>
                    </template>
                  </template>
                </a-table>
              </a-card>
            </a-tab-pane>

            <a-tab-pane key="approval">
              <template #tab>
                <a-badge 
                  :count="pendingApprovalCount" 
                  :number-style="{ backgroundColor: '#ff4d4f' }"
                  :show-zero="false"
                >
                  <span>审批</span>
                </a-badge>
              </template>
              <a-tabs v-model:activeKey="approvalSubTab" @change="handleApprovalSubTabChange">
                <a-tab-pane key="daily-approval" tab="日报审批">
                  <a-card :bordered="false" style="margin-bottom: 16px">
                    <a-form layout="inline" :model="approvalDailySearchForm">
                      <a-form-item label="审批状态">
                        <a-select
                          v-model:value="approvalDailySearchForm.approval_status"
                          placeholder="选择审批状态"
                          allow-clear
                          style="width: 150px"
                        >
                          <a-select-option value="pending">待审批</a-select-option>
                          <a-select-option value="approved">已审批</a-select-option>
                          <a-select-option value="rejected">已拒绝</a-select-option>
                        </a-select>
                      </a-form-item>
                      <a-form-item label="开始日期">
                        <a-date-picker
                          v-model:value="approvalDailySearchForm.start_date"
                          placeholder="选择开始日期"
                          style="width: 150px"
                          format="YYYY-MM-DD"
                        />
                      </a-form-item>
                      <a-form-item label="结束日期">
                        <a-date-picker
                          v-model:value="approvalDailySearchForm.end_date"
                          placeholder="选择结束日期"
                          style="width: 150px"
                          format="YYYY-MM-DD"
                        />
                      </a-form-item>
                      <a-form-item>
                        <a-button type="primary" @click="handleApprovalDailySearch">查询</a-button>
                        <a-button style="margin-left: 8px" @click="handleApprovalDailyReset">重置</a-button>
                      </a-form-item>
                    </a-form>
                  </a-card>

                  <a-card :bordered="false">
                    <a-table
                      :scroll="{ x: 'max-content' }"
                      :columns="approvalDailyColumns"
                      :data-source="approvalDailyReports"
                      :loading="approvalDailyLoading"
                      :pagination="approvalDailyPagination"
                      row-key="id"
                      @change="handleApprovalDailyTableChange"
                    >
                      <template #bodyCell="{ column, record }">
                        <template v-if="column.key === 'status'">
                          <a-tag :color="getStatusColor(record.status)">
                            {{ getStatusText(record.status) }}
                          </a-tag>
                        </template>
                        <template v-else-if="column.key === 'date'">
                          {{ formatDate(record.date) }}
                        </template>
                        <template v-else-if="column.key === 'user'">
                          {{ record.user?.nickname || record.user?.username || '-' }}
                        </template>
                        <template v-else-if="column.key === 'project'">
                          {{ record.project?.name || '-' }}
                        </template>
                        <template v-else-if="column.key === 'approval_status'">
                          <a-tag :color="getApprovalStatusColor(record)">
                            {{ getApprovalStatusText(record) }}
                          </a-tag>
                        </template>
                        <template v-else-if="column.key === 'action'">
                          <a-space>
                            <a-button type="link" size="small" @click="handleApprovalDailyView(record)">
                              查看
                            </a-button>
                            <a-button
                              v-if="canApproveDaily(record)"
                              type="link"
                              size="small"
                              style="color: #52c41a"
                              @click="handleDailyApproveClick(record)"
                            >
                              审批
                            </a-button>
                          </a-space>
                        </template>
                      </template>
                    </a-table>
                  </a-card>
                </a-tab-pane>

                <a-tab-pane key="weekly-approval" tab="周报审批">
                  <a-card :bordered="false" style="margin-bottom: 16px">
                    <a-form layout="inline" :model="approvalWeeklySearchForm">
                      <a-form-item label="审批状态">
                        <a-select
                          v-model:value="approvalWeeklySearchForm.approval_status"
                          placeholder="选择审批状态"
                          allow-clear
                          style="width: 150px"
                        >
                          <a-select-option value="pending">待审批</a-select-option>
                          <a-select-option value="approved">已审批</a-select-option>
                          <a-select-option value="rejected">已拒绝</a-select-option>
                        </a-select>
                      </a-form-item>
                      <a-form-item label="开始日期">
                        <a-date-picker
                          v-model:value="approvalWeeklySearchForm.start_date"
                          placeholder="选择开始日期"
                          style="width: 150px"
                          format="YYYY-MM-DD"
                        />
                      </a-form-item>
                      <a-form-item label="结束日期">
                        <a-date-picker
                          v-model:value="approvalWeeklySearchForm.end_date"
                          placeholder="选择结束日期"
                          style="width: 150px"
                          format="YYYY-MM-DD"
                        />
                      </a-form-item>
                      <a-form-item>
                        <a-button type="primary" @click="handleApprovalWeeklySearch">查询</a-button>
                        <a-button style="margin-left: 8px" @click="handleApprovalWeeklyReset">重置</a-button>
                      </a-form-item>
                    </a-form>
                  </a-card>

                  <a-card :bordered="false">
                    <a-table
                      :scroll="{ x: 'max-content' }"
                      :columns="approvalWeeklyColumns"
                      :data-source="approvalWeeklyReports"
                      :loading="approvalWeeklyLoading"
                      :pagination="approvalWeeklyPagination"
                      row-key="id"
                      @change="handleApprovalWeeklyTableChange"
                    >
                      <template #bodyCell="{ column, record }">
                        <template v-if="column.key === 'status'">
                          <a-tag :color="getStatusColor(record.status)">
                            {{ getStatusText(record.status) }}
                          </a-tag>
                        </template>
                        <template v-else-if="column.key === 'week'">
                          {{ formatDate(record.week_start) }} ~ {{ formatDate(record.week_end) }}
                        </template>
                        <template v-else-if="column.key === 'user'">
                          {{ record.user?.nickname || record.user?.username || '-' }}
                        </template>
                        <template v-else-if="column.key === 'project'">
                          {{ record.project?.name || '-' }}
                        </template>
                        <template v-else-if="column.key === 'approval_status'">
                          <a-tag :color="getApprovalStatusColor(record)">
                            {{ getApprovalStatusText(record) }}
                          </a-tag>
                        </template>
                        <template v-else-if="column.key === 'action'">
                          <a-space>
                            <a-button type="link" size="small" @click="handleApprovalWeeklyView(record)">
                              查看
                            </a-button>
                            <a-button
                              v-if="canApproveWeekly(record)"
                              type="link"
                              size="small"
                              style="color: #52c41a"
                              @click="handleWeeklyApproveClick(record)"
                            >
                              审批
                            </a-button>
                          </a-space>
                        </template>
                      </template>
                    </a-table>
                  </a-card>
                </a-tab-pane>
              </a-tabs>
            </a-tab-pane>
          </a-tabs>
        </div>
      </a-layout-content>
    </a-layout>

    <!-- 日报编辑/创建模态框 -->
    <a-modal
      :mask-closable="true"
      v-model:open="dailyModalVisible"
      :title="dailyModalTitle"
      :width="800"
      @ok="handleDailySubmitForm"
      @cancel="handleDailyCancel"
    >
      <a-form
        ref="dailyFormRef"
        :model="dailyFormData"
        :rules="dailyFormRules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="日期" name="date">
          <a-date-picker
            v-model:value="dailyFormData.date"
            placeholder="选择日期"
            style="width: 100%"
            format="YYYY-MM-DD"
            :disabled="!!dailyFormData.id"
          />
        </a-form-item>
        <a-form-item label="工作内容" name="content">
          <MarkdownEditor
            ref="dailyContentEditorRef"
            v-model="dailyFormData.content"
            placeholder="请输入工作内容（支持Markdown）"
            :rows="8"
            :project-id="dailyFormData.project_id || 0"
          />
        </a-form-item>
        <a-form-item label="工时" name="hours">
          <a-input-number
            v-model:value="dailyFormData.hours"
            placeholder="工时（小时）"
            :min="0"
            :precision="2"
            style="width: 100%"
          />
        </a-form-item>
        <a-form-item label="项目" name="project_id">
          <a-select
            v-model:value="dailyFormData.project_id"
            placeholder="选择项目（可选）"
            allow-clear
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
        <a-form-item label="任务" name="task_ids">
          <a-tooltip :title="!dailyFormData.project_id ? '请先选择项目' : ''">
            <a-select
              v-model:value="dailyFormData.task_ids"
              mode="multiple"
              :placeholder="!dailyFormData.project_id ? '请先选择项目' : '选择任务（可选，支持多选）'"
              allow-clear
              show-search
              :filter-option="filterTaskOption"
              :loading="taskLoading"
              :disabled="!dailyFormData.project_id"
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
          </a-tooltip>
        </a-form-item>
        <a-form-item label="审批人" name="approver_ids">
          <a-select
            v-model:value="dailyFormData.approver_ids"
            mode="multiple"
            placeholder="选择审批人（可选，支持多选）"
            allow-clear
            show-search
            :filter-option="filterUserOption"
          >
            <a-select-option
              v-for="user in users"
              :key="user.id"
              :value="user.id"
            >
              {{ user.nickname || user.username }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 周报编辑/创建模态框 -->
    <a-modal
      :mask-closable="true"
      v-model:open="weeklyModalVisible"
      :title="weeklyModalTitle"
      :width="800"
      @ok="handleWeeklySubmitForm"
      @cancel="handleWeeklyCancel"
    >
      <a-form
        ref="weeklyFormRef"
        :model="weeklyFormData"
        :rules="weeklyFormRules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="周开始日期" name="week_start">
          <a-date-picker
            v-model:value="weeklyFormData.week_start"
            placeholder="选择周开始日期"
            style="width: 100%"
            format="YYYY-MM-DD"
          />
        </a-form-item>
        <a-form-item label="周结束日期" name="week_end">
          <a-date-picker
            v-model:value="weeklyFormData.week_end"
            placeholder="选择周结束日期"
            style="width: 100%"
            format="YYYY-MM-DD"
          />
        </a-form-item>
        <a-form-item label="工作总结" name="summary">
          <MarkdownEditor
            ref="weeklySummaryEditorRef"
            v-model="weeklyFormData.summary"
            placeholder="请输入工作总结（支持Markdown）"
            :rows="8"
            :project-id="weeklyFormData.project_id || 0"
          />
        </a-form-item>
        <a-form-item label="下周计划" name="next_week_plan">
          <MarkdownEditor
            ref="weeklyPlanEditorRef"
            v-model="weeklyFormData.next_week_plan"
            placeholder="请输入下周计划（支持Markdown）"
            :rows="8"
            :project-id="weeklyFormData.project_id || 0"
          />
        </a-form-item>
        <a-form-item label="项目" name="project_id">
          <a-select
            v-model:value="weeklyFormData.project_id"
            placeholder="选择项目（可选）"
            allow-clear
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
        <a-form-item label="任务" name="task_ids">
          <a-tooltip :title="!weeklyFormData.project_id ? '请先选择项目' : ''">
            <a-select
              v-model:value="weeklyFormData.task_ids"
              mode="multiple"
              :placeholder="!weeklyFormData.project_id ? '请先选择项目' : '选择任务（可选，支持多选）'"
              allow-clear
              show-search
              :filter-option="filterTaskOption"
              :loading="taskLoading"
              :disabled="!weeklyFormData.project_id"
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
          </a-tooltip>
        </a-form-item>
        <a-form-item label="审批人" name="approver_ids">
          <a-select
            v-model:value="weeklyFormData.approver_ids"
            mode="multiple"
            placeholder="选择审批人（可选，支持多选）"
            allow-clear
            show-search
            :filter-option="filterUserOption"
          >
            <a-select-option
              v-for="user in users"
              :key="user.id"
              :value="user.id"
            >
              {{ user.nickname || user.username }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 日报审批弹窗 -->
    <a-modal
      :mask-closable="false"
      v-model:open="dailyApproveModalVisible"
      title="审批日报"
      :width="900"
      @ok="handleDailyApproveSubmit"
      @cancel="handleDailyApproveCancel"
    >
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }" v-if="dailyApproveData">
        <a-form-item label="日期">
          <span>{{ dailyApproveData.date ? formatDate(dailyApproveData.date) : '-' }}</span>
        </a-form-item>
        <a-form-item label="工作内容">
          <div v-html="renderMarkdown(dailyApproveData.content || '')" style="max-height: 300px; overflow-y: auto; border: 1px solid #d9d9d9; padding: 12px; border-radius: 4px;"></div>
        </a-form-item>
        <a-form-item label="工时">
          <span>{{ dailyApproveData.hours }} 小时</span>
        </a-form-item>
        <a-form-item label="项目">
          <span>{{ dailyApproveData.project?.name || '-' }}</span>
        </a-form-item>
        <a-form-item label="任务">
          <span>{{ dailyApproveData.tasks?.map(t => t.title).join('、') || '-' }}</span>
        </a-form-item>
        <a-form-item label="审批人">
          <div v-if="dailyApproveData.approvers && dailyApproveData.approvers.length > 0">
            <a-space size="small" wrap>
              <span v-for="approver in dailyApproveData.approvers" :key="approver.id">
                <a-tag :color="getApproverStatusColor(dailyApproveData, approver.id)">
                  {{ approver.nickname || approver.username }}: {{ getApproverStatusText(dailyApproveData, approver.id) }}
                </a-tag>
              </span>
            </a-space>
          </div>
          <span v-else>-</span>
        </a-form-item>
        <a-form-item label="审批记录" v-if="dailyApproveData.approval_records && dailyApproveData.approval_records.length > 0">
          <a-list :data-source="dailyApproveData.approval_records" size="small" bordered>
            <template #renderItem="{ item }">
              <a-list-item>
                <a-list-item-meta>
                  <template #title>
                    <span>{{ item.approver?.nickname || item.approver?.username || '未知' }}</span>
                    <a-tag :color="getApproverStatusColor(dailyApproveData, item.approver_id)" style="margin-left: 8px;">
                      {{ getApproverStatusText(dailyApproveData, item.approver_id) }}
                    </a-tag>
                  </template>
                  <template #description>
                    <div v-if="item.comment" style="margin-top: 4px; color: #666;">
                      <strong>批注：</strong>{{ item.comment }}
                    </div>
                    <div style="margin-top: 4px; font-size: 12px; color: #999;">
                      {{ item.updated_at ? formatDateTime(item.updated_at) : (item.created_at ? formatDateTime(item.created_at) : '') }}
                    </div>
                  </template>
                </a-list-item-meta>
              </a-list-item>
            </template>
          </a-list>
        </a-form-item>
        <a-form-item label="批注" name="comment">
          <a-textarea
            v-model:value="dailyApproveComment"
            placeholder="请输入批注（可选）"
            :rows="4"
          />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="handleDailyApproveCancel">取消</a-button>
          <a-button type="primary" danger @click="handleDailyApproveReject">拒绝</a-button>
          <a-button type="primary" @click="handleDailyApproveSubmit">通过</a-button>
        </a-space>
      </template>
    </a-modal>

    <!-- 周报审批弹窗 -->
    <a-modal
      :mask-closable="false"
      v-model:open="weeklyApproveModalVisible"
      title="审批周报"
      :width="900"
      @ok="handleWeeklyApproveSubmit"
      @cancel="handleWeeklyApproveCancel"
    >
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }" v-if="weeklyApproveData">
        <a-form-item label="周期">
          <span>{{ weeklyApproveData.week_start ? formatDate(weeklyApproveData.week_start) : '-' }} 至 {{ weeklyApproveData.week_end ? formatDate(weeklyApproveData.week_end) : '-' }}</span>
        </a-form-item>
        <a-form-item label="工作总结">
          <div v-html="renderMarkdown(weeklyApproveData.summary || '')" style="max-height: 300px; overflow-y: auto; border: 1px solid #d9d9d9; padding: 12px; border-radius: 4px;"></div>
        </a-form-item>
        <a-form-item label="下周计划">
          <div v-html="renderMarkdown(weeklyApproveData.next_week_plan || '')" style="max-height: 300px; overflow-y: auto; border: 1px solid #d9d9d9; padding: 12px; border-radius: 4px;"></div>
        </a-form-item>
        <a-form-item label="项目">
          <span>{{ weeklyApproveData.project?.name || '-' }}</span>
        </a-form-item>
        <a-form-item label="任务">
          <span>{{ weeklyApproveData.tasks?.map(t => t.title).join('、') || '-' }}</span>
        </a-form-item>
        <a-form-item label="审批人">
          <div v-if="weeklyApproveData.approvers && weeklyApproveData.approvers.length > 0">
            <a-space size="small" wrap>
              <span v-for="approver in weeklyApproveData.approvers" :key="approver.id">
                <a-tag :color="getApproverStatusColor(weeklyApproveData, approver.id)">
                  {{ approver.nickname || approver.username }}: {{ getApproverStatusText(weeklyApproveData, approver.id) }}
                </a-tag>
              </span>
            </a-space>
          </div>
          <span v-else>-</span>
        </a-form-item>
        <a-form-item label="审批记录" v-if="weeklyApproveData.approval_records && weeklyApproveData.approval_records.length > 0">
          <a-list :data-source="weeklyApproveData.approval_records" size="small" bordered>
            <template #renderItem="{ item }">
              <a-list-item>
                <a-list-item-meta>
                  <template #title>
                    <span>{{ item.approver?.nickname || item.approver?.username || '未知' }}</span>
                    <a-tag :color="getApproverStatusColor(weeklyApproveData, item.approver_id)" style="margin-left: 8px;">
                      {{ getApproverStatusText(weeklyApproveData, item.approver_id) }}
                    </a-tag>
                  </template>
                  <template #description>
                    <div v-if="item.comment" style="margin-top: 4px; color: #666;">
                      <strong>批注：</strong>{{ item.comment }}
                    </div>
                    <div style="margin-top: 4px; font-size: 12px; color: #999;">
                      {{ item.updated_at ? formatDateTime(item.updated_at) : (item.created_at ? formatDateTime(item.created_at) : '') }}
                    </div>
                  </template>
                </a-list-item-meta>
              </a-list-item>
            </template>
          </a-list>
        </a-form-item>
        <a-form-item label="批注" name="comment">
          <a-textarea
            v-model:value="weeklyApproveComment"
            placeholder="请输入批注（可选）"
            :rows="4"
          />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="handleWeeklyApproveCancel">取消</a-button>
          <a-button type="primary" danger @click="handleWeeklyApproveReject">拒绝</a-button>
          <a-button type="primary" @click="handleWeeklyApproveSubmit">通过</a-button>
        </a-space>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch, computed } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import dayjs, { type Dayjs } from 'dayjs'
import { formatDateTime, formatDate } from '@/utils/date'
import AppHeader from '@/components/AppHeader.vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import { usePermissionStore } from '@/stores/permission'
import { useAuthStore } from '@/stores/auth'
import { getUsers } from '@/api/user'
import {
  getDailyReports,
  getDailyReport,
  createDailyReport,
  updateDailyReport,
  deleteDailyReport,
  updateDailyReportStatus,
  getWeeklyReports,
  getWeeklyReport,
  createWeeklyReport,
  updateWeeklyReport,
  deleteWeeklyReport,
  updateWeeklyReportStatus,
  approveDailyReport,
  approveWeeklyReport,
  type DailyReport,
  type WeeklyReport,
  type CreateDailyReportRequest,
  type CreateWeeklyReportRequest
} from '@/api/report'
import { getProjects, type Project } from '@/api/project'
import { getTasks, type Task } from '@/api/task'
import { getDashboard } from '@/api/dashboard'
import { uploadFile } from '@/api/attachment'
import { marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/github.css'

// 配置marked
marked.setOptions({
  highlight: function(code: string, lang: string) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return hljs.highlight(code, { language: lang }).value
      } catch (err) {
        console.error('Highlight error:', err)
      }
    }
    return hljs.highlightAuto(code).value
  },
  breaks: true,
  gfm: true
} as any)

const route = useRoute()
const activeTab = ref<'daily' | 'weekly' | 'approval'>('daily')
const approvalSubTab = ref<'daily-approval' | 'weekly-approval'>('daily-approval')

// 日报相关
const dailyLoading = ref(false)
const dailyReports = ref<DailyReport[]>([])
const dailySearchForm = reactive({
  status: undefined as string | undefined,
  start_date: undefined as Dayjs | undefined,
  end_date: undefined as Dayjs | undefined,
  project_id: undefined as number | undefined
})
const dailyPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showTotal: (total: number) => `共 ${total} 条`,
  showSizeChanger: true,
  showQuickJumper: true
})
const dailyColumns = [
  { title: '日期', key: 'date', width: 120 },
  { title: '工作内容', dataIndex: 'content', key: 'content', ellipsis: true },
  { title: '工时', dataIndex: 'hours', key: 'hours', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '审批状态', key: 'approval_status', width: 200 },
  { title: '项目', key: 'project', width: 120 },
  { title: '任务', key: 'task', width: 150, ellipsis: true },
  { title: '创建时间', key: 'created_at', width: 180 },
  { title: '操作', key: 'action', width: 200, fixed: 'right' as const }
]

const dailyModalVisible = ref(false)
const dailyModalTitle = ref('新增日报')
const dailyFormRef = ref()
const dailyContentEditorRef = ref()
const dailyFormData = reactive<{
  id?: number
  date?: Dayjs
  content?: string
  hours?: number
  project_id?: number
  task_ids?: number[] // 任务ID数组（多选）
  approver_ids?: number[] // 审批人ID数组（多选）
}>({
  date: undefined,
  content: '',
  hours: 0,
  project_id: undefined,
  task_ids: [],
  approver_ids: []
})
const dailyFormRules = {
  date: [{ required: true, message: '请选择日期', trigger: 'change' }]
}

// 周报相关
const weeklyLoading = ref(false)
const weeklyReports = ref<WeeklyReport[]>([])
const weeklySearchForm = reactive({
  status: undefined as string | undefined,
  start_date: undefined as Dayjs | undefined,
  end_date: undefined as Dayjs | undefined,
  project_id: undefined as number | undefined
})
const weeklyPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showTotal: (total: number) => `共 ${total} 条`,
  showSizeChanger: true,
  showQuickJumper: true
})
const weeklyColumns = [
  { title: '周期', key: 'week', width: 200 },
  { title: '工作总结', dataIndex: 'summary', key: 'summary', ellipsis: true },
  { title: '下周计划', dataIndex: 'next_week_plan', key: 'next_week_plan', ellipsis: true },
  { title: '状态', key: 'status', width: 100 },
  { title: '审批状态', key: 'approval_status', width: 200 },
  { title: '项目', key: 'project', width: 120 },
  { title: '任务', key: 'task', width: 150, ellipsis: true },
  { title: '创建时间', key: 'created_at', width: 180 },
  { title: '操作', key: 'action', width: 200, fixed: 'right' as const }
]

const weeklyModalVisible = ref(false)
const weeklyModalTitle = ref('新增周报')
const weeklyFormRef = ref()
const weeklySummaryEditorRef = ref()
const weeklyPlanEditorRef = ref()
const weeklyFormData = reactive<{
  id?: number
  week_start?: Dayjs
  week_end?: Dayjs
  summary?: string
  next_week_plan?: string
  project_id?: number
  task_ids?: number[] // 任务ID数组（多选）
  approver_ids?: number[] // 审批人ID数组（多选）
}>({
  week_start: undefined,
  week_end: undefined,
  summary: '',
  next_week_plan: '',
  project_id: undefined,
  task_ids: [],
  approver_ids: []
})
const weeklyFormRules = {
  week_start: [{ required: true, message: '请选择周开始日期', trigger: 'change' }],
  week_end: [{ required: true, message: '请选择周结束日期', trigger: 'change' }]
}

// 审批日报相关
const approvalDailyLoading = ref(false)
const approvalDailyReports = ref<DailyReport[]>([])
const approvalDailySearchForm = reactive({
  approval_status: undefined as string | undefined,
  start_date: undefined as Dayjs | undefined,
  end_date: undefined as Dayjs | undefined
})
const approvalDailyPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showTotal: (total: number) => `共 ${total} 条`,
  showSizeChanger: true,
  showQuickJumper: true
})
const approvalDailyColumns = [
  { title: '日期', key: 'date', width: 120 },
  { title: '提交人', key: 'user', width: 120 },
  { title: '工作内容', dataIndex: 'content', key: 'content', ellipsis: true },
  { title: '工时', dataIndex: 'hours', key: 'hours', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '项目', key: 'project', width: 120 },
  { title: '审批状态', key: 'approval_status', width: 120 },
  { title: '操作', key: 'action', width: 150, fixed: 'right' as const }
]

// 审批周报相关
const approvalWeeklyLoading = ref(false)
const approvalWeeklyReports = ref<WeeklyReport[]>([])
const approvalWeeklySearchForm = reactive({
  approval_status: undefined as string | undefined,
  start_date: undefined as Dayjs | undefined,
  end_date: undefined as Dayjs | undefined
})
const approvalWeeklyPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showTotal: (total: number) => `共 ${total} 条`,
  showSizeChanger: true,
  showQuickJumper: true
})
const approvalWeeklyColumns = [
  { title: '周期', key: 'week', width: 200 },
  { title: '提交人', key: 'user', width: 120 },
  { title: '工作总结', dataIndex: 'summary', key: 'summary', ellipsis: true },
  { title: '状态', key: 'status', width: 100 },
  { title: '项目', key: 'project', width: 120 },
  { title: '审批状态', key: 'approval_status', width: 120 },
  { title: '操作', key: 'action', width: 150, fixed: 'right' as const }
]

// 公共数据
const projects = ref<Project[]>([])
const availableTasks = ref<Task[]>([])
const taskLoading = ref(false)
const users = ref<any[]>([]) // 用户列表（用于审批人选择）
const pendingApprovalCount = ref(0) // 待审批数量

// 权限管理
const permissionStore = usePermissionStore()
const authStore = useAuthStore()
const isAdmin = computed(() => permissionStore.hasRole('admin'))

// 加载日报列表
const loadDailyReports = async () => {
  dailyLoading.value = true
  try {
    const params: any = {
      page: dailyPagination.current,
      size: dailyPagination.pageSize
    }
    if (dailySearchForm.status) {
      params.status = dailySearchForm.status
    }
    if (dailySearchForm.start_date && dailySearchForm.start_date.isValid()) {
      params.start_date = dailySearchForm.start_date.format('YYYY-MM-DD')
    }
    if (dailySearchForm.end_date && dailySearchForm.end_date.isValid()) {
      params.end_date = dailySearchForm.end_date.format('YYYY-MM-DD')
    }
    if (dailySearchForm.project_id) {
      params.project_id = dailySearchForm.project_id
    }
    const response = await getDailyReports(params)
    dailyReports.value = response.list
    dailyPagination.total = response.total
  } catch (error: any) {
    message.error(error.message || '加载日报列表失败')
  } finally {
    dailyLoading.value = false
  }
}

// 加载周报列表
const loadWeeklyReports = async () => {
  weeklyLoading.value = true
  try {
    const params: any = {
      page: weeklyPagination.current,
      size: weeklyPagination.pageSize
    }
    if (weeklySearchForm.status) {
      params.status = weeklySearchForm.status
    }
    if (weeklySearchForm.start_date && weeklySearchForm.start_date.isValid()) {
      params.start_date = weeklySearchForm.start_date.format('YYYY-MM-DD')
    }
    if (weeklySearchForm.end_date && weeklySearchForm.end_date.isValid()) {
      params.end_date = weeklySearchForm.end_date.format('YYYY-MM-DD')
    }
    if (weeklySearchForm.project_id) {
      params.project_id = weeklySearchForm.project_id
    }
    const response = await getWeeklyReports(params)
    weeklyReports.value = response.list
    weeklyPagination.total = response.total
  } catch (error: any) {
    message.error(error.message || '加载周报列表失败')
  } finally {
    weeklyLoading.value = false
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

// 加载任务列表（用于关联选择）
const loadTasksForProject = async () => {
  const projectId = dailyFormData.project_id || weeklyFormData.project_id
  if (!projectId) {
    availableTasks.value = []
    message.warning('请先选择项目')
    return
  }
  
  // 如果已经有任务列表且项目ID没变化，不需要重新加载
  if (availableTasks.value.length > 0) {
    const currentProjectId = availableTasks.value[0]?.project_id
    if (currentProjectId === projectId) {
      return
    }
  }
  
  taskLoading.value = true
  try {
    const response = await getTasks({ project_id: projectId, size: 1000 })
    availableTasks.value = response.list
  } catch (error: any) {
    console.error('加载任务列表失败:', error)
    message.error('加载任务列表失败')
  } finally {
    taskLoading.value = false
  }
}

// 加载用户列表（用于审批人选择）
const loadUsers = async () => {
  try {
    const response = await getUsers({ size: 1000 })
    users.value = response.list || []
  } catch (error: any) {
    console.error('加载用户列表失败:', error)
  }
}

// 用户选择器过滤函数
const filterUserOption = (input: string, option: any) => {
  const user = users.value.find(u => u.id === option.value)
  if (!user) return false
  const keyword = input.toLowerCase()
  return (
    (user.nickname || '').toLowerCase().includes(keyword) ||
    (user.username || '').toLowerCase().includes(keyword)
  )
}

// 标签页切换
const handleTabChange = (key: string) => {
  activeTab.value = key as 'daily' | 'weekly' | 'approval'
  if (key === 'daily') {
    loadDailyReports()
  } else if (key === 'weekly') {
    loadWeeklyReports()
  } else if (key === 'approval') {
    handleApprovalSubTabChange(approvalSubTab.value)
  }
}

// 审批子标签页切换
const handleApprovalSubTabChange = (key: string) => {
  approvalSubTab.value = key as 'daily-approval' | 'weekly-approval'
  if (key === 'daily-approval') {
    loadApprovalDailyReports()
  } else if (key === 'weekly-approval') {
    loadApprovalWeeklyReports()
  }
}

// 日报搜索
const handleDailySearch = () => {
  dailyPagination.current = 1
  loadDailyReports()
}

// 日报重置
const handleDailyReset = () => {
  dailySearchForm.status = undefined
  dailySearchForm.start_date = undefined
  dailySearchForm.end_date = undefined
  dailySearchForm.project_id = undefined
  dailyPagination.current = 1
  loadDailyReports()
}

// 日报表格变化
const handleDailyTableChange = (pag: any) => {
  dailyPagination.current = pag.current
  dailyPagination.pageSize = pag.pageSize
  loadDailyReports()
}

// 创建日报
const handleCreate = () => {
  if (activeTab.value === 'daily') {
    dailyModalTitle.value = '新增日报'
    dailyFormData.id = undefined
    dailyFormData.date = dayjs()
    dailyFormData.content = ''
    dailyFormData.hours = 0
    dailyFormData.project_id = undefined
    dailyFormData.task_ids = []
    dailyFormData.approver_ids = []
    dailyModalVisible.value = true
  } else {
    weeklyModalTitle.value = '新增周报'
    weeklyFormData.id = undefined
    // 默认设置为本周
    const today = dayjs()
    weeklyFormData.week_start = today.startOf('week').add(1, 'day') // 周一
    weeklyFormData.week_end = today.endOf('week').add(1, 'day') // 周日
    weeklyFormData.summary = ''
    weeklyFormData.next_week_plan = ''
    weeklyFormData.project_id = undefined
    weeklyFormData.task_ids = []
    weeklyFormData.approver_ids = []
    weeklyModalVisible.value = true
  }
}

// 清理内容中的失效 blob URL
const cleanBlobUrls = (content: string): string => {
  if (!content) return content
  // 移除所有 blob URL 的图片标记
  // 匹配格式: ![alt](blob:...)
  return content.replace(/!\[([^\]]*)\]\(blob:[^)]+\)/g, '')
}

// 编辑日报
const handleDailyEdit = (record: DailyReport) => {
  dailyModalTitle.value = '编辑日报'
  dailyFormData.id = record.id
  dailyFormData.date = dayjs(record.date)
  // 清理失效的 blob URL
  dailyFormData.content = cleanBlobUrls(record.content || '')
  dailyFormData.hours = record.hours
  dailyFormData.project_id = record.project_id
  dailyFormData.task_ids = record.tasks?.map(t => t.id) || []
  dailyFormData.approver_ids = record.approvers?.map(a => a.id) || []
  dailyModalVisible.value = true
  if (dailyFormData.project_id) {
    loadTasksForProject()
  }
}

// 提交日报表单
const handleDailySubmitForm = async () => {
  try {
    await dailyFormRef.value.validate()
    
    // 上传Markdown编辑器中的本地图片
    let content = dailyFormData.content || ''
    if (dailyContentEditorRef.value && dailyFormData.project_id) {
      try {
        content = await dailyContentEditorRef.value.uploadLocalImages(async (file: File, projectId: number) => {
          const attachment = await uploadFile(file, projectId)
          return attachment
        })
      } catch (error: any) {
        console.error('上传图片失败:', error)
        message.warning('部分图片上传失败，请检查')
      }
    }
    
    const data: CreateDailyReportRequest = {
      date: dailyFormData.date!.format('YYYY-MM-DD'),
      content: content, // 使用已上传图片的content
      hours: dailyFormData.hours,
      project_id: dailyFormData.project_id,
      task_ids: dailyFormData.task_ids && dailyFormData.task_ids.length > 0 ? dailyFormData.task_ids : undefined,
      approver_ids: dailyFormData.approver_ids && dailyFormData.approver_ids.length > 0 ? dailyFormData.approver_ids : undefined
    }
    if (dailyFormData.id) {
      await updateDailyReport(dailyFormData.id, data)
      message.success('更新成功')
    } else {
      await createDailyReport(data)
      message.success('创建成功')
    }
    dailyModalVisible.value = false
    loadDailyReports()
    if (activeTab.value === 'approval') {
      loadApprovalDailyReports()
    }
    loadPendingApprovalCount()
  } catch (error: any) {
    if (error.errorFields) {
      return
    }
    message.error(error.message || '操作失败')
  }
}

// 提交日报（状态改为已提交）
const handleDailySubmit = async (record: DailyReport) => {
  try {
    await updateDailyReportStatus(record.id, { status: 'submitted' })
    message.success('提交成功')
    loadDailyReports()
    if (activeTab.value === 'approval') {
      loadApprovalDailyReports()
    }
    loadPendingApprovalCount()
  } catch (error: any) {
    message.error(error.message || '提交失败')
  }
}

// 删除日报
const handleDailyDelete = async (id: number) => {
  try {
    await deleteDailyReport(id)
    message.success('删除成功')
    loadDailyReports()
    if (activeTab.value === 'approval') {
      loadApprovalDailyReports()
    }
    loadPendingApprovalCount()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

// 取消日报表单
const handleDailyCancel = () => {
  dailyFormRef.value?.resetFields()
  availableTasks.value = []
}

// 周报搜索
const handleWeeklySearch = () => {
  weeklyPagination.current = 1
  loadWeeklyReports()
}

// 周报重置
const handleWeeklyReset = () => {
  weeklySearchForm.status = undefined
  weeklySearchForm.start_date = undefined
  weeklySearchForm.end_date = undefined
  weeklySearchForm.project_id = undefined
  weeklyPagination.current = 1
  loadWeeklyReports()
}

// 周报表格变化
const handleWeeklyTableChange = (pag: any) => {
  weeklyPagination.current = pag.current
  weeklyPagination.pageSize = pag.pageSize
  loadWeeklyReports()
}

// 编辑周报
const handleWeeklyEdit = (record: WeeklyReport) => {
  weeklyModalTitle.value = '编辑周报'
  weeklyFormData.id = record.id
  weeklyFormData.week_start = dayjs(record.week_start)
  weeklyFormData.week_end = dayjs(record.week_end)
  // 清理失效的 blob URL
  weeklyFormData.summary = cleanBlobUrls(record.summary || '')
  weeklyFormData.next_week_plan = cleanBlobUrls(record.next_week_plan || '')
  weeklyFormData.project_id = record.project_id
  weeklyFormData.task_ids = record.tasks?.map(t => t.id) || []
  weeklyFormData.approver_ids = record.approvers?.map(a => a.id) || []
  weeklyModalVisible.value = true
  if (weeklyFormData.project_id) {
    loadTasksForProject()
  }
}

// 提交周报表单
const handleWeeklySubmitForm = async () => {
  try {
    await weeklyFormRef.value.validate()
    
    // 上传Markdown编辑器中的本地图片
    let summary = weeklyFormData.summary || ''
    let nextWeekPlan = weeklyFormData.next_week_plan || ''
    
    if (weeklyFormData.project_id) {
      // 上传工作总结中的图片
      if (weeklySummaryEditorRef.value) {
        try {
          summary = await weeklySummaryEditorRef.value.uploadLocalImages(async (file: File, projectId: number) => {
            const attachment = await uploadFile(file, projectId)
            return attachment
          })
        } catch (error: any) {
          console.error('上传工作总结图片失败:', error)
          message.warning('部分图片上传失败，请检查')
        }
      }
      
      // 上传下周计划中的图片
      if (weeklyPlanEditorRef.value) {
        try {
          nextWeekPlan = await weeklyPlanEditorRef.value.uploadLocalImages(async (file: File, projectId: number) => {
            const attachment = await uploadFile(file, projectId)
            return attachment
          })
        } catch (error: any) {
          console.error('上传下周计划图片失败:', error)
          message.warning('部分图片上传失败，请检查')
        }
      }
    }
    
    const data: CreateWeeklyReportRequest = {
      week_start: weeklyFormData.week_start!.format('YYYY-MM-DD'),
      week_end: weeklyFormData.week_end!.format('YYYY-MM-DD'),
      summary: summary, // 使用已上传图片的summary
      next_week_plan: nextWeekPlan, // 使用已上传图片的next_week_plan
      project_id: weeklyFormData.project_id,
      task_ids: weeklyFormData.task_ids && weeklyFormData.task_ids.length > 0 ? weeklyFormData.task_ids : undefined,
      approver_ids: weeklyFormData.approver_ids && weeklyFormData.approver_ids.length > 0 ? weeklyFormData.approver_ids : undefined
    }
    if (weeklyFormData.id) {
      await updateWeeklyReport(weeklyFormData.id, data)
      message.success('更新成功')
    } else {
      await createWeeklyReport(data)
      message.success('创建成功')
    }
    weeklyModalVisible.value = false
    loadWeeklyReports()
    if (activeTab.value === 'approval') {
      loadApprovalWeeklyReports()
    }
    loadPendingApprovalCount()
  } catch (error: any) {
    if (error.errorFields) {
      return
    }
    message.error(error.message || '操作失败')
  }
}

// 提交周报（状态改为已提交）
const handleWeeklySubmit = async (record: WeeklyReport) => {
  try {
    await updateWeeklyReportStatus(record.id, { status: 'submitted' })
    message.success('提交成功')
    loadWeeklyReports()
    if (activeTab.value === 'approval') {
      loadApprovalWeeklyReports()
    }
    loadPendingApprovalCount()
  } catch (error: any) {
    message.error(error.message || '提交失败')
  }
}


// 删除周报
const handleWeeklyDelete = async (id: number) => {
  try {
    await deleteWeeklyReport(id)
    message.success('删除成功')
    loadWeeklyReports()
    if (activeTab.value === 'approval') {
      loadApprovalWeeklyReports()
    }
    loadPendingApprovalCount()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

// 取消周报表单
const handleWeeklyCancel = () => {
  weeklyFormRef.value?.resetFields()
  availableTasks.value = []
}

// 审批相关状态
const dailyApproveModalVisible = ref(false)
const dailyApproveData = ref<DailyReport | null>(null)
const dailyApproveComment = ref('')

const weeklyApproveModalVisible = ref(false)
const weeklyApproveData = ref<WeeklyReport | null>(null)
const weeklyApproveComment = ref('')

// 判断是否可以审批日报
const canApproveDaily = (record: DailyReport): boolean => {
  if (record.status !== 'submitted') return false
  // 检查当前用户是否是审批人
  const currentUser = authStore.user
  if (!currentUser) return false
  if (isAdmin.value) return true
  return record.approvers?.some(a => a.id === currentUser.id) || false
}

// 判断是否可以审批周报
const canApproveWeekly = (record: WeeklyReport): boolean => {
  if (record.status !== 'submitted') return false
  // 检查当前用户是否是审批人
  const currentUser = authStore.user
  if (!currentUser) return false
  if (isAdmin.value) return true
  return record.approvers?.some(a => a.id === currentUser.id) || false
}

// 打开日报审批弹窗
const handleDailyApproveClick = async (record: DailyReport) => {
  try {
    // 获取完整的报告详情
    const fullRecord = await getDailyReport(record.id)
    dailyApproveData.value = fullRecord
    // 检查是否已有审批记录
    const currentUser = authStore.user
    if (currentUser) {
      const existingApproval = fullRecord.approval_records?.find(r => r.approver_id === currentUser.id)
      if (existingApproval) {
        dailyApproveComment.value = existingApproval.comment || ''
      } else {
        dailyApproveComment.value = ''
      }
    }
    dailyApproveModalVisible.value = true
  } catch (error: any) {
    message.error(error.message || '加载报告详情失败')
  }
}

// 提交日报审批（通过）
const handleDailyApproveSubmit = async () => {
  if (!dailyApproveData.value) return
  try {
    await approveDailyReport(dailyApproveData.value.id, {
      status: 'approved',
      comment: dailyApproveComment.value
    })
    message.success('审批通过')
    dailyApproveModalVisible.value = false
    loadDailyReports()
    if (activeTab.value === 'approval') {
      loadApprovalDailyReports()
    }
    loadPendingApprovalCount()
  } catch (error: any) {
    message.error(error.message || '审批失败')
  }
}

// 拒绝日报审批
const handleDailyApproveReject = async () => {
  if (!dailyApproveData.value) return
  try {
    await approveDailyReport(dailyApproveData.value.id, {
      status: 'rejected',
      comment: dailyApproveComment.value
    })
    message.success('已拒绝')
    dailyApproveModalVisible.value = false
    loadDailyReports()
    if (activeTab.value === 'approval') {
      loadApprovalDailyReports()
    }
    loadPendingApprovalCount()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  }
}

// 取消日报审批
const handleDailyApproveCancel = () => {
  dailyApproveModalVisible.value = false
  dailyApproveData.value = null
  dailyApproveComment.value = ''
}

// 打开周报审批弹窗
const handleWeeklyApproveClick = async (record: WeeklyReport) => {
  try {
    // 获取完整的报告详情
    const fullRecord = await getWeeklyReport(record.id)
    weeklyApproveData.value = fullRecord
    // 检查是否已有审批记录
    const currentUser = authStore.user
    if (currentUser) {
      const existingApproval = fullRecord.approval_records?.find(r => r.approver_id === currentUser.id)
      if (existingApproval) {
        weeklyApproveComment.value = existingApproval.comment || ''
      } else {
        weeklyApproveComment.value = ''
      }
    }
    weeklyApproveModalVisible.value = true
  } catch (error: any) {
    message.error(error.message || '加载报告详情失败')
  }
}

// 提交周报审批（通过）
const handleWeeklyApproveSubmit = async () => {
  if (!weeklyApproveData.value) return
  try {
    await approveWeeklyReport(weeklyApproveData.value.id, {
      status: 'approved',
      comment: weeklyApproveComment.value
    })
    message.success('审批通过')
    weeklyApproveModalVisible.value = false
    loadWeeklyReports()
    if (activeTab.value === 'approval') {
      loadApprovalWeeklyReports()
    }
    loadPendingApprovalCount()
  } catch (error: any) {
    message.error(error.message || '审批失败')
  }
}

// 拒绝周报审批
const handleWeeklyApproveReject = async () => {
  if (!weeklyApproveData.value) return
  try {
    await approveWeeklyReport(weeklyApproveData.value.id, {
      status: 'rejected',
      comment: weeklyApproveComment.value
    })
    message.success('已拒绝')
    weeklyApproveModalVisible.value = false
    loadWeeklyReports()
    if (activeTab.value === 'approval') {
      loadApprovalWeeklyReports()
    }
    loadPendingApprovalCount()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  }
}

// 取消周报审批
const handleWeeklyApproveCancel = () => {
  weeklyApproveModalVisible.value = false
  weeklyApproveData.value = null
  weeklyApproveComment.value = ''
}

// 渲染Markdown
const renderMarkdown = (content: string): string => {
  if (!content || content.trim() === '') {
    return '<p class="empty-text">暂无内容</p>'
  }
  
  // 先清理失效的 blob URL（避免显示错误）
  const cleanedContent = cleanBlobUrls(content)
  
  let html = marked.parse(cleanedContent) as string
  
  // 处理图片URL，确保相对路径的图片能正确显示
  // 将相对路径的图片URL转换为绝对路径
  html = html.replace(/<img([^>]*)\ssrc=["']([^"']+)["']([^>]*)>/gi, (match, before, src, after) => {
    // 如果是相对路径（以 /uploads/ 开头），保持不变（Vite代理会处理）
    // 如果是 blob: URL，移除（因为已经失效）
    if (src.startsWith('blob:')) {
      return '' // 移除失效的 blob URL 图片
    }
    // 如果是完整的 HTTP/HTTPS URL，保持不变
    if (src.startsWith('http://') || src.startsWith('https://')) {
      return match
    }
    // 如果是相对路径（以 /uploads/ 开头），保持不变
    if (src.startsWith('/uploads/')) {
      return match
    }
    // 如果是其他相对路径，可能需要添加 /uploads/ 前缀
    return `<img${before} src="${src.startsWith('/') ? src : `/uploads/${src}`}"${after}>`
  })
  
  return html
}

// 获取状态颜色
const getStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    draft: 'default',
    submitted: 'processing',
    approved: 'success',
    rejected: 'error'
  }
  return colors[status] || 'default'
}

// 获取状态文本
const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    draft: '草稿',
    submitted: '已提交',
    approved: '已审批',
    rejected: '已拒绝'
  }
  return texts[status] || status
}

// 加载审批日报列表
const loadApprovalDailyReports = async () => {
  approvalDailyLoading.value = true
  try {
    const params: any = {
      page: approvalDailyPagination.current,
      size: approvalDailyPagination.pageSize,
      for_approval: true // 获取需要审批的报告
    }
    if (approvalDailySearchForm.start_date && approvalDailySearchForm.start_date.isValid()) {
      params.start_date = approvalDailySearchForm.start_date.format('YYYY-MM-DD')
    }
    if (approvalDailySearchForm.end_date && approvalDailySearchForm.end_date.isValid()) {
      params.end_date = approvalDailySearchForm.end_date.format('YYYY-MM-DD')
    }
    const response = await getDailyReports(params)
    const currentUserId = authStore.user?.id
    
    // 根据审批状态过滤
    let filtered = response.list
    if (approvalDailySearchForm.approval_status) {
      filtered = filtered.filter((report: DailyReport) => {
        if (!report.approval_records || report.approval_records.length === 0) {
          return approvalDailySearchForm.approval_status === 'pending'
        }
        const myApproval = report.approval_records.find((r: any) => r.approver_id === currentUserId)
        if (!myApproval) {
          return approvalDailySearchForm.approval_status === 'pending'
        }
        return myApproval.status === approvalDailySearchForm.approval_status
      })
    }
    approvalDailyReports.value = filtered
    // 如果前端过滤了，使用过滤后的数量；否则使用后端返回的总数
    approvalDailyPagination.total = approvalDailySearchForm.approval_status ? filtered.length : response.total
  } catch (error: any) {
    message.error(error.message || '加载审批日报列表失败')
  } finally {
    approvalDailyLoading.value = false
  }
}

// 加载审批周报列表
const loadApprovalWeeklyReports = async () => {
  approvalWeeklyLoading.value = true
  try {
    const params: any = {
      page: approvalWeeklyPagination.current,
      size: approvalWeeklyPagination.pageSize,
      for_approval: true // 获取需要审批的报告
    }
    if (approvalWeeklySearchForm.start_date && approvalWeeklySearchForm.start_date.isValid()) {
      params.start_date = approvalWeeklySearchForm.start_date.format('YYYY-MM-DD')
    }
    if (approvalWeeklySearchForm.end_date && approvalWeeklySearchForm.end_date.isValid()) {
      params.end_date = approvalWeeklySearchForm.end_date.format('YYYY-MM-DD')
    }
    const response = await getWeeklyReports(params)
    const currentUserId = authStore.user?.id
    
    // 根据审批状态过滤
    let filtered = response.list
    if (approvalWeeklySearchForm.approval_status) {
      filtered = filtered.filter((report: WeeklyReport) => {
        if (!report.approval_records || report.approval_records.length === 0) {
          return approvalWeeklySearchForm.approval_status === 'pending'
        }
        const myApproval = report.approval_records.find((r: any) => r.approver_id === currentUserId)
        if (!myApproval) {
          return approvalWeeklySearchForm.approval_status === 'pending'
        }
        return myApproval.status === approvalWeeklySearchForm.approval_status
      })
    }
    approvalWeeklyReports.value = filtered
    // 如果前端过滤了，使用过滤后的数量；否则使用后端返回的总数
    approvalWeeklyPagination.total = approvalWeeklySearchForm.approval_status ? filtered.length : response.total
  } catch (error: any) {
    message.error(error.message || '加载审批周报列表失败')
  } finally {
    approvalWeeklyLoading.value = false
  }
}

// 获取审批状态颜色（用于审批列表，显示当前用户的审批状态）
const getApprovalStatusColor = (record: DailyReport | WeeklyReport) => {
  const currentUserId = authStore.user?.id
  if (!record.approval_records || record.approval_records.length === 0) {
    return 'orange' // 待审批
  }
  const myApproval = record.approval_records.find((r: any) => r.approver_id === currentUserId)
  if (!myApproval) {
    return 'orange' // 待审批
  }
  if (myApproval.status === 'approved') {
    return 'success' // 已通过
  } else if (myApproval.status === 'rejected') {
    return 'error' // 已拒绝
  }
  return 'orange' // 待审批
}

// 获取审批状态文本（用于审批列表，显示当前用户的审批状态）
const getApprovalStatusText = (record: DailyReport | WeeklyReport) => {
  const currentUserId = authStore.user?.id
  if (!record.approval_records || record.approval_records.length === 0) {
    return '待审批'
  }
  const myApproval = record.approval_records.find((r: any) => r.approver_id === currentUserId)
  if (!myApproval) {
    return '待审批'
  }
  if (myApproval.status === 'approved') {
    return '已通过'
  } else if (myApproval.status === 'rejected') {
    return '已拒绝'
  }
  return '待审批'
}

// 获取指定审批人的状态颜色
const getApproverStatusColor = (record: DailyReport | WeeklyReport, approverId: number) => {
  if (!record.approval_records || record.approval_records.length === 0) {
    return 'orange' // 待审批
  }
  const approval = record.approval_records.find((r: any) => r.approver_id === approverId)
  if (!approval) {
    return 'orange' // 待审批
  }
  if (approval.status === 'approved') {
    return 'success' // 已通过
  } else if (approval.status === 'rejected') {
    return 'error' // 已拒绝
  }
  return 'orange' // 待审批
}

// 获取指定审批人的状态文本
const getApproverStatusText = (record: DailyReport | WeeklyReport, approverId: number) => {
  if (!record.approval_records || record.approval_records.length === 0) {
    return '待审批'
  }
  const approval = record.approval_records.find((r: any) => r.approver_id === approverId)
  if (!approval) {
    return '待审批'
  }
  if (approval.status === 'approved') {
    return '已通过'
  } else if (approval.status === 'rejected') {
    return '已拒绝'
  }
  return '待审批'
}

// 审批日报搜索
const handleApprovalDailySearch = () => {
  approvalDailyPagination.current = 1
  loadApprovalDailyReports()
}

// 审批日报重置
const handleApprovalDailyReset = () => {
  approvalDailySearchForm.approval_status = undefined
  approvalDailySearchForm.start_date = undefined
  approvalDailySearchForm.end_date = undefined
  approvalDailyPagination.current = 1
  loadApprovalDailyReports()
}

// 审批日报表格变化
const handleApprovalDailyTableChange = (pag: any) => {
  approvalDailyPagination.current = pag.current
  approvalDailyPagination.pageSize = pag.pageSize
  loadApprovalDailyReports()
}

// 审批周报搜索
const handleApprovalWeeklySearch = () => {
  approvalWeeklyPagination.current = 1
  loadApprovalWeeklyReports()
}

// 审批周报重置
const handleApprovalWeeklyReset = () => {
  approvalWeeklySearchForm.approval_status = undefined
  approvalWeeklySearchForm.start_date = undefined
  approvalWeeklySearchForm.end_date = undefined
  approvalWeeklyPagination.current = 1
  loadApprovalWeeklyReports()
}

// 审批周报表格变化
const handleApprovalWeeklyTableChange = (pag: any) => {
  approvalWeeklyPagination.current = pag.current
  approvalWeeklyPagination.pageSize = pag.pageSize
  loadApprovalWeeklyReports()
}

// 查看审批日报详情
const handleApprovalDailyView = async (record: DailyReport) => {
  try {
    const report = await getDailyReport(record.id)
    dailyApproveData.value = report
    dailyApproveComment.value = ''
    dailyApproveModalVisible.value = true
  } catch (error: any) {
    message.error(error.message || '加载日报详情失败')
  }
}

// 查看审批周报详情
const handleApprovalWeeklyView = async (record: WeeklyReport) => {
  try {
    const report = await getWeeklyReport(record.id)
    weeklyApproveData.value = report
    weeklyApproveComment.value = ''
    weeklyApproveModalVisible.value = true
  } catch (error: any) {
    message.error(error.message || '加载周报详情失败')
  }
}

// 加载待审批数量
const loadPendingApprovalCount = async () => {
  try {
    const dashboardData = await getDashboard()
    pendingApprovalCount.value = dashboardData.reports.pending_approval || 0
  } catch (error: any) {
    console.error('加载待审批数量失败:', error)
  }
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

// 监听项目变化，重新加载任务
watch(() => dailyFormData.project_id, () => {
  dailyFormData.task_ids = []
  if (dailyFormData.project_id) {
    loadTasksForProject()
  } else {
    availableTasks.value = []
  }
})

watch(() => weeklyFormData.project_id, () => {
  weeklyFormData.task_ids = []
  if (weeklyFormData.project_id) {
    loadTasksForProject()
  } else {
    availableTasks.value = []
  }
})

onMounted(() => {
  // 读取路由查询参数
  // 注意：状态字段默认保持为空，不从路由参数自动设置
  if (route.query.tab) {
    activeTab.value = route.query.tab as 'daily' | 'weekly' | 'approval'
    if (activeTab.value === 'approval' && route.query.subTab) {
      approvalSubTab.value = route.query.subTab as 'daily-approval' | 'weekly-approval'
    }
  }
  
  loadDailyReports()
  loadProjects()
  loadUsers()
  loadPendingApprovalCount()
})

// 监听标签页切换，刷新列表和待审批数量
watch(activeTab, () => {
  if (activeTab.value === 'daily') {
    loadDailyReports()
  } else if (activeTab.value === 'weekly') {
    loadWeeklyReports()
  } else if (activeTab.value === 'approval') {
    handleApprovalSubTabChange(approvalSubTab.value)
    loadPendingApprovalCount()
  }
})

// 监听路由变化，从其他页面返回时刷新列表
watch(() => route.query, () => {
  if (route.query.tab) {
    const tab = route.query.tab as string
    if (tab === 'approval') {
      activeTab.value = 'approval'
      if (route.query.subTab) {
        approvalSubTab.value = route.query.subTab as 'daily-approval' | 'weekly-approval'
      }
    } else if (tab === 'daily' || tab === 'weekly') {
      activeTab.value = tab as 'daily' | 'weekly'
    }
  }
}, { immediate: false })
</script>

<style scoped>
.report-management {
  min-height: 100vh;
}

.content {
  padding: 24px;
  background: #f0f2f5;
}

.content-inner {
  max-width: 100%;
  margin: 0 auto;
  width: 100%;
}

/* 审批弹窗中的Markdown渲染样式 */
:deep(.markdown-preview) {
  word-wrap: break-word;
  line-height: 1.6;
}

:deep(.markdown-preview p) {
  margin-bottom: 8px;
}

:deep(.markdown-preview h1),
:deep(.markdown-preview h2),
:deep(.markdown-preview h3) {
  margin-top: 16px;
  margin-bottom: 8px;
  font-weight: 600;
}

:deep(.markdown-preview code) {
  background-color: #f6f8fa;
  padding: 2px 4px;
  border-radius: 3px;
  font-size: 85%;
}

:deep(.markdown-preview pre) {
  background-color: #f6f8fa;
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
}

:deep(.markdown-preview ul),
:deep(.markdown-preview ol) {
  padding-left: 24px;
  margin-bottom: 8px;
}

:deep(.empty-text) {
  color: #999;
  font-style: italic;
}

/* 审批弹窗中的图片样式 */
:deep(.markdown-preview img),
div[v-html] :deep(img) {
  max-width: 100%;
  height: auto;
  display: block;
  margin: 16px 0;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* 确保审批弹窗中的图片容器可以滚动 */
div[v-html] {
  word-wrap: break-word;
}
</style>


