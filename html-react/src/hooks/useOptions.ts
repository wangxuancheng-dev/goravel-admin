import { useCallback, useEffect, useState } from 'react'
import { getOptions, type OptionItem } from '@/api/option'
import logger from '@/utils/logger'

export type OptionTreeNode = {
  title: string
  value: number | string
  children?: OptionTreeNode[]
}

/** Prefer `options` tree (id/name/label); fall back to `list` only if needed. */
export function normalizeOptions(data: unknown): OptionItem[] {
  if (Array.isArray(data)) return data
  if (data && typeof data === 'object') {
    const obj = data as { options?: OptionItem[]; list?: OptionItem[] }
    if (Array.isArray(obj.options)) return obj.options
    if (Array.isArray(obj.list)) return obj.list
  }
  return []
}

/** Map option tree nodes for Ant Design TreeSelect. */
export function toOptionTreeData(items: OptionItem[]): OptionTreeNode[] {
  return items.map((item) => {
    const value = (item.value ?? item.id ?? item.ID) as number | string
    const title = String(item.label ?? item.name ?? item.Name ?? item.title ?? value ?? '')
    const children = item.children
    return {
      title,
      value,
      children: Array.isArray(children) ? toOptionTreeData(children) : undefined,
    }
  })
}

export function mapOptions(
  items: OptionItem[],
  labelKey = 'name',
  valueKey = 'id',
): Array<{ label: string; value: string | number }> {
  return items.map((item) => ({
    label: String(item.label ?? item[labelKey] ?? item.name ?? ''),
    value: (item.value ?? item[valueKey] ?? item.id) as string | number,
  }))
}

export function useOptions(type: string | null, enabled = true) {
  const [options, setOptions] = useState<OptionItem[]>([])
  const [loading, setLoading] = useState(false)

  const reload = useCallback(async () => {
    if (!type || !enabled) return
    setLoading(true)
    try {
      const res = await getOptions(type)
      setOptions(normalizeOptions(res.data))
    } catch (error) {
      logger.error(`load options ${type} failed:`, error)
      setOptions([])
    } finally {
      setLoading(false)
    }
  }, [type, enabled])

  useEffect(() => {
    void reload()
  }, [reload])

  return {
    options,
    selectOptions: mapOptions(options),
    loading,
    reload,
  }
}
