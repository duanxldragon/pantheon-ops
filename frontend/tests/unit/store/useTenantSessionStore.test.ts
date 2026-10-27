import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getCurrentTenant } = vi.hoisted(() => ({ getCurrentTenant: vi.fn() }));

vi.mock('../../../src/modules/auth/tenant/api', () => ({ getCurrentTenant }));

import {
  isTenantScopedSession,
  useTenantSessionStore,
} from '../../../src/store/useTenantSessionStore';

const freeTenant = {
  mode: 'multi' as const,
  tenantId: 300,
  resolvedBy: 'subject',
  tenant: { id: 300, code: 'free', name: 'Free', status: 'active', plan: 'free', role: 'owner', isActive: true },
};

describe('tenant session store', () => {
  beforeEach(() => {
    getCurrentTenant.mockReset();
    useTenantSessionStore.getState().resetTenantSession();
  });

  it('loads the tenant context once and reuses it for later consumers', async () => {
    getCurrentTenant.mockResolvedValue(freeTenant);

    const [first, second] = await Promise.all([
      useTenantSessionStore.getState().fetchTenantSession(),
      useTenantSessionStore.getState().fetchTenantSession(),
    ]);

    expect(first).toEqual(freeTenant);
    expect(second).toEqual(freeTenant);
    expect(getCurrentTenant).toHaveBeenCalledTimes(1);
    expect(useTenantSessionStore.getState().loaded).toBe(true);
  });

  it('drops the previous session context on reset and reloads for the next login', async () => {
    getCurrentTenant.mockResolvedValueOnce(freeTenant);
    await useTenantSessionStore.getState().fetchTenantSession();

    useTenantSessionStore.getState().resetTenantSession();
    expect(useTenantSessionStore.getState().context).toBeNull();
    expect(useTenantSessionStore.getState().loaded).toBe(false);

    const enterpriseTenant = {
      ...freeTenant,
      tenantId: 303,
      tenant: { ...freeTenant.tenant, id: 303, plan: 'enterprise' },
    };
    getCurrentTenant.mockResolvedValueOnce(enterpriseTenant);
    await useTenantSessionStore.getState().fetchTenantSession();

    expect(getCurrentTenant).toHaveBeenCalledTimes(2);
    expect(useTenantSessionStore.getState().context?.tenant?.plan).toBe('enterprise');
  });

  it('treats only multi-mode tenant ids as tenant-scoped sessions', () => {
    expect(isTenantScopedSession(freeTenant)).toBe(true);
    expect(isTenantScopedSession({ ...freeTenant, tenantId: 0 })).toBe(false);
    expect(isTenantScopedSession({ ...freeTenant, mode: 'compat' })).toBe(false);
    expect(isTenantScopedSession(null)).toBe(false);
  });
});
