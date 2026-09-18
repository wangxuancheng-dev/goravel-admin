/** Backend success / error envelope (must match Goravel admin API). */
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data?: T
  trace_id?: string
  error_code?: string
  errors?: Record<string, string[]>
}

export interface PaginatedData<T = unknown> {
  list?: T[]
  data?: T[]
  total: number
  page?: number
  page_size?: number
  /** Effective write disk for current tenant/request (attachments Index). */
  write_disk?: string
  /** Whether large-file chunk upload is available (local/public only). */
  chunk_upload_supported?: boolean
}

/**
 * List API accepted by useListPage / useTableData.
 * Row type is refined by transformData (or cast when omitted).
 */
export type ListFetchFn = (
  params?: Record<string, unknown>,
) => Promise<ApiResponse<PaginatedData>>

export interface PaginationState {
  page: number
  pageSize: number
  total: number
}

/** Augmented axios / business error used by request interceptors. */
export interface ApiError extends Error {
  code?: number | string
  errorCode?: string
  data?: unknown
  response?: unknown
  translatedMessage?: string
  /** When true, global interceptor already showed a toast — pages should not duplicate. */
  __handled?: boolean
  config?: {
    url?: string
    skipErrorMessage?: boolean
    silent?: boolean
  }
}

export const ERROR_CODES = {
  GOOGLE_CODE_REQUIRED: 'google_code_required',
  GOOGLE_CODE_INVALID: 'google_code_invalid',
  ACCOUNT_DISABLED: 'account_disabled',
  UNAUTHORIZED: 'unauthorized',
  FORBIDDEN: 'forbidden',
  CAPTCHA_REQUIRED: 'captcha_required',
  CAPTCHA_INVALID: 'captcha_invalid',
  CAPTCHA_EXPIRED: 'captcha_expired',
  TOO_MANY_REQUESTS: 'too_many_requests',
  NETWORK_ERROR: 'network_error',
  TIMEOUT: 'timeout',
} as const

export type ErrorCode = (typeof ERROR_CODES)[keyof typeof ERROR_CODES]
