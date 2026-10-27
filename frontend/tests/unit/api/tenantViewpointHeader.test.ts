import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import type { InternalAxiosRequestConfig } from 'axios';
import request from '../../../src/api/request';
import {
  PLATFORM_VIEWPOINT_TENANT_ID,
  clearTenantViewpoint,
  writeTenantViewpointId,
} from '../../../src/core/tenant/viewpoint';

/**
 * The request interceptor is the only place the tenant viewpoint reaches the
 * wire (contract §3.1 source 1). Instead of stubbing axios we swap the adapter
 * so every assertion runs through the real interceptor chain, and we capture
 * the config that would have been sent.
 */
let captured: InternalAxiosRequestConfig | null = null;

beforeEach(() => {
  captured = null;
  sessionStorage.clear();
  localStorage.clear();
  request.defaults.adapter = async (config) => {
    captured = config as InternalAxiosRequestConfig;
    return {
      data: { code: 200, data: { ok: true } },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
});

afterEach(() => {
  clearTenantViewpoint();
  delete request.defaults.adapter;
});

const sentTenantHeader = () =>
  (captured?.headers as Record<string, unknown> | undefined)?.['X-Tenant-Id'];

describe('request interceptor tenant viewpoint header', () => {
  it('omits X-Tenant-Id when no viewpoint is pinned', async () => {
    await request.get('/system/user/list');

    expect(sentTenantHeader()).toBeUndefined();
  });

  it('omits X-Tenant-Id for the platform-global viewpoint', async () => {
    writeTenantViewpointId(PLATFORM_VIEWPOINT_TENANT_ID);

    await request.get('/system/user/list');

    // Platform scope resolves server-side to tenant 0; the header would only
    // add a value the backend has to ignore.
    expect(sentTenantHeader()).toBeUndefined();
  });

  it('attaches the pinned tenant to business requests', async () => {
    writeTenantViewpointId(42);

    await request.get('/system/user/list');

    expect(sentTenantHeader()).toBe('42');
  });

  it('attaches the pinned tenant to writes as well as reads', async () => {
    writeTenantViewpointId(42);

    await request.post('/system/user', { username: 'sam' });

    expect(sentTenantHeader()).toBe('42');
  });

  it('never attaches the viewpoint to auth endpoints', async () => {
    writeTenantViewpointId(42);

    await request.post('/auth/login', { username: 'sam' });

    // Auth flows resolve tenancy from the membership gate, not from a header.
    expect(sentTenantHeader()).toBeUndefined();
  });

  it('drops the header again once the viewpoint is cleared', async () => {
    writeTenantViewpointId(42);
    await request.get('/system/user/list');
    expect(sentTenantHeader()).toBe('42');

    clearTenantViewpoint();
    await request.get('/system/user/list');

    expect(sentTenantHeader()).toBeUndefined();
  });
});
