import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { projectConsumerReviewRegistry } from '../../../scripts/business-overlay/rebuild-from-base.mjs';
const checker = path.resolve('scripts/harness/check-review.mjs');
function fixture(t) {
 const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ops-review-projection-'));
 t.after(() => fs.rmSync(root, {recursive:true,force:true}));
 const base = path.join(root,'base'); const consumer = path.join(root,'consumer');
 const entry = '.harness/archive/2026-09/evidence/old-base/review.md';
 const write = (dir,file,value) => { const p=path.join(dir,file);fs.mkdirSync(path.dirname(p),{recursive:true});fs.writeFileSync(p,typeof value==='string'?value:JSON.stringify(value)); };
 write(base,'config/foundation-consumer-artifacts.json',{schemaVersion:1,sourceOwner:'pantheon-base',reviewLegacyRegistry:{path:'config/review-loop-legacy-artifacts.json',artifactRoot:'.harness/archive'}});
 write(base,'config/review-loop-legacy-artifacts.json',{schemaVersion:1,entries:[entry]});
 write(base,entry,'historical review that cannot be restated');
 fs.mkdirSync(consumer,{recursive:true});return {base,consumer,entry,write};
}
test('excluded Base history is recorded, while an invalid active consumer review still fails',t=>{
 const {base,consumer,entry,write}=fixture(t);
 const before=fs.readFileSync(path.join(base,'config/review-loop-legacy-artifacts.json'),'utf8');
 assert.deepEqual(projectConsumerReviewRegistry(base,consumer,{replaceRoots:['.harness']},'abc'),['config/review-loop-legacy-artifacts.json']);
 const config=JSON.parse(fs.readFileSync(path.join(consumer,'config/review-loop-legacy-artifacts.json')));
 assert.deepEqual(config.entries,[]);assert.deepEqual(config.projection.externalBaseArtifacts,[entry]);
 assert.equal(fs.readFileSync(path.join(base,'config/review-loop-legacy-artifacts.json'),'utf8'),before);
 write(consumer,'.harness/evidence/current/review.md','current invalid review');
 const result=spawnSync(process.execPath,[checker,'--root',consumer,'--strict'],{encoding:'utf8'});
 assert.equal(result.status,1);assert.match(result.stdout,/missing ## Machine Readable/);
 assert.doesNotMatch(result.stdout,/stale legacy/);
});
test('retained archives remain listed and missing retained artifacts fail',t=>{
 const {base,consumer,entry,write}=fixture(t);
 assert.throws(()=>projectConsumerReviewRegistry(base,consumer,{},'abc'),/Missing retained/);
 write(consumer,entry,'retained history');assert.deepEqual(projectConsumerReviewRegistry(base,consumer,{},'abc'),[]);
 fs.rmSync(path.join(base,entry));assert.throws(()=>projectConsumerReviewRegistry(base,consumer,{replaceRoots:['.harness']},'abc'),/Stale Base/);
});
test('current or traversal entries cannot be projected away',t=>{
 const {base,consumer,write}=fixture(t);
 for(const entry of ['.harness/evidence/current/review.md','.harness/archive/../evidence/current/review.md']) {
  write(base,'config/review-loop-legacy-artifacts.json',{entries:[entry]});
  assert.throws(()=>projectConsumerReviewRegistry(base,consumer,{replaceRoots:['.harness']},'abc'),/canonical archive/);
 }
});
