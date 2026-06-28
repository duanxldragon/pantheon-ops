import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Descriptions,
  Form,
  Grid,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconSearch } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '../../../core/format/dateTime';
import { usePermission } from '../../../hooks/usePermission';
import {
  getPermissionWorkbenchRemediationEvents,
  type PermissionWorkbenchPermission,
  type PermissionWorkbenchQuery,
  type PermissionWorkbenchRemediationEvent,
  type PermissionWorkbenchRole,
  type PermissionWorkbenchResp,
} from './api';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  getPagedItems,
  FilterPanel,
  PageEmpty,
  PageLoading,
  PageRequestError,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  withTableColumnPriority,
} from '../../../components';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

function dedupePermissions(items: PermissionWorkbenchPermission[]) {
  const seen = new Set<string>();
  return items.filter((item) => {
    const identity = `${item.kind}:${item.key}:${item.path || ''}`;
    if (seen.has(identity)) {
      return false;
    }
    seen.add(identity);
    return true;
  });
}

const emptyWorkbenchQuery: PermissionWorkbenchQuery = {
  roleKey: '',
  status: undefined,
  integrity: undefined,
  coverage: undefined,
};

interface PermissionWorkbenchTabProps {
  roleOptions: Array<{ label: string; value: string }>;
  utilityActions?: React.ReactNode;
  workbench: PermissionWorkbenchResp | null;
  workbenchLoading: boolean;
  workbenchError: unknown;
  workbenchQuery: PermissionWorkbenchQuery;
  onWorkbenchQueryChange: (query: PermissionWorkbenchQuery) => void;
  onRetryLoadWorkbench: () => void;
  detailRole: PermissionWorkbenchRole | null;
  onDetailRoleChange: (role: PermissionWorkbenchRole | null) => void;
  remediateRolePolicies: (role: PermissionWorkbenchRole) => Promise<void>;
  remediatingRoleKey: string;
}

export const PermissionWorkbenchTab: React.FC<PermissionWorkbenchTabProps> = ({
  roleOptions,
  utilityActions,
  workbench,
  workbenchLoading,
  workbenchError,
  workbenchQuery,
  onWorkbenchQueryChange,
  onRetryLoadWorkbench,
  detailRole,
  onDetailRoleChange,
  remediateRolePolicies,
  remediatingRoleKey,
}) => {
  const { t } = useTranslation();
  const { isAdmin } = usePermission();
  const canCreate = isAdmin;

  const [workbenchForm] = Form.useForm<PermissionWorkbenchQuery>();
  const [viewMode, setViewMode] = useState<'pending' | 'all'>('all');
  const [tablePagination, setTablePagination] = useState({ current: 1, pageSize: 10 });
  const [remediationEvents, setRemediationEvents] = useState<PermissionWorkbenchRemediationEvent[]>(
    [],
  );

  // Load remediation events when detail role changes
  useEffect(() => {
    let cancelled = false;
    if (!detailRole) {
      const timer = globalThis.setTimeout(() => {
        if (!cancelled) {
          setRemediationEvents([]);
        }
      }, 0);
      return () => {
        cancelled = true;
        globalThis.clearTimeout(timer);
      };
    }
    void getPermissionWorkbenchRemediationEvents({ roleKey: detailRole.roleKey, limit: 5 })
      .then((events) => {
        if (!cancelled) {
          setRemediationEvents(events);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setRemediationEvents([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [detailRole]);

  const searchWorkbench = () => {
    const values = workbenchForm.getFieldsValue();
    setTablePagination((current) => ({ ...current, current: 1 }));
    onWorkbenchQueryChange({
      ...workbenchQuery,
      ...values,
    });
  };

  const resetWorkbench = () => {
    workbenchForm.setFieldsValue(emptyWorkbenchQuery);
    setTablePagination({ current: 1, pageSize: 10 });
    onWorkbenchQueryChange(emptyWorkbenchQuery);
  };

  const translateTitleKey = (key?: string, fallback?: string) => {
    if (!key) {
      return fallback || '-';
    }
    return t(key, { defaultValue: fallback || key });
  };

  const overviewCards = useMemo(() => {
    const overview = workbench?.overview;
    return [
      {
        title: t('system.permission.workbench.pendingRoles'),
        value: overview?.pendingRemediationRoleCount ?? 0,
      },
      {
        title: t('system.permission.workbench.remediatedRoles'),
        value: overview?.remediatedRoleCount ?? 0,
      },
      {
        title: t('system.permission.workbench.unknownAssignments'),
        value: overview?.unknownPermissionAssignmentCount ?? 0,
      },
      {
        title: t('system.permission.workbench.recentRemediations'),
        value: overview?.recentRemediationCount ?? 0,
      },
    ];
  }, [t, workbench]);

  const displayedRoles = useMemo(() => {
    const roles = workbench?.roles ?? [];
    if (viewMode === 'all') {
      return roles;
    }
    return roles.filter((role) => role.governanceStatus === 'pending');
  }, [viewMode, workbench?.roles]);

  const { currentPage: tableCurrentPage } = getPagedItems(
    displayedRoles,
    tablePagination.current,
    tablePagination.pageSize,
  );

  const renderGovernanceStatusTag = (role: PermissionWorkbenchRole) => {
    if (role.governanceStatus === 'pending') {
      return <Tag color="red">{t('system.permission.workbench.status.pending')}</Tag>;
    }
    if (role.governanceStatus === 'remediated') {
      return <Tag color="green">{t('system.permission.workbench.status.remediated')}</Tag>;
    }
    return <Tag>{t('system.permission.workbench.status.clean')}</Tag>;
  };

  const remediationTimelineRows = useMemo(
    () =>
      remediationEvents.map((event) => ({
        ...event,
        actionLabel:
          event.action === 'remediated'
            ? t('system.permission.workbench.timeline.remediated')
            : t('system.permission.workbench.timeline.noop'),
        stateLabel: `${event.beforeState} -> ${event.afterState}`,
      })),
    [remediationEvents, t],
  );

  const detailPagePermissions = useMemo(
    () => dedupePermissions(detailRole?.pagePermissions ?? []),
    [detailRole?.pagePermissions],
  );
  const detailActionPermissions = useMemo(
    () => dedupePermissions(detailRole?.actionPermissions ?? []),
    [detailRole?.actionPermissions],
  );
  const detailUnknownPermissions = useMemo(
    () => dedupePermissions(detailRole?.unknownPermissions ?? []),
    [detailRole?.unknownPermissions],
  );

  const workbenchColumns: ColumnProps<PermissionWorkbenchRole>[] = [
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
      title: t('system.permission.workbench.governanceStatus'),
      dataIndex: 'governanceStatus',
      width: TABLE_COLUMN_WIDTH.status,
      render: (_: unknown, row: PermissionWorkbenchRole) => renderGovernanceStatusTag(row),
    },
    withTableColumnPriority(
      {
        title: t('system.permission.workbench.navCount'),
        dataIndex: 'menuCount',
        width: TABLE_COLUMN_WIDTH.count,
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('system.permission.workbench.pageCount'),
        dataIndex: 'pagePermissionCount',
        width: TABLE_COLUMN_WIDTH.count,
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('system.permission.workbench.actionCount'),
        dataIndex: 'actionPermissionCount',
        width: TABLE_COLUMN_WIDTH.count,
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('system.permission.workbench.apiCount'),
        dataIndex: 'apiPolicyCount',
        width: TABLE_COLUMN_WIDTH.count,
      },
      'low',
    ),
    {
      title: t('system.permission.workbench.coverage'),
      dataIndex: 'coverage',
      width: TABLE_COLUMN_WIDTH.tagGroup,
      render: (_: unknown, row: PermissionWorkbenchRole) => (
        <Space size={4} wrap>
          {row.hasPageGap ? (
            <Tag color="orange">{t('system.permission.workbench.coverage.pageGap')}</Tag>
          ) : null}
          {row.hasApiGap ? (
            <Tag color="red">{t('system.permission.workbench.coverage.apiGap')}</Tag>
          ) : null}
          {!row.hasPageGap && !row.hasApiGap ? (
            <Tag color="green">{t('system.permission.workbench.coverage.complete')}</Tag>
          ) : null}
        </Space>
      ),
    },
    withTableColumnPriority(
      {
        title: t('system.permission.workbench.unknownCount'),
        dataIndex: 'unknownPermissionCount',
        width: TABLE_COLUMN_WIDTH.count,
        render: (value: number) =>
          value > 0 ? <Tag color="orange">{value}</Tag> : <Tag color="green">0</Tag>,
      },
      'medium',
    ),
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.single,
      fixed: 'right',
      render: (_: unknown, row: PermissionWorkbenchRole) => (
        <Button type="text" size="small" onClick={() => onDetailRoleChange(row)}>
          {t('common.detail')}
        </Button>
      ),
    },
  ];

  return (
    <>
      <Space direction="vertical" size={12} className="permission-workbench">
        <div className="page-panel permission-workbench__context">
          <div className="permission-workbench__context-copy">
            <span className="permission-workbench__context-kicker">
              {t('system.permission.workbench.positioningTitle')}
            </span>
            <Typography.Text type="secondary" className="permission-workbench__context-desc">
              {t('system.permission.workbench.positioningHint')}
            </Typography.Text>
          </div>
          <Space
            size={8}
            className="permission-workbench__context-tools system-list__work-actions"
            wrap
          >
            {utilityActions}
            <div className="permission-workbench__view-switch">
              <Button
                type={viewMode === 'pending' ? 'primary' : 'secondary'}
                size="small"
                onClick={() => {
                  setViewMode('pending');
                  setTablePagination((current) => ({ ...current, current: 1 }));
                }}
              >
                {t('system.permission.workbench.view.pending')}
              </Button>
              <Button
                type={viewMode === 'all' ? 'primary' : 'secondary'}
                size="small"
                onClick={() => {
                  setViewMode('all');
                  setTablePagination((current) => ({ ...current, current: 1 }));
                }}
              >
                {t('system.permission.workbench.view.all')}
              </Button>
            </div>
          </Space>
        </div>

        {workbench ? (
          <Row gutter={[12, 12]} className="permission-workbench__overview">
            {overviewCards.map((item) => (
              <Col xs={24} sm={12} lg={6} key={item.title}>
                <Card className="page-stat-panel permission-workbench__overview-card">
                  <Typography.Text type="secondary">{item.title}</Typography.Text>
                  <Typography.Title heading={4} style={{ margin: '8px 0 0' }}>
                    {item.value}
                  </Typography.Title>
                </Card>
              </Col>
            ))}
          </Row>
        ) : null}

        <FilterPanel>
          <Form form={workbenchForm} layout="vertical" onSubmit={() => searchWorkbench()}>
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
                <FormItem label={t('system.permission.workbench.integrity')} field="integrity">
                  <Select
                    allowClear
                    options={[
                      {
                        label: t('system.permission.workbench.integrity.unknown'),
                        value: 'unknown',
                      },
                      {
                        label: t('system.permission.workbench.integrity.clean'),
                        value: 'clean',
                      },
                    ]}
                  />
                </FormItem>
              </Col>
              <Col span={6}>
                <FormItem label={t('system.permission.workbench.coverage')} field="coverage">
                  <Select
                    allowClear
                    options={[
                      {
                        label: t('system.permission.workbench.coverage.pageGap'),
                        value: 'page-gap',
                      },
                      {
                        label: t('system.permission.workbench.coverage.apiGap'),
                        value: 'api-gap',
                      },
                      {
                        label: t('system.permission.workbench.coverage.complete'),
                        value: 'complete',
                      },
                    ]}
                  />
                </FormItem>
              </Col>
              <Col span={4}>
                <FormItem className="filter-panel__action-item">
                  <Space>
                    <Button type="primary" htmlType="submit" icon={<IconSearch />}>
                      {t('common.search')}
                    </Button>
                    <Button onClick={resetWorkbench}>{t('common.reset')}</Button>
                  </Space>
                </FormItem>
              </Col>
            </Row>
          </Form>
        </FilterPanel>

        <Card className="page-panel system-list__table-card">
          {workbenchLoading && !workbench ? <PageLoading /> : null}
          {workbenchError && !workbench ? (
            <PageRequestError error={workbenchError} onRetry={onRetryLoadWorkbench} />
          ) : null}
          {!workbenchLoading && !workbenchError && displayedRoles.length === 0 ? (
            <PageEmpty description={t('common.noData')} />
          ) : null}
          {!workbenchLoading && !(workbenchError && !workbench) && displayedRoles.length > 0 ? (
            <AppTable<PermissionWorkbenchRole>
              className="system-list__table"
              rowKey="id"
              data={displayedRoles}
              columns={workbenchColumns}
              loading={workbenchLoading}
              scroll={{ x: 'max-content' }}
              pagination={buildStandardPagination(t, {
                current: tableCurrentPage,
                pageSize: tablePagination.pageSize,
                total: displayedRoles.length,
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
          detailRole
            ? `${detailRole.roleName} · ${detailRole.roleKey}`
            : t('system.permission.workbench.detailTitle')
        }
        visible={Boolean(detailRole)}
        size="detail"
        onCancel={() => onDetailRoleChange(null)}
        footer={null}
      >
        {detailRole ? (
          <Space direction="vertical" size={16} className="detail-stack">
            <Card
              className="detail-panel-card"
              title={t('system.permission.workbench.currentStatusSection')}
            >
              <Descriptions
                column={2}
                data={[
                  {
                    label: t('system.permission.workbench.governanceStatus'),
                    value: renderGovernanceStatusTag(detailRole),
                  },
                  {
                    label: t('system.permission.workbench.remediationAction'),
                    value:
                      detailRole.lastRemediationAction === 'remediated'
                        ? t('system.permission.workbench.timeline.remediated')
                        : detailRole.lastRemediationAction === 'noop'
                          ? t('system.permission.workbench.timeline.noop')
                          : '-',
                  },
                  {
                    label: t('system.permission.workbench.remediationTime'),
                    value: detailRole.lastRemediationAt || '-',
                  },
                  {
                    label: t('system.role.status'),
                    value:
                      detailRole.status === 1
                        ? t('system.user.status.enabled')
                        : t('system.user.status.disabled'),
                  },
                ]}
              />
            </Card>

            <Card
              className="detail-panel-card"
              title={t('system.permission.workbench.currentGapSection')}
            >
              <Descriptions
                column={4}
                data={[
                  { label: t('system.permission.workbench.navCount'), value: detailRole.menuCount },
                  {
                    label: t('system.permission.workbench.pageCount'),
                    value: detailRole.pagePermissionCount,
                  },
                  {
                    label: t('system.permission.workbench.actionCount'),
                    value: detailRole.actionPermissionCount,
                  },
                  {
                    label: t('system.permission.workbench.apiCount'),
                    value: detailRole.apiPolicyCount,
                  },
                  {
                    label: t('system.permission.workbench.apiRequiredCount'),
                    value: detailRole.requiredApiPolicyCount,
                  },
                  {
                    label: t('system.permission.workbench.apiMissingCount'),
                    value: detailRole.missingApiPolicyCount,
                  },
                  {
                    label: t('system.permission.workbench.unknownCount'),
                    value: detailRole.unknownPermissionCount,
                  },
                  {
                    label: t('system.permission.workbench.coverage'),
                    value:
                      [
                        detailRole.hasPageGap
                          ? t('system.permission.workbench.coverage.pageGap')
                          : '',
                        detailRole.hasApiGap
                          ? t('system.permission.workbench.coverage.apiGap')
                          : '',
                      ]
                        .filter(Boolean)
                        .join(' / ') || t('system.permission.workbench.coverage.complete'),
                  },
                ]}
              />
            </Card>

            <Card
              className="detail-panel-card"
              title={t('system.permission.workbench.remediationActionSection')}
              extra={
                canCreate && detailRole.missingApiPolicies.length > 0 ? (
                  <Button
                    type="primary"
                    size="small"
                    loading={remediatingRoleKey === detailRole.roleKey}
                    onClick={() => {
                      void remediateRolePolicies(detailRole);
                    }}
                  >
                    {t('system.permission.workbench.remediateAction')}
                  </Button>
                ) : null
              }
            >
              {detailRole.missingApiPolicies.length > 0 ? (
                <AppTable
                  rowKey={(record) => `${record.method}-${record.path}`}
                  data={detailRole.missingApiPolicies}
                  columns={[
                    {
                      title: t('system.permission.method'),
                      dataIndex: 'method',
                      render: (value: string) => <Tag color="red">{value}</Tag>,
                    },
                    {
                      title: t('system.permission.path'),
                      dataIndex: 'path',
                    },
                  ]}
                  pagination={false}
                  emptyText={t('common.noData')}
                />
              ) : (
                <Typography.Text type="secondary">
                  {t('system.permission.workbench.noRemediationActions')}
                </Typography.Text>
              )}
            </Card>

            <Card
              className="detail-panel-card"
              title={t('system.permission.workbench.remediationTimelineSection')}
            >
              <AppTable<
                PermissionWorkbenchRemediationEvent & { actionLabel: string; stateLabel: string }
              >
                rowKey="id"
                data={remediationTimelineRows}
                columns={[
                  {
                    title: t('system.permission.workbench.remediationAction'),
                    dataIndex: 'actionLabel',
                    render: (value: string, row) => (
                      <Tag color={row.action === 'remediated' ? 'green' : 'arcoblue'}>{value}</Tag>
                    ),
                  },
                  {
                    title: t('system.permission.workbench.remediationState'),
                    dataIndex: 'stateLabel',
                  },
                  {
                    title: t('system.permission.workbench.remediationCreated'),
                    dataIndex: 'createdCount',
                  },
                  {
                    title: t('system.permission.workbench.remediationSkipped'),
                    dataIndex: 'skippedCount',
                  },
                  {
                    title: t('system.permission.workbench.remediationTime'),
                    dataIndex: 'createdAt',
                    render: (value: string) => formatDateTime(value),
                  },
                ]}
                pagination={false}
                emptyText={t('common.noData')}
              />
            </Card>

            <Card
              className="detail-panel-card"
              title={t('system.permission.workbench.rawCoverageSection')}
            >
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Space direction="vertical" size={6} style={{ width: '100%' }}>
                  <Typography.Text bold>
                    {t('system.permission.workbench.navSection')}
                  </Typography.Text>
                  <Space wrap>
                    {detailRole.menus.length > 0 ? (
                      detailRole.menus.map((item) => (
                        <Tag key={`${item.id}-${item.path}`}>
                          {translateTitleKey(item.titleKey, item.path)}
                        </Tag>
                      ))
                    ) : (
                      <Typography.Text type="secondary">{t('common.noData')}</Typography.Text>
                    )}
                  </Space>
                </Space>

                <Space direction="vertical" size={6} style={{ width: '100%' }}>
                  <Typography.Text bold>
                    {t('system.permission.workbench.pageSection')}
                  </Typography.Text>
                  <Space wrap>
                    {detailPagePermissions.length > 0 ? (
                      detailPagePermissions.map((item) => (
                        <Tag key={`${item.kind}:${item.key}:${item.path || ''}`} color="arcoblue">
                          {translateTitleKey(item.titleKey, item.key)}
                        </Tag>
                      ))
                    ) : (
                      <Typography.Text type="secondary">{t('common.noData')}</Typography.Text>
                    )}
                  </Space>
                </Space>

                <Space direction="vertical" size={6} style={{ width: '100%' }}>
                  <Typography.Text bold>
                    {t('system.permission.workbench.actionSection')}
                  </Typography.Text>
                  <Space wrap>
                    {detailActionPermissions.length > 0 ? (
                      detailActionPermissions.map((item) => (
                        <Tag key={`${item.kind}:${item.key}:${item.path || ''}`} color="green">
                          {translateTitleKey(item.titleKey, item.key)}
                        </Tag>
                      ))
                    ) : (
                      <Typography.Text type="secondary">{t('common.noData')}</Typography.Text>
                    )}
                  </Space>
                </Space>

                <Space direction="vertical" size={6} style={{ width: '100%' }}>
                  <Typography.Text bold>
                    {t('system.permission.workbench.apiSection')}
                  </Typography.Text>
                  <AppTable
                    rowKey="id"
                    data={detailRole.apiPolicies}
                    columns={[
                      {
                        title: t('system.permission.method'),
                        dataIndex: 'method',
                        render: (value: string) => <Tag color="arcoblue">{value}</Tag>,
                      },
                      {
                        title: t('system.permission.path'),
                        dataIndex: 'path',
                      },
                    ]}
                    pagination={false}
                    emptyText={t('common.noData')}
                  />
                </Space>

                {detailUnknownPermissions.length > 0 ? (
                  <Space direction="vertical" size={6} style={{ width: '100%' }}>
                    <Typography.Text bold>
                      {t('system.permission.workbench.unknownSection')}
                    </Typography.Text>
                    <Space wrap>
                      {detailUnknownPermissions.map((item) => (
                        <Tag key={`${item.kind}:${item.key}:${item.path || ''}`} color="orange">
                          {item.key}
                        </Tag>
                      ))}
                    </Space>
                  </Space>
                ) : null}
              </Space>
            </Card>
          </Space>
        ) : null}
      </AppModal>
    </>
  );
};
