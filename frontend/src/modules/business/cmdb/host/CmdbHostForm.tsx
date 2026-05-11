import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, Input, InputNumber, Select, Button, Space } from '@arco-design/web-react';
import { IconPlus, IconDelete } from '@arco-design/web-react/icon';
import type { HostRow } from './api';
import { getLabelSchemaList, type LabelSchemaRow } from '../label/api';
import { isFreeValueLabel, labelValueOptions } from '../label/options';
import SubmitBar from '../../../../components/patterns/SubmitBar';

interface Props {
  editing: HostRow | null;
  onSubmit: (values: any) => void;
  onCancel: () => void;
  submitting: boolean;
}

export default function CmdbHostForm({ editing, onSubmit, onCancel, submitting }: Props) {
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

  useEffect(() => {
    if (editing) {
      form.setFieldsValue({
        hostname: editing.hostname,
        ip: editing.ip,
        sshPort: editing.sshPort || 22,
        os: editing.os,
        osVersion: editing.osVersion,
        cpuCores: editing.cpuCores,
        memoryGb: editing.memoryGb,
        diskGb: editing.diskGb,
        labels: editing.labelValues?.length
          ? editing.labelValues.map((l) => ({ key: l.key, val: l.val }))
          : [],
        owner: editing.owner,
        remark: editing.remark,
      });
    } else {
      form.resetFields();
      form.setFieldsValue({ sshPort: 22, os: 'linux', labels: [] });
    }
  }, [editing, form]);

  const handleFinish = async () => {
    const values = await form.validate();
    const labels = (values.labels || []).filter((l: any) => l.key && l.val);
    onSubmit({ ...values, labels: labels.length ? labels : undefined });
  };

  const handleLabelKeyChange = (index: number) => {
    const labels = form.getFieldValue('labels') || [];
    labels[index] = { ...labels[index], val: undefined };
    form.setFieldValue('labels', labels);
  };

  return (
    <Form form={form} layout="vertical" onSubmit={handleFinish}>
      <Form.Item
        label={t('business.cmdb.host.hostname')}
        field="hostname"
        rules={[{ required: true, message: t('common.required') }]}
      >
        <Input />
      </Form.Item>
      <Form.Item
        label={t('business.cmdb.host.ip')}
        field="ip"
        rules={[{ required: true, message: t('common.required') }]}
      >
        <Input placeholder="192.168.1.1" />
      </Form.Item>
      <Form.Item label={t('business.cmdb.host.sshPort')} field="sshPort">
        <InputNumber min={1} max={65535} />
      </Form.Item>
      <Form.Item
        label={t('business.cmdb.host.os')}
        field="os"
        rules={[{ required: true }]}
      >
        <Select>
          <Select.Option value="linux">{t('business.cmdb.host.os.linux')}</Select.Option>
          <Select.Option value="windows">{t('business.cmdb.host.os.windows')}</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item label={t('business.cmdb.host.osVersion')} field="osVersion">
        <Input />
      </Form.Item>
      <Form.Item label={t('business.cmdb.host.cpuCores')} field="cpuCores">
        <InputNumber min={0} />
      </Form.Item>
      <Form.Item label={t('business.cmdb.host.memoryGb')} field="memoryGb">
        <InputNumber min={0} step={0.5} />
      </Form.Item>
      <Form.Item label={t('business.cmdb.host.diskGb')} field="diskGb">
        <InputNumber min={0} />
      </Form.Item>
      <Form.Item label={t('business.cmdb.host.labels')}>
        <Form.List field="labels">
          {(fields, { add, remove }) => (
            <>
              {fields.map((field, index) => (
                <Space key={field.key} align="start" style={{ marginBottom: 8 }}>
                  <Form.Item field={`labels[${index}].key`} noStyle>
                    <Select
                      allowCreate
                      showSearch
                      placeholder={t('business.cmdb.label.key')}
                      style={{ width: 160 }}
                      onChange={() => handleLabelKeyChange(index)}
                    >
                      {labelSchemas.map((schema) => (
                        <Select.Option key={schema.key} value={schema.key}>
                          {schema.name} ({schema.key})
                        </Select.Option>
                      ))}
                    </Select>
                  </Form.Item>
                  <Form.Item field={`labels[${index}].val`} noStyle>
                    <Select
                      allowCreate={isFreeValueLabel(
                        labelSchemas,
                        form.getFieldValue(`labels[${index}].key`),
                      )}
                      showSearch
                      placeholder={t('business.cmdb.label.val')}
                      style={{ width: 200 }}
                    >
                      {labelValueOptions(
                        labelSchemas,
                        form.getFieldValue(`labels[${index}].key`),
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
                  />
                </Space>
              ))}
              <Button type="dashed" icon={<IconPlus />} onClick={() => add()}>
                {t('business.cmdb.label.add')}
              </Button>
            </>
          )}
        </Form.List>
      </Form.Item>
      <Form.Item label={t('business.cmdb.host.owner')} field="owner">
        <Input />
      </Form.Item>
      <Form.Item label={t('business.cmdb.host.remark')} field="remark">
        <Input.TextArea />
      </Form.Item>
      <SubmitBar
        onCancel={onCancel}
        loading={submitting}
        submitText={editing ? t('common.save') : t('common.create')}
      />
    </Form>
  );
}
