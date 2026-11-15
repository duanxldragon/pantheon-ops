"""Fresh, isolated P0 runtime evidence; never replay previous acceptance attempts."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import secrets
import socket
import subprocess
import time
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
EVIDENCE = ROOT / '.harness/evidence/2026-11-14-ops-latest-foundation-and-plans/verification/current-p0'
FIXTURE = ROOT / '.local/ops-acceptance/o4-final-gates-fixture-v2-20261111-r1/private/fixture-env-v2.json'
FIXTURE_SHA = '8796d1a6c2b599e8e5406df8848638287d93827fb4436e6373c271b79cca5f05'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8')


def source_map():
    return {str(p.relative_to(ROOT)).replace('\\', '/'): digest(p)
            for p in (ROOT / 'backend').rglob('*')
            if p.is_file() and p.suffix in ('.go', '.sql', '.mod', '.sum')}


def fixture():
    if digest(FIXTURE) != FIXTURE_SHA:
        raise RuntimeError('Private fixture identity changed')
    return json.loads(FIXTURE.read_text(encoding='utf-8'))


def child_env(private):
    env = dict(os.environ, **fixture())
    env.update(CGO_ENABLED='1', GOWORK='off')
    scratch = private / 'go-temp'
    scratch.mkdir()
    env.update(TEMP=str(scratch), TMP=str(scratch), TMPDIR=str(scratch))
    env['PATH'] = 'D:/msys64/mingw64/bin;' + env.get('PATH', '')
    return env


def command(name, args, private, env, timeout):
    started = time.monotonic()
    output = private / (name + '.stdout.log')
    errors = private / (name + '.stderr.log')
    before = source_map()
    code, failure = None, None
    with output.open('xb') as out, errors.open('xb') as err:
        try:
            process = subprocess.Popen(args, cwd=ROOT / 'backend', env=env,
                                       stdin=subprocess.DEVNULL, stdout=out, stderr=err,
                                       creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
            write(private / (name + '.process.json'), {'pid': process.pid, 'command': args})
            try:
                code = process.wait(timeout=timeout)
            except subprocess.TimeoutExpired:
                process.kill()
                code = process.wait()
                failure = 'timeout; child stopped; logs retained'
        except OSError as exc:
            failure = type(exc).__name__
    after = source_map()
    changed = [p for p in before.keys() | after.keys() if before.get(p) != after.get(p)]
    events = []
    if '-json' in args:
        for line in output.read_text(encoding='utf-8', errors='replace').splitlines():
            try:
                event = json.loads(line)
            except ValueError:
                continue
            if event.get('Action') in ('pass', 'fail', 'skip'):
                events.append({k: event[k] for k in ('Action', 'Package', 'Test') if k in event})
    receipt = {'command': args, 'exitCode': code, 'failure': failure,
               'elapsedSeconds': round(time.monotonic() - started, 3),
               'sourceStable': not changed, 'changedSources': changed,
               'sourceCount': len(before), 'fixtureSHA256': FIXTURE_SHA,
               'stdoutSHA256': digest(output), 'stderrSHA256': digest(errors),
               'privateLogs': True, 'credentialsPrinted': False,
               'testPasses': sum(e['Action'] == 'pass' and 'Test' in e for e in events),
               'packagePasses': sum(e['Action'] == 'pass' and 'Test' not in e for e in events),
               'skipped': [e for e in events if e['Action'] == 'skip'],
               'failed': [e for e in events if e['Action'] == 'fail']}
    write(private / (name + '.source-before.json'), before)
    write(private / (name + '.source-after.json'), after)
    write(EVIDENCE / (private.name + '-' + name + '.json'), receipt)
    print(json.dumps(receipt))
    return code == 0 and failure is None


def remote_readonly(private):
    module_path = ROOT / '.local/ops-acceptance/remote.py'
    spec = importlib.util.spec_from_file_location('current_p0_remote', module_path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    receipt = {'host': '115.190.251.143', 'remoteMutations': 0, 'credentialsPrinted': False}
    try:
        with module.connect() as client:
            _, output, errors = client.exec_command('uname -s; command -v kubectl; command -v docker; ss -ltn', timeout=20)
            stdout = output.read(256 << 10)
            stderr = errors.read(64 << 10)
            receipt.update(connected=True, exitCode=output.channel.recv_exit_status())
            (private / 'remote-readonly.stdout.log').write_bytes(stdout)
            (private / 'remote-readonly.stderr.log').write_bytes(stderr)
            receipt['stdoutSHA256'] = digest(private / 'remote-readonly.stdout.log')
    except Exception as exc:
        receipt.update(connected=False, errorType=type(exc).__name__, winerror=getattr(exc, 'winerror', None))
    write(EVIDENCE / (private.name + '-remote-readonly.json'), receipt)
    print(json.dumps(receipt))
    return receipt.get('connected') and receipt.get('exitCode') == 0


def provider_readonly(private):
    receipt = {'endpoint': 'http://115.190.251.143:9090/api/v1/query?query=vector%281%29',
               'method': 'GET', 'remoteMutations': 0, 'credentialsPrinted': False,
               'coverage': 'provider-readonly-preflight; not composed-product acceptance'}
    try:
        with urllib.request.urlopen(receipt['endpoint'], timeout=15) as response:
            payload = response.read((1 << 20) + 1)
            if len(payload) > 1 << 20:
                raise RuntimeError('Provider response budget exceeded')
            (private / 'provider-readonly.json').write_bytes(payload)
            envelope = json.loads(payload)
            receipt.update(httpStatus=response.status, responseSHA256=hashlib.sha256(payload).hexdigest(),
                           providerSuccess=envelope.get('status') == 'success',
                           vectorOne=any(row.get('value', [None, None])[1] == '1'
                                         for row in envelope.get('data', {}).get('result', [])))
    except Exception as exc:
        receipt.update(providerSuccess=False, errorType=type(exc).__name__)
    write(EVIDENCE / (private.name + '-provider-readonly.json'), receipt)
    print(json.dumps(receipt))
    return receipt.get('httpStatus') == 200 and receipt.get('providerSuccess') and receipt.get('vectorOne')


def start_public(private, executable, manifest):
    import pymysql
    executable, manifest = Path(executable).resolve(), Path(manifest).resolve()
    if not executable.is_relative_to(ROOT / '.tmp') or not executable.is_file():
        raise RuntimeError('Only a task-built executable is accepted')
    pins = json.loads(manifest.read_text(encoding='utf-8'))
    if pins != source_map():
        raise RuntimeError('Built source map is not the current product source')
    expected_migration = max(int(path.name.split('_', 1)[0])
                             for path in (ROOT / 'backend/pkg/database/migrations').glob('*.up.sql'))
    values = fixture()
    match = re.fullmatch(r'([^:]+):([^@]*)@tcp\(([^:]+):(\d+)\)/([^?]+)\?.*', values['PANTHEON_TEST_DSN'])
    if not match:
        raise RuntimeError('Unsupported isolated fixture DSN')
    user, password, host, port, _ = match.groups()
    schema = 'ops_baseline_test_p0_' + secrets.token_hex(6)
    if host != '127.0.0.1':
        raise RuntimeError('Nonlocal database refused')
    def free_port():
        with socket.socket() as probe:
            probe.bind(('127.0.0.1', 0))
            return probe.getsockname()[1]
    api_port, redis_port = free_port(), free_port()
    database = pymysql.connect(host=host, port=int(port), user=user, password=password,
                               autocommit=True, connect_timeout=5)
    with database.cursor() as cursor:
        cursor.execute('SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name=%s', (schema,))
        if cursor.fetchone()[0]:
            raise RuntimeError('Exclusive database name already exists')
        cursor.execute('CREATE DATABASE ' + schema + ' CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci')
    admin_password, redis_password = secrets.token_urlsafe(24) + '!Aa1', secrets.token_urlsafe(32)
    env = dict(os.environ)
    env.update(PANTHEON_DSN=f'{user}:{password}@tcp({host}:{port})/{schema}?charset=utf8mb4&parseTime=True&loc=UTC',
               PANTHEON_ENV='development', GIN_MODE='release', PANTHEON_PORT=str(api_port),
               PANTHEON_AUTO_MIGRATE='false', PANTHEON_REDIS_ADDR=f'127.0.0.1:{redis_port}',
               PANTHEON_REDIS_PASSWORD=redis_password, PANTHEON_REDIS_REQUIRED='true',
               PANTHEON_INITIAL_ADMIN_PASSWORD=admin_password, PANTHEON_ENABLE_DYNAMIC_MODULES='false',
               PANTHEON_BUSINESS_SECRET_KEY=secrets.token_hex(32), PANTHEON_DEPLOY_CREDENTIAL_KEY=secrets.token_hex(32))
    for key in ('ACCESS_TOKEN_SECRET', 'REFRESH_TOKEN_SECRET', 'OP_TOKEN_SECRET', 'SETTING_SECRET', 'MFA_SECRET'):
        env['PANTHEON_' + key] = secrets.token_urlsafe(48)
    write(private / 'environment.json', {k: v for k, v in env.items() if k.startswith('PANTHEON_')})
    process_handles = []
    def launch(name, command_args, cwd):
        out, err = (private / (name + '.stdout.log')).open('xb'), (private / (name + '.stderr.log')).open('xb')
        process = subprocess.Popen(command_args, cwd=cwd, env=env, stdin=subprocess.DEVNULL,
                                   stdout=out, stderr=err, creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
        process_handles.append((process, out, err))
        write(private / (name + '.process.json'), {'pid': process.pid, 'executable': command_args[0]})
        return process
    redis_binary = ROOT / '.tmp/ops-20261114-backup/redis-relocated/redis-server.exe'
    receipt = {'schema': schema, 'apiBaseUrl': f'http://127.0.0.1:{api_port}/api/v1',
               'redisPort': redis_port, 'executableSHA256': digest(executable),
               'sourceCount': len(pins), 'sourceStable': True, 'autoMigrate': False,
               'privateDirectory': str(private), 'previousServicesChanged': False,
               'credentialsPrinted': False, 'status': 'starting'}
    try:
        redis = launch('redis', [str(redis_binary), '--bind', '127.0.0.1', '--port', str(redis_port),
                                '--requirepass', redis_password, '--save', '', '--appendonly', 'no'], redis_binary.parent)
        deadline = time.monotonic() + 20
        while time.monotonic() < deadline:
            if redis.poll() is not None:
                raise RuntimeError('Owned Redis startup failed')
            with socket.socket() as probe:
                if probe.connect_ex(('127.0.0.1', redis_port)) == 0:
                    break
            time.sleep(.25)
        else:
            raise RuntimeError('Owned Redis startup timeout')
        api = launch('api', [str(executable)], ROOT / 'backend')
        receipt.update(apiPID=api.pid, redisPID=redis.pid)
        deadline = time.monotonic() + 120
        while time.monotonic() < deadline:
            if api.poll() is not None:
                raise RuntimeError('Native API startup failed')
            if sum(p.stat().st_size for p in private.glob('*.log')) > 16 << 20:
                raise RuntimeError('Startup log budget exceeded')
            try:
                with urllib.request.urlopen(receipt['apiBaseUrl'] + '/health', timeout=2) as response:
                    if response.status == 200:
                        break
            except OSError:
                time.sleep(.5)
        else:
            raise RuntimeError('Native API startup timeout')
        with database.cursor() as cursor:
            cursor.execute('SELECT version, dirty FROM ' + schema + '.schema_migrations')
            version, dirty = cursor.fetchone()
        if dirty or version != expected_migration:
            raise RuntimeError('Formal migrations not ready')
        receipt['sourceStable'] = pins == source_map()
        if not receipt['sourceStable']:
            raise RuntimeError('Source changed during native API startup')
        receipt.update(status='ready', migrationVersion=version, expectedMigrationVersion=expected_migration, migrationDirty=bool(dirty))
        write(private / 'public-config.json', {'apiBaseUrl': receipt['apiBaseUrl'], 'platformAdmin': {'username': 'admin', 'password': admin_password},
                                              'providerUrl': 'http://115.190.251.143:9090', 'executable': str(executable),
                                              'executableSHA256': digest(executable), 'sourceManifest': str(manifest),
                                              'sourceManifestSHA256': digest(manifest), 'runId': private.name.replace('-', '_'),
                                              'apiPID': api.pid, 'redisPID': redis.pid, 'schema': schema,
                                              'runtimeReceipt': str(EVIDENCE / (private.name + '-public-start.json'))})
    except Exception as exc:
        receipt.update(status='failed', errorType=type(exc).__name__, failure=str(exc))
        for process, _, _ in reversed(process_handles):
            if process.poll() is None:
                process.kill()
                process.wait()
        receipt['ownedProcessesStopped'] = True
    finally:
        database.close()
        for _, out, err in process_handles:
            out.close()
            err.close()
        write(EVIDENCE / (private.name + '-public-start.json'), receipt)
        print(json.dumps(receipt))
    return receipt['status'] == 'ready'


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('mode', choices=('sql-race', 'observability-race', 'build', 'remote-readonly', 'provider-readonly', 'envelope-oracle', 'public-start'))
    parser.add_argument('--timeout', type=int, default=1800)
    parser.add_argument('--executable')
    parser.add_argument('--source-manifest')
    args = parser.parse_args()
    private = ROOT / '.tmp' / ('current-p0-' + time.strftime('%Y%m%dT%H%M%S') + '-' + secrets.token_hex(4))
    private.mkdir(parents=True)
    write(private / 'attempt.json', {'mode': args.mode, 'run': private.name, 'startedAt': time.time()})
    if args.mode == 'remote-readonly':
        success = remote_readonly(private)
    elif args.mode == 'provider-readonly':
        success = provider_readonly(private)
    elif args.mode == 'public-start':
        if not args.executable or not args.source_manifest:
            raise RuntimeError('Executable and complete build source manifest required')
        success = start_public(private, args.executable, args.source_manifest)
    else:
        env = child_env(private)
        go = 'D:/Go/bin/go.exe'
        if args.mode == 'envelope-oracle':
            success = command('envelope-oracle', [go, 'test', '-race', '-p=1', '-count=1', '-buildvcs=false', '-mod=readonly', '-json',
                                                 '-run', '^TestDatasourceLiveEnvelopeMatchesNativeSuccess$', './modules/business/observability'],
                              private, env, args.timeout)
        elif args.mode in ('sql-race', 'observability-race'):
            packages = ['./modules/business/observability']
            if args.mode == 'sql-race':
                packages.extend(('./modules/business/k8s', './modules/business/opsinit'))
            success = command(args.mode, [go, 'test', '-race', '-p=1', '-count=1', '-buildvcs=false', '-mod=readonly', '-json', *packages],
                              private, env, args.timeout)
        else:
            success = command('build', [go, 'build', '-buildvcs=false', '-mod=readonly', '-o', str(private / 'pantheon-ops.exe'), './cmd/server'],
                              private, env, args.timeout)
            if success:
                write(EVIDENCE / (private.name + '-executable.json'), {'path': str(private / 'pantheon-ops.exe'), 'sha256': digest(private / 'pantheon-ops.exe')})
    return 0 if success else 1


if __name__ == '__main__':
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(json.dumps({'failed': True, 'errorType': type(exc).__name__, 'credentialsPrinted': False}))
        raise SystemExit(1)
