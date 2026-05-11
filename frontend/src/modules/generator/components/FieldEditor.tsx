import React, { useState } from 'react';
import {
  Button,
  Card,
  Form,
  Grid,
  Input,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { AppModal, showAppModalConfirm } from '../../../components';
import './FieldEditor.css';

import {
  applyFieldTemplate,
  FIELD_TEMPLATE_DEFINITIONS,
  normalizeField,
  type EnumOption,
  type FieldTemplateKey,
  type FieldType,
  type ModuleField,
} from '../schema';
import { TYPE_MAPPING } from '../type-mapping';

interface FieldEditorProps {
  fields: ModuleField[];
  onChange: (fields: ModuleField[]) => void;
}

const FormItem = Form.Item;
const { Row, Col } = Grid;

function enumOptionsToText(options?: EnumOption[]) {
  return (options ?? [])
    .map((item) => `${item.value}|${item.label}${item.color ? `|${item.color}` : ''}`)
    .join('\n');
}

function enumOptionsToEnglishText(options?: EnumOption[]) {
  return (options ?? []).map((item) => `${item.value}|${item.labelEn || item.label}`).join('\n');
}

function parseEnumOptions(text?: string): EnumOption[] {
  return (text || '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [value, label, color] = line.split('|').map((item) => item.trim());
      return {
        value,
        label: label || value,
        color: color || undefined,
      };
    })
    .filter((item) => item.value);
}

function parseEnumOptionTranslations(text?: string): Record<string, string> {
  return (text || '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .reduce<Record<string, string>>((acc, line) => {
      const [value, label] = line.split('|').map((item) => item.trim());
      if (value) {
        acc[value] = label || value;
      }
      return acc;
    }, {});
}

export const FieldEditor: React.FC<FieldEditorProps> = ({ fields, onChange }) => {
  const { t } = useTranslation();
  const resolveTemplateText = (locale: 'zh-CN' | 'en-US', key: string, fallback: string) =>
    t(key, { lng: locale, defaultValue: fallback });
  const [visible, setVisible] = useState(false);
  const [editingFieldName, setEditingFieldName] = useState<string | null>(null);
  const [templateToApply, setTemplateToApply] = useState<FieldTemplateKey>('none');
  const [form] = Form.useForm<
    ModuleField & { enumOptionsText?: string; enumOptionsEnText?: string; unique?: boolean }
  >();

  const openCreateModal = () => {
    setEditingFieldName(null);
    setTemplateToApply('none');
    form.resetFields();
    form.setFieldsValue({
      type: 'string',
      required: false,
      searchable: false,
      sortable: false,
      visibleInList: true,
      visibleInForm: true,
      unique: false,
      enumOptionsText: '',
      enumOptionsEnText: '',
      templateKey: 'none',
    });
    setVisible(true);
  };

  const openEditModal = (record: ModuleField) => {
    setEditingFieldName(record.name);
    setTemplateToApply(record.templateKey || 'none');
    form.setFieldsValue({
      ...record,
      unique: record.validation?.unique ?? false,
      enumOptionsText: enumOptionsToText(record.enumOptions),
      enumOptionsEnText: enumOptionsToEnglishText(record.enumOptions),
      templateKey: record.templateKey || 'none',
    });
    setVisible(true);
  };

  const handleDelete = (name: string) => {
    showAppModalConfirm({
      title: t('generator.fieldEditor.confirmDelete'),
      content: t('generator.fieldEditor.confirmDelete.content'),
      onOk: () => {
        onChange(fields.filter((item) => item.name !== name));
        message.success(t('common.deleteSuccess'));
      },
    });
  };

  const handleTemplateApply = (templateKey: FieldTemplateKey) => {
    setTemplateToApply(templateKey);
    if (templateKey === 'none') {
      return;
    }
    const preset = applyFieldTemplate(templateKey, resolveTemplateText);
    if (!preset) {
      return;
    }
    form.setFieldsValue({
      ...preset,
      unique: preset.validation?.unique ?? false,
      enumOptionsText: enumOptionsToText(preset.enumOptions),
      enumOptionsEnText: enumOptionsToEnglishText(preset.enumOptions),
      templateKey,
    });
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validate();
      const enumOptions =
        values.type === 'enum' ? parseEnumOptions(values.enumOptionsText) : undefined;
      const enumOptionTranslations =
        values.type === 'enum' ? parseEnumOptionTranslations(values.enumOptionsEnText) : {};
      const nextField = normalizeField({
        name: values.name,
        type: values.type,
        label: values.label,
        labelEn: values.labelEn,
        required: values.required,
        searchable: values.searchable,
        sortable: values.sortable,
        visibleInList: values.visibleInList,
        visibleInForm: values.visibleInForm,
        defaultValue: values.defaultValue,
        placeholder: values.placeholder,
        placeholderEn: values.placeholderEn,
        helpText: values.helpText,
        helpTextEn: values.helpTextEn,
        dictCode: values.type === 'enum' ? values.dictCode : undefined,
        enumOptions: enumOptions?.map((item) => ({
          ...item,
          labelEn: enumOptionTranslations[item.value] || item.labelEn || item.label,
        })),
        templateKey: values.templateKey || undefined,
        validation: {
          ...values.validation,
          unique: values.unique,
          enum: enumOptions?.map((item) => item.value),
        },
      });

      const duplicated = fields.some(
        (field) => field.name === nextField.name && field.name !== editingFieldName,
      );
      if (duplicated) {
        message.error(t('generator.fieldEditor.duplicateName'));
        return;
      }

      const nextFields = editingFieldName
        ? fields.map((field) => (field.name === editingFieldName ? nextField : field))
        : [...fields, nextField];
      onChange(nextFields);
      setVisible(false);
      message.success(editingFieldName ? t('common.updateSuccess') : t('common.createSuccess'));
    } catch {
      // form validate handled by Arco
    }
  };

  const columns = [
    {
      title: t('generator.fieldEditor.name'),
      dataIndex: 'name',
      width: 160,
    },
    {
      title: t('generator.fieldEditor.label'),
      dataIndex: 'label',
      width: 160,
    },
    {
      title: t('generator.fieldEditor.type'),
      dataIndex: 'type',
      width: 120,
      render: (type: FieldType) => <code>{TYPE_MAPPING[type].go}</code>,
    },
    {
      title: t('generator.fieldEditor.unique'),
      dataIndex: 'validation',
      width: 90,
      render: (validation: ModuleField['validation']) => (validation?.unique ? '✓' : '-'),
    },
    {
      title: t('generator.fieldEditor.enumDict'),
      dataIndex: 'dictCode',
      width: 140,
      render: (dictCode: string, record: ModuleField) =>
        record.type === 'enum' ? (
          <Tag color="arcoblue">{dictCode || t('generator.fieldEditor.enumInline')}</Tag>
        ) : (
          '-'
        ),
    },
    {
      title: t('generator.fieldEditor.template'),
      dataIndex: 'templateKey',
      width: 140,
      render: (templateKey: FieldTemplateKey) => {
        const definition = FIELD_TEMPLATE_DEFINITIONS.find((item) => item.key === templateKey);
        return definition ? <Tag>{t(definition.labelKey)}</Tag> : '-';
      },
    },
    {
      title: t('common.actions'),
      width: 120,
      render: (_: unknown, record: ModuleField) => (
        <Space>
          <Button type="text" onClick={() => openEditModal(record)}>
            <IconEdit />
          </Button>
          <Button type="text" status="danger" onClick={() => handleDelete(record.name)}>
            <IconDelete />
          </Button>
        </Space>
      ),
    },
  ];

  const currentType = form.getFieldValue('type') as FieldType | undefined;

  return (
    <div>
      <Card size="small" className="generator-field-editor__section">
        <Space wrap>
          <Typography.Text type="secondary">{t('generator.fieldEditor.templates')}</Typography.Text>
          {FIELD_TEMPLATE_DEFINITIONS.map((template) => (
            <Button
              key={template.key}
              size="small"
              onClick={() => {
                const preset = applyFieldTemplate(template.key, resolveTemplateText);
                if (!preset || fields.some((field) => field.name === preset.name)) {
                  message.warning(t('generator.fieldEditor.templateExists'));
                  return;
                }
                onChange([...fields, normalizeField(preset)]);
              }}
            >
              {t(template.labelKey)}
            </Button>
          ))}
        </Space>
      </Card>

      <Button
        type="primary"
        onClick={openCreateModal}
        className="generator-field-editor__add-button"
      >
        <IconPlus /> {t('generator.fieldEditor.addField')}
      </Button>

      <Table columns={columns} data={fields} rowKey="name" pagination={false} />

      <AppModal
        title={editingFieldName ? t('generator.fieldEditor.edit') : t('generator.fieldEditor.add')}
        visible={visible}
        onOk={handleSubmit}
        onCancel={() => setVisible(false)}
        size="detail"
      >
        <Form form={form} layout="vertical" onSubmit={handleSubmit}>
          <Row gutter={16}>
            <Col span={12}>
              <FormItem label={t('generator.fieldEditor.template')} field="templateKey">
                <Select value={templateToApply} onChange={handleTemplateApply}>
                  <Select.Option value="none">
                    {t('generator.fieldEditor.template.none')}
                  </Select.Option>
                  {FIELD_TEMPLATE_DEFINITIONS.map((template) => (
                    <Select.Option key={template.key} value={template.key}>
                      {t(template.labelKey)}
                    </Select.Option>
                  ))}
                </Select>
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem
                label={t('generator.fieldEditor.name')}
                field="name"
                rules={[
                  { required: true, message: t('common.required') },
                  {
                    match: /^[a-z][a-zA-Z0-9]*$/,
                    message: t('generator.fieldEditor.name.pattern'),
                  },
                ]}
                disabled={!!editingFieldName}
              >
                <Input placeholder="orderNo" onPressEnter={() => form.submit()} />
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem
                label={t('generator.fieldEditor.label')}
                field="label"
                rules={[{ required: true, message: t('common.required') }]}
              >
                <Input
                  placeholder={t('generator.fieldEditor.label.placeholder')}
                  onPressEnter={() => form.submit()}
                />
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem label={t('generator.fieldEditor.labelEn')} field="labelEn">
                <Input
                  placeholder={t('generator.fieldEditor.labelEn.placeholder')}
                  onPressEnter={() => form.submit()}
                />
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem
                label={t('generator.fieldEditor.type')}
                field="type"
                rules={[{ required: true, message: t('common.required') }]}
              >
                <Select>
                  <Select.Option value="string">
                    {t('generator.fieldEditor.type.string')}
                  </Select.Option>
                  <Select.Option value="text">{t('generator.fieldEditor.type.text')}</Select.Option>
                  <Select.Option value="int">{t('generator.fieldEditor.type.int')}</Select.Option>
                  <Select.Option value="float">
                    {t('generator.fieldEditor.type.float')}
                  </Select.Option>
                  <Select.Option value="bool">{t('generator.fieldEditor.type.bool')}</Select.Option>
                  <Select.Option value="date">{t('generator.fieldEditor.type.date')}</Select.Option>
                  <Select.Option value="enum">{t('generator.fieldEditor.type.enum')}</Select.Option>
                  <Select.Option value="relation">
                    {t('generator.fieldEditor.type.relation')}
                  </Select.Option>
                </Select>
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem label={t('generator.fieldEditor.placeholder')} field="placeholder">
                <Input onPressEnter={() => form.submit()} />
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem label={t('generator.fieldEditor.placeholderEn')} field="placeholderEn">
                <Input onPressEnter={() => form.submit()} />
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem label={t('generator.fieldEditor.defaultValue')} field="defaultValue">
                <Input onPressEnter={() => form.submit()} />
              </FormItem>
            </Col>
            <Col span={24}>
              <FormItem label={t('generator.fieldEditor.helpText')} field="helpText">
                <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
              </FormItem>
            </Col>
            <Col span={24}>
              <FormItem label={t('generator.fieldEditor.helpTextEn')} field="helpTextEn">
                <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
              </FormItem>
            </Col>
            {currentType === 'enum' ? (
              <>
                <Col span={12}>
                  <FormItem label={t('generator.fieldEditor.dictCode')} field="dictCode">
                    <Input
                      placeholder={t('generator.fieldEditor.dictCode.placeholder')}
                      onPressEnter={() => form.submit()}
                    />
                  </FormItem>
                </Col>
                <Col span={24}>
                  <FormItem label={t('generator.fieldEditor.enumOptions')} field="enumOptionsText">
                    <Input.TextArea
                      autoSize={{ minRows: 4, maxRows: 8 }}
                      placeholder={t('generator.fieldEditor.enumOptions.placeholder')}
                    />
                  </FormItem>
                </Col>
                <Col span={24}>
                  <FormItem
                    label={t('generator.fieldEditor.enumOptionsEn')}
                    field="enumOptionsEnText"
                  >
                    <Input.TextArea
                      autoSize={{ minRows: 4, maxRows: 8 }}
                      placeholder={t('generator.fieldEditor.enumOptionsEn.placeholder')}
                    />
                  </FormItem>
                </Col>
              </>
            ) : null}
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <FormItem
                label={t('generator.fieldEditor.required')}
                field="required"
                triggerPropName="checked"
              >
                <Switch />
              </FormItem>
            </Col>
            <Col span={8}>
              <FormItem
                label={t('generator.fieldEditor.unique')}
                field="unique"
                triggerPropName="checked"
              >
                <Switch />
              </FormItem>
            </Col>
            <Col span={8}>
              <FormItem
                label={t('generator.fieldEditor.searchable')}
                field="searchable"
                triggerPropName="checked"
              >
                <Switch />
              </FormItem>
            </Col>
            <Col span={8}>
              <FormItem
                label={t('generator.fieldEditor.sortable')}
                field="sortable"
                triggerPropName="checked"
              >
                <Switch />
              </FormItem>
            </Col>
            <Col span={8}>
              <FormItem
                label={t('generator.fieldEditor.visibleInList')}
                field="visibleInList"
                triggerPropName="checked"
                initialValue
              >
                <Switch />
              </FormItem>
            </Col>
            <Col span={8}>
              <FormItem
                label={t('generator.fieldEditor.visibleInForm')}
                field="visibleInForm"
                triggerPropName="checked"
                initialValue
              >
                <Switch />
              </FormItem>
            </Col>
          </Row>
        </Form>
      </AppModal>
    </div>
  );
};
