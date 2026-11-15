"""Verify O3 against the pinned isolated fixture; never replay external mutations."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parents[5]
HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('o3_runtime', ROOT / 'scripts/testing/current-p0-runtime.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
RUN = HERE / ('run-' + str(time.time_ns()))
RUN.mkdir()
env = helper.child_env(RUN)
env['GOCACHE'] = 'D:/cache/go/build'
# Live provider tests are explicitly outside this local isolated MySQL run.
env.pop('PANTHEON_ARGO_PRODUCT_FIXTURE', None)
command = ['go', 'test', '-race', '-p=1', '-buildvcs=false', '-mod=readonly', '-json', './modules/business/cicd/argocd', './modules/business/k8s/release']

def pins():
    return {str(p.relative_to(ROOT)).replace('\\', '/'): hashlib.sha256(p.read_bytes()).hexdigest()
            for area in ('backend/modules/business/cicd/argocd', 'backend/modules/business/k8s/release')
            for p in (ROOT / area).rglob('*.go')}

before = pins()
started = time.monotonic()
with (RUN / 'stdout.log').open('xb') as out, (RUN / 'stderr.log').open('xb') as err:
    completed = subprocess.run(command, cwd=ROOT / 'backend', env=env, stdin=subprocess.DEVNULL, stdout=out, stderr=err,
                               timeout=420, creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
after = pins()
events = []
for line in (RUN / 'stdout.log').read_text(encoding='utf-8', errors='replace').splitlines():
    try:
        event = json.loads(line)
    except ValueError:
        continue
    if event.get('Action') in ('pass', 'fail', 'skip'):
        events.append({k: event[k] for k in ('Action', 'Package', 'Test') if k in event})
receipt = {'command': command, 'exitCode': completed.returncode, 'elapsedSeconds': round(time.monotonic()-started, 3),
           'sourceStable': before == after, 'sourceCount': len(before), 'fixtureSHA256': helper.FIXTURE_SHA,
           'testPasses': sum(e['Action'] == 'pass' and 'Test' in e for e in events),
           'skipped': [e for e in events if e['Action'] == 'skip'], 'failed': [e for e in events if e['Action'] == 'fail'],
           'credentialsPrinted': False, 'privateLogs': str(RUN.relative_to(ROOT)).replace('\\', '/')}
(RUN / 'source-before.json').write_text(json.dumps(before, indent=2), encoding='utf-8')
(RUN / 'source-after.json').write_text(json.dumps(after, indent=2), encoding='utf-8')
(RUN / 'receipt.json').write_text(json.dumps(receipt, indent=2), encoding='utf-8')
print(json.dumps(receipt))
raise SystemExit(completed.returncode)
