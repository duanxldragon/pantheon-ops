import fs from 'node:fs/promises';
import path from 'node:path';

export async function readGoModulePath(repoRoot: string) {
  const candidates = [path.join(repoRoot, 'backend', 'go.mod'), path.join(repoRoot, 'go.mod')];
  for (const goModPath of candidates) {
    try {
      const goMod = await fs.readFile(goModPath, 'utf8');
      const moduleMatch = goMod.match(/^module\s+(\S+)\s*$/mu);
      if (!moduleMatch) {
        throw new Error(`Go module directive not found: ${goModPath}`);
      }
      return moduleMatch[1];
    } catch (error) {
      if (!(error && typeof error === 'object' && 'code' in error && error.code === 'ENOENT')) {
        throw error;
      }
    }
  }
  throw new Error(`Go module file not found under repository root: ${repoRoot}`);
}
