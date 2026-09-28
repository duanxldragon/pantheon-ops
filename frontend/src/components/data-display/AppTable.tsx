import React, { useEffect, useMemo, useState } from 'react';
import { Button, Checkbox, Dropdown, Menu, Table, Tooltip } from '@arco-design/web-react';
import { IconArrowDown, IconArrowUp, IconRefresh, IconSettings } from '@arco-design/web-react/icon';
import type { PaginationProps } from '@arco-design/web-react/es/Pagination/interface';
import type {
  ColumnProps,
  SorterInfo,
  TableProps,
} from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import PageEmpty from '../feedback/PageEmpty';
import {
  getPaginationCurrentPage,
  getPaginationPageSize,
  getPaginationTotalPages,
  isPaginationConfig,
} from '../table/crossPageSelection';
import {
  applyAppTablePreferences,
  createDefaultAppTablePreferences,
  readAppTablePreferences,
  resolveAppTableColumnKey,
  writeAppTablePreferences,
  type AppTableColumnMeta,
  type AppTableDensity,
  type AppTablePreferences,
} from './tablePreferences';

interface AppTableProps<T> extends TableProps<T> {
  emptyText?: React.ReactNode;
  viewKey?: string;
  defaultDensity?: AppTableDensity;
}

type AppTableColumnProps<T> = ColumnProps<T> & {
  columnKey?: string;
  defaultVisible?: boolean;
  hideable?: boolean;
};

type PaginationNodeProps = PaginationProps & {
  children?: React.ReactNode;
};

type TableChangeHandler<T> = (
  pagination: PaginationProps,
  sorter: SorterInfo | SorterInfo[],
  filters: Partial<Record<keyof T, string[]>>,
  extra: {
    currentData: T[];
    currentAllData: T[];
    action: 'sort' | 'filter' | 'paginate';
  },
) => void;

type TablePagePosition = 'tl' | 'tr' | 'bl' | 'br' | 'topCenter' | 'bottomCenter' | undefined;

const DEFAULT_SELECTION_COLUMN_WIDTH = 44;

function needsHorizontalScroll<T>(columns?: AppTableColumnProps<T>[]): boolean {
  if (!Array.isArray(columns) || columns.length === 0) {
    return false;
  }

  return columns.some((column) => {
    if (Array.isArray(column.children) && column.children.length > 0) {
      return needsHorizontalScroll(column.children);
    }
    return (
      Boolean(column.fixed) || typeof column.width === 'number' || typeof column.width === 'string'
    );
  });
}

function getColumnClassName<T>(column: AppTableColumnProps<T>) {
  if (Array.isArray(column.className)) {
    return column.className.join(' ');
  }
  return column.className || '';
}

function filterResponsiveColumns<T>(
  columns: AppTableColumnProps<T>[] | undefined,
  viewportWidth: number,
): AppTableColumnProps<T>[] | undefined {
  if (!Array.isArray(columns) || columns.length === 0) {
    return columns;
  }

  return columns.reduce<AppTableColumnProps<T>[]>((result, column) => {
    const className = getColumnClassName(column);
    const hideOnLarge = className.includes('app-table__col--hide-lg') && viewportWidth <= 1440;
    const hideOnMedium = className.includes('app-table__col--hide-md') && viewportWidth <= 1280;

    if (hideOnLarge || hideOnMedium) {
      return result;
    }

    if (Array.isArray(column.children) && column.children.length > 0) {
      const children = filterResponsiveColumns(column.children, viewportWidth);
      if (!children || children.length === 0) {
        return result;
      }
      result.push({ ...column, children });
      return result;
    }

    // On phone-width viewports a fixed action column can cover almost the whole
    // table, hiding the data columns behind it; let it scroll with the rest.
    if (column.fixed && viewportWidth <= 768) {
      result.push({ ...column, fixed: undefined });
      return result;
    }

    result.push(column);
    return result;
  }, []);
}

function getAppTableColumnMeta<T>(
  columns: AppTableColumnProps<T>[] | undefined,
): AppTableColumnMeta[] {
  if (!Array.isArray(columns)) {
    return [];
  }
  return columns.reduce<AppTableColumnMeta[]>((result, column) => {
    const key = resolveAppTableColumnKey(column);
    if (!key) {
      return result;
    }
    const meta: AppTableColumnMeta = {
      key,
      defaultVisible: column.defaultVisible !== false,
      hideable: column.hideable !== false,
    };
    if (column.width !== undefined) {
      meta.width = column.width;
    }
    result.push(meta);
    return result;
  }, []);
}

function getColumnTitleText(title: React.ReactNode, fallback: string) {
  return typeof title === 'string' || typeof title === 'number' ? String(title) : fallback;
}

function getStorage() {
  return globalThis.window?.localStorage;
}

function createBoundaryPaginationItem<T>(
  type: 'first' | 'last',
  paginationProps: PaginationNodeProps,
  ariaLabel: string,
  onTableChange?: TableChangeHandler<T>,
) {
  const currentPage = getPaginationCurrentPage(paginationProps);
  const totalPages = getPaginationTotalPages(paginationProps);
  const pageSize = getPaginationPageSize(paginationProps);
  const isFirst = type === 'first';
  const targetPage = isFirst ? 1 : totalPages;
  const disabled =
    Boolean(paginationProps.disabled) ||
    totalPages <= 1 ||
    (isFirst ? currentPage <= 1 : currentPage >= totalPages);
  const classNames = ['arco-pagination-item', 'arco-pagination-item-step'];
  if (disabled) {
    classNames.push('arco-pagination-item-disabled');
  }

  const triggerBoundaryPageChange = () => {
    if (disabled) {
      return;
    }
    if (paginationProps.onChange) {
      paginationProps.onChange(targetPage, pageSize);
      return;
    }
    onTableChange?.(
      { current: targetPage, pageSize },
      [],
      {},
      {
        currentData: [],
        currentAllData: [],
        action: 'paginate',
      },
    );
  };

  return (
    <button
      type="button"
      className={[...classNames, `app-table__pagination-item-${type}`].join(' ')}
      aria-label={ariaLabel}
      disabled={disabled}
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        triggerBoundaryPageChange();
      }}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          event.stopPropagation();
          triggerBoundaryPageChange();
        }
      }}
    >
      <span aria-hidden="true" className="app-table__pagination-boundary-glyph">
        {isFirst ? '《' : '》'}
      </span>
    </button>
  );
}

function renderNativePagination(paginationNode: React.ReactNode, pagePosition: TablePagePosition) {
  return (
    <div className={getPaginationWrapperClassName(pagePosition)}>
      <div className="app-table__pagination-shell">
        <div className="app-table__pagination-native">{paginationNode}</div>
      </div>
    </div>
  );
}

function createDecoratedPaginationNode<T>(
  paginationNode: React.ReactElement<PaginationNodeProps>,
  firstPageAriaLabel: string,
  lastPageAriaLabel: string,
  onTableChange?: TableChangeHandler<T>,
) {
  const originalItemRender = paginationNode.props.itemRender;
  return React.createElement(paginationNode.type as React.ElementType<PaginationNodeProps>, {
    ...paginationNode.props,
    itemRender: (page, type, originElement) => {
      const renderedOrigin = originalItemRender
        ? originalItemRender(page, type, originElement)
        : originElement;

      if (type === 'prev') {
        return (
          <span className="app-table__pagination-step-group">
            {createBoundaryPaginationItem<T>(
              'first',
              paginationNode.props,
              firstPageAriaLabel,
              onTableChange,
            )}
            <span className="app-table__pagination-step-origin">{renderedOrigin}</span>
          </span>
        );
      }

      if (type === 'next') {
        return (
          <span className="app-table__pagination-step-group">
            <span className="app-table__pagination-step-origin">{renderedOrigin}</span>
            {createBoundaryPaginationItem<T>(
              'last',
              paginationNode.props,
              lastPageAriaLabel,
              onTableChange,
            )}
          </span>
        );
      }

      return renderedOrigin;
    },
  });
}

function useAppTableViewPreferences(
  viewKey: string | undefined,
  columnMeta: AppTableColumnMeta[],
  defaultDensity: AppTableDensity,
) {
  const canPersistView = Boolean(viewKey && columnMeta.length > 0);
  const persistedPreferences = useMemo(
    () => {
      if (!canPersistView || !viewKey) {
        return null;
      }
      return (
        readAppTablePreferences(getStorage(), viewKey, columnMeta) ||
        createDefaultAppTablePreferences(viewKey, columnMeta, defaultDensity)
      );
    },
    [canPersistView, viewKey, defaultDensity, columnMeta],
  );
  const [localPreferences, setLocalPreferences] = useState<{
    viewKey: string;
    preferences: AppTablePreferences;
  } | null>(null);
  const preferences =
    localPreferences && localPreferences.viewKey === viewKey
      ? localPreferences.preferences
      : persistedPreferences;

  const updatePreferences = (nextPreferences: AppTablePreferences) => {
    if (!viewKey) {
      return;
    }
    setLocalPreferences({ viewKey, preferences: nextPreferences });
    writeAppTablePreferences(getStorage(), {
      ...nextPreferences,
      updatedAt: new Date().toISOString(),
    });
  };

  const setColumnVisible = (key: string, visible: boolean) => {
    if (!preferences) {
      return;
    }
    updatePreferences({
      ...preferences,
      columns: preferences.columns.map((column) =>
        column.key === key ? { ...column, visible } : column,
      ),
    });
  };

  const moveColumn = (key: string, direction: -1 | 1) => {
    if (!preferences) {
      return;
    }
    const ordered = [...preferences.columns].sort((left, right) => left.order - right.order);
    const currentIndex = ordered.findIndex((column) => column.key === key);
    const targetIndex = currentIndex + direction;
    if (currentIndex < 0 || targetIndex < 0 || targetIndex >= ordered.length) {
      return;
    }
    const [current] = ordered.splice(currentIndex, 1);
    ordered.splice(targetIndex, 0, current);
    updatePreferences({
      ...preferences,
      columns: ordered.map((column, index) => ({ ...column, order: index })),
    });
  };

  const resetPreferences = () => {
    if (viewKey) {
      updatePreferences(createDefaultAppTablePreferences(viewKey, columnMeta, defaultDensity));
    }
  };

  const setDensity = (density: AppTableDensity) => {
    if (preferences) {
      updatePreferences({ ...preferences, density });
    }
  };

  return {
    canPersistView,
    preferences,
    setColumnVisible,
    moveColumn,
    resetPreferences,
    setDensity,
  };
}

interface AppTableViewSettingsProps<T> {
  tableColumns: AppTableColumnProps<T>[] | undefined;
  columnMeta: AppTableColumnMeta[];
  preferences: AppTablePreferences;
  setDensity: (density: AppTableDensity) => void;
  setColumnVisible: (key: string, visible: boolean) => void;
  moveColumn: (key: string, direction: -1 | 1) => void;
  resetPreferences: () => void;
}

function AppTableViewSettings<T>({
  tableColumns,
  columnMeta,
  preferences,
  setDensity,
  setColumnVisible,
  moveColumn,
  resetPreferences,
}: Readonly<AppTableViewSettingsProps<T>>) {
  const { t } = useTranslation();
  const orderedPreferences = [...preferences.columns].sort((left, right) => left.order - right.order);

  return (
    <Menu className="app-table__view-menu">
      <Menu.Item key="density" disabled>
        {t('common.tableView.density')}
      </Menu.Item>
      {(['compact', 'standard', 'comfortable'] as AppTableDensity[]).map((density) => (
        <Menu.Item key={`density-${density}`} onClick={() => setDensity(density)}>
          <span className="app-table__view-menu-row">
            <span>{t(`common.tableView.density.${density}`)}</span>
            {preferences.density === density ? <span aria-hidden="true">✓</span> : null}
          </span>
        </Menu.Item>
      ))}
      <Menu.Item key="columns" disabled>
        {t('common.tableView.columns')}
      </Menu.Item>
      {orderedPreferences.map((preference, index) => {
        const sourceColumn = tableColumns?.find(
          (column) => resolveAppTableColumnKey(column) === preference.key,
        );
        const meta = columnMeta.find((column) => column.key === preference.key);
        const label = getColumnTitleText(sourceColumn?.title, preference.key);
        return (
          <Menu.Item key={`column-${preference.key}`} className="app-table__view-column-item">
            <span className="app-table__view-column">
              <Checkbox
                checked={preference.visible}
                disabled={!meta?.hideable}
                onChange={(checked) => setColumnVisible(preference.key, Boolean(checked))}
              >
                {label}
              </Checkbox>
              <span className="app-table__view-column-order">
                <Button
                  size="mini"
                  icon={<IconArrowUp />}
                  disabled={index === 0}
                  aria-label={t('common.tableView.moveColumnUp', { column: label })}
                  onClick={(event) => {
                    event.stopPropagation();
                    moveColumn(preference.key, -1);
                  }}
                />
                <Button
                  size="mini"
                  icon={<IconArrowDown />}
                  disabled={index === orderedPreferences.length - 1}
                  aria-label={t('common.tableView.moveColumnDown', { column: label })}
                  onClick={(event) => {
                    event.stopPropagation();
                    moveColumn(preference.key, 1);
                  }}
                />
              </span>
            </span>
          </Menu.Item>
        );
      })}
      <Menu.Item key="reset" onClick={resetPreferences}>
        <IconRefresh /> {t('common.tableView.reset')}
      </Menu.Item>
    </Menu>
  );
}

function createEnhancedPaginationRenderer<T>({
  pagination,
  renderPagination,
  pagePosition,
  firstPageAriaLabel,
  lastPageAriaLabel,
  onTableChange,
}: {
  pagination: TableProps<T>['pagination'];
  renderPagination: TableProps<T>['renderPagination'];
  pagePosition: TablePagePosition;
  firstPageAriaLabel: string;
  lastPageAriaLabel: string;
  onTableChange: TableChangeHandler<T> | undefined;
}) {
  if (!isPaginationConfig(pagination)) {
    return renderPagination;
  }

  return (paginationNode?: React.ReactNode) => {
    if (!React.isValidElement<PaginationNodeProps>(paginationNode)) {
      return renderPagination
        ? renderPagination(paginationNode)
        : renderNativePagination(paginationNode, pagePosition);
    }
    const shouldDecorate =
      !paginationNode.props.simple && getPaginationTotalPages(paginationNode.props) > 1;
    const decoratedPaginationNode = shouldDecorate
      ? createDecoratedPaginationNode<T>(
          paginationNode,
          firstPageAriaLabel,
          lastPageAriaLabel,
          onTableChange,
        )
      : paginationNode;
    const callerNode = renderPagination
      ? renderPagination(decoratedPaginationNode)
      : decoratedPaginationNode;
    return renderPagination ? callerNode : renderNativePagination(callerNode, pagePosition);
  };
}

interface AppTableContentProps<T> {
  emptyText: React.ReactNode;
  enhancedRenderPagination: TableProps<T>['renderPagination'];
  effectiveRowSelection: TableProps<T>['rowSelection'];
  effectiveScroll: TableProps<T>['scroll'];
  loading: TableProps<T>['loading'];
  pagePosition: TablePagePosition;
  pagination: TableProps<T>['pagination'];
  responsiveColumns: AppTableColumnProps<T>[] | undefined;
  rows: T[];
  settingsPanel: React.ReactNode;
  tableProps: Omit<
    AppTableProps<T>,
    | 'columns'
    | 'data'
    | 'defaultDensity'
    | 'emptyText'
    | 'loading'
    | 'pagePosition'
    | 'pagination'
    | 'renderPagination'
    | 'scroll'
    | 'viewKey'
  >;
  viewportWidth: number;
}

function AppTableContent<T>({
  emptyText,
  enhancedRenderPagination,
  effectiveRowSelection,
  effectiveScroll,
  loading,
  pagePosition,
  pagination,
  responsiveColumns,
  rows,
  settingsPanel,
  tableProps,
  viewportWidth,
}: Readonly<AppTableContentProps<T>>) {
  const { t } = useTranslation();

  return (
    <>
      {settingsPanel ? (
        <div className="app-table__view-toolbar">
          <Dropdown trigger="click" position="br" droplist={settingsPanel}>
            <Tooltip content={t('common.tableView.settings')}>
              <Button
                size="small"
                icon={<IconSettings />}
                aria-label={t('common.tableView.settings')}
              >
                {viewportWidth > 768 ? t('common.tableView.settings') : null}
              </Button>
            </Tooltip>
          </Dropdown>
        </div>
      ) : null}
      {viewportWidth <= 768 ? (
        <div className="app-table__mobile-hint">
          <span>{t('common.tableRecordSummary', { count: rows.length })}</span>
          {needsHorizontalScroll(responsiveColumns) ? <span>{t('common.tableSwipeHint')}</span> : null}
        </div>
      ) : null}
      {!loading && rows.length === 0 ? (
        <PageEmpty description={emptyText} />
      ) : (
        <Table
          {...tableProps}
          className={tableProps.className ? `app-table ${tableProps.className}` : 'app-table'}
          columns={responsiveColumns}
          scroll={effectiveScroll}
          size={tableProps.size || 'small'}
          data={rows}
          loading={loading}
          rowSelection={effectiveRowSelection}
          pagePosition={pagePosition}
          pagination={pagination}
          renderPagination={enhancedRenderPagination}
        />
      )}
    </>
  );
}

function AppTable<T>(props: Readonly<AppTableProps<T>>) {
  const {
    data,
    loading,
    emptyText,
    columns,
    viewKey,
    defaultDensity = 'standard',
    scroll,
    pagination,
    renderPagination,
    pagePosition,
    ...rest
  } = props;
  const { t } = useTranslation();
  const rows = Array.isArray(data) ? data : [];
  const tableColumns = columns as AppTableColumnProps<T>[] | undefined;
  const columnMeta = getAppTableColumnMeta(tableColumns);
  const {
    canPersistView,
    preferences,
    setColumnVisible,
    moveColumn,
    resetPreferences,
    setDensity,
  } = useAppTableViewPreferences(viewKey, columnMeta, defaultDensity);
  const [viewportWidth, setViewportWidth] = useState(() =>
    globalThis.window === undefined ? Number.MAX_SAFE_INTEGER : globalThis.innerWidth,
  );

  useEffect(() => {
    if (globalThis.window === undefined) {
      return undefined;
    }

    const syncViewportWidth = () => {
      setViewportWidth(globalThis.innerWidth);
    };

    syncViewportWidth();
    globalThis.addEventListener('resize', syncViewportWidth);
    return () => globalThis.removeEventListener('resize', syncViewportWidth);
  }, []);

  const preferredColumns = preferences
    ? applyAppTablePreferences(tableColumns || [], preferences)
    : tableColumns;
  const responsiveColumns = filterResponsiveColumns(preferredColumns, viewportWidth);
  const effectiveScroll =
    scroll?.x !== undefined || !needsHorizontalScroll(responsiveColumns)
      ? scroll
      : { ...scroll, x: 'max-content' as const };
  const effectiveRowSelection =
    rest.rowSelection && typeof rest.rowSelection === 'object'
      ? {
          columnWidth: DEFAULT_SELECTION_COLUMN_WIDTH,
          ...rest.rowSelection,
        }
      : rest.rowSelection;

  const firstPageAriaLabel = t('common.pagination.firstPage', { defaultValue: 'First page' });
  const lastPageAriaLabel = t('common.pagination.lastPage', { defaultValue: 'Last page' });
  const settingsPanel =
    canPersistView && preferences ? (
      <AppTableViewSettings
        tableColumns={tableColumns}
        columnMeta={columnMeta}
        preferences={preferences}
        setDensity={setDensity}
        setColumnVisible={setColumnVisible}
        moveColumn={moveColumn}
        resetPreferences={resetPreferences}
      />
    ) : null;
  const enhancedRenderPagination = createEnhancedPaginationRenderer<T>({
    pagination,
    renderPagination,
    pagePosition,
    firstPageAriaLabel,
    lastPageAriaLabel,
    onTableChange: rest.onChange,
  });
  const shellClassName = preferences
    ? `app-table-shell app-table-shell--density-${preferences.density}`
    : 'app-table-shell';

  return (
    <div className={shellClassName}>
      <AppTableContent
        emptyText={emptyText}
        enhancedRenderPagination={enhancedRenderPagination}
        effectiveRowSelection={effectiveRowSelection}
        effectiveScroll={effectiveScroll}
        loading={loading}
        pagePosition={pagePosition}
        pagination={pagination}
        responsiveColumns={responsiveColumns}
        rows={rows}
        settingsPanel={settingsPanel}
        tableProps={rest}
        viewportWidth={viewportWidth}
      />
    </div>
  );
}
function getPaginationWrapperClassName(pagePosition: TablePagePosition) {
  const classNames = ['arco-table-pagination'];
  if (pagePosition === 'tl' || pagePosition === 'bl') {
    classNames.push('arco-table-pagination-left');
  }
  if (pagePosition === 'topCenter' || pagePosition === 'bottomCenter') {
    classNames.push('arco-table-pagination-center');
  }
  if (pagePosition === 'tl' || pagePosition === 'tr' || pagePosition === 'topCenter') {
    classNames.push('arco-table-pagination-top');
  }
  return classNames.join(' ');
}

export default AppTable;
