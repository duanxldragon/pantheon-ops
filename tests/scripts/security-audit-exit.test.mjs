import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import assert from 'node:assert/strict';
import {spawnSync, execFileSync} from 'node:child_process';
import test from 'node:test';

const source=fs.readFileSync('.github/workflows/security.yml','utf8');
function shellFor(name) {
 const step=source.split('      - name: '+name)[1];
 assert.ok(step,'missing workflow step: '+name);
 const lines=step.split(/\r?\n/);
 const start=lines.findIndex(l=>/^        run: \|\s*$/.test(l));
 assert.ok(start>=0,'missing multiline shell');
 const body=[];
 for(const line of lines.slice(start+1)) {
  if(line.trim()&&!line.startsWith('          '))break;
  body.push(line.slice(10));
 }
 return body.join('\n');
}
// Git for Windows keeps bash in <root>/usr/bin or <root>/bin. `where git` may
// return a mingw64-internal path whose sibling bash does not exist, so walk up
// from that git until a known bash layout is found.
function winBash() {
 const gitPath=execFileSync('where.exe', ['git'], {encoding:'utf8',windowsHide:true}).trim().split(/\r?\n/)[0];
 let dir=path.dirname(gitPath);
 for(let up=0;up<4;up+=1) {
  for(const candidate of ['usr/bin/bash.exe','bin/bash.exe']) {
   if(fs.existsSync(path.join(dir,candidate)))return path.join(dir,candidate);
  }
  dir=path.dirname(dir);
 }
 return 'bash';
}
const bash = process.platform === 'win32' ? winBash() : 'bash';
function cleanup(fixture) {
 assert.equal(path.dirname(path.resolve(fixture)), path.resolve(os.tmpdir()), 'cleanup stays within test temp directory');
 fs.rmSync(fixture,{recursive:true,force:true});
}
for(const name of ['Report root npm vulnerabilities','Report frontend npm vulnerabilities']) {
 test(name+' preserves vulnerability, scan and installation failures',()=>{
  for(const [ciCode,auditCode] of [[0,0],[0,1],[0,42],[93,0]]) {
   const fixture=fs.mkdtempSync(path.join(os.tmpdir(),'pantheon-audit-gate-'));
   try {
    fs.mkdirSync(path.join(fixture,'security-reports'));
    const frontend=name.includes('frontend');
    const cwd=frontend?path.join(fixture,'frontend'):fixture;
    fs.mkdirSync(cwd,{recursive:true});
    const npm=`npm() { if [ "$1" = "ci" ]; then return ${ciCode}; fi; printf '{"fixture":true}\n'; return ${auditCode}; }\n`;
    // The audit step has no `set -e`, but the test banner adds -e, so a failing
    // `npm ci` aborts before `npm audit` runs and the report is not written.
    const result=spawnSync(bash,['--noprofile','--norc','-e','-o','pipefail','-c',npm+shellFor(name)],{cwd,encoding:'utf8',windowsHide:true});
    assert.ifError(result.error);
    assert.equal(result.status,ciCode||auditCode,`ci=${ciCode} audit=${auditCode}: ${result.stderr}`);
    if(!ciCode) {
     const file=path.join(fixture,'security-reports',frontend?'npm-audit-frontend.json':'npm-audit-root.json');
     assert.equal(JSON.parse(fs.readFileSync(file,'utf8')).fixture,true,'report survives nonzero exit');
    }
   } finally {cleanup(fixture);}
  }
 });
}
test('dependency-vulnerabilities job runs only on schedule, dispatch, or push',()=>{
 // Advisory events (pull_request, merge_group) are made non-blocking by skipping
 // the whole job, not by softening the gate shell. Assert that job-level policy
 // exactly as written in the workflow.
 const job=source.split(/^  dependency-vulnerabilities:$/m)[1].split(/^  [a-z][a-z-]*:$/m)[0];
 const jobIf=(job.match(/if:\s*github\.event_name[^\n]*/)||[])[0]||'';
 assert.match(jobIf,/schedule/,'runs on schedule');
 assert.match(jobIf,/workflow_dispatch/,'runs on workflow_dispatch');
 assert.match(jobIf,/push/,'runs on push');
 assert.doesNotMatch(jobIf,/pull_request/,'must not run on pull_request');
 assert.doesNotMatch(jobIf,/merge_group/,'must not run on merge_group');
});
test('enforcement gate fails closed when any scanner fails',()=>{
 const fixture=fs.mkdtempSync(path.join(os.tmpdir(),'pantheon-enforcement-'));
 try {
  const shell=shellFor('Enforce dependency vulnerability gate');
  for(const failed of ['GOVULNCHECK_RESULT','NPM_AUDIT_ROOT_RESULT','NPM_AUDIT_FRONTEND_RESULT','OSV_CALL_ANALYSIS_RESULT']) {
   const env={...process.env,GOVULNCHECK_RESULT:'success',NPM_AUDIT_ROOT_RESULT:'success',NPM_AUDIT_FRONTEND_RESULT:'success',OSV_CALL_ANALYSIS_RESULT:'success'};
   env[failed]='failure';
   const result=spawnSync(bash,['--noprofile','--norc','-e','-o','pipefail','-c',shell],{cwd:fixture,env,encoding:'utf8',windowsHide:true});
   assert.ifError(result.error);
   assert.equal(result.status,1,`${failed} must fail closed: ${result.stderr}`);
  }
  const okEnv={...process.env,GOVULNCHECK_RESULT:'success',NPM_AUDIT_ROOT_RESULT:'success',NPM_AUDIT_FRONTEND_RESULT:'success',OSV_CALL_ANALYSIS_RESULT:'success'};
  const okResult=spawnSync(bash,['--noprofile','--norc','-e','-o','pipefail','-c',shell],{cwd:fixture,env:okEnv,encoding:'utf8',windowsHide:true});
  assert.ifError(okResult.error);
  assert.equal(okResult.status,0,`all success must pass: ${okResult.stderr}`);
 } finally {cleanup(fixture);}
});
