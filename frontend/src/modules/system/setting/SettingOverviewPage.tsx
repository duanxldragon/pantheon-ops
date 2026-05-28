import React, { useMemo } from 'react';
import { Button, Card, List, Space, Tag, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import {
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
} from '../../../components';
import { usePermission } from '../../../hooks/usePermission';
import {
  getSettingGroupPath,
  resolveSettingGroupMeta,
  settingGroups,
  type SettingGroupKey,
} from './settingGroups';
import { useSettingCatalog } from './useSettingCatalog';
import '../list-page.css';

const SettingOverviewPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { isAdmin, hasPerm } = usePermission();
  const canViewSettings = isAdmin || hasPerm('system:setting:list');
  const {
    loading,
    error,
    settings,
    overview,
    groupedSettings,
  } = useSettingCatalog();

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

  const renderErrorState = () => {
    if (isNetworkRequestError(error)) {
      return <PageNetworkError timeout={isTimeoutRequestError(error)} onRetry={() => window.location.reload()} />;
    }
    if (isServerRequestError(error)) {
      return <PageServerError onRetry={() => window.location.reload()} />;
    }
    return <PageError onRetry={() => window.location.reload()} />;
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
                  return (
                    <button
                      key={group.groupKey}
                      type="button"
                      className="setting-page__group-nav-item"
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
            <div className="setting-overview-page__group-grid">
              {settingGroups
                .filter((meta) => groupedSettings.some((group) => group.groupKey === meta.key))
                .map((meta) => {
                  const group = groupedSettings.find((item) => item.groupKey === meta.key);
                  const issueCount = groupIssueCounts[meta.key] || 0;
                  return (
                    <Card
                      key={meta.key}
                      className="page-panel setting-overview-page__group-card"
                    >
                      <Space direction="vertical" size={10} style={{ width: '100%' }}>
                        <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
                          <Typography.Text style={{ fontWeight: 600 }}>
                            {t(meta.titleKey)}
                          </Typography.Text>
                          {issueCount > 0 ? (
                            <Tag color={meta.tone === 'danger' ? 'red' : 'orange'}>
                              {t('system.setting.overview.risks')}
                            </Tag>
                          ) : null}
                        </Space>
                        <Typography.Paragraph type="secondary" style={{ margin: 0 }}>
                          {t(meta.descriptionKey, '')}
                        </Typography.Paragraph>
                        <Typography.Text type="secondary">
                          {t('common.total', { count: group?.items.length ?? 0 })}
                        </Typography.Text>
                        <Button
                          type="outline"
                          size="small"
                          disabled={!canViewSettings}
                          onClick={() => {
                            navigate(getSettingGroupPath(meta.key));
                          }}
                        >
                          {t('system.setting.overview.openGroup')}
                        </Button>
                      </Space>
                    </Card>
                  );
                })}
              <Card className="page-panel setting-overview-page__group-card">
                <Space direction="vertical" size={10} style={{ width: '100%' }}>
                  <Typography.Text style={{ fontWeight: 600 }}>
                    {t('system.setting.overview.runtime')}
                  </Typography.Text>
                  {overview?.issues && overview.issues.length > 0 ? (
                    <List
                      size="small"
                      split={false}
                      dataSource={overview.issues}
                      render={(issue) => (
                        <List.Item>
                          <Space direction="vertical" size={4}>
                            <Space size={6}>
                              <Tag color={issue.severity === 'critical' ? 'red' : 'orange'}>
                                {t(`system.setting.overview.severity.${issue.severity}`)}
                              </Tag>
                              <Typography.Text style={{ fontWeight: 500 }}>
                                {t(`system.setting.item.${issue.settingKey}`, issue.settingKey)}
                              </Typography.Text>
                            </Space>
                            <Typography.Text type="secondary">
                              {t(issue.reasonKey)}
                            </Typography.Text>
                          </Space>
                        </List.Item>
                      )}
                    />
                  ) : (
                    <Typography.Text type="secondary">
                      {t('system.setting.overview.noRisks')}
                    </Typography.Text>
                  )}
                </Space>
              </Card>
            </div>
          </>
        ) : null}
      </Space>
    </PageContainer>
  );
};

export default SettingOverviewPage;
