import request from '../utils/request'

export interface Build {
  id: number
  build_number: string
  status: 'pending' | 'building' | 'success' | 'failed'
  branch?: string
  commit?: string
  build_time?: string
  project_id: number
  project?: any
  creator_id: number
  creator?: any
  version?: any
  created_at?: string
  updated_at?: string
}

export interface BuildListResponse {
  list: Build[]
  total: number
  page: number
  page_size: number
}

export interface CreateBuildRequest {
  build_number: string
  status?: 'pending' | 'building' | 'success' | 'failed'
  branch?: string
  commit?: string
  build_time?: string
  project_id: number
}

export interface UpdateBuildRequest {
  build_number?: string
  status?: 'pending' | 'building' | 'success' | 'failed'
  branch?: string
  commit?: string
  build_time?: string
}

// 获取构建列表
export const getBuilds = (params?: {
  keyword?: string
  project_id?: number
  status?: string
  branch?: string
  creator_id?: number
  page?: number
  size?: number
}) => {
  return request.get<BuildListResponse>('/builds', { params })
}

// 获取构建详情
export const getBuild = (id: number) => {
  return request.get<Build>(`/builds/${id}`)
}

// 创建构建
export const createBuild = (data: CreateBuildRequest) => {
  return request.post<Build>('/builds', data)
}

// 更新构建
export const updateBuild = (id: number, data: UpdateBuildRequest) => {
  return request.put<Build>(`/builds/${id}`, data)
}

// 删除构建
export const deleteBuild = (id: number) => {
  return request.delete(`/builds/${id}`)
}

// 更新构建状态
export const updateBuildStatus = (id: number, status: string) => {
  return request.patch<Build>(`/builds/${id}/status`, { status })
}

