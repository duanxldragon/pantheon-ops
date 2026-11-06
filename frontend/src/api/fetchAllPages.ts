/**
 * Fetches every row of a paginated list endpoint.
 *
 * The backend caps pageSize (`NormalizePagination(…, 100)` in
 * backend/pkg/common/handler.go), so a single large page silently truncates
 * once a module exceeds the cap. Pages are aggregated at a fixed `pageSize`
 * instead, which is the only way to get the full set without changing the API
 * contract.
 *
 * Termination is dual: a short page (`items.length < pageSize`, which also
 * covers `total === 0`) or aggregated rows reaching `total`. `maxPages` is a
 * guard so a broken/missing `total` can never loop forever — it throws rather
 * than returning a silently truncated set.
 */
export async function fetchAllPages<T>(
  fetchPage: (page: number, pageSize: number) => Promise<{ items: T[]; total: number }>,
  pageSize = 100,
  maxPages = 200,
): Promise<T[]> {
  const all: T[] = [];
  for (let page = 1; page <= maxPages; page += 1) {
    const result = await fetchPage(page, pageSize);
    all.push(...result.items);
    if (result.items.length < pageSize || all.length >= result.total) {
      return all;
    }
  }
  throw new Error(
    `fetchAllPages: reached ${maxPages} pages without collecting the full set (total still pending)`,
  );
}