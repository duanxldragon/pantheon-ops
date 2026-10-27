import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Select, Space, Tag, Typography } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconClose, IconUserGroup } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { usePublicSettings } from '../../../../core/setting/publicSettings';
import {
  GovernanceSummaryBar,
  PageContainer,
  SearchToolbar,
  TABLE_COLUMN_WIDTH,
} from '../../../../components';
import '../../../../assets/list-page.css';
import {
  getAllTenantMembers,
  type MemberUsageListResp,
  type MemberUsageQuery,
  type TenantMemberUsageRow,
} from './api';
import {
  renderDateTime,
  renderPlanTag,
  renderQuota,
  renderStatusTag,
  tenantStatusTone,
} from './tenantListCells';
import { TenantListBody, TenantListTable } from './TenantListParts';
import './TenantMemberList.css';

const memberStatusTone: Record<string, string> = {
  active: 'green',
  disabled: 'gray',
};

const roleTone: Record<string, string> = {
  owner: 'orange',
  admin: 'arcoblue',
  member: 'gray',
};

interface MemberQueryState {
  tenantId?: number;
  keyword: string;
  status?: string;
  role?: string;
  page: number;
  pageSize: number;
}

const TenantMemberList: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const publicSettings = usePublicSettings();
  const tenantModeEnabled = publicSettings.tenantMode === 'multi';

  const tenantIdFromUrl = readTenantId(searchParams.get('tenantId'));

  const [query, setQuery] = useState<MemberQueryState>({
    tenantId: tenantIdFromUrl,
    keyword: '',
    page: 1,
    pageSize: 20,
  });
  const [state, setState] = useState<MemberUsageListResp>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const unlimitedText = t('system.tenant.overview.quota.unlimited');

  const loadData = useCallback(async (next: MemberQueryState) => {
    setLoading(true);
    setError(null);
    try {
      const params: MemberUsageQuery = {
        tenantId: next.tenantId,
        keyword: next.keyword.trim() || undefined,
        status: next.status as MemberUsageQuery['status'],
        role: next.role as MemberUsageQuery['role'],
        page: next.page,
        pageSize: next.pageSize,
      };
      setState(await getAllTenantMembers(params));
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

  // The drill-down lives in the URL: a link to one tenant's members is
  // something an operator can paste into chat or reopen, and it keeps the page
  // free of a tenant picker that would need `system:tenant:list` to populate.
  // A changed id is a different query, so the page resets. This is adjusted
  // during render rather than in an effect, which would render twice.
  const [urlTenantId, setUrlTenantId] = useState(tenantIdFromUrl);
  if (urlTenantId !== tenantIdFromUrl) {
    setUrlTenantId(tenantIdFromUrl);
    setQuery((prev) => ({ ...prev, tenantId: tenantIdFromUrl, page: 1 }));
  }

  const search = useCallback((patch: Partial<MemberQueryState>) => {
    setQuery((prev) => ({ ...prev, ...patch, page: 1 }));
  }, []);

  const reset = useCallback(() => {
    setSearchParams({});
    setQuery({ keyword: '', page: 1, pageSize: 20 });
  }, [setSearchParams]);

  const scopeTenant = state.items[0];
  const scopeLabel = scopeTenant
    ? scopeTenant.tenantName || scopeTenant.tenantCode
    : String(query.tenantId ?? '');

  const handleTableChange = useCallback((page: number, pageSize: number) => {
    setQuery((prev) => ({ ...prev, page, pageSize }));
  }, []);

  const columns = useMemo<ColumnProps<TenantMemberUsageRow>[]>(
    () => [
      {
        title: t('system.tenant.member.column.tenant'),
        dataIndex: 'tenantName',
        width: TABLE_COLUMN_WIDTH.lg,
        render: (value: string, row) => (
          <Space direction="vertical" size={0}>
            <Typography.Text>{value || row.tenantCode}</Typography.Text>
            <Typography.Text type="secondary" className="tenant-member-page__tenant-code">
              {row.tenantCode}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: t('system.tenant.member.column.tenantStatus'),
        dataIndex: 'tenantStatus',
        width: TABLE_COLUMN_WIDTH.status,
        render: renderStatusTag(t, tenantStatusTone, 'system.tenant.status'),
      },
      {
        title: t('system.tenant.member.column.plan'),
        dataIndex: 'plan',
        width: TABLE_COLUMN_WIDTH.sm,
        render: renderPlanTag(t),
      },
      {
        title: t('system.tenant.member.column.usage'),
        dataIndex: 'tenantUsage',
        width: TABLE_COLUMN_WIDTH.md,
        render: renderQuota<TenantMemberUsageRow>(unlimitedText, (row) => row.maxUsers),
      },
      {
        title: t('system.tenant.member.column.account'),
        dataIndex: 'username',
        width: TABLE_COLUMN_WIDTH.md,
      },
      {
        title: t('system.tenant.member.column.nickname'),
        dataIndex: 'nickname',
        width: TABLE_COLUMN_WIDTH.name,
        render: (value: string) => value || '-',
      },
      {
        title: t('system.tenant.member.column.role'),
        dataIndex: 'memberRole',
        width: TABLE_COLUMN_WIDTH.sm,
        render: renderStatusTag(t, roleTone, 'system.tenant.role'),
      },
      {
        title: t('system.tenant.member.column.status'),
        dataIndex: 'memberStatus',
        width: TABLE_COLUMN_WIDTH.status,
        render: renderStatusTag(t, memberStatusTone, 'system.tenant.member.status'),
      },
      {
        title: t('system.tenant.member.column.joinedAt'),
        dataIndex: 'joinedAt',
        width: TABLE_COLUMN_WIDTH.datetime,
        render: renderDateTime(),
      },
    ],
    [t, unlimitedText],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template tenant-member-page">
        <GovernanceSummaryBar
          eyebrow={t('system.tenant.member.hero.eyebrow')}
          title={t('system.tenant.member.hero.title')}
          metrics={[
            { key: 'members', label: t('system.tenant.member.metric.total'), value: state.total },
          ]}
        />

        <SearchToolbar
          keyword={query.keyword}
          keywordPlaceholder={t('system.tenant.member.search.placeholder')}
          onKeywordChange={(keyword) => search({ keyword })}
          inlineFilters={
            <>
              <Select
                allowClear
                placeholder={t('system.tenant.member.filter.role')}
                value={query.role}
                onChange={(value) => search({ role: value })}
              >
                <Select.Option value="owner">{t('system.tenant.role.owner')}</Select.Option>
                <Select.Option value="admin">{t('system.tenant.role.admin')}</Select.Option>
                <Select.Option value="member">{t('system.tenant.role.member')}</Select.Option>
              </Select>
              <Select
                allowClear
                placeholder={t('system.tenant.member.filter.status')}
                value={query.status}
                onChange={(value) => search({ status: value })}
              >
                <Select.Option value="active">
                  {t('system.tenant.member.status.active')}
                </Select.Option>
                <Select.Option value="disabled">
                  {t('system.tenant.member.status.disabled')}
                </Select.Option>
              </Select>
            </>
          }
          hasActiveFilters={Boolean(query.keyword || query.status || query.role || query.tenantId)}
          onClearAll={reset}
        />

        <Card className="page-panel system-list__table-card">
          {query.tenantId ? (
            <div className="tenant-member-page__scope">
              <Tag bordered={false} color="arcoblue" icon={<IconUserGroup />}>
                {t('system.tenant.member.scope', { name: scopeLabel })}
              </Tag>
              <Button
                type="text"
                size="mini"
                icon={<IconClose />}
                onClick={() => setSearchParams({})}
              >
                {t('system.tenant.member.scope.clear')}
              </Button>
            </div>
          ) : null}
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
              <TenantListTable<TenantMemberUsageRow>
                rows={state.items}
                columns={columns}
                rowKey={memberRowKey}
                loading={loading}
                page={state.page}
                pageSize={state.pageSize}
                total={state.total}
                emptyText={t('system.tenant.member.empty')}
                onPageChange={handleTableChange}
              />
            }
          />
        </Card>
      </Space>
    </PageContainer>
  );
};

function readTenantId(value: string | null): number | undefined {
  const parsed = Number(value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : undefined;
}

// One account can hold a membership in several tenants, so the user id alone is
// not a row key; the tenant it belongs to is part of the identity.
function memberRowKey(row: TenantMemberUsageRow): string {
  return `${row.tenantId}:${row.userId}`;
}

export default TenantMemberList;
