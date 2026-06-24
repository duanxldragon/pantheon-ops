import type { PaginationProps } from '@arco-design/web-react/es/Pagination/interface';
import React from 'react';

const DEFAULT_SIZE_OPTIONS = [10, 20, 50, 100];

type TranslateFn = (key: string, options?: Record<string, unknown>) => string;

interface StandardPaginationOptions {
  current?: number;
  pageSize?: number;
  total?: number;
  onChange?: PaginationProps['onChange'];
  sizeOptions?: number[];
  sizeCanChange?: boolean;
  pageSizeChangeResetCurrent?: boolean;
  size?: PaginationProps['size'];
  showTotal?: PaginationProps['showTotal'];
}

export function buildStandardPagination(
  t: TranslateFn,
  options: StandardPaginationOptions,
): PaginationProps {
  const {
    current = 1,
    pageSize = 10,
    total = 0,
    onChange,
    sizeOptions = DEFAULT_SIZE_OPTIONS,
    sizeCanChange = true,
    pageSizeChangeResetCurrent = false,
    size = 'small',
    showTotal = (count: number) =>
      React.createElement(
        'span',
        { className: 'app-table__pagination-total-text' },
        React.createElement('span', null, t('common.total')),
        React.createElement('strong', null, count.toLocaleString()),
      ),
  } = options;

  return {
    current,
    pageSize,
    total,
    onChange,
    showJumper: true,
    pageSizeChangeResetCurrent,
    sizeCanChange,
    sizeOptions,
    size,
    showTotal,
  };
}

interface PagedItemsResult<T> {
  currentPage: number;
  totalPages: number;
  pageItems: T[];
}

export function getPagedItems<T>(items: T[], current: number, pageSize: number): PagedItemsResult<T> {
  const safePageSize = Math.max(1, pageSize);
  const totalPages = Math.max(1, Math.ceil(items.length / safePageSize));
  const currentPage = Math.min(Math.max(1, current), totalPages);
  const startIndex = (currentPage - 1) * safePageSize;
  return {
    currentPage,
    totalPages,
    pageItems: items.slice(startIndex, startIndex + safePageSize),
  };
}

export { DEFAULT_SIZE_OPTIONS as STANDARD_PAGINATION_SIZE_OPTIONS };
