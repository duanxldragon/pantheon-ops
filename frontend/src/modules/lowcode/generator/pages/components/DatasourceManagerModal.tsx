import React from 'react';
import {
  Button,
  Card,
  Grid,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Space,
  Table,
  Tag,
} from '@arco-design/web-react';
const { Row, Col } = Grid;
import { IconEdit } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import type {
  GeneratorDatasource,
  UpsertGeneratorDatasourcePayload,
} from '../../api';

const FormItem = Form.Item;

export interface DatasourceManagerProps {
  visible?: boolean;
  editingId: string | null;
  saving: boolean;
  form: ReturnType<typeof Form.useForm<UpsertGeneratorDatasourcePayload>>[0];
  items: GeneratorDatasource[];
  onClose?: () => void;
  onEditItem: (item: GeneratorDatasource) => void;
  onDeleteItem: (id: string) => void;
  onTestItem: (id: string) => void;
  onSave: () => Promise<void>;
  onReset: () => void;
}

const DatasourceManagerModal: React.FC<DatasourceManagerProps> = ({
  editingId,
  saving,
  form,
  items,
  onEditItem,
  onDeleteItem,
  onTestItem,
  onSave,
  onReset,
}) => {
  const { t } = useTranslation();

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Table
        pagination={false}
        rowKey="id"
        data={items.filter((item) => !item.isCurrent)}
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
                <Button size="mini" type="text" onClick={() => onEditItem(record)}>
                  <IconEdit /> {t('common.edit')}
                </Button>
                <Button size="mini" type="text" onClick={() => onTestItem(record.id)}>
                  {t('generator.datasource.test')}
                </Button>
                <Popconfirm
                  title={t('generator.datasource.deleteConfirm')}
                  onOk={() => onDeleteItem(record.id)}
                >
                  <Button size="mini" type="text" status="danger">
                    {t('common.delete')}
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
          editingId
            ? t('generator.datasource.editTitle')
            : t('generator.datasource.createTitle')
        }
      >
        <Form form={form} layout="vertical" onSubmit={onSave}>
          <Row gutter={16}>
            <Col xs={24} md={12}>
              <FormItem
                field="name"
                label={t('generator.datasource.name')}
                rules={[{ required: true, message: t('common.required') }]}
              >
                <Input placeholder={t('generator.datasource.namePlaceholder')} />
              </FormItem>
            </Col>
            <Col xs={24} md={12}>
              <FormItem field="driver" label={t('generator.datasource.driver')} initialValue="mysql">
                <Input disabled />
              </FormItem>
            </Col>
            <Col xs={24} md={12}>
              <FormItem
                field="host"
                label={t('generator.datasource.host')}
                rules={[{ required: true, message: t('common.required') }]}
              >
                <Input placeholder={t('generator.datasource.hostPlaceholder')} />
              </FormItem>
            </Col>
            <Col xs={24} md={12}>
              <FormItem
                field="port"
                label={t('generator.datasource.port')}
                initialValue={3306}
                rules={[{ required: true, message: t('common.required') }]}
              >
                <InputNumber placeholder={t('generator.datasource.portPlaceholder')} />
              </FormItem>
            </Col>
            <Col xs={24} md={12}>
              <FormItem
                field="databaseName"
                label={t('generator.datasource.databaseName')}
                rules={[{ required: true, message: t('common.required') }]}
              >
                <Input placeholder={t('generator.datasource.databasePlaceholder')} />
              </FormItem>
            </Col>
            <Col xs={24} md={12}>
              <FormItem
                field="username"
                label={t('generator.datasource.username')}
                rules={[{ required: true, message: t('common.required') }]}
              >
                <Input placeholder={t('generator.datasource.usernamePlaceholder')} />
              </FormItem>
            </Col>
            <Col xs={24} md={12}>
              <FormItem
                field="password"
                label={t('generator.datasource.password')}
                extra={editingId ? t('generator.datasource.passwordOptional') : undefined}
              >
                <Input.Password placeholder={t('generator.datasource.passwordPlaceholder')} />
              </FormItem>
            </Col>
            <Col xs={24} md={12}>
              <FormItem field="status" label={t('generator.datasource.status')} initialValue={1}>
                <Input disabled />
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
            <Button type="primary" loading={saving} onClick={onSave}>
              {editingId ? t('common.save') : t('common.create')}
            </Button>
          </Space>
        </Form>
      </Card>
    </Space>
  );
};

export default DatasourceManagerModal;
