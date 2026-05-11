import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Message,
  Select,
  Space,
  Switch,
  Tabs,
  Tag,
  Typography,
} from '@arco-design/web-react';
import type { PaginationProps } from '@arco-design/web-react/es/Pagination/interface';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { IconRefresh } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import {
  AppTable,
  FormSection,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  PageActions,
  PageContainer,
  PageEmpty,
  PageError,
  PageHeader,
  PageLoading,
  PageNetworkError,
  PageServerError,
  SubmitBar,
  TABLE_ACTION_COLUMN_WIDTH,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../../components';
import { formatDateTime } from '../../../core/format/dateTime';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import { invalidateRouteWarmData, resolveRouteWarmData } from '../../../core/router/prefetch';
import {
  applyPantheonDefaultTheme,
  clearPantheonThemePreference,
  pantheonThemeOptions,
  type PantheonThemeKey,
} from '../../../core/theme/theme';
import {
  hasExplicitLanguagePreference,
  refreshPublicSettings,
} from '../../../core/settings/publicSettings';
import { usePermission } from '../../../hooks/usePermission';
import { SUPPORTED_LOCALES, switchI18nLanguage } from '../../../i18n';
import {
  exportSettingAudit,
  getSettingOverview,
  updateSettingGroup,
  getSettingAuditList,
  getSettingList,
  refreshSettingCache,
  type SettingAuditChange,
  type SettingAuditRow,
  type SettingItem,
  type SettingOverviewResp,
} from './api';
import '../../../core/styles/list-page.css';

const FormItem = Form.Item;

const groupOrder = ['basic', 'platform', 'security', 'login', 'audit', 'upload', 'i18n', 'ui'];
const defaultAuditPageSize = 5;
const auditRetentionSettingKeys = new Set([
  'audit.login_log_retention_options',
  'audit.operation_log_retention_options',
  'audit.session_cleanup_retention_options',
]);
const auditRetentionDaySettingKeys = new Set([
  'audit.login_log_retention_days',
  'audit.operation_log_retention_days',
  'audit.session_retention_days',
]);
const integerSettingKeys = new Set([
  'security.password_min_length',
  'login.max_failed_attempts',
  'login.lock_minutes',
  'login.session_idle_minutes',
  'upload.max_file_size',
]);
const recommendedAuditRetentionOptions = [1, 7, 30, 90, 180, 365];
const recommendedAuditRetentionDayOptions = [30, 90, 180, 365];

type SettingFormValue = string | number | boolean | string[] | undefined;

function normalizeAuditRetentionTagValues(rawValues: Array<string | number>) {
  const normalized = Array.from(
    new Set(
      rawValues
        .map((item) => Number(String(item).trim()))
        .filter((item) => Number.isInteger(item) && item > 0 && item <= 365),
    ),
  ).sort((left, right) => left - right);
  return normalized.map((item) => String(item));
}

function parseAuditRetentionSettingValue(rawValue: string) {
  try {
    const parsed = JSON.parse(rawValue) as unknown;
    if (!Array.isArray(parsed)) {
      return [];
    }
    return normalizeAuditRetentionTagValues(parsed as Array<string | number>);
  } catch {
    return [];
  }
}

function resolveAuditRetentionDefaultValues(item: SettingItem) {
  const parsedDefaults = parseAuditRetentionSettingValue(item.defaultValue);
  if (parsedDefaults.length > 0) {
    return parsedDefaults;
  }
  return ['1', '7', '30'];
}

function parseDefaultFieldValue(item: SettingItem): SettingFormValue {
  if (auditRetentionSettingKeys.has(item.settingKey)) {
    return resolveAuditRetentionDefaultValues(item);
  }
  if (item.valueType === 'number') {
    return item.defaultValue === '' ? undefined : Number(item.defaultValue);
  }
  if (item.valueType === 'boolean') {
    return item.defaultValue === 'true';
  }
  return item.defaultValue;
}

function isSameStringArray(left: string[], right: string[]) {
  if (left.length !== right.length) {
    return false;
  }
  return left.every((value, index) => value === right[index]);
}

function buildFormValues(items: SettingItem[]) {
  return items.reduce<Record<string, SettingFormValue>>((acc, item) => {
    if (item.isEncrypted === 1) {
      acc[item.settingKey] = '';
      return acc;
    }
    if (auditRetentionSettingKeys.has(item.settingKey)) {
      acc[item.settingKey] = parseAuditRetentionSettingValue(item.settingValue);
      return acc;
    }
    if (item.valueType === 'number') {
      acc[item.settingKey] = item.settingValue === '' ? undefined : Number(item.settingValue);
    } else if (item.valueType === 'boolean') {
      acc[item.settingKey] = item.settingValue === 'true';
    } else {
      acc[item.settingKey] = item.settingValue;
    }
    return acc;
  }, {});
}

function serializeSettingValue(item: SettingItem, rawValue: SettingFormValue) {
  if (auditRetentionSettingKeys.has(item.settingKey)) {
    const normalized = normalizeAuditRetentionTagValues(Array.isArray(rawValue) ? rawValue : []);
    if (normalized.length === 0) {
      throw new Error('setting.value.invalid_audit_retention');
    }
    return JSON.stringify(normalized.map((option) => Number(option)));
  }
  if (item.valueType === 'boolean') {
    return String(Boolean(rawValue));
  }
  if (item.valueType === 'number') {
    if (typeof rawValue === 'number') {
      if (!Number.isFinite(rawValue)) {
        throw new Error('setting.value.invalid_number');
      }
      return String(rawValue);
    }
    if (
      typeof rawValue === 'string' &&
      rawValue.trim() !== '' &&
      Number.isFinite(Number(rawValue.trim()))
    ) {
      return rawValue.trim();
    }
    throw new Error('setting.value.invalid_number');
  }
  return String(rawValue ?? '');
}

function readFormFieldValue(
  values: Record<string, unknown>,
  fieldKey: string,
  fallbackReader: (name: string) => unknown,
) {
  const directValue = fallbackReader(fieldKey);
  if (directValue !== undefined) {
    return directValue as SettingFormValue;
  }

  if (Object.prototype.hasOwnProperty.call(values, fieldKey)) {
    return values[fieldKey] as SettingFormValue;
  }

  return fieldKey.split('.').reduce<unknown>((current, segment) => {
    if (current && typeof current === 'object' && segment in (current as Record<string, unknown>)) {
      return (current as Record<string, unknown>)[segment];
    }
    return undefined;
  }, values) as SettingFormValue;
}

function buildRequiredFieldMessage(t: ReturnType<typeof useTranslation>['t'], field: string) {
  return t('common.requiredField', { field });
}

const SettingPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { isAdmin, hasPerm } = usePermission();
  const canUpdateSetting = isAdmin || hasPerm('system:setting:update');
  const canRefreshCache = isAdmin || hasPerm('system:setting:refresh');
  const canExportAudit = isAdmin || hasPerm('system:setting:export');
  const canViewOperationLog = isAdmin || hasPerm('system:operation-log:list');
  const [loading, setLoading] = useState(false);
  const [submittingGroup, setSubmittingGroup] = useState<string | null>(null);
  const [refreshingCache, setRefreshingCache] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [settings, setSettings] = useState<SettingItem[]>([]);
  const [overview, setOverview] = useState<SettingOverviewResp | null>(null);
  const [activeGroup, setActiveGroup] = useState(groupOrder[0]);
  const [auditRows, setAuditRows] = useState<SettingAuditRow[]>([]);
  const [auditTotal, setAuditTotal] = useState(0);
  const [auditLoading, setAuditLoading] = useState(false);
  const [auditQuery, setAuditQuery] = useState({ page: 1, pageSize: defaultAuditPageSize });
  const [form] = Form.useForm<Record<string, SettingFormValue>>();
  const governanceRail = useGovernanceRail();

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const rows = await resolveRouteWarmData('/system/setting', 'list:default', () =>
        getSettingList(),
      );
      setSettings(rows);
      try {
        const overviewResp = await resolveRouteWarmData('/system/setting', 'overview', () =>
          getSettingOverview(),
        );
        setOverview(overviewResp);
      } catch {
        setOverview(null);
      }
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [loadData]);

  const loadAudit = useCallback(
    async (groupKey: string, page = 1, pageSize = defaultAuditPageSize) => {
      setAuditLoading(true);
      try {
        const result = await getSettingAuditList({ groupKey, page, pageSize });
        setAuditRows(result.items);
        setAuditTotal(result.total);
        setAuditQuery({
          page: result.page || page,
          pageSize: result.pageSize || pageSize,
        });
      } catch {
        Message.error(t('common.loadFailed'));
      } finally {
        setAuditLoading(false);
      }
    },
    [t],
  );

  const groupedSettings = useMemo(() => {
    const buckets = new Map<string, SettingItem[]>();
    settings.forEach((item) => {
      const list = buckets.get(item.groupKey) || [];
      list.push(item);
      buckets.set(item.groupKey, list);
    });
    return groupOrder
      .filter((groupKey) => buckets.has(groupKey))
      .map((groupKey) => ({ groupKey, items: buckets.get(groupKey) || [] }));
  }, [settings]);

  const activeSettingGroup =
    groupedSettings.find((item) => item.groupKey === activeGroup) || groupedSettings[0];
  const activeGroupKey = activeSettingGroup?.groupKey;

  useRefreshSubscription('system:setting:changed', () => {
    invalidateRouteWarmData('/system/setting', ['list:default', 'overview']);
    void loadData();
    if (activeGroupKey && canViewOperationLog) {
      void loadAudit(activeGroupKey, auditQuery.page, auditQuery.pageSize);
    }
  });

  useEffect(() => {
    if (!activeSettingGroup) {
      return;
    }
    form.setFieldsValue(buildFormValues(activeSettingGroup.items));
  }, [activeSettingGroup, form]);

  useEffect(() => {
    if (!activeGroupKey || !canViewOperationLog) {
      return;
    }
    const timer = window.setTimeout(() => {
      void loadAudit(activeGroupKey, 1, defaultAuditPageSize);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [activeGroupKey, canViewOperationLog, loadAudit]);

  const resetActiveGroupValues = () => {
    if (!activeSettingGroup) {
      return;
    }
    form.setFieldsValue(buildFormValues(activeSettingGroup.items));
  };

  const handleSubmit = async () => {
    const group = activeSettingGroup;
    if (!group || !canUpdateSetting) {
      return;
    }
    const fieldNames = group.items.map((item) => item.settingKey);
    let values;
    try {
      values = await form.validate(fieldNames);
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      throw error;
    }
    try {
      const items = group.items.map((item) => ({
        settingKey: item.settingKey,
        settingValue: serializeSettingValue(
          item,
          readFormFieldValue(values as Record<string, unknown>, item.settingKey, (name) =>
            form.getFieldValue(name),
          ),
        ),
      }));
      setSubmittingGroup(group.groupKey);
      await updateSettingGroup(group.groupKey, {
        items,
      });
      if (group.groupKey === 'ui') {
        const nextTheme = values['ui.default_theme'];
        if (typeof nextTheme === 'string') {
          applyPantheonDefaultTheme(nextTheme as PantheonThemeKey);
          clearPantheonThemePreference();
        }
      }
      if (['basic', 'platform', 'i18n', 'ui'].includes(group.groupKey)) {
        const publicSettings = await refreshPublicSettings().catch(() => null);
        if (group.groupKey === 'i18n' && publicSettings && !hasExplicitLanguagePreference()) {
          await switchI18nLanguage(publicSettings.defaultLanguage).catch(() => undefined);
        }
      }
      Message.success(t('common.updateSuccess'));
      invalidateRouteWarmData('/system/setting', ['list:default', 'overview']);
      publishRefresh('system:setting:changed', 'system/setting');
      await loadData();
      await loadAudit(group.groupKey, 1, auditQuery.pageSize);
    } catch (submitError) {
      if (submitError instanceof Error && submitError.message === 'setting.value.invalid_number') {
        Message.error(t('setting.value.invalid_number'));
      } else if (
        submitError instanceof Error &&
        submitError.message === 'setting.value.invalid_audit_retention'
      ) {
        Message.error(t('system.setting.audit.retentionRequired'));
      }
    } finally {
      setSubmittingGroup(null);
    }
  };

  const handleRefreshCache = async () => {
    if (!activeGroupKey) {
      return;
    }
    setRefreshingCache(true);
    try {
      await refreshSettingCache({ groupKeys: [activeGroupKey] });
      Message.success(t('system.setting.cache.refreshSuccess'));
      invalidateRouteWarmData('/system/setting', ['list:default', 'overview']);
      publishRefresh('system:setting:changed', 'system/setting');
      await loadData();
      await loadAudit(activeGroupKey, 1, auditQuery.pageSize);
    } finally {
      setRefreshingCache(false);
    }
  };

  const formatDefaultValueLabel = (item: SettingItem) => {
    if (auditRetentionSettingKeys.has(item.settingKey)) {
      const defaultValues = resolveAuditRetentionDefaultValues(item);
      return defaultValues
        .map((option) => t('common.keepRecentDays', { count: Number(option) }))
        .join(' / ');
    }
    if (item.settingKey === 'upload.storage_driver') {
      return t(
        `system.setting.option.upload.storage_driver.${item.defaultValue}`,
        item.defaultValue,
      );
    }
    if (item.settingKey === 'i18n.default_language') {
      return t(`app.language.${item.defaultValue}`, item.defaultValue);
    }
    if (item.settingKey === 'ui.default_theme') {
      const matchedTheme = pantheonThemeOptions.find((theme) => theme.key === item.defaultValue);
      return matchedTheme ? t(matchedTheme.labelKey) : item.defaultValue;
    }
    if (item.settingKey === 'platform.app_mode') {
      return t(`system.setting.option.platform.app_mode.${item.defaultValue}`, item.defaultValue);
    }
    if (item.valueType === 'boolean') {
      return item.defaultValue === 'true' ? t('common.yes') : t('common.no');
    }
    if (item.defaultValue === '') {
      return t('system.setting.defaultValueEmpty');
    }
    return item.defaultValue;
  };

  const renderField = (item: SettingItem) => {
    const label = t(`system.setting.item.${item.settingKey}`, item.settingKey);
    const isWideField =
      auditRetentionSettingKeys.has(item.settingKey) ||
      item.valueType === 'json' ||
      item.isEncrypted === 1;
    const fieldClassName = isWideField
      ? 'setting-page__field setting-page__field--full'
      : 'setting-page__field';
    const remark = t(item.remark, '');
    const help = (
      <Space direction="vertical" size={4}>
        {remark ? <span>{remark}</span> : null}
        {item.isEncrypted === 1 ? (
          <Space size={8} wrap>
            <Tag color="red">{t('system.setting.encrypted')}</Tag>
            <Typography.Text type="secondary">
              {item.hasValue === 1
                ? t('system.setting.leaveEmptyToKeep')
                : t('system.setting.encryptedEmptyHint')}
            </Typography.Text>
          </Space>
        ) : (
          <Space size={12} wrap>
            <Button
              type="text"
              size="small"
              onClick={() => {
                form.setFieldValue(item.settingKey, parseDefaultFieldValue(item));
              }}
            >
              {t('system.setting.restoreDefault')}
            </Button>
            <Typography.Text type="secondary">
              {t('system.setting.defaultValueHint', { value: formatDefaultValueLabel(item) })}
            </Typography.Text>
          </Space>
        )}
      </Space>
    );

    if (item.settingKey === 'ui.default_theme') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Select
            options={pantheonThemeOptions.map((theme) => ({
              label: `${t(theme.labelKey)} · ${t(theme.descriptionKey)}`,
              value: theme.key,
            }))}
          />
        </FormItem>
      );
    }
    if (item.settingKey === 'platform.app_mode') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Select
            options={[
              {
                label: t('system.setting.option.platform.app_mode.enterprise'),
                value: 'enterprise',
              },
              { label: t('system.setting.option.platform.app_mode.consumer'), value: 'consumer' },
              { label: t('system.setting.option.platform.app_mode.hybrid'), value: 'hybrid' },
            ]}
          />
        </FormItem>
      );
    }
    if (item.settingKey === 'upload.storage_driver') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Select
            options={[
              { label: t('system.setting.option.upload.storage_driver.local'), value: 'local' },
              { label: t('system.setting.option.upload.storage_driver.s3'), value: 's3' },
            ]}
          />
        </FormItem>
      );
    }
    if (item.settingKey === 'i18n.default_language') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Select
            options={SUPPORTED_LOCALES.map((locale) => ({
              label: t(`app.language.${locale}`),
              value: locale,
            }))}
          />
        </FormItem>
      );
    }
    if (auditRetentionSettingKeys.has(item.settingKey)) {
      const savedValues = parseAuditRetentionSettingValue(item.settingValue);
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
          rules={[
            {
              required: true,
              type: 'array',
              minLength: 1,
              message: t('system.setting.audit.retentionRequired'),
            },
            {
              validator: (_value, callback) => {
                const currentValue = form.getFieldValue(item.settingKey);
                const normalized = normalizeAuditRetentionTagValues(
                  Array.isArray(currentValue) ? currentValue : [],
                );
                if (normalized.length === 0) {
                  callback(t('system.setting.audit.retentionRequired'));
                  return;
                }
                if (normalized.length !== (Array.isArray(currentValue) ? currentValue.length : 0)) {
                  callback(t('system.setting.audit.retentionInvalid'));
                  return;
                }
                callback();
              },
            },
          ]}
        >
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            <Select
              mode="multiple"
              allowCreate
              placeholder={t('system.setting.audit.retentionPlaceholder')}
              options={recommendedAuditRetentionOptions.map((option) => ({
                label: t('common.keepRecentDays', { count: option }),
                value: String(option),
              }))}
              onChange={(value) => {
                form.setFieldValue(
                  item.settingKey,
                  normalizeAuditRetentionTagValues(value as Array<string | number>),
                );
              }}
            />
            <FormItem noStyle shouldUpdate>
              {() => {
                const currentValue = normalizeAuditRetentionTagValues(
                  Array.isArray(form.getFieldValue(item.settingKey))
                    ? (form.getFieldValue(item.settingKey) as Array<string | number>)
                    : [],
                );
                const dirty = !isSameStringArray(currentValue, savedValues);
                if (!dirty) {
                  return null;
                }
                return <Tag color="orange">{t('system.setting.audit.unsavedChanges')}</Tag>;
              }}
            </FormItem>
          </Space>
        </FormItem>
      );
    }

    if (item.valueType === 'boolean') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
          triggerPropName="checked"
        >
          <Switch checkedText={t('common.yes')} uncheckedText={t('common.no')} />
        </FormItem>
      );
    }
    if (item.valueType === 'number') {
      if (auditRetentionDaySettingKeys.has(item.settingKey)) {
        return (
          <FormItem
            key={item.settingKey}
            className={fieldClassName}
            field={item.settingKey}
            label={label}
            extra={help}
            rules={[{ required: true, message: buildRequiredFieldMessage(t, label) }]}
          >
            <Select
              options={recommendedAuditRetentionDayOptions.map((option) => ({
                label: t('system.setting.audit.retentionDaysOption', { count: option }),
                value: option,
              }))}
            />
          </FormItem>
        );
      }
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
          rules={[{ required: true, message: buildRequiredFieldMessage(t, label) }]}
        >
          <InputNumber
            style={{ width: '100%' }}
            precision={integerSettingKeys.has(item.settingKey) ? 0 : undefined}
            min={integerSettingKeys.has(item.settingKey) ? 1 : undefined}
          />
        </FormItem>
      );
    }
    if (item.isEncrypted === 1) {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Input.Password
            placeholder={
              item.hasValue === 1
                ? t('system.setting.leaveEmptyToKeep')
                : t('system.setting.encryptedPlaceholder')
            }
            onPressEnter={() => form.submit()}
          />
        </FormItem>
      );
    }
    if (item.valueType === 'json') {
      return (
        <FormItem
          key={item.settingKey}
          className={fieldClassName}
          field={item.settingKey}
          label={label}
          extra={help}
        >
          <Input.TextArea autoSize={{ minRows: 4, maxRows: 10 }} />
        </FormItem>
      );
    }
    return (
      <FormItem
        key={item.settingKey}
        className={fieldClassName}
        field={item.settingKey}
        label={label}
        extra={help}
      >
        <Input onPressEnter={() => form.submit()} />
      </FormItem>
    );
  };

  const renderErrorState = () => {
    if (isNetworkRequestError(error)) {
      return (
        <PageNetworkError
          timeout={isTimeoutRequestError(error)}
          onRetry={() => {
            void loadData();
          }}
        />
      );
    }
    if (isServerRequestError(error)) {
      return (
        <PageServerError
          onRetry={() => {
            void loadData();
          }}
        />
      );
    }
    return (
      <PageError
        onRetry={() => {
          void loadData();
        }}
      />
    );
  };

  const renderAuditChange = (change: SettingAuditChange) => {
    const label = t(`system.setting.item.${change.settingKey}`, change.settingKey);
    if (change.isEncrypted === 1) {
      return (
        <Space size={6} wrap>
          <Typography.Text>{label}</Typography.Text>
          <Tag color="red">{t('system.setting.audit.sensitiveChanged')}</Tag>
        </Space>
      );
    }
    const oldValue = change.oldValue || '-';
    const newValue = change.newValue || '-';
    return (
      <Space size={6} wrap>
        <Typography.Text>{label}</Typography.Text>
        <Typography.Text type="secondary">{oldValue}</Typography.Text>
        <Typography.Text type="secondary">→</Typography.Text>
        <Typography.Text>{newValue}</Typography.Text>
      </Space>
    );
  };

  const auditColumns: ColumnProps<SettingAuditRow>[] = [
    {
      title: t('system.setting.audit.operator'),
      dataIndex: 'operName',
      render: (value: string) => value || '-',
    },
    withTableColumnPriority(
      {
        title: t('system.setting.audit.ip'),
        dataIndex: 'operIp',
        render: (value: string) => value || '-',
      },
      'medium',
    ),
    {
      title: t('system.setting.audit.changes'),
      dataIndex: 'changes',
      render: (changes: SettingAuditChange[]) => (
        <Space direction="vertical" size={4}>
          {changes.length > 0 ? (
            changes.map((change) => <div key={change.settingKey}>{renderAuditChange(change)}</div>)
          ) : (
            <Typography.Text type="secondary">{t('system.setting.audit.noChange')}</Typography.Text>
          )}
        </Space>
      ),
    },
    {
      title: t('system.setting.audit.status'),
      dataIndex: 'status',
      render: (value: number) =>
        value === 1 ? (
          <Tag color="green">{t('auth.loginLog.status.success')}</Tag>
        ) : (
          <Tag color="red">{t('auth.loginLog.status.failed')}</Tag>
        ),
    },
    {
      title: t('system.setting.audit.operTime'),
      dataIndex: 'operTime',
      render: (value: string) => formatDateTime(value),
    },
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.single,
      render: (_, record) => (
        <Button
          type="text"
          size="small"
          disabled={!canViewOperationLog}
          onClick={() => navigate(`/system/operation-log?detailId=${record.id}`)}
        >
          {t('system.setting.audit.viewOperationLog')}
        </Button>
      ),
    },
  ];

  const handleAuditTableChange: TableProps<SettingAuditRow>['onChange'] = (pagination) => {
    if (!activeGroupKey) {
      return;
    }
    void loadAudit(
      activeGroupKey,
      pagination.current || 1,
      pagination.pageSize || auditQuery.pageSize,
    );
  };

  const handleExportAudit = async () => {
    if (!activeGroupKey) {
      return;
    }
    await exportSettingAudit({ groupKey: activeGroupKey });
  };

  const heroStats = overview
    ? [
        {
          key: 'total',
          label: t('system.setting.overview.totalSettings'),
          value: overview.totalSettingCount,
          hint: t('system.setting.hero.totalHint'),
        },
        {
          key: 'public',
          label: t('system.setting.overview.publicSettings'),
          value: overview.publicSettingCount,
          hint: t('system.setting.hero.publicHint'),
        },
        {
          key: 'risk',
          label: t('system.setting.overview.risks'),
          value: overview.riskCount,
          hint: t('system.setting.hero.riskHint'),
        },
      ]
    : [];

  const governanceSummaryItems = overview
    ? [
        {
          label: t('system.setting.overview.runtime'),
          value: t(
            `system.setting.option.upload.storage_driver.${overview.storageDriver}`,
            overview.storageDriver,
          ),
          description: t('system.setting.hero.storageHint'),
        },
        {
          label: t('system.setting.item.i18n.default_language'),
          value: t(`app.language.${overview.defaultLanguage}`, overview.defaultLanguage),
          description: t('system.setting.hero.languageHint'),
        },
        {
          label: t('system.setting.item.ui.default_theme'),
          value: overview.defaultTheme,
          description: t('system.setting.hero.themeHint'),
        },
        ...overview.issues.slice(0, 4).map((issue) => ({
          tone: issue.severity === 'critical' ? ('danger' as const) : ('warning' as const),
          label: (
            <Tag color={issue.severity === 'critical' ? 'red' : 'orange'}>
              {t(`system.setting.overview.severity.${issue.severity}`)}
            </Tag>
          ),
          value: t(`system.setting.item.${issue.settingKey}`, issue.settingKey),
          description: t(issue.reasonKey),
        })),
      ]
    : [];

  return (
    <PageContainer>
      <PageHeader
        title={t('system.menu.setting')}
        extra={
          overview ? (
            <PageActions>
              <GovernanceRailToggleButton
                expanded={governanceRail.expanded}
                onToggle={governanceRail.toggle}
              >
                {t('system.setting.hero.summaryTitle')}
              </GovernanceRailToggleButton>
            </PageActions>
          ) : undefined
        }
      />
      <Space direction="vertical" size={12} className="system-page-template setting-page">
        {overview ? (
          <Card className="page-panel system-page-hero system-list__hero setting-page__hero">
            <div className="system-page-hero__top">
              <div className="system-page-hero__copy">
                <span className="system-page-hero__eyebrow">
                  {t('system.setting.hero.eyebrow')}
                </span>
                <Typography.Title heading={5} className="system-page-hero__title">
                  {t('system.setting.hero.title')}
                </Typography.Title>
                <Typography.Paragraph type="secondary" className="system-page-hero__desc">
                  {t('system.setting.hero.desc')}
                </Typography.Paragraph>
              </div>
            </div>
            <div className="system-page-kpi-grid">
              {heroStats.map((item) => (
                <div key={item.key} className="system-page-kpi">
                  <span className="system-page-kpi__label">{item.label}</span>
                  <span className="system-page-kpi__value">{item.value}</span>
                  <span className="system-page-kpi__hint">{item.hint}</span>
                </div>
              ))}
            </div>
          </Card>
        ) : (
          <PageHeader
            subtitle={t('system.setting.hero.title')}
            className="setting-page__fallback-header"
          />
        )}
        {loading && settings.length === 0 ? <PageLoading /> : null}
        {error && settings.length === 0 ? renderErrorState() : null}
        {!loading && !error && settings.length === 0 ? (
          <PageEmpty description={t('system.setting.empty')} />
        ) : null}
        {settings.length > 0 ? (
          <>
            <Card className="page-panel setting-page__config-card">
              <Tabs
                type="rounded"
                activeTab={activeSettingGroup?.groupKey}
                onChange={setActiveGroup}
              >
                {groupedSettings.map((group) => (
                  <Tabs.TabPane
                    key={group.groupKey}
                    title={t(`system.setting.group.${group.groupKey}`)}
                  />
                ))}
              </Tabs>
              {activeSettingGroup ? (
                <Form
                  form={form}
                  layout="vertical"
                  onSubmit={() => {
                    void handleSubmit();
                  }}
                >
                  <Space
                    direction="vertical"
                    size={10}
                    className="dialog-form-stack setting-page__form-stack"
                    style={{ marginTop: 10 }}
                  >
                    <FormSection
                      title={t(`system.setting.group.${activeSettingGroup.groupKey}`)}
                      description={t(`system.setting.groupHint.${activeSettingGroup.groupKey}`, '')}
                    >
                      <div className="setting-page__field-grid">
                        {activeSettingGroup.items.map(renderField)}
                      </div>
                    </FormSection>
                    <Typography.Text type="secondary" className="setting-page__save-hint">
                      {t('system.setting.saveHint')}
                    </Typography.Text>
                    <div className="setting-page__actions">
                      <Space className="setting-page__meta-actions">
                        <Button
                          size="small"
                          icon={<IconRefresh />}
                          loading={refreshingCache}
                          onClick={() => {
                            void handleRefreshCache();
                          }}
                          disabled={!canRefreshCache || !activeGroupKey}
                        >
                          {t('system.setting.cache.refresh')}
                        </Button>
                      </Space>
                      <SubmitBar
                        loading={submittingGroup === activeSettingGroup.groupKey}
                        submitDisabled={!canUpdateSetting}
                        onCancel={resetActiveGroupValues}
                        onSubmit={() => {
                          void handleSubmit();
                        }}
                      />
                    </div>
                  </Space>
                </Form>
              ) : null}
            </Card>
            {activeSettingGroup && canViewOperationLog ? (
              <Card className="page-panel system-list__table-card setting-page__audit-card">
                <div className="setting-page__audit-header">
                  <div>
                    <Typography.Text style={{ fontWeight: 600 }}>
                      {t('system.setting.audit.title')}
                    </Typography.Text>
                    <Typography.Paragraph type="secondary" style={{ margin: '4px 0 0' }}>
                      {t('common.total', { count: auditTotal })}
                    </Typography.Paragraph>
                  </div>
                  <Space>
                    <Button
                      size="small"
                      onClick={() => {
                        void handleExportAudit();
                      }}
                      disabled={!canExportAudit}
                    >
                      {t('common.export')}
                    </Button>
                  </Space>
                </div>
                <AppTable<SettingAuditRow>
                  className="system-list__table"
                  rowKey="id"
                  data={auditRows}
                  columns={auditColumns}
                  loading={auditLoading}
                  scroll={{ x: 'max-content' }}
                  onChange={handleAuditTableChange}
                  pagination={
                    {
                      current: auditQuery.page,
                      pageSize: auditQuery.pageSize,
                      total: auditTotal,
                      showJumper: true,
                      pageSizeChangeResetCurrent: false,
                      sizeCanChange: true,
                      sizeOptions: [5, 10, 20, 50],
                      size: 'small',
                      showTotal: (count: number) => t('common.total', { count }),
                    } as PaginationProps
                  }
                />
              </Card>
            ) : null}
          </>
        ) : null}
      </Space>

      <GovernanceInsightDrawer
        title={t('system.setting.hero.summaryTitle')}
        visible={governanceRail.expanded && Boolean(overview)}
        onClose={governanceRail.close}
        noteTitle={
          overview?.issues.length
            ? t('system.setting.hero.sideTitle')
            : t('system.setting.overview.noRisks')
        }
        noteDescription={t('system.setting.hero.sideDesc')}
        noteTone={overview?.issues.length ? 'warning' : 'neutral'}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
    </PageContainer>
  );
};

export default SettingPage;
