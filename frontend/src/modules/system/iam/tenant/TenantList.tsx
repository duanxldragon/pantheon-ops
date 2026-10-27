import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Grid,
  Input,
  Popconfirm,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import PasswordInput from '../../../../components/patterns/forms/PasswordInput';
import { message } from '../../../../components/feedback/message';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconEdit, IconPlus, IconUserGroup } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { isArcoFormValidationError } from '../../../../core/arco/formValidation';
import { formatDateTime } from '../../../../core/format/dateTime';
import { usePublicSettings } from '../../../../core/setting/publicSettings';
import { usePermission } from '../../../../hooks/usePermission';
import {
  addTenantMember,
  createTenant,
  deleteTenant,
  getPlanList,
  getTenantList,
  getTenantMembers,
  removeTenantMember,
  updateTenant,
  type PlanRow,
  type TenantEditableStatus,
  type TenantListResp,
  type TenantMemberRow,
  type TenantPayload,
  type TenantRow,
  type TenantUpdatePayload,
} from './api';
import { getSecurityOverview } from '../../../auth/security/api';
import { normalizeTenantPlan } from '../../../../core/entitlements/plans';
import { definedPlanOptions, editorPlanOptions, type PlanOption } from './planOptions';
import TenantPlanMatrix from './TenantPlanMatrix';
import { isRequestError } from '../../../../api/request';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  SearchToolbar,
  FormSection,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  FormModalFooter,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageLoading,
  PageRequestError,
  PermissionAction,
  SystemRowActions,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  useGovernanceRail,
} from '../../../../components';
import '../../../../assets/list-page.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

interface TenantListQueryState {
  keyword: string;
  status?: string;
  plan?: string;
  page: number;
  pageSize: number;
}

interface TenantFormValues {
  code: string;
  name: string;
  plan?: string;
  status?: TenantEditableStatus;
  metadata?: string;
  adminUsername?: string;
  adminPassword?: string;
  adminNickname?: string;
}

const emptyQuery: TenantListQueryState = { keyword: '', page: 1, pageSize: 10 };

const planTone: Record<string, string> = {
  free: 'gray',
  basic: 'blue',
  professional: 'arcoblue',
  enterprise: 'orange',
};

const statusTone: Record<string, string> = {
  active: 'green',
  suspended: 'orange',
  archived: 'gray',
  deleted: 'red',
};

const TenantList: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { isAdmin, hasPerm } = usePermission();
  const publicSettings = usePublicSettings();
  // The console is reachable in both modes. Under compat there is no tenant
  // dimension to manage yet, so the page explains how to enable it instead of
  // disappearing from the navigation (which left operators with no way to
  // discover the feature).
  const tenantModeEnabled = publicSettings.tenantMode === 'multi';
  const canCreate = isAdmin || hasPerm('system:tenant:create');
  const canUpdate = isAdmin || hasPerm('system:tenant:update');
  const canDelete = isAdmin || hasPerm('system:tenant:delete');
  const canManageMember = isAdmin || hasPerm('system:tenant:member-add');

  // The request interceptor already reports the server's message for a
  // RequestError. Re-reporting it here stacked two toasts — a specific one and a
  // generic "action failed" that said strictly less — so the local fallback is
  // reserved for failures the interceptor does not speak for: a rejected form
  // field, or a non-request exception.
  const reportUnexpectedError = useCallback(
    (err: unknown) => {
      if (isArcoFormValidationError(err) || isRequestError(err)) {
        return;
      }
      message.error(t('common.actionFailed'));
    },
    [t],
  );

  const governanceRail = useGovernanceRail();

  // i18next echoes the key when a translation is missing, which is how the
  // console ended up rendering raw `system.tenant.status.archived` strings.
  // Falling back to the backend value keeps the cell readable either way.
  const statusLabel = useCallback(
    (value: string) => t(`system.tenant.status.${value}`, { defaultValue: value }),
    [t],
  );
  const planLabel = useCallback(
    (value: string) => t(`system.tenant.plan.${value}`, { defaultValue: value }),
    [t],
  );
  const roleLabel = useCallback(
    (value: string) => t(`system.tenant.role.${value}`, { defaultValue: value }),
    [t],
  );

  const [query, setQuery] = useState<TenantListQueryState>(emptyQuery);
  const [state, setState] = useState<TenantListResp>({ items: [], total: 0, page: 1, size: 10 });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const [formVisible, setFormVisible] = useState(false);
  const [editing, setEditing] = useState<TenantRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [planCodeDraft, setPlanCodeDraft] = useState<string>('free');
  const [plans, setPlans] = useState<PlanRow[] | null>(null);
  const [form] = Form.useForm<TenantFormValues>();
  // The tenant administrator's password is validated against the deployment's
  // configurable policy rather than a number copied into this file, so the form
  // and the server cannot disagree about what is acceptable. 6 is the seeded
  // default and the server's own fallback.
  const [adminPasswordMinLength, setAdminPasswordMinLength] = useState(6);

  const [memberVisible, setMemberVisible] = useState(false);
  const [memberTarget, setMemberTarget] = useState<TenantRow | null>(null);
  const [members, setMembers] = useState<TenantMemberRow[]>([]);
  const [memberTotal, setMemberTotal] = useState(0);
  const [memberPage, setMemberPage] = useState(1);
  const [memberPageSize, setMemberPageSize] = useState(10);
  const [memberLoading, setMemberLoading] = useState(false);
  const [memberForm] = Form.useForm<{ userId: string; role: string }>();

  const loadData = useCallback(async (next: TenantListQueryState) => {
    setLoading(true);
    setError(null);
    try {
      const resp = await getTenantList({
        keyword: next.keyword.trim() || undefined,
        status: next.status as never,
        plan: next.plan as never,
        page: next.page,
        size: next.pageSize,
      });
      setState(resp);
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

  // Resolve the password policy once so the create form validates against the
  // value the server will actually enforce. A failure is not surfaced: the
  // server remains the authority and still rejects a short password, so the
  // only cost of falling back is a less precise client-side hint.
  useEffect(() => {
    let cancelled = false;
    getSecurityOverview()
      .then((overview) => {
        const min = overview?.policy?.passwordMinLength;
        if (!cancelled && typeof min === 'number' && min > 0) {
          setAdminPasswordMinLength(min);
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

  // Plan codes live in `tenant_plans` now (contract §2.4), so the editor and the
  // filter must offer what this installation defines instead of the four literal
  // tiers this file used to hardcode. A failed read stays silent: the server is
  // still the authority on save (`PlanStore.ValidateAssignment`), so falling back
  // to the built-in four only costs the operator-defined choices.
  useEffect(() => {
    let cancelled = false;
    getPlanList({ skipErrorMessage: true })
      .then((resp) => {
        if (!cancelled) setPlans(resp?.items ?? []);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

  const filterPlanOptions = useMemo(() => definedPlanOptions(plans), [plans]);
  const editorOptions = useMemo(
    () => editorPlanOptions(plans, editing?.plan ?? ''),
    [editing, plans],
  );

  const planOptionLabel = useCallback(
    (option: PlanOption) => (option.builtin ? t(`system.tenant.plan.${option.code}`) : option.name),
    [t],
  );

  // The capability matrix is a frontend ladder of the four built-in tiers
  // (core/entitlements/plans), so an operator-defined code resolves to free.
  // Quotas still come from the plan definition; only the console surfaces do.
  const planDraft = normalizeTenantPlan(planCodeDraft);
  const planOutsideLadder = planCodeDraft !== '' && planDraft !== planCodeDraft;

  const search = useCallback((patch: Partial<TenantListQueryState>) => {
    setQuery((prev) => ({ ...prev, ...patch, page: 1 }));
  }, []);

  const reset = useCallback(() => setQuery(emptyQuery), []);

  const handleTableChange = useCallback((pagination: { current?: number; pageSize?: number }) => {
    setQuery((prev) => ({
      ...prev,
      page: pagination.current ?? 1,
      pageSize: pagination.pageSize ?? prev.pageSize,
    }));
  }, []);

  const loadMembers = useCallback(
    async (tenant: TenantRow, page: number, pageSize: number) => {
      setMemberLoading(true);
      try {
        const resp = await getTenantMembers(tenant.id, { page, pageSize });
        setMembers(resp.items);
        setMemberTotal(resp.total);
        setMemberPage(resp.page);
        setMemberPageSize(resp.pageSize);
      } catch (err) {
        reportUnexpectedError(err);
      } finally {
        setMemberLoading(false);
      }
    },
    [reportUnexpectedError],
  );

  const openMembers = useCallback(
    (row: TenantRow) => {
      setMemberTarget(row);
      setMemberVisible(true);
      memberForm.resetFields();
      void loadMembers(row, 1, 10);
    },
    [loadMembers, memberForm],
  );

  const closeMembers = useCallback(() => {
    setMemberVisible(false);
    setMemberTarget(null);
    setMembers([]);
  }, []);

  const openCreate = useCallback(() => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ plan: 'free' });
    setPlanCodeDraft('free');
    setFormVisible(true);
  }, [form]);

  const openEdit = useCallback(
    (row: TenantRow) => {
      setEditing(row);
      form.setFieldsValue({
        code: row.code,
        name: row.name,
        plan: row.plan,
        // A soft-deleted row only shows up under an explicit status=deleted
        // filter; leaving the field empty keeps the select honest instead of
        // silently proposing a lifecycle value it does not offer.
        status: row.status === 'deleted' ? undefined : row.status,
        metadata: row.metadata ?? '',
      });
      setPlanCodeDraft((row.plan ?? '').trim());
      setFormVisible(true);
    },
    [form],
  );

  const handleSubmit = useCallback(async () => {
    try {
      const values = await form.validate();
      setSubmitting(true);
      if (editing) {
        const payload: TenantUpdatePayload = {
          name: values.name,
          status: values.status,
          plan: values.plan as never,
          metadata: values.metadata,
        };
        await updateTenant(editing.id, payload);
        message.success(t('common.updateSuccess'));
      } else {
        const payload: TenantPayload = {
          code: values.code,
          name: values.name,
          plan: values.plan as never,
          metadata: values.metadata,
        };
        // The console always provisions an initial administrator; an empty
        // tenant would have no account to sign in with.
        const adminUsername = String(values.adminUsername ?? '').trim();
        payload.admin_username = adminUsername;
        payload.admin_password = values.adminPassword;
        payload.admin_nickname = values.adminNickname?.trim() || adminUsername;
        await createTenant(payload);
        message.success(t('common.createSuccess'));
      }
      setFormVisible(false);
      await loadData(query);
    } catch (err) {
      reportUnexpectedError(err);
    } finally {
      setSubmitting(false);
    }
  }, [editing, form, loadData, query, reportUnexpectedError, t]);

  const handleDelete = useCallback(
    async (row: TenantRow) => {
      try {
        await deleteTenant(row.id);
        message.success(t('common.deleteSuccess'));
        await loadData(query);
      } catch (err) {
        reportUnexpectedError(err);
      }
    },
    [loadData, query, reportUnexpectedError, t],
  );

  const handleAddMember = useCallback(async () => {
    if (!memberTarget) {
      return;
    }
    try {
      const values = await memberForm.validate();
      await addTenantMember(memberTarget.id, {
        user_id: Number(values.userId),
        role: values.role as never,
      });
      message.success(t('system.tenant.memberAddSuccess'));
      memberForm.resetFields();
      await loadMembers(memberTarget, memberPage, memberPageSize);
    } catch (err) {
      reportUnexpectedError(err);
    }
  }, [loadMembers, memberForm, memberPage, memberPageSize, memberTarget, reportUnexpectedError, t]);

  const handleRemoveMember = useCallback(
    async (member: TenantMemberRow) => {
      if (!memberTarget) {
        return;
      }
      try {
        await removeTenantMember(memberTarget.id, member.user_id);
        message.success(t('common.deleteSuccess'));
        await loadMembers(memberTarget, memberPage, memberPageSize);
      } catch (err) {
        reportUnexpectedError(err);
      }
    },
    [loadMembers, memberPage, memberPageSize, memberTarget, reportUnexpectedError, t],
  );

  // The server counts these over the whole filtered set. Deriving them from
  // `state.items` (what this page used to do) only ever described the current
  // page, so the cards disagreed with `total` as soon as there was more than
  // one page and archived/deleted rows were invisible to them entirely.
  const summary = state.summary;
  const summaryItems = useMemo(
    () => [
      {
        key: 'total',
        label: t('system.tenant.summary.total'),
        value: summary?.total ?? state.total,
      },
      {
        key: 'active',
        label: t('system.tenant.summary.active'),
        value: summary?.active ?? 0,
      },
      {
        key: 'suspended',
        label: t('system.tenant.summary.suspended'),
        value: summary?.suspended ?? 0,
      },
      {
        key: 'archived',
        label: t('system.tenant.summary.archived'),
        value: summary?.archived ?? 0,
      },
    ],
    [state.total, summary, t],
  );

  const columns = useMemo<ColumnProps<TenantRow>[]>(
    () => [
      {
        title: t('system.tenant.code'),
        dataIndex: 'code',
        width: TABLE_COLUMN_WIDTH.code,
      },
      {
        title: t('system.tenant.name'),
        dataIndex: 'name',
        width: TABLE_COLUMN_WIDTH.name,
      },
      {
        title: t('system.tenant.plan'),
        dataIndex: 'plan',
        width: TABLE_COLUMN_WIDTH.sm,
        render: (value: string) => <Tag color={planTone[value] ?? 'gray'}>{planLabel(value)}</Tag>,
      },
      {
        title: t('system.tenant.status'),
        dataIndex: 'status',
        width: TABLE_COLUMN_WIDTH.status,
        render: (value: string) => (
          <Tag color={statusTone[value] ?? 'gray'}>{statusLabel(value)}</Tag>
        ),
      },
      {
        title: t('system.tenant.createdAt'),
        dataIndex: 'created_at',
        width: TABLE_COLUMN_WIDTH.datetime,
        render: (value) => (
          <Typography.Text className="system-list__datetime-text">
            {formatDateTime(value as string, { withSeconds: true })}
          </Typography.Text>
        ),
      },
      {
        title: t('common.actions'),
        dataIndex: 'operation',
        width: TABLE_ACTION_COLUMN_WIDTH.medium,
        fixed: 'right' as const,
        render: (_, row) => (
          <SystemRowActions
            actions={[
              {
                key: 'members',
                text: t('system.tenant.members'),
                icon: <IconUserGroup />,
                disabled: !canManageMember,
                onClick: () => openMembers(row),
              },
              {
                key: 'edit',
                text: t('common.edit'),
                icon: <IconEdit />,
                disabled: !canUpdate,
                onClick: () => openEdit(row),
              },
              {
                key: 'delete',
                text: t('common.delete'),
                icon: <IconDelete />,
                status: 'danger',
                disabled: !canDelete,
                confirm: {
                  title: t('system.tenant.deleteConfirm', { name: row.name }),
                  onOk: () => {
                    void handleDelete(row);
                  },
                },
              },
            ]}
          />
        ),
      },
    ],
    [
      canDelete,
      canManageMember,
      canUpdate,
      handleDelete,
      openEdit,
      openMembers,
      planLabel,
      statusLabel,
      t,
    ],
  );

  const tableData = state.items;

  const tableContent =
    tableData.length === 0 && !loading ? (
      <PageEmpty description={t('system.tenant.empty')} />
    ) : (
      <AppTable<TenantRow>
        className="system-list__table"
        rowKey="id"
        data={tableData}
        columns={columns}
        loading={loading}
        scroll={{ x: 'max-content' }}
        onChange={handleTableChange as TableProps<TenantRow>['onChange']}
        emptyText={t('system.tenant.empty')}
        pagination={buildStandardPagination(t, {
          current: state.page,
          pageSize: state.size,
          total: state.total,
          onChange: (page, pageSize) => {
            handleTableChange({ current: page, pageSize });
          },
        })}
      />
    );

  // Resolve the body once, as a plain conditional chain: compat guide, first
  // load, request error, table. Reading top-to-bottom keeps the four states
  // legible, which a nested ternary inside the JSX did not.
  let bodyContent: React.ReactNode;
  if (!tenantModeEnabled) {
    bodyContent = (
      <Space direction="vertical" size={16} className="tenant-list__compat-guide">
        <PageEmpty description={t('system.tenant.compat.desc')} />
        <Button type="primary" onClick={() => navigate('/system/setting/platform')}>
          {t('system.tenant.compat.action')}
        </Button>
      </Space>
    );
  } else if (loading && tableData.length === 0) {
    bodyContent = <PageLoading />;
  } else if (error && tableData.length === 0) {
    bodyContent = (
      <PageRequestError
        error={error}
        onRetry={() => {
          void loadData(query);
        }}
      />
    );
  } else {
    bodyContent = tableContent;
  }

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template tenant-list-page">
        <GovernanceSummaryBar
          eyebrow={t('system.tenant.hero.eyebrow')}
          title={t('system.tenant.hero.title')}
          description={t('system.tenant.hero.desc')}
          metrics={summaryItems.map((item) => ({
            key: item.key,
            label: item.label,
            value: item.value,
          }))}
          action={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('system.tenant.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
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
                  <Select.Option value="archived">
                    {t('system.tenant.status.archived')}
                  </Select.Option>
                  <Select.Option value="deleted">{t('system.tenant.status.deleted')}</Select.Option>
                </Select>
                <Select
                  allowClear
                  placeholder={t('system.tenant.filter.plan')}
                  value={query.plan}
                  onChange={(value) => search({ plan: value })}
                >
                  {filterPlanOptions.map((option) => (
                    <Select.Option key={option.code} value={option.code}>
                      {planOptionLabel(option)}
                    </Select.Option>
                  ))}
                </Select>
              </>
            }
            hasActiveFilters={Boolean(query.keyword || query.status || query.plan)}
            onClearAll={reset}
          />
          <Card className="page-panel system-list__table-card">
            <ListHeaderActions
              primary={
                <PermissionAction allowed={canCreate} tooltip={t('common.noPermissionAction')}>
                  <Button type="primary" icon={<IconPlus />} onClick={openCreate}>
                    {t('common.add')}
                  </Button>
                </PermissionAction>
              }
            />
            {bodyContent}
          </Card>
        </>
      </Space>

      <GovernanceInsightDrawer
        title={t('system.tenant.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('system.tenant.hero.sideLead')}
        noteDescription={t('system.tenant.hero.sideDesc')}
      >
        <GovernanceRailSummary items={summaryItems} />
      </GovernanceInsightDrawer>

      <AppModal
        title={editing ? t('system.tenant.edit') : t('system.tenant.create')}
        visible={formVisible}
        size="lg"
        onCancel={() => setFormVisible(false)}
        footer={
          <FormModalFooter
            onCancel={() => setFormVisible(false)}
            onSubmit={() => {
              void handleSubmit();
            }}
            loading={submitting}
            submitText={editing ? t('common.save') : t('common.create')}
          />
        }
        unmountOnExit
      >
        <Form<TenantFormValues> form={form} layout="vertical">
          <FormSection>
            <Row gutter={16}>
              <Col span={12}>
                <FormItem
                  field="code"
                  label={t('system.tenant.code')}
                  rules={[
                    { required: !editing, message: t('system.tenant.codeRequired') },
                    { minLength: 2, message: t('system.tenant.codeLength') },
                    { maxLength: 64, message: t('system.tenant.codeLength') },
                  ]}
                >
                  <Input
                    placeholder={t('system.tenant.codePlaceholder')}
                    disabled={Boolean(editing)}
                  />
                </FormItem>
              </Col>
              <Col span={12}>
                <FormItem
                  field="name"
                  label={t('system.tenant.name')}
                  rules={[
                    { required: true, message: t('system.tenant.nameRequired') },
                    { minLength: 2, message: t('system.tenant.nameLength') },
                    { maxLength: 128, message: t('system.tenant.nameLength') },
                  ]}
                >
                  <Input placeholder={t('system.tenant.namePlaceholder')} />
                </FormItem>
              </Col>
              <Col span={12}>
                <FormItem field="plan" label={t('system.tenant.plan')}>
                  <Select onChange={(value) => setPlanCodeDraft(String(value ?? '').trim())}>
                    {editorOptions.map((option) => (
                      <Select.Option key={option.code} value={option.code}>
                        {planOptionLabel(option)}
                      </Select.Option>
                    ))}
                  </Select>
                </FormItem>
              </Col>
              {/* Initial administrator (create only). Provisioning makes the
                  tenant usable immediately: roles, admin account, owner
                  membership, Casbin policies and a settings copy. Leaving it
                  empty creates a shell tenant nobody can log into. */}
              {!editing ? (
                <>
                  <Col span={12}>
                    <FormItem
                      field="adminUsername"
                      label={t('system.tenant.adminUsername')}
                      extra={t('system.tenant.adminUsernameHint')}
                      rules={[
                        {
                          required: true,
                          message: t('system.tenant.adminUsernameRequired'),
                        },
                        {
                          minLength: 2,
                          message: t('system.tenant.adminUsernameLength'),
                        },
                        {
                          maxLength: 64,
                          message: t('system.tenant.adminUsernameLength'),
                        },
                      ]}
                    >
                      <Input placeholder={t('system.tenant.adminUsernamePlaceholder')} />
                    </FormItem>
                  </Col>
                  <Col span={12}>
                    <FormItem field="adminNickname" label={t('system.tenant.adminNickname')}>
                      <Input placeholder={t('system.tenant.adminNicknamePlaceholder')} />
                    </FormItem>
                  </Col>
                  <Col span={12}>
                    <FormItem
                      field="adminPassword"
                      label={t('system.tenant.adminPassword')}
                      rules={[
                        {
                          validator: (value, callback) => {
                            if (!value) {
                              callback(t('system.tenant.adminPasswordRequired'));
                              return;
                            }
                            if (String(value).length < adminPasswordMinLength) {
                              callback(
                                t('system.tenant.adminPasswordTooShort', {
                                  count: adminPasswordMinLength,
                                }),
                              );
                              return;
                            }
                            callback();
                          },
                        },
                      ]}
                    >
                      <PasswordInput placeholder={t('system.tenant.adminPasswordPlaceholder')} />
                    </FormItem>
                  </Col>
                </>
              ) : null}
              {/* Status is lifecycle-only: a new tenant is always created active
                  (the create DTO has no status field), so the control appears
                  only when editing. */}
              {editing ? (
                <Col span={12}>
                  <FormItem field="status" label={t('system.tenant.status')}>
                    <Select>
                      <Select.Option value="active">
                        {t('system.tenant.status.active')}
                      </Select.Option>
                      <Select.Option value="suspended">
                        {t('system.tenant.status.suspended')}
                      </Select.Option>
                      <Select.Option value="archived">
                        {t('system.tenant.status.archived')}
                      </Select.Option>
                    </Select>
                  </FormItem>
                </Col>
              ) : null}
              <Col span={24}>
                <FormItem field="metadata" label={t('system.tenant.metadata')}>
                  <Input.TextArea placeholder={t('system.tenant.metadataPlaceholder')} />
                </FormItem>
              </Col>
            </Row>
            <TenantPlanMatrix plan={planDraft} />
            {planOutsideLadder ? (
              <Typography.Paragraph type="warning">
                {t('system.tenant.plan.outsideLadder', { plan: planCodeDraft })}
              </Typography.Paragraph>
            ) : null}
          </FormSection>
        </Form>
      </AppModal>

      <AppModal
        title={t('system.tenant.membersTitle', { name: memberTarget?.name ?? '' })}
        visible={memberVisible}
        size="lg"
        footer={null}
        onCancel={closeMembers}
        unmountOnExit
      >
        <Form form={memberForm} layout="inline" className="tenant-member-add-form">
          <FormItem
            field="userId"
            label={t('system.tenant.memberUserId')}
            rules={[{ required: true, message: t('system.tenant.memberUserIdRequired') }]}
          >
            <Input placeholder={t('system.tenant.memberUserIdPlaceholder')} />
          </FormItem>
          <FormItem field="role" label={t('system.tenant.memberRole')} initialValue="member">
            <Select>
              <Select.Option value="owner">{t('system.tenant.role.owner')}</Select.Option>
              <Select.Option value="admin">{t('system.tenant.role.admin')}</Select.Option>
              <Select.Option value="member">{t('system.tenant.role.member')}</Select.Option>
            </Select>
          </FormItem>
          <PermissionAction allowed={canManageMember} tooltip={t('common.noPermissionAction')}>
            <Button type="primary" icon={<IconPlus />} onClick={() => void handleAddMember()}>
              {t('common.add')}
            </Button>
          </PermissionAction>
        </Form>
        <AppTable<TenantMemberRow>
          className="system-list__table"
          rowKey="id"
          data={members}
          loading={memberLoading}
          scroll={{ x: 'max-content' }}
          emptyText={t('system.tenant.membersEmpty')}
          columns={[
            {
              title: t('system.tenant.memberUsername'),
              dataIndex: 'username',
              render: (_, row) => row.nickname || row.username || String(row.user_id),
            },
            {
              title: t('system.tenant.memberRole'),
              dataIndex: 'role',
              render: (value: string) => roleLabel(value),
            },
            {
              title: t('system.tenant.memberJoinedAt'),
              dataIndex: 'created_at',
              render: (value) => formatDateTime(value as string, { withSeconds: true }),
            },
            {
              title: t('common.actions'),
              dataIndex: 'operation',
              width: TABLE_ACTION_COLUMN_WIDTH.compact,
              render: (_, row) => (
                <Popconfirm
                  title={t('system.tenant.memberRemoveConfirm')}
                  onOk={() => {
                    void handleRemoveMember(row);
                  }}
                >
                  <Button status="danger" type="text" disabled={!canManageMember}>
                    {t('system.tenant.memberRemove')}
                  </Button>
                </Popconfirm>
              ),
            },
          ]}
          pagination={buildStandardPagination(t, {
            current: memberPage,
            pageSize: memberPageSize,
            total: memberTotal,
            onChange: (page, pageSize) => {
              if (memberTarget) {
                void loadMembers(memberTarget, page, pageSize);
              }
            },
          })}
        />
      </AppModal>
    </PageContainer>
  );
};

export default TenantList;
