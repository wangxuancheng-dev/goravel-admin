import { createCRUDApi<<if or .HasExport .HasImport>>, extendApi<<end>> } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'
<<if or .HasExport .HasImport>>
import request from '@/utils/request'
<<end>>

const base<<.ModelName>>Api = createCRUDApi('<<.ModuleName>>s')

<<if or .HasExport .HasImport>>
const <<.ModuleName>>Api = extendApi(base<<.ModelName>>Api, {
<<if .HasExport>>
  export: (params?: Record<string, unknown>) =>
    request({
      url: '/<<.ModuleName>>s/export',
      method: 'post',
      data: params,
    })<<if .HasImport>>,<<end>>
<<end>>
<<if .HasImport>>
  import: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request({
      url: '/<<.ModuleName>>s/import',
      method: 'post',
      data: formData,
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
  },
<<end>>
})
<<else>>
const <<.ModuleName>>Api = base<<.ModelName>>Api
<<end>>

export async function get<<.ModelName>>List(params?: Record<string, unknown>) {
  return normalizeListResponse(await <<.ModuleName>>Api.list(params))
}

export const get<<.ModelName>>Detail = <<.ModuleName>>Api.detail

<<if .HasCreate>>
export const create<<.ModelName>> = <<.ModuleName>>Api.create
<<end>>

<<if .HasEdit>>
export const update<<.ModelName>> = <<.ModuleName>>Api.update
<<end>>

<<if .HasDelete>>
export const delete<<.ModelName>> = <<.ModuleName>>Api.delete
<<end>>

<<if .HasExport>>
export const export<<.ModelName>> = <<.ModuleName>>Api.export
<<end>>

<<if .HasImport>>
export const import<<.ModelName>> = <<.ModuleName>>Api.import
<<end>>
