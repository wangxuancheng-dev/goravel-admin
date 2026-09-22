import { Select } from 'antd'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useAppStore } from '@/stores/app'
import { PRESET_TIMEZONES } from '@/utils/timezoneOptions'

function parseOffsetMinutes(label: string): number {
  const match = label.match(/^UTC([+-])(\d{2}):(\d{2})/)
  if (!match) return Number.POSITIVE_INFINITY
  const sign = match[1] === '-' ? -1 : 1
  const hours = Number.parseInt(match[2], 10)
  const minutes = Number.parseInt(match[3], 10)
  return sign * (hours * 60 + minutes)
}

function formatOffsetLabel(tz: string): string {
  if (!tz) return ''
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
    // ignore errors and fall back to raw name
  }
  return tz
}

export default function TimezoneSwitch() {
  const { t } = useTranslation()
  const timezone = useAppStore((s) => s.timezone)
  const setTimezone = useAppStore((s) => s.setTimezone)

  const options = useMemo(() => {
    const map = new Map(PRESET_TIMEZONES.map((item) => [item.value, item.label]))
    if (timezone && !map.has(timezone)) {
      map.set(timezone, formatOffsetLabel(timezone))
    }
    return Array.from(map.entries())
      .map(([value, label]) => ({ value, label }))
      .sort((a, b) => {
        const diff = parseOffsetMinutes(a.label) - parseOffsetMinutes(b.label)
        return diff !== 0 ? diff : a.label.localeCompare(b.label)
      })
  }, [timezone])

  return (
    <Select
      className="timezone-switch"
      size="small"
      showSearch
      value={timezone}
      placeholder={t('header.timezone')}
      style={{ width: 190 }}
      options={options}
      optionFilterProp="label"
      filterOption={(input, option) =>
        (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
      }
      onChange={(value) => setTimezone(value)}
    />
  )
}
