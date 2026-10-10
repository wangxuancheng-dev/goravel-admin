import type { TFunction } from 'i18next'
import type { SearchField } from '@/components/SearchForm'
import { buildSearchParams } from '@/utils/buildSearchParams'
import { entityField } from '@/utils/normalize'

export const quoteInitialSearchForm: Record<string, unknown> = {

  quote_no: '',
  customer_name: '',
  status: '',
  remark: '',
  created_at: [],
  updated_at: [],
}

export type QuoteSearchForm = typeof quoteInitialSearchForm

export interface QuoteRow {
  id: number | string

  quote_no?: unknown
  customer_name?: unknown
  status?: unknown
  remark?: unknown
  created_at?: unknown
  updated_at?: unknown
}

export function buildQuoteListParams(
  form: QuoteSearchForm,
  baseParams: Record<string, unknown>,
) {
  return buildSearchParams(form, baseParams)
}

export function createQuoteSearchFields(t: TFunction): SearchField[] {
  return [

    {
      name: 'quote_no',
      label: t('quote_no', { defaultValue: 'quote number' }),
      
    },
    {
      name: 'customer_name',
      label: t('customer_name', { defaultValue: 'customer name' }),
      
    },
    {
      name: 'status',
      label: t('common.status'),
      
    },
    {
      name: 'remark',
      label: t('remark', { defaultValue: 'remark' }),
      
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

export function transformQuoteRow(row: Record<string, unknown>): QuoteRow {
  return {
    id: entityField(row, 'id', '')!,

    quote_no: entityField(row, 'quote_no', ''),
    customer_name: entityField(row, 'customer_name', ''),
    status: entityField(row, 'status', 0),
    remark: entityField(row, 'remark', ''),
    created_at: entityField(row, 'created_at', ''),
    updated_at: entityField(row, 'updated_at', ''),
  }
}
