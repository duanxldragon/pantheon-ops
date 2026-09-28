import { useCallback, useMemo, useState } from 'react';
import type { PaginationProps } from '@arco-design/web-react/es/Pagination/interface';

export interface UsePaginationOptions {
  defaultCurrent?: number;
  defaultPageSize?: number;
  total?: number;
}

export interface UsePaginationResult {
  current: number;
  pageSize: number;
  total: number;
  paginationProps: PaginationProps;
  setCurrent: (page: number) => void;
  setPageSize: (size: number) => void;
  reset: () => void;
}

export function usePagination(options: UsePaginationOptions = {}): UsePaginationResult {
  const { defaultCurrent = 1, defaultPageSize = 10, total = 0 } = options;

  const [current, setCurrent] = useState(defaultCurrent);
  const [pageSize, setPageSize] = useState(defaultPageSize);

  const changePageSize = useCallback((size: number) => {
    setPageSize(size);
    setCurrent(1);
  }, []);

  const reset = useCallback(() => {
    setCurrent(defaultCurrent);
    setPageSize(defaultPageSize);
  }, [defaultCurrent, defaultPageSize]);

  const paginationProps = useMemo<PaginationProps>(
    () => ({
      current,
      pageSize,
      total,
      onChange: (page) => setCurrent(page),
      showJumper: true,
      sizeCanChange: true,
      sizeOptions: [10, 20, 50, 100],
      size: 'small' as const,
    }),
    [current, pageSize, total],
  );

  return {
    current,
    pageSize,
    total,
    paginationProps,
    setCurrent,
    setPageSize: changePageSize,
    reset,
  };
}
