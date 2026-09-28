import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Popconfirm,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import type {
  ColumnProps,
  SorterInfo,
  TableProps,
} from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconDownload, IconEye } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import {
  buildCrossPageRowSelection,
  type CrossPageRowKey,
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
import { getSettingGroup, type SettingGroup } from '../setting/api';
import { loadRetentionSetting } from './retentionSetting';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  GovernanceCleanupBar,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageLoading,
  PageRequestError,
  PermissionAction,
  SearchToolbar,
  SystemRowActions,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  TimeRangeFilter,
  type GovernanceCleanupPayload,
  type TimeRangeFilterValue,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../../components';
import { formatDateTime } from '../../../core/format/dateTime';
import { usePermission } from '../../../hooks/usePermission';
import '../components/shared/list-page.css';
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

const defaultRetentionOptions = [1, 7, 30];

const emptyQuery: OperationLogQuery = {
  keyword: '',
  title: '',
  operName: '',
  status: undefined,
  sourceDomain: undefined,
  sourcePage: undefined,
  failureCategory: undefined,
  page: 1,
  pageSize: 10,
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function resolveConfigSourcePageKey(path: string) {
  if (path.includes('/system/upload')) {
    return 'system.audit.sourcePage.upload';
  }
  if (path.includes('/system/i18n')) {
    return 'system.audit.sourcePage.i18n';
  }
  return 'system.audit.sourcePage.setting';
}

function resolveIamSourcePageKey(path: string) {
  if (path.includes('/system/user')) {
    return 'system.audit.sourcePage.user';
  }
  if (path.includes('/system/role')) {
    return 'system.audit.sourcePage.role';
  }
  if (path.includes('/system/menu')) {
    return 'system.audit.sourcePage.menu';
  }
  return 'system.audit.sourcePage.permission';
}

function failureCategoryColor(category: string) {
  if (category === 'validation') {
    return 'orange';
  }
  if (category === 'auth') {
    return 'gold';
  }
  if (category === 'permission') {
    return 'red';
  }
  if (category === 'server') {
    return 'magenta';
  }
  return 'arcoblue';
}

function safeParseJSON(raw: string): unknown {
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

function getStoredSourceMeta(log: OperationLogRow): AuditSourceMeta | null {
  const storedSourceDomain = log.sourceDomain?.trim();
  const storedSourcePage = log.sourcePage?.trim();
  if (!storedSourceDomain && !storedSourcePage) {
    return null;
  }
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

// 路径推断规则表：命中任一片段即归入对应域，pageKey 可按路径细分。
const AUDIT_SOURCE_PATH_RULES: Array<{
  fragments: string[];
  layerKey: string;
  domainKey: string;
  resolvePageKey: (path: string) => string;
}> = [
  {
    fragments: ['/system/setting', '/system/upload', '/system/i18n'],
    layerKey: 'system.audit.sourceLayer.system',
    domainKey: 'system.audit.sourceDomain.config',
    resolvePageKey: resolveConfigSourcePageKey,
  },
  {
    fragments: ['/system/operation-log'],
    layerKey: 'system.audit.sourceLayer.system',
    domainKey: 'system.audit.sourceDomain.audit',
    resolvePageKey: () => 'system.audit.sourcePage.operationLog',
  },
  {
    fragments: ['/system/login-log', '/system/session', '/auth/'],
    layerKey: 'system.audit.sourceLayer.system',
    domainKey: 'system.audit.sourceDomain.auth',
    resolvePageKey: (path) =>
      path.includes('/system/session')
        ? 'system.audit.sourcePage.session'
        : 'system.audit.sourcePage.loginLog',
  },
  {
    fragments: ['/system/user', '/system/role', '/system/menu', '/system/permission'],
    layerKey: 'system.audit.sourceLayer.system',
    domainKey: 'system.audit.sourceDomain.iam',
    resolvePageKey: resolveIamSourcePageKey,
  },
  {
    fragments: ['/system/dept', '/system/post'],
    layerKey: 'system.audit.sourceLayer.system',
    domainKey: 'system.audit.sourceDomain.org',
    resolvePageKey: (path) =>
      path.includes('/system/dept')
        ? 'system.audit.sourcePage.dept'
        : 'system.audit.sourcePage.post',
  },
  {
    fragments: ['/dashboard'],
    layerKey: 'system.audit.sourceLayer.platform',
    domainKey: 'system.audit.sourceDomain.platform',
    resolvePageKey: () => 'system.audit.sourcePage.dashboard',
  },
];

function getAuditSourceMeta(log: OperationLogRow): AuditSourceMeta {
  const storedMeta = getStoredSourceMeta(log);
  if (storedMeta) {
    return storedMeta;
  }
  const path = log.operUrl.trim();
  const rule = AUDIT_SOURCE_PATH_RULES.find((candidate) =>
    candidate.fragments.some((fragment) => path.includes(fragment)),
  );
  if (rule) {
    return {
      layerKey: rule.layerKey,
      domainKey: rule.domainKey,
      pageKey: rule.resolvePageKey(path),
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
      color: failureCategoryColor(storedFailureCategory),
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

type TranslateFn = (key: string, options?: Record<string, unknown>) => string;

const i18nLifecycleActionKeys: Record<string, string> = {
  observe: 'i18n.lifecycle.observe',
  archive: 'i18n.lifecycle.archive',
  delete: 'i18n.lifecycle.delete',
};

function formatI18nLifecycleAction(value: string, t: TranslateFn) {
  const key = i18nLifecycleActionKeys[value];
  if (key) {
    return t(key);
  }
  return value || '-';
}

function formatI18nLifecycleLabel(value: string, t: TranslateFn) {
  return value ? t(`i18n.lifecycle.status.${value}`, { defaultValue: value }) : '-';
}

function toArcoSortOrder(sortOrder?: OperationLogQuery['sortOrder']) {
  if (sortOrder === 'asc') {
    return 'ascend';
  }
  if (sortOrder === 'desc') {
    return 'descend';
  }
  return undefined;
}

function parseTableSorter(sorter?: SorterInfo | SorterInfo[]): {
  sortField?: string;
  sortOrder?: 'asc' | 'desc';
} {
  const currentSorter = Array.isArray(sorter) ? sorter[0] : sorter;
  if (!currentSorter?.direction) {
    return { sortField: undefined, sortOrder: undefined };
  }
  return {
    sortField: String(currentSorter.field),
    sortOrder: currentSorter.direction === 'ascend' ? 'asc' : 'desc',
  };
}

interface DetailLookupHandlers {
  onFound: (log: OperationLogRow) => void;
  onLoadStart: () => void;
  onLoaded: (log: OperationLogRow) => void;
  onLoadError: () => void;
  onLoadEnd: () => void;
}

// setTimeout(0) wrappers keep setState out of the synchronous effect body
// (react-hooks/set-state-in-effect) — preserve them when editing.
function scheduleDetailLookup(
  detailId: number,
  rows: OperationLogRow[],
  handlers: DetailLookupHandlers,
) {
  const matchedLog = rows.find((item) => item.id === detailId);
  if (matchedLog) {
    const timer = globalThis.setTimeout(() => {
      handlers.onFound(matchedLog);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }
  const timer = globalThis.setTimeout(() => {
    handlers.onLoadStart();
    getOperationLog(detailId)
      .then((detail) => {
        handlers.onLoaded(detail);
      })
      .catch(() => {
        handlers.onLoadError();
      })
      .finally(() => {
        handlers.onLoadEnd();
      });
  }, 0);
  return () => globalThis.clearTimeout(timer);
}

function hasActiveOperationLogFilters(query: OperationLogQuery, advancedActiveCount: number) {
  return Boolean(
    query.keyword ||
    query.status !== undefined ||
    (query.sourceDomain !== undefined && query.sourceDomain !== '') ||
    query.startedAt ||
    advancedActiveCount > 0,
  );
}

const DetailSummaryHeader: React.FC<{ log: OperationLogRow }> = ({ log }) => {
  const { t } = useTranslation();
  const businessTypeMeta = getBusinessTypeMeta(log.businessType, t);
  const sourceMeta = getAuditSourceMeta(log);
  const methodText = normalizeMethod(log.method);
  return (
    <div className="audit-detail-summary">
      <div className="audit-detail-summary__copy">
        <Typography.Text className="audit-detail-summary__title">
          {t(log.title, { defaultValue: log.title || '-' })}
        </Typography.Text>
        <Typography.Text className="audit-detail-summary__desc" type="secondary">
          {log.operUrl || '-'}
        </Typography.Text>
      </div>
      <Space wrap>
        <Tag color={log.status === 1 ? 'green' : 'red'}>
          {log.status === 1 ? t('common.success') : t('common.failed')}
        </Tag>
        <Tag color={businessTypeMeta.color}>{businessTypeMeta.label}</Tag>
        <Tag color="purple">{t(sourceMeta.domainKey)}</Tag>
        <Tag color="cyan">{t(sourceMeta.pageKey)}</Tag>
        {methodText ? <Tag color="arcoblue">{methodText}</Tag> : null}
        <Tag color="gray">{t('system.audit.costTimeValue', { count: log.costTime })}</Tag>
      </Space>
    </div>
  );
};

const DetailBasicInfo: React.FC<{ log: OperationLogRow }> = ({ log }) => {
  const { t } = useTranslation();
  const sourceMeta = getAuditSourceMeta(log);
  const businessTypeMeta = getBusinessTypeMeta(log.businessType, t);
  const methodText = normalizeMethod(log.method);
  const handlerText = formatHandlerName(log.method);
  return (
    <Descriptions
      column={2}
      data={[
        {
          label: t('system.audit.title'),
          value: t(log.title, { defaultValue: log.title || '-' }),
        },
        {
          label: t('system.audit.operTime'),
          value: formatDateTime(log.operTime, { withSeconds: true }),
        },
        { label: t('system.audit.operName'), value: log.operName || '-' },
        { label: t('system.audit.operIp'), value: log.operIp || '-' },
        { label: t('system.audit.operUrl'), value: log.operUrl || '-' },
        { label: t('system.audit.sourceLayer'), value: t(sourceMeta.layerKey) },
        { label: t('system.audit.sourceDomain'), value: t(sourceMeta.domainKey) },
        { label: t('system.audit.sourcePage'), value: t(sourceMeta.pageKey) },
        { label: t('system.audit.businessType'), value: businessTypeMeta.label || '-' },
        { label: t('system.audit.method'), value: methodText || '-' },
        {
          label: t('system.audit.costTime'),
          value: t('system.audit.costTimeValue', { count: log.costTime }),
        },
        ...(handlerText ? [{ label: t('system.audit.handler'), value: handlerText }] : []),
      ]}
    />
  );
};

const DetailFailureCard: React.FC<{ log: OperationLogRow }> = ({ log }) => {
  const { t } = useTranslation();
  if (log.status !== 2) {
    return null;
  }
  const resultPreview = extractOperationResult(log.jsonResult);
  const failureMeta = getFailureMeta(log, resultPreview);
  const sourceMeta = getAuditSourceMeta(log);
  const translatedErrorText = log.errorMsg ? t(log.errorMsg, { defaultValue: log.errorMsg }) : '';
  const translatedResultMessage = resultPreview.message
    ? t(resultPreview.message, { defaultValue: resultPreview.message })
    : '';
  return (
    <Card
      className="detail-panel-card detail-panel-card--danger"
      title={t('system.audit.failureReason')}
      size="small"
    >
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Space wrap>
          {failureMeta ? <Tag color={failureMeta.color}>{t(failureMeta.typeKey)}</Tag> : null}
          <Tag color="purple">{t(sourceMeta.domainKey)}</Tag>
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
                  {
                    label: t('system.audit.failureSummary'),
                    value: t(failureMeta.summaryKey),
                  },
                ]
              : []),
            ...(translatedErrorText
              ? [{ label: t('system.audit.failureText'), value: translatedErrorText }]
              : []),
            ...(log.errorMsg ? [{ label: t('system.audit.errorKey'), value: log.errorMsg }] : []),
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
  );
};

const SettingAuditCard: React.FC<{ log: OperationLogRow }> = ({ log }) => {
  const { t } = useTranslation();
  const payload = extractSettingAuditPayload(log.operParam);
  if (!payload || payload.changes.length === 0) {
    return null;
  }
  return (
    <Card className="detail-panel-card" title={t('system.audit.requestSummary')} size="small">
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Space wrap>
          {payload.groupKey ? (
            <Tag color="arcoblue">
              {t(`system.setting.group.${payload.groupKey}`, payload.groupKey)}
            </Tag>
          ) : null}
          <Tag color="gold">{t('system.audit.changeCount', { count: payload.changes.length })}</Tag>
        </Space>
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          {payload.changes.map((change) => (
            <Space key={change.settingKey} size={8} wrap>
              <Typography.Text>
                {t(`system.setting.item.${change.settingKey}`, change.settingKey)}
              </Typography.Text>
              {change.isEncrypted === 1 ? (
                <Tag color="red">{t('system.setting.audit.sensitiveChanged')}</Tag>
              ) : (
                <>
                  <Typography.Text type="secondary">{change.oldValue || '-'}</Typography.Text>
                  <Typography.Text type="secondary">→</Typography.Text>
                  <Typography.Text>{change.newValue || '-'}</Typography.Text>
                </>
              )}
            </Space>
          ))}
        </Space>
      </Space>
    </Card>
  );
};

const I18nLifecycleParamCard: React.FC<{ log: OperationLogRow }> = ({ log }) => {
  const { t } = useTranslation();
  const payload = extractI18nLifecycleAuditPayload(log.operParam);
  if (!payload) {
    return null;
  }
  return (
    <Card className="detail-panel-card" title={t('system.audit.requestSummary')} size="small">
      <Descriptions
        column={2}
        data={[
          {
            label: t('i18n.audit.action'),
            value: formatI18nLifecycleAction(payload.action, t),
          },
          { label: t('common.module'), value: payload.module || '-' },
          {
            label: t('i18n.lifecycle.fromStatus'),
            value: formatI18nLifecycleLabel(payload.fromStatus, t),
          },
          {
            label: t('i18n.lifecycle.toStatus'),
            value: formatI18nLifecycleLabel(payload.toStatus, t),
          },
          ...(typeof payload.observationThresholdDays === 'number'
            ? [
                {
                  label: t('i18n.lifecycle.observationThreshold'),
                  value: t('i18n.lifecycle.observationThresholdValue', {
                    count: payload.observationThresholdDays,
                  }),
                },
              ]
            : []),
          ...(typeof payload.confirmArchived === 'boolean'
            ? [
                {
                  label: t('i18n.lifecycle.confirmArchived'),
                  value: payload.confirmArchived ? t('common.yes') : t('common.no'),
                },
              ]
            : []),
          ...(payload.affectedKeys.length > 0
            ? [
                {
                  label: t('i18n.audit.affectedKeys'),
                  value: payload.affectedKeys.join(', '),
                },
              ]
            : []),
        ]}
      />
    </Card>
  );
};

const I18nLifecycleResultCard: React.FC<{ payload: I18nLifecycleAuditPreview }> = ({ payload }) => {
  const { t } = useTranslation();
  return (
    <Card className="detail-panel-card" title={t('system.audit.resultSummary')} size="small">
      <Descriptions
        column={2}
        data={[
          {
            label: t('i18n.audit.action'),
            value: formatI18nLifecycleAction(payload.action, t),
          },
          { label: t('common.module'), value: payload.module || '-' },
          {
            label: t('i18n.lifecycle.fromStatus'),
            value: formatI18nLifecycleLabel(payload.fromStatus, t),
          },
          {
            label: t('i18n.lifecycle.toStatus'),
            value: formatI18nLifecycleLabel(payload.toStatus, t),
          },
          {
            label: t('system.audit.responseData'),
            value: t('i18n.lifecycle.affectedRowsValue', {
              count: payload.affectedRows || 0,
            }),
          },
          {
            label: t('i18n.audit.affectedKeys'),
            value: payload.affectedKeys.length > 0 ? payload.affectedKeys.join(', ') : '-',
          },
        ]}
      />
    </Card>
  );
};

const PlainResultSummaryCard: React.FC<{ log: OperationLogRow }> = ({ log }) => {
  const { t } = useTranslation();
  const resultPreview = extractOperationResult(log.jsonResult);
  const translatedResultMessage = resultPreview.message
    ? t(resultPreview.message, { defaultValue: resultPreview.message })
    : '';
  const hasContent =
    typeof resultPreview.code === 'number' ||
    Boolean(translatedResultMessage) ||
    resultPreview.data !== undefined;
  if (!hasContent) {
    return null;
  }
  return (
    <Card className="detail-panel-card" title={t('system.audit.resultSummary')} size="small">
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
                      : String(resultPreview.data as Exclude<JsonValue, object>),
                },
              ]
            : []),
        ]}
      />
    </Card>
  );
};

const DetailResultSection: React.FC<{ log: OperationLogRow }> = ({ log }) => {
  const lifecycleResult = extractI18nLifecycleAuditResult(log.jsonResult);
  if (lifecycleResult) {
    return <I18nLifecycleResultCard payload={lifecycleResult} />;
  }
  return <PlainResultSummaryCard log={log} />;
};

const rawPreStyle: React.CSSProperties = {
  margin: 0,
  whiteSpace: 'pre-wrap',
  maxHeight: 220,
  overflow: 'auto',
};

const OperationLogDetail: React.FC<{ log: OperationLogRow }> = ({ log }) => {
  const { t } = useTranslation();
  return (
    <Space direction="vertical" size={16} className="detail-stack">
      <DetailSummaryHeader log={log} />
      <DetailBasicInfo log={log} />
      <DetailFailureCard log={log} />
      <SettingAuditCard log={log} />
      <I18nLifecycleParamCard log={log} />
      <DetailResultSection log={log} />
      <Card className="detail-panel-card" title={t('system.audit.operParam')} size="small">
        <pre style={rawPreStyle}>{formatAuditRaw(log.operParam)}</pre>
      </Card>
      <Card className="detail-panel-card" title={t('system.audit.jsonResult')} size="small">
        <pre style={rawPreStyle}>{formatAuditRaw(log.jsonResult)}</pre>
      </Card>
    </Space>
  );
};

const BatchActionBar: React.FC<{
  canDelete: boolean;
  selectedCount: number;
  onClearSelection: () => void;
  onBatchDelete: () => void;
}> = ({ canDelete, selectedCount, onClearSelection, onBatchDelete }) => {
  const { t } = useTranslation();
  return (
    <>
      <Typography.Text type="secondary">
        {t('common.selectedCount', { count: selectedCount })}
      </Typography.Text>
      <Button type="text" size="small" disabled={selectedCount === 0} onClick={onClearSelection}>
        {t('common.clearSelection')}
      </Button>
      <PermissionAction allowed={canDelete} tooltip={t('common.noPermissionAction')}>
        <Popconfirm
          disabled={selectedCount === 0 || !canDelete}
          title={t('system.audit.batchDeleteConfirm', { count: selectedCount })}
          onOk={onBatchDelete}
        >
          <Button
            status="danger"
            icon={<IconDelete />}
            disabled={selectedCount === 0 || !canDelete}
          >
            {t('common.deleteSelected')}
          </Button>
        </Popconfirm>
      </PermissionAction>
    </>
  );
};

const OperationLogList: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canExport = isAdmin || hasPerm('system:operation-log:export');
  const canDelete = isAdmin || hasPerm('system:operation-log:delete');
  const canClear = isAdmin || hasPerm('system:operation-log:clear');
  const governanceRail = useGovernanceRail();
  const [data, setData] = useState<OperationLogRow[]>([]);
  const [total, setTotal] = useState(0);
  const [successCount, setSuccessCount] = useState(0);
  const [failedCount, setFailedCount] = useState(0);
  const [retentionDays, setRetentionDays] = useState<number>(30);
  const [retentionOptions, setRetentionOptions] = useState<number[]>(() =>
    [...defaultRetentionOptions].sort((left, right) => right - left),
  );
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<unknown>(null);
  const [query, setQuery] = useState<OperationLogQuery>(emptyQuery);
  const [detailVisible, setDetailVisible] = useState(false);
  const [currentLog, setCurrentLog] = useState<OperationLogRow | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const [selectedRowKeys, setSelectedRowKeys] = useState<CrossPageRowKey[]>([]);

  const loadData = useCallback(
    async (nextQuery: OperationLogQuery = query) => {
      setLoading(true);
      setLoadError(null);
      try {
        const result = await getOperationLogList(nextQuery);
        setData(result.items);
        setTotal(result.total);
        setSuccessCount(result.successCount ?? 0);
        setFailedCount(result.failedCount ?? 0);
      } catch (requestError) {
        setLoadError(requestError);
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

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      getSettingGroup('audit')
        .then((group: SettingGroup) =>
          loadRetentionSetting(
            group,
            'audit.operation_log_retention_options',
            setRetentionOptions,
            setRetentionDays,
          ),
        )
        .catch(() => undefined);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, []);

  useEffect(() => {
    const detailId = Number(searchParams.get('detailId') || 0);
    if (!detailId || Number.isNaN(detailId)) {
      return;
    }

    return scheduleDetailLookup(detailId, data, {
      onFound: (log) => {
        setCurrentLog(log);
        setDetailVisible(true);
        setDetailLoading(false);
      },
      onLoadStart: () => setDetailLoading(true),
      onLoaded: (detail) => {
        setCurrentLog(detail);
        setDetailVisible(true);
      },
      onLoadError: () => message.error(t('common.loadFailed')),
      onLoadEnd: () => setDetailLoading(false),
    });
  }, [data, searchParams, t]);

  const search = (values: Partial<OperationLogQuery>) => {
    setSelectedRowKeys([]);
    setQuery((current) => ({
      ...current,
      ...values,
      page: 1,
    }));
  };

  const reset = () => {
    setSelectedRowKeys([]);
    setQuery(emptyQuery);
  };

  const advancedActiveCount = [query.failureCategory, query.sourcePage].filter(
    (value) => value !== undefined && value !== '',
  ).length;

  const hasActiveFilters = hasActiveOperationLogFilters(query, advancedActiveCount);

  const handleDelete = async (id: number) => {
    try {
      await deleteOperationLog(id);
      message.success(t('system.audit.deleteSuccess'));
      void loadData();
    } catch {
      // message.error already handled by request interceptor
    }
  };

  const handleTimeRangeChange = (value: Required<TimeRangeFilterValue>) => {
    setSelectedRowKeys([]);
    setQuery((current) => ({ ...current, ...value, page: 1 }));
  };

  const handleCleanup = async (payload: GovernanceCleanupPayload) => {
    try {
      const resp =
        payload.mode === 'range'
          ? await cleanupOperationLogs({
              startedAt: payload.startedAt,
              endedAt: payload.endedAt,
            })
          : await cleanupOperationLogs({ retentionDays: payload.retentionDays });
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
    const ids = selectedRowKeys.map(Number);
    if (ids.some((id) => !Number.isSafeInteger(id) || id <= 0)) {
      message.error(t('common.actionFailed'));
      return;
    }
    try {
      const resp = await batchDeleteOperationLogs({ ids });
      message.success(t('system.audit.batchDeleteSuccess', { count: resp.deletedCount }));
      setSelectedRowKeys([]);
      void loadData();
    } catch {
      message.error(t('common.actionFailed'));
    }
  };

  const handleClearSelection = () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    setSelectedRowKeys([]);
    message.success(t('common.clearSelectionSuccess'));
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

  const sortableColumn = (
    field: NonNullable<OperationLogQuery['sortField']>,
  ): Partial<ColumnProps<OperationLogRow>> => ({
    sorter: true,
    sortOrder: query.sortField === field ? toArcoSortOrder(query.sortOrder) : undefined,
  });

  const handleTableChange: TableProps<OperationLogRow>['onChange'] = (pagination, sorter) => {
    const { sortField: nextSortField, sortOrder: nextSortOrder } = parseTableSorter(sorter);
    if (nextSortField !== query.sortField || nextSortOrder !== query.sortOrder) {
      setSelectedRowKeys([]);
    }
    setQuery({
      ...query,
      page: pagination.current || 1,
      pageSize: pagination.pageSize || query.pageSize || emptyQuery.pageSize,
      sortField: nextSortField,
      sortOrder: nextSortOrder,
    });
  };

  const columns: ColumnProps<OperationLogRow>[] = [
    {
      title: t('system.audit.title'),
      dataIndex: 'title',
      width: TABLE_COLUMN_WIDTH.tagGroup,
      ellipsis: true,
      ...sortableColumn('title'),
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
        ...sortableColumn('operName'),
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
        // Backend stores a stable i18n key (location.*); legacy rows keep raw
        // text, which passes through via defaultValue.
        render: (value: string) => (value ? t(value, { defaultValue: value }) : '-'),
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
      ...sortableColumn('status'),
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
        // Full "YYYY-MM-DD HH:mm:ss" plus the sort affordance needs more than the
        // shared datetime width, otherwise the timestamp wraps to two lines.
        width: 200,
        ...sortableColumn('operTime'),
        render: (value: string) => (
          <Typography.Text className="system-list__datetime-text">
            {formatDateTime(value, { withSeconds: true })}
          </Typography.Text>
        ),
      },
      'medium',
    ),
    {
      title: t('common.action'),
      fixed: 'right',
      width: TABLE_ACTION_COLUMN_WIDTH.compact,
      render: (_, record) => (
        <SystemRowActions
          actions={[
            {
              key: 'detail',
              text: t('common.detail'),
              icon: <IconEye />,
              onClick: () => showDetail(record),
            },
            {
              key: 'delete',
              text: t('common.delete'),
              icon: <IconDelete />,
              hidden: !canDelete,
              status: 'danger',
              confirm: {
                title: t('common.deleteConfirm'),
                onOk: () => handleDelete(record.id),
              },
            },
          ]}
        />
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
    ],
    [canExport, failedCount, successCount, t, total],
  );

  const tableContent =
    data.length === 0 && !loading ? (
      <PageEmpty />
    ) : (
      <AppTable<OperationLogRow>
        className="system-list__table"
        rowKey="id"
        data={data}
        columns={columns}
        loading={loading}
        scroll={{ x: 'max-content' }}
        onChange={handleTableChange}
        rowSelection={
          canDelete
            ? buildCrossPageRowSelection({
                rows: data,
                getRowKey: (row) => row.id,
                selectedRowKeys,
                setSelectedRowKeys,
              })
            : undefined
        }
        pagination={buildStandardPagination(t, {
          current: query.page || emptyQuery.page,
          pageSize: query.pageSize || emptyQuery.pageSize,
          total,
        })}
      />
    );

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
          <SearchToolbar
            keyword={query.keyword ?? ''}
            keywordPlaceholder={t('system.audit.search.placeholder')}
            onKeywordChange={(keyword) => search({ keyword })}
            inlineFilters={
              <>
                <Select
                  allowClear
                  placeholder={t('system.audit.status')}
                  value={query.status}
                  onChange={(value) => search({ status: value })}
                  options={[
                    { label: t('common.success'), value: 1 },
                    { label: t('common.failed'), value: 2 },
                  ]}
                />
                <Select
                  allowClear
                  placeholder={t('system.audit.sourceDomain')}
                  value={query.sourceDomain}
                  onChange={(value) => search({ sourceDomain: value })}
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
                <TimeRangeFilter
                  value={{ startedAt: query.startedAt, endedAt: query.endedAt }}
                  onChange={handleTimeRangeChange}
                />
              </>
            }
            advancedFilters={
              <>
                <div className="search-toolbar__popover-field">
                  <label>{t('system.audit.failureCategory')}</label>
                  <Select
                    allowClear
                    placeholder={t('system.audit.failureCategory')}
                    value={query.failureCategory}
                    onChange={(value) => search({ failureCategory: value })}
                    options={[
                      { label: t('system.audit.failureType.validation'), value: 'validation' },
                      { label: t('system.audit.failureType.auth'), value: 'auth' },
                      { label: t('system.audit.failureType.permission'), value: 'permission' },
                      { label: t('system.audit.failureType.server'), value: 'server' },
                      { label: t('system.audit.failureType.business'), value: 'business' },
                    ]}
                  />
                </div>
                <div className="search-toolbar__popover-field">
                  <label>{t('system.audit.sourcePage')}</label>
                  <Select
                    allowClear
                    placeholder={t('system.audit.sourcePage')}
                    value={query.sourcePage}
                    onChange={(value) => search({ sourcePage: value })}
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
                </div>
              </>
            }
            advancedActiveCount={advancedActiveCount}
            hasActiveFilters={hasActiveFilters}
            onClearAll={reset}
          />

          <Card className="page-panel system-list__table-card">
            <GovernanceCleanupBar
              showCleanup={canClear}
              retentionDays={retentionDays}
              retentionOptions={retentionOptions}
              onRetentionChange={setRetentionDays}
              retentionLabel={(option) => t('common.keepRecentDays', { count: option })}
              confirmTitle={t('common.cleanupIrreversibleWarning')}
              actionLabel={t('common.cleanupLogs')}
              confirmActionLabel={t('common.cleanup')}
              cleanupModeLabel={t('common.cleanupMode')}
              cleanupModeOptions={[
                { label: t('common.cleanupModeRetention'), value: 'retention' },
                { label: t('common.cleanupModeRange'), value: 'range' },
              ]}
              rangeStartLabel={t('common.cleanupRangeStart')}
              rangeEndLabel={t('common.cleanupRangeEnd')}
              rangeRequiredMessage={t('common.cleanupRangeRequired')}
              onConfirm={handleCleanup}
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
                  <BatchActionBar
                    canDelete={canDelete}
                    selectedCount={selectedRowKeys.length}
                    onClearSelection={handleClearSelection}
                    onBatchDelete={() => {
                      void handleBatchDelete();
                    }}
                  />
                ) : undefined
              }
            />

            {loading && data.length === 0 ? <PageLoading /> : null}
            {loadError && !loading ? (
              <PageRequestError
                error={loadError}
                onRetry={() => {
                  void loadData(query);
                }}
              />
            ) : (
              tableContent
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
        {currentLog && <OperationLogDetail log={currentLog} />}
      </AppModal>
    </PageContainer>
  );
};

export default OperationLogList;
