export type RequestErrorKind =
  'business' | 'unauthorized' | 'forbidden' | 'server' | 'network' | 'timeout' | 'unknown';

interface TransportErrorLike {
  code?: string;
  message?: string;
  response?: {
    status?: number;
  };
}

export function resolveBusinessErrorKind(code?: number): RequestErrorKind {
  if (code === 401) {
    return 'unauthorized';
  }
  if (code === 403) {
    return 'forbidden';
  }
  if ((code || 0) >= 500) {
    return 'server';
  }
  return 'business';
}

export function resolveTransportErrorKind(error: TransportErrorLike): RequestErrorKind {
  if (error.code === 'ECONNABORTED' || error.message?.toLowerCase().includes('timeout')) {
    return 'timeout';
  }

  const status = error.response?.status;
  if (status === undefined) {
    return 'network';
  }
  if (status === 401) {
    return 'unauthorized';
  }
  if (status === 403) {
    return 'forbidden';
  }
  if (status >= 500) {
    return 'server';
  }
  return 'business';
}

export function resolveFallbackMessageKey(
  kind: RequestErrorKind,
  explicitFallbackKey?: string,
): string {
  if (explicitFallbackKey) {
    return explicitFallbackKey;
  }
  switch (kind) {
    case 'timeout':
      return 'network.timeout';
    case 'network':
      return 'network.error';
    default:
      return 'request.failed';
  }
}

export function shouldSuppressAuthMessage(
  messageKey: string | undefined,
  kind: RequestErrorKind,
  pathname: string,
  logoutTransition: boolean,
): boolean {
  const isTokenMessage = messageKey?.startsWith('token.') ?? false;
  const isAuthError =
    kind === 'unauthorized' ||
    messageKey === 'session.invalid' ||
    messageKey === 'session.idle_timeout' ||
    isTokenMessage;
  if (isAuthError) {
    return logoutTransition || pathname === '/login';
  }
  return false;
}
