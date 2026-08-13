import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const frontendDirectory = fileURLToPath(new URL('../frontend/', import.meta.url));
const result = spawnSync('npm run type-check', {
  cwd: frontendDirectory,
  shell: true,
  stdio: 'inherit',
});

if (typeof result.status === 'number') {
  process.exit(result.status);
}

process.exit(1);
