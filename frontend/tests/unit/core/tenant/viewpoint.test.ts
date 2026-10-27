import { afterEach, describe, expect, it } from 'vitest';
import {
  PLATFORM_VIEWPOINT_TENANT_ID,
  businessTenantIdForHeader,
  clearTenantViewpoint,
  isTenantViewpointActive,
  readTenantViewpointId,
  writeTenantViewpointId,
} from '../../../../src/core/tenant/viewpoint';

const VIEWPOINT_KEY = 'pantheon_tenant_viewpoint';

afterEach(() => {
  sessionStorage.clear();
});

describe('tenant viewpoint storage', () => {
  it('reports viewpoint-off when nothing is stored', () => {
    expect(readTenantViewpointId()).toBeNull();
    expect(isTenantViewpointActive()).toBe(false);
    expect(businessTenantIdForHeader()).toBeNull();
  });

  it('round-trips a pinned tenant id', () => {
    writeTenantViewpointId(42);

    expect(sessionStorage.getItem(VIEWPOINT_KEY)).toBe('42');
    expect(readTenantViewpointId()).toBe(42);
    expect(isTenantViewpointActive()).toBe(true);
    expect(businessTenantIdForHeader()).toBe(42);
  });

  it('round-trips the platform-global viewpoint', () => {
    writeTenantViewpointId(PLATFORM_VIEWPOINT_TENANT_ID);

    expect(readTenantViewpointId()).toBe(PLATFORM_VIEWPOINT_TENANT_ID);
    // Platform scope is active but must not pin a tenant on the wire: the
    // backend already resolves a platform subject to tenant 0.
    expect(isTenantViewpointActive()).toBe(true);
    expect(businessTenantIdForHeader()).toBeNull();
  });

  it('clears the stored viewpoint', () => {
    writeTenantViewpointId(7);
    clearTenantViewpoint();

    expect(sessionStorage.getItem(VIEWPOINT_KEY)).toBeNull();
    expect(readTenantViewpointId()).toBeNull();
    expect(businessTenantIdForHeader()).toBeNull();
  });

  it.each(['abc', '', '  ', '-3', '1.5', 'Infinity', 'NaN', '{}'])(
    'treats the corrupt stored value %j as viewpoint-off and drops it',
    (raw) => {
      sessionStorage.setItem(VIEWPOINT_KEY, raw);

      expect(readTenantViewpointId()).toBeNull();
      expect(sessionStorage.getItem(VIEWPOINT_KEY)).toBeNull();
      expect(businessTenantIdForHeader()).toBeNull();
    },
  );

  it('never derives a header value from the platform viewpoint', () => {
    sessionStorage.setItem(VIEWPOINT_KEY, '0');

    expect(businessTenantIdForHeader()).toBeNull();
  });
});
