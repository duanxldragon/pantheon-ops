import React from 'react';
import {
  Button,
  Card,
  Form,
  Grid,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
} from '@arco-design/web-react';

const { Row, Col } = Grid;
import type { FormInstance } from '@arco-design/web-react';
import { IconCode, IconDelete, IconEdit } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';

import { AppModal } from '../../../../components';
import type { GeneratorDatasource, UpsertGeneratorDatasourcePayload } from '../api';

const FormItem = Form.Item;

interface DatasourceManagerModalProps {
  visible: boolean;
  onClose: () => void;
  datasources: GeneratorDatasource[];
  editingDatasourceId: string | null;
  datasourceForm: FormInstance<UpsertGeneratorDatasourcePayload>;
  datasourceSaving: boolean;
  onEdit: (item: GeneratorDatasource) => void;
  onTest: (id: string) => void;
  onDelete: (id: string) => void;
  onReset: () => void;
  onSave: () => void;
}

const DatasourceManagerModal: React.FC<DatasourceManagerModalProps> = ({
  visible,
  onClose,
  datasources,
  editingDatasourceId,
  datasourceForm,
  datasourceSaving,
  onEdit,
  onTest,
  onDelete,
  onReset,
  onSave,
}) => {
  const { t } = useTranslation();

  return (
    <AppModal
      title={t('generator.datasource.manageTitle')}
      visible={visible}
      onCancel={onClose}
      footer={null}
      size="xl"
    >
      <Space direction="vertical" className="generator-wizard__full" size={16}>
        <Table
          pagination={false}
          rowKey="id"
          data={datasources}
          columns={[
            { title: t('generator.datasource.name'), dataIndex: 'name' },
            { title: t('generator.datasource.databaseName'), dataIndex: 'databaseName' },
            { title: t('generator.datasource.host'), dataIndex: 'host' },
            {
              title: t('generator.datasource.status'),
              dataIndex: 'status',
              render: (value: number) => (
                <Tag color={value === 1 ? 'green' : 'gray'}>
                  {value === 1 ? t('system.user.status.enabled') : t('system.user.status.disabled')}
                </Tag>
              ),
            },
            {
              title: t('common.action'),
              render: (_: unknown, record: GeneratorDatasource) => (
                <Space>
                  <Button size="mini" type="text" onClick={() => onEdit(record)}>
                    <IconEdit /> {t('common.edit')}
                  </Button>
                  <Button size="mini" type="text" onClick={() => void onTest(record.id)}>
                    <IconCode /> {t('generator.datasource.test')}
                  </Button>
                  <Popconfirm
                    title={t('generator.datasource.deleteConfirm')}
                    onOk={() => onDelete(record.id)}
                  >
                    <Button size="mini" type="text" status="danger">
                      <IconDelete /> {t('common.delete')}
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
          noDataElement={t('generator.datasource.empty')}
        />
        <Card
          size="small"
          title={
            editingDatasourceId
              ? t('generator.datasource.editTitle')
              : t('generator.datasource.createTitle')
          }
        >
          <Form
            form={datasourceForm}
            layout="vertical"
            onSubmit={() => {
              void onSave();
            }}
          >
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <FormItem
                  field="name"
                  label={t('generator.datasource.name')}
                  rules={[{ required: true, message: t('common.required') }]}
                >
                  <Input
                    placeholder={t('generator.datasource.namePlaceholder')}
                    onPressEnter={() => datasourceForm.submit()}
                  />
                </FormItem>
              </Col>
              <Col xs={24} md={12}>
                <FormItem
                  field="driver"
                  label={t('generator.datasource.driver')}
                  initialValue="mysql"
                >
                  <Select>
                    <Select.Option value="mysql">MySQL</Select.Option>
                  </Select>
                </FormItem>
              </Col>
              <Col xs={24} md={12}>
                <FormItem
                  field="host"
                  label={t('generator.datasource.host')}
                  rules={[{ required: true, message: t('common.required') }]}
                >
                  <Input
                    placeholder={t('generator.datasource.hostPlaceholder')}
                    onPressEnter={() => datasourceForm.submit()}
                  />
                </FormItem>
              </Col>
              <Col xs={24} md={12}>
                <FormItem
                  field="port"
                  label={t('generator.datasource.port')}
                  initialValue={3306}
                  rules={[{ required: true, message: t('common.required') }]}
                >
                  <InputNumber
                    placeholder={t('generator.datasource.portPlaceholder')}
                    className="generator-wizard__number-input"
                  />
                </FormItem>
              </Col>
              <Col xs={24} md={12}>
                <FormItem
                  field="databaseName"
                  label={t('generator.datasource.databaseName')}
                  rules={[{ required: true, message: t('common.required') }]}
                >
                  <Input
                    placeholder={t('generator.datasource.databasePlaceholder')}
                    onPressEnter={() => datasourceForm.submit()}
                  />
                </FormItem>
              </Col>
              <Col xs={24} md={12}>
                <FormItem
                  field="username"
                  label={t('generator.datasource.username')}
                  rules={[{ required: true, message: t('common.required') }]}
                >
                  <Input
                    placeholder={t('generator.datasource.usernamePlaceholder')}
                    onPressEnter={() => datasourceForm.submit()}
                  />
                </FormItem>
              </Col>
              <Col xs={24} md={12}>
                <FormItem
                  field="password"
                  label={t('generator.datasource.password')}
                  extra={
                    editingDatasourceId ? t('generator.datasource.passwordOptional') : undefined
                  }
                >
                  <Input.Password
                    placeholder={t('generator.datasource.passwordPlaceholder')}
                    onPressEnter={() => datasourceForm.submit()}
                  />
                </FormItem>
              </Col>
              <Col xs={24} md={12}>
                <FormItem field="status" label={t('generator.datasource.status')} initialValue={1}>
                  <Select>
                    <Select.Option value={1}>{t('system.user.status.enabled')}</Select.Option>
                    <Select.Option value={0}>{t('system.user.status.disabled')}</Select.Option>
                  </Select>
                </FormItem>
              </Col>
              <Col xs={24}>
                <FormItem field="remark" label={t('i18n.remark')}>
                  <Input.TextArea autoSize={{ minRows: 2, maxRows: 3 }} />
                </FormItem>
              </Col>
            </Row>
            <Space>
              <Button onClick={onReset}>{t('common.reset')}</Button>
              <Button type="primary" loading={datasourceSaving} onClick={() => void onSave()}>
                {editingDatasourceId ? t('common.save') : t('common.create')}
              </Button>
            </Space>
          </Form>
        </Card>
      </Space>
    </AppModal>
  );
};

export default DatasourceManagerModal;
