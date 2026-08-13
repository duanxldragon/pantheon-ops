import fs from 'node:fs/promises';
import path from 'node:path';

export async function readGoBackendImportPrefix(repoRoot: string) {
  // Supported foundation layouts: Base owns backend/go.mod, while consumers
  // may own a root go.mod with the shared backend kept under backend/.
  const candidates = [
    { goModPath: path.join(repoRoot, 'backend', 'go.mod'), relativeBackendPath: '' },
    { goModPath: path.join(repoRoot, 'go.mod'), relativeBackendPath: 'backend' },
  ];
  for (const { goModPath, relativeBackendPath } of candidates) {
    try {
      const goMod = await fs.readFile(goModPath, 'utf8');
      const moduleMatch = goMod.match(/^module\s+(\S+)\s*$/mu);
      if (!moduleMatch) {
        throw new Error(`Go module directive not found: ${goModPath}`);
      }
      return relativeBackendPath
        ? path.posix.join(moduleMatch[1], relativeBackendPath)
        : moduleMatch[1];
    } catch (error) {
      if (!(error && typeof error === 'object' && 'code' in error && error.code === 'ENOENT')) {
        throw error;
      }
    }
  }
  throw new Error(`Go module file not found under repository root: ${repoRoot}`);
}
