export interface ArcoFormValidationErrorLike {
  errors?: unknown;
}

export function isArcoFormValidationError(error: unknown): error is ArcoFormValidationErrorLike {
  return typeof error === 'object' && error !== null && 'errors' in error;
}

export function isLikelyEmailAddress(value: string): boolean {
  const normalized = String(value || '').trim();
  const atIndex = normalized.indexOf('@');
  if (atIndex <= 0 || atIndex !== normalized.lastIndexOf('@') || atIndex === normalized.length - 1) {
    return false;
  }

  const localPart = normalized.slice(0, atIndex);
  const domainPart = normalized.slice(atIndex + 1);
  if (!localPart || !domainPart || domainPart.startsWith('.') || domainPart.endsWith('.')) {
    return false;
  }

  const domainLabels = domainPart.split('.');
  if (domainLabels.length < 2 || domainLabels.some((label) => label.trim().length === 0)) {
    return false;
  }

  return true;
}
