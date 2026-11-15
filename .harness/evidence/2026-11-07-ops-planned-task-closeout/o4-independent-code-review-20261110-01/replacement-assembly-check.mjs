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
const f=fixture({after(){}}); const copy=fs.copyFileSync; let fired=false;
const attacked=path.join(f.cacheRoot,'assembled','docs'); const retained=path.join(f.temp,'retained-assembled-docs');
fs.copyFileSync=function(source,target,...args){
  if(target===path.join(attacked,'base.md') && !fired){
    fired=true;fs.renameSync(attacked,retained);fs.mkdirSync(attacked);fs.writeFileSync(target,'foreign original must remain');
  }
  return copy.call(this,source,target,...args);
};
let result,error;
try{result=f.rebuild();}catch(e){error=e;}finally{fs.copyFileSync=copy;}
assert.equal(fired,true);assert.equal(error,undefined,error?.message);
assert.equal(result.receipt.sourceStable,true);assert.equal(fs.existsSync(result.receiptPath),true);
assert.equal(fs.readFileSync(path.join(attacked,'base.md'),'utf8'),'Reviewed dirty candidate fixture.\n');
assert.equal(fs.statSync(retained).isDirectory(),true);
const evidence={status:'DEFECT_REPRODUCED',foreignFileOverwritten:true,successReceiptEmitted:true,sourceStableClaim:true,fixtureRoot:f.temp,receiptPath:result.receiptPath,note:'Actual rebuildFromCandidate small fixture; directory replaced immediately before copy; no forced primary failure.'};
fs.writeFileSync(new URL('./replacement-assembly-results.json',import.meta.url),JSON.stringify(evidence,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({status:evidence.status,foreignFileOverwritten:true,successReceiptEmitted:true,sourceStableClaim:true},null,2));
