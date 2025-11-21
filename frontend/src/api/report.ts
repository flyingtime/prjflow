import request from '../utils/request'

export interface DailyReport {
  id: number
  date: string
  content?: string
  hours: number
  status: 'draft' | 'submitted' | 'approved'
  user_id: number
  user?: any
  project_id?: number
  project?: any
  task_id?: number
  task?: any
  created_at?: string
  updated_at?: string
}

export interface WeeklyReport {
  id: number
  week_start: string
  week_end: string
  summary?: string
  next_week_plan?: string
  status: 'draft' | 'submitted' | 'approved'
  user_id: number
  user?: any
  project_id?: number
  project?: any
  task_id?: number
  task?: any
  created_at?: string
  updated_at?: string
}

export interface DailyReportListResponse {
  list: DailyReport[]
  total: number
  page: number
  page_size: number
}

export interface WeeklyReportListResponse {
  list: WeeklyReport[]
  total: number
  page: number
  page_size: number
}

export interface CreateDailyReportRequest {
  date: string
  content?: string
  hours?: number
  status?: 'draft' | 'submitted' | 'approved'
  project_id?: number
  task_id?: number
}

export interface UpdateDailyReportRequest {
  date?: string
  content?: string
  hours?: number
  status?: 'draft' | 'submitted' | 'approved'
  project_id?: number
  task_id?: number
}

export interface CreateWeeklyReportRequest {
  week_start: string
  week_end: string
  summary?: string
  next_week_plan?: string
  status?: 'draft' | 'submitted' | 'approved'
  project_id?: number
  task_id?: number
}

export interface UpdateWeeklyReportRequest {
  week_start?: string
  week_end?: string
  summary?: string
  next_week_plan?: string
  status?: 'draft' | 'submitted' | 'approved'
  project_id?: number
  task_id?: number
}

export interface UpdateReportStatusRequest {
  status: 'draft' | 'submitted' | 'approved'
}

// 日报相关API
export const getDailyReports = async (params?: {
  status?: string
  start_date?: string
  end_date?: string
  project_id?: number
  user_id?: number
  page?: number
  size?: number
}): Promise<DailyReportListResponse> => {
  return request.get('/daily-reports', { params })
}

export const getDailyReport = async (id: number): Promise<DailyReport> => {
  return request.get(`/daily-reports/${id}`)
}

export const createDailyReport = async (data: CreateDailyReportRequest): Promise<DailyReport> => {
  return request.post('/daily-reports', data)
}

export const updateDailyReport = async (id: number, data: UpdateDailyReportRequest): Promise<DailyReport> => {
  return request.put(`/daily-reports/${id}`, data)
}

export const deleteDailyReport = async (id: number): Promise<void> => {
  return request.delete(`/daily-reports/${id}`)
}

export const updateDailyReportStatus = async (id: number, data: UpdateReportStatusRequest): Promise<DailyReport> => {
  return request.patch(`/daily-reports/${id}/status`, data)
}

// 周报相关API
export const getWeeklyReports = async (params?: {
  status?: string
  start_date?: string
  end_date?: string
  project_id?: number
  user_id?: number
  page?: number
  size?: number
}): Promise<WeeklyReportListResponse> => {
  return request.get('/weekly-reports', { params })
}

export const getWeeklyReport = async (id: number): Promise<WeeklyReport> => {
  return request.get(`/weekly-reports/${id}`)
}

export const createWeeklyReport = async (data: CreateWeeklyReportRequest): Promise<WeeklyReport> => {
  return request.post('/weekly-reports', data)
}

export const updateWeeklyReport = async (id: number, data: UpdateWeeklyReportRequest): Promise<WeeklyReport> => {
  return request.put(`/weekly-reports/${id}`, data)
}

export const deleteWeeklyReport = async (id: number): Promise<void> => {
  return request.delete(`/weekly-reports/${id}`)
}

export const updateWeeklyReportStatus = async (id: number, data: UpdateReportStatusRequest): Promise<WeeklyReport> => {
  return request.patch(`/weekly-reports/${id}/status`, data)
}

