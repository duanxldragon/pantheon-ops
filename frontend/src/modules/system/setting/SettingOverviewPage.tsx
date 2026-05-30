import React, { useMemo } from 'react';
import { Button, Space, Tag, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import {
  GovernanceSummaryBar,
  PageSplitLayout,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
  SideRailItem,
  SideRailPanel,
  SideRailStack,
} from '../../../components';
import { usePermission } from '../../../hooks/usePermission';
import {
  resolveSettingGroupMeta,
  settingGroups,
  type SettingGroupKey,
} from './settingGroups';
import SettingGroupWorkspace from './SettingGroupWorkspace';
import { useSettingCatalog } from './useSettingCatalog';
import '../list-page.css';

const SettingOverviewPage: React.FC = () => {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const { isAdmin, hasPerm } = usePermission();
  const canViewSettings = isAdmin || hasPerm('system:setting:list');
  const canUpdateSetting = isAdmin || hasPerm('system:setting:update');
  const canRefreshCache = isAdmin || hasPerm('system:setting:refresh');
  const canExportAudit = isAdmin || hasPerm('system:setting:export');
  const canViewOperationLog = isAdmin || hasPerm('system:operation-log:list');
  const { loading, error, settings, overview, groupedSettings, reload } = useSettingCatalog();
  const visibleGroups = useMemo(
    () => settingGroups.filter((meta) => groupedSettings.some((group) => group.groupKey === meta.key)),
    [groupedSettings],
  );

  const heroStats = useMemo(() => {
    if (!overview) {
      return [];
    }
    return [
      {
        key: 'total',
        label: t('system.setting.overview.totalSettings'),
        value: overview.totalSettingCount,
        description: t('system.setting.hero.totalHint'),
      },
      {
        key: 'public',
        label: t('system.setting.overview.publicSettings'),
        value: overview.publicSettingCount,
        description: t('system.setting.hero.publicHint'),
      },
      {
        key: 'encrypted',
        label: t('system.setting.overview.encryptedSettings'),
        value: overview.encryptedSettingCount,
        description: t('system.setting.hero.encryptedHint'),
      },
      {
        key: 'risk',
        label: t('system.setting.overview.risks'),
        value: overview.riskCount,
        description: t('system.setting.hero.riskHint'),
      },
    ];
  }, [overview, t]);

  const groupIssueCounts = useMemo(() => {
    return (overview?.issues || []).reduce<Record<string, number>>((acc, issue) => {
      acc[issue.groupKey] = (acc[issue.groupKey] || 0) + 1;
      return acc;
    }, {});
  }, [overview]);

  const activeGroupKey = useMemo(() => {
    const requested = searchParams.get('group');
    if (requested && visibleGroups.some((group) => group.key === requested)) {
      return requested as SettingGroupKey;
    }
    return visibleGroups[0]?.key;
  }, [searchParams, visibleGroups]);

  const activeGroup = useMemo(
    () => groupedSettings.find((group) => group.groupKey === activeGroupKey),
    [activeGroupKey, groupedSettings],
  );

  const runtimeSummaryItems = useMemo(
    () => [
      {
        key: 'storage',
        label: t('system.setting.hero.storageHint'),
        value: overview?.storageDriver || '-',
      },
      {
        key: 'language',
        label: t('system.setting.hero.languageHint'),
        value: overview?.defaultLanguage || '-',
      },
      {
        key: 'theme',
        label: t('system.setting.hero.themeHint'),
        value: overview?.defaultTheme || '-',
      },
      {
        key: 'risk',
        label: t('system.setting.overview.risks'),
        value: overview?.riskCount ?? 0,
        tone: (overview?.riskCount || 0) > 0 ? ('warning' as const) : ('neutral' as const),
        description:
          (overview?.riskCount || 0) > 0
            ? t('system.setting.hero.riskHint')
            : t('system.setting.overview.noRisks'),
      },
    ],
    [overview?.defaultLanguage, overview?.defaultTheme, overview?.riskCount, overview?.storageDriver, t],
  );

  const switchGroup = (groupKey: SettingGroupKey) => {
    const nextParams = new URLSearchParams(searchParams);
    nextParams.set('group', groupKey);
    setSearchParams(nextParams, { replace: true });
  };

  const renderErrorState = () => {
    if (isNetworkRequestError(error)) {
      return <PageNetworkError timeout={isTimeoutRequestError(error)} onRetry={() => globalThis.location.reload()} />;
    }
    if (isServerRequestError(error)) {
      return <PageServerError onRetry={() => globalThis.location.reload()} />;
    }
    return <PageError onRetry={() => globalThis.location.reload()} />;
  };

  return (
    <PageContainer>
      <Space
        direction="vertical"
        size={12}
        className="system-page-template setting-page setting-overview-page"
      >
        {loading && settings.length === 0 ? <PageLoading /> : null}
        {error && settings.length === 0 ? renderErrorState() : null}
        {!loading && !error && settings.length === 0 ? (
          <PageEmpty description={t('system.setting.empty')} />
        ) : null}
        {settings.length > 0 ? (
          <>
            <PageSplitLayout
              className="setting-overview-page__layout"
              railClassName="setting-overview-page__rail"
              rail={
                <>
                  {overview ? (
                    <SideRailPanel title={t('system.setting.overview.runtime')}>
                      <SideRailStack>
                        {runtimeSummaryItems.map((item) => (
                          <SideRailItem
                            key={item.key}
                            label={item.label}
                            value={item.value}
                            tone={item.tone}
                            description={item.description}
                          />
                        ))}
                      </SideRailStack>
                    </SideRailPanel>
                  ) : null}
                  <SideRailPanel title={t('system.setting.overview.risks')}>
                    {overview?.issues && overview.issues.length > 0 ? (
                      <div className="setting-overview-page__risk-list">
                        {overview.issues.map((issue) => {
                          const meta = resolveSettingGroupMeta(issue.groupKey);
                          return (
                            <div
                              key={`${issue.groupKey}-${issue.settingKey}-${issue.reasonKey}`}
                              className={`setting-overview-page__risk-item setting-overview-page__risk-item--${issue.severity}`}
                            >
                              <div className="setting-overview-page__risk-header">
                                <Space size={6} wrap>
                                  <Tag color={issue.severity === 'critical' ? 'red' : 'orange'}>
                                    {t(`system.setting.overview.severity.${issue.severity}`)}
                                  </Tag>
                                  <Tag>{t(meta.titleKey)}</Tag>
                                </Space>
                                <Button
                                  type="text"
                                  size="small"
                                  onClick={() => switchGroup(issue.groupKey as SettingGroupKey)}
                                >
                                  {t('common.detail')}
                                </Button>
                              </div>
                              <Typography.Text className="setting-overview-page__risk-title">
                                {t(`system.setting.item.${issue.settingKey}`, issue.settingKey)}
                              </Typography.Text>
                              <Typography.Text
                                type="secondary"
                                className="setting-overview-page__risk-desc"
                              >
                                {t(issue.reasonKey)}
                              </Typography.Text>
                            </div>
                          );
                        })}
                      </div>
                    ) : (
                      <Typography.Text type="secondary">
                        {t('system.setting.overview.noRisks')}
                      </Typography.Text>
                    )}
                  </SideRailPanel>
                </>
              }
            >
              <div className="setting-page__governance-stack">
                <GovernanceSummaryBar
                  className="setting-page__governance-bar"
                  eyebrow={t('system.setting.hero.eyebrow')}
                  title={t('system.setting.hero.title')}
                  description={t('system.setting.hero.desc')}
                  metrics={heroStats.map((item) => ({
                    key: item.key,
                    label: item.label,
                    value: item.value,
                    description: item.description,
                  }))}
                />
                <div className="setting-overview-page__anchor-strip" role="tablist" aria-label={t('system.setting.hero.title')}>
                  {visibleGroups.map((meta) => {
                      const group = groupedSettings.find((item) => item.groupKey === meta.key);
                      const issueCount = groupIssueCounts[meta.key] || 0;
                      const active = activeGroupKey === meta.key;
                      const tabId = `setting-group-tab-${meta.key}`;
                      const panelId = `setting-group-section-${meta.key}`;
                      return (
                        <button
                          key={meta.key}
                          type="button"
                          id={tabId}
                          role="tab"
                          aria-selected={active}
                          aria-controls={panelId}
                          tabIndex={active ? 0 : -1}
                          className={`setting-overview-page__anchor-item${active ? ' setting-overview-page__anchor-item--active' : ''}`}
                          onClick={() => {
                            switchGroup(meta.key);
                          }}
                        >
                          <span className="setting-overview-page__anchor-title-row">
                            <span className="setting-overview-page__anchor-title">
                              {t(meta.titleKey)}
                            </span>
                            {issueCount > 0 ? (
                              <Tag color={meta.tone === 'danger' ? 'red' : 'orange'}>
                                {t('common.total', { count: issueCount })}
                              </Tag>
                            ) : null}
                          </span>
                          <span className="setting-overview-page__anchor-desc">
                            {t(meta.descriptionKey, '')}
                          </span>
                          <span className="setting-overview-page__anchor-meta">
                            {t('common.total', { count: group?.items.length ?? 0 })}
                          </span>
                        </button>
                      );
                    })}
                </div>
                {activeGroup ? (
                  <SettingGroupWorkspace
                    key={activeGroup.groupKey}
                    sectionId={`setting-group-section-${activeGroup.groupKey}`}
                    labelledById={`setting-group-tab-${activeGroup.groupKey}`}
                    className="setting-overview-page__workspace"
                    groupKey={activeGroup.groupKey}
                    groupItems={activeGroup.items}
                    canUpdateSetting={canViewSettings && canUpdateSetting}
                    canRefreshCache={canViewSettings && canRefreshCache}
                    canExportAudit={canViewSettings && canExportAudit}
                    canViewOperationLog={canViewSettings && canViewOperationLog}
                    showAuditCard
                    onReload={reload}
                  />
                ) : null}
              </div>
            </PageSplitLayout>
          </>
        ) : null}
      </Space>
    </PageContainer>
  );
};

export default SettingOverviewPage;
