"""O6 targeted verification using fresh schemas and hashed private fixtures."""
import importlib.util
import json
from pathlib import Path
import secrets
import subprocess
import time
import hashlib

ROOT = Path(__file__).resolve().parents[2]
EVIDENCE = ROOT / '.harness/evidence/2026-11-14-ops-latest-foundation-and-plans/verification/current-o6'

def run():
    spec = importlib.util.spec_from_file_location('p0_runtime', ROOT / 'scripts/testing/current-p0-runtime.py')
    p0 = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(p0)
    private = EVIDENCE / ('go-' + secrets.token_hex(4))
    private.mkdir(parents=True)
    env = p0.child_env(private)
    # Only authorized test configuration keys enter the child process.
    args = ['go', 'test', '-race', '-p=1', '-mod=readonly', '-count=1', '-json', '-run', 'Dashboard',
            './modules/business/deploy', './modules/business/k8s/cluster', './modules/business/k8s/workload',
            './modules/business/observability', './modules/business/shared/dashboard']
    before = p0.source_map()
    started = time.monotonic()
    stdout, stderr = private / 'stdout.log', private / 'stderr.log'
    with stdout.open('xb') as out, stderr.open('xb') as err:
        result = subprocess.run(args, cwd=ROOT / 'backend', env=env, stdout=out, stderr=err, timeout=600,
                                creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
    events = []
    for line in stdout.read_text(encoding='utf-8', errors='replace').splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if event.get('Action') in ('pass', 'fail', 'skip'):
            events.append({k: event[k] for k in ('Action', 'Package', 'Test') if k in event})
    after = p0.source_map()
    receipt = {'command': args, 'exitCode': result.returncode, 'elapsedSeconds': round(time.monotonic()-started, 2),
               'fixtureSHA256': p0.FIXTURE_SHA, 'credentialsPrinted': False,
               'sourceStable': before == after, 'changedSources': [k for k in before.keys() | after.keys() if before.get(k) != after.get(k)],
               'testEvents': events, 'stdoutSHA256': p0.digest(stdout), 'stderrSHA256': p0.digest(stderr),
               'privateLogPath': str(private.relative_to(ROOT)).replace('\\', '/')}
    p0.write(private / 'source-before.json', before)
    p0.write(private / 'source-after.json', after)
    p0.write(EVIDENCE / (private.name + '.json'), receipt)
    print(json.dumps(receipt))
    return result.returncode

if __name__ == '__main__':
    raise SystemExit(run())
