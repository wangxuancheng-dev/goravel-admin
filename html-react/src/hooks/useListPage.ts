import { useCallback, useEffect, useRef, useState } from 'react'
import { buildSearchParams } from '@/utils/buildSearchParams'
import logger from '@/utils/logger'
import type { ApiResponse, ListFetchFn, PaginatedData, PaginationState } from '@/types'

export interface UseListPageOptions<T, S extends Record<string, unknown>> {
  fetchApi: ListFetchFn
  initialSearchForm?: S
  fieldMapping?: Record<string, string>
  defaultSort?: string
  transformData?: ((row: Record<string, unknown>) => T) | null
  onLoadSuccess?: ((rows: T[], res: ApiResponse<PaginatedData>) => void) | null
  buildParams?: ((searchForm: S, baseParams: Record<string, unknown>) => Record<string, unknown>) | null
  onSearch?: (() => void) | null
  onReset?: (() => void) | null
  selectionIdKey?: string
  autoLoad?: boolean
}

function toOrderBy(
  field: string | undefined,
  order: 'ascend' | 'descend' | null | undefined,
  fieldMapping: Record<string, string>,
  defaultSort: string,
) {
  if (!field || !order) return defaultSort
  const mapped = fieldMapping[field] || field
  return `${mapped}:${order === 'ascend' ? 'asc' : 'desc'}`
}

export function useListPage<T = Record<string, unknown>, S extends Record<string, unknown> = Record<string, unknown>>(
  options: UseListPageOptions<T, S>,
) {
  const {
    fetchApi,
    initialSearchForm = {} as S,
    fieldMapping = {},
    defaultSort = 'id:desc',
    transformData = null,
    onLoadSuccess = null,
    buildParams = null,
    onSearch = null,
    onReset = null,
    selectionIdKey = 'id',
    autoLoad = true,
  } = options

  const [searchForm, setSearchForm] = useState<S>({ ...initialSearchForm })
  const [selectedRows, setSelectedRows] = useState<T[]>([])
  const [orderBy, setOrderBy] = useState(defaultSort)
  const [pagination, setPagination] = useState<PaginationState>({
    page: 1,
    pageSize: 10,
    total: 0,
  })
  const [tableData, setTableData] = useState<T[]>([])
  const [loading, setLoading] = useState(false)

  const searchFormRef = useRef(searchForm)
  const orderByRef = useRef(orderBy)
  searchFormRef.current = searchForm
  orderByRef.current = orderBy

  const fetchRows = useCallback(
    async (params: Record<string, unknown> = {}) => {
      setLoading(true)
      try {
        const res = await fetchApi(params)
        const rawList = (res.data?.list ?? res.data?.data ?? []) as unknown[]
        let rows: T[] = []

        if (Array.isArray(rawList)) {
          rows = rawList.map((item) => {
            const row = item as Record<string, unknown>
            if (transformData) {
              return transformData(row)
            }
            return row as T
          })
        }

        setTableData(rows)
        setPagination((prev) => ({
          ...prev,
          page: Number(params.page ?? prev.page),
          pageSize: Number(params.page_size ?? prev.pageSize),
          total: Number(res.data?.total ?? 0),
        }))

        onLoadSuccess?.(rows, res)
      } catch (error) {
        logger.error('loadData failed:', error)
        throw error
      } finally {
        setLoading(false)
      }
    },
    [fetchApi, transformData, onLoadSuccess],
  )

  const loadData = useCallback(
    async (
      pageParams: { currentPage?: number; pageSize?: number } | null = null,
      orderOverride?: string,
    ) => {
      const page = pageParams?.currentPage ?? pagination.page
      const pageSize = pageParams?.pageSize ?? pagination.pageSize

      if (pageParams) {
        setPagination((prev) => ({
          ...prev,
          page,
          pageSize,
        }))
      }

      const baseParams = {
        page,
        page_size: pageSize,
        order_by: orderOverride ?? orderByRef.current,
      }

      const params =
        buildParams && typeof buildParams === 'function'
          ? buildParams(searchFormRef.current, baseParams)
          : buildSearchParams(searchFormRef.current, baseParams)

      await fetchRows(params)
    },
    [pagination.page, pagination.pageSize, buildParams, fetchRows],
  )

  const loadDataRef = useRef(loadData)
  loadDataRef.current = loadData

  const refresh = useCallback(async () => {
    await loadData()
  }, [loadData])

  const handleSearch = useCallback(() => {
    onSearch?.()
    setPagination((prev) => ({ ...prev, page: 1 }))
    void loadData({ currentPage: 1 })
  }, [onSearch, loadData])

  const handleReset = useCallback(() => {
    onReset?.()
    setSearchForm({ ...initialSearchForm })
    setOrderBy(defaultSort)
    orderByRef.current = defaultSort
    setPagination((prev) => ({ ...prev, page: 1 }))
    setTimeout(() => {
      void loadDataRef.current({ currentPage: 1 }, defaultSort)
    }, 0)
  }, [onReset, initialSearchForm, defaultSort])

  /** Compatible with SearchForm.onChange without page-level casts. */
  const onSearchFormChange = useCallback((values: Record<string, unknown>) => {
    setSearchForm(values as S)
  }, [])

  const handleSortChange = useCallback(
    (field?: string, order?: 'ascend' | 'descend' | null) => {
      const next = toOrderBy(field, order, fieldMapping, defaultSort)
      setOrderBy(next)
      orderByRef.current = next
      setPagination((prev) => ({ ...prev, page: 1 }))
      void loadData({ currentPage: 1 }, next)
    },
    [fieldMapping, defaultSort, loadData],
  )

  const selectedIds = selectedRows.map((row) => {
    const record = row as Record<string, unknown>
    return record[selectionIdKey] ?? record.ID
  })

  useEffect(() => {
    if (autoLoad) {
      void loadDataRef.current()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount-only
  }, [])

  return {
    pagination,
    setPagination,
    tableData,
    loading,
    searchForm,
    setSearchForm,
    onSearchFormChange,
    selectedRows,
    setSelectedRows,
    selectedIds,
    orderBy,
    loadData,
    refresh,
    handleSearch,
    handleReset,
    handleSortChange,
  }
}
