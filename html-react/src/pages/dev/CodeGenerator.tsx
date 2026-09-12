import {
  Alert,
  Button,
  Card,
  Checkbox,
  Col,
  Descriptions,
  Divider,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Radio,
  Row,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
} from 'antd'
import { DeleteOutlined, FileTextOutlined, PlusOutlined, SettingOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import PageContainer from '@/components/PageContainer'
import { useCodeGenerator } from './code-generator/useCodeGenerator'
import type { CodeGeneratorField } from '@/api/codeGenerator'

export default function CodeGenerator() {
  const { t } = useTranslation()
  const cg = useCodeGenerator()

  const fieldColumns = [
    {
      title: t('table.index'),
      width: 60,
      render: (_: unknown, __: CodeGeneratorField, index: number) => index + 1,
    },
    {
      title: t('code_generator.field_name'),
      width: 150,
      render: (_: unknown, row: CodeGeneratorField, index: number) => (
        <Input
          value={row.name}
          placeholder={t('code_generator.field_name_placeholder')}
          onChange={(e) => cg.updateField(index, { name: e.target.value })}
        />
      ),
    },
    {
      title: t('code_generator.field_type'),
      width: 150,
      render: (_: unknown, row: CodeGeneratorField, index: number) => (
        <Select
          style={{ width: '100%' }}
          value={row.type}
          options={cg.fieldTypes.map((type) => ({ value: type.value, label: type.label }))}
          onChange={(value) => cg.updateField(index, { type: value })}
        />
      ),
    },
    {
      title: t('code_generator.field_label'),
      width: 150,
      render: (_: unknown, row: CodeGeneratorField, index: number) => (
        <Input
          value={row.label}
          placeholder={t('code_generator.field_label_placeholder')}
          onChange={(e) => cg.updateField(index, { label: e.target.value })}
        />
      ),
    },
    {
      title: t('code_generator.form_type'),
      width: 150,
      render: (_: unknown, row: CodeGeneratorField, index: number) => (
        <Select
          style={{ width: '100%' }}
          value={row.form_type}
          options={cg.formTypes}
          onChange={(value) => cg.updateField(index, { form_type: value })}
        />
      ),
    },
    {
      title: t('code_generator.search_type'),
      width: 120,
      render: (_: unknown, row: CodeGeneratorField, index: number) => (
        <Select
          style={{ width: '100%' }}
          value={row.search_type}
          options={['like', '=', '>', '>=', '<', '<=', '!=', 'in'].map((value) => ({ value, label: value.toUpperCase() }))}
          onChange={(value) => cg.updateField(index, { search_type: value })}
        />
      ),
    },
    {
      title: t('code_generator.search_ui_type'),
      width: 150,
      render: (_: unknown, row: CodeGeneratorField, index: number) => (
        <Select
          style={{ width: '100%' }}
          value={row.search_ui_type}
          options={[
            { value: 'input', label: t('code_generator.search_ui_types.input') },
            { value: 'select', label: t('code_generator.search_ui_types.select') },
            { value: 'date', label: t('code_generator.search_ui_types.date') },
            { value: 'datetime', label: t('code_generator.search_ui_types.datetime') },
            { value: 'daterange', label: t('code_generator.search_ui_types.daterange') },
            { value: 'datetimerange', label: t('code_generator.search_ui_types.datetimerange') },
          ]}
          onChange={(value) => cg.updateField(index, { search_ui_type: value })}
        />
      ),
    },
    {
      title: t('code_generator.relation'),
      width: 150,
      render: (_: unknown, row: CodeGeneratorField, index: number) => (
        <Button size="small" onClick={() => cg.handleEditRelation(index)}>
          {row.relation ? t('code_generator.edit_relation') : t('code_generator.add_relation')}
        </Button>
      ),
    },
    {
      title: t('code_generator.field_config'),
      width: 100,
      render: (_: unknown, row: CodeGeneratorField, index: number) =>
        row.form_type === 'select' ||
        row.form_type === 'radio' ||
        row.form_type === 'checkbox' ||
        row.type === 'decimal' ||
        row.search_ui_type === 'select' ? (
          <Button size="small" icon={<SettingOutlined />} onClick={() => cg.handleEditFieldConfig(index)} />
        ) : null,
    },
    {
      title: t('code_generator.field_options'),
      width: 320,
      render: (_: unknown, row: CodeGeneratorField, index: number) => (
        <Space wrap size={[8, 4]}>
          <Checkbox checked={row.required} onChange={(e) => cg.updateField(index, { required: e.target.checked })}>
            {t('code_generator.required')}
          </Checkbox>
          <Checkbox checked={row.searchable} onChange={(e) => cg.updateField(index, { searchable: e.target.checked })}>
            {t('code_generator.searchable')}
          </Checkbox>
          <Checkbox checked={row.sortable} onChange={(e) => cg.updateField(index, { sortable: e.target.checked })}>
            {t('code_generator.sortable')}
          </Checkbox>
          <Checkbox checked={row.show_in_list} onChange={(e) => cg.updateField(index, { show_in_list: e.target.checked })}>
            {t('code_generator.show_in_list')}
          </Checkbox>
          <Checkbox checked={row.show_in_form} onChange={(e) => cg.updateField(index, { show_in_form: e.target.checked })}>
            {t('code_generator.show_in_form')}
          </Checkbox>
        </Space>
      ),
    },
    {
      title: t('table.operation'),
      width: 80,
      fixed: 'end' as const,
      render: (_: unknown, __: CodeGeneratorField, index: number) => (
        <Button danger size="small" icon={<DeleteOutlined />} onClick={() => cg.handleRemoveField(index)} />
      ),
    },
  ]

  const currentField = cg.currentFieldIndex != null ? cg.fields[cg.currentFieldIndex] : null

  return (
    <PageContainer title={t('code_generator.title')}>
      <Card
        extra={
          cg.activeMode === 'manual' ? (
            <Space>
              <Button loading={cg.installing} onClick={() => void cg.handleInstallModule()}>
                {t('code_generator.install_module')}
              </Button>
              <Button type="primary" loading={cg.generating} icon={<FileTextOutlined />} onClick={() => void cg.handleGenerate()}>
                {t('code_generator.generate')}
              </Button>
            </Space>
          ) : null
        }
      >
        <Tabs
          activeKey={cg.activeMode}
          onChange={cg.setActiveMode}
          type="card"
          items={[
            {
              key: 'manual',
              label: t('code_generator.manual_mode'),
              children: (
                <Form form={cg.form} layout="vertical" initialValues={{ module_name: '', table_name: '' }}>
                  <Row gutter={20}>
                    <Col span={12}>
                      <Form.Item label={t('code_generator.select_table')}>
                        <Select
                          showSearch
                          allowClear
                          value={cg.selectedTable || undefined}
                          placeholder={t('code_generator.select_table_placeholder')}
                          options={cg.tables.map((table) => ({ value: table, label: table }))}
                          onChange={(value) => void cg.handleTableChange(value || '')}
                        />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Row gutter={20}>
                    <Col span={12}>
                      <Form.Item
                        name="module_name"
                        label={t('code_generator.module_name')}
                        rules={[{ required: true, message: t('code_generator.module_name_required') }]}
                      >
                        <Input placeholder={t('code_generator.module_name_placeholder')} />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item
                        name="table_name"
                        label={t('code_generator.table_name')}
                        rules={[{ required: true, message: t('code_generator.table_name_required') }]}
                      >
                        <Input placeholder={t('code_generator.table_name_placeholder')} />
                      </Form.Item>
                    </Col>
                  </Row>

                  <Row gutter={20}>
                    <Col span={12}>
                      <Form.Item
                        name="menu_title"
                        label={t('code_generator.menu_title')}
                      >
                        <Input
                          placeholder={t('code_generator.menu_title_placeholder')}
                          value={cg.menuTitle}
                          onChange={(e) => {
                            cg.setMenuTitle(e.target.value)
                            cg.form.setFieldValue('menu_title', e.target.value)
                          }}
                        />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item label={t('code_generator.parent_menu')}>
                        <Select
                          allowClear
                          showSearch
                          optionFilterProp="label"
                          value={cg.parentMenuSlug}
                          options={cg.parentMenuOptions}
                          placeholder={t('code_generator.parent_menu_placeholder')}
                          onChange={(value) => cg.setParentMenuSlug(value || '')}
                        />
                      </Form.Item>
                    </Col>
                  </Row>

                  <Row gutter={20}>
                    <Col span={12}>
                      <Form.Item label={t('code_generator.menu_sort')}>
                        <InputNumber
                          min={0}
                          style={{ width: '100%' }}
                          value={cg.menuSort}
                          onChange={(value) => cg.setMenuSort(Number(value) || 0)}
                        />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item label={t('code_generator.install_options')}>
                        <Space wrap>
                          <Checkbox checked={cg.installEnabled} onChange={(e) => cg.setInstallEnabled(e.target.checked)}>
                            {t('code_generator.install_on_save')}
                          </Checkbox>
                        </Space>
                      </Form.Item>
                    </Col>
                  </Row>

                  <Divider>{t('code_generator.fields_config')}</Divider>

                  <Form.Item label={t('code_generator.generated_files')}>
                    <Checkbox.Group
                      value={cg.files}
                      onChange={(values) => cg.setFiles(values as string[])}
                      options={cg.fileTypes.map((item) => ({ value: item.value, label: item.label }))}
                    />
                  </Form.Item>

                  <Form.Item label={t('code_generator.function_options')}>
                    <Checkbox.Group
                      value={cg.options}
                      onChange={(values) => cg.setOptions(values as string[])}
                      options={[
                        { value: 'has_create', label: t('code_generator.has_create') },
                        { value: 'has_edit', label: t('code_generator.has_edit') },
                        { value: 'has_delete', label: t('code_generator.has_delete') },
                        { value: 'enable_batch_actions', label: t('code_generator.enable_batch_actions') },
                        { value: 'show_toolbar', label: t('code_generator.show_toolbar') },
                        { value: 'is_tree_list', label: t('code_generator.is_tree_list') },
                      ]}
                    />
                  </Form.Item>

                  <Form.Item label={t('code_generator.export_mode')}>
                    <Radio.Group value={cg.exportMode} onChange={(e) => cg.setExportMode(e.target.value)}>
                      <Radio value="none">{t('code_generator.export_mode_none')}</Radio>
                      <Radio value="sync">{t('code_generator.export_mode_sync')}</Radio>
                      <Radio value="async">{t('code_generator.export_mode_async')}</Radio>
                    </Radio.Group>
                  </Form.Item>

                  <Form.Item label={t('code_generator.import_mode')}>
                    <Radio.Group value={cg.importMode} onChange={(e) => cg.setImportMode(e.target.value)}>
                      <Radio value="none">{t('code_generator.import_mode_none')}</Radio>
                      <Radio value="sync">{t('code_generator.import_mode_sync')}</Radio>
                      <Radio value="async">{t('code_generator.import_mode_async')}</Radio>
                    </Radio.Group>
                  </Form.Item>

                  <Button type="primary" icon={<PlusOutlined />} onClick={cg.handleAddField}>
                    {t('code_generator.add_field')}
                  </Button>

                  <Table
                    style={{ marginTop: 20 }}
                    bordered
                    size="small"
                    rowKey={(_, index) => String(index)}
                    dataSource={cg.fields}
                    columns={fieldColumns}
                    pagination={false}
                    scroll={{ x: 1800 }}
                  />

                  <Divider>{t('code_generator.code_preview')}</Divider>

                  <Tabs
                    activeKey={cg.activeTab}
                    onChange={cg.setActiveTab}
                    type="card"
                    items={cg.fileTypes.map((fileType) => ({
                      key: fileType.value,
                      label: fileType.label,
                      children: (
                        <div>
                          <Button
                            type="primary"
                            size="small"
                            loading={cg.previewing === fileType.value}
                            onClick={() => void cg.handlePreview(fileType.value)}
                          >
                            {t('code_generator.refresh_preview')}
                          </Button>
                          {cg.previewCodeMap[fileType.value] ? (
                            <pre style={{ marginTop: 12, padding: 12, background: '#f5f5f5', overflow: 'auto', maxHeight: 480 }}>
                              <code>{cg.previewCodeMap[fileType.value]}</code>
                            </pre>
                          ) : (
                            <Empty style={{ marginTop: 24 }} description={t('code_generator.click_preview')} />
                          )}
                        </div>
                      ),
                    }))}
                  />
                </Form>
              ),
            },
            {
              key: 'ai',
              label: t('code_generator.ai_mode'),
              children: (
                <div>
                  {!cg.aiEnabled ? (
                    <Alert
                      type="warning"
                      showIcon
                      message={t('code_generator.ai_not_configured_title')}
                      description={t('code_generator.ai_not_configured_tip')}
                      style={{ marginBottom: 20 }}
                    />
                  ) : (
                    <>
                      <Alert type="info" showIcon message={t('code_generator.ai_mode_tip')} style={{ marginBottom: 20 }} />
                      <Form layout="vertical">
                          <Form.Item label={t('code_generator.ai_example_title')}>
                            <Space wrap>
                              {cg.aiExamplePrompts.map((prompt) => (
                                <Tag key={prompt} style={{ cursor: 'pointer' }} onClick={() => cg.applyAIExample(prompt)}>
                                  {prompt}
                                </Tag>
                              ))}
                            </Space>
                          </Form.Item>
                          <Form.Item label={t('code_generator.ai_description')}>
                            <Input.TextArea
                              rows={8}
                              value={cg.aiDescription}
                              placeholder={t('code_generator.ai_description_placeholder')}
                              onChange={(e) => cg.setAiDescription(e.target.value)}
                            />
                          </Form.Item>
                          <Form.Item>
                            <Space>
                              <Button type="primary" loading={cg.aiGenerating} icon={<FileTextOutlined />} onClick={() => void cg.handleGenerateWithAI()}>
                                {t('code_generator.ai_generate')}
                              </Button>
                              {cg.aiLastError ? (
                                <Button disabled={cg.aiGenerating} onClick={() => void cg.handleGenerateWithAI()}>
                                  {t('code_generator.ai_retry')}
                                </Button>
                              ) : null}
                              <Button disabled={!cg.aiGeneratedConfig} onClick={cg.handleApplyAIConfig}>
                                {t('code_generator.ai_apply_config')}
                              </Button>
                            </Space>
                          </Form.Item>
                        </Form>

                        {cg.aiGeneratedConfig ? (
                          <>
                            <Divider>{t('code_generator.ai_generated_config')}</Divider>
                            <Descriptions bordered column={2} size="small">
                              <Descriptions.Item label={t('code_generator.module_name')}>{cg.aiGeneratedConfig.module_name}</Descriptions.Item>
                              <Descriptions.Item label={t('code_generator.table_name')}>{cg.aiGeneratedConfig.table_name}</Descriptions.Item>
                              <Descriptions.Item label={t('code_generator.fields_count')} span={2}>
                                {cg.aiGeneratedConfig.fields?.length || 0}
                              </Descriptions.Item>
                            </Descriptions>
                            <Table
                              style={{ marginTop: 20 }}
                              bordered
                              size="small"
                              rowKey="name"
                              dataSource={cg.aiGeneratedConfig.fields || []}
                              pagination={false}
                              scroll={{ y: 400 }}
                              columns={[
                                { title: t('table.index'), width: 60, render: (_v, _r, i) => i + 1 },
                                { title: t('code_generator.field_name'), dataIndex: 'name', width: 120 },
                                { title: t('code_generator.field_label'), dataIndex: 'label', width: 120 },
                                { title: t('code_generator.field_type'), dataIndex: 'db_type', width: 100 },
                                { title: t('code_generator.form_type'), dataIndex: 'form_type', width: 120 },
                                {
                                  title: t('code_generator.required'),
                                  dataIndex: 'required',
                                  width: 80,
                                  render: (val: boolean) => <Tag color={val ? 'success' : 'default'}>{val ? t('common.yes') : t('common.no')}</Tag>,
                                },
                                {
                                  title: t('code_generator.searchable'),
                                  dataIndex: 'searchable',
                                  width: 80,
                                  render: (val: boolean) => <Tag color={val ? 'success' : 'default'}>{val ? t('common.yes') : t('common.no')}</Tag>,
                                },
                              ]}
                            />
                          </>
                        ) : null}
                    </>
                  )}
                </div>
              ),
            },
          ]}
        />
      </Card>

      <Modal
        title={t('code_generator.relation_config')}
        open={cg.relationDialogOpen}
        onCancel={() => cg.setRelationDialogOpen(false)}
        onOk={cg.handleSaveRelation}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <Form layout="vertical">
          <Form.Item label={t('code_generator.relation_table')}>
            <Input
              value={cg.relationForm.table}
              placeholder={t('code_generator.relation_table_placeholder')}
              onChange={(e) => cg.setRelationForm({ ...cg.relationForm, table: e.target.value })}
            />
          </Form.Item>
          <Form.Item label={t('code_generator.relation_type')}>
            <Select
              value={cg.relationForm.relation_type}
              options={[
                { value: 'belongsTo', label: 'belongsTo' },
                { value: 'hasOne', label: 'hasOne' },
                { value: 'hasMany', label: 'hasMany' },
              ]}
              onChange={(value) => cg.setRelationForm({ ...cg.relationForm, relation_type: value })}
            />
          </Form.Item>
          <Form.Item label={t('code_generator.foreign_key')}>
            <Input
              value={cg.relationForm.foreign_key}
              placeholder={t('code_generator.foreign_key_placeholder')}
              onChange={(e) => cg.setRelationForm({ ...cg.relationForm, foreign_key: e.target.value })}
            />
          </Form.Item>
          <Form.Item label={t('code_generator.display_field')}>
            <Input
              value={cg.relationForm.display_field}
              placeholder={t('code_generator.display_field_placeholder')}
              onChange={(e) => cg.setRelationForm({ ...cg.relationForm, display_field: e.target.value })}
            />
          </Form.Item>
          {currentField?.api_url ? (
            <Form.Item>
              <Checkbox
                checked={cg.relationForm.is_tree}
                onChange={(e) => cg.setRelationForm({ ...cg.relationForm, is_tree: e.target.checked })}
              >
                {t('code_generator.is_tree_desc')}
              </Checkbox>
            </Form.Item>
          ) : null}
        </Form>
      </Modal>

      <Modal
        title={t('code_generator.field_config')}
        open={cg.fieldConfigDialogOpen}
        onCancel={() => cg.setFieldConfigDialogOpen(false)}
        onOk={cg.handleSaveFieldConfig}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        {currentField?.type === 'decimal' ? (
          <Form layout="vertical">
            <Form.Item label={t('code_generator.precision')}>
              <InputNumber
                min={1}
                max={65}
                value={cg.fieldConfigForm.precision}
                onChange={(value) => cg.setFieldConfigForm({ ...cg.fieldConfigForm, precision: value || 8 })}
              />
            </Form.Item>
            <Form.Item label={t('code_generator.scale')}>
              <InputNumber
                min={0}
                max={30}
                value={cg.fieldConfigForm.scale}
                onChange={(value) => cg.setFieldConfigForm({ ...cg.fieldConfigForm, scale: value || 2 })}
              />
            </Form.Item>
          </Form>
        ) : (
          <Form layout="vertical">
            <Form.Item label={t('code_generator.option_type')}>
              <Radio.Group
                value={cg.fieldConfigForm.option_type}
                onChange={(e) => cg.setFieldConfigForm({ ...cg.fieldConfigForm, option_type: e.target.value })}
              >
                <Radio value="dictionary">{t('code_generator.option_type_dictionary')}</Radio>
                <Radio value="api">{t('code_generator.option_type_api')}</Radio>
              </Radio.Group>
            </Form.Item>
            {cg.fieldConfigForm.option_type === 'dictionary' ? (
              <Form.Item label={t('code_generator.dictionary_key')}>
                <Select
                  showSearch
                  allowClear
                  value={cg.fieldConfigForm.dictionary || undefined}
                  placeholder={t('code_generator.dictionary_key_placeholder')}
                  options={cg.dictionaryTypes.map((item) => ({ value: item, label: item }))}
                  onChange={(value) => cg.setFieldConfigForm({ ...cg.fieldConfigForm, dictionary: value || '' })}
                />
              </Form.Item>
            ) : (
              <>
                <Form.Item label={t('code_generator.api_url')}>
                  <Input
                    value={cg.fieldConfigForm.api_url}
                    placeholder={t('code_generator.api_url_placeholder')}
                    onChange={(e) => cg.setFieldConfigForm({ ...cg.fieldConfigForm, api_url: e.target.value })}
                  />
                </Form.Item>
                <Form.Item>
                  <Checkbox
                    checked={cg.fieldConfigForm.is_tree}
                    onChange={(e) => cg.setFieldConfigForm({ ...cg.fieldConfigForm, is_tree: e.target.checked })}
                  >
                    {t('code_generator.is_tree_desc')}
                  </Checkbox>
                </Form.Item>
              </>
            )}
          </Form>
        )}
      </Modal>
    </PageContainer>
  )
}
