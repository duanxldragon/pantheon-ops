import { Message } from '@arco-design/web-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import {
  refreshPublicSettings,
  hasExplicitLanguagePreference,
} from '../../../core/settings/publicSettings';
import { invalidateRouteWarmData, resolveRouteWarmData } from '../../../core/router/prefetch';
import {
  applyPantheonDefaultTheme,
  clearPantheonThemePreference,
  type PantheonThemeKey,
} from '../../../core/theme/theme';
import { switchI18nLanguage } from '../../../i18n';
import {
  getSettingAuditList,
  getSettingList,
  getSettingOverview,
  type SettingAuditRow,
  type SettingItem,
  type SettingOverviewResp,
} from './api';
import { settingGroupOrder } from './settingGroups';

export const defaultAuditPageSize = 5;
export const auditRetentionSettingKeys = new Set([
  'audit.login_log_retention_options',
  'audit.operation_log_retention_options',
  'audit.session_cleanup_retention_options',
]);
export const auditRetentionDaySettingKeys = new Set([
  'audit.login_log_retention_days',
  'audit.operation_log_retention_days',
  'audit.session_retention_days',
]);
export const integerSettingKeys = new Set([
  'security.password_min_length',
  'login.max_failed_attempts',
  'login.lock_minutes',
  'login.session_idle_minutes',
  'upload.max_file_size',
]);
export const recommendedAuditRetentionOptions = [1, 7, 30, 90, 180, 365];
export const recommendedAuditRetentionDayOptions = [30, 90, 180, 365];

export type SettingFormValue = string | number | boolean | string[] | undefined;

export interface GroupedSettingItems {
  groupKey: string;
  items: SettingItem[];
}

export function normalizeAuditRetentionTagValues(rawValues: Array<string | number>) {
  const normalized = Array.from(
    new Set(
      rawValues
        .map((item) => Number(String(item).trim()))
        .filter((item) => Number.isInteger(item) && item > 0 && item <= 365),
    ),
  ).sort((left, right) => left - right);
  return normalized.map((item) => String(item));
}

export function parseAuditRetentionSettingValue(rawValue: string) {
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

export function resolveAuditRetentionDefaultValues(item: SettingItem) {
  const parsedDefaults = parseAuditRetentionSettingValue(item.defaultValue);
  if (parsedDefaults.length > 0) {
    return parsedDefaults;
  }
  return ['1', '7', '30'];
}

export function parseDefaultFieldValue(item: SettingItem): SettingFormValue {
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

export function isSameStringArray(left: string[], right: string[]) {
  if (left.length !== right.length) {
    return false;
  }
  return left.every((value, index) => value === right[index]);
}

export function buildFormValues(items: SettingItem[]) {
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

export function serializeSettingValue(item: SettingItem, rawValue: SettingFormValue) {
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

export function readFormFieldValue(
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

export function invalidateSettingCaches() {
  invalidateRouteWarmData('/system/setting', ['list:default', 'overview']);
}

export function notifySettingChanged() {
  invalidateSettingCaches();
  publishRefresh('system:setting:changed', 'system/setting');
}

export async function runSettingGroupPostSaveEffects(
  groupKey: string,
  values: Record<string, SettingFormValue>,
) {
  if (groupKey === 'ui') {
    const nextTheme = values['ui.default_theme'];
    if (typeof nextTheme === 'string') {
      applyPantheonDefaultTheme(nextTheme as PantheonThemeKey);
      clearPantheonThemePreference();
    }
  }
  if (['basic', 'platform', 'i18n', 'ui'].includes(groupKey)) {
    const publicSettings = await refreshPublicSettings().catch(() => null);
    if (groupKey === 'i18n' && publicSettings && !hasExplicitLanguagePreference()) {
      await switchI18nLanguage(publicSettings.defaultLanguage).catch(() => undefined);
    }
  }
}

export function useSettingCatalog() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [settings, setSettings] = useState<SettingItem[]>([]);
  const [overview, setOverview] = useState<SettingOverviewResp | null>(null);
  const [auditRows, setAuditRows] = useState<SettingAuditRow[]>([]);
  const [auditTotal, setAuditTotal] = useState(0);
  const [auditLoading, setAuditLoading] = useState(false);
  const [auditQuery, setAuditQuery] = useState({ page: 1, pageSize: defaultAuditPageSize });

  const reload = useCallback(async () => {
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
    const timer = globalThis.setTimeout(() => {
      void reload();
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [reload]);

  useRefreshSubscription('system:setting:changed', () => {
    invalidateSettingCaches();
    void reload();
  });

  const groupedSettings = useMemo<GroupedSettingItems[]>(() => {
    const buckets = new Map<string, SettingItem[]>();
    settings.forEach((item) => {
      const list = buckets.get(item.groupKey) || [];
      list.push(item);
      buckets.set(item.groupKey, list);
    });
    return settingGroupOrder
      .filter((groupKey) => buckets.has(groupKey))
      .map((groupKey) => ({ groupKey, items: buckets.get(groupKey) || [] }));
  }, [settings]);

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
        return result;
      } catch {
        Message.error(t('common.loadFailed'));
        return null;
      } finally {
        setAuditLoading(false);
      }
    },
    [t],
  );

  const clearAudit = useCallback(() => {
    setAuditRows([]);
    setAuditTotal(0);
  }, []);

  return {
    loading,
    error,
    settings,
    overview,
    groupedSettings,
    auditRows,
    auditTotal,
    auditLoading,
    auditQuery,
    reload,
    loadAudit,
    clearAudit,
    setAuditQuery,
  };
}
