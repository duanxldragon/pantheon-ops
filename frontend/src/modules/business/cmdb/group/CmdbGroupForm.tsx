import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, Input, Select, Button, Space, Radio } from '@arco-design/web-react';
import { IconPlus, IconDelete } from '@arco-design/web-react/icon';
import type { GroupRow, ConditionRule } from './api';
import { getLabelSchemaList, type LabelSchemaRow } from '../label/api';
import { isFreeValueLabel, labelValueOptions } from '../label/options';
import SubmitBar from '../../../../components/patterns/SubmitBar';

interface Props {
  editing: GroupRow | null;
  initialParentId?: number | null;
  groupOptions: GroupRow[];
  onSubmit: (values: any) => void;
  onCancel: () => void;
  submitting: boolean;
}

function collectDescendantIds(group: GroupRow | null): Set<number> {
  const ids = new Set<number>();
  const visit = (item?: GroupRow) => {
    if (!item) return;
    ids.add(item.id);
    item.children?.forEach(visit);
  };
  group?.children?.forEach(visit);
  return ids;
}

function flattenGroupOptions(
  groups: GroupRow[],
  disabledIds: Set<number>,
  depth = 0,
): Array<{ label: string; value: number; disabled?: boolean }> {
  return groups.flatMap((group) => [
    {
      label: `${'  '.repeat(depth)}${group.name}`,
      value: group.id,
      disabled: disabledIds.has(group.id),
    },
    ...flattenGroupOptions(group.children || [], disabledIds, depth + 1),
  ]);
}

export default function CmdbGroupForm({
  editing,
  initialParentId,
  groupOptions,
  onSubmit,
  onCancel,
  submitting,
}: Props) {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [labelSchemas, setLabelSchemas] = useState<LabelSchemaRow[]>([]);

  const loadLabelSchemas = useCallback(async () => {
    try {
      const result = await getLabelSchemaList({ status: 'enabled' });
      setLabelSchemas(result);
    } catch {
      setLabelSchemas([]);
    }
  }, []);

  useEffect(() => {
    void loadLabelSchemas();
  }, [loadLabelSchemas]);
  const disabledParentIds = useMemo(() => {
    const ids = collectDescendantIds(editing);
    if (editing) {
      ids.add(editing.id);
    }
    return ids;
  }, [editing]);
  const parentOptions = useMemo(
    () => flattenGroupOptions(groupOptions, disabledParentIds),
    [groupOptions, disabledParentIds],
  );
  const opOptions = [
    { value: 'eq', label: t('business.cmdb.group.condition.op.eq') },
    { value: 'neq', label: t('business.cmdb.group.condition.op.neq') },
    { value: 'in', label: t('business.cmdb.group.condition.op.in') },
    { value: 'notIn', label: t('business.cmdb.group.condition.op.notIn') },
  ];

  useEffect(() => {
    if (editing) {
      form.setFieldsValue({
        name: editing.name,
        parentId: editing.parentId || undefined,
        description: editing.description,
        operator: editing.conditions?.operator || 'AND',
        rules: editing.conditions?.rules?.length
          ? editing.conditions.rules
          : [{ key: '', op: 'eq', val: '' }],
      });
    } else {
      form.resetFields();
      form.setFieldsValue({
        parentId: initialParentId || undefined,
        operator: 'AND',
        rules: [{ key: '', op: 'eq', val: '' }],
      });
    }
  }, [editing, form, initialParentId]);

  const handleFinish = async () => {
    const values = await form.validate();
    const rules: ConditionRule[] = (values.rules || []).filter(
      (r: any) => r.key && r.val !== undefined && r.val !== '',
    );
    onSubmit({
      name: values.name,
      parentId: values.parentId || 0,
      description: values.description,
      conditions: {
        operator: values.operator,
        rules,
      },
    });
  };

  const handleRuleKeyChange = (index: number) => {
    const rules = form.getFieldValue('rules') || [];
    rules[index] = { ...rules[index], val: undefined };
    form.setFieldValue('rules', rules);
  };

  return (
    <Form form={form} layout="vertical" onSubmit={handleFinish}>
      <Form.Item
        label={t('business.cmdb.group.name')}
        field="name"
        rules={[{ required: true, message: t('common.required') }]}
      >
        <Input />
      </Form.Item>
      <Form.Item label={t('business.cmdb.group.description')} field="description">
        <Input />
      </Form.Item>
      <Form.Item label={t('business.cmdb.group.parent')} field="parentId">
        <Select allowClear placeholder={t('business.cmdb.group.noParent')}>
          {parentOptions.map((option) => (
            <Select.Option key={option.value} value={option.value} disabled={option.disabled}>
              {option.label}
            </Select.Option>
          ))}
        </Select>
      </Form.Item>
      <Form.Item
        label={t('business.cmdb.group.condition.operator')}
        field="operator"
      >
        <Radio.Group>
          <Radio value="AND">{t('business.cmdb.group.condition.operator.and')}</Radio>
          <Radio value="OR">{t('business.cmdb.group.condition.operator.or')}</Radio>
        </Radio.Group>
      </Form.Item>
      <Form.Item label={t('business.cmdb.group.conditions')}>
        <Form.List field="rules">
          {(fields, { add, remove }) => (
            <>
              {fields.map((field, index) => (
                <Space key={field.key} align="start" style={{ marginBottom: 8 }}>
                  <Form.Item field={`rules[${index}].key`} noStyle
                    rules={[{ required: true }]}>
                    <Select
                      allowCreate
                      showSearch
                      placeholder={t('business.cmdb.group.condition.key')}
                      style={{ width: 130 }}
                      onChange={() => handleRuleKeyChange(index)}
                    >
                      {labelSchemas.map((schema) => (
                        <Select.Option key={schema.key} value={schema.key}>
                          {schema.name} ({schema.key})
                        </Select.Option>
                      ))}
                    </Select>
                  </Form.Item>
                  <Form.Item field={`rules[${index}].op`} noStyle>
                    <Select style={{ width: 110 }}>
                      {opOptions.map((o) => (
                        <Select.Option key={o.value} value={o.value}>
                          {o.label}
                        </Select.Option>
                      ))}
                    </Select>
                  </Form.Item>
                  <Form.Item field={`rules[${index}].val`} noStyle
                    rules={[{ required: true }]}>
                    <Select
                      allowCreate={isFreeValueLabel(
                        labelSchemas,
                        form.getFieldValue(`rules[${index}].key`),
                      )}
                      showSearch
                      placeholder={t('business.cmdb.group.condition.val')}
                      style={{ width: 180 }}
                    >
                      {labelValueOptions(
                        labelSchemas,
                        form.getFieldValue(`rules[${index}].key`),
                      ).map((option) => (
                        <Select.Option key={option} value={option}>
                          {option}
                        </Select.Option>
                      ))}
                    </Select>
                  </Form.Item>
                  <Button
                    type="text"
                    status="danger"
                    icon={<IconDelete />}
                    onClick={() => remove(index)}
                    disabled={fields.length === 1}
                  />
                </Space>
              ))}
              <Button type="dashed" icon={<IconPlus />} onClick={() => add({ key: '', op: 'eq', val: '' })}>
                {t('business.cmdb.group.condition.addRule')}
              </Button>
            </>
          )}
        </Form.List>
      </Form.Item>
      <SubmitBar
        onCancel={onCancel}
        loading={submitting}
        submitText={editing ? t('common.save') : t('common.create')}
      />
    </Form>
  );
}
