import { useCallback, useEffect, useMemo, useState } from 'react'
import { Tag } from 'antd'
import { useTranslation } from 'react-i18next'
import SimpleCrudPage from '@/components/SimpleCrudPage'
import StatusTag from '@/components/StatusTag'
import {
  createDictionary,
  deleteDictionary,
  getDictionaryList,
  getDictionaryTypes,
  updateDictionary,
} from '@/api/dictionary'

interface DictionaryRow {
  id: number | string
  type?: string
  label?: string
  value?: string
  translation_key?: string
  sort?: number
  status?: number
  is_system?: number
  created_at?: string
  name?: string
}

function isSystemDictionary(row?: DictionaryRow | null) {
  return Number(row?.is_system ?? 0) === 1
}

export default function DictionaryList() {
  const { t } = useTranslation()
  const [typeOptions, setTypeOptions] = useState<Array<{ label: string; value: string }>>([])

  const loadTypeOptions = useCallback(async () => {
    try {
      const res = await getDictionaryTypes()
      const types = ((res as { data?: { types?: string[] } })?.data?.types || []) as string[]
      setTypeOptions(types.map((item) => ({ label: item, value: item })))
    } catch {
      setTypeOptions([])
    }
  }, [])

  useEffect(() => {
    void loadTypeOptions()
  }, [loadTypeOptions])

  const searchFields = useMemo(
    () => [
      {
        name: 'type',
        label: t('dictionary.type'),
        type: 'select' as const,
        options: typeOptions,
      },
    ],
    [t, typeOptions],
  )

  const formFields = useMemo(
    () => [
      {
        name: 'type',
        label: t('dictionary.type'),
        required: true,
        type: 'select' as const,
        options: typeOptions,
        allowCreate: true,
        lockedWhenProtected: true,
      },
      { name: 'label', label: t('dictionary.label'), required: true },
      { name: 'value', label: t('dictionary.value'), required: true, lockedWhenProtected: true },
      { name: 'translation_key', label: t('dictionary.translation_key') },
      { name: 'sort', label: t('common.sort'), type: 'number' as const },
      { name: 'status', label: t('common.status'), type: 'status' as const, lockedWhenProtected: true },
    ],
    [t, typeOptions],
  )

  return (
    <SimpleCrudPage<DictionaryRow>
      title={t('menu.dictionary')}
      permissions={{
        store: 'dictionary.store',
        update: 'dictionary.update',
        destroy: 'dictionary.destroy',
      }}
      fetchApi={getDictionaryList}
      createApi={createDictionary}
      updateApi={updateDictionary}
      deleteApi={deleteDictionary}
      createTitle={t('dictionary.add_dictionary')}
      editTitle={t('dictionary.edit_dictionary')}
      initialSearchForm={{ type: '' }}
      searchFields={searchFields}
      formFields={formFields}
      isProtected={isSystemDictionary}
      onMutated={loadTypeOptions}
      transformRow={(row) => ({
        id: row.id as string | number,
        type: String(row.type ?? ''),
        label: String(row.label ?? ''),
        value: String(row.value ?? ''),
        translation_key: String(row.translation_key ?? ''),
        sort: Number(row.sort ?? 0),
        status: Number(row.status ?? 0),
        is_system: Number(row.is_system ?? 0),
        created_at: String(row.created_at ?? ''),
        name: String(row.label ?? row.type ?? ''),
      })}
      columns={[
        { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
        { title: t('dictionary.type'), dataIndex: 'type' },
        { title: t('dictionary.label'), dataIndex: 'label' },
        { title: t('dictionary.value'), dataIndex: 'value' },
        { title: t('dictionary.translation_key'), dataIndex: 'translation_key', ellipsis: true },
        {
          title: t('dictionary.is_system'),
          dataIndex: 'is_system',
          width: 100,
          render: (value: number) => (
            <Tag color={Number(value) === 1 ? 'gold' : 'default'}>
              {Number(value) === 1 ? t('dictionary.system_yes') : t('dictionary.system_no')}
            </Tag>
          ),
        },
        { title: t('common.sort'), dataIndex: 'sort', width: 80, sorter: true },
        {
          title: t('common.status'),
          dataIndex: 'status',
          width: 100,
          render: (status: number) => <StatusTag status={status} />,
        },
        { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
      ]}
    />
  )
}
