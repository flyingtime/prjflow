import request from '../utils/request'

export interface TestReport {
  id: number
  title: string
  content?: string
  result?: 'passed' | 'failed' | 'blocked'
  summary?: string
  creator_id: number
  creator?: any
  test_cases?: any[]
  created_at?: string
  updated_at?: string
}

export interface TestReportListResponse {
  list: TestReport[]
  total: number
  page: number
  page_size: number
}

export interface CreateTestReportRequest {
  title: string
  content?: string
  result?: 'passed' | 'failed' | 'blocked'
  summary?: string
  test_case_ids?: number[]
}

export interface UpdateTestReportRequest {
  title?: string
  content?: string
  result?: 'passed' | 'failed' | 'blocked'
  summary?: string
  test_case_ids?: number[]
}

export interface TestReportStatistics {
  total: number
  passed: number
  failed: number
  blocked: number
}

// 获取测试报告列表
export const getTestReports = (params?: {
  keyword?: string
  result?: string
  creator_id?: number
  test_case_id?: number
  page?: number
  size?: number
}) => {
  return request.get<TestReportListResponse>('/test-reports', { params })
}

// 获取测试报告详情
export const getTestReport = (id: number) => {
  return request.get<TestReport>(`/test-reports/${id}`)
}

// 创建测试报告
export const createTestReport = (data: CreateTestReportRequest) => {
  return request.post<TestReport>('/test-reports', data)
}

// 更新测试报告
export const updateTestReport = (id: number, data: UpdateTestReportRequest) => {
  return request.put<TestReport>(`/test-reports/${id}`, data)
}

// 删除测试报告
export const deleteTestReport = (id: number) => {
  return request.delete(`/test-reports/${id}`)
}

// 获取测试报告统计
export const getTestReportStatistics = (params?: {
  creator_id?: number
  keyword?: string
}) => {
  return request.get<TestReportStatistics>('/test-reports/statistics', { params })
}

