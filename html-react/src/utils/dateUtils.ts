export function formatDateTime(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

/** Display timestamps as YYYY-MM-DD HH:mm:ss (keeps wall-clock from ISO/RFC3339). */
export function formatDateTimeDisplay(value?: string | number | Date | null): string {
  if (value === null || value === undefined || value === '') return '-'
  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? '-' : formatDateTime(value)
  }
  if (typeof value === 'number') {
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? '-' : formatDateTime(d)
  }
  const s = String(value).trim()
  if (!s) return '-'
  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(s)) return s
  const matched = s.match(/^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2}:\d{2})/)
  if (matched) return `${matched[1]} ${matched[2]}`
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  return formatDateTime(d)
}

export function getDaysAgo(days = 7, setToStartOfDay = true): string {
  const date = new Date()
  date.setDate(date.getDate() - days)
  if (setToStartOfDay) {
    date.setHours(0, 0, 0, 0)
  }
  return formatDateTime(date)
}

export function getSevenDaysAgo(): string {
  return getDaysAgo(7, true)
}
