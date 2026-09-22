/** Preset IANA timezones (UTC-12 ~ UTC+14), shared with header TimezoneSwitch. */
export const PRESET_TIMEZONES = [
  { value: 'Etc/GMT+12', label: 'UTC-12:00 (Etc/GMT+12)' },
  { value: 'Pacific/Pago_Pago', label: 'UTC-11:00 (Pacific/Pago_Pago)' },
  { value: 'Pacific/Honolulu', label: 'UTC-10:00 (Pacific/Honolulu)' },
  { value: 'America/Anchorage', label: 'UTC-09:00 (America/Anchorage)' },
  { value: 'America/Los_Angeles', label: 'UTC-08:00 (America/Los_Angeles)' },
  { value: 'America/Denver', label: 'UTC-07:00 (America/Denver)' },
  { value: 'America/Chicago', label: 'UTC-06:00 (America/Chicago)' },
  { value: 'America/New_York', label: 'UTC-05:00 (America/New_York)' },
  { value: 'America/Halifax', label: 'UTC-04:00 (America/Halifax)' },
  { value: 'America/Argentina/Buenos_Aires', label: 'UTC-03:00 (America/Argentina/Buenos_Aires)' },
  { value: 'Etc/GMT+2', label: 'UTC-02:00 (Etc/GMT+2)' },
  { value: 'Atlantic/Azores', label: 'UTC-01:00 (Atlantic/Azores)' },
  { value: 'UTC', label: 'UTC+00:00 (UTC)' },
  { value: 'Europe/Berlin', label: 'UTC+01:00 (Europe/Berlin)' },
  { value: 'Europe/Athens', label: 'UTC+02:00 (Europe/Athens)' },
  { value: 'Europe/Moscow', label: 'UTC+03:00 (Europe/Moscow)' },
  { value: 'Asia/Dubai', label: 'UTC+04:00 (Asia/Dubai)' },
  { value: 'Asia/Karachi', label: 'UTC+05:00 (Asia/Karachi)' },
  { value: 'Asia/Dhaka', label: 'UTC+06:00 (Asia/Dhaka)' },
  { value: 'Asia/Bangkok', label: 'UTC+07:00 (Asia/Bangkok)' },
  { value: 'Asia/Shanghai', label: 'UTC+08:00 (Asia/Shanghai)' },
  { value: 'Asia/Tokyo', label: 'UTC+09:00 (Asia/Tokyo)' },
  { value: 'Australia/Sydney', label: 'UTC+10:00 (Australia/Sydney)' },
  { value: 'Pacific/Noumea', label: 'UTC+11:00 (Pacific/Noumea)' },
  { value: 'Pacific/Auckland', label: 'UTC+12:00 (Pacific/Auckland)' },
  { value: 'Pacific/Enderbury', label: 'UTC+13:00 (Pacific/Enderbury)' },
  { value: 'Pacific/Kiritimati', label: 'UTC+14:00 (Pacific/Kiritimati)' },
]

export const DEFAULT_TIMEZONE = 'UTC'

export function parseOffsetMinutes(label) {
  if (!label) return Number.POSITIVE_INFINITY
  const match = label.match(/^UTC([+-])(\d{2}):(\d{2})/)
  if (!match) return Number.POSITIVE_INFINITY
  const sign = match[1] === '-' ? -1 : 1
  const hours = Number.parseInt(match[2], 10)
  const minutes = Number.parseInt(match[3], 10)
  return sign * (hours * 60 + minutes)
}

export function formatOffsetLabel(tz) {
  if (!tz) return ''
  const preset = PRESET_TIMEZONES.find((item) => item.value === tz)
  if (preset) return preset.label
  try {
    const formatter = new Intl.DateTimeFormat('en-US', {
      timeZone: tz,
      timeZoneName: 'short',
      hour: '2-digit',
      minute: '2-digit',
    })
    const parts = formatter.formatToParts(new Date())
    const tzName = parts.find((part) => part.type === 'timeZoneName')?.value || ''
    const match = tzName.match(/([+-]\d{1,2})(?::(\d{2}))?/)
    if (match) {
      const sign = match[1].startsWith('-') ? '-' : '+'
      const hours = Math.abs(Number.parseInt(match[1], 10)).toString().padStart(2, '0')
      const minutes = (match[2] || '00').padStart(2, '0')
      return `UTC${sign}${hours}:${minutes} (${tz})`
    }
  } catch {
    // ignore
  }
  return tz
}

export function timezoneSelectOptions(current) {
  const map = new Map(PRESET_TIMEZONES.map((item) => [item.value, item.label]))
  if (current && !map.has(current)) {
    map.set(current, formatOffsetLabel(current))
  }
  return Array.from(map.entries())
    .map(([value, label]) => ({ value, label }))
    .sort((a, b) => {
      const offsetDiff = parseOffsetMinutes(a.label) - parseOffsetMinutes(b.label)
      if (offsetDiff !== 0) return offsetDiff
      return a.label.localeCompare(b.label)
    })
}
