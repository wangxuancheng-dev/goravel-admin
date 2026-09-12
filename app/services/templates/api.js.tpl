import request from '../utils/request'
import { createCRUDApi, extendApi } from '../utils/apiFactory'
import { normalizeListResponse } from '../utils/normalize'

const base<<.ModelName>>Api = createCRUDApi('<<.ModuleName>>s')

<<if or .HasExport .HasImport>>
const <<.ModuleName>>Api = extendApi(base<<.ModelName>>Api, {
<<if .HasExport>>
  export: (params) => {
    return request({
      url: '/<<.ModuleName>>s/export',
      method: 'post',
      data: params
    })
  }<<if .HasImport>>,<<end>>
<<end>>
<<if .HasImport>>
  import: (file) => {
    const formData = new FormData()
    formData.append('file', file)
    return request({
      url: '/<<.ModuleName>>s/import',
      method: 'post',
      data: formData,
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
  }
<<end>>
})
<<else>>
const <<.ModuleName>>Api = base<<.ModelName>>Api
<<end>>

export async function get<<.ModelName>>List(params) {
  const res = await <<.ModuleName>>Api.list(params)
  return normalizeListResponse(res)
}

export function get<<.ModelName>>Detail(id) {
  return <<.ModuleName>>Api.detail(id)
}

<<if .HasCreate>>
export function create<<.ModelName>>(data) {
  return <<.ModuleName>>Api.create(data)
}
<<end>>

<<if .HasEdit>>
export function update<<.ModelName>>(id, data) {
  return <<.ModuleName>>Api.update(id, data)
}
<<end>>

<<if .HasDelete>>
export function delete<<.ModelName>>(id) {
  return <<.ModuleName>>Api.delete(id)
}
<<end>>

<<if .HasExport>>
export function export<<.ModelName>>(params) {
  return <<.ModuleName>>Api.export(params)
}
<<end>>

<<if .HasImport>>
export function import<<.ModelName>>(file) {
  return <<.ModuleName>>Api.import(file)
}
<<end>>
