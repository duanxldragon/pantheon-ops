import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const evidence = path.dirname(fileURLToPath(import.meta.url));
const workspace = path.resolve(process.cwd());
const source = path.resolve(evidence, 'fixtures');
const target = path.resolve(workspace, '.tmp/o4-code-safety-initial-fixtures-20261110-01');
// One explicitly bounded rename of this Generator's NEW initial fixture root.
// No recursive deletion, no protected-stage change, no content relabeling.
assert.ok(source.startsWith(workspace+path.sep)); assert.ok(target.startsWith(path.join(workspace,'.tmp')+path.sep));
assert.equal(path.basename(evidence),'o4-generator-code-safety-repair-20261110-01');
assert.equal(fs.existsSync(target),false);
const identity = stat => [stat.dev,stat.ino,stat.size,stat.mtimeNs,stat.mode].map(String);
const entries = [];
const walk = directory => {
  for (const entry of fs.readdirSync(directory,{withFileTypes:true})) {
    const absolute = path.join(directory,entry.name), stat = fs.lstatSync(absolute,{bigint:true});
    entries.push({relative:path.relative(source,absolute),identity:identity(stat),directory:stat.isDirectory()});
    if(stat.isDirectory() && !stat.isSymbolicLink()) walk(absolute);
  }
};
walk(source);
const before = fs.lstatSync(source,{bigint:true});
assert.ok(before.isDirectory() && !before.isSymbolicLink());
const current = fs.lstatSync(source,{bigint:true});
assert.equal(current.dev,before.dev); assert.equal(current.ino,before.ino);
fs.renameSync(source,target);
const after = fs.lstatSync(target,{bigint:true});
assert.equal(after.dev,before.dev); assert.equal(after.ino,before.ino);
for(const entry of entries) assert.deepEqual(identity(fs.lstatSync(path.join(target,entry.relative),{bigint:true})),entry.identity);
fs.writeFileSync(path.join(evidence,'initial-fixture-retention.json'),JSON.stringify({taskId:'2026-11-07-ops-planned-task-closeout',source,target,rootIdentity:[String(before.dev),String(before.ino)],entriesVerified:entries.length,files:entries.filter(e=>!e.directory).length,deleted:0,reason:'Keep initial disposable large fixtures out of declared .harness metadata inputs. Original RED/failed/GREEN log paths map by replacing source prefix with target; every directory/file instance remains unchanged.'},null,2)+'\n',{flag:'wx'});
console.log(`Retained ${entries.length} unchanged file/directory instances at ${target}; deleted 0.`);
