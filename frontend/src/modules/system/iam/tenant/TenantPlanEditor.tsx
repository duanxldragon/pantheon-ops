import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Checkbox,
  Form,
  Grid,
  Input,
  InputNumber,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconEdit, IconPlus } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { isArcoFormValidationError } from '../../../../core/arco/formValidation';
import { isRequestError } from '../../../../api/request';
import { message } from '../../../../components/feedback/message';
import { usePermission } from '../../../../hooks/usePermission';
import {
  AppDrawer,
  AppModal,
  AppTable,
  FormModalFooter,
  FormSection,
  ListHeaderActions,
  PageEmpty,
  PageLoading,
  PageRequestError,
  PermissionAction,
  SystemRowActions,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
} from '../../../../components';
import {
  createPlan,
  getPlanList,
  updatePlan,
  type PlanCreatePayload,
  type PlanRow,
  type PlanStatus,
  type PlanUpdatePayload,
} from './api';
import { UNLIMITED_QUOTA, formatQuotaLimit, isUnlimitedQuota } from './quota';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

interface TenantPlanEditorProps {
  visible: boolean;
  onClose: () => void;
  onSaved: () => void;
}

interface PlanFormValues {
  code: string;
  name?: string;
  status?: PlanStatus;
  sort?: number;
  maxUsers?: number;
  maxRoles?: number;
  maxDepts?: number;
  maxApiKeys?: number;
  maxStorageMb?: number;
}

type QuotaField = 'maxUsers' | 'maxRoles' | 'maxDepts' | 'maxApiKeys' | 'maxStorageMb';

/** Quota limits the console can edit, in the order the form shows them. */
const QUOTA_FIELDS: readonly QuotaField[] = [
  'maxUsers',
  'maxRoles',
  'maxDepts',
  'maxApiKeys',
  'maxStorageMb',
];

const quotaLabelKey: Record<QuotaField, string> = {
  maxUsers: 'system.tenant.planEditor.quota.users',
  maxRoles: 'system.tenant.planEditor.quota.roles',
  maxDepts: 'system.tenant.planEditor.quota.depts',
  maxApiKeys: 'system.tenant.planEditor.quota.apiKeys',
  maxStorageMb: 'system.tenant.planEditor.quota.storage',
};

const planStatusTone: Record<PlanStatus, string> = {
  active: 'green',
  disabled: 'gray',
};

// Matches planCodePattern in tenant_plan_model.go. The server rejects anything
// else with tenant.plan.code_invalid, so the form says the same thing first.
const PLAN_CODE_PATTERN = /^[a-z][a-z0-9_-]{1,31}$/;

/**
 * One ceiling, spoken the way the backend stores it.
 *
 * "No ceiling" is the `-1` sentinel (`tenant.UnlimitedQuota`) while every real
 * limit must be at least 1 (`PlanQuotaReq.resolve` rejects 0). Asking an
 * operator to type `-1` is a contract they can get wrong, so the checkbox owns
 * that value and the number input only ever holds a real ceiling.
 */
const QuotaInput: React.FC<{
  value?: number;
  onChange?: (value?: number) => void;
  unlimitedLabel: string;
}> = ({ value, onChange, unlimitedLabel }) => {
  const unlimited = isUnlimitedQuota(value ?? 0);
  return (
    <Space size={8} className="tenant-plan-editor__quota">
      <InputNumber
        min={1}
        precision={0}
        disabled={unlimited}
        value={unlimited ? undefined : value}
        onChange={(next) => onChange?.(next as number | undefined)}
      />
      <Checkbox
        checked={unlimited}
        onChange={(checked) => onChange?.(checked ? UNLIMITED_QUOTA : undefined)}
      >
        {unlimitedLabel}
      </Checkbox>
    </Space>
  );
};

const TenantPlanEditor: React.FC<TenantPlanEditorProps> = ({ visible, onClose, onSaved }) => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canCreate = isAdmin || hasPerm('system:tenant:plan:create');
  const canUpdate = isAdmin || hasPerm('system:tenant:plan:update');

  const [rows, setRows] = useState<PlanRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const [formVisible, setFormVisible] = useState(false);
  const [editing, setEditing] = useState<PlanRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<PlanFormValues>();

  const unlimitedText = t('system.tenant.overview.quota.unlimited');

  const reportUnexpectedError = useCallback(
    (err: unknown) => {
      if (isArcoFormValidationError(err) || isRequestError(err)) {
        return;
      }
      message.error(t('common.actionFailed'));
    },
    [t],
  );

  const loadPlans = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await getPlanList();
      setRows(resp.items ?? []);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!visible) {
      return;
    }
    const timer = globalThis.setTimeout(() => {
      void loadPlans();
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadPlans, visible]);

  const openCreate = useCallback(() => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ status: 'active', sort: (rows.length + 1) * 10 });
    setFormVisible(true);
  }, [form, rows.length]);

  const openEdit = useCallback(
    (row: PlanRow) => {
      setEditing(row);
      form.setFieldsValue({
        code: row.code,
        name: row.name,
        status: row.status,
        sort: row.sort,
        maxUsers: row.maxUsers,
        maxRoles: row.maxRoles,
        maxDepts: row.maxDepts,
        maxApiKeys: row.maxApiKeys,
        maxStorageMb: row.maxStorageMb,
      });
      setFormVisible(true);
    },
    [form],
  );

  const handleSubmit = useCallback(async () => {
    try {
      const values = await form.validate();
      setSubmitting(true);
      // `resolve(requireAll)` demands all five ceilings whenever any is
      // present, so the form always sends the whole block instead of a partial
      // edit that the server would reject.
      const quota = {
        maxUsers: values.maxUsers,
        maxRoles: values.maxRoles,
        maxDepts: values.maxDepts,
        maxApiKeys: values.maxApiKeys,
        maxStorageMb: values.maxStorageMb,
      };
      if (editing) {
        const payload: PlanUpdatePayload = {
          name: values.name?.trim(),
          status: values.status,
          sort: values.sort ?? 0,
          ...quota,
        };
        await updatePlan(editing.code, payload);
        message.success(t('common.updateSuccess'));
      } else {
        const payload: PlanCreatePayload = {
          code: values.code.trim(),
          name: values.name?.trim(),
          status: values.status,
          sort: values.sort ?? 0,
          ...quota,
        };
        await createPlan(payload);
        message.success(t('common.createSuccess'));
      }
      setFormVisible(false);
      await loadPlans();
      onSaved();
    } catch (err) {
      reportUnexpectedError(err);
    } finally {
      setSubmitting(false);
    }
  }, [editing, form, loadPlans, onSaved, reportUnexpectedError, t]);

  const columns = useMemo<ColumnProps<PlanRow>[]>(
    () => [
      {
        title: t('system.tenant.planEditor.code'),
        dataIndex: 'code',
        width: TABLE_COLUMN_WIDTH.code,
      },
      {
        title: t('system.tenant.planEditor.name'),
        dataIndex: 'name',
        width: TABLE_COLUMN_WIDTH.name,
      },
      {
        title: t('system.tenant.status'),
        dataIndex: 'status',
        width: TABLE_COLUMN_WIDTH.status,
        render: (value: PlanStatus) => (
          <Tag color={planStatusTone[value] ?? 'gray'}>
            {t(`system.tenant.planEditor.status.${value}`, { defaultValue: value })}
          </Tag>
        ),
      },
      {
        title: t('system.tenant.planEditor.tenants'),
        dataIndex: 'tenantCount',
        width: TABLE_COLUMN_WIDTH.count,
      },
      {
        title: t('system.tenant.planEditor.quota.users'),
        dataIndex: 'maxUsers',
        width: TABLE_COLUMN_WIDTH.sm,
        render: (value: number) => formatQuotaLimit(value, unlimitedText),
      },
      {
        title: t('system.tenant.planEditor.quota.roles'),
        dataIndex: 'maxRoles',
        width: TABLE_COLUMN_WIDTH.sm,
        render: (value: number) => formatQuotaLimit(value, unlimitedText),
      },
      {
        title: t('system.tenant.planEditor.quota.depts'),
        dataIndex: 'maxDepts',
        width: TABLE_COLUMN_WIDTH.sm,
        render: (value: number) => formatQuotaLimit(value, unlimitedText),
      },
      {
        title: t('system.tenant.planEditor.sort'),
        dataIndex: 'sort',
        width: TABLE_COLUMN_WIDTH.count,
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
                key: 'edit',
                text: t('common.edit'),
                icon: <IconEdit />,
                disabled: !canUpdate,
                onClick: () => openEdit(row),
              },
            ]}
          />
        ),
      },
    ],
    [canUpdate, openEdit, t, unlimitedText],
  );

  let body: React.ReactNode;
  if (loading && rows.length === 0) {
    body = <PageLoading />;
  } else if (error && rows.length === 0) {
    body = (
      <PageRequestError
        error={error}
        onRetry={() => {
          void loadPlans();
        }}
      />
    );
  } else if (rows.length === 0) {
    body = <PageEmpty description={t('system.tenant.planEditor.empty')} />;
  } else {
    body = (
      <AppTable<PlanRow>
        className="system-list__table"
        rowKey="code"
        data={rows}
        columns={columns}
        loading={loading}
        scroll={{ x: 'max-content' }}
        pagination={false}
        emptyText={t('system.tenant.planEditor.empty')}
      />
    );
  }

  return (
    <AppDrawer
      title={t('system.tenant.planEditor.title')}
      visible={visible}
      size="xl"
      onCancel={onClose}
      footer={null}
    >
      <Space direction="vertical" size={16} className="tenant-plan-editor">
        <Typography.Text type="secondary">
          {t('system.tenant.planEditor.description')}
        </Typography.Text>
        <ListHeaderActions
          primary={
            <PermissionAction allowed={canCreate} tooltip={t('common.noPermissionAction')}>
              <Button type="primary" icon={<IconPlus />} onClick={openCreate}>
                {t('system.tenant.planEditor.create')}
              </Button>
            </PermissionAction>
          }
        />
        {body}
      </Space>

      <AppModal
        title={editing ? t('system.tenant.planEditor.edit') : t('system.tenant.planEditor.create')}
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
        <Form<PlanFormValues> form={form} layout="vertical">
          <FormSection>
            <Row gutter={16}>
              <Col span={12}>
                <FormItem
                  field="code"
                  label={t('system.tenant.planEditor.code')}
                  rules={[
                    { required: true, message: t('system.tenant.planEditor.codeRequired') },
                    {
                      validator: (value, callback) => {
                        if (value && !PLAN_CODE_PATTERN.test(String(value).trim())) {
                          callback(t('system.tenant.planEditor.codePattern'));
                          return;
                        }
                        callback();
                      },
                    },
                  ]}
                >
                  <Input
                    placeholder={t('system.tenant.planEditor.codePlaceholder')}
                    disabled={Boolean(editing)}
                  />
                </FormItem>
              </Col>
              <Col span={12}>
                <FormItem
                  field="name"
                  label={t('system.tenant.planEditor.name')}
                  rules={[{ required: true, message: t('system.tenant.planEditor.nameRequired') }]}
                >
                  <Input placeholder={t('system.tenant.planEditor.namePlaceholder')} />
                </FormItem>
              </Col>
              <Col span={12}>
                <FormItem field="status" label={t('system.tenant.status')}>
                  <Select>
                    <Select.Option value="active">
                      {t('system.tenant.planEditor.status.active')}
                    </Select.Option>
                    <Select.Option value="disabled">
                      {t('system.tenant.planEditor.status.disabled')}
                    </Select.Option>
                  </Select>
                </FormItem>
              </Col>
              <Col span={12}>
                <FormItem field="sort" label={t('system.tenant.planEditor.sort')}>
                  <InputNumber min={0} max={9999} precision={0} />
                </FormItem>
              </Col>
            </Row>
          </FormSection>

          <FormSection
            title={t('system.tenant.planEditor.quotaSection')}
            description={t('system.tenant.planEditor.quotaHint')}
          >
            <Row gutter={16}>
              {QUOTA_FIELDS.map((field) => (
                <Col key={field} span={12}>
                  <FormItem
                    field={field}
                    label={t(quotaLabelKey[field])}
                    rules={[
                      { required: true, message: t('system.tenant.planEditor.quotaRequired') },
                    ]}
                  >
                    <QuotaInput unlimitedLabel={t('system.tenant.planEditor.unlimited')} />
                  </FormItem>
                </Col>
              ))}
            </Row>
          </FormSection>
        </Form>
      </AppModal>
    </AppDrawer>
  );
};

export default TenantPlanEditor;
