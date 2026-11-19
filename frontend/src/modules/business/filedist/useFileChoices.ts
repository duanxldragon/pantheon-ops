import { useCallback, useState } from 'react';
import { useCatalogOptions } from '../service/useCatalogOptions';
import { getFileList, type FileListResponse } from './api';

export const FILE_CHOICE_PAGE_SIZE = 20;

export function useFileChoices(enabled: boolean) {
  const [query, setQuery] = useState({ keyword: '', page: 1, revision: 0 });
  const lookup = useCallback(
    async (signal: AbortSignal): Promise<FileListResponse[]> => {
      const response = await getFileList(
        { keyword: query.keyword, page: query.page, pageSize: FILE_CHOICE_PAGE_SIZE },
        signal,
      );
      // Keep the page's items and total in one activation-bound result. Reusing
      // this helper also clears old pages immediately before its 250ms debounce.
      return [response];
    },
    [query],
  );
  const { rows, loading, error } = useCatalogOptions(enabled, lookup);
  const search = useCallback(
    (keyword: string) => setQuery((previous) => ({ ...previous, keyword, page: 1 })),
    [],
  );
  const setPage = useCallback(
    (page: number) => setQuery((previous) => ({ ...previous, page })),
    [],
  );
  const reload = useCallback(
    () => setQuery((previous) => ({ ...previous, revision: previous.revision + 1 })),
    [],
  );
  return {
    items: rows[0]?.items || [],
    total: rows[0]?.total || 0,
    keyword: query.keyword,
    page: query.page,
    loading,
    error,
    search,
    setPage,
    reload,
  };
}
