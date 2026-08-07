import type { Page } from '@playwright/test';

export type CachedReadResponse = {
  status: number;
  contentType: string;
  body: string;
  headers?: Record<string, string>;
};

type SharedReadCacheOptions = {
  routePattern?: string;
  shouldHandleRequest?: (request: { method: string; url: URL }) => boolean;
};

export function buildSharedReadCacheKey(url: string) {
  const parsed = new URL(url);
  const nextParams = new URLSearchParams();
  Array.from(parsed.searchParams.entries())
    .sort(([leftKey, leftValue], [rightKey, rightValue]) => {
      if (leftKey === rightKey) {
        return leftValue.localeCompare(rightValue);
      }
      return leftKey.localeCompare(rightKey);
    })
    .forEach(([key, value]) => nextParams.append(key, value));
  const search = nextParams.toString();
  return search ? `${parsed.pathname}?${search}` : parsed.pathname;
}

export function shouldCacheSharedReadResponse(method: string, status: number) {
  return method === 'GET' && status >= 200 && status < 300;
}

export async function installSharedPageReadCache(
  page: Page,
  cache: Map<string, CachedReadResponse> = new Map<string, CachedReadResponse>(),
  options: SharedReadCacheOptions = {},
) {
  const routePattern = options.routePattern ?? '**/api/v1/**';
  const shouldHandleRequest = options.shouldHandleRequest
    ?? ((request: { method: string; url: URL }) => request.method === 'GET');

  await page.route(routePattern, async (route) => {
    const request = route.request();
    const requestUrl = new URL(request.url());
    if (!shouldHandleRequest({ method: request.method(), url: requestUrl })) {
      await route.fallback();
      return;
    }

    const cacheKey = buildSharedReadCacheKey(requestUrl.toString());
    const cached = cache.get(cacheKey);
    if (cached) {
      await route.fulfill(cached);
      return;
    }

    const response = await route.fetch();
    const body = await response.text();
    const csrfToken = response.headers()['x-csrf-token'];
    const cachedResponse: CachedReadResponse = {
      status: response.status(),
      contentType: response.headers()['content-type'] || 'application/json',
      body,
      headers: csrfToken ? { 'x-csrf-token': csrfToken } : undefined,
    };

    if (shouldCacheSharedReadResponse(request.method(), response.status())) {
      cache.set(cacheKey, cachedResponse);
    }

    await route.fulfill(cachedResponse);
  });

  return cache;
}
