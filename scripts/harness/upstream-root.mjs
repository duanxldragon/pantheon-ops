import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

export const DEFAULT_METHOD_CONFIG = 'config/method.config.json';

export function toRepoPath(root, filePath) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

export function readMethodConfig(root, configPath = DEFAULT_METHOD_CONFIG) {
  const absolutePath = path.isAbsolute(configPath) ? configPath : path.join(root, configPath);
  if (!fs.existsSync(absolutePath)) {
    return {};
  }

  try {
    return JSON.parse(fs.readFileSync(absolutePath, 'utf8'));
  } catch {
    return {};
  }
}

function normalizeCandidate(root, candidate) {
  if (!candidate || typeof candidate !== 'string') {
    return null;
  }
  return path.isAbsolute(candidate) ? candidate : path.resolve(root, candidate);
}

export function resolvePantheonHarnessRoot(root, options = {}) {
  const config = readMethodConfig(root, options.config);
  const candidates = [
    options.methodRoot,
    process.env.PANTHEON_HARNESS_ROOT,
    config.pantheonHarnessRoot,
    config.methodRoot,
    'pantheon-harness',
    '../pantheon-harness',
    '../../pantheon-harness',
  ]
    .map((candidate) => normalizeCandidate(root, candidate))
    .filter(Boolean);

  const seen = new Set();
  for (const candidate of candidates) {
    const resolved = path.resolve(candidate);
    if (seen.has(resolved)) {
      continue;
    }
    seen.add(resolved);
    if (
      fs.existsSync(path.join(resolved, 'VERSION')) &&
      fs.existsSync(path.join(resolved, 'patterns', 'METHOD_VERSION.json'))
    ) {
      return resolved;
    }
  }

  return path.resolve(candidates[0] ?? path.join(root, '..', 'pantheon-harness'));
}

export function formatExternalPath(root, externalRoot, repoPath) {
  return toRepoPath(root, path.join(externalRoot, repoPath));
}
