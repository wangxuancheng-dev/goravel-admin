import { useCallback, useMemo } from 'react'
import { Button, Card, Col, List, Row, Statistic, Tag, Typography } from 'antd'
import {
  EyeOutlined,
  MenuOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  ShoppingCartOutlined,
} from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import type { EChartsOption } from 'echarts'
import { useTranslation } from 'react-i18next'
import { useUserStore } from '@/stores/user'
import PageContainer from '@/components/PageContainer'
import { useDashboard } from './dashboard/useDashboard'
import './Dashboard.scss'

function hexToRgba(hex: string, alpha = 1) {
  const m = hex.match(/^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i)
  if (!m) return `rgba(64, 158, 255, ${alpha})`
  return `rgba(${parseInt(m[1], 16)}, ${parseInt(m[2], 16)}, ${parseInt(m[3], 16)}, ${alpha})`
}

export default function DashboardPage() {
  const { t } = useTranslation()
  const adminInfo = useUserStore((s) => s.adminInfo)
  const {
    loading,
    counts,
    weeklyActivity,
    accessSource,
    monthlyOperations,
    activities,
    darkMode,
    primaryColor,
    refresh,
  } = useDashboard()

  const textColor = darkMode ? '#e5eaf3' : '#303133'
  // 暗色取值与 stores/app.ts 的 colorBorder / colorBorderSecondary / colorBgElevated 保持一致
  const axisLineColor = darkMode ? '#2e3a50' : '#dcdfe6'
  const splitLineColor = darkMode ? 'rgba(255,255,255,0.08)' : '#ebeef5'
  const tooltipStyle = useMemo(
    () => ({
      backgroundColor: darkMode ? '#1a2436' : '#ffffff',
      borderColor: darkMode ? '#2e3a50' : '#dcdfe6',
      borderWidth: 1,
      textStyle: { color: darkMode ? '#f5f7fa' : '#303133' },
    }),
    [darkMode],
  )

  const cards = [
    {
      key: 'order_count_in_year',
      title: t('dashboard.order_count_in_year', { defaultValue: t('dashboard.orders') }),
      value: counts.order_count_in_year,
      icon: <ShoppingCartOutlined />,
      color: primaryColor,
    },
    {
      key: 'today_visits',
      title: t('dashboard.today_visits', { defaultValue: t('dashboard.users') }),
      value: counts.today_visits,
      icon: <EyeOutlined />,
      color: '#67C23A',
    },
    {
      key: 'role_count',
      title: t('dashboard.role_count', { defaultValue: t('dashboard.roles') }),
      value: counts.role_count,
      icon: <SafetyCertificateOutlined />,
      color: '#E6A23C',
    },
    {
      key: 'menu_count',
      title: t('dashboard.menu_count', { defaultValue: t('dashboard.admins') }),
      value: counts.menu_count,
      icon: <MenuOutlined />,
      color: '#F56C6C',
    },
  ]

  const weeklyActivityOption: EChartsOption = useMemo(
    () => ({
      tooltip: { trigger: 'axis', ...tooltipStyle },
      legend: {
        data: [t('dashboard.visits'), t('dashboard.active_admins')],
        textStyle: { color: textColor },
      },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: weeklyActivity.map((item) => item.date),
        axisLabel: { color: textColor },
        axisLine: { lineStyle: { color: axisLineColor } },
      },
      yAxis: {
        type: 'value',
        axisLabel: { color: textColor },
        axisLine: { lineStyle: { color: axisLineColor } },
        splitLine: { lineStyle: { color: splitLineColor } },
      },
      series: [
        {
          name: t('dashboard.visits'),
          type: 'line',
          smooth: true,
          data: weeklyActivity.map((item) => item.visits),
          itemStyle: { color: primaryColor },
          areaStyle: {
            color: {
              type: 'linear',
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: hexToRgba(primaryColor, 0.3) },
                { offset: 1, color: hexToRgba(primaryColor, 0.1) },
              ],
            },
          },
        },
        {
          name: t('dashboard.active_admins'),
          type: 'line',
          smooth: true,
          data: weeklyActivity.map((item) => item.users),
          itemStyle: { color: '#67C23A' },
        },
      ],
    }),
    [weeklyActivity, primaryColor, textColor, axisLineColor, splitLineColor, tooltipStyle, t],
  )

  const buildPieOption = useCallback(
    (title: string): EChartsOption => ({
      tooltip: { trigger: 'item', formatter: '{a} <br/>{b}: {c} ({d}%)', ...tooltipStyle },
      legend: { orient: 'vertical', left: 'left', top: 'middle', textStyle: { color: textColor } },
      series: [
        {
          name: title,
          type: 'pie',
          radius: ['40%', '70%'],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 10, borderColor: darkMode ? '#141c2b' : '#fff', borderWidth: 2 },
          label: { show: true, formatter: '{b}: {d}%', color: textColor },
          emphasis: { label: { show: true, fontSize: 16, fontWeight: 'bold', color: textColor } },
          data: accessSource.map((item) => ({ name: item.name, value: item.value })),
        },
      ],
    }),
    [accessSource, textColor, tooltipStyle, darkMode],
  )

  const accessSourceOption = useMemo(
    () => buildPieOption(t('dashboard.access_source')),
    [buildPieOption, t],
  )
  const deviceDistributionOption = useMemo(
    () => buildPieOption(t('dashboard.device_distribution')),
    [buildPieOption, t],
  )

  const monthlyOperationsOption: EChartsOption = useMemo(
    () => ({
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, ...tooltipStyle },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'value',
        axisLabel: { color: textColor },
        axisLine: { lineStyle: { color: axisLineColor } },
        splitLine: { lineStyle: { color: splitLineColor } },
      },
      yAxis: {
        type: 'category',
        data: monthlyOperations.map((item) => item.month),
        axisLabel: { color: textColor },
        axisLine: { lineStyle: { color: axisLineColor } },
      },
      series: [
        {
          name: t('dashboard.monthly_operations'),
          type: 'bar',
          data: monthlyOperations.map((item) => item.count),
          itemStyle: { color: primaryColor },
        },
      ],
    }),
    [monthlyOperations, primaryColor, textColor, axisLineColor, splitLineColor, tooltipStyle, t],
  )

  return (
    <PageContainer
      title={t('menu.dashboard')}
      extra={
        <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void refresh()}>
          {t('dashboard.refresh')}
        </Button>
      }
    >
      <Typography.Title level={3} className="dashboard-welcome">
        {t('dashboard.welcome')}
        {adminInfo?.nickname || adminInfo?.username ? `，${adminInfo.nickname || adminInfo.username}` : ''}
      </Typography.Title>
      <Typography.Paragraph type="secondary" className="dashboard-subtitle">
        {t('dashboard.subtitle')}
      </Typography.Paragraph>

      <Row gutter={[16, 16]}>
        {cards.map((card) => (
          <Col xs={24} sm={12} lg={6} key={card.key}>
            <Card loading={loading} className="dashboard-kpi-card" bordered={false}>
              <div className="dashboard-kpi">
                <div className="dashboard-kpi__icon" style={{ background: card.color }}>
                  {card.icon}
                </div>
                <Statistic title={card.title} value={card.value || 0} />
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={[16, 16]} className="dashboard-section">
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.weekly_activity')} loading={loading} className="dashboard-chart-card">
            <ReactECharts option={weeklyActivityOption} style={{ height: 320 }} notMerge lazyUpdate />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.access_source')} loading={loading} className="dashboard-chart-card">
            <ReactECharts option={accessSourceOption} style={{ height: 320 }} notMerge lazyUpdate />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} className="dashboard-section">
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.device_distribution')} loading={loading} className="dashboard-chart-card">
            <ReactECharts option={deviceDistributionOption} style={{ height: 320 }} notMerge lazyUpdate />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.monthly_operations')} loading={loading} className="dashboard-chart-card">
            <ReactECharts option={monthlyOperationsOption} style={{ height: 320 }} notMerge lazyUpdate />
          </Card>
        </Col>
      </Row>

      <Card title={t('dashboard.recent')} className="dashboard-section" loading={loading}>
        <List
          locale={{ emptyText: t('common.no_data') }}
          dataSource={activities}
          renderItem={(item) => (
            <List.Item>
              <List.Item.Meta
                title={
                  <span>
                    <strong>{item.user}</strong> {item.action}
                  </span>
                }
                description={item.time}
              />
              <Tag color={item.type === 'danger' ? 'error' : 'success'}>{item.status}</Tag>
            </List.Item>
          )}
        />
      </Card>
    </PageContainer>
  )
}
