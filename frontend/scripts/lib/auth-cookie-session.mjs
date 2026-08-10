function getHeaderValue(headers, name) {
  if (!headers) {
    return null;
  }
  if (typeof headers.get === 'function') {
    return headers.get(name);
  }
  const target = name.toLowerCase();
  for (const [key, value] of Object.entries(headers)) {
    if (key.toLowerCase() === target) {
      return value ?? null;
    }
  }
  return null;
}

export function extractCookieValue(setCookieHeader, name) {
  if (!setCookieHeader) {
    return null;
  }
  const normalizedHeader = String(setCookieHeader).replace(/\r?\n/g, ', ');
  const match = normalizedHeader.match(new RegExp(`(?:^|,\\s*)${name}=([^;]+)`));
  return match?.[1] ?? null;
}

export function readAuthCookieSession(headers, options = {}) {
  const {
    includeRefreshToken = true,
    csrfFallback = `pantheon-smoke-csrf-${Date.now()}`,
  } = options;
  const setCookieHeader = getHeaderValue(headers, 'set-cookie');
  const accessToken = extractCookieValue(setCookieHeader, 'pantheon_access_token');
  if (!accessToken) {
    throw new Error('Auth response did not include pantheon_access_token cookie');
  }
  const csrfToken =
    getHeaderValue(headers, 'x-csrf-token') ??
    extractCookieValue(setCookieHeader, 'pantheon_csrf_token') ??
    csrfFallback;

  const session = {
    accessToken,
    csrfToken,
  };

  if (includeRefreshToken) {
    const refreshToken = extractCookieValue(setCookieHeader, 'pantheon_refresh_token');
    if (!refreshToken) {
      throw new Error('Auth response did not include pantheon_refresh_token cookie');
    }
    session.refreshToken = refreshToken;
  }

  return session;
}
