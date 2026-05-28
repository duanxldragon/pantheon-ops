import React, { useEffect, useState } from 'react';
import { Table } from '@arco-design/web-react';
import type { PaginationProps } from '@arco-design/web-react/es/Pagination/interface';
import type { ColumnProps, SorterInfo, TableProps } from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import PageEmpty from '../feedback/PageEmpty';
import {
  getPaginationCurrentPage,
  getPaginationPageSize,
  getPaginationTotalPages,
  isPaginationConfig,
} from '../table/crossPageSelection';

interface AppTableProps<T> extends TableProps<T> {
  emptyText?: React.ReactNode;
}

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

type TablePagePosition =
  | 'tl'
  | 'tr'
  | 'bl'
  | 'br'
  | 'topCenter'
  | 'bottomCenter'
  | undefined;

function needsHorizontalScroll<T>(columns?: ColumnProps<T>[]): boolean {
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

function getColumnClassName<T>(column: ColumnProps<T>) {
  if (Array.isArray(column.className)) {
    return column.className.join(' ');
  }
  return column.className || '';
}

function filterResponsiveColumns<T>(
  columns: ColumnProps<T>[] | undefined,
  viewportWidth: number,
): ColumnProps<T>[] | undefined {
  if (!Array.isArray(columns) || columns.length === 0) {
    return columns;
  }

  return columns.reduce<ColumnProps<T>[]>((result, column) => {
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

    result.push(column);
    return result;
  }, []);
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

function renderNativePagination(
  paginationNode: React.ReactNode,
  pagePosition: TablePagePosition,
) {
  return (
    <div className={getPaginationWrapperClassName(pagePosition)}>
      <div className="app-table__pagination-shell">
        <div className="app-table__pagination-native">{paginationNode}</div>
      </div>
    </div>
  );
}

function AppTable<T>(props: AppTableProps<T>) {
  const {
    data,
    loading,
    emptyText,
    columns,
    scroll,
    pagination,
    renderPagination,
    pagePosition,
    ...rest
  } = props;
  const { t } = useTranslation();
  const rows = Array.isArray(data) ? data : [];
  const [viewportWidth, setViewportWidth] = useState(() =>
    typeof globalThis.document === 'undefined' ? 1920 : globalThis.innerWidth,
  );

  useEffect(() => {
    if (typeof globalThis.document === 'undefined') {
      return undefined;
    }

    const syncViewportWidth = () => {
      setViewportWidth(globalThis.innerWidth);
    };

    syncViewportWidth();
    globalThis.addEventListener('resize', syncViewportWidth);
    return () => globalThis.removeEventListener('resize', syncViewportWidth);
  }, []);

  const responsiveColumns = filterResponsiveColumns(columns, viewportWidth);
  const effectiveScroll =
    scroll?.x !== undefined || !needsHorizontalScroll(responsiveColumns)
      ? scroll
      : { ...scroll, x: 'max-content' as const };

  if (!loading && rows.length === 0) {
    return <PageEmpty description={emptyText} />;
  }

  const firstPageAriaLabel = t('common.pagination.firstPage', { defaultValue: 'First page' });
  const lastPageAriaLabel = t('common.pagination.lastPage', { defaultValue: 'Last page' });

  const enhancedRenderPagination =
    isPaginationConfig(pagination)
      ? (paginationNode?: React.ReactNode) => {
          if (!React.isValidElement<PaginationNodeProps>(paginationNode)) {
            return renderPagination
              ? renderPagination(paginationNode)
              : renderNativePagination(paginationNode, pagePosition);
          }

          const shouldDecorate =
            !paginationNode.props.simple && getPaginationTotalPages(paginationNode.props) > 1;

          if (!shouldDecorate) {
            return renderPagination
              ? renderPagination(paginationNode)
              : renderNativePagination(paginationNode, pagePosition);
          }

          const originalItemRender = paginationNode.props.itemRender;
          const decoratedPaginationNode = React.cloneElement(paginationNode, {
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
                      rest.onChange,
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
                      rest.onChange,
                    )}
                  </span>
                );
              }

              return renderedOrigin;
            },
          });
          const callerNode = renderPagination
            ? renderPagination(decoratedPaginationNode)
            : decoratedPaginationNode;

          return renderPagination
            ? callerNode
            : renderNativePagination(callerNode, pagePosition);
        }
      : renderPagination;

  return (
    <div className="app-table-shell">
      {viewportWidth <= 768 ? (
        <div className="app-table__mobile-hint">
          <span>{t('common.tableRecordSummary', { count: rows.length })}</span>
          {needsHorizontalScroll(responsiveColumns) ? (
            <span>{t('common.tableSwipeHint')}</span>
          ) : null}
        </div>
      ) : null}
      <Table
        {...rest}
        className={rest.className ? `app-table ${rest.className}` : 'app-table'}
        columns={responsiveColumns}
        scroll={effectiveScroll}
        size={rest.size || 'small'}
        data={rows}
        loading={loading}
        pagePosition={pagePosition}
        pagination={pagination}
        renderPagination={enhancedRenderPagination}
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
