import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { Button, Card, Select, Space, Tag } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { usePermission } from '@duanxldragon/pantheon-base-ui';
import { isForbiddenRequestError } from '../../../api/request';
import {
  AppTable, GovernanceSummaryBar, PageContainer, PageEmpty, PageError,
  PageLoading, PageForbidden, SearchToolbar, buildStandardPagination,
} from '../../../components';
import { getHostList, type HostListQuery, type HostRow } from '../cmdb/host/api';
import { getBizScopeOptions, type BizScopeOptionItem } from './api';
import '../../../assets/list-page.css';

const emptyQuery: HostListQuery = { keyword: '', page: 1, pageSize: 20 };

export default function BizScopeHostBindingList() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { hasPerm } = usePermission();
  const canReadHosts = hasPerm('business:cmdb:host:view');
  const [query, setQuery] = useState<HostListQuery>(emptyQuery);
  const [rows, setRows] = useState<HostRow[]>([]);
  const [scopes, setScopes] = useState<BizScopeOptionItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    if (!canReadHosts) return () => controller.abort();
    queueMicrotask(() => {
      if (controller.signal.aborted) return;
      setLoading(true);
      setError(null);
      void Promise.all([
        getHostList(query, controller.signal),
        getBizScopeOptions(controller.signal),
      ]).then(([response, options]) => {
        if (controller.signal.aborted) return;
        setRows(response.items);
        setTotal(response.total);
        setScopes(options);
      }).catch((requestError: unknown) => {
        if (!controller.signal.aborted) {
          setRows([]);
          setError(requestError);
        }
      }).finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    });
    return () => controller.abort();
  }, [canReadHosts, query, revision]);

  const columns = useMemo<ColumnProps<HostRow>[]>(() => [
    { title: t('business.cmdb.host.hostname'), dataIndex: 'hostname', width: 200 },
    { title: t('business.cmdb.host.ip'), dataIndex: 'ip', width: 180 },
    {
      title: t('business.bizscope.hostBinding.field.scope'),
      dataIndex: 'businessScopeName', width: 200,
      render: (_: unknown, row) => row.businessScopeName || t('business.bizscope.hostBinding.unbound'),
    },
    {
      title: t('business.cmdb.host.status'), dataIndex: 'status', width: 140,
      render: (_: unknown, row) => <Tag>{t(`business.cmdb.host.status.${row.status}`, { defaultValue: row.status })}</Tag>,
    },
    {
      title: t('common.action'), width: 120, fixed: 'right',
      render: (_: unknown, row) => hasPerm('business:cmdb:host:detail') ? (
        <Button type="text" size="small" onClick={() => navigate(`/business/cmdb/host/${row.id}`)}>
          {t('common.detail')}
        </Button>
      ) : null,
    },
  ], [hasPerm, navigate, t]);

  const updateQuery = (patch: Partial<HostListQuery>) => setQuery((current) => ({ ...current, ...patch, page: 1 }));

  if (!canReadHosts || isForbiddenRequestError(error)) return <PageContainer><PageForbidden /></PageContainer>;

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.bizscope.hero.eyebrow')}
          title={t('operations.bizscope.hostBinding.menu')}
          description={t('business.bizscope.hostBinding.description')}
          metrics={[{ key: 'total', label: t('business.bizscope.hostBinding.total'), value: total }]}
        />
        <SearchToolbar
          keyword={query.keyword ?? ''}
          keywordPlaceholder={t('business.bizscope.hostBinding.keywordPlaceholder')}
          onKeywordChange={(keyword) => updateQuery({ keyword })}
          inlineFilters={(
            <Select
              allowClear showSearch aria-label={t('business.bizscope.hostBinding.field.scope')}
              placeholder={t('business.bizscope.hostBinding.field.scope')}
              value={query.businessScopeId}
              onChange={(businessScopeId) => updateQuery({ businessScopeId })}
            >
              {scopes.map((scope) => <Select.Option key={scope.id} value={scope.id}>{scope.name}</Select.Option>)}
            </Select>
          )}
          hasActiveFilters={Boolean(query.keyword || query.businessScopeId)}
          onClearAll={() => setQuery(emptyQuery)}
        />
        <Card className="page-panel system-list__table-card">
          {loading ? <PageLoading /> : null}
          {!loading && error ? <PageError onRetry={() => setRevision((value) => value + 1)} /> : null}
          {!loading && !error && rows.length === 0 ? <PageEmpty description={t('business.bizscope.hostBinding.empty')} /> : null}
          {!loading && !error && rows.length > 0 ? (
            <AppTable
              className="system-list__table" rowKey="id" data={rows} columns={columns}
              scroll={{ x: 'max-content' }}
              pagination={buildStandardPagination(t, {
                current: query.page ?? 1, pageSize: query.pageSize ?? 20, total,
                onChange: (page, pageSize) => setQuery((current) => ({ ...current, page, pageSize })),
              })}
            />
          ) : null}
        </Card>
      </Space>
    </PageContainer>
  );
}
