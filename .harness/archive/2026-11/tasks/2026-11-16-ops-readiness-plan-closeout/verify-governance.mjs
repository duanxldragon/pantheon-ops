import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {readTaskManifest, validateTaskManifest} from '../../../scripts/task-manifest.mjs';
const root=process.cwd();
const id='2026-11-16-ops-readiness-plan-closeout';
const dir=`.harness/evidence/${id}`;
const hash=v=>crypto.createHash('sha256').update(v).digest('hex');
const commands=[];
const run=(label,args)=>{
  const result=spawnSync(process.execPath,args,{encoding:'utf8',maxBuffer:25*1024*1024});
  const output=`${dir}/${label}.json`;
  fs.writeFileSync(output,result.stdout||'{}');
  if(result.stderr)fs.writeFileSync(`${dir}/${label}.stderr.txt`,result.stderr);
  commands.push({command:`node ${args.join(' ')}`,cwd:root,status:result.status===0?'passed':'failed',exitCode:result.status,output});
  let report;try{report=JSON.parse(result.stdout)}catch{report={output:result.stdout.slice(-1800)}}
  console.log(label,JSON.stringify({exitCode:result.status,errorCount:report.errorCount,warningCount:report.warningCount,findingCount:report.findingCount,output:report.output}));
  return report;
};
const docs=fs.readdirSync('docs/harness/tasks').filter(n=>n.startsWith('2026-11-16-ops-')&&n.endsWith('.task.md')).map(n=>'docs/harness/tasks/'+n);
const scoped=run('scoped-task-packet',['scripts/harness/check-task-packet.mjs','--root','.','--json',...docs]);
for(const item of scoped.results??[])if(item.errors.length||item.warnings.length)console.log(item.file,item.errors,item.warnings);
const failures=[];
for(const doc of docs){const taskId=path.basename(doc,'.task.md');try{validateTaskManifest(JSON.parse(fs.readFileSync(`.harness/tasks/${taskId}/manifest.json`,'utf8')),{manifestPath:`.harness/tasks/${taskId}/manifest.json`});}catch(e){failures.push({taskId,error:e.message})}}
fs.writeFileSync(`${dir}/scoped-manifests.json`,JSON.stringify({fileCount:docs.length,failures},null,2)+'\n');
console.log('scoped-manifests',docs.length,JSON.stringify(failures));
const inventory=JSON.parse(fs.readFileSync(`${dir}/archive-inventory.json`,'utf8'));
let preserved=0;const archiveFailures=[];
for(const a of inventory.archival){
  try{const resolved=readTaskManifest(root,a.id);if(resolved.path!==a.targetTask+'/manifest.json')throw new Error('wrong manifest resolution');}catch(e){archiveFailures.push({id:a.id,error:e.message})}
  for(const [file,digest]of Object.entries(a.evidenceBefore)){if(hash(fs.readFileSync(a.targetEvidence+'/'+file))!==digest)archiveFailures.push({id:a.id,file,error:'evidence hash mismatch'});else preserved++;}
  for(const [file,digest]of Object.entries(a.tasksBefore)){
    if(a.id==='2026-11-13-base-v0187-consumption'&&file==='manifest.json')continue;
    if(hash(fs.readFileSync(a.targetTask+'/'+file))!==digest)archiveFailures.push({id:a.id,file,error:'task hash mismatch'});
  }
}
const lockUnchanged=hash(fs.readFileSync('foundation-release.lock.json'))===inventory.lockHash;
fs.writeFileSync(`${dir}/archive-verification.json`,JSON.stringify({archived:inventory.archival.length,evidenceFilesPreserved:preserved,lockUnchanged,failures:archiveFailures},null,2)+'\n');
console.log('archive-verification',JSON.stringify({archived:inventory.archival.length,preserved,lockUnchanged,failures:archiveFailures}));
const full=run('final-task-packet',['scripts/harness/check-task-packet.mjs','--root','.','--json']);
run('final-doc-links',['scripts/harness/check-doc-links.mjs','--root','.','--strict','--json']);
run('final-evidence',['scripts/harness/check-evidence.mjs','--root','.','--strict','--json']);
run('final-review',['scripts/harness/check-review.mjs','--root','.','--strict','--json']);
const comparisons={};
for(const label of ['task-packet','doc-links','evidence','review']){
  const before=JSON.parse(fs.readFileSync(`${dir}/baseline-${label}.json`,'utf8'));
  const after=JSON.parse(fs.readFileSync(`${dir}/final-${label}.json`,'utf8'));
  const normalize=s=>s.replace(/\.harness\/archive\/\d{4}-\d{2}\/(evidence|tasks)\//g,'.harness/$1/');
  const flatten=r=>(r.results??r.findings??[]).flatMap(x=>x.reason?[`${normalize(x.file)}|${normalize(x.reason)}`]:[...(x.errors??[]),...(x.historicalErrors??[])].map(e=>`${normalize(x.file)}|${normalize(e)}`));
  const b=new Set(flatten(before)),a=new Set(flatten(after));
  const missingScan=(before.results??[]).filter(x=>!(after.results??[]).some(y=>normalize(y.file)===normalize(x.file))).map(x=>x.file);
  comparisons[label]={before:{errors:before.errorCount,historicalErrors:before.historicalErrorCount,findings:before.findingCount,warnings:before.warningCount,files:before.fileCount},after:{errors:after.errorCount,historicalErrors:after.historicalErrorCount,findings:after.findingCount,warnings:after.warningCount,files:after.fileCount},newFindings:[...a].filter(x=>!b.has(x)),removedFindings:[...b].filter(x=>!a.has(x)),missingScan};
}
fs.writeFileSync(`${dir}/gate-comparison.json`,JSON.stringify(comparisons,null,2)+'\n');
for(const [k,v]of Object.entries(comparisons))console.log(k,'comparison',JSON.stringify({...v,removedFindings:v.removedFindings.length}));
fs.writeFileSync(`${dir}/verification-invocations.json`,JSON.stringify({taskId:id,commands},null,2)+'\n');
if(scoped.errorCount||failures.length||archiveFailures.length||!lockUnchanged||Object.values(comparisons).some(c=>c.newFindings.length||c.missingScan.length))process.exitCode=1;
