import React, { useEffect, useState } from 'react';
import { Button, Card, Typography } from '@arco-design/web-react';
import { IconDown, IconUp } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';

interface FilterPanelProps {
  children: React.ReactNode;
  title?: React.ReactNode;
  collapsible?: boolean;
  defaultCollapsed?: boolean;
  mobileCollapsed?: boolean;
}

const FilterPanel: React.FC<FilterPanelProps> = ({
  children,
  title,
  collapsible = true,
  defaultCollapsed = false,
  mobileCollapsed = true,
}) => {
  const { t } = useTranslation();
  const [collapsed, setCollapsed] = useState(defaultCollapsed);

  useEffect(() => {
    if (!mobileCollapsed || globalThis.document === undefined) {
      return undefined;
    }

    const mediaQuery = globalThis.matchMedia('(max-width: 768px)');
    const syncCollapsed = () => {
      if (mediaQuery.matches) {
        setCollapsed(true);
      }
    };

    syncCollapsed();
    mediaQuery.addEventListener('change', syncCollapsed);
    return () => mediaQuery.removeEventListener('change', syncCollapsed);
  }, [mobileCollapsed]);

  const panelTitle = title ?? t('common.filters');
  const canToggle = collapsible;
  const effectiveCollapsed = canToggle ? collapsed : false;

  return (
    <Card className={effectiveCollapsed ? 'filter-panel filter-panel--collapsed' : 'filter-panel'}>
      {canToggle ? (
        <div className="filter-panel__header">
          <Typography.Text className="filter-panel__title">{panelTitle}</Typography.Text>
          <Button
            size="small"
            type="text"
            className="filter-panel__toggle"
            aria-expanded={!collapsed}
            aria-label={collapsed ? t('common.expandFilters') : t('common.collapseFilters')}
            onClick={() => setCollapsed((value) => !value)}
          >
            {collapsed ? <IconDown /> : <IconUp />}
            {collapsed ? t('common.expand') : t('common.collapse')}
          </Button>
        </div>
      ) : null}
      <div className="filter-panel__body" hidden={effectiveCollapsed}>
        {children}
      </div>
    </Card>
  );
};

export default FilterPanel;
