import request from '../utils/request'

export interface VersionInfo {
  version: string
  build_time?: string
  git_commit?: string
  go_version: string
}

export const getVersionInfo = async (): Promise<VersionInfo> => {
  return request.get('/version')
}
