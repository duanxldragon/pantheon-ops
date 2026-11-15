import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';
const evidence = path.dirname(fileURLToPath(import.meta.url));
const workspace = process.cwd();
const temp = path.join(workspace, '.tmp');
fs.mkdirSync(temp, {recursive:true});
const root = fs.mkdtempSync(path.join(temp, 'o4-code-safety-'));
os.tmpdir = () => root;
fs.appendFileSync(path.join(evidence, 'fixture-roots-v2.jsonl'), JSON.stringify({root, pid:process.pid})+'\n');
// Process-local Git long-path setting; no repository/global config mutation.
const count = Number(process.env.GIT_CONFIG_COUNT ?? 0);
process.env.GIT_CONFIG_COUNT = String(count + 1);
process.env[`GIT_CONFIG_KEY_${count}`] = 'core.longpaths';
process.env[`GIT_CONFIG_VALUE_${count}`] = 'true';
const original = fs.rmSync;
fs.rmSync = function(target, options) {
  const absolute = path.resolve(target);
  if (!options?.recursive) return original.call(this, target, options);
  if (!absolute.startsWith(root + path.sep)) throw new Error('Refuse recursive teardown outside repair fixtures');
  if (!fs.existsSync(absolute)) return;
  if (path.dirname(absolute) === root) return; // Keep entire fixtures at original names.
  // Official cache/reassembly tests need an absent namespace for rebuilding.
  // Preserve their disposable subtree with one bounded rename, never delete it.
  // Candidate cleanup uses no native rm and cannot acquire authority here.
  const stat = fs.lstatSync(absolute, {bigint:true});
  if (!stat.isDirectory() || stat.isSymbolicLink()) throw new Error('Refuse unsupported retained move');
  const retained = path.join(path.dirname(absolute), path.basename(absolute)+'.retained-'+crypto.randomUUID());
  if (!retained.startsWith(root+path.sep) || !absolute.startsWith(workspace+path.sep)) throw new Error('Retained move outside exact fixture bounds');
  const current = fs.lstatSync(absolute, {bigint:true});
  if(current.dev !== stat.dev || current.ino !== stat.ino) throw new Error('Refuse observed fixture replacement');
  fs.renameSync(absolute, retained);
  fs.appendFileSync(path.join(evidence, 'retained-moves-v2.jsonl'), JSON.stringify({absolute, retained, dev:String(stat.dev), ino:String(stat.ino)})+'\n');
};
