import type { TFunction } from 'i18next'
import type { SearchField } from '@/components/SearchForm'
import { buildSearchParams } from '@/utils/buildSearchParams'
import { entityField } from '@/utils/normalize'

export const articleInitialSearchForm: Record<string, unknown> = {

  admin_id: '',
  title: '',
  content: '',
  status: '',
  created_at: [],
  updated_at: [],
}

export type ArticleSearchForm = typeof articleInitialSearchForm

export interface ArticleRow {
  id: number | string

  admin_id?: unknown
  admin?: unknown
  title?: unknown
  content?: unknown
  status?: unknown
  created_at?: unknown
  updated_at?: unknown
}

export function buildArticleListParams(
  form: ArticleSearchForm,
  baseParams: Record<string, unknown>,
) {
  return buildSearchParams(form, baseParams)
}

export function createArticleSearchFields(t: TFunction): SearchField[] {
  return [

    {
      name: 'admin_id',
      label: t('admin_id', { defaultValue: '管理员ID' }),
      
    },
    {
      name: 'title',
      label: t('title', { defaultValue: '标题' }),
      
    },
    {
      name: 'content',
      label: t('content', { defaultValue: '内容' }),
      
    },
    {
      name: 'status',
      label: t('common.status'),
      
      type: 'select',
      options: [
        { label: t('common.enabled'), value: 1 },
        { label: t('common.disabled'), value: 0 },
      ],
      
    },
    {
      name: 'created_at',
      label: t('common.created_at'),
      
    },
    {
      name: 'updated_at',
      label: t('common.updated_at'),
      
    },
  ]
}

export function transformArticleRow(row: Record<string, unknown>): ArticleRow {
  return {
    id: entityField(row, 'id', '')!,

    admin_id: entityField(row, 'admin_id', 0),
    admin: entityField(row, 'admin', null),
    title: entityField(row, 'title', ''),
    content: entityField(row, 'content', ''),
    status: entityField(row, 'status', 0),
    created_at: entityField(row, 'created_at', ''),
    updated_at: entityField(row, 'updated_at', ''),
  }
}

export function getadminDisplayName(value: unknown) {
  if (!value || typeof value !== 'object') return '-'
  const record = value as Record<string, unknown>
  return String(record['nickname'] ?? record['admin'] ?? '-')
}
