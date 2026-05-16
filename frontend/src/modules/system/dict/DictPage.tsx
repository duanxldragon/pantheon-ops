import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Card, Space, Tabs, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { usePermission } from '../../../hooks/usePermission';
import { useRefreshSubscription } from '../../../core/refresh/refreshBus';
import { resolveRouteWarmData } from '../../../core/router/prefetch';
import {
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  PageContainer,
  PageHeader,
  PageActions,
  useGovernanceRail,
} from '../../../components';
import { getDictTypeList, type DictTypeQuery, type DictTypeRow } from './api';
import DictTypeTab from './DictTypeTab';
import DictItemTab from './DictItemTab';
import '../../../core/styles/list-page.css';

const emptyTypeQuery: DictTypeQuery = {
  dictCode: '',
  dictName: '',
  status: undefined,
};

function isDefaultDictTypeQuery(query: DictTypeQuery) {
  return !query.dictCode && !query.dictName && query.status === undefined;
}

type DictTabKey = 'types' | 'items';
const DICT_SELECTED_TYPE_STORAGE_KEY = 'pantheon.dict.selectedTypeId';

const DictPage: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canCreate = isAdmin || hasPerm('system:dict:create');
  const canEdit = isAdmin || hasPerm('system:dict:update');
  const canDelete = isAdmin || hasPerm('system:dict:delete');
  const canBatchUpdate = isAdmin || hasPerm('system:dict:batch-update');
  const canBatchDelete = isAdmin || hasPerm('system:dict:batch-delete');
  const canRefresh = isAdmin || hasPerm('system:dict:refresh');
  const canExport = isAdmin || hasPerm('system:dict:export');
  const canImport = isAdmin || hasPerm('system:dict:import');

  const [typeRows, setTypeRows] = useState<DictTypeRow[]>([]);
  const [typeLoading, setTypeLoading] = useState(false);
  const [typeError, setTypeError] = useState<unknown>(null);
  const [typeQuery, setTypeQuery] = useState<DictTypeQuery>(emptyTypeQuery);
  const [activeTab, setActiveTab] = useState<DictTabKey>('types');
  const [selectedType, setSelectedType] = useState<DictTypeRow | null>(null);
  const selectedTypeId = selectedType?.id;
  const governanceRail = useGovernanceRail();
  const typeLoadRequestSeqRef = useRef(0);

  const selectType = useCallback((nextType: DictTypeRow | null) => {
    setSelectedType(nextType);
  }, []);

  const switchToItemsTab = useCallback(
    (row: DictTypeRow) => {
      selectType(row);
      setActiveTab('items');
    },
    [selectType],
  );

  const loadTypes = useCallback(
    async (nextQuery: DictTypeQuery = typeQuery) => {
      const requestSeq = typeLoadRequestSeqRef.current + 1;
      typeLoadRequestSeqRef.current = requestSeq;
      setTypeLoading(true);
      setTypeError(null);
      try {
        const rows = isDefaultDictTypeQuery(nextQuery)
          ? await resolveRouteWarmData('/system/dict', 'types:default', () =>
              getDictTypeList(nextQuery),
            )
          : await getDictTypeList(nextQuery);
        if (typeLoadRequestSeqRef.current !== requestSeq) {
          return;
        }
        setTypeRows(rows);
        if (rows.length === 0) {
          selectType(null);
          localStorage.removeItem(DICT_SELECTED_TYPE_STORAGE_KEY);
          return;
        }
        const savedTypeId = Number(localStorage.getItem(DICT_SELECTED_TYPE_STORAGE_KEY) || 0);
        const nextSelectedType =
          rows.find((item) => item.id === selectedTypeId) ||
          rows.find((item) => item.id === savedTypeId) ||
          rows[0];
        selectType(nextSelectedType);
      } catch (requestError) {
        if (typeLoadRequestSeqRef.current !== requestSeq) {
          return;
        }
        setTypeError(requestError);
      } finally {
        if (typeLoadRequestSeqRef.current === requestSeq) {
          setTypeLoading(false);
        }
      }
    },
    [selectType, selectedTypeId, typeQuery],
  );

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadTypes(typeQuery);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [loadTypes, typeQuery]);

  useRefreshSubscription('system:dict:changed', (payload) => {
    if (payload.source === 'system/dict') {
      return;
    }
    void loadTypes(typeQuery);
  });

  const typeSummary = useMemo(() => {
    const activeCount = typeRows.filter((item) => item.status === 1).length;
    const totalItems = typeRows.reduce((sum, item) => sum + (item.itemCount || 0), 0);
    return {
      total: typeRows.length,
      active: activeCount,
      disabled: typeRows.length - activeCount,
      items: totalItems,
    };
  }, [typeRows]);

  const heroStats = useMemo(
    () => [
      {
        key: 'types',
        label: t('system.dict.type'),
        value: typeSummary.total,
        hint: t('system.dict.hero.typeHint'),
      },
      {
        key: 'active',
        label: t('system.user.status.enabled'),
        value: typeSummary.active,
        hint: t('system.dict.hero.activeHint'),
      },
      {
        key: 'items',
        label: t('system.dict.item'),
        value: typeSummary.items,
        hint: t('system.dict.hero.itemHint'),
      },
      {
        key: 'selected',
        label: t('system.dict.hero.currentType'),
        value: selectedType?.dictCode || '-',
        hint: t('system.dict.hero.currentHint'),
      },
    ],
    [selectedType?.dictCode, t, typeSummary.active, typeSummary.items, typeSummary.total],
  );

  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('system.dict.hero.disabledTypes'),
        value: typeSummary.disabled,
        description: t('system.dict.hero.disabledHint'),
      },
      {
        label: t('system.dict.hero.refreshReady'),
        value: canRefresh ? t('common.yes') : t('common.no'),
        description: t('system.dict.hero.refreshHint'),
      },
      {
        label: t('system.dict.hero.importReady'),
        value: canImport ? t('common.yes') : t('common.no'),
        description: t('system.dict.hero.importHint'),
      },
    ],
    [canImport, canRefresh, t, typeSummary.disabled],
  );

  useEffect(() => {
    if (selectedType?.id) {
      localStorage.setItem(DICT_SELECTED_TYPE_STORAGE_KEY, String(selectedType.id));
      return;
    }
    localStorage.removeItem(DICT_SELECTED_TYPE_STORAGE_KEY);
  }, [selectedType?.id]);

  return (
    <PageContainer>
      <PageHeader
        title={t('system.menu.dict')}
        extra={
          <PageActions>
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('system.dict.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          </PageActions>
        }
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero">
          <div className="system-page-hero__top">
            <div className="system-page-hero__copy">
              <span className="system-page-hero__eyebrow">{t('system.dict.hero.eyebrow')}</span>
              <Typography.Title heading={5} className="system-page-hero__title">
                {t('system.dict.hero.title')}
              </Typography.Title>
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
        <Card className="page-panel system-list__table-card dict-page__table-card">
          <Tabs activeTab={activeTab} onChange={(value) => setActiveTab(value as DictTabKey)}>
            <Tabs.TabPane key="types" title={t('system.dict.type')}>
              <DictTypeTab
                typeRows={typeRows}
                typeLoading={typeLoading}
                typeError={typeError}
                typeQuery={typeQuery}
                typeSummary={typeSummary}
            canCreate={canCreate}
            canEdit={canEdit}
            canDelete={canDelete}
            canBatchUpdate={canBatchUpdate}
            canBatchDelete={canBatchDelete}
            canExport={canExport}
                canImport={canImport}
                onQueryChange={setTypeQuery}
                onReload={() => {
                  void loadTypes();
                }}
                onSelectType={selectType}
                onSwitchToItemsTab={switchToItemsTab}
              />
            </Tabs.TabPane>
            <Tabs.TabPane key="items" title={t('system.dict.item')}>
              <DictItemTab
                key={selectedType?.id ?? '__empty__'}
                selectedType={selectedType}
                typeRows={typeRows}
            canCreate={canCreate}
            canEdit={canEdit}
            canDelete={canDelete}
            canBatchUpdate={canBatchUpdate}
            canBatchDelete={canBatchDelete}
            canRefresh={canRefresh}
                canExport={canExport}
                canImport={canImport}
                onSelectType={selectType}
                onReloadTypes={() => {
                  void loadTypes(typeQuery);
                }}
              />
            </Tabs.TabPane>
          </Tabs>
        </Card>
      </Space>

      <GovernanceInsightDrawer
        title={t('system.dict.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('system.dict.hero.sideLead')}
        noteDescription={t('system.dict.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
    </PageContainer>
  );
};

export default DictPage;
