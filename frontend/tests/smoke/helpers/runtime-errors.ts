import { expect } from '@playwright/test';

const knownThirdPartyRuntimeErrorPatterns = [
  /^CopyReactDOM\.render is not a function$/,
];

export function expectOnlyAllowedRuntimeErrors(runtimeErrors: string[], allowedPatterns: RegExp[] = []) {
  const allowlist = [...knownThirdPartyRuntimeErrorPatterns, ...allowedPatterns];
  const unexpectedErrors = runtimeErrors.filter(
    (message) => !allowlist.some((pattern) => pattern.test(message)),
  );
  expect(unexpectedErrors).toEqual([]);
}
