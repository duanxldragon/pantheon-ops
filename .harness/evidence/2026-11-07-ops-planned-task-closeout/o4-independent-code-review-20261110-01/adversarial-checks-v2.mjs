import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { buildCandidateArtifact } from '../../../../../pantheon-base/scripts/foundation-release/build-candidate-artifact.mjs';
import * as consumer from '../../../../scripts/business-overlay/foundation-candidate.mjs';
import { rebuildFromCandidate } from '../../../../scripts/business-overlay/rebuild-from-candidate.mjs';
const source = fs.readFileSync('scripts/business-overlay/foundation-candidate.test.mjs', 'utf8');
const start = source.indexOf('function fixture(t) {');
const end = source.indexOf('\nfunction repin(', start);
const write = (root, file, bytes) => { const target = path.join(root, file); fs.mkdirSync(path.dirname(target), { recursive: true }); fs.writeFileSync(target, bytes); };
// Reuse only the existing small setup, preserving every root. Attack assertions are independent.
const fixture = new Function('fs','path','os','spawnSync','assert','buildCandidateArtifact','consumer','rebuildFromCandidate','write', source.slice(start,end) + '\nreturn fixture;')(fs,path,os,spawnSync,assert,buildCandidateArtifact,consumer,rebuildFromCandidate,write);
const results=[];
function run(name, fn) { try { results.push({name,...fn()}); } catch(e) { results.push({name,status:'HARNESS_ERROR',error:e.message}); process.exitCode=1; } }
run('single regular file exclusion', () => {
  const f=fixture({after(){}}); f.manifest.baseExcludedPaths=['docs/base.md'];
  write(f.ops,'business-overlay.json',JSON.stringify(f.manifest));
  let error; try { f.rebuild(); } catch(e) {error=e;}
  assert.match(error?.message ?? '', /Unknown directory ownership/);
  return {status:'DEFECT_REPRODUCED',error:error.message,fixtureRoot:f.temp};
});
run('replacement between exclusive mkdir and immediate identity capture', () => {
  const f=fixture({after(){}}); f.resolve(); const io=consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const attacked=path.join(f.cacheRoot,'assembled','docs'), retained=path.join(f.temp,'retained-created-docs');
  const originalMkdir=fs.mkdirSync, originalUnlink=fs.unlinkSync; let fired=false, attempted=false, error;
  fs.mkdirSync=function(target,...args) { const result=originalMkdir.call(this,target,...args); if(target===attacked && !fired) {fired=true;fs.renameSync(attacked,retained);originalMkdir(attacked);fs.writeFileSync(path.join(attacked,'foreign.txt'),'preserve');} return result;};
  try { io.mkdirSync(attacked); } finally {fs.mkdirSync=originalMkdir;}
  fs.unlinkSync=function(file) { if(file===path.join(attacked,'foreign.txt')) {attempted=true;throw new Error('Review preserved attempted foreign unlink');} return originalUnlink(file); };
  try { consumer.removeCandidateStage(f.cacheRoot,{cause:new Error('independent primary failure')}); } catch(e){error=e;} finally{fs.unlinkSync=originalUnlink;}
  assert.equal(fired,true); assert.equal(attempted,true); assert.equal(error.cause.message,'independent primary failure');
  assert.equal(fs.readFileSync(path.join(attacked,'foreign.txt'),'utf8'),'preserve'); assert.equal(fs.statSync(retained).isDirectory(),true);
  return {status:'DEFECT_REPRODUCED',foreignUnlinkAttempted:attempted,primaryPreserved:true,fixtureRoot:f.temp,error:error.message};
});
run('observed directory replacement before file copy', () => {
  const f=fixture({after(){}}); const snapshot=f.resolve(); const io=consumer.candidateAssemblyFileSystem(f.cacheRoot);
  const attacked=path.join(f.cacheRoot,'assembled','docs'), retained=path.join(f.temp,'retained-owned-docs'); io.mkdirSync(attacked);
  fs.renameSync(attacked,retained); fs.mkdirSync(attacked); const target=path.join(attacked,'base.md'); fs.writeFileSync(target,'foreign original must remain');
  io.copyFileSync(path.join(snapshot.baseRoot,'docs/base.md'),target);
  assert.equal(fs.readFileSync(target,'utf8'),'Reviewed dirty candidate fixture.\n');
  return {status:'DEFECT_REPRODUCED',foreignFileOverwritten:true,fixtureRoot:f.temp};
});
run('foreign regular sibling inside registered directory', () => {
  const f=fixture({after(){}}); f.resolve(); const io=consumer.candidateAssemblyFileSystem(f.cacheRoot); const directory=path.join(f.cacheRoot,'assembled');io.mkdirSync(directory);
  const foreign=path.join(directory,'foreign-sibling.txt'); fs.writeFileSync(foreign,'preserve'); let attempted=false,error;const unlink=fs.unlinkSync;
  fs.unlinkSync=function(file) {if(file===foreign){attempted=true;throw new Error('Review preserved foreign sibling');}return unlink(file);};
  try{consumer.removeCandidateStage(f.cacheRoot);}catch(e){error=e;}finally{fs.unlinkSync=unlink;}
  assert.equal(attempted,true);assert.equal(fs.readFileSync(foreign,'utf8'),'preserve');
  return {status:'DEFECT_REPRODUCED',foreignUnlinkAttempted:true,fixtureRoot:f.temp,error:error.message};
});
run('adjacent BigInt directory IDs refuse replacement', () => {
  const f=fixture({after(){}}); f.resolve();const io=consumer.candidateAssemblyFileSystem(f.cacheRoot), directory=path.join(f.cacheRoot,'assembled');const lstat=fs.lstatSync;let replacement=false,error;
  fs.lstatSync=function(file,options){const stat=lstat(file,options);if(file===directory && options?.bigint)stat.ino=replacement?9007199254740993n:9007199254740992n;return stat;};
  try{io.mkdirSync(directory);replacement=true;try{io.mkdirSync(directory);}catch(e){error=e;}}finally{fs.lstatSync=lstat;}
  assert.match(error.message,/Replaced directory identity/);return {status:'PASS',fixtureRoot:f.temp};
});
run('foreign stage sibling is outside cleanup ownership', () => {
  const f=fixture({after(){}});f.resolve();const sibling=path.join(path.dirname(f.cacheRoot),'foreign-sibling');fs.mkdirSync(sibling);fs.writeFileSync(path.join(sibling,'keep.txt'),'preserve');consumer.removeCandidateStage(f.cacheRoot);assert.equal(fs.readFileSync(path.join(sibling,'keep.txt'),'utf8'),'preserve');return {status:'PASS',fixtureRoot:f.temp};
});
fs.writeFileSync(new URL('./adversarial-results.json',import.meta.url),JSON.stringify(results,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify(results.map(({fixtureRoot,error,...rest})=>rest),null,2));

