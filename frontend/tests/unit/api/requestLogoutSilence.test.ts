import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { AxiosAdapter, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import axios from 'axios';
import request from '../../../src/api/request';
import { logout } from '../../../src/modules/auth/session/api';

const { errorMock } = vi.hoisted(() => ({ errorMock: vi.fn() }));

vi.mock('../../../src/components/feedback/message', () => ({
  message: {
    success: vi.fn(),
    error: errorMock,
    warning: vi.fn(),
    info: vi.fn(),
  },
}));

/**
 * An idle-timeout logout has no live session, so the backend's CSRF check
 * refuses `POST /auth/logout` with `csrf.missing`. Both logout dispatches are
 * best effort — AppLayout awaits them through resolveSilently and then tears
 * down local state anyway — so neither may reach the user as a failed action.
 * A refused response still travels over HTTP 200 with the business code in the
 * envelope (pkg/common/http.FailWithData), which is what the adapters below
 * return so the interceptor runs the real branch.
 */
const PROTECTED_READ_URL = '/system/user/list';
const EXPIRED_READ_ENVELOPE = { code: 401, message: 'auth.token_expired' };
const CSRF_ENVELOPE = { code: 403, message: 'csrf.missing' };

let sentConfigs: InternalAxiosRequestConfig[] = [];

const originalRequestAdapter = request.defaults.adapter;
const originalAxiosAdapter = axios.defaults.adapter;

const envelope = (
  config: InternalAxiosRequestConfig,
  body: { code: number; message?: string; data?: unknown },
): AxiosResponse => ({
  data: body,
  status: 200,
  statusText: 'OK',
  headers: {},
  config,
});

const adapterFor =
  (handler: (config: InternalAxiosRequestConfig) => AxiosResponse): AxiosAdapter =>
  async (config) => {
    sentConfigs.push(config);
    return handler(config);
  };

const refusesEveryRequest = (config: InternalAxiosRequestConfig) => envelope(config, CSRF_ENVELOPE);

// Only the logout dispatch is refused; the read reports an expired session,
// which is what drives the client-side teardown chain.
const refusesLogoutAfterAnExpiredRead = (config: InternalAxiosRequestConfig) =>
  envelope(config, config.url?.includes('/auth/logout') ? CSRF_ENVELOPE : EXPIRED_READ_ENVELOPE);

const logoutDispatches = () => sentConfigs.filter((config) => config.url?.includes('/auth/logout'));

beforeEach(() => {
  sentConfigs = [];
  errorMock.mockClear();
  sessionStorage.clear();
  localStorage.clear();
});

afterEach(() => {
  request.defaults.adapter = originalRequestAdapter;
  axios.defaults.adapter = originalAxiosAdapter;
  sessionStorage.clear();
  localStorage.clear();
});

describe('logout stays silent when the session is already gone', () => {
  it('drops the CSRF toast for the logout() wrapper', async () => {
    request.defaults.adapter = adapterFor(refusesEveryRequest);

    await expect(logout()).rejects.toThrow('csrf.missing');

    expect(errorMock).not.toHaveBeenCalled();
  });

  it('still toasts a csrf.missing refusal from a route the user is waiting on', async () => {
    // Guards the case above: the silence comes from the per-request flag, not
    // from an environment that never shows messages. Same response, same
    // interceptor — one extra toast only because this caller waits for it.
    request.defaults.adapter = adapterFor(refusesEveryRequest);

    await expect(request.get(PROTECTED_READ_URL)).rejects.toThrow();

    expect(errorMock).toHaveBeenCalledTimes(1);
  });

  it('drops the CSRF toast from the session-teardown logout dispatch', async () => {
    // clearClientSession() fires its own best-effort /auth/logout POST, so the
    // idle-logout path used to toast twice.
    request.defaults.adapter = adapterFor(refusesLogoutAfterAnExpiredRead);
    axios.defaults.adapter = async () => {
      throw new Error('refresh session unavailable');
    };

    await expect(request.get(PROTECTED_READ_URL)).rejects.toThrow();
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(logoutDispatches().length).toBeGreaterThan(0);
    // Only the read's own expired-session message is left.
    expect(errorMock).toHaveBeenCalledTimes(1);
  });
});
