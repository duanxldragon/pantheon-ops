import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Grid, Select, Space, Tag, Typography } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconEdit, IconUserGroup } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { usePublicSettings } from '../../../../core/setting/publicSettings';
import { usePermission } from '../../../../hooks/usePermission';
import {
  GovernanceSummaryBar,
  ListHeaderActions,
  PageContainer,
  PermissionAction,
  SearchToolbar,
  SystemRowActions,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
} from '../../../../components';
import '../../../../assets/list-page.css';
import {
  getTenantOverview,
  type OverviewPlanRow,
  type OverviewResp,
  type OverviewTenantRow,
  type TenantOverviewQuery,
} from './api';
import TenantPlanEditor from './TenantPlanEditor';
import {
  planTone,
  renderDateTime,
  renderPlanTag,
  renderQuota,
  renderStatusTag,
  tenantStatusTone,
} from './tenantListCells';
import { TenantListBody, TenantListTable } from './TenantListParts';
import { formatQuotaLimit } from './quota';
import './TenantOverview.css';

const Row = Grid.Row;
const Col = Grid.Col;

interface OverviewQueryState {
  keyword: string;
  status?: string;
  plan?: string;
  page: number;
  pageSize: number;
}

const emptyQuery: OverviewQueryState = { keyword: '', page: 1, pageSize: 10 };

const TenantOverview: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { isAdmin, hasPerm } = usePermission();
  const publicSettings = usePublicSettings();
  const tenantModeEnabled = publicSettings.tenantMode === 'multi';
  const canEditPlan =
    isAdmin || hasPerm('system:tenant:plan:create') || hasPerm('system:tenant:plan:update');

  const unlimitedText = t('system.tenant.overview.quota.unlimited');

  const [query, setQuery] = useState<OverviewQueryState>(emptyQuery);
  const [state, setState] = useState<OverviewResp>({
    totals: { members: 0, roles: 0, depts: 0 },
    plans: [],
    items: [],
    total: 0,
    page: 1,
    size: 10,
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [editorVisible, setEditorVisible] = useState(false);

  const loadData = useCallback(async (next: OverviewQueryState) => {
    setLoading(true);
    setError(null);
    try {
      const params: TenantOverviewQuery = {
        keyword: next.keyword.trim() || undefined,
        status: next.status as TenantOverviewQuery['status'],
        plan: next.plan || undefined,
        page: next.page,
        size: next.pageSize,
      };
      setState(await getTenantOverview(params));
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      void loadData(query);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadData, query]);

  const search = useCallback((patch: Partial<OverviewQueryState>) => {
    setQuery((prev) => ({ ...prev, ...patch, page: 1 }));
  }, []);

  const reset = useCallback(() => setQuery(emptyQuery), []);

  const handleTableChange = useCallback((page: number, pageSize: number) => {
    setQuery((prev) => ({ ...prev, page, pageSize }));
  }, []);

  const openTenantMembers = useCallback(
    (row: OverviewTenantRow) => {
      navigate(`/system/tenant-ops/member?tenantId=${row.tenantId}`);
    },
    [navigate],
  );

  const summaryItems = useMemo(() => {
    const summary = state.summary;
    return [
      {
        key: 'tenants',
        label: t('system.tenant.overview.metric.tenants'),
        value: summary?.total ?? state.total,
      },
      {
        key: 'members',
        label: t('system.tenant.overview.metric.members'),
        value: state.totals.members,
      },
      { key: 'roles', label: t('system.tenant.overview.metric.roles'), value: state.totals.roles },
      { key: 'depts', label: t('system.tenant.overview.metric.depts'), value: state.totals.depts },
    ];
  }, [state.summary, state.total, state.totals, t]);

  const columns = useMemo<ColumnProps<OverviewTenantRow>[]>(
    () => [
      { title: t('system.tenant.code'), dataIndex: 'code', width: TABLE_COLUMN_WIDTH.code },
      { title: t('system.tenant.name'), dataIndex: 'name', width: TABLE_COLUMN_WIDTH.name },
      {
        title: t('system.tenant.status'),
        dataIndex: 'status',
        width: TABLE_COLUMN_WIDTH.status,
        render: renderStatusTag(t, tenantStatusTone, 'system.tenant.status'),
      },
      {
        title: t('system.tenant.plan'),
        dataIndex: 'plan',
        width: TABLE_COLUMN_WIDTH.sm,
        render: renderPlanTag(t),
      },
      {
        title: t('system.tenant.overview.column.members'),
        dataIndex: 'members',
        width: TABLE_COLUMN_WIDTH.md,
        render: renderQuota<OverviewTenantRow>(unlimitedText, (row) => row.maxUsers),
      },
      {
        title: t('system.tenant.overview.column.roles'),
        dataIndex: 'roles',
        width: TABLE_COLUMN_WIDTH.sm,
        render: renderQuota<OverviewTenantRow>(unlimitedText, (row) => row.maxRoles),
      },
      {
        title: t('system.tenant.overview.column.depts'),
        dataIndex: 'depts',
        width: TABLE_COLUMN_WIDTH.sm,
        render: renderQuota<OverviewTenantRow>(unlimitedText, (row) => row.maxDepts),
      },
      {
        title: t('system.tenant.createdAt'),
        dataIndex: 'createdAt',
        width: TABLE_COLUMN_WIDTH.datetime,
        render: renderDateTime(),
      },
      {
        title: t('common.actions'),
        dataIndex: 'operation',
        width: TABLE_ACTION_COLUMN_WIDTH.single,
        fixed: 'right' as const,
        render: (_, row) => (
          <SystemRowActions
            actions={[
              {
                key: 'members',
                text: t('system.tenant.overview.viewMembers'),
                icon: <IconUserGroup />,
                onClick: () => openTenantMembers(row),
              },
            ]}
          />
        ),
      },
    ],
    [openTenantMembers, t, unlimitedText],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template tenant-overview-page">
        <GovernanceSummaryBar
          eyebrow={t('system.tenant.overview.hero.eyebrow')}
          title={t('system.tenant.overview.hero.title')}
          metrics={summaryItems}
          action={
            <PermissionAction allowed={canEditPlan} tooltip={t('common.noPermissionAction')}>
              <Button type="primary" icon={<IconEdit />} onClick={() => setEditorVisible(true)}>
                {t('system.tenant.overview.planEditor.action')}
              </Button>
            </PermissionAction>
          }
        />

        <TenantPlanDistribution rows={state.plans} loading={loading} />

        <SearchToolbar
          keyword={query.keyword}
          keywordPlaceholder={t('system.tenant.search.placeholder')}
          onKeywordChange={(keyword) => search({ keyword })}
          inlineFilters={
            <>
              <Select
                allowClear
                placeholder={t('system.tenant.filter.status')}
                value={query.status}
                onChange={(value) => search({ status: value })}
              >
                <Select.Option value="active">{t('system.tenant.status.active')}</Select.Option>
                <Select.Option value="suspended">
                  {t('system.tenant.status.suspended')}
                </Select.Option>
                <Select.Option value="archived">{t('system.tenant.status.archived')}</Select.Option>
                <Select.Option value="deleted">{t('system.tenant.status.deleted')}</Select.Option>
              </Select>
              <PlanFilterSelect
                value={query.plan}
                rows={state.plans}
                onChange={(plan) => search({ plan })}
              />
            </>
          }
          hasActiveFilters={Boolean(query.keyword || query.status || query.plan)}
          onClearAll={reset}
        />

        <Card className="page-panel system-list__table-card">
          <ListHeaderActions
            utility={
              <Typography.Text type="secondary" className="tenant-overview__table-hint">
                {t('system.tenant.overview.table.hint')}
              </Typography.Text>
            }
          />
          <TenantListBody
            tenantModeEnabled={tenantModeEnabled}
            loading={loading}
            rowCount={state.items.length}
            error={error}
            onRetry={() => {
              void loadData(query);
            }}
            onOpenPlatformSettings={() => navigate('/system/setting/platform')}
            table={
              <TenantListTable<OverviewTenantRow>
                rows={state.items}
                columns={columns}
                rowKey="tenantId"
                loading={loading}
                page={state.page}
                pageSize={state.size}
                total={state.total}
                emptyText={t('system.tenant.overview.empty')}
                onPageChange={handleTableChange}
              />
            }
          />
        </Card>
      </Space>

      <TenantPlanEditor
        visible={editorVisible}
        onClose={() => setEditorVisible(false)}
        onSaved={() => {
          void loadData(query);
        }}
      />
    </PageContainer>
  );
};

/**
 * The plan filter only offers codes the installation actually carries, plus the
 * unmanaged bucket. Listing definitions the operator created but never assigned
 * would produce an empty result that reads like a data problem.
 */
const PlanFilterSelect: React.FC<{
  value?: string;
  rows: OverviewPlanRow[];
  onChange: (plan: string) => void;
}> = ({ value, rows, onChange }) => {
  const { t } = useTranslation();
  return (
    <Select
      allowClear
      placeholder={t('system.tenant.filter.plan')}
      value={value}
      onChange={onChange}
    >
      {rows.map((row) => (
        <Select.Option key={row.plan || 'none'} value={row.plan}>
          {row.name || row.plan || t('system.tenant.overview.plan.unmanaged')}
        </Select.Option>
      ))}
    </Select>
  );
};

const TenantPlanDistribution: React.FC<{ rows: OverviewPlanRow[]; loading: boolean }> = ({
  rows,
  loading,
}) => {
  const { t } = useTranslation();
  const unlimitedText = t('system.tenant.overview.quota.unlimited');

  if (!loading && rows.length === 0) {
    return null;
  }

  return (
    <Card className="page-panel tenant-overview__plans">
      <Typography.Text type="secondary" className="tenant-overview__plans-title">
        {t('system.tenant.overview.planDistribution')}
      </Typography.Text>
      <Row gutter={16}>
        {rows.map((row) => (
          <Col key={row.plan || 'none'} xs={24} sm={12} md={8} lg={6}>
            <div className="tenant-overview__plan">
              <div className="tenant-overview__plan-head">
                <Tag color={row.known ? (planTone[row.plan] ?? 'arcoblue') : 'red'}>
                  {row.name || row.plan || t('system.tenant.overview.plan.unmanaged')}
                </Tag>
                <span className="tenant-overview__plan-count">
                  {t('system.tenant.overview.plan.tenants', { count: row.tenantCount })}
                </span>
              </div>
              <ul className="tenant-overview__plan-quotas">
                <li>
                  <span>{t('system.tenant.overview.column.members')}</span>
                  <strong>{formatQuotaLimit(row.quota.max_users, unlimitedText)}</strong>
                </li>
                <li>
                  <span>{t('system.tenant.overview.column.roles')}</span>
                  <strong>{formatQuotaLimit(row.quota.max_roles, unlimitedText)}</strong>
                </li>
                <li>
                  <span>{t('system.tenant.overview.column.depts')}</span>
                  <strong>{formatQuotaLimit(row.quota.max_depts, unlimitedText)}</strong>
                </li>
              </ul>
            </div>
          </Col>
        ))}
      </Row>
    </Card>
  );
};

export default TenantOverview;
