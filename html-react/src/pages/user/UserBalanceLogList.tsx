import { useEffect, useMemo, useState } from 'react'
import { Button, Card, Col, Row, Space, Statistic, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { ArrowLeftOutlined, SettingOutlined } from '@ant-design/icons'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { getUserBalanceLogList, getUserBalanceStatistics } from '@/api/userBalanceLog'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { useColumnSetting } from '@/hooks/useColumnSetting'
import PageContainer from '@/components/PageContainer'
import ColumnSettingDialog from '@/components/ColumnSettingDialog'
import SearchForm from '@/components/SearchForm'
import { entityField } from '@/utils/normalize'

interface LogRow {
  id: number | string
  type?: string
  amount?: number | string
  balance?: number | string
  source?: string
  description?: string
  created_at?: string
}

function formatMoney(amount: number | string | undefined) {
  return Number(amount || 0).toFixed(2)
}

export default function UserBalanceLogList() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const showError = useUnhandledError()
  const userId = searchParams.get('user_id') || ''
  const [stats, setStats] = useState<{
    total_income?: number | string
    total_expense?: number | string
    total_refund?: number | string
    current_balance?: number | string
  } | null>(null)

  const {
    tableData,
    loading,
    pagination,
    searchForm,
    setSearchForm,
    onSearchFormChange,
    loadData,
    handleSearch,
    handleReset,
    refresh,
    handleSortChange,
  } = useListPage<LogRow, Record<string, unknown>>({
    fetchApi: getUserBalanceLogList,
    initialSearchForm: { user_id: userId, type: '', source: '' },
    defaultSort: 'id:desc',
    transformData: (row) => {
      const record = row
      return {
        id: entityField(record, 'id', '')!,
        type: String(entityField(record, 'type', '') ?? ''),
        amount: entityField(record, 'amount', 0) as number | string,
        balance: entityField(record, 'balance', 0) as number | string,
        source: String(entityField(record, 'source', '') ?? ''),
        description: String(entityField(record, 'description', '') ?? ''),
        created_at: String(entityField(record, 'created_at', '') ?? ''),
      }
    },
  })

  const loadStats = async (uid = String(searchForm.user_id || userId || '')) => {
    if (!uid) {
      setStats(null)
      return
    }
    try {
      const res = await getUserBalanceStatistics({ user_id: uid })
      setStats((res.data as typeof stats) || null)
    } catch (error) {
      showError(error, t('common.query_failed'))
    }
  }

  useEffect(() => {
    if (userId && String(searchForm.user_id || '') !== userId) {
      setSearchForm({ ...searchForm, user_id: userId })
    }
    void loadStats(userId)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userId])

  const columns: ColumnsType<LogRow> = useMemo(() => {
    const typeTag = (type?: string) => {
      if (type === 'income') return <Tag color="success">{t('user.balance_income')}</Tag>
      if (type === 'expense') return <Tag color="error">{t('user.balance_expense')}</Tag>
      if (type === 'refund') return <Tag color="warning">{t('user.balance_refund')}</Tag>
      return type || '-'
    }
    const sourceLabel = (source?: string) => {
      const map: Record<string, string> = {
        order: t('user.source_order'),
        recharge: t('user.source_recharge'),
        withdraw: t('user.source_withdraw'),
        manual: t('user.source_manual'),
      }
      return (source && map[source]) || source || '-'
    }
    return [
      { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
      { title: t('user.type'), dataIndex: 'type', width: 100, render: (v: string) => typeTag(v) },
      {
        title: t('user.amount'),
        dataIndex: 'amount',
        width: 120,
        render: (amount, row) => (
          <span style={{ color: row.type === 'expense' ? '#ff4d4f' : '#52c41a' }}>
            {row.type === 'expense' ? '-' : '+'}¥{formatMoney(amount)}
          </span>
        ),
      },
      {
        title: t('user.balance'),
        dataIndex: 'balance',
        width: 120,
        render: (balance) => <span style={{ fontWeight: 600 }}>¥{formatMoney(balance)}</span>,
      },
      { title: t('user.source'), dataIndex: 'source', width: 100, render: (v: string) => sourceLabel(v) },
      { title: t('user.description'), dataIndex: 'description', ellipsis: true },
      { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
    ]
  }, [t])

  const {
    filteredColumns,
    open: columnSettingOpen,
    openColumnSetting,
    closeColumnSetting,
    allColumns,
    visibleColumns,
    columnOrder,
    fixedColumns,
    handleConfirm: handleColumnSettingConfirm,
  } = useColumnSetting('user_balance_log', columns)

  return (
    <PageContainer
      title={
        <Space>
          <Button type="link" icon={<ArrowLeftOutlined />} onClick={() => navigate(-1)}>
            {t('user.back')}
          </Button>
          <span>
            {t('user.balance_logs')}
            {searchForm.user_id ? ` (${t('user.user_id')}: ${String(searchForm.user_id)})` : ''}
          </span>
        </Space>
      }
      extra={
        <Space>
          <Button
            onClick={() => {
              void refresh()
              void loadStats()
            }}
          >
            {t('common.refresh')}
          </Button>
          <Button icon={<SettingOutlined />} onClick={openColumnSetting}>
            {t('common.column_setting')}
          </Button>
        </Space>
      }
    >
      {stats && (
        <Card style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col span={6}>
              <Statistic title={t('user.total_income')} value={formatMoney(stats.total_income)} prefix="¥" />
            </Col>
            <Col span={6}>
              <Statistic title={t('user.total_expense')} value={formatMoney(stats.total_expense)} prefix="¥" />
            </Col>
            <Col span={6}>
              <Statistic title={t('user.total_refund')} value={formatMoney(stats.total_refund)} prefix="¥" />
            </Col>
            <Col span={6}>
              <Statistic title={t('user.current_balance')} value={formatMoney(stats.current_balance)} prefix="¥" />
            </Col>
          </Row>
        </Card>
      )}

      <SearchForm
        fields={[
          { name: 'user_id', label: t('user.user_id') },
          {
            name: 'type',
            label: t('user.type'),
            type: 'select',
            options: [
              { label: t('user.balance_income'), value: 'income' },
              { label: t('user.balance_expense'), value: 'expense' },
              { label: t('user.balance_refund'), value: 'refund' },
            ],
          },
          {
            name: 'source',
            label: t('user.source'),
            type: 'select',
            options: [
              { label: t('user.source_order'), value: 'order' },
              { label: t('user.source_recharge'), value: 'recharge' },
              { label: t('user.source_withdraw'), value: 'withdraw' },
              { label: t('user.source_manual'), value: 'manual' },
            ],
          },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={() => {
          handleSearch()
          void loadStats(String(searchForm.user_id || ''))
        }}
        onReset={() => {
          handleReset()
          void loadStats(userId)
        }}
      />

      <Table<LogRow>
        rowKey="id"
        loading={loading}
        columns={filteredColumns}
        dataSource={tableData}
        scroll={{ x: 1000 }}
        pagination={{
          current: pagination.page,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
          showTotal: (total) => t('common.total', { total }),
        }}
        onChange={(pager, _f, sorter) =>
          handlePaginatedTableChange({ pager, sorter, pagination, loadData, handleSortChange })
        }
      />
      <ColumnSettingDialog
        open={columnSettingOpen}
        onClose={closeColumnSetting}
        allColumns={allColumns}
        visibleColumns={visibleColumns}
        columnOrder={columnOrder}
        fixedColumns={fixedColumns}
        onConfirm={handleColumnSettingConfirm}
      />
    </PageContainer>
  )
}
