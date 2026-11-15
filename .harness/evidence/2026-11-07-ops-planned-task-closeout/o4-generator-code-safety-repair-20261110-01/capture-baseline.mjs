import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const here = path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1'));
const pins = JSON.parse(fs.readFileSync(path.join(here, '../o4-generator-integration-repair/source-hashes.json')));
const sha = file => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
const sources = pins.map(p => {
  const bytes = fs.readFileSync(p.path);
  const baseline = path.join(here, 'before', p.path);
  fs.mkdirSync(path.dirname(baseline), {recursive:true});
  fs.writeFileSync(baseline, bytes, {flag:'wx'});
  return {...p, beforeSha256:sha(p.path), retainedBaselineSha256:sha(p.baselineSource)};
});
const protectedFiles = ['foundation-release.lock.json', 'business-overlay.json', '.foundation/allowed-drift.json'];
for (const folder of ['../o4-generator-integration-repair', '../o4-independent-code-review-20261110-01']) {
  const walk = dir => { for (const entry of fs.readdirSync(dir, {withFileTypes:true})) {
    const file = path.join(dir,entry.name);
    if(entry.isDirectory()) { if(entry.name !== 'fixtures') walk(file); }
    else protectedFiles.push(file);
  }};
  walk(path.join(here,folder));
}
const sealed = path.resolve('.local/ops-acceptance/o4-metadata-bound-current-v2-stable-real/sealed');
const walk = dir => { for (const entry of fs.readdirSync(dir, {withFileTypes:true})) {
  const file = path.join(dir,entry.name);
  if(entry.isDirectory()) walk(file); else protectedFiles.push(file);
}};
walk(sealed);
fs.writeFileSync(path.join(here,'before-hashes.json'),JSON.stringify({sources,protectedFiles:protectedFiles.map(file=>({path:path.relative(process.cwd(),file),sha256:sha(file)}))},null,2)+'\n',{flag:'wx'});
if(sources.some(p=>p.beforeSha256!==p.newSha256 || p.retainedBaselineSha256!==p.oldSha256)) throw new Error('Original six pins do not match');
console.log('Original six current/new and retained/old pins match; independent/original/sealed artifacts captured read-only.');
