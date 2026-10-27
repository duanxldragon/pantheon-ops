import React from 'react';
import { Button, Space } from '@arco-design/web-react';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import {
  AppTable,
  buildStandardPagination,
  PageEmpty,
  PageLoading,
  PageRequestError,
} from '../../../../components';

/**
 * The table shell both tenant-ops list pages render. Only components live in
 * this file; the shared cells are in ./tenantListCells.
 */

interface TenantListTableProps<T> {
  rows: T[];
  columns: ColumnProps<T>[];
  rowKey: string | ((row: T) => string);
  loading: boolean;
  page: number;
  pageSize: number;
  total: number;
  emptyText: string;
  onPageChange: (page: number, pageSize: number) => void;
}

/** The table, its pagination and its empty state - the shape both pages render. */
export function TenantListTable<T>({
  rows,
  columns,
  rowKey,
  loading,
  page,
  pageSize,
  total,
  emptyText,
  onPageChange,
}: Readonly<TenantListTableProps<T>>): React.ReactElement {
  const { t } = useTranslation();

  if (rows.length === 0 && !loading) {
    return <PageEmpty description={emptyText} />;
  }

  const handleTableChange: TableProps<T>['onChange'] = (pagination) => {
    onPageChange(pagination.current ?? 1, pagination.pageSize ?? pageSize);
  };

  return (
    <AppTable<T>
      className="system-list__table"
      rowKey={rowKey}
      data={rows}
      columns={columns}
      loading={loading}
      scroll={{ x: 'max-content' }}
      emptyText={emptyText}
      onChange={handleTableChange}
      pagination={buildStandardPagination(t, {
        current: page,
        pageSize,
        total,
        onChange: onPageChange,
      })}
    />
  );
}

interface TenantListBodyProps {
  tenantModeEnabled: boolean;
  loading: boolean;
  rowCount: number;
  error: Error | null;
  onRetry: () => void;
  onOpenPlatformSettings: () => void;
  table: React.ReactNode;
}

/**
 * Which of the four states the panel shows. An installation running in compat
 * mode has no cross-tenant data to list, so it gets the pointer to the platform
 * setting instead of an empty table.
 */
export const TenantListBody: React.FC<TenantListBodyProps> = ({
  tenantModeEnabled,
  loading,
  rowCount,
  error,
  onRetry,
  onOpenPlatformSettings,
  table,
}) => {
  const { t } = useTranslation();

  if (!tenantModeEnabled) {
    return (
      <Space direction="vertical" size={16}>
        <PageEmpty description={t('system.tenant.compat.desc')} />
        <Button type="primary" onClick={onOpenPlatformSettings}>
          {t('system.tenant.compat.action')}
        </Button>
      </Space>
    );
  }
  if (loading && rowCount === 0) {
    return <PageLoading />;
  }
  if (error && rowCount === 0) {
    return <PageRequestError error={error} onRetry={onRetry} />;
  }
  return <>{table}</>;
};
