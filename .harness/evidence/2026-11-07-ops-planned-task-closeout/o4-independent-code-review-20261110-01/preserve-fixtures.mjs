import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.join(path.dirname(fileURLToPath(import.meta.url)), 'fixtures');
os.tmpdir = () => root;
const original = fs.rmSync;
fs.rmSync = function(target, options) {
  const absolute = path.resolve(target);
  if (options?.recursive) {
    if (!absolute.startsWith(root + path.sep)) throw new Error('Review refuses recursive deletion outside owned fixtures');
    return; // Retain all fixture stages, replacements and refused roots for inspection.
  }
  return original.call(this, target, options);
};
