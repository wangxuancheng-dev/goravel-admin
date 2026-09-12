import { Tabs } from 'antd'
import { useTranslation } from 'react-i18next'
import PageContainer from '@/components/PageContainer'
import ExportList from './ExportList'
import ImportList from './ImportList'

export default function TaskCenter() {
  const { t } = useTranslation()

  return (
    <PageContainer title={t('menu.export')}>
      <Tabs
        items={[
          {
            key: 'exports',
            label: t('export.tab_exports'),
            children: <ExportList embedded />,
          },
          {
            key: 'imports',
            label: t('export.tab_imports'),
            children: <ImportList />,
          },
        ]}
      />
    </PageContainer>
  )
}
