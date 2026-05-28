import React, { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Grid,
  Input,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconEdit, IconSearch } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import { usePermission } from '../../../hooks/usePermission';
import {
  getPermissionDataScopePolicies,
  updatePermissionDataScopePolicy,
  type PermissionDataScopeMode,
  type PermissionDataScopePolicy,
  type PermissionDataScopeQuery,
} from './api';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  FilterPanel,
  PageEmpty,
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  withTableColumnPriority,
} from '../../../components';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

interface LoadDataOptions {
  silent?: boolean;
}

interface DataScopeEditorFormValues {
  mode: PermissionDataScopeMode;
  deptIdsText?: string;
}

interface PermissionDataScopeTabProps {
  roleOptions: Array<{ label: string; value: string }>;
}

export const PermissionDataScopeTab: React.FC<PermissionDataScopeTabProps> = ({ roleOptions }) => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canEdit = isAdmin || hasPerm('system:permission:update');

  const [dataScopeRows, setDataScopeRows] = useState<PermissionDataScopePolicy[]>([]);
  const [dataScopeLoading, setDataScopeLoading] = useState(false);
  const [dataScopeError, setDataScopeError] = useState<unknown>(null);
  const [dataScopeQuery, setDataScopeQuery] = useState<PermissionDataScopeQuery>({});
  const [tablePagination, setTablePagination] = useState({ current: 1, pageSize: 10 });
  const [dataScopeSubmittingRoleKey, setDataScopeSubmittingRoleKey] = useState('');
  const [dataScopeEditingRow, setDataScopeEditingRow] = useState<PermissionDataScopePolicy | null>(
    null,
  );
  const [dataScopeModeDraft, setDataScopeModeDraft] = useState<PermissionDataScopeMode>('all');
  const [dataScopeForm] = Form.useForm<PermissionDataScopeQuery>();
  const [dataScopeEditorForm] = Form.useForm<DataScopeEditorFormValues>();

  const loadDataScopePolicies = useCallback(
    async (nextQuery: PermissionDataScopeQuery = dataScopeQuery, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setDataScopeLoading(true);
        setDataScopeError(null);
      }
      try {
        const result = await getPermissionDataScopePolicies(nextQuery);
        setDataScopeRows(result.items);
      } catch (requestError) {
        setDataScopeError(requestError);
      } finally {
        if (!silent) {
          setDataScopeLoading(false);
        }
      }
    },
    [dataScopeQuery],
  );

  useEffect(() => {
    const timer = globalThis.setTimeout(() => void loadDataScopePolicies(dataScopeQuery), 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadDataScopePolicies, dataScopeQuery]);

  useEffect(() => {
    setTablePagination((current) => {
      const totalPages = Math.max(1, Math.ceil(dataScopeRows.length / current.pageSize));
      if (current.current <= totalPages) {
        return current;
      }
      return {
        ...current,
        current: totalPages,
      };
    });
  }, [dataScopeRows]);

  useRefreshSubscription(
    ['system:permission:changed', 'system:role:changed', 'system:menu:changed'],
    (payload) => {
      if (payload.source === 'system/permission') {
        return;
      }
      void loadDataScopePolicies(dataScopeQuery);
    },
  );

  const searchDataScope = () => {
    const values = dataScopeForm.getFieldsValue();
    setTablePagination((current) => ({ ...current, current: 1 }));
    setDataScopeQuery({
      ...dataScopeQuery,
      ...values,
    });
  };

  const resetDataScope = () => {
    dataScopeForm.setFieldsValue({});
    setTablePagination({ current: 1, pageSize: 10 });
    setDataScopeQuery({});
  };

  const openDataScopeEditor = (row: PermissionDataScopePolicy) => {
    setDataScopeEditingRow(row);
    setDataScopeModeDraft(row.mode);
    dataScopeEditorForm.setFieldsValue({
      mode: row.mode,
      deptIdsText: row.deptIds.join(','),
    });
  };

  const closeDataScopeEditor = () => {
    setDataScopeEditingRow(null);
    setDataScopeModeDraft('all');
    dataScopeEditorForm.resetFields();
  };

  const saveDataScopePolicy = async () => {
    if (!dataScopeEditingRow) {
      return;
    }
    let values;
    try {
      values = await dataScopeEditorForm.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      throw error;
    }
    const mode = values.mode || 'all';
    const deptIds = (values.deptIdsText || '')
      .split(',')
      .map((item) => Number(item.trim()))
      .filter((item) => Number.isInteger(item) && item > 0);
    if (mode === 'custom' && deptIds.length === 0) {
      message.error(t('permission.data_scope.dept_required'));
      return;
    }

    setDataScopeSubmittingRoleKey(dataScopeEditingRow.roleKey);
    try {
      await updatePermissionDataScopePolicy(dataScopeEditingRow.roleKey, {
        mode,
        deptIds: mode === 'custom' ? deptIds : [],
      });
      message.success(t('common.updateSuccess'));
      publishRefresh('system:permission:changed', 'system/permission');
      await loadDataScopePolicies(dataScopeQuery, { silent: true });
    } finally {
      setDataScopeSubmittingRoleKey('');
      closeDataScopeEditor();
    }
  };

  const renderRequestErrorState = (requestError: unknown, onRetry: () => void) => {
    if (isNetworkRequestError(requestError)) {
      return <PageNetworkError timeout={isTimeoutRequestError(requestError)} onRetry={onRetry} />;
    }
    if (isServerRequestError(requestError)) {
      return <PageServerError onRetry={onRetry} />;
    }
    return <PageError onRetry={onRetry} />;
  };

  const dataScopeColumns: ColumnProps<PermissionDataScopePolicy>[] = [
    { title: t('system.role.roleName'), dataIndex: 'roleName', width: TABLE_COLUMN_WIDTH.name },
    withTableColumnPriority(
      { title: t('system.role.roleKey'), dataIndex: 'roleKey', width: TABLE_COLUMN_WIDTH.code },
      'medium',
    ),
    withTableColumnPriority(
      {
        title: t('system.role.status'),
        dataIndex: 'status',
        width: TABLE_COLUMN_WIDTH.status,
        render: (value: number) => (
          <Tag color={value === 1 ? 'green' : 'red'}>
            {value === 1 ? t('system.user.status.enabled') : t('system.user.status.disabled')}
          </Tag>
        ),
      },
      'medium',
    ),
    {
      title: t('system.permission.dataScope.mode'),
      dataIndex: 'mode',
      width: TABLE_COLUMN_WIDTH.tagGroup,
      render: (value: PermissionDataScopeMode) => (
        <Tag color="arcoblue">
          {t(
            `system.permission.dataScope.mode.${value === 'dept_and_children' ? 'deptAndChildren' : value}`,
          )}
        </Tag>
      ),
    },
    {
      title: t('system.permission.dataScope.customDeptIds'),
      dataIndex: 'deptIds',
      width: TABLE_COLUMN_WIDTH.keyPath,
      render: (_: unknown, row: PermissionDataScopePolicy) => (
        <Space wrap>
          {row.deptIds.length > 0 ? (
            row.deptIds.map((deptId) => <Tag key={`${row.roleKey}-${deptId}`}>{deptId}</Tag>)
          ) : (
            <Typography.Text type="secondary">{t('common.noData')}</Typography.Text>
          )}
        </Space>
      ),
    },
    withTableColumnPriority(
      {
        title: t('system.permission.dataScope.policyState'),
        dataIndex: 'policyExists',
        width: TABLE_COLUMN_WIDTH.status,
        render: (value: boolean) => (
          <Tag color={value ? 'green' : 'gray'}>
            {t(
              value
                ? 'system.permission.dataScope.explicit'
                : 'system.permission.dataScope.default',
            )}
          </Tag>
        ),
      },
      'low',
    ),
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.single,
      fixed: 'right',
      render: (_: unknown, row: PermissionDataScopePolicy) => (
        <Button
          type="text"
          size="small"
          icon={<IconEdit />}
          disabled={!canEdit}
          onClick={() => openDataScopeEditor(row)}
        >
          {t('common.edit')}
        </Button>
      ),
    },
  ];

  return (
    <>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <FilterPanel>
          <Form form={dataScopeForm} layout="vertical" onSubmit={() => searchDataScope()}>
            <Row gutter={16}>
              <Col span={8}>
                <FormItem label={t('system.permission.roleKey')} field="roleKey">
                  <Select allowClear options={roleOptions} />
                </FormItem>
              </Col>
              <Col span={6}>
                <FormItem label={t('system.role.status')} field="status">
                  <Select
                    allowClear
                    options={[
                      { label: t('system.user.status.enabled'), value: 1 },
                      { label: t('system.user.status.disabled'), value: 2 },
                    ]}
                  />
                </FormItem>
              </Col>
              <Col span={6}>
                <FormItem className="filter-panel__action-item">
                  <Space>
                    <Button type="primary" htmlType="submit" icon={<IconSearch />}>
                      {t('common.search')}
                    </Button>
                    <Button onClick={resetDataScope}>{t('common.reset')}</Button>
                  </Space>
                </FormItem>
              </Col>
            </Row>
          </Form>
        </FilterPanel>
        <Card className="page-panel system-list__table-card">
          {dataScopeLoading && dataScopeRows.length === 0 ? <PageLoading /> : null}
          {dataScopeError && dataScopeRows.length === 0
            ? renderRequestErrorState(dataScopeError, () => {
                void loadDataScopePolicies(dataScopeQuery);
              })
            : null}
          {!dataScopeLoading && !dataScopeError && dataScopeRows.length === 0 ? (
            <PageEmpty description={t('common.noData')} />
          ) : null}
          {!(dataScopeError && dataScopeRows.length === 0) && dataScopeRows.length > 0 ? (
            <AppTable<PermissionDataScopePolicy>
              className="system-list__table"
              rowKey="roleKey"
              data={dataScopeRows}
              columns={dataScopeColumns}
              loading={dataScopeLoading}
              scroll={{ x: 'max-content' }}
              pagination={buildStandardPagination(t, {
                current: tablePagination.current,
                pageSize: tablePagination.pageSize,
                total: dataScopeRows.length,
              })}
              onChange={(pagination) => {
                setTablePagination({
                  current: pagination.current || 1,
                  pageSize: pagination.pageSize || tablePagination.pageSize,
                });
              }}
              emptyText={t('common.noData')}
            />
          ) : null}
        </Card>
      </Space>

      <AppModal
        title={
          dataScopeEditingRow
            ? `${dataScopeEditingRow.roleName} · ${dataScopeEditingRow.roleKey}`
            : t('system.permission.dataScope.tab')
        }
        visible={Boolean(dataScopeEditingRow)}
        size="md"
        confirmLoading={dataScopeSubmittingRoleKey === dataScopeEditingRow?.roleKey}
        onOk={() => {
          void saveDataScopePolicy();
        }}
        onCancel={closeDataScopeEditor}
      >
        <Form form={dataScopeEditorForm} layout="vertical">
          <FormItem label={t('system.permission.dataScope.mode')} field="mode">
            <Select
              value={dataScopeModeDraft}
              options={[
                { label: t('system.permission.dataScope.mode.all'), value: 'all' },
                { label: t('system.permission.dataScope.mode.self'), value: 'self' },
                { label: t('system.permission.dataScope.mode.dept'), value: 'dept' },
                {
                  label: t('system.permission.dataScope.mode.deptAndChildren'),
                  value: 'dept_and_children',
                },
                { label: t('system.permission.dataScope.mode.custom'), value: 'custom' },
              ]}
              onChange={(value) => {
                const nextMode = (value as PermissionDataScopeMode) || 'all';
                setDataScopeModeDraft(nextMode);
                dataScopeEditorForm.setFieldValue('mode', nextMode);
              }}
            />
          </FormItem>
          <FormItem
            label={t('system.permission.dataScope.customDeptIds')}
            field="deptIdsText"
            extra={t('system.permission.dataScope.customDeptIds.placeholder')}
          >
            <Input.TextArea
              autoSize={{ minRows: 2, maxRows: 4 }}
              disabled={dataScopeModeDraft !== 'custom'}
              placeholder={t('system.permission.dataScope.customDeptIds.placeholder')}
            />
          </FormItem>
        </Form>
      </AppModal>
    </>
  );
};
