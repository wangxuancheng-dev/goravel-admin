import request from '../utils/request'
import { createCRUDApi, extendApi } from '../utils/apiFactory'
import { normalizeListResponse } from '../utils/normalize'

const baseOrderApi = createCRUDApi('orders')

const orderApi = extendApi(baseOrderApi, {
  export: (params) => {
    return request({
      url: '/orders/export',
      method: 'post',
      data: params
    })
  },
  exportStatus: (exportId) => {
    return request({
      url: `/orders/export/status/${exportId}`,
      method: 'get'
    })
  },
  import: (file) => {
    const formData = new FormData()
    formData.append('file', file)
    return request({
      url: '/orders/import',
      method: 'post',
      data: formData,
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
  }
})

export async function getOrderList(params) {
  const res = await orderApi.list(params)
  return normalizeListResponse(res)
}

export function getOrderDetail(id, params = {}) {
  if (params.order_no) {
    return request({
      url: `/orders/${id || 0}`,
      method: 'get',
      params: { order_no: params.order_no }
    })
  }
  return orderApi.detail(id)
}

export const {
  create: createOrder,
  update: updateOrder
} = orderApi

export function deleteOrder(id, params = {}) {
  return request({
    url: `/orders/${id}`,
    method: 'delete',
    params: Object.keys(params).length ? params : undefined
  })
}

export const exportOrder = orderApi.export
export const getExportStatus = orderApi.exportStatus
export const importOrder = orderApi.import
