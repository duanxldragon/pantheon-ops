/**
 * 动态模块管理 - 模块管理页面
 *
 * 显示已注册模块列表,支持注册/卸载操作
 */

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Form,
  Popconfirm,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { IconDelete, IconPlus, IconRefresh } from '@arco-design/web-react/icon';
import AppTable from '../../../components/data-display/AppTable';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { ensureOperationVerified, isRequestError } from '../../../api/request';
import { message } from '../../../components/feedback/message';
import PermissionAction from '../../../components/patterns/PermissionAction';
import { invalidateRouteWarmData, resolveRouteWarmData } from '../../../core/router/prefetch';
import { usePermission } from '../../../hooks/usePermission';

import {
  getRegisteredModules,
  deleteModuleRecord,
  purgeModule,
  repairRegistries,
  registerModule,
  unregisterModule,
  type ModuleRegistration,
} from './api';
import {
  AppModal,
  ListHeaderActions,
  PageContainer,
  PageError,
  PageHeader,
  PageLoading,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  withTableColumnPriority,
} from '../../../components';
import { SECONDARY_VERIFY_CANCELLED_ERROR } from '../../../components/feedback/secondaryVerifyController';
import '../../../core/styles/list-page.css';

const moduleManagerWarmDataKeys = ['modules:registered'];

function invalidateModuleManagerWarmData() {
  invalidateRouteWarmData('/system/modules', moduleManagerWarmDataKeys);
}

function isManagedRegistration(record: ModuleRegistration) {
  return !record.builtIn && Boolean(record.tableName);
}

function isBusinessStaticRegistration(record: ModuleRegistration) {
  return !record.builtIn && record.scope === 'business' && !record.tableName;
}

function statusColor(status: number) {
  if (status === 1) return 'green';
  if (status === 3) return 'orange';
  if (status === 4) return 'red';
  return 'gray';
}

const ModuleManager: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { isAdmin, hasPerm } = usePermission();
  const [purgeForm] = Form.useForm<{ dropTable: boolean; confirmed: boolean }>();
  const canOpenGenerator = isAdmin || hasPerm('system:generator:use');
  const canRegister = isAdmin || hasPerm('system:module:register');
  const canUnregister = isAdmin || hasPerm('system:module:unregister');
  const canDeleteRecord = isAdmin || hasPerm('system:module:delete_record');
  const canPurge = isAdmin || hasPerm('system:module:purge');
  const canRepair = isAdmin || hasPerm('system:module:repair');
  const [modules, setModules] = useState<ModuleRegistration[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [featureDisabled, setFeatureDisabled] = useState(false);
  const [purgeTarget, setPurgeTarget] = useState<ModuleRegistration | null>(null);
  const [purging, setPurging] = useState(false);
  const [purgeConfirmed, setPurgeConfirmed] = useState(false);
  const [repairing, setRepairing] = useState(false);

  const loadData = useCallback(async (options?: { force?: boolean }) => {
    if (options?.force) {
      invalidateModuleManagerWarmData();
    }
    setLoading(true);
    setError(null);
    setFeatureDisabled(false);
    try {
      const result = options?.force
        ? await getRegisteredModules()
        : await resolveRouteWarmData('/system/modules', 'modules:registered', () =>
            getRegisteredModules(),
          );
      setModules(result);
    } catch (requestError) {
      if (isRequestError(requestError) && requestError.messageKey === 'module.dynamic.disabled') {
        setFeatureDisabled(true);
        setModules([]);
        return;
      }
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

  const handleUnregister = async (name: string) => {
    try {
      await ensureOperationVerified();
      await unregisterModule(name, false);
      message.success(t('generator.moduleManager.unregisterSuccess'));
      invalidateModuleManagerWarmData();
      await loadData();
    } catch (error) {
      if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
        return;
      }
      message.error(t('generator.moduleManager.unregisterError'));
    }
  };

  const handleRegister = async (name: string) => {
    try {
      await ensureOperationVerified();
      await registerModule({ name });
      message.success(t('generator.moduleManager.registerSuccess'));
      invalidateModuleManagerWarmData();
      await loadData();
    } catch (error) {
      if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
        return;
      }
      message.error(t('generator.moduleManager.registerError'));
    }
  };

  const handleDeleteRecord = async (name: string) => {
    try {
      await ensureOperationVerified();
      await deleteModuleRecord(name);
      message.success(t('generator.moduleManager.deleteRecordSuccess'));
      invalidateModuleManagerWarmData();
      await loadData();
    } catch (error) {
      if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
        return;
      }
      message.error(t('generator.moduleManager.deleteRecordError'));
    }
  };

  const openPurgeModal = (record: ModuleRegistration) => {
    purgeForm.setFieldsValue({ dropTable: false, confirmed: false });
    setPurgeConfirmed(false);
    setPurgeTarget(record);
  };

  const closePurgeModal = () => {
    if (purging) {
      return;
    }
    setPurgeTarget(null);
    setPurgeConfirmed(false);
    purgeForm.resetFields();
  };

  const handlePurge = async () => {
    if (!purgeTarget) {
      return;
    }
    try {
      const values = await purgeForm.validate();
      await ensureOperationVerified();
      setPurging(true);
      await purgeModule(purgeTarget.name, {
        purgeSource: true,
        dropTable: Boolean(values.dropTable),
      });
      message.success(t('generator.moduleManager.purgeSuccess'));
      closePurgeModal();
      invalidateModuleManagerWarmData();
      await loadData();
    } catch (error) {
      if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
        return;
      }
      message.error(t('generator.moduleManager.purgeError'));
    } finally {
      setPurging(false);
    }
  };

  const handleRepair = async () => {
    try {
      await ensureOperationVerified();
      setRepairing(true);
      const result = await repairRegistries();
      message.success(
        t('generator.moduleManager.repairSuccess', {
          refs: result.summary.generatedRegistryRefs,
          marked: result.summary.markedUninstalledModules,
        }),
      );
      invalidateModuleManagerWarmData();
      await loadData();
    } catch (error) {
      if (error instanceof Error && error.message === SECONDARY_VERIFY_CANCELLED_ERROR) {
        return;
      }
      message.error(t('generator.moduleManager.repairError'));
    } finally {
      setRepairing(false);
    }
  };

  const stats = useMemo(
    () => ({
      total: modules.length,
      active: modules.filter((item) => item.status === 1).length,
      pending: modules.filter((item) => item.status === 3).length,
      uninstalled: modules.filter((item) => item.status === 2).length,
      failed: modules.filter((item) => item.status === 4).length,
    }),
    [modules],
  );

  const columns = [
    {
      title: t('generator.moduleManager.name'),
      dataIndex: 'name',
      width: TABLE_COLUMN_WIDTH.identity,
      render: (name: string) => <span>{name}</span>,
    },
    {
      title: t('generator.moduleManager.displayName'),
      dataIndex: 'displayName',
      width: TABLE_COLUMN_WIDTH.name,
    },
    {
      title: t('generator.moduleManager.scope'),
      dataIndex: 'scope',
      width: TABLE_COLUMN_WIDTH.scope,
      render: (scope: string) => (
        <Tag color={scope === 'system' ? 'blue' : scope === 'platform' ? 'purple' : 'green'}>
          {scope}
        </Tag>
      ),
    },
    {
      title: t('generator.moduleManager.source'),
      dataIndex: 'source',
      width: TABLE_COLUMN_WIDTH.code,
      render: (source: string) => (
        <Tag
          color={
            source === 'generated' || source === 'database' || source === 'manual'
              ? 'green'
              : 'arcoblue'
          }
        >
          {t(`generator.moduleManager.source.${source || 'core'}`)}
        </Tag>
      ),
    },
    withTableColumnPriority(
      {
        title: t('generator.moduleManager.owner'),
        dataIndex: 'owner',
        width: TABLE_COLUMN_WIDTH.owner,
        render: (value?: string) => value || '-',
      },
      'medium',
    ),
    withTableColumnPriority(
      {
        title: t('generator.moduleManager.boundedContext'),
        dataIndex: 'boundedContext',
        width: TABLE_COLUMN_WIDTH.owner,
        render: (value?: string) => value || '-',
      },
      'medium',
    ),
    withTableColumnPriority(
      {
        title: t('generator.moduleManager.tableName'),
        dataIndex: 'tableName',
        width: TABLE_COLUMN_WIDTH.name,
        render: (tableName: string) => (tableName ? <span>{tableName}</span> : <span>-</span>),
      },
      'low',
    ),
    {
      title: t('generator.moduleManager.status'),
      dataIndex: 'status',
      width: TABLE_COLUMN_WIDTH.status,
      render: (status: number) => (
        <Tag color={statusColor(status)}>
          {status === 1
            ? t('generator.moduleManager.status.active')
            : status === 3
              ? t('generator.moduleManager.status.pending')
              : status === 4
                ? t('generator.moduleManager.status.failed')
                : t('generator.moduleManager.status.uninstalled')}
        </Tag>
      ),
    },
    withTableColumnPriority(
      {
        title: t('generator.moduleManager.installedAt'),
        dataIndex: 'installedAt',
        width: TABLE_COLUMN_WIDTH.datetime,
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('generator.moduleManager.diagnostics'),
        width: TABLE_COLUMN_WIDTH.diagnostics,
        render: (_value: unknown, record: ModuleRegistration) => {
          if (record.lastError) {
            return (
              <Space direction="vertical" size={2}>
                <Tag color="red">{t('generator.moduleManager.diagnostics.failed')}</Tag>
                <Typography.Text type="secondary">{t(record.lastError)}</Typography.Text>
              </Space>
            );
          }
          if (record.lastVerifiedAt) {
            return (
              <Space direction="vertical" size={2}>
                <Tag color="green">{t('generator.moduleManager.diagnostics.verified')}</Tag>
                <Typography.Text type="secondary">{record.lastVerifiedAt}</Typography.Text>
              </Space>
            );
          }
          return <span>-</span>;
        },
      },
      'low',
    ),
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.wide,
      render: (_value: unknown, record: ModuleRegistration) => {
        const managedRegistration = isManagedRegistration(record);
        const businessStaticRegistration = isBusinessStaticRegistration(record);
        const canPurgeRecord = managedRegistration || businessStaticRegistration;
        return (
          <Space size={4} className="system-list__actions">
            {record.builtIn ? (
              <Tag color="arcoblue">{t('generator.moduleManager.builtIn')}</Tag>
            ) : null}
            {record.status === 2 && managedRegistration ? (
              <>
                <PermissionAction allowed={canRegister} tooltip={t('common.noPermissionAction')}>
                  <Button
                    type="text"
                    disabled={featureDisabled}
                    onClick={() => {
                      void handleRegister(record.name);
                    }}
                  >
                    <IconPlus /> {t('generator.moduleManager.register')}
                  </Button>
                </PermissionAction>
                <PermissionAction
                  allowed={canDeleteRecord}
                  tooltip={t('common.noPermissionAction')}
                >
                  <Popconfirm
                    title={t('generator.moduleManager.confirmDeleteRecord')}
                    disabled={featureDisabled || !canDeleteRecord}
                    onOk={() => handleDeleteRecord(record.name)}
                  >
                    <Button
                      type="text"
                      status="danger"
                      disabled={featureDisabled || !canDeleteRecord}
                    >
                      <IconDelete /> {t('generator.moduleManager.deleteRecord')}
                    </Button>
                  </Popconfirm>
                </PermissionAction>
              </>
            ) : null}
            {record.status !== 2 && managedRegistration && canUnregister ? (
              <PermissionAction allowed={canUnregister} tooltip={t('common.noPermissionAction')}>
                <Popconfirm
                  title={t('generator.moduleManager.confirmUninstall')}
                  disabled={featureDisabled || !canUnregister}
                  onOk={() => handleUnregister(record.name)}
                >
                  <Button type="text" status="danger" disabled={featureDisabled || !canUnregister}>
                    <IconDelete /> {t('generator.moduleManager.unregister')}
                  </Button>
                </Popconfirm>
              </PermissionAction>
            ) : null}
            {canPurgeRecord ? (
              <PermissionAction allowed={canPurge} tooltip={t('common.noPermissionAction')}>
                <Button
                  type="text"
                  status="danger"
                  disabled={featureDisabled || !canPurge}
                  onClick={() => openPurgeModal(record)}
                >
                  <IconDelete /> {t('generator.moduleManager.purge')}
                </Button>
              </PermissionAction>
            ) : null}
          </Space>
        );
      },
    },
  ];

  if (loading && modules.length === 0) {
    return <PageLoading />;
  }

  if (error) {
    return (
      <PageError
        onRetry={() => {
          void loadData({ force: true });
        }}
      />
    );
  }

  return (
    <PageContainer>
      <PageHeader
        title={t('generator.moduleManager.title')}
        extra={
          <ListHeaderActions
            className="module-manager-page__header-actions"
            utility={
              <>
                <Button size="small" onClick={() => void loadData({ force: true })}>
                  <IconRefresh /> {t('common.refresh')}
                </Button>
                <PermissionAction allowed={canRepair} tooltip={t('common.noPermissionAction')}>
                  <Button
                    size="small"
                    disabled={featureDisabled || repairing}
                    loading={repairing}
                    onClick={() => void handleRepair()}
                  >
                    <IconRefresh /> {t('generator.moduleManager.repair')}
                  </Button>
                </PermissionAction>
              </>
            }
            primary={
              <PermissionAction allowed={canOpenGenerator} tooltip={t('common.noPermissionAction')}>
                <Button
                  size="small"
                  type="primary"
                  disabled={featureDisabled}
                  onClick={() => navigate('/system/generator')}
                >
                  <IconPlus /> {t('generator.moduleManager.registerNew')}
                </Button>
              </PermissionAction>
            }
          />
        }
      />

      <Space direction="vertical" size={12} className="system-page-template module-manager-page">
        <Card className="page-panel system-list__table-card module-manager-page__card">
          <div className="module-manager-page__intro">
            <div className="module-manager-page__copy">
              <span className="system-page-hero__eyebrow">
                {t('generator.moduleManager.title')}
              </span>
              <Typography.Paragraph className="module-manager-page__desc">
                {t('generator.moduleManager.description')}
              </Typography.Paragraph>
            </div>
            <div className="module-manager-page__notice-stack">
              {featureDisabled ? (
                <Alert type="warning" content={t('generator.moduleManager.disabledHint')} />
              ) : null}
              {modules.some((item) => item.status === 3) ? (
                <Alert type="warning" content={t('generator.moduleManager.pendingHint')} />
              ) : null}
              <Alert type="info" content={t('generator.moduleManager.repairHint')} />
            </div>
          </div>
          <div className="module-manager-page__stats">
            <Card size="small" className="module-manager-page__stat-card">
              <Typography.Text type="secondary">
                {t('generator.moduleManager.stats.total')}
              </Typography.Text>
              <Typography.Title heading={6} style={{ margin: 0 }}>
                {stats.total}
              </Typography.Title>
            </Card>
            <Card size="small" className="module-manager-page__stat-card">
              <Typography.Text type="secondary">
                {t('generator.moduleManager.stats.active')}
              </Typography.Text>
              <Typography.Title heading={6} style={{ margin: 0 }}>
                {stats.active}
              </Typography.Title>
            </Card>
            <Card size="small" className="module-manager-page__stat-card">
              <Typography.Text type="secondary">
                {t('generator.moduleManager.stats.pending')}
              </Typography.Text>
              <Typography.Title heading={6} style={{ margin: 0 }}>
                {stats.pending}
              </Typography.Title>
            </Card>
            <Card size="small" className="module-manager-page__stat-card">
              <Typography.Text type="secondary">
                {t('generator.moduleManager.stats.uninstalled')}
              </Typography.Text>
              <Typography.Title heading={6} style={{ margin: 0 }}>
                {stats.uninstalled}
              </Typography.Title>
            </Card>
            <Card size="small" className="module-manager-page__stat-card">
              <Typography.Text type="secondary">
                {t('generator.moduleManager.stats.failed')}
              </Typography.Text>
              <Typography.Title heading={6} style={{ margin: 0 }}>
                {stats.failed}
              </Typography.Title>
            </Card>
          </div>
          <AppTable
            columns={columns}
            data={modules}
            rowKey="name"
            pagination={false}
            emptyText={
              featureDisabled
                ? t('generator.moduleManager.readOnlyEmpty')
                : t('generator.moduleManager.empty')
            }
          />
        </Card>
      </Space>
      <AppModal
        title={t('generator.moduleManager.purgeModal.title')}
        visible={Boolean(purgeTarget)}
        onCancel={closePurgeModal}
        onOk={() => void handlePurge()}
        okButtonProps={{ status: 'danger', disabled: !purgeConfirmed, loading: purging }}
        okText={t('generator.moduleManager.purge')}
        cancelText={t('common.cancel')}
        size="md"
      >
        {purgeTarget ? (
          <Form form={purgeForm} layout="vertical">
            <Alert
              type="error"
              style={{ marginBottom: 16 }}
              content={t('generator.moduleManager.purgeModal.warning')}
            />
            <Typography.Paragraph style={{ marginBottom: 12 }}>
              {t('generator.moduleManager.purgeModal.summary', {
                module: purgeTarget.displayName || purgeTarget.name,
                name: purgeTarget.name,
              })}
            </Typography.Paragraph>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
              {t('generator.moduleManager.purgeModal.impact')}
            </Typography.Paragraph>
            <Space direction="vertical" size={8} style={{ width: '100%', marginBottom: 16 }}>
              <Typography.Text>
                {t('generator.moduleManager.purgeModal.removeRecord')}
              </Typography.Text>
              {purgeTarget.tableName ? (
                <Typography.Text>
                  {t('generator.moduleManager.purgeModal.removeSource')}
                </Typography.Text>
              ) : null}
              <Typography.Text type="secondary">
                {purgeTarget.tableName
                  ? t('generator.moduleManager.purgeModal.keepTable', {
                      table: purgeTarget.tableName,
                    })
                  : t('generator.moduleManager.purgeModal.noTable')}
              </Typography.Text>
            </Space>
            {purgeTarget.tableName ? (
              <Form.Item field="dropTable" triggerPropName="checked">
                <Checkbox>
                  {t('generator.moduleManager.purgeModal.dropTable', {
                    table: purgeTarget.tableName,
                  })}
                </Checkbox>
              </Form.Item>
            ) : null}
            <Form.Item field="confirmed" triggerPropName="checked">
              <Checkbox onChange={(checked) => setPurgeConfirmed(Boolean(checked))}>
                {t('generator.moduleManager.purgeModal.confirmLabel')}
              </Checkbox>
            </Form.Item>
          </Form>
        ) : null}
      </AppModal>
    </PageContainer>
  );
};

export default ModuleManager;
