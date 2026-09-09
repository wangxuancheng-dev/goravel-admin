/**
 * Flexible cron humanizer for Goravel 5/6-field expressions.
 * Prefers structured field parsing over exact-string matches.
 */
export function describeCron(cron, t) {
  const expression = String(cron || '').trim()
  if (!expression) return ''

  const parts = expression.split(/\s+/)
  if (parts.length === 6) {
    return describeSixField(parts, t) || t('schedule.cron_custom', { cron: expression })
  }
  if (parts.length !== 5) {
    return t('schedule.cron_custom', { cron: expression })
  }
  return describeFiveField(parts, t) || t('schedule.cron_custom', { cron: expression })
}

function describeSixField(parts, t) {
  const [second, minute, hour, day, month, weekday] = parts
  if (!isWildcard(day) || !isWildcard(month) || !isWildcard(weekday)) return ''

  if (isWildcard(second) && isWildcard(minute) && isWildcard(hour)) {
    return t('schedule.cron_every_second')
  }
  const everySec = parseStep(second)
  if (everySec && isWildcard(minute) && isWildcard(hour)) {
    return t('schedule.cron_every_n_seconds', { n: everySec })
  }
  if (isNumber(second) && isWildcard(minute) && isWildcard(hour)) {
    return t('schedule.cron_every_minute_at_second', { second: Number(second) })
  }
  return ''
}

function describeFiveField(parts, t) {
  const [minute, hour, day, month, weekday] = parts

  // */n * * * *
  if (isWildcard(hour) && isWildcard(day) && isWildcard(month) && isWildcard(weekday)) {
    if (isWildcard(minute)) return t('schedule.cron_every_minute')
    const everyMin = parseStep(minute)
    if (everyMin) return t('schedule.cron_every_n_minutes', { n: everyMin })
    if (isNumber(minute)) {
      return t('schedule.cron_hourly_at_minute', { minute: pad(minute) })
    }
  }

  // 0 */n * * *  /  m */n * * *
  if (isWildcard(day) && isWildcard(month) && isWildcard(weekday)) {
    const everyHour = parseStep(hour)
    if (everyHour && (minute === '0' || isNumber(minute))) {
      if (minute === '0') return t('schedule.cron_every_n_hours', { n: everyHour })
      return t('schedule.cron_every_n_hours_at_minute', { n: everyHour, minute: pad(minute) })
    }
    if (isWildcard(hour) === false && isNumber(hour) && isNumber(minute)) {
      return t('schedule.cron_daily_at', { time: `${pad(hour)}:${pad(minute)}` })
    }
    if (hour === '*' && minute === '0') {
      return t('schedule.cron_hourly')
    }
  }

  // 0 0 1 * *
  if (minute === '0' && hour === '0' && day === '1' && isWildcard(month) && isWildcard(weekday)) {
    return t('schedule.cron_monthly')
  }

  // 0 0 * * 0  weekly Sunday midnight
  if (minute === '0' && hour === '0' && isWildcard(day) && isWildcard(month) && isNumber(weekday)) {
    return t('schedule.cron_weekly_at', {
      weekday: weekdayLabel(Number(weekday), t),
      time: '00:00',
    })
  }

  // m h * * w
  if (isNumber(minute) && isNumber(hour) && isWildcard(day) && isWildcard(month) && isNumber(weekday)) {
    return t('schedule.cron_weekly_at', {
      weekday: weekdayLabel(Number(weekday), t),
      time: `${pad(hour)}:${pad(minute)}`,
    })
  }

  // m h d * *
  if (isNumber(minute) && isNumber(hour) && isNumber(day) && isWildcard(month) && isWildcard(weekday)) {
    return t('schedule.cron_monthly_on_day_at', {
      day: Number(day),
      time: `${pad(hour)}:${pad(minute)}`,
    })
  }

  return ''
}

function isWildcard(value) {
  return value === '*'
}

function isNumber(value) {
  return /^\d+$/.test(value)
}

function parseStep(value) {
  const match = String(value).match(/^\*\/(\d+)$/)
  return match ? Number(match[1]) : 0
}

function pad(value) {
  return String(value).padStart(2, '0')
}

function weekdayLabel(day, t) {
  const keys = [
    'schedule.cron_weekday_sun',
    'schedule.cron_weekday_mon',
    'schedule.cron_weekday_tue',
    'schedule.cron_weekday_wed',
    'schedule.cron_weekday_thu',
    'schedule.cron_weekday_fri',
    'schedule.cron_weekday_sat',
  ]
  return t(keys[day] || 'schedule.cron_weekday_sun')
}
