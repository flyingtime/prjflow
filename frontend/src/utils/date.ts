import dayjs from 'dayjs'

/**
 * 格式化日期时间为 "年-月-日 时:分:秒" 格式
 * @param date 日期字符串或Date对象或dayjs对象
 * @returns 格式化后的日期时间字符串，格式：YYYY-MM-DD HH:mm:ss
 */
export function formatDateTime(date: string | Date | dayjs.Dayjs | null | undefined): string {
  if (!date) return '-'
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

/**
 * 格式化日期为 "年-月-日" 格式
 * @param date 日期字符串或Date对象或dayjs对象
 * @returns 格式化后的日期字符串，格式：YYYY-MM-DD
 */
export function formatDate(date: string | Date | dayjs.Dayjs | null | undefined): string {
  if (!date) return '-'
  return dayjs(date).format('YYYY-MM-DD')
}

