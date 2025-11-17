import request from '../utils/request'

export interface InitStatus {
  initialized: boolean
}

export interface WeChatConfigRequest {
  wechat_app_id: string
  wechat_app_secret: string
}

export interface InitRequest {
  code: string
  state?: string
}

export interface InitResponse {
  message: string
  token: string
  user: {
    id: number
    username: string
    avatar?: string
    roles: string[]
  }
}

export interface QRCodeResponse {
  ticket: string
  qrCodeUrl: string
  expireSeconds: number
}

export const checkInitStatus = async (): Promise<InitStatus> => {
  return request.get('/init/status')
}

export const saveWeChatConfig = async (data: WeChatConfigRequest): Promise<{ message: string }> => {
  return request.post('/init/wechat-config', data)
}

export const getInitQRCode = async (): Promise<QRCodeResponse> => {
  const data = await request.get('/init/qrcode')
  return {
    ticket: data.ticket,
    qrCodeUrl: data.qr_code_url,
    expireSeconds: data.expire_seconds
  }
}

export const initSystem = async (data: InitRequest): Promise<InitResponse> => {
  return request.post('/init', data)
}

