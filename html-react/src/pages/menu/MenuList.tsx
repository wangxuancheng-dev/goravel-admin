import { useEffect, useMemo, useState } from 'react'
import { App, Button, Form, Input, InputNumber, Modal, Radio, Select, Space, Table, Tag, TreeSelect } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { createMenu, deleteMenu, getMenuList, updateMenu } from '@/api/menu'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import PageContainer from '@/components/PageContainer'
import StatusTag from '@/components/StatusTag'
import PermissionButton from '@/components/PermissionButton'
import { entityField, normalizeTreeList } from '@/utils/normalize'
import { excludeNodeAndChildren, flattenTree } from '@/utils/tree'

interface MenuRow {
  id: number | string
  title?: string
  name?: string
  slug?: string
  path?: string
  component?: string
  icon?: string
  type?: number
  status?: number
  sort?: number
  parent_id?: number | string
  link_type?: number
  open_type?: number
  no_cache?: number
  is_hidden?: number
  children?: MenuRow[]
}

type TreeSelectNode = { title: string; value: number | string; children?: TreeSelectNode[] }

function flattenOrTree(list: unknown[]): MenuRow[] {
  return normalizeTreeList(list as Record<string, unknown>[]).map((item) => {
    const row = item as Record<string, unknown>
    return {
      id: entityField(row, 'id', '')!,
      title: String(entityField(row, 'title', '') ?? entityField(row, 'name', '') ?? ''),
      name: String(entityField(row, 'title', '') ?? entityField(row, 'name', '') ?? ''),
      slug: String(entityField(row, 'slug', '') ?? ''),
      path: String(entityField(row, 'path', '') ?? ''),
      component: String(entityField(row, 'component', '') ?? ''),
      icon: String(entityField(row, 'icon', '') ?? ''),
      type: Number(entityField(row, 'type', 1) ?? 1),
      status: Number(entityField(row, 'status', 1) ?? 1),
      sort: Number(entityField(row, 'sort', 0) ?? 0),
      parent_id: entityField(row, 'parent_id', 0) as number | string,
      link_type: Number(entityField(row, 'link_type', 1) ?? 1) || 1,
      open_type: Number(entityField(row, 'open_type', 1) ?? 1) || 1,
      no_cache: Number(entityField(row, 'no_cache', 0) ?? 0),
      is_hidden: Number(entityField(row, 'is_hidden', 0) ?? 0),
      children: Array.isArray(row.children)
        ? flattenOrTree(row.children as unknown[])
        : undefined,
    }
  })
}

function toTreeSelect(items: MenuRow[]): TreeSelectNode[] {
  return items.map((item) => ({
    title: String(item.title || item.name || item.id),
    value: item.id,
    children: item.children?.length ? toTreeSelect(item.children) : undefined,
  }))
}

function collectIds(items: MenuRow[]): Array<string | number> {
  return flattenTree(items).map((node) => (node as MenuRow).id)
}

export default function MenuList() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [form] = Form.useForm()
  const linkType = Form.useWatch('link_type', form)
  const [data, setData] = useState<MenuRow[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [editId, setEditId] = useState<string | number | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [expandedKeys, setExpandedKeys] = useState<Array<string | number>>([])

  const load = async () => {
    setLoading(true)
    try {
      const res = await getMenuList({ page_size: 1000 })
      const list = (res.data?.list ?? res.data?.data ?? res.data ?? []) as unknown[]
      const rows = flattenOrTree(Array.isArray(list) ? list : [])
      setData(rows)
      setExpandedKeys(collectIds(rows))
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const defaultFormValues = {
    title: '',
    slug: '',
    path: '',
    component: '',
    icon: '',
    type: 2,
    status: 1,
    sort: 0,
    parent_id: 0,
    link_type: 1,
    open_type: 1,
    no_cache: 0,
    is_hidden: 0,
  }

  const { toolbar, confirmDelete } = useCrudActions({
    createPermission: 'menu.store',
    onRefresh: load,
    onCreate: () => {
      setEditId(null)
      form.setFieldsValue(defaultFormValues)
      setOpen(true)
    },
    deleteApi: deleteMenu,
  })

  const parentTreeData = useMemo(() => {
    const filtered = editId ? excludeNodeAndChildren(data, editId) : data
    return [{ title: t('menu_page.top_menu'), value: 0 }, ...toTreeSelect(filtered)]
  }, [data, editId, t])

  const allExpanded = expandedKeys.length > 0

  const typeLabel = (type?: number) => {
    if (type === 1) return t('menu_page.type_directory')
    if (type === 3) return t('menu_page.type_button')
    return t('menu_page.type_menu')
  }

  const columns: ColumnsType<MenuRow> = [
    { title: t('common.name'), dataIndex: 'title', width: 200 },
    { title: t('common.slug'), dataIndex: 'slug', width: 120 },
    { title: t('common.path'), dataIndex: 'path', width: 140 },
    { title: t('common.component'), dataIndex: 'component', ellipsis: true, width: 160 },
    {
      title: t('menu_management.link_type'),
      dataIndex: 'link_type',
      width: 110,
      render: (v: number) => (
        <Tag color={v === 1 ? 'blue' : 'green'}>
          {v === 1 ? t('menu_management.link_type_internal') : t('menu_management.link_type_external')}
        </Tag>
      ),
    },
    {
      title: t('menu_management.open_type'),
      dataIndex: 'open_type',
      width: 120,
      render: (v: number, row) =>
        row.link_type === 2 ? (
          <Tag color={v === 1 ? 'default' : 'orange'}>
            {v === 1 ? t('menu_management.open_type_iframe') : t('menu_management.open_type_new_window')}
          </Tag>
        ) : (
          '-'
        ),
    },
    {
      title: t('menu_management.no_cache'),
      dataIndex: 'no_cache',
      width: 90,
      render: (v: number) =>
        v === 1 ? <Tag>{t('menu_management.no_cache_no')}</Tag> : <Tag color="success">{t('menu_management.no_cache_yes')}</Tag>,
    },
    {
      title: t('menu_management.is_hidden'),
      dataIndex: 'is_hidden',
      width: 90,
      render: (v: number) => (
        <Tag color={v === 0 ? 'success' : 'default'}>
          {v === 0 ? t('menu_management.is_hidden_show') : t('menu_management.is_hidden_hide')}
        </Tag>
      ),
    },
    {
      title: t('common.type'),
      dataIndex: 'type',
      width: 90,
      render: (type: number) => <Tag>{typeLabel(type)}</Tag>,
    },
    {
      title: t('common.status'),
      dataIndex: 'status',
      width: 90,
      render: (status: number) => <StatusTag status={status} />,
    },
    { title: t('common.sort'), dataIndex: 'sort', width: 70 },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 160,
      fixed: 'end',
      render: (_, row) => (
        <Space>
          {getButtonState('menu.update').show && (
            <PermissionButton
              permission="menu.update"
              type="link"
              onClick={() => {
                setEditId(row.id)
                form.setFieldsValue({
                  ...row,
                  parent_id: Number(row.parent_id || 0),
                  link_type: row.link_type || 1,
                  open_type: row.open_type || 1,
                  no_cache: row.no_cache ?? 0,
                  is_hidden: row.is_hidden ?? 0,
                })
                setOpen(true)
              }}
            >
              {t('common.edit')}
            </PermissionButton>
          )}
          {getButtonState('menu.destroy').show && (
            <PermissionButton
              permission="menu.destroy"
              type="link"
              danger
              onClick={() => confirmDelete(row.id, row.title)}
            >
              {t('common.delete')}
            </PermissionButton>
          )}
        </Space>
      ),
    },
  ]

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const link = Number(values.link_type || 1)
      const payload = {
        ...values,
        title: values.title,
        parent_id: Number(values.parent_id || 0),
        type: Number(values.type),
        status: Number(values.status),
        sort: Number(values.sort || 0),
        link_type: link,
        open_type: Number(values.open_type || 1),
        no_cache: Number(values.no_cache || 0),
        is_hidden: Number(values.is_hidden || 0),
        component: link === 1 ? values.component : '',
      }
      if (editId) {
        await updateMenu(editId, payload)
        message.success(t('common.update_success'))
      } else {
        await createMenu(payload)
        message.success(t('common.create_success'))
      }
      setOpen(false)
      await load()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <PageContainer
      title={t('menu.menu_management')}
      extra={
        <Space>
          <Button onClick={() => setExpandedKeys(allExpanded ? [] : collectIds(data))}>
            {allExpanded ? t('common.collapse') : t('common.expand')}
          </Button>
          {toolbar}
        </Space>
      }
    >
      <Table<MenuRow>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={data}
        pagination={false}
        scroll={{ x: 1600 }}
        expandable={{
          expandedRowKeys: expandedKeys,
          onExpandedRowsChange: (keys) => setExpandedKeys(keys as Array<string | number>),
        }}
      />
      <Modal
        open={open}
        title={editId ? t('menu_page.edit') : t('menu_page.add')}
        onCancel={() => setOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnHidden
        width={680}
      >
        <Form form={form} layout="vertical" initialValues={defaultFormValues}>
          <Form.Item name="title" label={t('common.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="slug" label={t('common.slug')} rules={[{ required: true }]}>
            <Input placeholder={t('menu_management.slug_placeholder')} />
          </Form.Item>
          <Form.Item name="parent_id" label={t('common.parent')}>
            <TreeSelect
              allowClear
              treeDefaultExpandAll
              treeData={parentTreeData}
              placeholder={t('common.parent')}
            />
          </Form.Item>
          <Form.Item name="link_type" label={t('menu_management.link_type')}>
            <Radio.Group
              options={[
                { label: t('menu_management.link_type_internal'), value: 1 },
                { label: t('menu_management.link_type_external'), value: 2 },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="path"
            label={t('common.path')}
            rules={[
              { required: true, message: t('menu_management.path_required') },
              ...(linkType === 2
                ? [
                    {
                      type: 'url' as const,
                      message: t('menu_management.path_url_invalid'),
                    },
                  ]
                : []),
            ]}
          >
            <Input
              placeholder={
                linkType === 2
                  ? t('menu_management.path_placeholder_external')
                  : t('menu_management.path_placeholder_internal')
              }
            />
          </Form.Item>
          {linkType === 2 ? (
            <Form.Item name="open_type" label={t('menu_management.open_type')}>
              <Radio.Group
                options={[
                  { label: t('menu_management.open_type_iframe'), value: 1 },
                  { label: t('menu_management.open_type_new_window'), value: 2 },
                ]}
              />
            </Form.Item>
          ) : (
            <>
              <Form.Item name="component" label={t('common.component')}>
                <Input placeholder={t('menu_management.component_placeholder')} />
              </Form.Item>
              <Form.Item name="no_cache" label={t('menu_management.no_cache')}>
                <Radio.Group
                  options={[
                    { label: t('menu_management.no_cache_yes'), value: 0 },
                    { label: t('menu_management.no_cache_no'), value: 1 },
                  ]}
                />
              </Form.Item>
            </>
          )}
          <Form.Item name="icon" label={t('common.icon')}>
            <Input placeholder="Setting" />
          </Form.Item>
          <Form.Item name="type" label={t('common.type')}>
            <Select
              options={[
                { label: t('menu_page.type_directory'), value: 1 },
                { label: t('menu_page.type_menu'), value: 2 },
                { label: t('menu_page.type_button'), value: 3 },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="is_hidden"
            label={t('menu_management.is_hidden')}
            extra={t('menu_management.is_hidden_tip')}
          >
            <Radio.Group
              options={[
                { label: t('menu_management.is_hidden_show'), value: 0 },
                { label: t('menu_management.is_hidden_hide'), value: 1 },
              ]}
            />
          </Form.Item>
          <Form.Item name="sort" label={t('common.sort')}>
            <InputNumber style={{ width: '100%' }} min={0} />
          </Form.Item>
          <Form.Item name="status" label={t('common.status')}>
            <Radio.Group
              options={[
                { label: t('common.enabled'), value: 1 },
                { label: t('common.disabled'), value: 0 },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
