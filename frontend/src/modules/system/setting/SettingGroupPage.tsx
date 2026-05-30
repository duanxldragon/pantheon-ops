import React, { useEffect, useMemo } from 'react';
import { Space, Tag, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import {
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
  GovernanceSummaryBar,
} from '../../../components';
import { usePermission } from '../../../hooks/usePermission';
import {
  getSettingGroupPath,
  isSettingGroupKey,
  resolveSettingGroupMeta,
  settingGroups,
  type SettingGroupKey,
} from './settingGroups';
import SettingGroupWorkspace from './SettingGroupWorkspace';
import { useSettingCatalog } from './useSettingCatalog';
import '../list-page.css';

const SettingGroupPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { groupKey } = useParams();
  const { isAdmin, hasPerm } = usePermission();
  const canUpdateSetting = isAdmin || hasPerm('system:setting:update');
  const canRefreshCache = isAdmin || hasPerm('system:setting:refresh');
  const canExportAudit = isAdmin || hasPerm('system:setting:export');
  const canViewOperationLog = isAdmin || hasPerm('system:operation-log:list');
  const { loading, error, settings, overview, groupedSettings, reload } = useSettingCatalog();

  const activeGroupMeta = resolveSettingGroupMeta(groupKey);
  const activeSettingGroup =
    groupedSettings.find((item) => item.groupKey === activeGroupMeta.key) || groupedSettings[0];
  const groupIssueCounts = useMemo(() => {
    return (overview?.issues || []).reduce<Record<string, number>>((acc, issue) => {
      acc[issue.groupKey] = (acc[issue.groupKey] || 0) + 1;
      return acc;
    }, {});
  }, [overview]);
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

  useEffect(() => {
    if (groupKey && isSettingGroupKey(groupKey)) {
      return;
    }
    navigate(getSettingGroupPath(settingGroups[0].key), { replace: true });
  }, [groupKey, navigate]);

  const renderErrorState = () => {
    if (isNetworkRequestError(error)) {
      return (
        <PageNetworkError
          timeout={isTimeoutRequestError(error)}
          onRetry={() => {
            void reload();
          }}
        />
      );
    }
    if (isServerRequestError(error)) {
      return (
        <PageServerError
          onRetry={() => {
            void reload();
          }}
        />
      );
    }
    return (
      <PageError
        onRetry={() => {
          void reload();
        }}
      />
    );
  };

  return (
    <PageContainer>
      <Space direction="vertical" size={12} className="system-page-template setting-page setting-group-page">
        {loading && settings.length === 0 ? <PageLoading /> : null}
        {error && settings.length === 0 ? renderErrorState() : null}
        {!loading && !error && settings.length === 0 ? (
          <PageEmpty description={t('system.setting.empty')} />
        ) : null}
        {settings.length > 0 ? (
          <>
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
              <div className="setting-page__group-nav-grid">
                {groupedSettings.map((group) => {
                  const meta = resolveSettingGroupMeta(group.groupKey);
                  const issueCount = groupIssueCounts[group.groupKey] || 0;
                  const active = activeSettingGroup?.groupKey === group.groupKey;
                  return (
                    <button
                      key={group.groupKey}
                      type="button"
                      className={`setting-page__group-nav-item${active ? ' setting-page__group-nav-item--active' : ''}`}
                      onClick={() => {
                        navigate(getSettingGroupPath(group.groupKey as SettingGroupKey));
                      }}
                    >
                      <span className="setting-page__group-nav-title-row">
                        <span className="setting-page__group-nav-title">{t(meta.titleKey)}</span>
                        {issueCount > 0 ? (
                          <Tag color={meta.tone === 'danger' ? 'red' : 'orange'}>
                            {t('common.total', { count: issueCount })}
                          </Tag>
                        ) : null}
                      </span>
                      <span className="setting-page__group-nav-desc">
                        {t(meta.descriptionKey, '')}
                      </span>
                      <span className="setting-page__group-nav-meta">
                        {t('common.total', { count: group.items.length })}
                      </span>
                    </button>
                  );
                })}
              </div>
              {overview ? (
                <div className="setting-page__runtime-strip">
                  <Typography.Text type="secondary">
                    {t('system.setting.hero.storageHint')} {overview.storageDriver || '-'}
                  </Typography.Text>
                  <Typography.Text type="secondary">
                    {t('system.setting.hero.languageHint')} {overview.defaultLanguage || '-'}
                  </Typography.Text>
                  <Typography.Text type="secondary">
                    {t('system.setting.hero.themeHint')} {overview.defaultTheme || '-'}
                  </Typography.Text>
                </div>
              ) : null}
            </div>
            {activeSettingGroup ? (
              <SettingGroupWorkspace
                groupKey={activeSettingGroup.groupKey}
                groupItems={activeSettingGroup.items}
                canUpdateSetting={canUpdateSetting}
                canRefreshCache={canRefreshCache}
                canExportAudit={canExportAudit}
                canViewOperationLog={canViewOperationLog}
                showAuditCard
                onReload={reload}
                onViewOperationLog={(id) => {
                  navigate(`/system/operation-log?detailId=${id}`);
                }}
              />
            ) : null}
          </>
        ) : null}
      </Space>
    </PageContainer>
  );
};

export default SettingGroupPage;
