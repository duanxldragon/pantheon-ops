import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Card,
  Button,
  Checkbox,
  Descriptions,
  Form,
  Grid,
  Input,
  Popconfirm,
  Select,
  Space,
  Tag,
  Typography,
  List,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import {
  IconDelete,
  IconEdit,
  IconEye,
  IconRefresh,
  IconSearch,
} from '@arco-design/web-react/icon';
import { IconDownload } from '@arco-design/web-react/icon';
import type { TFunction } from 'i18next';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import { showImportResult } from '../../../api/importExport';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../../components/table/crossPageSelection';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  FilterPanel,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  ImportCsvButton,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
  PermissionAction,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  TableBatchActionBar,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../../components';
import { usePermission } from '../../../hooks/usePermission';
import { SUPPORTED_LOCALES, reloadI18nResources } from '../../../i18n';
import { isRequestError } from '../../../api/request';
import { getRegisteredModules } from '../dynamicmodule/api';
import {
  archiveObservedUnusedKeys,
  batchDeleteI18n,
  createI18n,
  deleteArchivedUnusedKeys,
  deleteI18n,
  downloadI18nImportTemplate,
  exportI18n,
  fillI18nMissingLocales,
  getI18nAudit,
  getI18nDetail,
  getI18nList,
  getI18nMissingLocales,
  getI18nOverview,
  hydrateBuiltinI18nLocales,
  importI18n,
  previewI18nRename,
  refreshI18nCache,
  refreshI18nLocales,
  renameI18nKey,
  startUnusedObservation,
  syncI18nKeys,
  type I18nCreateReq,
  type I18nAuditResp,
  type I18nOverviewResp,
  type I18nMissingLocaleItem,
  type I18nRenamePreviewResp,
  type I18nRenameSuggestion,
  type I18nStalePlaceholderItem,
  updateI18n,
  type I18nQuery,
  type SystemI18n,
} from './api';
import '../list-page.css';

interface I18nRenameFormValues {
  module: string;
  sourceKey?: string;
  oldKey: string;
  newKey: string;
  confirmSourceUpdated: boolean;
}

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;
const { Text } = Typography;

interface I18nDuplicateConflictState {
  key: string;
  locale: string;
  module?: string;
}

function buildRenameMigrationReport(preview: I18nRenamePreviewResp, t: TFunction) {
  const lines: string[] = [
    `# ${t('i18n.rename.report.title')}`,
    '',
    `${t('i18n.rename.report.module')}: ${preview.module}`,
    `${t('i18n.rename.oldKey')}: ${preview.oldKey}`,
    `${t('i18n.rename.newKey')}: ${preview.newKey}`,
    `${t('i18n.rename.preview.affectedRows', { count: preview.affectedRows })}`,
    `${t('i18n.rename.report.affectedLocales')}: ${preview.affectedLocales.join(', ') || '-'}`,
    `${t('i18n.rename.report.existingTargetRows')}: ${preview.existingTargetRows}`,
    `${t('i18n.rename.report.existingTargetLocales')}: ${preview.existingTargetLocales.join(', ') || '-'}`,
    `${t('i18n.rename.report.requiresCodeMigration')}: ${preview.requiresCodeMigration ? t('common.yes') : t('common.no')}`,
    '',
    `## ${t('i18n.rename.preview.referenceTitle')}`,
  ];

  if (preview.referenceFiles.length === 0) {
    lines.push(`- ${t('i18n.rename.report.referenceEmpty')}`);
  } else {
    preview.referenceFiles.forEach((file) => {
      lines.push('');
      lines.push(`- ${t('i18n.rename.report.referenceFile')}: ${file.path}`);
      lines.push(`  ${t('i18n.rename.report.referenceMatches')}: ${file.matchCount}`);
      lines.push(
        `  ${t('i18n.rename.report.referenceSuggestedReplacement')}: ${file.suggestedReplacement || preview.newKey}`,
      );
      file.matches.forEach((match) => {
        lines.push(
          `  - ${t('i18n.rename.report.referenceLocation', { line: match.line, column: match.column })}`,
        );
        lines.push(`    ${t('i18n.rename.report.referenceBefore')}: ${match.snippet}`);
        lines.push(`    ${t('i18n.rename.report.referenceAfter')}: ${match.replacementHint}`);
      });
    });
  }

  lines.push('');
  lines.push(`## ${t('i18n.rename.report.checklistTitle')}`);
  lines.push(`1. ${t('i18n.rename.report.checklist1')}`);
  lines.push(`2. ${t('i18n.rename.report.checklist2')}`);
  lines.push(`3. ${t('i18n.rename.report.checklist3')}`);
  lines.push(`4. ${t('i18n.rename.report.checklist4')}`);
  lines.push('');
  return lines.join('\n');
}

const emptyQuery: I18nQuery = {
  module: '',
  group: '',
  locale: '',
  key: '',
  page: 1,
  pageSize: 10,
};

function requiredRule(t: TFunction, labelKey: string) {
  return { required: true, message: t('common.requiredField', { field: t(labelKey) }) };
}

interface LoadDataOptions {
  silent?: boolean;
}

const I18nList: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canEdit = isAdmin || hasPerm('system:i18n:update');
  const canCreate = isAdmin || hasPerm('system:i18n:create');
  const canDelete = isAdmin || hasPerm('system:i18n:delete');
  const canExport = isAdmin || hasPerm('system:i18n:export');
  const canImport = isAdmin || hasPerm('system:i18n:import');
  const canRefresh = isAdmin || hasPerm('system:i18n:refresh');
  const canHydrateBuiltin = canEdit || canImport;

  const [query, setQuery] = useState<I18nQuery>(emptyQuery);
  const [rows, setRows] = useState<SystemI18n[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailLoadingKey, setDetailLoadingKey] = useState('');
  const [detailVisible, setDetailVisible] = useState(false);
  const [createVisible, setCreateVisible] = useState(false);
  const [editVisible, setEditVisible] = useState(false);
  const [syncVisible, setSyncVisible] = useState(false);
  const [currentRow, setCurrentRow] = useState<SystemI18n | null>(null);
  const [syncedKeys, setSyncedKeys] = useState<string[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<(string | number)[]>([]);
  const [form] = Form.useForm();
  const [createForm] = Form.useForm();
  const [queryForm] = Form.useForm<I18nQuery>();
  const [overview, setOverview] = useState<I18nOverviewResp | null>(null);
  const [missingLocaleRows, setMissingLocaleRows] = useState<I18nMissingLocaleItem[]>([]);
  const [missingLocaleVisible, setMissingLocaleVisible] = useState(false);
  const [fillingMissingLocales, setFillingMissingLocales] = useState(false);
  const [hydratingBuiltinLocales, setHydratingBuiltinLocales] = useState(false);
  const [missingLocaleModuleFilter, setMissingLocaleModuleFilter] = useState('');
  const [auditVisible, setAuditVisible] = useState(false);
  const [auditLoading, setAuditLoading] = useState(false);
  const [unusedLifecycleLoading, setUnusedLifecycleLoading] = useState(false);
  const [audit, setAudit] = useState<I18nAuditResp | null>(null);
  const [renameVisible, setRenameVisible] = useState(false);
  const [renamePreviewLoading, setRenamePreviewLoading] = useState(false);
  const [renameSubmitting, setRenameSubmitting] = useState(false);
  const [renamePreview, setRenamePreview] = useState<I18nRenamePreviewResp | null>(null);
  const [renameForm] = Form.useForm<I18nRenameFormValues>();
  const [registeredModuleOptions, setRegisteredModuleOptions] = useState<string[]>([]);
  const [createDuplicateConflict, setCreateDuplicateConflict] =
    useState<I18nDuplicateConflictState | null>(null);
  const [secondaryReady, setSecondaryReady] = useState(false);
  const detailRequestKeyRef = useRef('');
  const secondaryBootstrapScheduledRef = useRef(false);
  const governanceRail = useGovernanceRail();

  const loadData = useCallback(
    async (nextQuery: I18nQuery = query, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setLoading(true);
        setError(null);
      }
      try {
        const resp = await getI18nList(nextQuery);
        setRows(resp.items);
        setTotal(resp.total);
        if (!secondaryBootstrapScheduledRef.current) {
          secondaryBootstrapScheduledRef.current = true;
          globalThis.setTimeout(() => {
            setSecondaryReady(true);
          }, 0);
        }
      } catch (requestError) {
        setError(requestError);
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [query],
  );

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      void loadData(query);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadData, query]);

  useEffect(() => {
    if (!secondaryReady) {
      return;
    }
    getRegisteredModules()
      .then((modules) => {
        setRegisteredModuleOptions(
          modules
            .map((item) => item.name)
            .filter(Boolean)
            .sort(),
        );
      })
      .catch(() => setRegisteredModuleOptions([]));
  }, [secondaryReady]);

  const loadOverview = useCallback(async () => {
    try {
      const resp = await getI18nOverview();
      setOverview(resp);
    } catch {
      setOverview(null);
    }
  }, []);

  const loadAudit = useCallback(async () => {
    setAuditLoading(true);
    try {
      const resp = await getI18nAudit();
      setAudit(resp);
    } catch {
      setAudit(null);
      message.error(t('i18n.audit.error'));
    } finally {
      setAuditLoading(false);
    }
  }, [t]);

  const loadMissingLocales = useCallback(async (module?: string) => {
    try {
      const resp = await getI18nMissingLocales(module);
      setMissingLocaleRows(resp.items || []);
    } catch {
      setMissingLocaleRows([]);
    }
  }, []);

  const reloadMissingLocaleRows = useCallback(async () => {
    await loadMissingLocales(
      missingLocaleVisible ? missingLocaleModuleFilter || undefined : undefined,
    );
  }, [loadMissingLocales, missingLocaleModuleFilter, missingLocaleVisible]);

  useEffect(() => {
    if (!secondaryReady) {
      return undefined;
    }
    const timer = globalThis.setTimeout(() => {
      void loadOverview();
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadOverview, secondaryReady]);

  useRefreshSubscription('system:i18n:changed', (payload) => {
    if (payload.source === 'system/i18n') {
      return;
    }
    void loadData(query);
    if (secondaryReady) {
      void loadOverview();
    }
    void reloadMissingLocaleRows();
    if (auditVisible) {
      void loadAudit();
    }
  });

  useEffect(() => {
    if (!missingLocaleVisible) {
      return;
    }
    const timer = globalThis.setTimeout(() => {
      void loadMissingLocales(missingLocaleModuleFilter || undefined);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadMissingLocales, missingLocaleModuleFilter, missingLocaleVisible]);

  const moduleOptions = useMemo(
    () =>
      Array.from(
        new Set([...registeredModuleOptions, ...rows.map((item) => item.module).filter(Boolean)]),
      ).sort((a, b) => a.localeCompare(b)),
    [registeredModuleOptions, rows],
  );
  const groupOptions = useMemo(
    () => Array.from(new Set(rows.map((item) => item.group).filter(Boolean))).sort((a, b) => a.localeCompare(b)),
    [rows],
  );

  const summary = useMemo(
    () => ({
      total,
      selected: selectedRowKeys.length,
    }),
    [selectedRowKeys.length, total],
  );
  const heroStats = useMemo(
    () => [
      {
        key: 'entries',
        label: t('i18n.stats.entries', { count: overview?.totalEntries || total }),
        value: overview?.totalEntries || total,
        hint: t('i18n.hero.entriesHint'),
      },
      {
        key: 'modules',
        label: t('i18n.hero.modules'),
        value: overview?.moduleCount || moduleOptions.length,
        hint: t('i18n.hero.modulesHint'),
      },
      {
        key: 'missing',
        label: t('i18n.hero.missingLocales'),
        value: overview?.missingLocaleCount || 0,
        hint: t('i18n.hero.missingHint'),
      },
      {
        key: 'selected',
        label: t('common.selected'),
        value: summary.selected,
        hint: t('i18n.hero.selectedHint'),
      },
    ],
    [
      moduleOptions.length,
      overview?.missingLocaleCount,
      overview?.moduleCount,
      overview?.totalEntries,
      summary.selected,
      t,
      total,
    ],
  );

  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('i18n.hero.groups'),
        value: overview?.groupCount || groupOptions.length,
        description: t('i18n.hero.groupsHint'),
      },
      {
        tone: 'warning' as const,
        label: t('i18n.hero.missingValues'),
        value: overview?.missingValueCount || 0,
        description: t('i18n.hero.missingValuesHint'),
      },
      {
        label: t('i18n.hero.refreshReady'),
        value: canRefresh ? t('common.yes') : t('common.no'),
        description: t('i18n.hero.refreshHint'),
      },
      ...(overview?.coverage.map((item) => ({
        label: item.locale,
        value: item.entryCount,
        description: t('i18n.stats.localeCoverage', {
          locale: item.locale,
          entries: item.entryCount,
          missing: item.missingCount,
        }),
      })) || []),
    ],
    [canRefresh, groupOptions.length, overview, t],
  );

  const handleSearch = () => {
    const values = queryForm.getFieldsValue();
    setSelectedRowKeys([]);
    setQuery({
      ...query,
      ...values,
      page: 1,
    });
  };

  const handleReset = () => {
    queryForm.setFieldsValue(emptyQuery);
    setSelectedRowKeys([]);
    setQuery(emptyQuery);
  };

  const locateToList = (nextQuery: Partial<I18nQuery>) => {
    const mergedQuery: I18nQuery = {
      ...emptyQuery,
      ...query,
      ...nextQuery,
      page: 1,
    };
    queryForm.setFieldsValue(mergedQuery);
    setQuery(mergedQuery);
    setAuditVisible(false);
  };

  const loadDetail = async (row: SystemI18n, mode: 'view' | 'edit') => {
    const requestKey = `${row.id}:${mode}`;
    if (detailRequestKeyRef.current === requestKey) {
      return;
    }
    detailRequestKeyRef.current = requestKey;
    setDetailLoadingKey(requestKey);
    setDetailLoading(true);
    try {
      const detail = await getI18nDetail(String(row.id));
      setCurrentRow(detail);
      if (mode === 'view') {
        setDetailVisible(true);
      } else {
        form.setFieldsValue({
          value: detail.value,
          remark: detail.remark,
        });
        setEditVisible(true);
      }
    } catch {
      message.error(t('i18n.detail.error'));
    } finally {
      detailRequestKeyRef.current = '';
      setDetailLoadingKey('');
      setDetailLoading(false);
    }
  };

  const handleDelete = async (row: SystemI18n) => {
    try {
      await deleteI18n(String(row.id));
      await reloadI18nResources();
      message.success(t('common.deleteSuccess'));
      publishRefresh('system:i18n:changed', 'system/i18n');
      await loadData(query, { silent: true });
      await loadOverview();
      await reloadMissingLocaleRows();
      if (auditVisible) {
        await loadAudit();
      }
    } catch {
      message.error(t('i18n.delete.error'));
    }
  };

  const handleRetryLoadData = useCallback(() => {
    void loadData(query);
  }, [loadData, query]);

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    try {
      const deletedCount = selectedRowKeys.length;
      await batchDeleteI18n(selectedRowKeys.map(String));
      await reloadI18nResources();
      message.success(t('i18n.batchDeleteSuccess', { count: deletedCount }));
      setSelectedRowKeys([]);
      publishRefresh('system:i18n:changed', 'system/i18n');
      await loadData(query, { silent: true });
      await loadOverview();
      await reloadMissingLocaleRows();
      if (auditVisible) {
        await loadAudit();
      }
    } catch {
      message.error(t('i18n.delete.error'));
    }
  };

  const handleRefreshCache = async () => {
    try {
      await refreshI18nCache();
      await reloadI18nResources();
      message.success(t('i18n.refreshSuccess'));
      publishRefresh('system:i18n:changed', 'system/i18n');
      await loadData(query, { silent: true });
      await loadOverview();
      await reloadMissingLocaleRows();
      if (auditVisible) {
        await loadAudit();
      }
    } catch {
      message.error(t('common.actionFailed'));
    }
  };

  const handleRefreshSelected = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const locales = Array.from(
      new Set(rows.filter((row) => selectedRowKeys.includes(row.id)).map((row) => row.locale)),
    );
    try {
      await refreshI18nLocales(locales);
      await reloadI18nResources();
      message.success(t('i18n.refreshSuccess'));
      publishRefresh('system:i18n:changed', 'system/i18n');
    } catch {
      message.error(t('i18n.refresh.error'));
    }
  };

  const handleExport = async () => {
    await exportI18n(query);
  };

  const handleExportModule = async (moduleName: string) => {
    await exportI18n({ module: moduleName });
  };

  const handleDownloadTemplate = async () => {
    try {
      await downloadI18nImportTemplate();
    } catch {
      message.error(t('i18n.import.template.error'));
    }
  };

  const handleImport = async (file: File) => {
    try {
      const result = await importI18n(file);
      showImportResult(result, t, {
        autoDownloadErrors: true,
        errorFileName: 'system-i18n-import-errors.csv',
      });
      if (result.applied) {
        await reloadI18nResources();
        publishRefresh('system:i18n:changed', 'system/i18n');
        await loadData(query, { silent: true });
        await loadOverview();
        await reloadMissingLocaleRows();
        if (auditVisible) {
          await loadAudit();
        }
      }
    } catch {
      message.error(t('i18n.import.error'));
    }
  };

  const handleSyncKeys = async () => {
    try {
      const resp = await syncI18nKeys();
      message.success(t('i18n.syncSuccess', { count: resp.count }));
      setSyncedKeys(resp.keys || []);
      setSyncVisible(true);
      publishRefresh('system:i18n:changed', 'system/i18n');
      await loadData(query, { silent: true });
      await loadOverview();
      await reloadMissingLocaleRows();
      if (auditVisible) {
        await loadAudit();
      }
    } catch {
      message.error(t('i18n.syncFailed'));
    }
  };

  const handleSave = async () => {
    if (!currentRow) {
      return;
    }
    let values;
    try {
      values = await form.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      throw error;
    }
    setSubmitting(true);
    try {
      await updateI18n(String(currentRow.id), values);
      await reloadI18nResources();
      message.success(t('common.updateSuccess'));
      setEditVisible(false);
      publishRefresh('system:i18n:changed', 'system/i18n');
      await loadData(query, { silent: true });
      await loadOverview();
      await reloadMissingLocaleRows();
      if (auditVisible) {
        await loadAudit();
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleFillMissingLocales = async () => {
    setFillingMissingLocales(true);
    try {
      const resp = await fillI18nMissingLocales(missingLocaleModuleFilter || undefined);
      message.success(t('i18n.fillMissingLocales.success', { count: resp.created }));
      await reloadI18nResources();
      publishRefresh('system:i18n:changed', 'system/i18n');
      await loadData(query, { silent: true });
      await loadOverview();
      await loadMissingLocales(missingLocaleModuleFilter || undefined);
      if (auditVisible) {
        await loadAudit();
      }
    } catch {
      message.error(t('i18n.fill_missing_locales.error'));
    } finally {
      setFillingMissingLocales(false);
    }
  };

  const handleHydrateBuiltinLocales = async (moduleName?: string) => {
    setHydratingBuiltinLocales(true);
    try {
      const resp = await hydrateBuiltinI18nLocales(moduleName);
      message.success(
        t('i18n.hydrateBuiltin.success', { created: resp.created, updated: resp.updated }),
      );
      await reloadI18nResources();
      publishRefresh('system:i18n:changed', 'system/i18n');
      await loadData(query, { silent: true });
      await loadOverview();
      await reloadMissingLocaleRows();
      if (auditVisible) {
        await loadAudit();
      }
    } catch {
      message.error(t('i18n.hydrate_builtin.error'));
    } finally {
      setHydratingBuiltinLocales(false);
    }
  };

  const openCreateModal = (initialValues?: Partial<I18nCreateReq>) => {
    setCreateDuplicateConflict(null);
    createForm.resetFields();
    createForm.setFieldsValue({
      group: 'messages',
      locale: 'zh-CN',
      ...initialValues,
    });
    setCreateVisible(true);
  };

  const openCreateFromMissingLocale = (item: I18nMissingLocaleItem, locale: string) => {
    openCreateModal({
      module: item.module,
      group: item.group || 'messages',
      key: item.key,
      locale,
      value: `[${item.key}]`,
      remark: t('i18n.missingLocales.prefillRemark'),
    });
  };

  const resolveCreateDuplicateConflict = async (key: string, locale: string) => {
    try {
      const resp = await getI18nList({
        key,
        locale,
        page: 1,
        pageSize: 20,
      });
      const exactMatch = (resp.items || []).find(
        (item) => item.key === key && item.locale === locale,
      );
      setCreateDuplicateConflict({
        key,
        locale,
        module: exactMatch?.module,
      });
    } catch {
      setCreateDuplicateConflict({
        key,
        locale,
      });
    }
  };

  const handleCreate = async () => {
    let values;
    try {
      values = await createForm.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      throw error;
    }
    setSubmitting(true);
    try {
      await createI18n(values);
      await reloadI18nResources();
      message.success(t('common.createSuccess'));
      setCreateVisible(false);
      createForm.resetFields();
      publishRefresh('system:i18n:changed', 'system/i18n');
      await loadData(query, { silent: true });
      await loadOverview();
      await reloadMissingLocaleRows();
      if (auditVisible) {
        await loadAudit();
      }
    } catch (error) {
      if (isRequestError(error) && error.messageKey === 'i18n.key.duplicate') {
        await resolveCreateDuplicateConflict(values.key, values.locale);
        return;
      }
      message.error(t('i18n.create.error'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleOpenAudit = async () => {
    setAuditVisible(true);
    await loadAudit();
  };

  const refreshAfterLifecycleChange = async () => {
    await reloadI18nResources();
    await loadData(query, { silent: true });
    await loadOverview();
    await reloadMissingLocaleRows();
    await loadAudit();
  };

  const handleStartUnusedObservation = async (moduleName?: string) => {
    setUnusedLifecycleLoading(true);
    try {
      const resp = await startUnusedObservation(moduleName);
      message.success(t('i18n.lifecycle.observe.success', { count: resp.affectedKeys.length }));
      await refreshAfterLifecycleChange();
    } catch {
      message.error(t('i18n.lifecycle.observe.error'));
    } finally {
      setUnusedLifecycleLoading(false);
    }
  };

  const handleArchiveObservedUnusedKeys = async (moduleName?: string) => {
    setUnusedLifecycleLoading(true);
    try {
      const resp = await archiveObservedUnusedKeys(moduleName);
      message.success(t('i18n.lifecycle.archive.success', { count: resp.affectedKeys.length }));
      await refreshAfterLifecycleChange();
    } catch {
      message.error(t('i18n.lifecycle.archive.error'));
    } finally {
      setUnusedLifecycleLoading(false);
    }
  };

  const handleDeleteArchivedUnusedKeys = async (moduleName?: string) => {
    setUnusedLifecycleLoading(true);
    try {
      const resp = await deleteArchivedUnusedKeys(moduleName, true);
      message.success(t('i18n.lifecycle.delete.success', { count: resp.affectedKeys.length }));
      await refreshAfterLifecycleChange();
    } catch {
      message.error(t('i18n.lifecycle.delete.error'));
    } finally {
      setUnusedLifecycleLoading(false);
    }
  };

  const handleLocateConflict = (moduleName: string, key: string) => {
    locateToList({ module: moduleName, key });
  };

  const handleLocateUnusedKey = (moduleName: string, key: string) => {
    locateToList({ module: moduleName, key });
  };

  const handleLocateStalePlaceholder = (item: I18nStalePlaceholderItem) => {
    locateToList({ module: item.module, locale: item.locale, key: item.key });
  };

  const loadRenamePreview = async (payload: I18nRenameFormValues) => {
    setRenamePreviewLoading(true);
    try {
      const resp = await previewI18nRename({
        module: payload.module,
        oldKey: payload.oldKey,
        newKey: payload.newKey,
      });
      setRenamePreview(resp);
    } catch {
      setRenamePreview(null);
      message.error(t('i18n.rename.preview.error'));
    } finally {
      setRenamePreviewLoading(false);
    }
  };

  const handleOpenRenameRepair = async (moduleName: string, oldKey: string, newKey: string) => {
    renameForm.setFieldsValue({
      module: moduleName,
      oldKey,
      newKey,
      confirmSourceUpdated: false,
    });
    setRenamePreview(null);
    setRenameVisible(true);
    await loadRenamePreview({
      module: moduleName,
      oldKey,
      newKey,
      confirmSourceUpdated: false,
    });
  };

  const handlePreviewRename = async () => {
    let values;
    try {
      values = await renameForm.validate(['module', 'oldKey', 'newKey']);
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      throw error;
    }
    await loadRenamePreview({
      module: values.module || '',
      oldKey: values.oldKey || '',
      newKey: values.newKey || '',
      confirmSourceUpdated: Boolean(values.confirmSourceUpdated),
    });
  };

  const handleExecuteRename = async () => {
    let values;
    try {
      values = await renameForm.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      throw error;
    }
    if (renamePreview?.requiresCodeMigration && !values.confirmSourceUpdated) {
      message.warning(t('i18n.rename.confirmSourceRequired'));
      return;
    }
    setRenameSubmitting(true);
    try {
      const resp = await renameI18nKey({
        module: values.module,
        oldKey: values.oldKey,
        newKey: values.newKey,
        confirmSourceUpdated: Boolean(values.confirmSourceUpdated),
      });
      message.success(t('i18n.rename.execute.success', { count: resp.renamedRows }));
      setRenameVisible(false);
      setRenamePreview(null);
      await reloadI18nResources();
      publishRefresh('system:i18n:changed', 'system/i18n');
      await loadData(query, { silent: true });
      await loadOverview();
      await reloadMissingLocaleRows();
      if (auditVisible) {
        await loadAudit();
      }
    } catch {
      message.error(t('i18n.rename.execute.error'));
    } finally {
      setRenameSubmitting(false);
    }
  };

  const handleDownloadRenameReport = () => {
    if (!renamePreview) {
      return;
    }
    const content = buildRenameMigrationReport(renamePreview, t);
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8;' });
    const url = globalThis.URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download =
      `${renamePreview.module}-${renamePreview.oldKey}-migration-report.txt`.replace(
        /[\\/:\s]+/g,
        '-',
      );
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    globalThis.URL.revokeObjectURL(url);
    message.success(t('i18n.rename.report.downloadSuccess'));
  };

  const columns: ColumnProps<SystemI18n>[] = [
    withTableColumnPriority(
      {
        title: t('i18n.module'),
        dataIndex: 'module',
        width: TABLE_COLUMN_WIDTH.code,
        render: (value: string) => <Tag color="arcoblue">{value}</Tag>,
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('i18n.group'),
        dataIndex: 'group',
        width: TABLE_COLUMN_WIDTH.status,
        render: (value: string) => <Tag>{value}</Tag>,
      },
      'low',
    ),
    {
      title: t('i18n.key'),
      dataIndex: 'key',
      width: TABLE_COLUMN_WIDTH.treeLabel,
      render: (value: string) => (
        <Text copyable style={{ display: 'block' }} ellipsis={{ showTooltip: true }}>
          {value}
        </Text>
      ),
    },
    withTableColumnPriority(
      {
        title: t('i18n.locale'),
        dataIndex: 'locale',
        width: TABLE_COLUMN_WIDTH.status,
        render: (value: string) => <Tag>{value}</Tag>,
      },
      'medium',
    ),
    {
      title: t('i18n.value'),
      dataIndex: 'value',
      width: TABLE_COLUMN_WIDTH.body,
      render: (value: string) => {
        const isMissing = !value || value.startsWith('[');
        return (
          <Text
            style={{
              display: 'block',
              color: isMissing ? '#F53F3F' : undefined,
              wordBreak: 'break-word',
            }}
            ellipsis={{ rows: 2, showTooltip: true }}
          >
            {value || '-'}
          </Text>
        );
      },
    },
    withTableColumnPriority(
      {
        title: t('i18n.createdAt'),
        dataIndex: 'createdAt',
        width: TABLE_COLUMN_WIDTH.datetime,
        sorter: true,
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('i18n.updatedAt'),
        dataIndex: 'updatedAt',
        width: TABLE_COLUMN_WIDTH.datetime,
        sorter: true,
      },
      'low',
    ),
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.medium,
      fixed: 'right',
      render: (_: unknown, row: SystemI18n) => (
        <Space className="system-list__actions">
          <Button
            type="text"
            icon={<IconEye />}
            loading={detailLoadingKey === `${row.id}:view`}
            disabled={detailLoading && detailLoadingKey !== `${row.id}:view`}
            onClick={() => void loadDetail(row, 'view')}
          >
            {t('common.detail')}
          </Button>
          {canEdit ? (
            <Button
              type="text"
              icon={<IconEdit />}
              loading={detailLoadingKey === `${row.id}:edit`}
              disabled={detailLoading && detailLoadingKey !== `${row.id}:edit`}
              onClick={() => void loadDetail(row, 'edit')}
            >
              {t('common.edit')}
            </Button>
          ) : null}
          {canDelete ? (
            <Popconfirm title={t('common.deleteConfirm')} onOk={() => handleDelete(row)}>
              <Button type="text" status="danger" icon={<IconDelete />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          ) : null}
        </Space>
      ),
    },
  ];

  const handleTableChange: TableProps<SystemI18n>['onChange'] = (_pagination, sorter) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : sorter;
    const direction = currentSorter?.direction;
    const field = typeof currentSorter?.field === 'string' ? currentSorter.field : '';
    setSelectedRowKeys([]);
    setQuery((prev) => ({
      ...prev,
      sortBy: direction ? field : '',
      sortOrder: direction === 'descend' ? 'desc' : 'asc',
      page: 1,
    }));
  };

  const visibleSelectedRowKeys = useMemo(
    () => getVisibleSelectedRowKeys(selectedRowKeys, rows.map((row) => row.id)),
    [rows, selectedRowKeys],
  );

  if (loading && rows.length === 0) {
    return <PageLoading />;
  }

  if (error) {
    if (isNetworkRequestError(error)) {
      return (
        <PageNetworkError timeout={isTimeoutRequestError(error)} onRetry={handleRetryLoadData} />
      );
    }
    if (isServerRequestError(error)) {
      return <PageServerError onRetry={handleRetryLoadData} />;
    }
    return <PageError onRetry={handleRetryLoadData} />;
  }

  return (
    <PageContainer>
      <Space direction="vertical" size={12} className="system-page-template i18n-list-page">
        <GovernanceSummaryBar
          eyebrow={t('i18n.hero.eyebrow')}
          title={t('i18n.hero.title')}
          description={t('i18n.hero.desc')}
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
              {t('i18n.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
          <FilterPanel>
            <Form form={queryForm} layout="vertical" onSubmit={() => handleSearch()}>
              <Row gutter={16}>
                <Col span={8}>
                  <FormItem label={t('i18n.key')} field="key">
                    <Input
                      allowClear
                      prefix={<IconSearch />}
                      onPressEnter={() => queryForm.submit()}
                    />
                  </FormItem>
                </Col>
                <Col span={4}>
                  <FormItem label={t('i18n.module')} field="module">
                    <Select allowClear placeholder={t('i18n.module')}>
                      {moduleOptions.map((moduleName) => (
                        <Select.Option key={moduleName} value={moduleName}>
                          {moduleName}
                        </Select.Option>
                      ))}
                    </Select>
                  </FormItem>
                </Col>
                <Col span={4}>
                  <FormItem label={t('i18n.group')} field="group">
                    <Select allowClear placeholder={t('i18n.group.placeholder')}>
                      {groupOptions.map((groupName) => (
                        <Select.Option key={groupName} value={groupName}>
                          {groupName}
                        </Select.Option>
                      ))}
                    </Select>
                  </FormItem>
                </Col>
                <Col span={4}>
                  <FormItem label={t('i18n.locale')} field="locale">
                    <Select allowClear placeholder={t('i18n.locale')}>
                      {(overview?.locales || [...SUPPORTED_LOCALES]).map((locale) => (
                        <Select.Option key={locale} value={locale}>
                          {locale}
                        </Select.Option>
                      ))}
                    </Select>
                  </FormItem>
                </Col>
                <Col span={4}>
                  <FormItem className="filter-panel__action-item">
                    <Space size={6}>
                      <Button size="small" type="primary" htmlType="submit" icon={<IconSearch />}>
                        {t('common.search')}
                      </Button>
                      <Button size="small" onClick={handleReset}>
                        {t('common.reset')}
                      </Button>
                    </Space>
                  </FormItem>
                </Col>
              </Row>
            </Form>
          </FilterPanel>

          <Card className="page-panel system-list__table-card i18n-list-page__table-card">
            <div className="system-list__table-head">
              <div className="system-list__table-head-copy">
                <Typography.Text className="system-list__table-head-title">
                  {t('i18n.viewTitle')}
                </Typography.Text>
                <Typography.Paragraph type="secondary" className="system-list__table-head-desc">
                  {t('common.total', { count: summary.total })}
                </Typography.Paragraph>
              </div>
            </div>
            {canDelete || canRefresh ? (
              <TableBatchActionBar
                selectedCount={selectedRowKeys.length}
                selectedText={t('common.selectedCount', { count: selectedRowKeys.length })}
                clearText={t('common.clearSelection')}
                clearSuccessText={t('common.clearSelectionSuccess')}
                onClear={() => setSelectedRowKeys([])}
                prefixActions={
                  <ListHeaderActions
                    className="i18n-list-page__header-actions"
                    utility={
                      <>
                        <Button
                          size="small"
                          icon={<IconRefresh />}
                          onClick={() => void handleSyncKeys()}
                          disabled={!canRefresh}
                        >
                          {t('common.refresh')}
                        </Button>
                        <Button size="small" icon={<IconEye />} onClick={() => void handleOpenAudit()}>
                          {t('i18n.audit.action')}
                        </Button>
                        {canHydrateBuiltin ? (
                          <Button
                            size="small"
                            status="warning"
                            loading={hydratingBuiltinLocales}
                            onClick={() => void handleHydrateBuiltinLocales(query.module || undefined)}
                          >
                            {t('i18n.hydrateBuiltin.action')}
                          </Button>
                        ) : null}
                        {canExport ? (
                          <Button
                            size="small"
                            icon={<IconDownload />}
                            onClick={() => void handleExport()}
                          >
                            {t('i18n.export')}
                          </Button>
                        ) : null}
                        {canImport ? (
                          <>
                            <Button size="small" onClick={() => void handleDownloadTemplate()}>
                              {t('common.downloadTemplate')}
                            </Button>
                            <ImportCsvButton
                              onSelect={(file) => {
                                void handleImport(file);
                              }}
                            >
                              {t('i18n.import')}
                            </ImportCsvButton>
                          </>
                        ) : null}
                        <Button
                          size="small"
                          status="warning"
                          icon={<IconRefresh />}
                          onClick={() => void handleRefreshCache()}
                          disabled={!canRefresh}
                        >
                          {t('i18n.refreshCache')}
                        </Button>
                      </>
                    }
                    primary={
                      canCreate ? (
                        <Button size="small" type="primary" onClick={() => openCreateModal()}>
                          {t('common.create')}
                        </Button>
                      ) : null
                    }
                  />
                }
                hint={!canDelete || !canRefresh ? t('common.batchActionPermissionHint') : undefined}
                actions={
                  <>
                    <PermissionAction allowed={canRefresh} tooltip={t('common.noPermissionAction')}>
                      <Button
                        size="small"
                        onClick={() => void handleRefreshSelected()}
                        disabled={selectedRowKeys.length === 0 || !canRefresh}
                      >
                        {t('i18n.refreshSelected')}
                      </Button>
                    </PermissionAction>
                    <PermissionAction allowed={canDelete} tooltip={t('common.noPermissionAction')}>
                      <Popconfirm
                        title={t('i18n.batchDeleteConfirm')}
                        onOk={() => void handleBatchDelete()}
                        disabled={selectedRowKeys.length === 0 || !canDelete}
                      >
                        <Button
                          size="small"
                          status="danger"
                          icon={<IconDelete />}
                          disabled={selectedRowKeys.length === 0 || !canDelete}
                        >
                          {t('i18n.batchDelete')}
                        </Button>
                      </Popconfirm>
                    </PermissionAction>
                  </>
                }
              />
            ) : null}

            {rows.length === 0 ? (
              <PageEmpty description={t('common.noData')} />
            ) : (
              <AppTable<SystemI18n>
                className="system-list__table"
                rowKey="id"
                loading={loading}
                columns={columns}
                data={rows}
                onChange={handleTableChange}
                rowSelection={
                  canDelete || canRefresh
                    ? {
                        selectedRowKeys: visibleSelectedRowKeys,
                        checkCrossPage: true,
                        preserveSelectedRowKeys: true,
                        onChange: (keys) =>
                          setSelectedRowKeys((currentKeys) =>
                            mergeCrossPageSelection(
                              currentKeys,
                              keys,
                              rows.map((row) => row.id),
                            ),
                          ),
                      }
                    : undefined
                }
                scroll={{ x: 'max-content' }}
                pagination={buildStandardPagination(t, {
                  total,
                  current: query.page,
                  pageSize: query.pageSize,
                  onChange: (page, pageSize) => setQuery({ ...query, page, pageSize }),
                })}
              />
            )}
          </Card>
        </>
      </Space>

      <GovernanceInsightDrawer
        title={t('i18n.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('i18n.audit.action')}
        noteDescription={t('i18n.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>

      <AppModal
        title={t('i18n.audit.title')}
        visible={auditVisible}
        size="detail"
        footer={
          <Space>
            <Button onClick={() => setAuditVisible(false)}>{t('common.close')}</Button>
            <Button loading={auditLoading} onClick={() => void loadAudit()}>
              {t('common.refresh')}
            </Button>
            <Button
              type="outline"
              loading={unusedLifecycleLoading}
              disabled={!audit || audit.unusedKeys.length === 0 || !canEdit}
              onClick={() => void handleStartUnusedObservation()}
            >
              {t('i18n.lifecycle.observe.all')}
            </Button>
            <Button
              type="outline"
              status="warning"
              loading={unusedLifecycleLoading}
              disabled={
                !audit || audit.unusedKeys.every((item) => !item.eligibleForArchive) || !canEdit
              }
              onClick={() => void handleArchiveObservedUnusedKeys()}
            >
              {t('i18n.lifecycle.archive.all')}
            </Button>
            <Popconfirm
              title={t('i18n.lifecycle.delete.confirm')}
              onOk={() => void handleDeleteArchivedUnusedKeys()}
            >
              <Button
                type="primary"
                status="danger"
                loading={unusedLifecycleLoading}
                disabled={
                  !audit || audit.unusedKeys.every((item) => !item.eligibleForDelete) || !canDelete
                }
              >
                {t('i18n.lifecycle.delete.all')}
              </Button>
            </Popconfirm>
          </Space>
        }
        onCancel={() => setAuditVisible(false)}
      >
        {auditLoading && !audit ? (
          <PageLoading />
        ) : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Card className="page-panel" title={t('i18n.audit.modules')}>
              {audit?.modules?.length ? (
                <List
                  bordered={false}
                  dataSource={audit.modules}
                  render={(item) => (
                    <List.Item key={item.module}>
                      <Space direction="vertical" size={4} style={{ width: '100%' }}>
                        <Space wrap style={{ justifyContent: 'space-between', width: '100%' }}>
                          <Text className="font-semibold">{item.module}</Text>
                          <Space>
                            <Button
                              size="mini"
                              onClick={() => void handleExportModule(item.module)}
                            >
                              {t('i18n.audit.exportModule')}
                            </Button>
                            <Button
                              size="mini"
                              type="outline"
                              disabled={item.unusedKeyCount === 0 || !canEdit}
                              loading={unusedLifecycleLoading}
                              onClick={() => void handleStartUnusedObservation(item.module)}
                            >
                              {t('i18n.lifecycle.observe.module')}
                            </Button>
                            <Button
                              size="mini"
                              type="outline"
                              status="warning"
                              disabled={item.observingKeyCount === 0 || !canEdit}
                              loading={unusedLifecycleLoading}
                              onClick={() => void handleArchiveObservedUnusedKeys(item.module)}
                            >
                              {t('i18n.lifecycle.archive.module')}
                            </Button>
                            <Popconfirm
                              title={t('i18n.lifecycle.delete.confirmModule')}
                              onOk={() => void handleDeleteArchivedUnusedKeys(item.module)}
                            >
                              <Button
                                size="mini"
                                status="danger"
                                disabled={item.archivedKeyCount === 0 || !canDelete}
                                loading={unusedLifecycleLoading}
                              >
                                {t('i18n.lifecycle.delete.module')}
                              </Button>
                            </Popconfirm>
                          </Space>
                        </Space>
                        <Space wrap>
                          <Tag>{t('i18n.audit.entryCount', { count: item.entryCount })}</Tag>
                          <Tag>{t('i18n.audit.keyCount', { count: item.keyCount })}</Tag>
                          <Tag color={item.unusedKeyCount > 0 ? 'orange' : 'green'}>
                            {t('i18n.audit.unusedCount', { count: item.unusedKeyCount })}
                          </Tag>
                          <Tag color={item.duplicateKeyCount > 0 ? 'red' : 'green'}>
                            {t('i18n.audit.duplicateCount', { count: item.duplicateKeyCount })}
                          </Tag>
                          <Tag color={item.missingLocaleCount > 0 ? 'orange' : 'green'}>
                            {t('i18n.audit.missingLocaleCount', { count: item.missingLocaleCount })}
                          </Tag>
                          <Tag color={item.placeholderCount > 0 ? 'orange' : 'green'}>
                            {t('i18n.audit.placeholderCount', { count: item.placeholderCount })}
                          </Tag>
                          <Tag color={item.stalePlaceholderCount > 0 ? 'red' : 'green'}>
                            {t('i18n.audit.stalePlaceholderCount', {
                              count: item.stalePlaceholderCount,
                            })}
                          </Tag>
                          <Tag color={item.observingKeyCount > 0 ? 'gold' : 'green'}>
                            {t('i18n.lifecycle.observingCount', { count: item.observingKeyCount })}
                          </Tag>
                          <Tag color={item.archivedKeyCount > 0 ? 'red' : 'green'}>
                            {t('i18n.lifecycle.archivedCount', { count: item.archivedKeyCount })}
                          </Tag>
                        </Space>
                      </Space>
                    </List.Item>
                  )}
                />
              ) : (
                <Text type="secondary">{t('i18n.audit.empty')}</Text>
              )}
            </Card>

            <Card className="page-panel" title={t('i18n.audit.duplicateKeys')}>
              {audit?.duplicateKeys?.length ? (
                <List
                  bordered={false}
                  dataSource={audit.duplicateKeys}
                  render={(item, index) => (
                    <List.Item key={`${item.key}-${item.modules.join('|')}-${index}`}>
                      <Space direction="vertical" size={4} style={{ width: '100%' }}>
                        <Text copyable>{item.key}</Text>
                        <Space wrap>
                          {item.modules.map((moduleName) => (
                            <Button
                              key={`${item.key}-${moduleName}`}
                              size="mini"
                              status="danger"
                              type="outline"
                              onClick={() => handleLocateConflict(moduleName, item.key)}
                            >
                              {moduleName}
                            </Button>
                          ))}
                        </Space>
                        <Text type="secondary">
                          {t('i18n.audit.groupsLabel')}: {item.groups.join(', ') || '-'} ·{' '}
                          {t('i18n.audit.localesLabel')}: {item.locales.join(', ') || '-'}
                        </Text>
                        {item.suggestions.length > 0 ? (
                          <Space direction="vertical" size={4}>
                            <Text type="secondary">{t('i18n.audit.renameSuggestions')}</Text>
                            {item.suggestions.map((suggestion: I18nRenameSuggestion) => (
                              <Space key={`${item.key}-${suggestion.module}`} wrap>
                                <Text
                                  copyable
                                >{`${suggestion.module} -> ${suggestion.suggestedKey}`}</Text>
                                <Button
                                  size="mini"
                                  type="outline"
                                  status="danger"
                                  disabled={!canEdit}
                                  onClick={() =>
                                    void handleOpenRenameRepair(
                                      suggestion.module,
                                      item.key,
                                      suggestion.suggestedKey,
                                    )
                                  }
                                >
                                  {t('i18n.rename.action')}
                                </Button>
                              </Space>
                            ))}
                          </Space>
                        ) : null}
                      </Space>
                    </List.Item>
                  )}
                />
              ) : (
                <Text type="secondary">{t('i18n.audit.duplicateEmpty')}</Text>
              )}
            </Card>

            <Card className="page-panel" title={t('i18n.audit.unusedKeys')}>
              {audit?.unusedKeys?.length ? (
                <List
                  bordered={false}
                  dataSource={audit.unusedKeys}
                  render={(item, index) => (
                    <List.Item key={`${item.key}-${item.module}-${index}`}>
                      <Space direction="vertical" size={4} style={{ width: '100%' }}>
                        <Text copyable>{item.key}</Text>
                        <Space wrap>
                          {item.modules.map((moduleName: string) => (
                            <Button
                              key={`${item.key}-${moduleName}`}
                              size="mini"
                              type="outline"
                              status="warning"
                              onClick={() => handleLocateUnusedKey(moduleName, item.key)}
                            >
                              {moduleName}
                            </Button>
                          ))}
                          {item.placeholder ? (
                            <Tag color="gold">{t('i18n.audit.placeholderTag')}</Tag>
                          ) : null}
                          <Tag
                            color={
                              item.lifecycleStatus === 'archived'
                                ? 'red'
                                : item.lifecycleStatus === 'observing'
                                  ? 'gold'
                                  : 'green'
                            }
                          >
                            {t(`i18n.lifecycle.status.${item.lifecycleStatus}`)}
                          </Tag>
                          {item.eligibleForArchive ? (
                            <Tag color="orange">{t('i18n.lifecycle.readyToArchive')}</Tag>
                          ) : null}
                          {item.eligibleForDelete ? (
                            <Tag color="red">{t('i18n.lifecycle.readyToDelete')}</Tag>
                          ) : null}
                        </Space>
                        <Text type="secondary">
                          {t('i18n.audit.localesLabel')}: {item.locales.join(', ') || '-'}
                        </Text>
                        {item.lifecycleMarkedAt ? (
                          <Text type="secondary">
                            {t('i18n.lifecycle.markedAt')}: {item.lifecycleMarkedAt} ·{' '}
                            {t('i18n.lifecycle.observingDays', { count: item.observingDays })}
                          </Text>
                        ) : null}
                      </Space>
                    </List.Item>
                  )}
                />
              ) : (
                <Text type="secondary">{t('i18n.audit.unusedEmpty')}</Text>
              )}
            </Card>

            <Card className="page-panel" title={t('i18n.audit.stalePlaceholders')}>
              {audit?.stalePlaceholders?.length ? (
                <List
                  bordered={false}
                  dataSource={audit.stalePlaceholders}
                  render={(item: I18nStalePlaceholderItem) => (
                    <List.Item key={`${item.id}-${item.locale}`}>
                      <Space direction="vertical" size={4} style={{ width: '100%' }}>
                        <Space wrap style={{ justifyContent: 'space-between', width: '100%' }}>
                          <Text copyable>{item.key}</Text>
                          <Button
                            size="mini"
                            status="warning"
                            onClick={() => handleLocateStalePlaceholder(item)}
                          >
                            {t('i18n.audit.locateAction')}
                          </Button>
                        </Space>
                        <Text type="secondary">
                          {item.module || '-'} / {item.group || 'messages'} / {item.locale}
                        </Text>
                        <Space wrap>
                          <Tag color="red">
                            {t('i18n.audit.staleDays', { count: item.staleDays })}
                          </Tag>
                          <Tag color="gold">{t('i18n.audit.placeholderTag')}</Tag>
                        </Space>
                      </Space>
                    </List.Item>
                  )}
                />
              ) : (
                <Text type="secondary">
                  {t('i18n.audit.stalePlaceholdersEmpty', {
                    days: audit?.stalePlaceholderThresholdDays || 30,
                  })}
                </Text>
              )}
            </Card>
          </Space>
        )}
      </AppModal>

      <AppModal
        title={t('i18n.rename.title')}
        visible={renameVisible}
        size="detail"
        confirmLoading={renameSubmitting}
        onOk={() => void handleExecuteRename()}
        onCancel={() => {
          setRenameVisible(false);
          setRenamePreview(null);
        }}
        footer={
          <Space>
            <Button
              onClick={() => {
                setRenameVisible(false);
                setRenamePreview(null);
              }}
            >
              {t('common.close')}
            </Button>
            <Button loading={renamePreviewLoading} onClick={() => void handlePreviewRename()}>
              {t('i18n.rename.preview.action')}
            </Button>
            <Button
              icon={<IconDownload />}
              disabled={!renamePreview}
              onClick={() => handleDownloadRenameReport()}
            >
              {t('i18n.rename.report.download')}
            </Button>
            <Button
              type="primary"
              status="warning"
              loading={renameSubmitting}
              disabled={!renamePreview?.canExecute}
              onClick={() => void handleExecuteRename()}
            >
              {t('i18n.rename.execute.action')}
            </Button>
          </Space>
        }
      >
        <Form
          form={renameForm}
          layout="vertical"
          onSubmit={() => {
            void handlePreviewRename();
          }}
        >
          <Row gutter={16}>
            <Col span={8}>
              <FormItem
                label={t('i18n.module')}
                field="module"
                rules={[requiredRule(t, 'i18n.module')]}
              >
                <Input onPressEnter={() => renameForm.submit()} />
              </FormItem>
            </Col>
            <Col span={8}>
              <FormItem
                label={t('i18n.rename.oldKey')}
                field="oldKey"
                rules={[requiredRule(t, 'i18n.rename.oldKey')]}
              >
                <Input onPressEnter={() => renameForm.submit()} />
              </FormItem>
            </Col>
            <Col span={8}>
              <FormItem
                label={t('i18n.rename.newKey')}
                field="newKey"
                rules={[requiredRule(t, 'i18n.rename.newKey')]}
              >
                <Input onPressEnter={() => renameForm.submit()} />
              </FormItem>
            </Col>
          </Row>
          <FormItem field="confirmSourceUpdated" triggerPropName="checked">
            <Checkbox>{t('i18n.rename.confirmSourceUpdated')}</Checkbox>
          </FormItem>
        </Form>
        {renamePreview ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Card className="page-panel" title={t('i18n.rename.preview.title')}>
              <Space wrap>
                <Tag color="arcoblue">
                  {t('i18n.rename.preview.affectedRows', { count: renamePreview.affectedRows })}
                </Tag>
                <Tag color={renamePreview.existingTargetRows > 0 ? 'red' : 'green'}>
                  {t('i18n.rename.preview.targetRows', { count: renamePreview.existingTargetRows })}
                </Tag>
                <Tag color={renamePreview.requiresCodeMigration ? 'orange' : 'green'}>
                  {t('i18n.rename.preview.referenceFiles', {
                    count: renamePreview.referenceFiles.length,
                  })}
                </Tag>
              </Space>
              <Text type="secondary" style={{ display: 'block', marginTop: 12 }}>
                {t('i18n.rename.preview.localeSummary')}:{' '}
                {renamePreview.affectedLocales.join(', ') || '-'}
              </Text>
              {renamePreview.existingTargetLocales.length > 0 ? (
                <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
                  {t('i18n.rename.preview.targetLocaleSummary')}:{' '}
                  {renamePreview.existingTargetLocales.join(', ')}
                </Text>
              ) : null}
            </Card>

            <Card className="page-panel" title={t('i18n.rename.preview.referenceTitle')}>
              {renamePreview.referenceFiles.length > 0 ? (
                <List
                  bordered={false}
                  dataSource={renamePreview.referenceFiles}
                  render={(item) => (
                    <List.Item key={item.path}>
                      <Space direction="vertical" size={6} style={{ width: '100%' }}>
                        <Text copyable>{item.path}</Text>
                        <Text type="secondary">
                          {t('i18n.rename.preview.matchCount', { count: item.matchCount })}
                        </Text>
                        {item.matches.map((match) => (
                          <Space
                            key={`${item.path}-${match.line}-${match.column}`}
                            direction="vertical"
                            size={2}
                            style={{ width: '100%' }}
                          >
                            <Text type="secondary">
                              {t('i18n.rename.preview.matchLocation', {
                                line: match.line,
                                column: match.column,
                              })}
                            </Text>
                            <Text code>{match.snippet || '-'}</Text>
                            <Text code>{match.replacementHint || '-'}</Text>
                          </Space>
                        ))}
                      </Space>
                    </List.Item>
                  )}
                />
              ) : (
                <Text type="secondary">{t('i18n.rename.preview.referenceEmpty')}</Text>
              )}
            </Card>

            <Card className="page-panel" title={t('i18n.rename.preview.warningTitle')}>
              <Space direction="vertical" size={6}>
                <Text type="secondary">{t('i18n.rename.preview.warning1')}</Text>
                <Text type="secondary">{t('i18n.rename.preview.warning2')}</Text>
                <Text type="secondary">{t('i18n.rename.preview.warning3')}</Text>
              </Space>
            </Card>
          </Space>
        ) : null}
      </AppModal>

      <AppModal
        title={t('i18n.missingLocales.title')}
        visible={missingLocaleVisible}
        size="detail"
        footer={
          <Space>
            <Button onClick={() => setMissingLocaleVisible(false)}>{t('common.close')}</Button>
            <Button
              type="primary"
              loading={fillingMissingLocales}
              disabled={missingLocaleRows.length === 0}
              onClick={() => void handleFillMissingLocales()}
            >
              {t('i18n.fillMissingLocales.action')}
            </Button>
            <Button
              status="warning"
              loading={hydratingBuiltinLocales}
              disabled={!canHydrateBuiltin}
              onClick={() =>
                void handleHydrateBuiltinLocales(missingLocaleModuleFilter || undefined)
              }
            >
              {t('i18n.hydrateBuiltin.action')}
            </Button>
          </Space>
        }
        onCancel={() => setMissingLocaleVisible(false)}
      >
        <>
          <Form layout="vertical" style={{ marginBottom: 16 }}>
            <FormItem label={t('i18n.missingLocales.moduleFilter')}>
              <Select
                allowClear
                placeholder={t('i18n.module')}
                value={missingLocaleModuleFilter || undefined}
                onChange={(value) => setMissingLocaleModuleFilter(value || '')}
              >
                {moduleOptions.map((moduleName) => (
                  <Select.Option key={moduleName} value={moduleName}>
                    {moduleName}
                  </Select.Option>
                ))}
              </Select>
            </FormItem>
          </Form>
          {missingLocaleRows.length === 0 ? (
            <Text type="secondary">{t('i18n.missingLocales.empty')}</Text>
          ) : (
            <List
              bordered={false}
              dataSource={missingLocaleRows}
              render={(item, index) => (
                <List.Item key={`${item.key}-${index}`}>
                  <Space direction="vertical" size={4} style={{ width: '100%' }}>
                    <Text copyable>{item.key}</Text>
                    <Text type="secondary">
                      {item.module || '-'} / {item.group || 'messages'}
                    </Text>
                    <Space wrap>
                      {item.missingLocales.map((locale) => (
                        <Button
                          key={`${item.key}-${locale}`}
                          size="mini"
                          type="outline"
                          status="warning"
                          onClick={() => openCreateFromMissingLocale(item, locale)}
                        >
                          {locale}
                        </Button>
                      ))}
                    </Space>
                  </Space>
                </List.Item>
              )}
            />
          )}
        </>
      </AppModal>

      <AppModal
        title={t('i18n.createTitle')}
        visible={createVisible}
        size="lg"
        confirmLoading={submitting}
        onOk={() => void handleCreate()}
        onCancel={() => setCreateVisible(false)}
      >
        <Form
          form={createForm}
          layout="vertical"
          onSubmit={() => {
            void handleCreate();
          }}
          onValuesChange={(changedValues) => {
            if (
              createDuplicateConflict &&
              ('key' in changedValues || 'locale' in changedValues || 'module' in changedValues)
            ) {
              setCreateDuplicateConflict(null);
            }
          }}
        >
          {createDuplicateConflict ? (
            <Card size="small" className="page-panel" style={{ marginBottom: 16 }}>
              <Space direction="vertical" size={4} style={{ width: '100%' }}>
                <Text type="error">
                  {t('i18n.create.duplicateBlocked', {
                    key: createDuplicateConflict.key,
                    locale: createDuplicateConflict.locale,
                  })}
                </Text>
                <Text type="secondary">
                  {t('i18n.create.duplicateOwner', {
                    module: createDuplicateConflict.module || '-',
                  })}
                </Text>
              </Space>
            </Card>
          ) : null}
          <Row gutter={16}>
            <Col span={12}>
              <FormItem
                label={t('i18n.module')}
                field="module"
                rules={[requiredRule(t, 'i18n.module')]}
              >
                <Input onPressEnter={() => createForm.submit()} />
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem label={t('i18n.group')} field="group" initialValue="messages">
                <Input onPressEnter={() => createForm.submit()} />
              </FormItem>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <FormItem label={t('i18n.key')} field="key" rules={[requiredRule(t, 'i18n.key')]}>
                <Input onPressEnter={() => createForm.submit()} />
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem
                label={t('i18n.locale')}
                field="locale"
                rules={[requiredRule(t, 'i18n.locale')]}
                initialValue="zh-CN"
              >
                <Select>
                  {SUPPORTED_LOCALES.map((locale) => (
                    <Select.Option key={locale} value={locale}>
                      {locale}
                    </Select.Option>
                  ))}
                </Select>
              </FormItem>
            </Col>
          </Row>
          <FormItem label={t('i18n.value')} field="value" rules={[requiredRule(t, 'i18n.value')]}>
            <Input.TextArea autoSize={{ minRows: 4, maxRows: 8 }} />
          </FormItem>
          <FormItem label={t('i18n.remark')} field="remark">
            <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
          </FormItem>
        </Form>
      </AppModal>

      <AppModal
        title={t('i18n.viewTitle')}
        visible={detailVisible}
        size="detail"
        footer={<Button onClick={() => setDetailVisible(false)}>{t('common.close')}</Button>}
        onCancel={() => setDetailVisible(false)}
      >
        <Descriptions
          column={1}
          labelStyle={{ width: 140 }}
          data={[
            { label: t('i18n.module'), value: currentRow?.module || '-' },
            { label: t('i18n.group'), value: currentRow?.group || '-' },
            { label: t('i18n.key'), value: currentRow?.key || '-' },
            { label: t('i18n.locale'), value: currentRow?.locale || '-' },
            { label: t('i18n.value'), value: currentRow?.value || '-' },
            { label: t('i18n.remark'), value: currentRow?.remark || '-' },
            { label: t('i18n.createdAt'), value: currentRow?.createdAt || '-' },
            { label: t('i18n.updatedAt'), value: currentRow?.updatedAt || '-' },
          ]}
        />
      </AppModal>

      <AppModal
        title={t('i18n.editTitle')}
        visible={editVisible}
        size="lg"
        confirmLoading={submitting || detailLoading}
        onOk={() => void handleSave()}
        onCancel={() => setEditVisible(false)}
      >
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <FormItem label={t('i18n.module')}>
                <Text>{currentRow?.module || '-'}</Text>
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem label={t('i18n.group')}>
                <Text>{currentRow?.group || '-'}</Text>
              </FormItem>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <FormItem label={t('i18n.key')}>
                <Text copyable>{currentRow?.key || '-'}</Text>
              </FormItem>
            </Col>
            <Col span={12}>
              <FormItem label={t('i18n.locale')}>
                <Text>{currentRow?.locale || '-'}</Text>
              </FormItem>
            </Col>
          </Row>
          <FormItem label={t('i18n.value')} field="value" rules={[requiredRule(t, 'i18n.value')]}>
            <Input.TextArea autoSize={{ minRows: 4, maxRows: 8 }} />
          </FormItem>
          <FormItem label={t('i18n.remark')} field="remark">
            <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
          </FormItem>
        </Form>
      </AppModal>

      <AppModal
        title={t('i18n.syncSuccess', { count: syncedKeys.length })}
        visible={syncVisible}
        size="detail"
        footer={<Button onClick={() => setSyncVisible(false)}>{t('common.close')}</Button>}
        onCancel={() => setSyncVisible(false)}
      >
        {syncedKeys.length === 0 ? (
          <Text type="secondary">{t('app.command.empty')}</Text>
        ) : (
          <List
            bordered={false}
            dataSource={syncedKeys}
            render={(item, index) => (
              <List.Item key={`${item}-${index}`}>
                <Text copyable>{item}</Text>
              </List.Item>
            )}
          />
        )}
      </AppModal>
    </PageContainer>
  );
};

export default I18nList;
