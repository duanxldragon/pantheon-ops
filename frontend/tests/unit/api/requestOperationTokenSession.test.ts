import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { AxiosAdapter, InternalAxiosRequestConfig } from 'axios';
import axios from 'axios';
import request, { clearOperationToken } from '../../../src/api/request';

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
 * The secondary-verification token is issued for one session: the backend
 * compares the token's session claim against the cookie that comes with it. It
 * lives in sessionStorage (per tab) while the session it belongs to ends at
 * logout, so the logout chain has to drop the key too — otherwise the next
 * subject's first sensitive request carries a foreign token and the backend
 * refuses it with `auth.operation.verification_mismatch`.
 */
const OPERATION_TOKEN_STORAGE_KEY = 'pantheon_op_token';
const STALE_TOKEN = 'token-of-the-previous-subject';
const PROTECTED_READ_URL = '/system/user/list';

let sentConfigs: InternalAxiosRequestConfig[] = [];
let unauthorizedReadsRemaining = 0;

const originalRequestAdapter = request.defaults.adapter;
const originalAxiosAdapter = axios.defaults.adapter;

/**
 * The real chain is exercised instead of stubbing internals: every response
 * comes from a swapped adapter, so the actual request interceptor (which
 * attaches the token) and the actual response interceptor (which runs
 * clearClientSession) both execute.
 */
const clientAdapter: AxiosAdapter = async (config) => {
  sentConfigs.push(config);
  let expiresThisRead = false;
  if (config.url === PROTECTED_READ_URL && unauthorizedReadsRemaining > 0) {
    // Only the configured number of reads is refused; refusing the retry as
    // well would send the chain back through refresh forever.
    unauthorizedReadsRemaining -= 1;
    expiresThisRead = true;
  }
  return {
    data: expiresThisRead
      ? { code: 401, message: 'auth.token_expired' }
      : { code: 200, data: { rows: [] } },
    status: 200,
    statusText: 'OK',
    headers: {},
    config,
  };
};

// The refresh endpoint is called on the bare axios instance, so it needs its
// own adapter. Failing it is what triggers the client-side logout chain.
const failingRefreshAdapter: AxiosAdapter = async (config) => {
  sentConfigs.push(config as InternalAxiosRequestConfig);
  throw new Error('refresh session unavailable');
};

const headerOf = (name: string) =>
  (sentConfigs[0]?.headers as Record<string, unknown> | undefined)?.[name];

beforeEach(() => {
  sentConfigs = [];
  unauthorizedReadsRemaining = 0;
  errorMock.mockClear();
  sessionStorage.clear();
  localStorage.clear();
  sessionStorage.setItem(
    OPERATION_TOKEN_STORAGE_KEY,
    JSON.stringify({
      token: STALE_TOKEN,
      acquiredAt: Date.now(),
      expiresAt: Date.now() + 5 * 60 * 1000,
    }),
  );
});

afterEach(() => {
  request.defaults.adapter = originalRequestAdapter;
  axios.defaults.adapter = originalAxiosAdapter;
  sessionStorage.clear();
});

describe('operation token client-session boundary', () => {
  it('attaches the stored operation token while the session is alive', async () => {
    request.defaults.adapter = clientAdapter;

    await request.get(PROTECTED_READ_URL);

    expect(headerOf('X-Operation-Token')).toBe(STALE_TOKEN);
  });

  it('drops the operation token when the session ends in a failed refresh', async () => {
    unauthorizedReadsRemaining = 1;
    request.defaults.adapter = clientAdapter;
    axios.defaults.adapter = failingRefreshAdapter;

    // The chain ends in redirectToLogin, which assigns location.href; jsdom
    // cannot navigate, so it reports the attempt while the test continues.
    await expect(request.get(PROTECTED_READ_URL)).rejects.toThrow();
    // Let the fire-and-forget /auth/logout dispatched by the chain settle.
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(sessionStorage.getItem(OPERATION_TOKEN_STORAGE_KEY)).toBeNull();
    const logoutRequest = sentConfigs.find((config) => config.url?.includes('/auth/logout'));
    expect(logoutRequest).toBeDefined();
    // /auth/logout is not a secure-action route, so the token can be dropped
    // before it goes out without tripping the verification retry.
    expect(
      (logoutRequest?.headers as Record<string, unknown>)?.['X-Operation-Token'],
    ).toBeUndefined();
  });

  it('stops sending a stale token on later requests once it was cleared', async () => {
    clearOperationToken();

    request.defaults.adapter = clientAdapter;
    await request.get(PROTECTED_READ_URL);

    expect(headerOf('X-Operation-Token')).toBeUndefined();
  });
});
