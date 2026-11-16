import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Select, Space, Tag } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppTable,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  useGovernanceRail,
} from '../../../components';
import { getBizScopeHosts, getBizScopeOptions, type BizScopeHostRow } from './api';
import '../../../assets/list-page.css';

const key = (suffix: string) => `business.bizscope.hostBinding.${suffix}`;

export default function HostBinding() {
  const { t } = useTranslation();
  const [scopes, setScopes] = useState<{ label: string; value: number }[]>([]);
  const [scopeId, setScopeId] = useState<number | undefined>();
  const [data, setData] = useState<BizScopeHostRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const governanceRail = useGovernanceRail();

  useEffect(() => {
    getBizScopeOptions()
      .then((items) => setScopes((items ?? []).map((item) => ({ label: item.name, value: item.id }))))
      .catch(() => undefined);
  }, []);

  const loadData = useCallback(async (nextScope: number | undefined) => {
    if (!nextScope) {
      setData([]);
      setTotal(0);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await getBizScopeHosts(nextScope);
      setData(result.items);
      setTotal(result.total);
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData(scopeId);
    });
  }, [loadData, scopeId]);

  const heroMetrics = useMemo(
    () => [{ key: 'total', label: t(key('total')), value: total }],
    [t, total],
  );

  const governanceSummaryItems = useMemo(
    () => [{ label: t(key('total')), value: total, description: '' }],
    [t, total],
  );

  const columns = useMemo<ColumnProps<BizScopeHostRow>[]>(
    () => [
      { title: t(key('field.scope')), dataIndex: 'businessScopeName', minWidth: 140 },
      { title: t('business.cmdb.host.hostname'), dataIndex: 'hostname', minWidth: 180 },
      { title: t('business.cmdb.host.ip'), dataIndex: 'ip', width: 150 },
      { title: t('business.cmdb.host.os'), dataIndex: 'os', width: 160 },
      {
        title: t('business.cmdb.host.status'),
        dataIndex: 'status',
        width: 110,
        render: (_: unknown, row) => (
          <Tag color={row.status === 'online' ? 'green' : 'gray'}>{t(`business.cmdb.host.status.${row.status}`)}</Tag>
        ),
      },
    ],
    [t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.bizscope.hero.eyebrow')}
          title={t('operations.bizscope.hostBinding.menu')}
          description={t(key('description'))}
          metrics={heroMetrics}
          action={
            <GovernanceRailToggleButton expanded={governanceRail.expanded} onToggle={governanceRail.toggle}>
              {t('operations.bizscope.hostBinding.menu')}
            </GovernanceRailToggleButton>
          }
        />
        <Card className="page-panel">
          <Space style={{ marginBottom: 16 }}>
            <Select
              allowClear
              placeholder={t(key('field.scope'))}
              value={scopeId}
              onChange={(value) => setScopeId(value as number | undefined)}
              style={{ minWidth: 260 }}
            >
              {(scopes ?? []).map((scope) => (
                <Select.Option key={scope.value} value={scope.value}>
                  {scope.label}
                </Select.Option>
              ))}
            </Select>
            {total > 0 ? <Tag color="arcoblue">{t(key('total'))}: {total}</Tag> : null}
          </Space>
          {!scopeId ? <PageEmpty description={t(key('field.scope'))} /> : null}
          {scopeId && loading ? <PageLoading /> : null}
          {scopeId && error ? <PageError onRetry={() => void loadData(scopeId)} /> : null}
          {scopeId && !loading && !error && data.length === 0 ? <PageEmpty description={t(key('empty'))} /> : null}
          {scopeId && !loading && !error && data.length > 0 ? (
            <AppTable rowKey="id" data={data} columns={columns} scroll={{ x: 'max-content' }} />
          ) : null}
        </Card>
      </Space>
      <GovernanceInsightDrawer visible={governanceRail.expanded} title={t('operations.bizscope.hostBinding.menu')} onClose={governanceRail.close}>
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
    </PageContainer>
  );
}