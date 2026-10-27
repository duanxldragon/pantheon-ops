import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Message, Select, Space, Tag } from '@arco-design/web-react';
import { IconSearch } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppTable,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
} from '../../../components';
import { getDatasourceList, queryLogs, type DatasourceRow } from './api';
import '../../../assets/list-page.css';

interface LogRow {
  id: string;
  timestamp: string;
  labels: Record<string, string>;
  message: string;
}

function formatTimestamp(value: string) {
  const milliseconds = value.length > 13 ? Number(value.slice(0, 13)) : Number(value);
  return Number.isFinite(milliseconds) ? new Date(milliseconds).toLocaleString() : value;
}

export default function LogExplorer() {
  const { t } = useTranslation();
  const [datasources, setDatasources] = useState<DatasourceRow[]>([]);
  const [datasourceId, setDatasourceId] = useState<number>();
  const [query, setQuery] = useState('{app=~".+"}');
  const [items, setItems] = useState<LogRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [hasQueried, setHasQueried] = useState(false);

  useEffect(() => {
    queueMicrotask(async () => {
      try {
        const response = await getDatasourceList();
        const lokiSources = response.items.filter((item) => item.type === 'loki');
        setDatasources(lokiSources);
        setDatasourceId(lokiSources.find((item) => item.isDefault)?.id ?? lokiSources[0]?.id);
      } catch (requestError) {
        setError(requestError);
      }
    });
  }, []);

  const executeQuery = useCallback(async () => {
    if (!datasourceId || !query.trim()) {
      Message.warning(t('business.observability.logs.queryRequired'));
      return;
    }
    setLoading(true);
    setError(null);
    setHasQueried(true);
    try {
      const response = await queryLogs(datasourceId, { query: query.trim(), limit: 500 });
      const rows: LogRow[] = [];
      for (const stream of response.data?.result ?? []) {
        for (const [timestamp, message] of stream.values ?? []) {
          rows.push({ id: `${timestamp}-${rows.length}`, timestamp, labels: stream.stream ?? {}, message });
        }
      }
      rows.sort((left, right) => right.timestamp.localeCompare(left.timestamp));
      setItems(rows);
    } catch (requestError) {
      setError(requestError);
      setItems([]);
      Message.error(t('business.observability.logs.queryFailed'));
    } finally {
      setLoading(false);
    }
  }, [datasourceId, query, t]);

  const columns = useMemo<ColumnProps<LogRow>[]>(
    () => [
      { title: t('business.observability.logs.field.time'), dataIndex: 'timestamp', width: 190, render: (value: string) => formatTimestamp(value) },
      {
        title: t('business.observability.logs.field.labels'),
        dataIndex: 'labels',
        width: 300,
        render: (_: unknown, row) => (
          <Space size={4} wrap>
            {Object.entries(row.labels).map(([key, value]) => <Tag key={key}>{key}={value}</Tag>)}
          </Space>
        ),
      },
      { title: t('business.observability.logs.field.message'), dataIndex: 'message', ellipsis: true },
    ],
    [t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.observability.logs.hero.eyebrow')}
          title={t('operations.observability.logs.menu')}
          description={t('business.observability.logs.hero.title')}
          metrics={[
            { key: 'sources', label: t('business.observability.logs.hero.sources'), value: datasources.length },
            { key: 'results', label: t('business.observability.logs.hero.results'), value: items.length },
          ]}
        />
        <SearchToolbar
          keyword={query}
          keywordPlaceholder={t('business.observability.logs.queryPlaceholder')}
          onKeywordChange={setQuery}
          inlineFilters={
            <Select
              value={datasourceId}
              placeholder={t('business.observability.common.datasource')}
              style={{ width: 240 }}
              onChange={setDatasourceId}
            >
              {datasources.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}
            </Select>
          }
          hasActiveFilters={Boolean(query)}
          onClearAll={() => setQuery('')}
          extra={<Button type="primary" icon={<IconSearch />} loading={loading} onClick={() => void executeQuery()}>{t('common.search')}</Button>}
        />
        <Card className="page-panel system-list__table-card">
          {loading && items.length === 0 ? <PageLoading /> : null}
          {error && items.length === 0 ? <PageError onRetry={() => void executeQuery()} /> : null}
          {!loading && !error && datasources.length === 0 ? <PageEmpty description={t('business.observability.logs.noDatasource')} /> : null}
          {!loading && !error && datasources.length > 0 && hasQueried && items.length === 0 ? <PageEmpty description={t('business.observability.logs.empty')} /> : null}
          {!loading && !error && items.length > 0 ? <AppTable rowKey="id" columns={columns} data={items} pagination={{ pageSize: 50 }} scroll={{ x: 'max-content' }} /> : null}
        </Card>
      </Space>
    </PageContainer>
  );
}
