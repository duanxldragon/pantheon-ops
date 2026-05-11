import React, { useEffect, useState } from 'react';
import { Table } from '@arco-design/web-react';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import PageEmpty from '../feedback/PageEmpty';

interface AppTableProps<T> extends TableProps<T> {
  emptyText?: React.ReactNode;
}

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

function AppTable<T>(props: AppTableProps<T>) {
  const { data, loading, emptyText, columns, scroll, ...rest } = props;
  const { t } = useTranslation();
  const rows = Array.isArray(data) ? data : [];
  const [viewportWidth, setViewportWidth] = useState(() =>
    typeof window === 'undefined' ? 1920 : window.innerWidth,
  );

  useEffect(() => {
    if (typeof window === 'undefined') {
      return undefined;
    }

    const syncViewportWidth = () => {
      setViewportWidth(window.innerWidth);
    };

    syncViewportWidth();
    window.addEventListener('resize', syncViewportWidth);
    return () => window.removeEventListener('resize', syncViewportWidth);
  }, []);

  const responsiveColumns = filterResponsiveColumns(columns, viewportWidth);
  const effectiveScroll =
    scroll?.x !== undefined || !needsHorizontalScroll(responsiveColumns)
      ? scroll
      : { ...scroll, x: 'max-content' as const };

  if (!loading && rows.length === 0) {
    return <PageEmpty description={emptyText} />;
  }

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
      />
    </div>
  );
}

export default AppTable;
