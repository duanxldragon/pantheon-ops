const truthyPattern = /^(1|true|yes|on)$/i;

export function shouldPreserveSmokeFixtures() {
  return truthyPattern.test(process.env.PANTHEON_SMOKE_PRESERVE_FIXTURES ?? '');
}

export function shouldCleanupSmokeFixtures() {
  return !shouldPreserveSmokeFixtures();
}

export function logPreservedSmokeFixtures(scope: string) {
  if (!shouldPreserveSmokeFixtures()) {
    return;
  }
  // Keep the console line stable so preserved runs are easy to grep from CI or local output.
  console.info(`[smoke-fixtures] preserved ${scope}`);
}

export async function runOptionalSmokeCleanup(scope: string, cleanup: () => Promise<void>) {
  if (shouldCleanupSmokeFixtures()) {
    await cleanup();
    return;
  }
  logPreservedSmokeFixtures(scope);
}
