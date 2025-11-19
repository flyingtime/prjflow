import request from '../utils/request'

export interface PlanExecution {
  id: number
  name: string
  description?: string
  status: 'pending' | 'in_progress' | 'completed' | 'cancelled'
  progress: number
  plan_id: number
  plan?: Plan
  start_date?: string
  end_date?: string
  assignee_id?: number
  assignee?: any
  task_id?: number
  task?: any
  created_at?: string
  updated_at?: string
}

export interface Plan {
  id: number
  name: string
  description?: string
  type: 'project_plan'
  status: 'draft' | 'active' | 'completed' | 'cancelled'
  start_date?: string
  end_date?: string
  project_id?: number
  project?: any
  creator_id: number
  creator?: any
  executions?: PlanExecution[]
  progress?: number
  created_at?: string
  updated_at?: string
}

export interface PlanListResponse {
  list: Plan[]
  total: number
  page: number
  page_size: number
}

export interface CreatePlanRequest {
  name: string
  description?: string
  type: 'project_plan'
  status?: 'draft' | 'active' | 'completed' | 'cancelled'
  project_id?: number
  start_date?: string
  end_date?: string
}

export interface CreatePlanExecutionRequest {
  name: string
  description?: string
  status?: 'pending' | 'in_progress' | 'completed' | 'cancelled'
  progress?: number
  start_date?: string
  end_date?: string
  assignee_id?: number
  task_id?: number
}

export interface UpdatePlanStatusRequest {
  status: 'draft' | 'active' | 'completed' | 'cancelled'
}

export interface UpdatePlanExecutionStatusRequest {
  status: 'pending' | 'in_progress' | 'completed' | 'cancelled'
}

export interface UpdatePlanExecutionProgressRequest {
  progress: number
}

// 计划相关API
export const getPlans = async (params?: {
  keyword?: string
  type?: string
  status?: string
  project_id?: number
  creator_id?: number
  page?: number
  page_size?: number
}): Promise<PlanListResponse> => {
  return request.get('/plans', { params })
}

export const getPlan = async (id: number): Promise<Plan> => {
  return request.get(`/plans/${id}`)
}

export const createPlan = async (data: CreatePlanRequest): Promise<Plan> => {
  return request.post('/plans', data)
}

export const updatePlan = async (id: number, data: Partial<CreatePlanRequest>): Promise<Plan> => {
  return request.put(`/plans/${id}`, data)
}

export const deletePlan = async (id: number): Promise<void> => {
  return request.delete(`/plans/${id}`)
}

export const updatePlanStatus = async (id: number, data: UpdatePlanStatusRequest): Promise<Plan> => {
  return request.patch(`/plans/${id}/status`, data)
}

// 计划执行相关API
export const getPlanExecutions = async (planId: number, params?: {
  status?: string
  assignee_id?: number
}): Promise<PlanExecution[]> => {
  return request.get(`/plans/${planId}/executions`, { params })
}

export const getPlanExecution = async (planId: number, executionId: number): Promise<PlanExecution> => {
  return request.get(`/plans/${planId}/executions/${executionId}`)
}

export const createPlanExecution = async (planId: number, data: CreatePlanExecutionRequest): Promise<PlanExecution> => {
  return request.post(`/plans/${planId}/executions`, data)
}

export const updatePlanExecution = async (planId: number, executionId: number, data: Partial<CreatePlanExecutionRequest>): Promise<PlanExecution> => {
  return request.put(`/plans/${planId}/executions/${executionId}`, data)
}

export const deletePlanExecution = async (planId: number, executionId: number): Promise<void> => {
  return request.delete(`/plans/${planId}/executions/${executionId}`)
}

export const updatePlanExecutionStatus = async (planId: number, executionId: number, data: UpdatePlanExecutionStatusRequest): Promise<PlanExecution> => {
  return request.patch(`/plans/${planId}/executions/${executionId}/status`, data)
}

export const updatePlanExecutionProgress = async (planId: number, executionId: number, data: UpdatePlanExecutionProgressRequest): Promise<PlanExecution> => {
  return request.patch(`/plans/${planId}/executions/${executionId}/progress`, data)
}

