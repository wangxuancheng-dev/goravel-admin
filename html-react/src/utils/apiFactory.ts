import request from './request'
import type { ApiResponse, PaginatedData } from '@/types'

export interface CrudApi<T = unknown> {
  list: (params?: Record<string, unknown>) => Promise<ApiResponse<PaginatedData<T>>>
  detail: (id: string | number) => Promise<ApiResponse<T>>
  create: (data: unknown) => Promise<ApiResponse<T>>
  update: (id: string | number, data: unknown) => Promise<ApiResponse<T>>
  delete: (id: string | number, data?: Record<string, unknown>) => Promise<ApiResponse<unknown>>
  batchDelete: (ids: Array<string | number>) => Promise<ApiResponse<unknown>>
}

export function createCRUDApi<T = unknown>(resource: string): CrudApi<T> {
  return {
    list: (params) =>
      request<PaginatedData<T>>({
        url: `/${resource}`,
        method: 'get',
        params,
      }),
    detail: (id) =>
      request<T>({
        url: `/${resource}/${id}`,
        method: 'get',
      }),
    create: (data) =>
      request<T>({
        url: `/${resource}`,
        method: 'post',
        data,
      }),
    update: (id, data) =>
      request<T>({
        url: `/${resource}/${id}`,
        method: 'put',
        data,
      }),
    delete: (id, data) =>
      request({
        url: `/${resource}/${id}`,
        method: 'delete',
        ...(data ? { data } : {}),
      }),
    batchDelete: (ids) =>
      request({
        url: `/${resource}/batch`,
        method: 'delete',
        data: { ids },
      }),
  }
}

export function extendApi<TBase extends object, TExtra extends object>(
  baseApi: TBase,
  customMethods: TExtra,
): TBase & TExtra {
  return {
    ...baseApi,
    ...customMethods,
  }
}
