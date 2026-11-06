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
const bash = process.platform === 'win32'
 ? path.resolve(path.dirname(execFileSync('where.exe', ['git'], {encoding:'utf8'}).trim().split(/\r?\n/)[0]), '..', 'bin', 'bash.exe')
 : 'bash';
function cleanup(fixture) {
 assert.equal(path.dirname(path.resolve(fixture)), path.resolve(os.tmpdir()), 'cleanup stays within test temp directory');
 fs.rmSync(fixture,{recursive:true,force:true});
}
for(const name of ['Report npm vulnerabilities (root)','Report npm vulnerabilities (frontend)']) {
 test(name+' preserves vulnerability, scan and installation failures',()=>{
  for(const [ciCode,auditCode] of [[0,0],[0,1],[0,42],[93,0]]) {
   const fixture=fs.mkdtempSync(path.join(os.tmpdir(),'pantheon-audit-gate-'));
   try {
    fs.mkdirSync(path.join(fixture,'security-reports'));
    const frontend=name.endsWith('(frontend)');
    const cwd=frontend?path.join(fixture,'frontend'):fixture;
    fs.mkdirSync(cwd,{recursive:true});
    const npm=`npm() { if [ "$1" = "ci" ]; then return ${ciCode}; fi; printf '{"fixture":true}\n'; return ${auditCode}; }\n`;
    const result=spawnSync(bash,['--noprofile','--norc','-e','-o','pipefail','-c',npm+shellFor(name)],{cwd,encoding:'utf8'});
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
test('enforcement blocks failed scanners on strict events and preserves PR policy',()=>{
 const fixture=fs.mkdtempSync(path.join(os.tmpdir(),'pantheon-enforcement-'));
 try {
  for(const event of ['push','schedule','workflow_dispatch','pull_request','merge_group']) {
   for(const failed of ['','GOVULNCHECK_RESULT','NPM_AUDIT_ROOT_RESULT','NPM_AUDIT_FRONTEND_RESULT']) {
    const env={...process.env,EVENT_NAME:event,GOVULNCHECK_RESULT:'success',NPM_AUDIT_ROOT_RESULT:'success',NPM_AUDIT_FRONTEND_RESULT:'success',GITHUB_STEP_SUMMARY:'summary.md'};
    if(failed)env[failed]='failure';
    const result=spawnSync(bash,['--noprofile','--norc','-e','-o','pipefail','-c',shellFor('Enforce gate')],{cwd:fixture,env,encoding:'utf8'});
    assert.ifError(result.error);
    const advisory=event==='pull_request'||event==='merge_group';
    assert.equal(result.status,failed&&!advisory?1:0,`${event} ${failed}: ${result.stderr}`);
   }
  }
 } finally {cleanup(fixture);}
});