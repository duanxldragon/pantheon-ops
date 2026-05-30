import React, { useCallback, useMemo, useState } from 'react';
import {
  Button,
  Form,
  Grid,
  Input,
  Popconfirm,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import {
  IconDelete,
  IconDownload,
  IconEdit,
  IconPlus,
  IconSearch,
} from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import { message } from '../../../components/feedback/message';
import { showImportResult } from '../../../api/importExport';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import { publishRefresh } from '../../../core/refresh/refreshBus';
import { invalidateRouteWarmDataMany } from '../../../core/router/prefetch';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  FilterPanel,
  FormSection,
  ImportCsvButton,
  ListHeaderActions,
  PageEmpty,
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
  SubmitBar,
  TABLE_ACTION_COLUMN_WIDTH,
  TableBatchActionBar,
  PermissionAction,
  TABLE_COLUMN_WIDTH,
  withTableColumnPriority,
} from '../../../components';
import {
  batchDeleteDictTypes,
  batchUpdateDictTypeStatus,
  createDictType,
  deleteDictType,
  downloadDictTypeImportTemplate,
  exportDictTypes,
  importDictTypes,
  updateDictType,
  type DictTypePayload,
  type DictTypeQuery,
  type DictTypeRow,
} from './api';
import '../list-page.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;
const { Text } = Typography;

const emptyTypeQuery: DictTypeQuery = {
  dictCode: '',
  dictName: '',
  status: undefined,
};

const emptyTypeForm: DictTypePayload = {
  dictCode: '',
  dictName: '',
  module: 'system',
  status: 1,
  remark: '',
};

export interface TypeSummary {
  total: number;
  active: number;
  disabled: number;
  items: number;
}

interface DictTypeTabProps {
  typeRows: DictTypeRow[];
  typeLoading: boolean;
  typeError: unknown;
  typeQuery: DictTypeQuery;
  canCreate: boolean;
  canEdit: boolean;
  canDelete: boolean;
  canBatchUpdate: boolean;
  canBatchDelete: boolean;
  canExport: boolean;
  canImport: boolean;
  onQueryChange: (query: DictTypeQuery) => void;
  onReload: () => void;
  onSelectType: (row: DictTypeRow | null) => void;
  onSwitchToItemsTab: (row: DictTypeRow) => void;
}

const DictTypeTab: React.FC<DictTypeTabProps> = ({
  typeRows,
  typeLoading,
  typeError,
  typeQuery,
  canCreate,
  canEdit,
  canDelete,
  canBatchUpdate,
  canBatchDelete,
  canExport,
  canImport,
  onQueryChange,
  onReload,
  onSelectType,
  onSwitchToItemsTab,
}) => {
  const { t } = useTranslation();

  const [selectedTypeRowKeys, setSelectedTypeRowKeys] = useState<(string | number)[]>([]);
  const [typeTablePage, setTypeTablePage] = useState(1);
  const [typeTablePageSize, setTypeTablePageSize] = useState(10);
  const [typeVisible, setTypeVisible] = useState(false);
  const [editingType, setEditingType] = useState<DictTypeRow | null>(null);
  const [typeSubmitting, setTypeSubmitting] = useState(false);
  const [typeForm] = Form.useForm<DictTypePayload>();
  const [queryForm] = Form.useForm<DictTypeQuery>();

  const invalidateCaches = useCallback((dictCode?: string) => {
    const targets: Array<{ path: string; resourceKeys?: string[] }> = [
      { path: '/system/dict', resourceKeys: ['types:default'] },
    ];
    if (dictCode) {
      targets.push({ path: '/system/dict', resourceKeys: [`items:${dictCode}:default`] });
    }
    invalidateRouteWarmDataMany(targets);
  }, []);

  const handleSearch = () => {
    const values = queryForm.getFieldsValue();
    setTypeTablePage(1);
    onQueryChange({
      ...emptyTypeQuery,
      ...values,
    });
  };

  const handleReset = () => {
    queryForm.setFieldsValue(emptyTypeQuery);
    setTypeTablePage(1);
    onQueryChange(emptyTypeQuery);
  };

  const typeTableDisplayPage = React.useMemo(() => {
    const totalPages = Math.max(1, Math.ceil(typeRows.length / Math.max(1, typeTablePageSize)));
    return typeTablePage > totalPages ? totalPages : typeTablePage;
  }, [typeRows.length, typeTablePage, typeTablePageSize]);

  const openCreateType = () => {
    setEditingType(null);
    typeForm.setFieldsValue(emptyTypeForm);
    setTypeVisible(true);
  };

  const openEditType = useCallback(
    (row: DictTypeRow) => {
      setEditingType(row);
      typeForm.setFieldsValue({
        dictCode: row.dictCode,
        dictName: row.dictName,
        module: row.module,
        status: row.status,
        remark: row.remark,
      });
      setTypeVisible(true);
    },
    [typeForm],
  );

  const submitTypeForm = async () => {
    let values;
    try {
      values = await typeForm.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      return;
    }
    setTypeSubmitting(true);
    try {
      if (editingType) {
        await updateDictType(editingType.id, values);
        message.success(t('common.updateSuccess'));
      } else {
        await createDictType(values);
        message.success(t('common.createSuccess'));
      }
      invalidateCaches();
      publishRefresh('system:dict:changed', 'system/dict');
      setTypeVisible(false);
      onReload();
    } catch {
      message.error(t('common.actionFailed'));
    } finally {
      setTypeSubmitting(false);
    }
  };

  const removeType = useCallback(
    async (row: DictTypeRow) => {
      await deleteDictType(row.id);
      message.success(t('common.deleteSuccess'));
      invalidateCaches(row.dictCode);
      publishRefresh('system:dict:changed', 'system/dict');
      onReload();
    },
    [t, invalidateCaches, onReload],
  );

  const handleBatchTypeStatus = async (status: 1 | 2) => {
    if (selectedTypeRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const typeIds = selectedTypeRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchUpdateDictTypeStatus({ typeIds, status });
    message.success(t('system.dict.type.batchStatusSuccess', { count: result.updatedCount }));
    invalidateCaches();
    publishRefresh('system:dict:changed', 'system/dict');
    setSelectedTypeRowKeys([]);
    onReload();
  };

  const handleBatchTypeDelete = async () => {
    if (selectedTypeRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const ids = selectedTypeRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchDeleteDictTypes({ ids });
    const messageKey =
      result.failedCount > 0 ? 'common.batchDeletePartialSuccess' : 'common.batchDeleteSuccess';
    message[result.failedCount > 0 ? 'warning' : 'success'](
      t(messageKey, { deleted: result.deletedCount, failed: result.failedCount }),
    );
    invalidateCaches();
    publishRefresh('system:dict:changed', 'system/dict');
    setSelectedTypeRowKeys([]);
    onReload();
  };

  const handleExportTypes = async () => {
    await exportDictTypes(typeQuery);
  };

  const handleDownloadTypeTemplate = async () => {
    await downloadDictTypeImportTemplate();
  };

  const handleImportTypes = async (file: File) => {
    const result = await importDictTypes(file);
    showImportResult(result, t);
    if (result.applied) {
      invalidateCaches();
      publishRefresh('system:dict:changed', 'system/dict');
      onReload();
    }
  };

  const typeColumns: ColumnProps<DictTypeRow>[] = useMemo(
    () => [
      {
        title: t('system.dict.dictCode'),
        dataIndex: 'dictCode',
        width: TABLE_COLUMN_WIDTH.code,
        render: (value: string, row: DictTypeRow) => (
          <Button type="text" style={{ padding: 0 }} onClick={() => onSelectType(row)}>
            {value}
          </Button>
        ),
      },
      {
        title: t('system.dict.dictName'),
        dataIndex: 'dictName',
        width: TABLE_COLUMN_WIDTH.name,
        render: (value: string) => <Text ellipsis={{ showTooltip: true }}>{t(value, value)}</Text>,
      },
      withTableColumnPriority(
        {
          title: t('system.dict.module'),
          dataIndex: 'module',
          width: TABLE_COLUMN_WIDTH.code,
          ellipsis: true,
        },
        'medium',
      ),
      withTableColumnPriority(
      {
        title: t('system.dict.item'),
        dataIndex: 'itemCount',
        width: TABLE_COLUMN_WIDTH.count,
        render: (_: unknown, row: DictTypeRow) => (
          <Text>{`${row.activeItemCount}/${row.itemCount}`}</Text>
        ),
        },
        'low',
      ),
      {
        title: t('system.dict.status'),
        dataIndex: 'status',
        width: TABLE_COLUMN_WIDTH.status,
        render: (value: number) => (
          <Tag color={value === 1 ? 'green' : 'red'}>
            {value === 1 ? t('system.user.status.enabled') : t('system.user.status.disabled')}
          </Tag>
        ),
      },
      {
        title: t('common.action'),
        width: TABLE_ACTION_COLUMN_WIDTH.wide,
        fixed: 'right',
        render: (_: unknown, row: DictTypeRow) => (
          <Space size={4} className="system-list__actions dict-page__row-actions">
            <Button size="small" type="text" onClick={() => onSwitchToItemsTab(row)}>
              {t('system.dict.item')}
            </Button>
            <Button
              size="small"
              icon={<IconEdit />}
              onClick={() => openEditType(row)}
              disabled={!canEdit}
            >
              {t('common.edit')}
            </Button>
            <Popconfirm
              title={t('common.deleteConfirm')}
              onOk={() => removeType(row)}
              disabled={!canDelete}
            >
              <Button size="small" status="danger" icon={<IconDelete />} disabled={!canDelete}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [t, onSelectType, onSwitchToItemsTab, canEdit, canDelete, openEditType, removeType],
  );

  const renderRequestErrorState = useCallback((requestError: unknown, onRetry: () => void) => {
    if (isNetworkRequestError(requestError)) {
      return <PageNetworkError timeout={isTimeoutRequestError(requestError)} onRetry={onRetry} />;
    }
    if (isServerRequestError(requestError)) {
      return <PageServerError onRetry={onRetry} />;
    }
    return <PageError onRetry={onRetry} />;
  }, []);

  const typeBatchActionDisabled = !canBatchUpdate || selectedTypeRowKeys.length === 0;
  const typeBatchDeleteDisabled = !canBatchDelete || selectedTypeRowKeys.length === 0;

  return (
    <>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <FilterPanel>
          <Form form={queryForm} layout="vertical" onSubmit={() => handleSearch()}>
            <Row gutter={16}>
              <Col xs={24} md={12} lg={8}>
                <FormItem label={t('system.dict.dictCode')} field="dictCode">
                  <Input onPressEnter={() => queryForm.submit()} />
                </FormItem>
              </Col>
              <Col xs={24} md={12} lg={8}>
                <FormItem label={t('system.dict.dictName')} field="dictName">
                  <Input onPressEnter={() => queryForm.submit()} />
                </FormItem>
              </Col>
              <Col xs={24} md={12} lg={4}>
                <FormItem label={t('system.dict.status')} field="status">
                  <Select
                    allowClear
                    options={[
                      { label: t('system.user.status.enabled'), value: 1 },
                      { label: t('system.user.status.disabled'), value: 2 },
                    ]}
                  />
                </FormItem>
              </Col>
              <Col xs={24} md={12} lg={4}>
                <FormItem className="filter-panel__action-item">
                  <Space>
                    <Button type="primary" htmlType="submit" icon={<IconSearch />}>
                      {t('common.search')}
                    </Button>
                    <Button onClick={handleReset}>{t('common.reset')}</Button>
                  </Space>
                </FormItem>
              </Col>
            </Row>
          </Form>
        </FilterPanel>
        <TableBatchActionBar
          selectedCount={selectedTypeRowKeys.length}
          selectedText={t('common.selectedCount', { count: selectedTypeRowKeys.length })}
          clearText={t('common.clearSelection')}
          clearSuccessText={t('common.clearSelectionSuccess')}
          onClear={() => setSelectedTypeRowKeys([])}
          prefixActions={
            <ListHeaderActions
              className="dict-page__actions"
              utility={
                <>
                  <Button
                    icon={<IconDownload />}
                    onClick={() => {
                      void handleExportTypes();
                    }}
                    disabled={!canExport}
                  >
                    {t('common.export')}
                  </Button>
                  <Button
                    onClick={() => {
                      void handleDownloadTypeTemplate();
                    }}
                    disabled={!canImport}
                  >
                    {t('common.downloadTemplate')}
                  </Button>
                  <ImportCsvButton
                    disabled={!canImport}
                    onSelect={(file) => {
                      void handleImportTypes(file);
                    }}
                  >
                    {t('common.import')}
                  </ImportCsvButton>
                </>
              }
              primary={
                <Button
                  type="primary"
                  icon={<IconPlus />}
                  onClick={openCreateType}
                  disabled={!canCreate}
                >
                  {t('common.add')}
                </Button>
              }
            />
          }
          hint={
            !canBatchUpdate || !canBatchDelete ? t('common.batchActionPermissionHint') : undefined
          }
          actions={
            <>
              <PermissionAction allowed={canBatchUpdate} tooltip={t('common.noPermissionAction')}>
                <Popconfirm
                  title={t('system.dict.type.batchEnableConfirm')}
                  onOk={() => {
                    void handleBatchTypeStatus(1);
                  }}
                  disabled={typeBatchActionDisabled}
                >
                  <Button disabled={typeBatchActionDisabled}>{t('system.dict.batchEnable')}</Button>
                </Popconfirm>
              </PermissionAction>
              <PermissionAction allowed={canBatchUpdate} tooltip={t('common.noPermissionAction')}>
                <Popconfirm
                  title={t('system.dict.type.batchDisableConfirm')}
                  onOk={() => {
                    void handleBatchTypeStatus(2);
                  }}
                  disabled={typeBatchActionDisabled}
                >
                  <Button
                    status={typeBatchActionDisabled ? undefined : 'warning'}
                    disabled={typeBatchActionDisabled}
                  >
                    {t('system.dict.batchDisable')}
                  </Button>
                </Popconfirm>
              </PermissionAction>
              <PermissionAction allowed={canBatchDelete} tooltip={t('common.noPermissionAction')}>
                <Popconfirm
                  title={t('system.dict.type.batchDeleteConfirm')}
                  onOk={() => {
                    void handleBatchTypeDelete();
                  }}
                  disabled={typeBatchDeleteDisabled}
                >
                  <Button
                    status="danger"
                    icon={<IconDelete />}
                    disabled={typeBatchDeleteDisabled}
                  >
                    {t('common.deleteSelected')}
                  </Button>
                </Popconfirm>
              </PermissionAction>
            </>
          }
        />

        {typeLoading && typeRows.length === 0 ? <PageLoading /> : null}
        {typeError && typeRows.length === 0
          ? renderRequestErrorState(typeError, () => {
              onReload();
            })
          : null}
        {!typeLoading && !typeError && typeRows.length === 0 ? (
          <PageEmpty description={t('system.dict.typeEmpty')} />
        ) : null}
        {!typeLoading && !(typeError && typeRows.length === 0) && typeRows.length > 0 ? (
          <AppTable<DictTypeRow>
            className="system-list__table"
            rowKey="id"
            columns={typeColumns}
            data={typeRows}
            loading={typeLoading}
            rowSelection={{
              selectedRowKeys: selectedTypeRowKeys,
              onChange: (keys) => setSelectedTypeRowKeys(keys),
            }}
            emptyText={t('system.dict.typeEmpty')}
            pagination={buildStandardPagination(t, {
              total: typeRows.length,
              current: typeTableDisplayPage,
              pageSize: typeTablePageSize,
              onChange: (page, pageSize) => {
                setTypeTablePage(page);
                setTypeTablePageSize(pageSize);
              },
            })}
            scroll={{ x: 'max-content' }}
          />
        ) : null}
      </Space>

      <AppModal
        title={editingType ? t('system.dict.typeEdit') : t('system.dict.typeCreate')}
        visible={typeVisible}
        size="md"
        onCancel={() => setTypeVisible(false)}
        footer={
          <SubmitBar
            onCancel={() => setTypeVisible(false)}
            onSubmit={() => {
              void submitTypeForm();
            }}
            loading={typeSubmitting}
            submitText={editingType ? t('common.save') : t('common.add')}
          />
        }
        unmountOnExit
      >
        <Form
          form={typeForm}
          layout="vertical"
          onSubmit={() => {
            void submitTypeForm();
          }}
        >
          <Space direction="vertical" size={20} className="dialog-form-stack">
            <FormSection title={t('common.basicInfo')}>
              <FormItem
                label={t('system.dict.dictCode')}
                field="dictCode"
                rules={[{ required: true, message: t('system.dict.dictCodeRequired') }]}
              >
                <Input onPressEnter={() => typeForm.submit()} />
              </FormItem>
              <FormItem
                label={t('system.dict.dictName')}
                field="dictName"
                rules={[{ required: true, message: t('system.dict.dictNameRequired') }]}
              >
                <Input onPressEnter={() => typeForm.submit()} />
              </FormItem>
              <FormItem label={t('system.dict.module')} field="module">
                <Input onPressEnter={() => typeForm.submit()} />
              </FormItem>
              <FormItem label={t('system.dict.status')} field="status">
                <Select
                  options={[
                    { label: t('system.user.status.enabled'), value: 1 },
                    { label: t('system.user.status.disabled'), value: 2 },
                  ]}
                />
              </FormItem>
              <FormItem label={t('system.dict.remark')} field="remark">
                <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
              </FormItem>
            </FormSection>
          </Space>
        </Form>
      </AppModal>
    </>
  );
};

export default DictTypeTab;
