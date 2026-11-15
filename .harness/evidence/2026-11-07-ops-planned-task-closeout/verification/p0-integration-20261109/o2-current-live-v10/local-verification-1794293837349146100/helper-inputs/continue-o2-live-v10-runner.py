"""O2 owned acceptance phases; credentials/DSN are injected through stdin only.

Root owns execution after reviewing/resealing continue-o2-live-v10-root-phase-request.json.
This script never installs a provider or touches production namespaces/services.
"""
from pathlib import Path
import importlib.util
import argparse
import base64
import hashlib
import json
import os
import re
import secrets
import subprocess
import sys
import time
from datetime import datetime, timezone

ROOT = Path(__file__).resolve().parents[2]
LOCAL = ROOT / '.local/ops-acceptance'
EVIDENCE = ROOT / '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v10'
CASES = ('success', 'failure-retry', 'pause-resume', 'cancel', 'concurrent',
         'revoke-credential', 'revoke-rotation', 'revoke-source', 'revoke-host', 'start-negative', 'hostkey-negative', 'cold-restart')
REQUEST = LOCAL / 'continue-o2-live-v10-phase-request.json'
spec = importlib.util.spec_from_file_location('v10_runner_contract', LOCAL / 'continue-o2-live-v10-contract.py')
C = importlib.util.module_from_spec(spec)
spec.loader.exec_module(C)


def write(name, value):
    path = EVIDENCE / ('continue-o2-live-v10-' + name + '.json')
    path.open('x', encoding='utf-8').write(json.dumps(value, ensure_ascii=False, indent=2) + '\n')
    return path


def source_hashes():
    return C.inventory()



def artifacts():
    return C.artifacts()



def request(phase='execute', dependencies=None):
    # Retain failure evidence without overwriting any sealed proposal/request.
    value = {'phase': phase, 'dependencies': dependencies or [], 'executionDisabled': True}
    write('runner-blocker-' + str(time.time_ns()), value)
    return value



def build():
    return C.load('verify.py').main()



def remote_command(client, script, timeout=18):
    deadline = time.monotonic() + min(timeout, 18)
    stdin, stdout, stderr = client.exec_command('/bin/bash -se', timeout=timeout)
    channel = stdout.channel
    try:
        channel.settimeout(max(0.001, deadline - time.monotonic()))
        stdin.write(script)
        stdin.channel.shutdown_write()
        chunks=[]
        size=0
        while True:
            if time.monotonic() >= deadline:
                raise TimeoutError('finite SSH command budget exhausted')
            if channel.recv_ready():
                chunk=channel.recv(65536); chunks.append(chunk); size+=len(chunk)
                if size > 65536:
                    raise RuntimeError('checked SSH output exceeds finite bound')
            if channel.recv_stderr_ready():
                channel.recv_stderr(65536)  # Drain without retaining provider errors.
            if channel.exit_status_ready() and not channel.recv_ready() and not channel.recv_stderr_ready():
                code=channel.recv_exit_status()
                if code:
                    raise RuntimeError('remote checked operation failed; output suppressed')
                return b''.join(chunks).decode('utf-8', 'replace')
            time.sleep(min(0.02, max(0.001, deadline - time.monotonic())))
    finally:
        channel.close()


def pin(client):
    public = remote_command(client, 'cat /etc/ssh/ssh_host_ecdsa_key.pub\n').split()
    if len(public) < 2 or public[0] != 'ecdsa-sha2-nistp256':
        raise RuntimeError('expected previously verified ECDSA public host key unavailable')
    return 'SHA256:' + base64.b64encode(hashlib.sha256(base64.b64decode(public[1], validate=True)).digest()).decode().rstrip('=')


def cleanup(client, identity):
    # No glob deletion, symlink traversal, rm -r, process enumeration or service stop.
    # Exact .owner and marker bytes must match this invocation before removal.
    script = """import hashlib,json,os,pathlib,stat
identity=IDENTITY
root=pathlib.Path('/opt/pantheon-p0-20261109')
target=root/identity
assert target.parent==root and target.name==identity
assert not target.is_symlink() and target.resolve()==target and root.resolve()==root
owner=target/'.owner'
assert owner.is_file() and not owner.is_symlink() and owner.read_text()==identity
allowed={'.owner','started','gate','done'}
entries=list(target.iterdir());assert {p.name for p in entries}<=allowed
receipts=[]
for p in entries:
 assert not p.is_symlink() and stat.S_ISREG(p.stat().st_mode)
 raw=p.read_bytes();assert raw==identity.encode()
 receipts.append({'file':p.name,'sha256':hashlib.sha256(raw).hexdigest(),'bytes':len(raw)})
for p in entries:p.unlink()
target.rmdir()
print(json.dumps({'phase':'exact-owned-cleanup','directory':str(target),'files':receipts,'absent':not target.exists(),'processKill':False}))
""".replace('IDENTITY', repr(identity))
    # Existing Python3 only; no installation and no untrusted interpolation.
    out = remote_command(client, "python3 - <<'O2_CLEANUP'\n" + script + '\nO2_CLEANUP\n')
    return json.loads(out)


def preflight():
    raise RuntimeError("v9 remote preflight remains root-owned; review and reseal before enabling this entry")


def root_preflight_after_review():
    from remote import connect
    receipt = {'host': '115.190.251.143', 'scope': 'pinned read-only SSH; no remote mutation or DB creation'}
    try:
        with connect() as client:
            fingerprint = pin(client)
            remote_command(client, "test \"$(readlink -f /opt/pantheon-p0-20261109)\" = /opt/pantheon-p0-20261109\ncommand -v python3 >/dev/null\n")
            receipt.update(exitCode=0, pinned=True, hostFingerprint=fingerprint)
    except Exception as error:
        # errno13 is an execution permission signal; no blind repetition or bypass.
        receipt.update(exitCode=1, errorType=type(error).__name__, errno=getattr(error, 'errno', None),
                       remaining='root execution required; no true VM case attempted')
        request('execute', ['Preflight SSH failed in restricted tool; root executes through its approved tool',
                            'Existing local isolated MySQL33316 reachable', 'Build source hashes current'])
    write('preflight-result', receipt)
    print(json.dumps(receipt))
    return receipt['exitCode']


def execute(selected):
    raise RuntimeError("candidate execution remains root-owned; remove this guard only after root review and reseal")


def root_execute_after_review(selected):
    from remote import connect, load_credential
    import pymysql
    receipt = {'host': '115.190.251.143', 'origin': 'public current DeployService/default SSH on actual VM; independent SSH state and fresh OS child process',
               'run': secrets.token_hex(6), 'cases': [], 'positivePassed': 0, 'negativePassed': 0, 'failed': 0, 'driverFailures': 0,
               'approval': 'generator evidence only; no independent O2/P0 approval', 'artifacts': artifacts()}
    client = None
    started = time.monotonic()
    try:
        prior = json.loads((EVIDENCE / 'continue-o2-live-v10-build-ready-result.json').read_text())
        expected = json.loads((EVIDENCE / 'continue-o2-live-v10-build-ready-source-after.json').read_text())
        if not prior['sourceStable'] or source_hashes() != expected or artifacts() != prior['artifacts']:
            raise RuntimeError('build/source identity changed; rebuild required')
        client = connect()
        client.get_transport().set_keepalive(30)
        fingerprint = pin(client)
        row, secret = load_credential()
        key = secrets.token_hex(32)
        for case in selected:
            identity = 'o2-live-' + receipt['run'] + '-' + case
            assert re.fullmatch(r'o2-live-[a-z0-9-]{8,70}', identity)
            directory = '/opt/pantheon-p0-20261109/' + identity
            database = 'ops_p0_o2_live_' + receipt['run'] + '_' + case.replace('-', '_')
            assert re.fullmatch(r'ops_p0_o2_live_[a-z0-9_]{8,90}', database)
            current = {'case': case, 'identity': identity, 'database': database, 'databaseRetention': 'retained; no drop', 'events': []}
            receipt['cases'].append(current)
            case_started = time.monotonic()
            # Reserve 40s inside the 420s cold-case budget for exact local-child
            # termination (<=10s), remote-shell quiescence (9s) and SSH cleanup (18s).
            case_budget = 380 if case == 'cold-restart' else 130
            def remaining_budget(limit):
                remaining = case_budget - (time.monotonic() - case_started)
                if remaining <= 0:
                    raise TimeoutError('finite case budget exhausted')
                return min(limit, remaining)
            process = None
            created = False
            try:
                remote_command(client, f"test \"$(readlink -f /opt/pantheon-p0-20261109)\" = /opt/pantheon-p0-20261109\ntest ! -e '{directory}'\nmkdir -m 700 -- '{directory}'\nprintf '{identity}' > '{directory}/.owner'\n")
                created = True
                db = pymysql.connect(host='127.0.0.1', port=33316, user='root', connect_timeout=5, autocommit=True)
                try:
                    with db.cursor() as cursor:
                        cursor.execute('CREATE DATABASE `' + database + '` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci')
                finally:
                    db.close()
                cfg = {'DSN': 'root@tcp(127.0.0.1:33316)/' + database + '?parseTime=true&loc=Local',
                       'Key': key, 'User': row['username'], 'AuthMode': row['auth_mode'], 'Secret': secret,
                       'Fingerprint': fingerprint, 'Identity': identity, 'Case': case, 'Mode': 'crash-start' if case == 'cold-restart' else 'matrix'}
                process = subprocess.Popen([str(LOCAL / 'continue-o2-live-v10-driver.exe')], stdin=subprocess.PIPE,
                                           stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding='utf-8')
                output, errors = process.communicate(json.dumps(cfg) + '\n', timeout=remaining_budget(130))
                current.update(driverExitCode=process.returncode, stderrPresent=bool(errors))
                if case == 'cold-restart':
                    initial = [json.loads(line) for line in output.splitlines()]
                    ready = [e for e in initial if e.get('phase') == 'actual-process-crash-ready']
                    if process.returncode != 75 or len(ready) != 1:
                        # Persist the same safe driver events before rejecting missing crash readiness.
                        current['events'].extend(initial)
                        current['runnerFailurePhase'] = 'initial-crash-start-ready-validation'
                        current['initialDriverOutputRetained'] = True
                        raise RuntimeError('actual owned process crash did not reach persisted active lease')
                    current['deliberateCrashExitCode'] = 75
                    cfg['TaskID'] = ready[0]['taskId']
                    cfg['AttemptID'] = ready[0]['attemptId']
                    cfg['AttemptNo'] = ready[0]['attemptNo']
                    cfg['LeaseExpiresUnix'] = ready[0]['leaseExpiresUnix']
                    cfg['Mode'] = 'queue-probe'
                    process = subprocess.Popen([str(LOCAL / 'continue-o2-live-v10-driver.exe')], stdin=subprocess.PIPE,
                                               stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding='utf-8')
                    probe, probe_errors = process.communicate(json.dumps(cfg) + '\n', timeout=remaining_budget(35))
                    current['activeLeaseProbeExitCode'] = process.returncode
                    output += probe
                    # Let the actual interrupted remote shell exhaust its finite
                    # loop before enabling a marker for the recovered attempt.
                    time.sleep(10)
                    remote_command(client, f"test ! -f '{directory}/done'\nprintf '{identity}' > '{directory}/gate'\n")
                    deadline = ready[0]['leaseExpiresUnix'] + 2
                    if deadline - time.time() > 310:
                        raise RuntimeError('recorded actual lease exceeds finite five-minute budget')
                    while time.time() < deadline:
                        remaining = deadline - time.time()
                        print(json.dumps({'phase': 'actual-lease-wait', 'case': case, 'remainingSeconds': round(remaining)}), flush=True)
                        time.sleep(min(30, remaining, remaining_budget(30)))
                    cfg['Mode'] = 'recover'
                    process = subprocess.Popen([str(LOCAL / 'continue-o2-live-v10-driver.exe')], stdin=subprocess.PIPE,
                                               stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding='utf-8')
                    recovered, recovered_errors = process.communicate(json.dumps(cfg) + '\n', timeout=remaining_budget(65))
                    output += recovered
                    current.update(driverExitCode=process.returncode, recoveryExitCode=process.returncode,
                                   stderrPresent=bool(errors or probe_errors or recovered_errors),
                                   realLeaseWait=True, leaseMutation=False)
                    if current['activeLeaseProbeExitCode']:
                        current['activeLeaseProbeFailed'] = True
                cfg.clear()
                for line in output.splitlines():
                    try:
                        event = json.loads(line)
                    except ValueError:
                        current['unexpectedOutput'] = True
                        continue
                    if not isinstance(event, dict) or not ({'phase', 'check'} & event.keys()):
                        current['unexpectedOutput'] = True
                        continue
                    current['events'].append(event)
                    if event.get('phase') == 'driver-failure':
                        receipt['driverFailures'] += 1
                    if 'check' in event:
                        if event.get('passed') is True:
                            field = 'negativePassed' if event['kind'] == 'negative' else 'positivePassed'
                            receipt[field] += 1
                        else:
                            receipt['failed'] += 1
                    print(json.dumps(event), flush=True)
                summaries = [e for e in current['events'] if e.get('phase') == 'driver-result']
                if not summaries or any(e.get('failed') != 0 for e in summaries):
                    current['missingOrFailedDriverSummary'] = True
                if not any('check' in e for e in current['events']):
                    current['missingAssertions'] = True
                if process.returncode or current.get('unexpectedOutput') or current.get('activeLeaseProbeFailed') or current.get('missingOrFailedDriverSummary') or current.get('missingAssertions') or any(e.get('passed') is False for e in current['events'] if 'check' in e) or any(e.get('phase') == 'driver-failure' for e in current['events']):
                    current['status'] = 'failed'
                else:
                    current['status'] = 'passed'
            except Exception as error:
                current.update(status='blocked', errorType=type(error).__name__, errno=getattr(error, 'errno', None))
                # Kill only our exact Popen object, never a discovered PID/process name.
                if process and process.poll() is None:
                    process.kill()
                    process.wait(timeout=10)
                    current['ownedLocalDriverTerminated'] = True
            finally:
                if created:
                    try:
                        # All owned scripts have <=8-second finite loops. Wait after
                        # termination so cleanup cannot race a disconnected remote shell.
                        time.sleep(9)
                        current['cleanup'] = cleanup(client, identity)
                    except Exception as error:
                        current['cleanup'] = {'failed': True, 'errorType': type(error).__name__, 'retainedExactDirectory': directory}
                write('case-' + case + '-' + receipt['run'], current)
                print(json.dumps({'phase': 'case-finished', 'case': case, 'status': current.get('status'),
                                  'driverExitCode': current.get('driverExitCode'), 'cleanup': current.get('cleanup')}), flush=True)
    except Exception as error:
        receipt.update(blocker={'errorType': type(error).__name__, 'errno': getattr(error, 'errno', None)})
        request('execute', ['Resolve recorded execution blocker without bypass', 'Existing MySQL33316 reachable', 'Build hashes match current source'])
    finally:
        if client:
            client.close()
        receipt.update(seconds=round(time.monotonic() - started, 3), sourceStable=source_hashes() == locals().get('expected', {}))
        receipt['exitCode'] = 0 if len(receipt['cases']) == len(selected) and all(c.get('status') == 'passed' and c.get('cleanup', {}).get('absent') for c in receipt['cases']) and receipt['sourceStable'] and receipt['failed'] == 0 and receipt['driverFailures'] == 0 else 1
        write('matrix-result-' + receipt['run'], receipt)
        print(json.dumps({'phase': 'matrix-result', 'run': receipt['run'], 'positivePassed': receipt['positivePassed'],
                          'negativePassed': receipt['negativePassed'], 'failed': receipt['failed'], 'driverFailures': receipt['driverFailures'], 'failedCases': sum(c.get('status') != 'passed' for c in receipt['cases']), 'casesAttempted': len(receipt['cases']),
                          'exitCode': receipt['exitCode'], 'sourceStable': receipt['sourceStable']}), flush=True)
    return receipt['exitCode']


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--phase', choices=('build', 'preflight', 'execute'), required=True)
    parser.add_argument('--case', action='append', choices=CASES)
    args = parser.parse_args()
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    if args.phase == 'build':
        return build()
    if args.phase == 'preflight':
        return preflight()
    return execute(args.case or CASES)


if __name__ == '__main__':
    raise SystemExit(main())
