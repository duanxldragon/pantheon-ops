import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.join(path.dirname(fileURLToPath(import.meta.url)), 'fixtures');
fs.mkdirSync(root, { recursive: true });
os.tmpdir = () => root;
const original = fs.rmSync;
fs.rmSync = function(target, options) {
  const absolute = path.resolve(target);
  if (options?.recursive) {
    if (!absolute.startsWith(root + path.sep)) throw new Error('Refuse recursive teardown outside repair fixtures');
    return; // Every refused/replaced fixture remains; no path-only recursive deletion.
  }
  return original.call(this, target, options);
};
