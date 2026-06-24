import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Form, Message } from '@arco-design/web-react';
import type { TableProps } from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import { isArcoFormValidationError } from '../../../../core/arco/formValidation';
import {
  exportSettingAudit,
  getSettingAuditList,
  refreshSettingCache,
  updateSettingGroup,
  type SettingAuditRow,
  type SettingItem,
} from './api';
import SettingAuditCard from './SettingAuditCard';
import SettingGroupForm from './SettingGroupForm';
import {
  buildFormValues,
  defaultAuditPageSize,
  notifySettingChanged,
  readFormFieldValue,
  runSettingGroupPostSaveEffects,
  serializeSettingValue,
  type SettingFormValue,
} from './useSettingCatalog';

interface SettingGroupWorkspaceProps {
  groupKey: string;
  groupItems: SettingItem[];
  canUpdateSetting: boolean;
  canRefreshCache: boolean;
  canExportAudit: boolean;
  canViewOperationLog: boolean;
  showAuditCard?: boolean;
  sectionId?: string;
  labelledById?: string;
  className?: string;
  onReload: () => Promise<void>;
  onViewOperationLog?: (id: number) => void;
}

const SettingGroupWorkspace: React.FC<SettingGroupWorkspaceProps> = ({
  groupKey,
  groupItems,
  canUpdateSetting,
  canRefreshCache,
  canExportAudit,
  canViewOperationLog,
  showAuditCard = false,
  sectionId,
  labelledById,
  className,
  onReload,
  onViewOperationLog,
}) => {
  const { t } = useTranslation();
  const [form] = Form.useForm<Record<string, SettingFormValue>>();
  const [submittingGroup, setSubmittingGroup] = useState<string | null>(null);
  const [refreshingCache, setRefreshingCache] = useState(false);
  const [auditRows, setAuditRows] = useState<SettingAuditRow[]>([]);
  const [auditTotal, setAuditTotal] = useState(0);
  const [auditLoading, setAuditLoading] = useState(false);
  const [auditQuery, setAuditQuery] = useState({ page: 1, pageSize: defaultAuditPageSize });
  const shouldShowAuditCard = showAuditCard && groupKey === 'audit' && canViewOperationLog;

  useEffect(() => {
    form.setFieldsValue(buildFormValues(groupItems));
  }, [form, groupItems]);

  const fieldNames = useMemo(() => groupItems.map((item) => item.settingKey), [groupItems]);

  const loadAudit = useCallback(
    async (page = 1, pageSize = defaultAuditPageSize) => {
      if (!shouldShowAuditCard) {
        return null;
      }
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
    [groupKey, shouldShowAuditCard, t],
  );

  useEffect(() => {
    if (!shouldShowAuditCard) {
      return;
    }
    const timer = globalThis.setTimeout(() => {
      void loadAudit(1, defaultAuditPageSize);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [groupKey, loadAudit, shouldShowAuditCard]);

  const resetGroupValues = () => {
    form.setFieldsValue(buildFormValues(groupItems));
  };

  const handleSubmit = async () => {
    if (!canUpdateSetting) {
      return;
    }
    let values;
    try {
      values = await form.validate(fieldNames);
    } catch (submitError) {
      if (isArcoFormValidationError(submitError)) {
        return;
      }
      throw submitError;
    }
    try {
      const items = groupItems.map((item) => ({
        settingKey: item.settingKey,
        settingValue: serializeSettingValue(
          item,
          readFormFieldValue(values as Record<string, unknown>, item.settingKey, (name) =>
            form.getFieldValue(name),
          ),
        ),
      }));
      setSubmittingGroup(groupKey);
      await updateSettingGroup(groupKey, { items });
      await runSettingGroupPostSaveEffects(groupKey, values as Record<string, SettingFormValue>);
      Message.success(t('common.updateSuccess'));
      notifySettingChanged();
      await onReload();
      if (shouldShowAuditCard) {
        await loadAudit(1, auditQuery.pageSize);
      }
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
    setRefreshingCache(true);
    try {
      await refreshSettingCache({ groupKeys: [groupKey] });
      Message.success(t('system.setting.cache.refreshSuccess'));
      notifySettingChanged();
      await onReload();
      if (shouldShowAuditCard) {
        await loadAudit(1, auditQuery.pageSize);
      }
    } finally {
      setRefreshingCache(false);
    }
  };

  const handleAuditTableChange: TableProps<SettingAuditRow>['onChange'] = (pagination) => {
    void loadAudit(pagination.current || 1, pagination.pageSize || auditQuery.pageSize);
  };

  const handleExportAudit = async () => {
    await exportSettingAudit({ groupKey });
  };

  return (
    <section
      id={sectionId}
      role={sectionId ? 'tabpanel' : undefined}
      aria-labelledby={labelledById}
      className={className ? `setting-group-workspace ${className}` : 'setting-group-workspace'}
    >
      <SettingGroupForm
        form={form}
        activeGroupKey={groupKey}
        activeGroupItems={groupItems}
        canUpdateSetting={canUpdateSetting}
        canRefreshCache={canRefreshCache}
        refreshingCache={refreshingCache}
        submittingGroup={submittingGroup}
        onRefreshCache={() => {
          void handleRefreshCache();
        }}
        onSubmit={() => {
          void handleSubmit();
        }}
        onCancel={resetGroupValues}
      />
      {shouldShowAuditCard ? (
        <SettingAuditCard
          rows={auditRows}
          total={auditTotal}
          loading={auditLoading}
          page={auditQuery.page}
          pageSize={auditQuery.pageSize}
          canExportAudit={canExportAudit}
          canViewOperationLog={canViewOperationLog}
          onChange={handleAuditTableChange}
          onExport={() => {
            void handleExportAudit();
          }}
          onViewOperationLog={(id) => {
            onViewOperationLog?.(id);
          }}
        />
      ) : null}
    </section>
  );
};

export default SettingGroupWorkspace;
