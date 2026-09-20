import { useMemo, useState } from 'react'
import { App, Button, Checkbox, DatePicker, Form, Input, InputNumber, Modal, Radio, Select, Space, Switch, Table, Tag } from 'antd'
import dayjs from 'dayjs'
import { useTranslation } from 'react-i18next'
import PageContainer from '@/components/PageContainer'
import PermissionButton from '@/components/PermissionButton'
import SearchForm from '@/components/SearchForm'
import { useListPage } from '@/hooks/useListPage'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import {
  checkDemoActivityActive,
  createDemoActivity,
  deleteDemoActivity,
  getDemoActivityList,
  syncDemoActivities,
  updateDemoActivity,
} from '@/api/demoActivity'

interface DemoActivityRow {
  id: number | string
  title?: string
  schedule_type?: string
  start_at?: string | null
  end_at?: string | null
  daily_start?: string
  daily_end?: string
  weekdays?: string
  month_days?: string
  month_day_start?: number
  month_day_end?: number
  year_start?: string
  year_end?: string
  timezone?: string
  status?: number
  enabled?: boolean
  created_at?: string
}

export default function DemoActivityList() {
  const { t } = useTranslation()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [form] = Form.useForm()
  const [open, setOpen] = useState(false)
  const [editId, setEditId] = useState<string | number | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const scheduleType = Form.useWatch('schedule_type', form)

  const {
    tableData,
    loading,
    pagination,
    searchForm,
    onSearchFormChange,
    loadData,
    handleSearch,
    handleReset,
    handleSortChange,
    refresh,
  } = useListPage<DemoActivityRow>({
    fetchApi: getDemoActivityList,
    initialSearchForm: { title: '' },
    normalizeRows: true,
  })

  const statusLabel = (status?: number) => {
    if (status === 1) return t('demo_activity.status_running')
    if (status === 2) return t('demo_activity.status_ended')
    return t('demo_activity.status_pending')
  }

  const statusColor = (status?: number) => {
    if (status === 1) return 'success'
    if (status === 2) return 'default'
    return 'processing'
  }

  const openCreate = () => {
    setEditId(null)
    form.setFieldsValue({
      title: '',
      schedule_type: 'once',
      timezone: 'Asia/Shanghai',
      enabled: true,
      start_at: dayjs().add(1, 'minute'),
      end_at: dayjs().add(1, 'hour'),
      daily_start: '09:00',
      daily_end: '18:00',
      weekdays: ['1', '2', '3', '4', '5'],
      month_days: '1,15',
      month_mode: 'days',
      month_day_start: 1,
      month_day_end: 5,
      year_start: '03-01',
      year_end: '03-15',
    })
    setOpen(true)
  }

  const openEdit = (row: DemoActivityRow) => {
    setEditId(row.id)
    form.setFieldsValue({
      title: row.title,
      schedule_type: row.schedule_type || 'once',
      timezone: row.timezone || 'Asia/Shanghai',
      enabled: row.enabled !== false,
      start_at: row.start_at ? dayjs(row.start_at) : null,
      end_at: row.end_at ? dayjs(row.end_at) : null,
      daily_start: row.daily_start || '09:00',
      daily_end: row.daily_end || '18:00',
      weekdays: (row.weekdays || '').split(',').filter(Boolean),
      month_days: row.month_days || '',
      month_mode: row.month_days ? 'days' : 'range',
      month_day_start: row.month_day_start || 1,
      month_day_end: row.month_day_end || 5,
      year_start: row.year_start || '03-01',
      year_end: row.year_end || '03-15',
    })
    setOpen(true)
  }

  const typeLabel = (v?: string) => {
    switch (v) {
      case 'daily':
        return t('demo_activity.type_daily')
      case 'weekly':
        return t('demo_activity.type_weekly')
      case 'monthly':
        return t('demo_activity.type_monthly')
      case 'yearly':
        return t('demo_activity.type_yearly')
      default:
        return t('demo_activity.type_once')
    }
  }

  const buildPayload = (values: Record<string, unknown>) => {
    const type = String(values.schedule_type || 'once')
    const payload: Record<string, unknown> = {
      title: values.title,
      schedule_type: type,
      timezone: values.timezone || 'Asia/Shanghai',
      enabled: !!values.enabled,
    }
    if (type === 'once') {
      payload.start_at = values.start_at ? (values.start_at as dayjs.Dayjs).toISOString() : ''
      payload.end_at = values.end_at ? (values.end_at as dayjs.Dayjs).toISOString() : ''
    } else if (type === 'daily') {
      payload.daily_start = values.daily_start
      payload.daily_end = values.daily_end
    } else if (type === 'weekly') {
      const days = Array.isArray(values.weekdays) ? values.weekdays : []
      payload.weekdays = days.join(',')
      payload.daily_start = values.daily_start
      payload.daily_end = values.daily_end
    } else if (type === 'monthly') {
      if (values.month_mode === 'range') {
        payload.month_day_start = Number(values.month_day_start || 0)
        payload.month_day_end = Number(values.month_day_end || 0)
        payload.month_days = ''
        payload.daily_start = values.daily_start || ''
        payload.daily_end = values.daily_end || ''
      } else {
        payload.month_days = values.month_days
        payload.month_day_start = 0
        payload.month_day_end = 0
        payload.daily_start = values.daily_start
        payload.daily_end = values.daily_end
      }
    } else if (type === 'yearly') {
      payload.year_start = values.year_start
      payload.year_end = values.year_end
      payload.daily_start = values.daily_start || ''
      payload.daily_end = values.daily_end || ''
    }
    return payload
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const payload = buildPayload(values)
      if (editId == null) {
        await createDemoActivity(payload)
        message.success(t('common.create_success'))
      } else {
        await updateDemoActivity(editId, payload)
        message.success(t('common.update_success'))
      }
      setOpen(false)
      refresh()
    } catch (err) {
      showError(err)
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = (row: DemoActivityRow) => {
    modal.confirm({
      title: t('common.confirm_delete'),
      onOk: async () => {
        try {
          await deleteDemoActivity(row.id)
          message.success(t('common.delete_success'))
          refresh()
        } catch (err) {
          showError(err)
        }
      },
    })
  }

  const handleCheck = async (row: DemoActivityRow) => {
    try {
      const res = (await checkDemoActivityActive(row.id)) as {
        data?: { is_active_live?: boolean; stored_status?: number; desired_status?: number }
      }
      const data = res.data || {}
      modal.info({
        title: t('demo_activity.active_check'),
        content: (
          <div>
            <div>
              {t('demo_activity.stored_status')}: {statusLabel(data.stored_status)}
            </div>
            <div>
              {t('demo_activity.desired_status')}: {statusLabel(data.desired_status)}
            </div>
            <div>
              {t('demo_activity.is_active_live')}:{' '}
              {data.is_active_live ? t('common.yes') : t('common.no')}
            </div>
          </div>
        ),
      })
    } catch (err) {
      showError(err)
    }
  }

  const handleSync = async () => {
    setSyncing(true)
    try {
      const res = (await syncDemoActivities()) as { data?: { updated?: number } }
      message.success(t('demo_activity.sync_success', { count: res.data?.updated ?? 0 }))
      refresh()
    } catch (err) {
      showError(err)
    } finally {
      setSyncing(false)
    }
  }

  const columns = useMemo(
    () => [
      { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
      { title: t('demo_activity.title'), dataIndex: 'title', ellipsis: true },
      {
        title: t('demo_activity.schedule_type'),
        dataIndex: 'schedule_type',
        width: 100,
        render: (v: string) => typeLabel(v),
      },
      {
        title: t('demo_activity.window'),
        key: 'window',
        render: (_: unknown, row: DemoActivityRow) => {
          switch (row.schedule_type) {
            case 'daily':
              return `${row.daily_start || '-'} ~ ${row.daily_end || '-'}`
            case 'weekly':
              return `${row.weekdays || '-'} @ ${row.daily_start || '-'}-${row.daily_end || '-'}`
            case 'monthly':
              if (row.month_days)
                return `${row.month_days} @ ${row.daily_start || '-'}-${row.daily_end || '-'}`
              return `D${row.month_day_start || '?'}~D${row.month_day_end || '?'}`
            case 'yearly':
              return `${row.year_start || '-'} ~ ${row.year_end || '-'}`
            default:
              return `${row.start_at || '-'} ~ ${row.end_at || '-'}`
          }
        },
      },
      {
        title: t('demo_activity.status'),
        dataIndex: 'status',
        width: 110,
        render: (v: number) => <Tag color={statusColor(v)}>{statusLabel(v)}</Tag>,
      },
      {
        title: t('common.enabled'),
        dataIndex: 'enabled',
        width: 90,
        render: (v: boolean) =>
          v !== false ? (
            <Tag color="success">{t('common.enabled')}</Tag>
          ) : (
            <Tag>{t('common.disabled')}</Tag>
          ),
      },
      {
        title: t('table.operation'),
        key: 'ops',
        width: 280,
        render: (_: unknown, row: DemoActivityRow) => (
          <Space wrap>
            <PermissionButton
              permission="demo_activity.update"
              type="link"
              size="small"
              disabled={getButtonState('demo_activity.update').disabled}
              onClick={() => openEdit(row)}
            >
              {t('common.edit')}
            </PermissionButton>
            <PermissionButton
              permission="demo_activity.active_check"
              type="link"
              size="small"
              disabled={getButtonState('demo_activity.active_check').disabled}
              onClick={() => handleCheck(row)}
            >
              {t('demo_activity.active_check')}
            </PermissionButton>
            <PermissionButton
              permission="demo_activity.destroy"
              type="link"
              size="small"
              danger
              disabled={getButtonState('demo_activity.destroy').disabled}
              onClick={() => handleDelete(row)}
            >
              {t('common.delete')}
            </PermissionButton>
          </Space>
        ),
      },
    ],
    [t, getButtonState]
  )

  return (
    <PageContainer title={t('menu.demo_activity')}>
      <div style={{ marginBottom: 12 }}>
        <SearchForm
          fields={[{ name: 'title', label: t('demo_activity.title') }]}
          values={searchForm}
          onChange={onSearchFormChange}
          onSearch={handleSearch}
          onReset={handleReset}
        />
      </div>
      <Space style={{ marginBottom: 12 }}>
        <PermissionButton
          permission="demo_activity.store"
          type="primary"
          disabled={getButtonState('demo_activity.store').disabled}
          onClick={openCreate}
        >
          {t('demo_activity.add')}
        </PermissionButton>
        <PermissionButton
          permission="demo_activity.sync"
          disabled={getButtonState('demo_activity.sync').disabled}
          loading={syncing}
          onClick={handleSync}
        >
          {t('demo_activity.sync_now')}
        </PermissionButton>
        <Button onClick={() => loadData()}>{t('common.refresh')}</Button>
      </Space>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={tableData}
        columns={columns}
        pagination={pagination}
        onChange={(pag, _f, sorter) => handlePaginatedTableChange(pag, sorter, handleSortChange, loadData)}
      />

      <Modal
        open={open}
        title={editId == null ? t('demo_activity.add') : t('demo_activity.edit')}
        onCancel={() => setOpen(false)}
        onOk={handleSubmit}
        confirmLoading={submitting}
        destroyOnClose
        width={640}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="title" label={t('demo_activity.title')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="schedule_type" label={t('demo_activity.schedule_type')} rules={[{ required: true }]}>
            <Radio.Group>
              <Radio.Button value="once">{t('demo_activity.type_once')}</Radio.Button>
              <Radio.Button value="daily">{t('demo_activity.type_daily')}</Radio.Button>
              <Radio.Button value="weekly">{t('demo_activity.type_weekly')}</Radio.Button>
              <Radio.Button value="monthly">{t('demo_activity.type_monthly')}</Radio.Button>
              <Radio.Button value="yearly">{t('demo_activity.type_yearly')}</Radio.Button>
            </Radio.Group>
          </Form.Item>
          <Form.Item name="timezone" label={t('demo_activity.timezone')}>
            <Select
              options={[
                { value: 'Asia/Shanghai', label: 'Asia/Shanghai' },
                { value: 'UTC', label: 'UTC' },
              ]}
            />
          </Form.Item>
          {scheduleType === 'once' && (
            <>
              <Form.Item name="start_at" label={t('demo_activity.start_at')} rules={[{ required: true }]}>
                <DatePicker showTime style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item name="end_at" label={t('demo_activity.end_at')} rules={[{ required: true }]}>
                <DatePicker showTime style={{ width: '100%' }} />
              </Form.Item>
            </>
          )}
          {scheduleType === 'daily' && (
            <>
              <Form.Item name="daily_start" label={t('demo_activity.daily_start')} rules={[{ required: true }]}>
                <Input placeholder="09:00" />
              </Form.Item>
              <Form.Item name="daily_end" label={t('demo_activity.daily_end')} rules={[{ required: true }]}>
                <Input placeholder="18:00" />
              </Form.Item>
            </>
          )}
          {scheduleType === 'weekly' && (
            <>
              <Form.Item name="weekdays" label={t('demo_activity.weekdays')} rules={[{ required: true }]}>
                <Checkbox.Group
                  options={[
                    { label: t('demo_activity.weekday_1'), value: '1' },
                    { label: t('demo_activity.weekday_2'), value: '2' },
                    { label: t('demo_activity.weekday_3'), value: '3' },
                    { label: t('demo_activity.weekday_4'), value: '4' },
                    { label: t('demo_activity.weekday_5'), value: '5' },
                    { label: t('demo_activity.weekday_6'), value: '6' },
                    { label: t('demo_activity.weekday_7'), value: '7' },
                  ]}
                />
              </Form.Item>
              <Form.Item name="daily_start" label={t('demo_activity.daily_start')} rules={[{ required: true }]}>
                <Input placeholder="09:00" />
              </Form.Item>
              <Form.Item name="daily_end" label={t('demo_activity.daily_end')} rules={[{ required: true }]}>
                <Input placeholder="18:00" />
              </Form.Item>
            </>
          )}
          {scheduleType === 'monthly' && (
            <>
              <Form.Item name="month_mode" label={t('demo_activity.month_mode')} initialValue="days">
                <Radio.Group>
                  <Radio.Button value="days">{t('demo_activity.month_mode_days')}</Radio.Button>
                  <Radio.Button value="range">{t('demo_activity.month_mode_range')}</Radio.Button>
                </Radio.Group>
              </Form.Item>
              <Form.Item noStyle shouldUpdate={(prev, cur) => prev.month_mode !== cur.month_mode}>
                {() =>
                  form.getFieldValue('month_mode') === 'range' ? (
                    <>
                      <Form.Item name="month_day_start" label={t('demo_activity.month_day_start')} rules={[{ required: true }]}>
                        <InputNumber min={1} max={31} style={{ width: '100%' }} />
                      </Form.Item>
                      <Form.Item name="month_day_end" label={t('demo_activity.month_day_end')} rules={[{ required: true }]}>
                        <InputNumber min={1} max={31} style={{ width: '100%' }} />
                      </Form.Item>
                    </>
                  ) : (
                    <Form.Item name="month_days" label={t('demo_activity.month_days')} rules={[{ required: true }]}>
                      <Input placeholder="1,15,28" />
                    </Form.Item>
                  )
                }
              </Form.Item>
              <Form.Item name="daily_start" label={t('demo_activity.daily_start')}>
                <Input placeholder="09:00" />
              </Form.Item>
              <Form.Item name="daily_end" label={t('demo_activity.daily_end')}>
                <Input placeholder="18:00" />
              </Form.Item>
            </>
          )}
          {scheduleType === 'yearly' && (
            <>
              <Form.Item name="year_start" label={t('demo_activity.year_start')} rules={[{ required: true }]}>
                <Input placeholder="03-01" />
              </Form.Item>
              <Form.Item name="year_end" label={t('demo_activity.year_end')} rules={[{ required: true }]}>
                <Input placeholder="03-15" />
              </Form.Item>
              <Form.Item name="daily_start" label={t('demo_activity.daily_start')}>
                <Input placeholder="09:00" />
              </Form.Item>
              <Form.Item name="daily_end" label={t('demo_activity.daily_end')}>
                <Input placeholder="18:00" />
              </Form.Item>
            </>
          )}
          <Form.Item name="enabled" label={t('common.enabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
