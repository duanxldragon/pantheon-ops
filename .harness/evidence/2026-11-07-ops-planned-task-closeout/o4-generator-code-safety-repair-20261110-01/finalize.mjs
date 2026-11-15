import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';
const here = path.dirname(fileURLToPath(import.meta.url));
const sha = file => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
const integrity = JSON.parse(fs.readFileSync(path.join(here, 'integrity.json')));
for(const p of integrity.sources) assert.equal(sha(p.path),p.afterSha256);
for(const p of integrity.extra) assert.equal(sha(p.path),p.sha256);
const stats = file => {
  const log = fs.readFileSync(path.join(here,file),'utf8');
  const value = name => Number(new RegExp(`ℹ ${name} (\\d+)`).exec(log)?.[1]);
  return {log:file,tests:value('tests'),passed:value('pass'),failed:value('fail'),skipped:value('skipped'),sha256:sha(path.join(here,file))};
};
const red = stats('red.log'), final = stats('all-suites-final-v2.log');
assert.equal(red.failed,7); assert.equal(final.tests,171); assert.equal(final.passed,171); assert.equal(final.failed,0); assert.equal(final.skipped,0);
const importPath = './.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-code-safety-repair-20261110-01/preserve-fixtures-v2.mjs';
const suiteCommand = `node --import '${importPath}' --test scripts/business-overlay/foundation-candidate.test.mjs scripts/business-overlay/candidate-ownership.test.mjs tests/scripts/business-overlay/*.test.mjs`;
const commands = {
  taskId:'2026-11-07-ops-planned-task-closeout',tier:'L2',role:'Generator/executor',
  commands:[
    {command:"node --import './.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-code-safety-repair-20261110-01/preserve-fixtures.mjs' --test scripts/business-overlay/candidate-ownership.test.mjs",exitCode:1,...red,note:'Four confirmed independent defects reproduced before implementation; original imported modules loaded from six matching before pins.'},
    {command:suiteCommand,exitCode:0,...final,durationMs:178365.7997,note:'All 155 existing candidate/overlay cases and 16 ownership cases. Includes real small-fixture assembly, actual local Go modfile normalization, immutable fresh --attempt replay and all canonical/tamper/allowlist/official-lock contracts.'},
    {command:'node scripts/business-overlay/check-business-overlay.mjs',exitCode:0,log:'check-business-overlay.log',result:'OK business overlay contract (56 declared paths, rebuilt tree)'},
    {command:'node --check <each of the six original scoped source/test files and candidate-ownership.test.mjs>',exitCode:0,checks:integrity.syntax,result:'7 native syntax checks pass; LSP unavailable'},
    {command:'node .harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-code-safety-repair-20261110-01/integrity.mjs',exitCode:0,result:'6 old/new baseline pins; 35 protected files, 1141 snapshot files, 4634 report-owned files unchanged; NEW repair-only.patch'},
    {command:'node .harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-code-safety-repair-20261110-01/retained-canonical-check.mjs',exitCode:0,result:'Read-only original failed tree: 19 canonical outputs, 801/801 shared bytes, 10 migrations; no assembly/runtime success'},
    {command:'node .harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-code-safety-repair-20261110-01/retain-initial-fixtures.mjs',exitCode:0,result:'30904 NEW initial fixture file/directory instances retained with unchanged identities outside .harness metadata; 0 deleted; initial-fixture-retention.json gives exact prefix mapping'},
    {command:'node scripts/business-overlay/rebuild-from-candidate.mjs --help',exitCode:0,result:'CLI help succeeds; full CLI consumption/assembly not run'},
    {command:'git diff --check -- scripts/business-overlay/rebuild-from-base.mjs scripts/business-overlay/check-shared-drift.mjs tests/scripts/business-overlay/check-shared-drift.test.mjs',exitCode:0,result:'No whitespace errors; untracked repaired files separately covered by native syntax and NEW before/current diff'}
  ],
  iterations:['green-v1.log','green-v2.log','green-v3.log','green-v4.log','green-final-v1.log','candidate-full-v1.log','candidate-focused-v2.log','overlay-full-v1.log','overlay-full-v2.log','all-suites-final-v1.log'].map(stats),
  boundaries:{directoryCreationAuthority:'unproven; root/snapshot/assembler/work directories retained, never deletion-authorized by post-mkdir lstat',namespaceAtomicity:'Node/Windows checks and mkdir/open/unlink/subprocess/receipt are separate syscalls; hostile interference within last-check/syscall windows is unsupported; empty mkdir replacement cannot be authenticated by path inspection',fileWrites:'O_EXCL creation + fstat identity; existing-file open without O_TRUNC, recheck before descriptor mutation and after close; unknown/replaced files refused',cleanup:'Entire-tree preflight rejects unknown ordinary files and directories before deletion. Whole-stage cleanup always throws deferred-unproven with the original cause. Individual file removal requires recorded creation identity and full preflight; unlink remains a non-atomic namespace operation.',go:'Alternate modfile workspace outside assembled, observed-chain and target checks before/after Go. Go-written workspace outputs have no deletion authority; stable reads import through owned target descriptors. No atomic Go provenance guarantee.',testRetention:'Disposable official cache/reassembly fixtures use bounded single rename instead of recursive deletion, with retained-moves-v2.jsonl; fixture roots/identities retained. No original/protected-stage mutation.',reviews:'Independent CODE and ARCH required before root fresh full attempt. No self-approval, actual full-source assembly, P0/runtime/release acceptance.'},
  gaps:['CodeGraph/LSP/ast-grep not callable','No full candidate-source assembly','No semantic frontend build/typecheck, CGO race or paired runtime in this owned script-only scope','No remote/credentials/release/lock/waiver/Base/backend/frontend/src changes']
};
fs.writeFileSync(path.join(here,'commands.json'),JSON.stringify(commands,null,2)+'\n',{flag:'wx'});
fs.writeFileSync(path.join(here,'source-hashes.json'),JSON.stringify([...integrity.sources.map(p=>({path:p.path,oldSha256:p.beforeSha256,newSha256:p.afterSha256,originalOldSha256:p.oldSha256,originalRepairSha256:p.newSha256,baselineSource:path.relative(process.cwd(),path.join(here,'before',p.path))})),...integrity.extra.map(p=>({...p,newSha256:p.sha256,newFile:true}))],null,2)+'\n',{flag:'wx'});
console.log('Final Generator evidence sealed: RED 7 failures; GREEN 171/171, 0 skipped; independent CODE/ARCH required.');
