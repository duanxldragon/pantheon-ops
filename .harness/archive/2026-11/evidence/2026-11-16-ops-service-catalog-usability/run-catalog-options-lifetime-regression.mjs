import path from 'node:path';
import { fileURLToPath } from 'node:url';

// Evidence-only runner: canonical unit-test/overlay ownership is held by root.
// Reuse the installed app config and dependencies without changing shared files.
const frontendRoot = fileURLToPath(new URL('../../../frontend/', import.meta.url));
const { startVitest } = await import(new URL('../../../frontend/node_modules/vitest/dist/node.js', import.meta.url));
await startVitest([], {
  root: path.dirname(frontendRoot.replace(/[\\/]$/, '')),
  config: path.join(frontendRoot, 'vitest.config.ts'),
  include: ['.harness/evidence/2026-11-16-ops-service-catalog-usability/catalog-options-lifetime.regression.test.ts'],
  run: true,
  pool: 'threads',
  maxWorkers: 1,
  fileParallelism: false,
  reporters: ['verbose'],
}, {
  resolve: {
    alias: {
      vitest: path.join(frontendRoot, 'node_modules/vitest/dist/index.js'),
      '@testing-library/react': path.join(frontendRoot, 'node_modules/@testing-library/react/dist/index.js'),
    },
  },
});
