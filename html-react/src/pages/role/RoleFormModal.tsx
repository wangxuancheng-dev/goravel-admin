import { useEffect, useMemo, useState } from 'react'
import { App, Form, Input, InputNumber, Modal, Radio, Select, Spin, Tree, TreeSelect } from 'antd'
import type { DataNode } from 'antd/es/tree'
import { useTranslation } from 'react-i18next'
import { createRole, getRoleDetail, updateRole } from '@/api/role'
import { getMenuTree } from '@/api/menu'
import { getPermissionList } from '@/api/permission'
import { getOptions, type OptionItem } from '@/api/option'
import { normalizeOptions, toOptionTreeData } from '@/hooks/useOptions'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { entityField, normalizeEntity } from '@/utils/normalize'
import { resolveMenuTitle, resolvePermissionTitle } from '@/utils/menuTitle'
import {
  buildCheckedKeysFromIds,
  buildMenuPermissionTree,
  collectIdsFromCheckedKeys,
  flattenMenuList,
  flattenPermissionList,
  type TreeNodeData,
} from '@/utils/rolePermissionTree'

interface RoleFormModalProps {
  open: boolean
  editId?: string | number | null
  onClose: () => void
  onSuccess: () => void
}

const PROTECTED = 'super-admin'

export default function RoleFormModal({ open, editId, onClose, onSuccess }: RoleFormModalProps) {
  const { t, i18n } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [menus, setMenus] = useState<Record<string, unknown>[]>([])
  const [permissions, setPermissions] = useState<Record<string, unknown>[]>([])
  const [checkedKeys, setCheckedKeys] = useState<string[]>([])
  const [slug, setSlug] = useState('')
  const [deptTree, setDeptTree] = useState<OptionItem[]>([])
  const dataScope = Form.useWatch('data_scope', form)

  const isProtected = slug === PROTECTED
  const deptTreeData = useMemo(() => toOptionTreeData(deptTree), [deptTree])

  const dataScopeOptions = useMemo(
    () => [
      { value: 1, label: t('role.data_scope_all') },
      { value: 2, label: t('role.data_scope_custom') },
      { value: 3, label: t('role.data_scope_dept') },
      { value: 4, label: t('role.data_scope_dept_and_child') },
      { value: 5, label: t('role.data_scope_self') },
    ],
    [t],
  )

  const treeTranslate = useMemo(
    () => ({
      menuTitle: (menu: Record<string, unknown>) =>
        resolveMenuTitle(t, {
          slug: String(menu.Slug || menu.slug || ''),
          fallback: String(menu.Title || menu.title || menu.Name || menu.name || menu.Slug || menu.slug || ''),
        }),
      permissionTitle: (perm: Record<string, unknown>) =>
        resolvePermissionTitle(i18n, {
          slug: String(perm.Slug || perm.slug || ''),
          fallback: String(
            perm.Description || perm.description || perm.Name || perm.name || perm.Slug || perm.slug || '',
          ),
        }),
      otherLabel: t('role.other_permissions', { defaultValue: '其他权限' }),
    }),
    [t, i18n, i18n.language],
  )

  const tree = useMemo<TreeNodeData[]>(() => {
    if (!menus.length && !permissions.length) return []
    return buildMenuPermissionTree(menus, permissions, treeTranslate)
  }, [menus, permissions, treeTranslate])

  useEffect(() => {
    if (!open) return

    let cancelled = false
    const boot = async () => {
      setLoading(true)
      try {
        const [menuRes, permRes, deptRes] = await Promise.all([
          getMenuTree(),
          getPermissionList({ page_size: 1000 }),
          getOptions('department'),
        ])
        if (cancelled) return

        const flatMenus = flattenMenuList(menuRes.data)
        const flatPerms = flattenPermissionList(permRes.data)
        setMenus(flatMenus)
        setPermissions(flatPerms)
        setDeptTree(normalizeOptions(deptRes.data))

        if (!editId) {
          form.setFieldsValue({
            name: '',
            slug: '',
            description: '',
            status: 1,
            sort: 0,
            data_scope: 1,
            department_ids: [],
          })
          setSlug('')
          setCheckedKeys([])
          return
        }

        const detail = await getRoleDetail(editId)
        if (cancelled) return
        const raw = (detail.data || {}) as Record<string, unknown>
        const data = normalizeEntity(
          (entityField(raw, 'role', raw) || {}) as Record<string, unknown>,
        ) as Record<string, unknown>
        const nextSlug = String(entityField(data, 'slug', '') ?? '')
        const menuIds = ((entityField(data, 'menu_ids', []) as number[]) || []).map(Number)
        const permissionIds = ((entityField(data, 'permission_ids', []) as number[]) || []).map(Number)
        const menusArr = (entityField(data, 'menus', []) as Array<Record<string, unknown>>) || []
        const permsArr = (entityField(data, 'permissions', []) as Array<Record<string, unknown>>) || []
        const deptsArr = (entityField(data, 'departments', []) as Array<Record<string, unknown>>) || []
        const resolvedMenuIds =
          menuIds.length > 0
            ? menuIds
            : menusArr.map((m) => Number(entityField(m, 'id', 0))).filter(Boolean)
        const resolvedPermIds =
          permissionIds.length > 0
            ? permissionIds
            : permsArr.map((p) => Number(entityField(p, 'id', 0))).filter(Boolean)
        const departmentIds = deptsArr.map((d) => Number(entityField(d, 'id', 0))).filter(Boolean)
        const scope = Number(entityField(data, 'data_scope', 1) ?? 1)

        form.setFieldsValue({
          name: entityField(data, 'name', ''),
          slug: nextSlug,
          description: entityField(data, 'description', ''),
          status: Number(entityField(data, 'status', 1)),
          sort: Number(entityField(data, 'sort', 0)),
          data_scope: nextSlug === PROTECTED ? 1 : scope || 1,
          department_ids: nextSlug === PROTECTED ? [] : departmentIds,
        })
        setSlug(nextSlug)
        const builtTree = buildMenuPermissionTree(flatMenus, flatPerms, treeTranslate)
        setCheckedKeys(buildCheckedKeysFromIds(builtTree, resolvedMenuIds, resolvedPermIds))
      } catch (error) {
        showError(error, i18n.t('common.query_failed'))
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void boot()
    return () => {
      cancelled = true
    }
  }, [open, editId, form, showError, i18n, treeTranslate])

  const antdTreeData = useMemo<DataNode[]>(() => {
    const convert = (nodes: TreeNodeData[]): DataNode[] =>
      nodes.map((n) => ({
        key: n.key,
        title: n.title,
        children: n.children ? convert(n.children) : undefined,
        disableCheckbox: isProtected,
      }))
    return convert(tree)
  }, [tree, isProtected])

  const handleOk = async () => {
    try {
      const values = await form.validateFields()
      if (isProtected && Number(values.status) === 0) {
        message.warning(t('role.protected_cannot_disable', { defaultValue: '超级管理员角色不能禁用' }))
        return
      }

      setSubmitting(true)
      const payload: Record<string, unknown> = {
        name: values.name,
        slug: values.slug,
        description: values.description || '',
        status: Number(values.status),
        sort: Number(values.sort) || 0,
        data_scope: isProtected ? 1 : Number(values.data_scope) || 1,
        department_ids:
          isProtected || Number(values.data_scope) !== 2
            ? []
            : ((values.department_ids as Array<string | number>) || [])
                .map((id) => Number(id))
                .filter((id) => id > 0),
      }

      if (!isProtected) {
        const ids = collectIdsFromCheckedKeys(tree, checkedKeys)
        payload.menu_ids = ids.menu_ids
        payload.permission_ids = ids.permission_ids
      }

      if (editId) {
        await updateRole(editId, payload)
        message.success(t('common.update_success'))
      } else {
        await createRole(payload)
        message.success(t('common.create_success'))
      }
      onSuccess()
      onClose()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal
      open={open}
      title={editId ? t('role.edit') : t('role.add')}
      onCancel={onClose}
      onOk={() => void handleOk()}
      confirmLoading={submitting}
      width={900}
      destroyOnHidden
      styles={{ body: { maxHeight: '70vh', overflow: 'auto' } }}
    >
      <Spin spinning={loading}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('role.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="slug" label={t('role.slug')} rules={[{ required: true }]}>
            <Input
              disabled={isProtected}
              onChange={(e) => setSlug(e.target.value)}
            />
          </Form.Item>
          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea rows={2} />
          </Form.Item>

          <Form.Item name="data_scope" label={t('role.data_scope')} initialValue={1}>
            <Select options={dataScopeOptions} disabled={isProtected} />
          </Form.Item>
          {!isProtected && Number(dataScope) === 2 ? (
            <Form.Item name="department_ids" label={t('role.data_scope_depts')}>
              <TreeSelect
                treeData={deptTreeData}
                treeCheckable
                showCheckedStrategy={TreeSelect.SHOW_PARENT}
                allowClear
                placeholder={t('role.data_scope_depts_placeholder')}
                style={{ width: '100%' }}
              />
            </Form.Item>
          ) : null}

          <Form.Item label={t('role.menus_and_permissions', { defaultValue: '菜单与权限' })}>
            {isProtected ? (
              <div style={{ color: 'rgba(0,0,0,0.45)' }}>
                {t('role.super_admin_has_all_permissions', {
                  defaultValue: '超级管理员拥有全部菜单与权限，无需单独配置',
                })}
              </div>
            ) : (
              <div
                style={{
                  border: '1px solid #f0f0f0',
                  borderRadius: 8,
                  padding: 12,
                  maxHeight: 420,
                  overflow: 'auto',
                }}
              >
                <Tree
                  checkable
                  selectable={false}
                  treeData={antdTreeData}
                  checkedKeys={checkedKeys}
                  onCheck={(keys) => {
                    const next = Array.isArray(keys) ? keys : keys.checked
                    setCheckedKeys(next.map(String))
                  }}
                />
              </div>
            )}
          </Form.Item>

          <Form.Item name="status" label={t('common.status')}>
            <Radio.Group
              options={[
                { label: t('common.enabled'), value: 1 },
                { label: t('common.disabled'), value: 0, disabled: isProtected },
              ]}
            />
          </Form.Item>
          <Form.Item name="sort" label={t('common.sort')}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Spin>
    </Modal>
  )
}
