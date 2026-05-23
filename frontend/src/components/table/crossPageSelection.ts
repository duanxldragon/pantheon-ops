export type CrossPageRowKey = string | number;
export type SharedPaginationConfig = {
  current?: number;
  defaultCurrent?: number;
  pageSize?: number;
  defaultPageSize?: number;
  total?: number;
};

const DEFAULT_PAGE_SIZE = 10;
const DEFAULT_CURRENT_PAGE = 1;

function normalizeRowKey(key: CrossPageRowKey) {
  return String(key);
}

export function isPaginationConfig(
  pagination: SharedPaginationConfig | boolean | undefined,
): pagination is SharedPaginationConfig {
  return Boolean(pagination) && typeof pagination === 'object';
}

export function getPaginationCurrentPage(pagination: SharedPaginationConfig) {
  return pagination.current ?? pagination.defaultCurrent ?? DEFAULT_CURRENT_PAGE;
}

export function getPaginationPageSize(pagination: SharedPaginationConfig) {
  return pagination.pageSize ?? pagination.defaultPageSize ?? DEFAULT_PAGE_SIZE;
}

export function getPaginationTotalPages(pagination: SharedPaginationConfig) {
  const pageSize = Math.max(1, getPaginationPageSize(pagination));
  const total = Math.max(0, pagination.total ?? 0);
  return Math.max(1, Math.ceil(total / pageSize));
}

export function getVisibleSelectedRowKeys(
  selectedRowKeys: CrossPageRowKey[],
  visibleRowKeys: CrossPageRowKey[],
) {
  const visibleKeySet = new Set(visibleRowKeys.map(normalizeRowKey));
  return selectedRowKeys.filter((key) => visibleKeySet.has(normalizeRowKey(key)));
}

export function mergeCrossPageSelection(
  selectedRowKeys: CrossPageRowKey[],
  nextVisibleSelectedRowKeys: CrossPageRowKey[],
  visibleRowKeys: CrossPageRowKey[],
) {
  const visibleKeySet = new Set(visibleRowKeys.map(normalizeRowKey));
  const hiddenSelectedRowKeys = selectedRowKeys.filter(
    (key) => !visibleKeySet.has(normalizeRowKey(key)),
  );
  const mergedRowKeys = [...hiddenSelectedRowKeys, ...nextVisibleSelectedRowKeys];
  const seen = new Set<string>();

  return mergedRowKeys.filter((key) => {
    const normalizedKey = normalizeRowKey(key);
    if (seen.has(normalizedKey)) {
      return false;
    }
    seen.add(normalizedKey);
    return true;
  });
}
