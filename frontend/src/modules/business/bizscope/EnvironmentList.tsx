import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Input, Message, Popconfirm, Select, Space, Tag } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-react/icon';
import {
  AppModal,
  AppTable,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
  TABLE_ACTION_COLUMN_WIDTH,
  buildStandardPagination,
  useGovernanceRail,
} from '../../../components';
import { usePermission } from '@duanxldragon/pantheon-base-ui';
import {
  createEnvironment,
  deleteEnvironment,
  getBizScopeOptions,
  getEnvironmentList,
  updateEnvironment,
  type EnvironmentListQuery,
  type EnvironmentPayload,
  type EnvironmentRow,
} from './api';
import EnvironmentForm from './EnvironmentForm';
import '../../../assets/list-page.css';

const prefix = 'business:bizscope:environment:';
const key = (suffix: string) => `business.bizscope.environment.${suffix}`;

const emptyQuery: EnvironmentListQuery = { code: '', name: '', page: 1, pageSize: 20 };

export default function EnvironmentList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [scopes, setScopes] = useState<{ label: string; value: number }[]>([]);
  const [scopeId, setScopeId] = useState<number | undefined>();
  const [data, setData] = useState<EnvironmentRow[]>([]);
  const [query, setQuery] = useState<EnvironmentListQuery>(emptyQuery);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<EnvironmentRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const governanceRail = useGovernanceRail();

  const canCreate = hasPerm(`${prefix}create`);
  const canUpdate = hasPerm(`${prefix}update`);
  const canDelete = hasPerm(`${prefix}delete`);

  useEffect(() => {
    getBizScopeOptions()
      .then((items) => setScopes((items ?? []).map((item) => ({ label: item.name, value: item.id }))))
      .catch(() => Message.error(t('common.loadFailed')));
  }, [t]);

  const loadData = useCallback(async (nextScope: number | undefined, nextQuery: EnvironmentListQuery) => {
    if (!nextScope) {
      setData([]);
      setTotal(0);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await getEnvironmentList(nextScope, nextQuery);
      setData(result.items);
      setTotal(result.total);
    } catch (requestError) {
      setError(requestError);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData(scopeId, query);
    });
  }, [loadData, scopeId, query]);

  const envTypeLabel = useCallback((envType: string) => t(key(envType)), [t]);

  const heroMetrics = useMemo(
    () => [{ key: 'total', label: t(key('total')), value: total }],
    [t, total],
  );

  const governanceSummaryItems = useMemo(
    () => [{ label: t(key('total')), value: total, description: '' }],
    [t, total],
  );

  const updateQuery = (patch: Partial<EnvironmentListQuery>) => setQuery((current) => ({ ...current, ...patch }));

  const handleSubmit = async (values: EnvironmentPayload) => {
    if (!scopeId) {
      Message.error(t(key('scopeRequired')));
      return;
    }
    setSubmitting(true);
    try {
      if (editing) {
        await updateEnvironment(scopeId, editing.id, values);
        Message.success(t('common.updateSuccess'));
      } else {
        await createEnvironment(scopeId, values);
        Message.success(t('common.createSuccess'));
      }
      setVisible(false);
      setEditing(null);
      await loadData(scopeId, query);
    } catch {
      Message.error(t(key('operationFailed')));
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = useCallback(async (row: EnvironmentRow) => {
    if (!scopeId) return;
    setSubmitting(true);
    try {
      await deleteEnvironment(scopeId, row.id);
      Message.success(t('common.deleteSuccess'));
      await loadData(scopeId, query);
    } catch {
      Message.error(t(key('operationFailed')));
    } finally {
      setSubmitting(false);
    }
  }, [loadData, query, scopeId, t]);

  const columns = useMemo<ColumnProps<EnvironmentRow>[]>(
    () => [
      { title: t(key('field.code')), dataIndex: 'code', width: 140, render: (_: unknown, row) => <Tag color="arcoblue">{row.code}</Tag> },
      { title: t(key('field.name')), dataIndex: 'name', minWidth: 200 },
      {
        title: t(key('field.envType')),
        dataIndex: 'envType',
        width: 140,
        render: (_: unknown, row) => <Tag color="purple">{envTypeLabel(row.envType)}</Tag>,
      },
      { title: t(key('field.remark')), dataIndex: 'remark', render: (_: unknown, row) => row.remark || '-' },
      {
        title: t('common.action'),
        width: TABLE_ACTION_COLUMN_WIDTH.medium,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canUpdate ? (
              <Button type="text" size="small" icon={<IconEdit />} onClick={() => { setEditing(row); setVisible(true); }}>
                {t('common.edit')}
              </Button>
            ) : null}
            {canDelete ? (
              <Popconfirm title={t(key('deleteConfirm'), { name: row.name })} onOk={async () => { await handleDelete(row); }}>
                <Button type="text" size="small" status="danger" icon={<IconDelete />}>
                  {t('common.delete')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    ],
    [canDelete, canUpdate, envTypeLabel, handleDelete, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.bizscope.hero.eyebrow')}
          title={t('operations.bizscope.environment.menu')}
          description={t(key('description'))}
          metrics={heroMetrics}
          action={
            <GovernanceRailToggleButton expanded={governanceRail.expanded} onToggle={governanceRail.toggle}>
              {t('operations.bizscope.environment.menu')}
            </GovernanceRailToggleButton>
          }
        />
        <SearchToolbar
          keyword={query.code ?? ''}
          keywordPlaceholder={t(key('keywordPlaceholder'))}
          onKeywordChange={(code) => updateQuery({ code })}
          inlineFilters={
            <Select
              allowClear
              placeholder={t(key('field.scope'))}
              value={scopeId}
              onChange={(value) => setScopeId(value as number | undefined)}
              style={{ minWidth: 180 }}
            >
              {(scopes ?? []).map((scope) => (
                <Select.Option key={scope.value} value={scope.value}>
                  {scope.label}
                </Select.Option>
              ))}
            </Select>
          }
          advancedFilters={
            <Input allowClear placeholder={t(key('field.name'))} value={query.name} onChange={(name) => updateQuery({ name })} />
          }
          advancedActiveCount={Number(Boolean(query.name))}
          hasActiveFilters={Boolean(query.code || query.name)}
          onClearAll={() => setQuery(emptyQuery)}
        />
        <Card className="page-panel system-list__table-card">
          <div style={{ paddingBottom: 12 }}>
            {canCreate ? (
              <ListHeaderActions
                primary={
                  <Button
                    type="primary"
                    icon={<IconPlus />}
                    onClick={() => {
                      setEditing(null);
                      setVisible(true);
                    }}
                  >
                    {t('common.add')}
                  </Button>
                }
              />
            ) : null}
          </div>
          {!scopeId ? <PageEmpty description={t(key('scopeRequired'))} /> : null}
          {scopeId && loading && data.length === 0 ? <PageLoading /> : null}
          {scopeId && error && data.length === 0 ? <PageError onRetry={() => void loadData(scopeId, query)} /> : null}
          {scopeId && !loading && !error && data.length === 0 ? <PageEmpty description={t(key('empty'))} /> : null}
          {scopeId && !loading && !error && data.length > 0 ? (
            <AppTable
              className="system-list__table"
              rowKey="id"
              data={data}
              columns={columns}
              scroll={{ x: 'max-content' }}
              pagination={buildStandardPagination(t, {
                current: query.page || 1,
                pageSize: query.pageSize || 20,
                total,
                onChange: (page, pageSize) => updateQuery({ page, pageSize }),
              })}
            />
          ) : null}
        </Card>
      </Space>
      <GovernanceInsightDrawer visible={governanceRail.expanded} title={t('operations.bizscope.environment.menu')} onClose={governanceRail.close}>
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
      <AppModal
        visible={visible}
        footer={null}
        title={editing ? t(key('editTitle')) : t(key('createTitle'))}
        onCancel={() => {
          setVisible(false);
          setEditing(null);
        }}
      >
        <EnvironmentForm
          mode={editing ? 'update' : 'create'}
          initialValues={editing || undefined}
          submitting={submitting}
          onSubmit={handleSubmit}
          onCancel={() => {
            setVisible(false);
            setEditing(null);
          }}
        />
      </AppModal>
    </PageContainer>
  );
}