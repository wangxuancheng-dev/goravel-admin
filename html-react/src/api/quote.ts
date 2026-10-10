import { createCRUDApi, extendApi } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'

import request from '@/utils/request'

const baseQuoteApi = createCRUDApi('quotes')

const quoteApi = extendApi(baseQuoteApi, {

  export: (params?: Record<string, unknown>) =>
    request({
      url: '/quotes/export',
      method: 'post',
      data: params,
    })

})

export async function getQuoteList(params?: Record<string, unknown>) {
  return normalizeListResponse(await quoteApi.list(params))
}

export const getQuoteDetail = quoteApi.detail

export const createQuote = quoteApi.create

export const updateQuote = quoteApi.update

export const deleteQuote = quoteApi.delete

export const exportQuote = quoteApi.export
