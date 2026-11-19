import { Alert, Button, Card, Empty, Space, Table, Tag, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { getPlatformTenantHealthBoard } from '@/api/platform'
import { formatDateTimeDisplay } from '@/utils/dateUtils'

interface BoardItem {
  id: number
  code: string
  name: string
  health_status: 'warn' | 'fail'
  issues: string[]
  last_ping_ok: boolean
  last_ping_ms: number
  health_checked_at: string | null
  maintenance: boolean
  stale: boolean
}

interface Board {
  checked: number
  stale: number
  never_checked: number
  last_checked_at: string | null
  stale_after_sec: number
  by_issue: Record<string, number>
  abnormal: number
  truncated: boolean
  items: BoardItem[]
}

// Fail-level issues first so the chip row reads by severity.
const ISSUE_ORDER = [
  'ping_fail',
  'provision_failed',
  'migrate_failed',
  'schema_failed',
  'storage_quota_full',
  'schema_behind',
  'storage_quota_high',
  'domain_verify_failed',
]
const FAIL_ISSUES = new Set(['ping_fail', 'provision_failed', 'migrate_failed', 'schema_failed', 'storage_quota_full'])

interface Props {
  /** Bump to reload, e.g. after a manual inspect. */
  reloadKey?: number
}

export default function TenantHealthBoard({ reloadKey = 0 }: Props) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [board, setBoard] = useState<Board | null>(null)
  const [issueFilter, setIssueFilter] = useState('')

  const issueLabel = useCallback(
    (code: string) => {
      const key = `tenant.health_issue_${code}`
      return i18n.exists(key) ? t(key) : code
    },
    [i18n, t],
  )

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getPlatformTenantHealthBoard()
      setBoard(((res as { data?: { board?: Board } })?.data?.board as Board) || null)
    } catch {
      setBoard(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load, reloadKey])

  const issueChips = useMemo(() => {
    const byIssue = board?.by_issue || {}
    return Object.keys(byIssue)
      .filter((code) => (byIssue[code] || 0) > 0)
      .sort((a, b) => {
        const ia = ISSUE_ORDER.indexOf(a)
        const ib = ISSUE_ORDER.indexOf(b)
        return (ia === -1 ? 99 : ia) - (ib === -1 ? 99 : ib) || a.localeCompare(b)
      })
      .map((code) => ({ code, count: byIssue[code] }))
  }, [board])

  // Drop a selected issue once a reload no longer reports it, so the table never sticks empty.
  const activeFilter = issueFilter && (board?.by_issue?.[issueFilter] || 0) > 0 ? issueFilter : ''

  const rows = useMemo(() => {
    const items = board?.items || []
    return activeFilter ? items.filter((it) => it.issues.includes(activeFilter)) : items
  }, [board, activeFilter])

  const columns: ColumnsType<BoardItem> = [
    {
      title: t('tenant.health_status'),
      dataIndex: 'health_status',
      width: 90,
      render: (v: string) => (
        <Tag color={v === 'fail' ? 'error' : 'warning'}>{v === 'fail' ? t('tenant.health_fail') : t('tenant.health_warn')}</Tag>
      ),
    },
    {
      title: t('menu.tenant'),
      dataIndex: 'code',
      render: (_: string, row) => (
        <Space direction="vertical" size={0}>
          <Space size={6}>
            <Typography.Text strong>{row.code}</Typography.Text>
            {row.maintenance ? <Tag color="default">{t('tenant.maintenance_on')}</Tag> : null}
          </Space>
          {row.name && row.name !== row.code ? (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {row.name}
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('tenant.health_board_issues'),
      dataIndex: 'issues',
      render: (issues: string[]) => (
        <Space size={[4, 4]} wrap>
          {issues.map((code) => (
            <Tag key={code} color={FAIL_ISSUES.has(code) ? 'red' : 'orange'}>
              {issueLabel(code)}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: t('tenant.health_board_ping'),
      dataIndex: 'last_ping_ms',
      width: 100,
      render: (ms: number, row) =>
        row.last_ping_ok ? `${ms} ms` : <Typography.Text type="danger">—</Typography.Text>,
    },
    {
      title: t('tenant.health_board_checked_at'),
      dataIndex: 'health_checked_at',
      width: 180,
      render: (v: string | null, row) => (
        <Typography.Text type={row.stale ? 'warning' : undefined}>
          {v ? formatDateTimeDisplay(v) : t('tenant.health_board_never')}
        </Typography.Text>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 110,
      render: (_: unknown, row) => (
        <Button type="link" size="small" onClick={() => navigate(`/platform/tenants?code=${encodeURIComponent(row.code)}`)}>
          {t('tenant.health_board_view')}
        </Button>
      ),
    },
  ]

  const staleHours = Math.round((board?.stale_after_sec || 7200) / 3600)

  return (
    <Card
      size="small"
      loading={loading && !board}
      title={t('tenant.health_board_title')}
      extra={
        <Space size={12}>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {board?.last_checked_at
              ? t('tenant.health_board_last_checked', { t: formatDateTimeDisplay(board.last_checked_at) })
              : t('tenant.health_board_never')}
          </Typography.Text>
          <Button type="link" size="small" onClick={() => void load()}>
            {t('common.refresh')}
          </Button>
        </Space>
      }
    >
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        {(board?.stale || 0) > 0 ? (
          <Alert type="warning" showIcon message={t('tenant.health_board_stale', { n: board?.stale, h: staleHours })} />
        ) : null}

        {issueChips.length > 0 ? (
          <Space size={[8, 8]} wrap>
            <Tag.CheckableTag checked={!activeFilter} onChange={() => setIssueFilter('')}>
              {t('tenant.health_board_all_issues')} ({board?.abnormal ?? 0})
            </Tag.CheckableTag>
            {issueChips.map(({ code, count }) => (
              <Tag.CheckableTag
                key={code}
                checked={activeFilter === code}
                onChange={(checked) => setIssueFilter(checked ? code : '')}
              >
                {issueLabel(code)} ({count})
              </Tag.CheckableTag>
            ))}
          </Space>
        ) : null}

        {board && board.abnormal === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('tenant.health_board_no_issue')} />
        ) : (
          <Table<BoardItem>
            rowKey="id"
            size="small"
            loading={loading}
            columns={columns}
            dataSource={rows}
            pagination={rows.length > 10 ? { pageSize: 10, size: 'small' } : false}
            scroll={{ x: 760 }}
          />
        )}

        {board?.truncated ? (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('tenant.health_board_truncated', { n: board.items.length })}
          </Typography.Text>
        ) : null}
      </Space>
    </Card>
  )
}
