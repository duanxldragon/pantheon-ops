import React, { useCallback, useEffect, useMemo, useState } from 'react';
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
  Tag,
  Typography,
} from '@arco-design/web-react';
import {
  IconCaretDown,
  IconCaretUp,
  IconDelete,
  IconDownload,
  IconEdit,
  IconPlus,
  IconRefresh,
  IconSearch,
} from '@arco-design/web-react/icon';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import { message } from '../../../components/feedback/message';
import { showImportResult } from '../../../api/importExport';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import { formatDateTime } from '../../../core/format/dateTime';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import { invalidateRouteWarmDataMany, resolveRouteWarmData } from '../../../core/router/prefetch';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../../components/table/crossPageSelection';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  FilterPanel,
  FormSection,
  ImportCsvButton,
  ListHeaderActions,
  PageEmpty,
  PageLoading,
  PageRequestError,
  SubmitBar,
  TABLE_ACTION_COLUMN_WIDTH,
  TableBatchActionBar,
  PermissionAction,
  TABLE_COLUMN_WIDTH,
  withTableColumnPriority,
} from '../../../components';
import {
  analyzeDictUsage,
  batchDeleteDictItems,
  batchUpdateDictItemStatus,
  createDictItem,
  deleteDictItem,
  downloadDictItemImportTemplate,
  exportDictItems,
  getDictItemList,
  importDictItems,
  refreshDictCache,
  reorderDictItem,
  updateDictItem,
  type DictItemPayload,
  type DictItemQuery,
  type DictItemRow,
  type DictTypeRow,
  type DictUsageAnalysisResp,
} from './api';
import '../components/shared/list-page.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;
const { Text } = Typography;

const emptyItemForm: DictItemPayload = {
  dictCode: '',
  itemLabelKey: '',
  itemValue: '',
  itemColor: '',
  sort: 0,
  status: 1,
  remark: '',
};

const emptyItemQuery: Omit<DictItemQuery, 'dictCode'> = {
  keyword: '',
  status: undefined,
  page: 1,
  pageSize: 10,
};

function isDefaultDictItemQuery(query: Omit<DictItemQuery, 'dictCode'>) {
  return (
    !query.keyword &&
    query.status === undefined &&
    (query.page ?? 1) === 1 &&
    (query.pageSize ?? 10) === 10
  );
}

interface DictItemTabProps {
  selectedType: DictTypeRow | null;
  typeRows: DictTypeRow[];
  canCreate: boolean;
  canEdit: boolean;
  canDelete: boolean;
  canBatchUpdate: boolean;
  canBatchDelete: boolean;
  canRefresh: boolean;
  canExport: boolean;
  canImport: boolean;
  onSelectType: (row: DictTypeRow | null) => void;
  onReloadTypes: () => void;
}

const DictItemTab: React.FC<DictItemTabProps> = ({
  selectedType,
  typeRows,
  canCreate,
  canEdit,
  canDelete,
  canBatchUpdate,
  canBatchDelete,
  canRefresh,
  canExport,
  canImport,
  onSelectType,
  onReloadTypes,
}) => {
  const { t } = useTranslation();

  const [itemRows, setItemRows] = useState<DictItemRow[]>([]);
  const [itemTotal, setItemTotal] = useState(0);
  const [itemLoading, setItemLoading] = useState(false);
  const [itemError, setItemError] = useState<unknown>(null);
  const [itemQuery, setItemQuery] = useState<Omit<DictItemQuery, 'dictCode'>>(emptyItemQuery);
  const [selectedItemRowKeys, setSelectedItemRowKeys] = useState<(string | number)[]>([]);
  const [itemVisible, setItemVisible] = useState(false);
  const [editingItem, setEditingItem] = useState<DictItemRow | null>(null);
  const [itemSubmitting, setItemSubmitting] = useState(false);
  const [usageVisible, setUsageVisible] = useState(false);
  const [usageLoading, setUsageLoading] = useState(false);
  const [usageAnalysis, setUsageAnalysis] = useState<DictUsageAnalysisResp | null>(null);
  const [itemForm] = Form.useForm<DictItemPayload>();
  const [itemQueryForm] = Form.useForm<Omit<DictItemQuery, 'dictCode'>>();

  const invalidateCaches = useCallback((dictCode?: string) => {
    const targets: Array<{ path: string; resourceKeys?: string[] }> = [
      { path: '/system/dict', resourceKeys: ['types:default'] },
    ];
    if (dictCode) {
      targets.push({ path: '/system/dict', resourceKeys: [`items:${dictCode}:default`] });
    }
    invalidateRouteWarmDataMany(targets);
  }, []);

  const loadItems = useCallback(
    async (nextQuery: Omit<DictItemQuery, 'dictCode'> = itemQuery, dictCode?: string) => {
      const currentCode = dictCode || selectedType?.dictCode;
      if (!currentCode) {
        setItemRows([]);
        setItemTotal(0);
        setItemError(null);
        return;
      }
      setItemLoading(true);
      setItemError(null);
      try {
        const resp = isDefaultDictItemQuery(nextQuery)
          ? await resolveRouteWarmData('/system/dict', `items:${currentCode}:default`, () =>
              getDictItemList({ dictCode: currentCode, ...nextQuery }),
            )
          : await getDictItemList({ dictCode: currentCode, ...nextQuery });
        setItemRows(resp.items);
        setItemTotal(resp.total);
      } catch (requestError) {
        setItemError(requestError);
      } finally {
        setItemLoading(false);
      }
    },
    [itemQuery, selectedType?.dictCode],
  );

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      void loadItems(itemQuery, selectedType?.dictCode);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [itemQuery, loadItems, selectedType?.dictCode]);

  useRefreshSubscription('system:dict:changed', (payload) => {
    if (payload.source === 'system/dict') {
      return;
    }
    void loadItems(itemQuery, selectedType?.dictCode);
    onReloadTypes();
  });

  const selectedTypeTitle = useMemo(
    () =>
      selectedType ? t(selectedType.dictName, selectedType.dictCode) : t('system.dict.item.empty'),
    [selectedType, t],
  );

  const selectedTypeOptions = useMemo(
    () =>
      typeRows.map((item) => ({
        label: `${t(item.dictName, item.dictCode)} (${item.dictCode})`,
        value: item.id,
      })),
    [t, typeRows],
  );

  const handleItemSearch = () => {
    const values = itemQueryForm.getFieldsValue();
    setSelectedItemRowKeys([]);
    setItemQuery({
      ...emptyItemQuery,
      ...values,
      page: 1,
    });
  };

  const handleItemReset = () => {
    itemQueryForm.setFieldsValue(emptyItemQuery);
    setSelectedItemRowKeys([]);
    setItemQuery(emptyItemQuery);
  };

  const handleSelectedTypeChange = (value?: string | number) => {
    const nextType = typeRows.find((item) => item.id === Number(value)) || null;
    setSelectedItemRowKeys([]);
    onSelectType(nextType);
  };

  const openCreateItem = () => {
    if (!selectedType) {
      return;
    }
    setEditingItem(null);
    itemForm.setFieldsValue({
      ...emptyItemForm,
      dictCode: selectedType.dictCode,
    });
    setItemVisible(true);
  };

  const openEditItem = useCallback(
    (row: DictItemRow) => {
      setEditingItem(row);
      itemForm.setFieldsValue({
        dictCode: row.dictCode,
        itemLabelKey: row.itemLabelKey,
        itemValue: row.itemValue,
        itemColor: row.itemColor,
        sort: row.sort,
        status: row.status,
        remark: row.remark,
      });
      setItemVisible(true);
    },
    [itemForm],
  );

  const submitItemForm = async () => {
    let values;
    try {
      values = await itemForm.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      throw error;
    }
    setItemSubmitting(true);
    try {
      if (editingItem) {
        await updateDictItem(editingItem.id, values);
        message.success(t('common.updateSuccess'));
      } else {
        await createDictItem(values);
        message.success(t('common.createSuccess'));
      }
      invalidateCaches(values.dictCode);
      publishRefresh('system:dict:changed', 'system/dict');
      setItemVisible(false);
      await loadItems(itemQuery, values.dictCode);
      onReloadTypes();
    } finally {
      setItemSubmitting(false);
    }
  };

  const removeItem = useCallback(
    async (row: DictItemRow) => {
      await deleteDictItem(row.id);
      message.success(t('common.deleteSuccess'));
      invalidateCaches(row.dictCode);
      publishRefresh('system:dict:changed', 'system/dict');
      await loadItems(itemQuery, row.dictCode);
      onReloadTypes();
    },
    [t, invalidateCaches, loadItems, itemQuery, onReloadTypes],
  );

  const handleRefreshCache = async () => {
    const codes = selectedType ? [selectedType.dictCode] : [];
    await refreshDictCache({ codes });
    message.success(t('system.dict.refreshSuccess'));
    invalidateCaches(selectedType?.dictCode);
    publishRefresh('system:dict:changed', 'system/dict');
    await loadItems(itemQuery, selectedType?.dictCode);
  };

  const handleExportItems = async () => {
    if (!selectedType) {
      return;
    }
    await exportDictItems({
      dictCode: selectedType.dictCode,
      keyword: itemQuery.keyword,
      status: itemQuery.status,
    });
  };

  const handleDownloadItemTemplate = async () => {
    await downloadDictItemImportTemplate();
  };

  const handleImportItems = async (file: File) => {
    const result = await importDictItems(file);
    showImportResult(result, t);
    if (result.applied) {
      invalidateCaches(selectedType?.dictCode);
      publishRefresh('system:dict:changed', 'system/dict');
      await loadItems(itemQuery, selectedType?.dictCode);
      onReloadTypes();
    }
  };

  const handleBatchItemStatus = async (status: 1 | 2) => {
    if (selectedItemRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const itemIds = selectedItemRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchUpdateDictItemStatus({ itemIds, status });
    message.success(t('system.dict.item.batchStatusSuccess', { count: result.updatedCount }));
    invalidateCaches(selectedType?.dictCode);
    publishRefresh('system:dict:changed', 'system/dict');
    setSelectedItemRowKeys([]);
    await loadItems(itemQuery, selectedType?.dictCode);
    onReloadTypes();
  };

  const handleBatchItemDelete = async () => {
    if (selectedItemRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const ids = selectedItemRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchDeleteDictItems({ ids });
    const messageKey =
      result.failedCount > 0 ? 'common.batchDeletePartialSuccess' : 'common.batchDeleteSuccess';
    message[result.failedCount > 0 ? 'warning' : 'success'](
      t(messageKey, { deleted: result.deletedCount, failed: result.failedCount }),
    );
    invalidateCaches(selectedType?.dictCode);
    publishRefresh('system:dict:changed', 'system/dict');
    setSelectedItemRowKeys([]);
    await loadItems(itemQuery, selectedType?.dictCode);
    onReloadTypes();
  };

  const handleReorderItem = useCallback(
    async (row: DictItemRow, direction: 'up' | 'down') => {
      await reorderDictItem(row.id, direction);
      invalidateCaches(row.dictCode);
      publishRefresh('system:dict:changed', 'system/dict');
      await loadItems(itemQuery, row.dictCode);
      onReloadTypes();
    },
    [invalidateCaches, loadItems, itemQuery, onReloadTypes],
  );

  const handleOpenUsageAnalysis = async () => {
    if (!selectedType?.dictCode) {
      return;
    }
    setUsageVisible(true);
    setUsageLoading(true);
    try {
      const result = await analyzeDictUsage(selectedType.dictCode);
      setUsageAnalysis(result);
    } catch {
      setUsageAnalysis(null);
      message.error(t('system.dict.usage.error'));
    } finally {
      setUsageLoading(false);
    }
  };

  const handleItemTableChange: TableProps<DictItemRow>['onChange'] = (pagination) => {
    setItemQuery((prev) => ({
      ...prev,
      page: pagination.current || prev.page || emptyItemQuery.page,
      pageSize: pagination.pageSize || prev.pageSize || emptyItemQuery.pageSize,
    }));
  };

  const itemColumns: ColumnProps<DictItemRow>[] = useMemo(
    () => [
      {
        title: t('system.dict.itemLabelKey'),
        dataIndex: 'itemLabelKey',
        width: TABLE_COLUMN_WIDTH.treeLabel,
        render: (value: string) => (
          <Space direction="vertical" size={2}>
            <Text ellipsis={{ showTooltip: true }}>{t(value, value)}</Text>
            <Text type="secondary" className="text-sm" ellipsis={{ showTooltip: true }}>
              {value}
            </Text>
          </Space>
        ),
      },
      {
        title: t('system.dict.itemValue'),
        dataIndex: 'itemValue',
        width: TABLE_COLUMN_WIDTH.name,
        ellipsis: true,
      },
      withTableColumnPriority(
        {
          title: t('system.dict.itemColor'),
          dataIndex: 'itemColor',
          width: TABLE_COLUMN_WIDTH.status,
          render: (value: string) => (value ? <Tag color={value}>{value}</Tag> : '-'),
        },
        'medium',
      ),
      withTableColumnPriority(
        { title: t('system.dict.sort'), dataIndex: 'sort', width: TABLE_COLUMN_WIDTH.count },
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
        render: (_: unknown, row: DictItemRow) => (
          <Space
            size={4}
            className="system-list__actions dict-page__row-actions dict-page__row-actions--items"
          >
            <Button
              size="small"
              type="text"
              icon={<IconCaretUp />}
              onClick={() => {
                void handleReorderItem(row, 'up');
              }}
              disabled={!canEdit}
            />
            <Button
              size="small"
              type="text"
              icon={<IconCaretDown />}
              onClick={() => {
                void handleReorderItem(row, 'down');
              }}
              disabled={!canEdit}
            />
            <Button
              size="small"
              icon={<IconEdit />}
              onClick={() => openEditItem(row)}
              disabled={!canEdit}
            >
              {t('common.edit')}
            </Button>
            <Popconfirm
              title={t('common.deleteConfirm')}
              onOk={() => removeItem(row)}
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
    [t, canEdit, canDelete, handleReorderItem, openEditItem, removeItem],
  );

  const itemBatchActionDisabled = !canBatchUpdate || selectedItemRowKeys.length === 0;
  const itemBatchDeleteDisabled = !canBatchDelete || selectedItemRowKeys.length === 0;
  const visibleSelectedItemRowKeys = useMemo(
    () =>
      getVisibleSelectedRowKeys(
        selectedItemRowKeys,
        itemRows.map((item) => item.id),
      ),
    [itemRows, selectedItemRowKeys],
  );

  return (
    <>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        {selectedType ? (
          <Card className="dict-workbench__context-card">
            <div className="dict-workbench__context-head">
              <div className="dict-workbench__context-copy">
                <div className="dict-workbench__context-title">{selectedTypeTitle}</div>
                <div className="dict-workbench__context-subtitle">{selectedType.dictCode}</div>
              </div>
              <Space wrap>
                <Tag color="arcoblue">{selectedType.module || 'system'}</Tag>
                <Tag color={selectedType.status === 1 ? 'green' : 'red'}>
                  {selectedType.status === 1
                    ? t('system.user.status.enabled')
                    : t('system.user.status.disabled')}
                </Tag>
              </Space>
            </div>
            <div className="dict-workbench__context-metrics">
              <span>
                {t('system.dict.item')}: {selectedType.itemCount || 0}
              </span>
              <span>
                {t('system.user.status.enabled')}: {selectedType.activeItemCount || 0}
              </span>
              <span>
                {t('system.user.status.disabled')}: {selectedType.disabledItemCount || 0}
              </span>
              <span>
                {t('common.search')}: {itemTotal}
              </span>
              <span>
                {t('i18n.updatedAt')}: {formatDateTime(selectedType.lastItemUpdatedAt)}
              </span>
            </div>
          </Card>
        ) : null}

        <FilterPanel>
          <Form form={itemQueryForm} layout="vertical" onSubmit={() => handleItemSearch()}>
            <Row gutter={16}>
              <Col xs={24} lg={8}>
                <FormItem label={t('system.dict.type')}>
                  <Select
                    allowClear={false}
                    placeholder={t('system.dict.type')}
                    value={selectedType?.id}
                    options={selectedTypeOptions}
                    onChange={handleSelectedTypeChange}
                  />
                </FormItem>
              </Col>
              <Col xs={24} lg={6}>
                <FormItem label={t('system.dict.dictCode')}>
                  <Input value={selectedType?.dictCode || ''} readOnly />
                </FormItem>
              </Col>
              <Col xs={24} lg={6}>
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
              <Col xs={24} lg={4}>
                <FormItem label={t('common.search')} field="keyword">
                  <Input
                    placeholder={t('system.dict.itemLabelKey')}
                    onPressEnter={() => itemQueryForm.submit()}
                  />
                </FormItem>
              </Col>
              <Col xs={24} lg={6}>
                <FormItem className="filter-panel__action-item">
                  <Space>
                    <Button type="primary" htmlType="submit" icon={<IconSearch />}>
                      {t('common.search')}
                    </Button>
                    <Button onClick={handleItemReset}>{t('common.reset')}</Button>
                  </Space>
                </FormItem>
              </Col>
            </Row>
          </Form>
        </FilterPanel>

        <TableBatchActionBar
          selectedCount={selectedItemRowKeys.length}
          selectedText={t('common.selectedCount', { count: selectedItemRowKeys.length })}
          clearText={t('common.clearSelection')}
          clearSuccessText={t('common.clearSelectionSuccess')}
          onClear={() => setSelectedItemRowKeys([])}
          prefixActions={
            <ListHeaderActions
              className="dict-page__actions"
              utility={
                <>
                  <Button
                    icon={<IconRefresh />}
                    onClick={() => {
                      void handleRefreshCache();
                    }}
                    disabled={!canRefresh || !selectedType}
                  >
                    {t('system.dict.refreshCache')}
                  </Button>
                  <Button
                    onClick={() => {
                      void handleOpenUsageAnalysis();
                    }}
                    disabled={!selectedType}
                  >
                    {t('system.dict.usage.action')}
                  </Button>
                  <Button
                    icon={<IconDownload />}
                    onClick={() => {
                      void handleExportItems();
                    }}
                    disabled={!canExport || !selectedType}
                  >
                    {t('common.export')}
                  </Button>
                  <Button
                    onClick={() => {
                      void handleDownloadItemTemplate();
                    }}
                    disabled={!canImport}
                  >
                    {t('common.downloadTemplate')}
                  </Button>
                  <ImportCsvButton
                    disabled={!canImport}
                    onSelect={(file) => {
                      void handleImportItems(file);
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
                  onClick={openCreateItem}
                  disabled={!canCreate || !selectedType}
                >
                  {t('system.dict.itemAdd')}
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
                  title={t('system.dict.item.batchEnableConfirm')}
                  onOk={() => {
                    void handleBatchItemStatus(1);
                  }}
                  disabled={itemBatchActionDisabled}
                >
                  <Button disabled={itemBatchActionDisabled}>{t('system.dict.batchEnable')}</Button>
                </Popconfirm>
              </PermissionAction>
              <PermissionAction allowed={canBatchUpdate} tooltip={t('common.noPermissionAction')}>
                <Popconfirm
                  title={t('system.dict.item.batchDisableConfirm')}
                  onOk={() => {
                    void handleBatchItemStatus(2);
                  }}
                  disabled={itemBatchActionDisabled}
                >
                  <Button
                    status={itemBatchActionDisabled ? undefined : 'warning'}
                    disabled={itemBatchActionDisabled}
                  >
                    {t('system.dict.batchDisable')}
                  </Button>
                </Popconfirm>
              </PermissionAction>
              <PermissionAction allowed={canBatchDelete} tooltip={t('common.noPermissionAction')}>
                <Popconfirm
                  title={t('system.dict.item.batchDeleteConfirm')}
                  onOk={() => {
                    void handleBatchItemDelete();
                  }}
                  disabled={itemBatchDeleteDisabled}
                >
                  <Button status="danger" icon={<IconDelete />} disabled={itemBatchDeleteDisabled}>
                    {t('common.deleteSelected')}
                  </Button>
                </Popconfirm>
              </PermissionAction>
            </>
          }
        />

        {!selectedType ? (
          <PageEmpty description={t('system.dict.item.empty')} />
        ) : (
          <>
            {itemLoading && itemRows.length === 0 ? <PageLoading /> : null}
            {itemError && itemRows.length === 0 ? (
              <PageRequestError
                error={itemError}
                onRetry={() => {
                  void loadItems(itemQuery, selectedType?.dictCode);
                }}
              />
            ) : null}
            {!itemLoading && !itemError && itemTotal === 0 ? (
              <PageEmpty description={t('system.dict.itemEmpty')} />
            ) : null}
            {!itemLoading && !(itemError && itemRows.length === 0) && itemTotal > 0 ? (
              <AppTable<DictItemRow>
                className="system-list__table"
                rowKey="id"
                columns={itemColumns}
                data={itemRows}
                loading={itemLoading}
                rowSelection={{
                  selectedRowKeys: visibleSelectedItemRowKeys,
                  checkCrossPage: true,
                  preserveSelectedRowKeys: true,
                  onChange: (keys) =>
                    setSelectedItemRowKeys((currentKeys) =>
                      mergeCrossPageSelection(
                        currentKeys,
                        keys,
                        itemRows.map((item) => item.id),
                      ),
                    ),
                }}
                emptyText={t('system.dict.itemEmpty')}
                onChange={handleItemTableChange}
                pagination={buildStandardPagination(t, {
                  total: itemTotal,
                  current: itemQuery.page,
                  pageSize: itemQuery.pageSize,
                  onChange: (page, pageSize) =>
                    setItemQuery((prev) => ({ ...prev, page, pageSize })),
                })}
                scroll={{ x: 'max-content' }}
              />
            ) : null}
          </>
        )}
      </Space>

      <AppModal
        title={t('system.dict.usage.title')}
        visible={usageVisible}
        size="detail"
        footer={<Button onClick={() => setUsageVisible(false)}>{t('common.close')}</Button>}
        onCancel={() => setUsageVisible(false)}
      >
        {usageLoading ? <PageLoading /> : null}
        {!usageLoading && usageAnalysis ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Card className="page-panel">
              <Space direction="vertical" size={6} style={{ width: '100%' }}>
                <Text>{`${t('system.dict.dictCode')}: ${usageAnalysis.dictCode}`}</Text>
                <Text>{`${t('system.dict.usage.referenceCount')}: ${usageAnalysis.referenceCount}`}</Text>
                <Text type="secondary">{`${t('system.dict.usage.root')}: ${usageAnalysis.scannedProjectRoot}`}</Text>
              </Space>
            </Card>
            {usageAnalysis.references.length === 0 ? (
              <PageEmpty description={t('system.dict.usage.empty')} />
            ) : (
              <Card className="page-panel">
                <Space direction="vertical" size={10} style={{ width: '100%' }}>
                  {usageAnalysis.references.map((item) => (
                    <div
                      key={`${item.filePath}:${item.line}:${item.column}`}
                      className="dict-usage__item"
                    >
                      <div className="dict-usage__head">
                        <Text copyable>{`${item.filePath}:${item.line}:${item.column}`}</Text>
                        <Space wrap size={8}>
                          <Tag>{item.domain}</Tag>
                          {item.moduleHint ? <Tag color="arcoblue">{item.moduleHint}</Tag> : null}
                        </Space>
                      </div>
                      <Text code>{item.snippet || '-'}</Text>
                    </div>
                  ))}
                </Space>
              </Card>
            )}
          </Space>
        ) : null}
      </AppModal>

      <AppModal
        title={editingItem ? t('system.dict.itemEdit') : t('system.dict.itemCreate')}
        visible={itemVisible}
        size="md"
        onCancel={() => setItemVisible(false)}
        footer={
          <SubmitBar
            onCancel={() => setItemVisible(false)}
            onSubmit={() => {
              void submitItemForm();
            }}
            loading={itemSubmitting}
            submitText={editingItem ? t('common.save') : t('common.add')}
          />
        }
        unmountOnExit
      >
        <Form
          form={itemForm}
          layout="vertical"
          onSubmit={() => {
            void submitItemForm();
          }}
        >
          <Space direction="vertical" size={20} className="dialog-form-stack">
            <FormSection title={t('system.dict.item')}>
              <FormItem
                label={t('system.dict.dictCode')}
                field="dictCode"
                rules={[{ required: true, message: t('system.dict.dictCodeRequired') }]}
              >
                <Input disabled />
              </FormItem>
              <FormItem
                label={t('system.dict.itemLabelKey')}
                field="itemLabelKey"
                rules={[{ required: true, message: t('system.dict.itemLabelKeyRequired') }]}
              >
                <Input onPressEnter={() => itemForm.submit()} />
              </FormItem>
              <FormItem
                label={t('system.dict.itemValue')}
                field="itemValue"
                rules={[{ required: true, message: t('system.dict.itemValueRequired') }]}
              >
                <Input onPressEnter={() => itemForm.submit()} />
              </FormItem>
              <FormItem label={t('system.dict.itemColor')} field="itemColor">
                <Input
                  placeholder={t('system.dict.itemColorPlaceholder')}
                  onPressEnter={() => itemForm.submit()}
                />
              </FormItem>
              <FormItem label={t('system.dict.sort')} field="sort">
                <InputNumber min={0} />
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

export default DictItemTab;
