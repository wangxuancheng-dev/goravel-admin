import { createCRUDApi, extendApi } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'

import request from '@/utils/request'

const baseArticleApi = createCRUDApi('articles')

const articleApi = extendApi(baseArticleApi, {

  export: (params?: Record<string, unknown>) =>
    request({
      url: '/articles/export',
      method: 'post',
      data: params,
    }),

  import: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request({
      url: '/articles/import',
      method: 'post',
      data: formData,
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
  },

})

export async function getArticleList(params?: Record<string, unknown>) {
  return normalizeListResponse(await articleApi.list(params))
}

export const getArticleDetail = articleApi.detail

export const createArticle = articleApi.create

export const updateArticle = articleApi.update

export const deleteArticle = articleApi.delete

export const exportArticle = articleApi.export

export const importArticle = articleApi.import
