import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Form,
  Grid,
  Input,
  Popconfirm,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconDownload, IconSearch, IconEye } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import { getSettingGroup } from '../setting/api';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../../components/table/crossPageSelection';
import {
  batchDeleteOperationLogs,
  cleanupOperationLogs,
  deleteOperationLog,
  exportOperationLogs,
  exportSelectedOperationLogs,
  getOperationLog,
  getOperationLogList,
  type OperationLogRow,
  type OperationLogQuery,
} from './api';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  FilterPanel,
  type GovernanceCleanupMode,
  GovernanceCleanupBar,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  PermissionAction,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../../components';
import { formatDateTime } from '../../../core/format/dateTime';
import { usePermission } from '../../../hooks/usePermission';
import '../list-page.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;
const httpMethodSet = new Set(['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS', 'HEAD']);

type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue };

interface SettingAuditChangePreview {
  settingKey: string;
  oldValue: string;
  newValue: string;
  isEncrypted: number;
}

interface SettingAuditPayloadPreview {
  groupKey: string;
  changes: SettingAuditChangePreview[];
}

interface I18nLifecycleAuditPreview {
  action: string;
  module: string;
  fromStatus: string;
  toStatus: string;
  confirmArchived?: boolean;
  observationThresholdDays?: number;
  affectedRows?: number;
  affectedKeys: string[];
}

interface OperationResultPreview {
  code?: number;
  message?: string;
  data?: unknown;
}

interface AuditSourceMeta {
  layerKey: string;
  domainKey: string;
  pageKey: string;
}

interface FailureMeta {
  typeKey: string;
  summaryKey: string;
  color: string;
}

const emptyQuery: OperationLogQuery = {
  title: '',
  operName: '',
  status: undefined,
  sourceDomain: undefined,
  sourcePage: undefined,
  failureCategory: undefined,
  page: 1,
  pageSize: 10,
};
const defaultRetentionOptions = [1, 7, 30];

function toCleanupTimestamp(value: string) {
  const normalized = String(value || '').trim();
  const match = normalized.match(
    /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/,
  );
  if (!match) {
    return undefined;
  }
  const [, year, month, day, hour, minute, second = '00'] = match;
  const localDate = new Date(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
    Number(second),
  );
  if (Number.isNaN(localDate.getTime())) {
    return undefined;
  }
  const offsetMinutes = -localDate.getTimezoneOffset();
  const sign = offsetMinutes >= 0 ? '+' : '-';
  const offsetHours = `${Math.floor(Math.abs(offsetMinutes) / 60)}`.padStart(2, '0');
  const offsetRemainMinutes = `${Math.abs(offsetMinutes) % 60}`.padStart(2, '0');
  return `${year}-${month}-${day}T${hour}:${minute}:${second}${sign}${offsetHours}:${offsetRemainMinutes}`;
}

function normalizeRetentionOptions(rawValue: string | undefined) {
  if (!rawValue) {
    return defaultRetentionOptions;
  }
  try {
    const parsed = JSON.parse(rawValue) as unknown;
    if (!Array.isArray(parsed)) {
      return defaultRetentionOptions;
    }
    const normalized = Array.from(
      new Set(
        parsed.map((item) => Number(item)).filter((item) => Number.isInteger(item) && item > 0),
      ),
    ).sort((left, right) => right - left);
    return normalized.length > 0 ? normalized : defaultRetentionOptions;
  } catch {
    return defaultRetentionOptions;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function safeParseJSON(raw: string): unknown | null {
  const trimmed = raw.trim();
  if (!trimmed) {
    return null;
  }
  try {
    return JSON.parse(trimmed) as unknown;
  } catch {
    return null;
  }
}

function isSensitiveAuditKey(key: string) {
  const normalized = key.toLowerCase().replaceAll('_', '');
  return ['password', 'token', 'secret', 'accesskey', 'apikey', 'credential'].some((token) =>
    normalized.includes(token),
  );
}

function sanitizeAuditValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map((item) => sanitizeAuditValue(item));
  }
  if (isRecord(value)) {
    return Object.entries(value).reduce<Record<string, unknown>>((acc, [key, item]) => {
      acc[key] = isSensitiveAuditKey(key) ? '***' : sanitizeAuditValue(item);
      return acc;
    }, {});
  }
  return value;
}

function formatAuditRaw(raw: string) {
  if (!raw.trim()) {
    return '-';
  }
  const parsed = safeParseJSON(raw);
  if (parsed === null) {
    return raw;
  }
  return JSON.stringify(sanitizeAuditValue(parsed), null, 2);
}

function extractOperationResult(raw: string): OperationResultPreview {
  const parsed = safeParseJSON(raw);
  if (!isRecord(parsed)) {
    return {};
  }
  return {
    code: typeof parsed.code === 'number' ? parsed.code : undefined,
    message: typeof parsed.message === 'string' ? parsed.message : undefined,
    data: parsed.data,
  };
}

function extractSettingAuditPayload(raw: string): SettingAuditPayloadPreview | null {
  const parsed = safeParseJSON(raw);
  if (!isRecord(parsed) || !Array.isArray(parsed.changes)) {
    return null;
  }
  return {
    groupKey: typeof parsed.groupKey === 'string' ? parsed.groupKey : '',
    changes: parsed.changes
      .filter((item): item is Record<string, unknown> => isRecord(item))
      .map((item) => ({
        settingKey: typeof item.settingKey === 'string' ? item.settingKey : '',
        oldValue: typeof item.oldValue === 'string' ? item.oldValue : '',
        newValue: typeof item.newValue === 'string' ? item.newValue : '',
        isEncrypted: typeof item.isEncrypted === 'number' ? item.isEncrypted : 0,
      }))
      .filter((item) => item.settingKey !== ''),
  };
}

function extractI18nLifecycleAuditPayload(raw: string): I18nLifecycleAuditPreview | null {
  const parsed = safeParseJSON(raw);
  if (!isRecord(parsed) || typeof parsed.action !== 'string') {
    return null;
  }
  return {
    action: parsed.action,
    module: typeof parsed.module === 'string' ? parsed.module : '',
    fromStatus: typeof parsed.fromStatus === 'string' ? parsed.fromStatus : '',
    toStatus: typeof parsed.toStatus === 'string' ? parsed.toStatus : '',
    confirmArchived:
      typeof parsed.confirmArchived === 'boolean' ? parsed.confirmArchived : undefined,
    observationThresholdDays:
      typeof parsed.observationThresholdDays === 'number'
        ? parsed.observationThresholdDays
        : undefined,
    affectedRows: typeof parsed.affectedRows === 'number' ? parsed.affectedRows : undefined,
    affectedKeys: Array.isArray(parsed.affectedKeys)
      ? parsed.affectedKeys.filter((item): item is string => typeof item === 'string')
      : [],
  };
}

function extractI18nLifecycleAuditResult(raw: string): I18nLifecycleAuditPreview | null {
  const parsed = safeParseJSON(raw);
  if (!isRecord(parsed) || !isRecord(parsed.data) || typeof parsed.data.action !== 'string') {
    return null;
  }
  const data = parsed.data;
  return {
    action: typeof data.action === 'string' ? data.action : '',
    module: typeof data.module === 'string' ? data.module : '',
    fromStatus: typeof data.fromStatus === 'string' ? data.fromStatus : '',
    toStatus: typeof data.toStatus === 'string' ? data.toStatus : '',
    affectedRows: typeof data.affectedRows === 'number' ? data.affectedRows : undefined,
    affectedKeys: Array.isArray(data.affectedKeys)
      ? data.affectedKeys.filter((item): item is string => typeof item === 'string')
      : [],
  };
}

function normalizeMethod(method: string) {
  const trimmed = method.trim();
  const upper = trimmed.toUpperCase();
  return httpMethodSet.has(upper) ? upper : '';
}

function formatHandlerName(method: string) {
  const trimmed = method.trim();
  if (!trimmed || normalizeMethod(trimmed)) {
    return '';
  }
  return trimmed.replace(/-fm$/, '').replace(/^.*\//, '');
}

function getBusinessTypeMeta(
  businessType: number,
  t: (key: string, options?: Record<string, unknown>) => string,
) {
  const mapping: Record<number, { key: string; color: string }> = {
    0: { key: 'system.audit.businessType.other', color: 'gray' },
    1: { key: 'system.audit.businessType.insert', color: 'green' },
    2: { key: 'system.audit.businessType.update', color: 'arcoblue' },
    3: { key: 'system.audit.businessType.delete', color: 'red' },
    4: { key: 'system.audit.businessType.grant', color: 'orange' },
    5: { key: 'system.audit.businessType.export', color: 'purple' },
    6: { key: 'system.audit.businessType.import', color: 'cyan' },
    7: { key: 'system.audit.businessType.force', color: 'red' },
    8: { key: 'system.audit.businessType.clean', color: 'magenta' },
    1001: { key: 'system.audit.businessType.settingUpdate', color: 'gold' },
  };
  const matched = mapping[businessType] || mapping[0];
  return {
    label: t(matched.key),
    color: matched.color,
  };
}

function getAuditSourceMeta(log: OperationLogRow): AuditSourceMeta {
  const storedSourceDomain = log.sourceDomain?.trim();
  const storedSourcePage = log.sourcePage?.trim();
  if (storedSourceDomain || storedSourcePage) {
    return {
      layerKey:
        storedSourceDomain === 'platform'
          ? 'system.audit.sourceLayer.platform'
          : 'system.audit.sourceLayer.system',
      domainKey: storedSourceDomain
        ? `system.audit.sourceDomain.${storedSourceDomain}`
        : 'system.audit.sourceDomain.other',
      pageKey: storedSourcePage
        ? `system.audit.sourcePage.${storedSourcePage}`
        : 'system.audit.sourcePage.other',
    };
  }
  const path = log.operUrl.trim();
  if (
    path.includes('/system/setting') ||
    path.includes('/system/upload') ||
    path.includes('/system/i18n')
  ) {
    return {
      layerKey: 'system.audit.sourceLayer.system',
      domainKey: 'system.audit.sourceDomain.config',
      pageKey: path.includes('/system/upload')
        ? 'system.audit.sourcePage.upload'
        : path.includes('/system/i18n')
          ? 'system.audit.sourcePage.i18n'
          : 'system.audit.sourcePage.setting',
    };
  }
  if (path.includes('/system/operation-log')) {
    return {
      layerKey: 'system.audit.sourceLayer.system',
      domainKey: 'system.audit.sourceDomain.audit',
      pageKey: 'system.audit.sourcePage.operationLog',
    };
  }
  if (
    path.includes('/system/login-log') ||
    path.includes('/system/session') ||
    path.includes('/auth/')
  ) {
    return {
      layerKey: 'system.audit.sourceLayer.system',
      domainKey: 'system.audit.sourceDomain.auth',
      pageKey: path.includes('/system/session')
        ? 'system.audit.sourcePage.session'
        : 'system.audit.sourcePage.loginLog',
    };
  }
  if (
    path.includes('/system/user') ||
    path.includes('/system/role') ||
    path.includes('/system/menu') ||
    path.includes('/system/permission')
  ) {
    return {
      layerKey: 'system.audit.sourceLayer.system',
      domainKey: 'system.audit.sourceDomain.iam',
      pageKey: path.includes('/system/user')
        ? 'system.audit.sourcePage.user'
        : path.includes('/system/role')
          ? 'system.audit.sourcePage.role'
          : path.includes('/system/menu')
            ? 'system.audit.sourcePage.menu'
            : 'system.audit.sourcePage.permission',
    };
  }
  if (path.includes('/system/dept') || path.includes('/system/post')) {
    return {
      layerKey: 'system.audit.sourceLayer.system',
      domainKey: 'system.audit.sourceDomain.org',
      pageKey: path.includes('/system/dept')
        ? 'system.audit.sourcePage.dept'
        : 'system.audit.sourcePage.post',
    };
  }
  if (path.includes('/dashboard')) {
    return {
      layerKey: 'system.audit.sourceLayer.platform',
      domainKey: 'system.audit.sourceDomain.platform',
      pageKey: 'system.audit.sourcePage.dashboard',
    };
  }
  return {
    layerKey: 'system.audit.sourceLayer.system',
    domainKey: 'system.audit.sourceDomain.other',
    pageKey: 'system.audit.sourcePage.other',
  };
}

function getFailureMeta(
  log: OperationLogRow,
  resultPreview: OperationResultPreview,
): FailureMeta | null {
  if (log.status !== 2) {
    return null;
  }
  const storedFailureCategory = log.failureCategory?.trim();
  if (storedFailureCategory) {
    return {
      typeKey: `system.audit.failureType.${storedFailureCategory}`,
      summaryKey: `system.audit.failureSummary.${storedFailureCategory}`,
      color:
        storedFailureCategory === 'validation'
          ? 'orange'
          : storedFailureCategory === 'auth'
            ? 'gold'
            : storedFailureCategory === 'permission'
              ? 'red'
              : storedFailureCategory === 'server'
                ? 'magenta'
                : 'arcoblue',
    };
  }
  const errorKey = `${log.errorMsg} ${resultPreview.message || ''}`.toLowerCase();
  const resultCode = resultPreview.code || 0;

  if (
    resultCode === 400 ||
    errorKey.includes('param.invalid') ||
    errorKey.includes('setting.value.') ||
    errorKey.includes('upload.file.')
  ) {
    return {
      typeKey: 'system.audit.failureType.validation',
      summaryKey: 'system.audit.failureSummary.validation',
      color: 'orange',
    };
  }
  if (
    resultCode === 401 ||
    errorKey.includes('auth.') ||
    errorKey.includes('refresh_token') ||
    errorKey.includes('login.error')
  ) {
    return {
      typeKey: 'system.audit.failureType.auth',
      summaryKey: 'system.audit.failureSummary.auth',
      color: 'gold',
    };
  }
  if (resultCode === 403 || errorKey.includes('permission.denied')) {
    return {
      typeKey: 'system.audit.failureType.permission',
      summaryKey: 'system.audit.failureSummary.permission',
      color: 'red',
    };
  }
  if (resultCode >= 500 || errorKey.includes('database.') || errorKey.includes('.error')) {
    return {
      typeKey: 'system.audit.failureType.server',
      summaryKey: 'system.audit.failureSummary.server',
      color: 'magenta',
    };
  }
  return {
    typeKey: 'system.audit.failureType.business',
    summaryKey: 'system.audit.failureSummary.business',
    color: 'arcoblue',
  };
}

const OperationLogList: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canExport = isAdmin || hasPerm('system:operation-log:export');
  const canClear = isAdmin || hasPerm('system:operation-log:clear');
  const canDelete = isAdmin || hasPerm('system:operation-log:delete');
  const governanceRail = useGovernanceRail();
  const [data, setData] = useState<OperationLogRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadFailed, setLoadFailed] = useState(false);
  const [query, setQuery] = useState<OperationLogQuery>(emptyQuery);
  const [queryForm] = Form.useForm<OperationLogQuery>();
  const [detailVisible, setDetailVisible] = useState(false);
  const [currentLog, setCurrentLog] = useState<OperationLogRow | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [retentionDays, setRetentionDays] = useState<number>(30);
  const [cleanupMode, setCleanupMode] = useState<GovernanceCleanupMode>('retention');
  const [cleanupRangeStart, setCleanupRangeStart] = useState('');
  const [cleanupRangeEnd, setCleanupRangeEnd] = useState('');
  const [retentionOptions, setRetentionOptions] = useState<number[]>(() =>
    [...defaultRetentionOptions].sort((left, right) => right - left),
  );
  const resultPreview = currentLog ? extractOperationResult(currentLog.jsonResult) : {};
  const settingAuditPayload = currentLog ? extractSettingAuditPayload(currentLog.operParam) : null;
  const i18nLifecycleParam = currentLog
    ? extractI18nLifecycleAuditPayload(currentLog.operParam)
    : null;
  const i18nLifecycleResult = currentLog
    ? extractI18nLifecycleAuditResult(currentLog.jsonResult)
    : null;
  const translatedErrorText = currentLog?.errorMsg
    ? t(currentLog.errorMsg, { defaultValue: currentLog.errorMsg })
    : '';
  const translatedResultMessage = resultPreview.message
    ? t(resultPreview.message, { defaultValue: resultPreview.message })
    : '';
  const methodText = currentLog ? normalizeMethod(currentLog.method) : '';
  const handlerText = currentLog ? formatHandlerName(currentLog.method) : '';
  const businessTypeMeta = currentLog ? getBusinessTypeMeta(currentLog.businessType, t) : null;
  const sourceMeta = currentLog ? getAuditSourceMeta(currentLog) : null;
  const failureMeta = currentLog ? getFailureMeta(currentLog, resultPreview) : null;

  const loadData = useCallback(
    async (nextQuery: OperationLogQuery = query) => {
      setLoading(true);
      setLoadFailed(false);
      try {
        const result = await getOperationLogList(nextQuery);
        setData(result.items);
        setTotal(result.total);
      } catch {
        setLoadFailed(true);
        message.error(t('common.loadFailed'));
      } finally {
        setLoading(false);
      }
    },
    [query, t],
  );

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      void loadData(query);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadData, query]);

  const applyOperationLogRetentionOptions = (group: any) => {
      const setting = group.items.find(
        (item: any) => item.settingKey === 'audit.operation_log_retention_options',
      );
      const nextOptions = normalizeRetentionOptions(setting?.settingValue);
      setRetentionOptions(nextOptions);
      setRetentionDays((current: any) => (nextOptions.includes(current) ? current : nextOptions[0]));
    };

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      getSettingGroup('audit')
        .then(applyOperationLogRetentionOptions)
        .catch(() => undefined);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, []);

  useEffect(() => {
    const detailId = Number(searchParams.get('detailId') || 0);
    if (!detailId || Number.isNaN(detailId)) {
      return;
    }

    const matchedLog = data.find((item) => item.id === detailId);
    if (matchedLog) {
      const timer = globalThis.setTimeout(() => {
        setCurrentLog(matchedLog);
        setDetailVisible(true);
        setDetailLoading(false);
      }, 0);
      return () => globalThis.clearTimeout(timer);
    }

    const timer = globalThis.setTimeout(() => {
      setDetailLoading(true);
      getOperationLog(detailId)
        .then((detail) => {
          setCurrentLog(detail);
          setDetailVisible(true);
        })
        .catch(() => {
          message.error(t('common.loadFailed'));
        })
        .finally(() => {
          setDetailLoading(false);
        });
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [data, searchParams, t]);

  const search = () => {
    const values = queryForm.getFieldsValue();
    setSelectedRowKeys([]);
    setQuery({
      ...query,
      ...values,
      page: 1,
    });
  };

  const reset = () => {
    queryForm.setFieldsValue(emptyQuery);
    setSelectedRowKeys([]);
    setQuery(emptyQuery);
  };

  const handleDelete = async (id: number) => {
    try {
      await deleteOperationLog(id);
      message.success(t('system.audit.deleteSuccess'));
      void loadData();
    } catch {
      // message.error already handled by request interceptor
    }
  };

  const handleCleanup = async () => {
    if (cleanupMode === 'range' && (!cleanupRangeStart || !cleanupRangeEnd)) {
      message.warning(t('common.cleanupRangeRequired'));
      return;
    }
    try {
      const resp = await cleanupOperationLogs(
        cleanupMode === 'range'
          ? {
              startedAt: toCleanupTimestamp(cleanupRangeStart),
              endedAt: toCleanupTimestamp(cleanupRangeEnd),
            }
          : { retentionDays },
      );
      message.success(t('system.audit.cleanupSuccess', { count: resp.clearedCount }));
      void loadData();
    } catch {
      message.error(t('common.actionFailed'));
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    try {
      const resp = await batchDeleteOperationLogs({ ids: selectedRowKeys });
      message.success(t('system.audit.batchDeleteSuccess', { count: resp.deletedCount }));
      setSelectedRowKeys([]);
      void loadData();
    } catch {
      message.error(t('common.actionFailed'));
    }
  };

  const showDetail = (log: OperationLogRow) => {
    setCurrentLog(log);
    setDetailVisible(true);
  };

  const closeDetail = () => {
    setDetailVisible(false);
    setCurrentLog(null);
    if (searchParams.has('detailId')) {
      const nextParams = new URLSearchParams(searchParams);
      nextParams.delete('detailId');
      setSearchParams(nextParams, { replace: true });
    }
  };

  const handleTableChange: TableProps<OperationLogRow>['onChange'] = (pagination) => {
    setQuery({
      ...query,
      page: pagination.current || 1,
      pageSize: pagination.pageSize || query.pageSize || emptyQuery.pageSize,
    });
  };

  const visibleSelectedRowKeys = useMemo(
    () => getVisibleSelectedRowKeys(selectedRowKeys, data.map((item) => item.id)),
    [data, selectedRowKeys],
  );

  const columns: ColumnProps<OperationLogRow>[] = [
    {
      title: t('system.audit.title'),
      dataIndex: 'title',
      width: TABLE_COLUMN_WIDTH.tagGroup,
      ellipsis: true,
      render: (value: string) => t(value, { defaultValue: value || '-' }),
    },
    {
      title: t('system.audit.sourceDomain'),
      width: TABLE_COLUMN_WIDTH.identity,
      render: (_, record) => {
        const recordSourceMeta = getAuditSourceMeta(record);
        return <Tag color="purple">{t(recordSourceMeta.domainKey)}</Tag>;
      },
    },
    withTableColumnPriority(
      {
        title: t('system.audit.operName'),
        dataIndex: 'operName',
        width: TABLE_COLUMN_WIDTH.identity,
        render: (value: string) => value || '-',
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('system.audit.operIp'),
        dataIndex: 'operIp',
        width: TABLE_COLUMN_WIDTH.identity,
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('auth.loginLog.location'),
        dataIndex: 'operLocation',
        width: TABLE_COLUMN_WIDTH.location,
        ellipsis: true,
      },
      'low',
    ),
    {
      title: t('system.audit.operUrl'),
      dataIndex: 'operUrl',
      width: TABLE_COLUMN_WIDTH.routePath,
      ellipsis: true,
    },
    {
      title: t('system.audit.status'),
      dataIndex: 'status',
      width: TABLE_COLUMN_WIDTH.diagnostics,
      render: (value: number, record) => (
        <Space direction="vertical" size={4}>
          <Tag color={value === 1 ? 'green' : 'red'}>
            {value === 1 ? t('common.success') : t('common.failed')}
          </Tag>
          {value === 2 && record.errorMsg ? (
            <>
              {(() => {
                const nextFailureMeta = getFailureMeta(
                  record,
                  extractOperationResult(record.jsonResult),
                );
                return nextFailureMeta ? (
                  <Tag color={nextFailureMeta.color}>{t(nextFailureMeta.typeKey)}</Tag>
                ) : null;
              })()}
              <Typography.Text type="error" ellipsis={{ rows: 1, showTooltip: true }}>
                {t(record.errorMsg, { defaultValue: record.errorMsg })}
              </Typography.Text>
            </>
          ) : null}
        </Space>
      ),
    },
    withTableColumnPriority(
      {
        title: t('system.audit.operTime'),
        dataIndex: 'operTime',
        width: TABLE_COLUMN_WIDTH.datetime,
        render: (value: string) => formatDateTime(value),
      },
      'medium',
    ),
    {
      title: t('common.action'),
      fixed: 'right',
      width: TABLE_ACTION_COLUMN_WIDTH.compact,
      render: (_, record) => (
        <Space size={4} className="system-list__actions">
          <Button type="text" size="small" icon={<IconEye />} onClick={() => showDetail(record)}>
            {t('common.detail')}
          </Button>
          {canDelete && (
            <Popconfirm title={t('common.deleteConfirm')} onOk={() => handleDelete(record.id)}>
              <Button type="text" status="danger" size="small" icon={<IconDelete />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const handleExport = async () => {
    if (selectedRowKeys.length > 0) {
      const selectedRows = data.filter((item) => selectedRowKeys.includes(item.id));
      if (selectedRows.length !== selectedRowKeys.length) {
        message.warning(
          t('common.exportCurrentPageSelectionOnly', {
            defaultValue: '已选记录包含跨页项，请切回对应页面后再导出。',
          }),
        );
        return;
      }
      exportSelectedOperationLogs(selectedRows);
      return;
    }
    await exportOperationLogs(query);
  };
  const successCount = useMemo(() => data.filter((item) => item.status === 1).length, [data]);
  const failedCount = useMemo(() => data.filter((item) => item.status !== 1).length, [data]);
  const heroStats = useMemo(
    () => [
      {
        key: 'total',
        label: t('common.total', { count: total }),
        value: total,
        hint: t('system.audit.hero.totalHint'),
      },
      {
        key: 'success',
        label: t('common.success'),
        value: successCount,
        hint: t('system.audit.hero.successHint'),
      },
      {
        key: 'failed',
        label: t('common.failed'),
        value: failedCount,
        hint: t('system.audit.hero.failedHint'),
      },
      {
        key: 'export',
        label: t('system.audit.hero.exportReady'),
        value: canExport ? t('common.yes') : t('common.no'),
        hint: t('system.audit.hero.exportHint'),
      },
      {
        key: 'cleanup',
        label: t('system.audit.hero.cleanupReady'),
        value: canClear ? t('common.yes') : t('common.no'),
        hint: t('system.audit.hero.cleanupHint'),
      },
    ],
    [canClear, canExport, failedCount, successCount, t, total],
  );

  const formatI18nLifecycleLabel = (value: string) =>
    value ? t(`i18n.lifecycle.status.${value}`, { defaultValue: value }) : '-';

  const formatI18nLifecycleAction = (value: string) => {
    if (value === 'observe') {
      return t('i18n.lifecycle.observe');
    }
    if (value === 'archive') {
      return t('i18n.lifecycle.archive');
    }
    if (value === 'delete') {
      return t('i18n.lifecycle.delete');
    }
    return value || '-';
  };

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('system.audit.hero.eyebrow')}
          title={t('system.audit.hero.title')}
          description={t('system.audit.hero.desc')}
          metrics={heroStats.slice(0, 3).map((item) => ({
            key: item.key,
            label: item.label,
            value: item.value,
          }))}
          action={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('system.audit.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
          <FilterPanel>
            <Form form={queryForm} layout="vertical" onSubmit={() => search()}>
              <Row gutter={16}>
                <Col span={6}>
                  <FormItem label={t('system.audit.title')} field="title">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem label={t('system.audit.operName')} field="operName">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem label={t('system.audit.status')} field="status">
                    <Select
                      allowClear
                      options={[
                        { label: t('common.success'), value: 1 },
                        { label: t('common.failed'), value: 2 },
                      ]}
                    />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem label={t('system.audit.sourceDomain')} field="sourceDomain">
                    <Select
                      allowClear
                      options={[
                        { label: t('system.audit.sourceDomain.platform'), value: 'platform' },
                        { label: t('system.audit.sourceDomain.auth'), value: 'auth' },
                        { label: t('system.audit.sourceDomain.iam'), value: 'iam' },
                        { label: t('system.audit.sourceDomain.org'), value: 'org' },
                        { label: t('system.audit.sourceDomain.config'), value: 'config' },
                        { label: t('system.audit.sourceDomain.audit'), value: 'audit' },
                        { label: t('system.audit.sourceDomain.other'), value: 'other' },
                      ]}
                    />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem label={t('system.audit.failureCategory')} field="failureCategory">
                    <Select
                      allowClear
                      options={[
                        { label: t('system.audit.failureType.validation'), value: 'validation' },
                        { label: t('system.audit.failureType.auth'), value: 'auth' },
                        { label: t('system.audit.failureType.permission'), value: 'permission' },
                        { label: t('system.audit.failureType.server'), value: 'server' },
                        { label: t('system.audit.failureType.business'), value: 'business' },
                      ]}
                    />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem label={t('system.audit.sourcePage')} field="sourcePage">
                    <Select
                      allowClear
                      options={[
                        { label: t('system.audit.sourcePage.dashboard'), value: 'dashboard' },
                        { label: t('system.audit.sourcePage.setting'), value: 'setting' },
                        { label: t('system.audit.sourcePage.upload'), value: 'upload' },
                        { label: t('system.audit.sourcePage.i18n'), value: 'i18n' },
                        { label: t('system.audit.sourcePage.operationLog'), value: 'operationLog' },
                        { label: t('system.audit.sourcePage.loginLog'), value: 'loginLog' },
                        { label: t('system.audit.sourcePage.session'), value: 'session' },
                        { label: t('system.audit.sourcePage.user'), value: 'user' },
                        { label: t('system.audit.sourcePage.role'), value: 'role' },
                        { label: t('system.audit.sourcePage.menu'), value: 'menu' },
                        { label: t('system.audit.sourcePage.permission'), value: 'permission' },
                        { label: t('system.audit.sourcePage.dept'), value: 'dept' },
                        { label: t('system.audit.sourcePage.post'), value: 'post' },
                        { label: t('system.audit.sourcePage.other'), value: 'other' },
                      ]}
                    />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem className="filter-panel__action-item">
                    <Space>
                      <Button type="primary" htmlType="submit" icon={<IconSearch />}>
                        {t('common.search')}
                      </Button>
                      <Button onClick={reset}>{t('common.reset')}</Button>
                    </Space>
                  </FormItem>
                </Col>
              </Row>
            </Form>
          </FilterPanel>

          <Card className="page-panel system-list__table-card">
            <GovernanceCleanupBar
              showCleanup={canClear}
              retentionDays={retentionDays}
              retentionOptions={retentionOptions}
              onRetentionChange={setRetentionDays}
              retentionLabel={(option) => t('common.keepRecentDays', { count: option })}
              cleanupMode={cleanupMode}
              onCleanupModeChange={setCleanupMode}
              cleanupModeLabel={t('common.cleanupMode')}
              cleanupModeOptions={[
                { label: t('common.cleanupModeRetention'), value: 'retention' },
                { label: t('common.cleanupModeRange'), value: 'range' },
              ]}
              rangeStart={cleanupRangeStart}
              rangeEnd={cleanupRangeEnd}
              onRangeStartChange={setCleanupRangeStart}
              onRangeEndChange={setCleanupRangeEnd}
              rangeStartLabel={t('common.cleanupRangeStart')}
              rangeEndLabel={t('common.cleanupRangeEnd')}
              confirmTitle={
                cleanupMode === 'range'
                  ? t('common.cleanupRangeConfirm')
                  : t('system.audit.cleanupConfirm', { count: retentionDays })
              }
              actionLabel={t('common.cleanupLogs')}
              onConfirm={() => {
                void handleCleanup();
              }}
              hint={t('system.audit.cleanupHint')}
              trailing={
                <Button
                  icon={<IconDownload />}
                  onClick={() => {
                    void handleExport();
                  }}
                  disabled={!canExport}
                >
                  {t('common.export')}
                </Button>
              }
              extraActions={
                canDelete ? (
                  <>
                    <Typography.Text type="secondary">
                      {t('common.selectedCount', { count: selectedRowKeys.length })}
                    </Typography.Text>
                    <Button
                      type="text"
                      size="small"
                      disabled={selectedRowKeys.length === 0}
                      onClick={() => {
                        if (selectedRowKeys.length === 0) {
                          return;
                        }
                        setSelectedRowKeys([]);
                        message.success(t('common.clearSelectionSuccess'));
                      }}
                    >
                      {t('common.clearSelection')}
                    </Button>
                    <PermissionAction
                      allowed={canDelete}
                      tooltip={t('common.noPermissionAction')}
                    >
                      <Popconfirm
                        disabled={selectedRowKeys.length === 0 || !canDelete}
                        title={t('system.audit.batchDeleteConfirm', {
                          count: selectedRowKeys.length,
                        })}
                        onOk={() => {
                          void handleBatchDelete();
                        }}
                      >
                        <Button
                          status="danger"
                          icon={<IconDelete />}
                          disabled={selectedRowKeys.length === 0 || !canDelete}
                        >
                          {t('common.deleteSelected')}
                        </Button>
                      </Popconfirm>
                    </PermissionAction>
                  </>
                ) : undefined
              }
            />

            {loading && data.length === 0 ? <PageLoading /> : null}
            {loadFailed && !loading ? (
              <PageError
                onRetry={() => {
                  void loadData(query);
                }}
              />
            ) : data.length === 0 && !loading ? (
              <PageEmpty />
            ) : (
              <AppTable<OperationLogRow>
                className="system-list__table"
                rowKey="id"
                data={data}
                columns={columns}
                loading={loading}
                scroll={{ x: 1240 }}
                onChange={handleTableChange}
                rowSelection={
                  canDelete
                    ? {
                        type: 'checkbox',
                        selectedRowKeys: visibleSelectedRowKeys,
                        checkCrossPage: true,
                        preserveSelectedRowKeys: true,
                        onChange: (keys) =>
                          setSelectedRowKeys((currentKeys) =>
                            mergeCrossPageSelection(
                              currentKeys,
                              keys as number[],
                              data.map((item) => item.id),
                            ) as number[],
                          ),
                      }
                    : undefined
                }
                pagination={buildStandardPagination(t, {
                  current: query.page || emptyQuery.page,
                  pageSize: query.pageSize || emptyQuery.pageSize,
                  total,
                })}
              />
            )}
          </Card>
        </>
      </Space>

      <GovernanceInsightDrawer
        title={t('system.audit.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('system.audit.failureSummary')}
        noteDescription={t('system.audit.hero.sideDesc')}
        noteTone="warning"
      >
        <GovernanceRailSummary
          items={[
            {
              label: t('common.success'),
              value: successCount,
              description: t('system.audit.hero.successHint'),
            },
            {
              tone: 'warning',
              label: t('common.failed'),
              value: failedCount,
              description: t('system.audit.hero.failedHint'),
            },
            {
              label: t('system.audit.hero.cleanupReady'),
              value: canClear ? t('common.yes') : t('common.no'),
              description: t('system.audit.hero.cleanupHint'),
            },
            {
              label: t('common.selected'),
              value: selectedRowKeys.length,
              description: t('system.audit.hero.selectedHint'),
            },
          ]}
        />
      </GovernanceInsightDrawer>

      <AppModal
        title={t('common.detail')}
        visible={detailVisible}
        onCancel={closeDetail}
        footer={null}
        size="detail"
      >
        {detailLoading && !currentLog ? <PageLoading /> : null}
        {currentLog && (
          <Space direction="vertical" size={16} className="detail-stack">
            <div className="audit-detail-summary">
              <div className="audit-detail-summary__copy">
                <Typography.Text className="audit-detail-summary__title">
                  {t(currentLog.title, { defaultValue: currentLog.title || '-' })}
                </Typography.Text>
                <Typography.Text className="audit-detail-summary__desc" type="secondary">
                  {currentLog.operUrl || '-'}
                </Typography.Text>
              </div>
              <Space wrap>
                <Tag color={currentLog.status === 1 ? 'green' : 'red'}>
                  {currentLog.status === 1 ? t('common.success') : t('common.failed')}
                </Tag>
                {businessTypeMeta ? (
                  <Tag color={businessTypeMeta.color}>{businessTypeMeta.label}</Tag>
                ) : null}
                {sourceMeta ? <Tag color="purple">{t(sourceMeta.domainKey)}</Tag> : null}
                {sourceMeta ? <Tag color="cyan">{t(sourceMeta.pageKey)}</Tag> : null}
                {methodText ? <Tag color="arcoblue">{methodText}</Tag> : null}
                <Tag color="gray">
                  {t('system.audit.costTimeValue', { count: currentLog.costTime })}
                </Tag>
              </Space>
            </div>

            <Descriptions
              column={2}
              data={[
                {
                  label: t('system.audit.title'),
                  value: t(currentLog.title, { defaultValue: currentLog.title || '-' }),
                },
                { label: t('system.audit.operTime'), value: formatDateTime(currentLog.operTime) },
                { label: t('system.audit.operName'), value: currentLog.operName || '-' },
                { label: t('system.audit.operIp'), value: currentLog.operIp || '-' },
                { label: t('system.audit.operUrl'), value: currentLog.operUrl || '-' },
                ...(sourceMeta
                  ? [{ label: t('system.audit.sourceLayer'), value: t(sourceMeta.layerKey) }]
                  : []),
                ...(sourceMeta
                  ? [{ label: t('system.audit.sourceDomain'), value: t(sourceMeta.domainKey) }]
                  : []),
                ...(sourceMeta
                  ? [{ label: t('system.audit.sourcePage'), value: t(sourceMeta.pageKey) }]
                  : []),
                { label: t('system.audit.businessType'), value: businessTypeMeta?.label || '-' },
                { label: t('system.audit.method'), value: methodText || '-' },
                {
                  label: t('system.audit.costTime'),
                  value: t('system.audit.costTimeValue', { count: currentLog.costTime }),
                },
                ...(handlerText ? [{ label: t('system.audit.handler'), value: handlerText }] : []),
              ]}
            />

            {currentLog.status === 2 && (
              <Card
                className="detail-panel-card detail-panel-card--danger"
                title={t('system.audit.failureReason')}
                size="small"
              >
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <Space wrap>
                    {failureMeta ? (
                      <Tag color={failureMeta.color}>{t(failureMeta.typeKey)}</Tag>
                    ) : null}
                    {sourceMeta ? <Tag color="purple">{t(sourceMeta.domainKey)}</Tag> : null}
                  </Space>
                  <Alert
                    type="error"
                    content={translatedErrorText || translatedResultMessage || t('common.failed')}
                  />
                  <Descriptions
                    column={1}
                    data={[
                      ...(failureMeta
                        ? [
                            {
                              label: t('system.audit.failureCategory'),
                              value: t(failureMeta.typeKey),
                            },
                          ]
                        : []),
                      ...(failureMeta
                        ? [
                            {
                              label: t('system.audit.failureSummary'),
                              value: t(failureMeta.summaryKey),
                            },
                          ]
                        : []),
                      ...(translatedErrorText
                        ? [{ label: t('system.audit.failureText'), value: translatedErrorText }]
                        : []),
                      ...(currentLog.errorMsg
                        ? [{ label: t('system.audit.errorKey'), value: currentLog.errorMsg }]
                        : []),
                      ...(typeof resultPreview.code === 'number'
                        ? [{ label: t('system.audit.responseCode'), value: resultPreview.code }]
                        : []),
                      ...(translatedResultMessage
                        ? [
                            {
                              label: t('system.audit.responseMessage'),
                              value: translatedResultMessage,
                            },
                          ]
                        : []),
                      ...(resultPreview.message && resultPreview.message !== translatedResultMessage
                        ? [
                            {
                              label: t('system.audit.responseMessageKey'),
                              value: resultPreview.message,
                            },
                          ]
                        : []),
                    ]}
                  />
                </Space>
              </Card>
            )}

            {settingAuditPayload && settingAuditPayload.changes.length > 0 ? (
              <Card
                className="detail-panel-card"
                title={t('system.audit.requestSummary')}
                size="small"
              >
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <Space wrap>
                    {settingAuditPayload.groupKey ? (
                      <Tag color="arcoblue">
                        {t(
                          `system.setting.group.${settingAuditPayload.groupKey}`,
                          settingAuditPayload.groupKey,
                        )}
                      </Tag>
                    ) : null}
                    <Tag color="gold">
                      {t('system.audit.changeCount', { count: settingAuditPayload.changes.length })}
                    </Tag>
                  </Space>
                  <Space direction="vertical" size={8} style={{ width: '100%' }}>
                    {settingAuditPayload.changes.map((change) => (
                      <Space key={change.settingKey} size={8} wrap>
                        <Typography.Text>
                          {t(`system.setting.item.${change.settingKey}`, change.settingKey)}
                        </Typography.Text>
                        {change.isEncrypted === 1 ? (
                          <Tag color="red">{t('system.setting.audit.sensitiveChanged')}</Tag>
                        ) : (
                          <>
                            <Typography.Text type="secondary">
                              {change.oldValue || '-'}
                            </Typography.Text>
                            <Typography.Text type="secondary">→</Typography.Text>
                            <Typography.Text>{change.newValue || '-'}</Typography.Text>
                          </>
                        )}
                      </Space>
                    ))}
                  </Space>
                </Space>
              </Card>
            ) : null}

            {i18nLifecycleParam ? (
              <Card
                className="detail-panel-card"
                title={t('system.audit.requestSummary')}
                size="small"
              >
                <Descriptions
                  column={2}
                  data={[
                    {
                      label: t('i18n.audit.action'),
                      value: formatI18nLifecycleAction(i18nLifecycleParam.action),
                    },
                    { label: t('common.module'), value: i18nLifecycleParam.module || '-' },
                    {
                      label: t('i18n.lifecycle.fromStatus'),
                      value: formatI18nLifecycleLabel(i18nLifecycleParam.fromStatus),
                    },
                    {
                      label: t('i18n.lifecycle.toStatus'),
                      value: formatI18nLifecycleLabel(i18nLifecycleParam.toStatus),
                    },
                    ...(typeof i18nLifecycleParam.observationThresholdDays === 'number'
                      ? [
                          {
                            label: t('i18n.lifecycle.observationThreshold'),
                            value: t('i18n.lifecycle.observationThresholdValue', {
                              count: i18nLifecycleParam.observationThresholdDays,
                            }),
                          },
                        ]
                      : []),
                    ...(typeof i18nLifecycleParam.confirmArchived === 'boolean'
                      ? [
                          {
                            label: t('i18n.lifecycle.confirmArchived'),
                            value: i18nLifecycleParam.confirmArchived
                              ? t('common.yes')
                              : t('common.no'),
                          },
                        ]
                      : []),
                    ...(i18nLifecycleParam.affectedKeys.length > 0
                      ? [
                          {
                            label: t('i18n.audit.affectedKeys'),
                            value: i18nLifecycleParam.affectedKeys.join(', '),
                          },
                        ]
                      : []),
                  ]}
                />
              </Card>
            ) : null}

            {i18nLifecycleResult ? (
              <Card
                className="detail-panel-card"
                title={t('system.audit.resultSummary')}
                size="small"
              >
                <Descriptions
                  column={2}
                  data={[
                    {
                      label: t('i18n.audit.action'),
                      value: formatI18nLifecycleAction(i18nLifecycleResult.action),
                    },
                    { label: t('common.module'), value: i18nLifecycleResult.module || '-' },
                    {
                      label: t('i18n.lifecycle.fromStatus'),
                      value: formatI18nLifecycleLabel(i18nLifecycleResult.fromStatus),
                    },
                    {
                      label: t('i18n.lifecycle.toStatus'),
                      value: formatI18nLifecycleLabel(i18nLifecycleResult.toStatus),
                    },
                    {
                      label: t('system.audit.responseData'),
                      value: t('i18n.lifecycle.affectedRowsValue', {
                        count: i18nLifecycleResult.affectedRows || 0,
                      }),
                    },
                    {
                      label: t('i18n.audit.affectedKeys'),
                      value:
                        i18nLifecycleResult.affectedKeys.length > 0
                          ? i18nLifecycleResult.affectedKeys.join(', ')
                          : '-',
                    },
                  ]}
                />
              </Card>
            ) : typeof resultPreview.code === 'number' ||
              translatedResultMessage ||
              resultPreview.data !== undefined ? (
              <Card
                className="detail-panel-card"
                title={t('system.audit.resultSummary')}
                size="small"
              >
                <Descriptions
                  column={2}
                  data={[
                    ...(typeof resultPreview.code === 'number'
                      ? [{ label: t('system.audit.responseCode'), value: resultPreview.code }]
                      : []),
                    ...(translatedResultMessage
                      ? [
                          {
                            label: t('system.audit.responseMessage'),
                            value: translatedResultMessage,
                          },
                        ]
                      : []),
                    ...(resultPreview.message && resultPreview.message !== translatedResultMessage
                      ? [
                          {
                            label: t('system.audit.responseMessageKey'),
                            value: resultPreview.message,
                          },
                        ]
                      : []),
                    ...(resultPreview.data !== undefined
                      ? [
                          {
                            label: t('system.audit.responseData'),
                            value:
                              isRecord(resultPreview.data) || Array.isArray(resultPreview.data)
                                ? t('system.audit.responseDataStructured')
                                : String(resultPreview.data as JsonValue),
                          },
                        ]
                      : []),
                  ]}
                />
              </Card>
            ) : null}

            <Card className="detail-panel-card" title={t('system.audit.operParam')} size="small">
              <pre style={{ margin: 0, whiteSpace: 'pre-wrap', maxHeight: 220, overflow: 'auto' }}>
                {formatAuditRaw(currentLog.operParam)}
              </pre>
            </Card>
            <Card className="detail-panel-card" title={t('system.audit.jsonResult')} size="small">
              <pre style={{ margin: 0, whiteSpace: 'pre-wrap', maxHeight: 220, overflow: 'auto' }}>
                {formatAuditRaw(currentLog.jsonResult)}
              </pre>
            </Card>
          </Space>
        )}
      </AppModal>
    </PageContainer>
  );
};

export default OperationLogList;
