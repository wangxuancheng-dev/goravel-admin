/**
 * Date/time helpers.
 */

/**
 * Format a Date as YYYY-MM-DD HH:mm:ss (local).
 * @param {Date} date
 * @returns {string}
 */
export function formatDateTime(date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

/**
 * Display timestamps as YYYY-MM-DD HH:mm:ss (keeps wall-clock from ISO/RFC3339).
 * @param {string|number|Date|null|undefined} value
 * @returns {string}
 */
export function formatDateTimeDisplay(value) {
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

/**
 * N days ago as YYYY-MM-DD HH:mm:ss.
 * @param {number} days
 * @param {boolean} setToStartOfDay
 * @returns {string}
 */
export function getDaysAgo(days = 7, setToStartOfDay = true) {
  const date = new Date()
  date.setDate(date.getDate() - days)
  if (setToStartOfDay) {
    date.setHours(0, 0, 0, 0)
  }
  return formatDateTime(date)
}

/**
 * N months ago as YYYY-MM-DD HH:mm:ss.
 * @param {number} months
 * @param {boolean} setToStartOfDay
 * @returns {string}
 */
export function getMonthsAgo(months = 1, setToStartOfDay = true) {
  const date = new Date()
  date.setMonth(date.getMonth() - months)
  if (setToStartOfDay) {
    date.setHours(0, 0, 0, 0)
  }
  return formatDateTime(date)
}

/**
 * N years ago as YYYY-MM-DD HH:mm:ss.
 * @param {number} years
 * @param {boolean} setToStartOfDay
 * @returns {string}
 */
export function getYearsAgo(years = 1, setToStartOfDay = true) {
  const date = new Date()
  date.setFullYear(date.getFullYear() - years)
  if (setToStartOfDay) {
    date.setHours(0, 0, 0, 0)
  }
  return formatDateTime(date)
}

/**
 * Relative past time by days/months/years.
 * @param {{ days?: number, months?: number, years?: number, setToStartOfDay?: boolean }} options
 * @returns {string}
 * @example
 * getTimeAgo({ days: 7 })
 * getTimeAgo({ months: 1 })
 * getTimeAgo({ years: 1 })
 * getTimeAgo({ days: 7, months: 1 })
 */
export function getTimeAgo({ days = 0, months = 0, years = 0, setToStartOfDay = true } = {}) {
  const date = new Date()

  if (years > 0) {
    date.setFullYear(date.getFullYear() - years)
  }
  if (months > 0) {
    date.setMonth(date.getMonth() - months)
  }
  if (days > 0) {
    date.setDate(date.getDate() - days)
  }

  if (setToStartOfDay) {
    date.setHours(0, 0, 0, 0)
  }

  return formatDateTime(date)
}

/** @returns {string} */
export function getSevenDaysAgo() {
  return getDaysAgo(7, true)
}

/** @returns {string} */
export function getOneMonthAgo() {
  return getMonthsAgo(1, true)
}

/** @returns {string} */
export function getThreeMonthsAgo() {
  return getMonthsAgo(3, true)
}