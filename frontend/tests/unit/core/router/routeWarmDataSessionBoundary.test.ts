import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  invalidateAllRouteWarmData,
  resolveRouteWarmData,
} from '../../../../src/core/router/prefetch';

/**
 * Warm data is keyed by `path::resourceKey` only — the key carries no subject,
 * tenant or session identity — and logout is an SPA navigation that keeps the
 * document (and therefore this module-level Map) alive. Without the boundary
 * clear, the first default query of the next sign-in in the same tab resolves
 * against the previous subject's rows inside the 30s TTL.
 */
const USER_LIST_PATH = '/system/user';
const USER_LIST_KEY = 'list:default';
const DEPT_TREE_PATH = '/system/dept';
const DEPT_TREE_KEY = 'tree:default';

describe('route warm-data cache session boundary', () => {
  beforeEach(() => {
    invalidateAllRouteWarmData();
  });

  it('serves a warmed entry to the next caller without re-reading', async () => {
    const loader = vi.fn(async () => ({ rows: ['alice'] }));

    const first = await resolveRouteWarmData(USER_LIST_PATH, USER_LIST_KEY, loader);
    const second = await resolveRouteWarmData(USER_LIST_PATH, USER_LIST_KEY, loader);

    expect(second).toBe(first);
    expect(loader).toHaveBeenCalledTimes(1);
  });

  it('invalidateAllRouteWarmData empties the cache for every path', async () => {
    await resolveRouteWarmData(USER_LIST_PATH, USER_LIST_KEY, async () => 'account-a-rows');
    await resolveRouteWarmData(DEPT_TREE_PATH, DEPT_TREE_KEY, async () => 'account-a-tree');

    invalidateAllRouteWarmData();

    const userListLoader = vi.fn(async () => 'account-b-rows');
    const deptTreeLoader = vi.fn(async () => 'account-b-tree');
    await expect(resolveRouteWarmData(USER_LIST_PATH, USER_LIST_KEY, userListLoader)).resolves.toBe(
      'account-b-rows',
    );
    await expect(resolveRouteWarmData(DEPT_TREE_PATH, DEPT_TREE_KEY, deptTreeLoader)).resolves.toBe(
      'account-b-tree',
    );
    expect(userListLoader).toHaveBeenCalledTimes(1);
    expect(deptTreeLoader).toHaveBeenCalledTimes(1);
  });

  it('drops an in-flight entry so a previous subject loader cannot land back in the cache', async () => {
    let releaseAccountALoader: (value: string) => void = () => undefined;
    const accountALoader = vi.fn(
      () =>
        new Promise<string>((resolve) => {
          releaseAccountALoader = resolve;
        }),
    );

    const inflight = resolveRouteWarmData(USER_LIST_PATH, USER_LIST_KEY, accountALoader);
    invalidateAllRouteWarmData();
    releaseAccountALoader('account-a-rows');
    await expect(inflight).resolves.toBe('account-a-rows');

    const accountBLoader = vi.fn(async () => 'account-b-rows');
    await expect(resolveRouteWarmData(USER_LIST_PATH, USER_LIST_KEY, accountBLoader)).resolves.toBe(
      'account-b-rows',
    );
    expect(accountBLoader).toHaveBeenCalledTimes(1);
  });

  it('never serves the previous subject rows across a logout/login in the same tab', async () => {
    // Account A warms its default list while authenticated.
    const accountALoader = vi.fn(async () => ({ rows: [{ id: 1, username: 'alice' }] }));
    await expect(
      resolveRouteWarmData(USER_LIST_PATH, USER_LIST_KEY, accountALoader),
    ).resolves.toEqual({ rows: [{ id: 1, username: 'alice' }] });

    // AppLayout.performLogout: the session is torn down and the shell merely
    // navigates to /login, so this is the only thing that can drop A's rows.
    invalidateAllRouteWarmData();
    // Login.completeLogin clears again on the far side of the boundary, which
    // also covers exits that never reached the logout path.
    invalidateAllRouteWarmData();

    // Account B's first default query must go to the wire for B's own rows.
    const accountBLoader = vi.fn(async () => ({ rows: [{ id: 2, username: 'bob' }] }));
    await expect(
      resolveRouteWarmData(USER_LIST_PATH, USER_LIST_KEY, accountBLoader),
    ).resolves.toEqual({ rows: [{ id: 2, username: 'bob' }] });
    expect(accountBLoader).toHaveBeenCalledTimes(1);
  });
});
