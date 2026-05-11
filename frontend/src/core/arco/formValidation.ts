export interface ArcoFormValidationErrorLike {
  errors?: unknown;
}

export function isArcoFormValidationError(error: unknown): error is ArcoFormValidationErrorLike {
  return typeof error === 'object' && error !== null && 'errors' in error;
}
