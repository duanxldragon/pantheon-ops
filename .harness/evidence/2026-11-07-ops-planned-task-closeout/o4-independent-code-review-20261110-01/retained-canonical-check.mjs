import fs from 'node:fs';
import path from 'node:path';
import { checkSharedDrift } from '../../../../scripts/business-overlay/check-shared-drift.mjs';
const original=path.resolve('.tmp/foundation-candidates/candidate-4315cee606c249a42c8a3372c26dee9a4658a9259229485de85953430c7f439f');
const descriptor=JSON.parse(fs.readFileSync('.local/ops-acceptance/o4-metadata-bound-current-v2-stable-real/sealed/candidate-descriptor.json'));
const logs=[];const savedLog=console.log,savedError=console.error;
console.log=(...args)=>logs.push({level:'log',text:args.join(' ')});console.error=(...args)=>logs.push({level:'error',text:args.join(' ')});
let exitCode,error;
try {exitCode=checkSharedDrift({root:path.join(original,'assembled'),snapshot:{baseRoot:path.join(original,'snapshot'),manifest:descriptor,baseCommit:descriptor.baseCommit},sourceRoot:process.cwd()});}
catch(e){error=e.message;exitCode=1;}finally{console.log=savedLog;console.error=savedError;}
const result={exitCode,error,logs,note:'Read-only checker on immutable failed original; this is not assembly/runtime success.'};
fs.writeFileSync(new URL('./retained-canonical-check.json',import.meta.url),JSON.stringify(result,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({exitCode,error,lines:logs.length,summary:logs.filter(l=>/^(OK |FAIL |DERIVED_|ALLOWLIST_NOT_COMPARED)/.test(l.text))},null,2));process.exitCode=exitCode;
