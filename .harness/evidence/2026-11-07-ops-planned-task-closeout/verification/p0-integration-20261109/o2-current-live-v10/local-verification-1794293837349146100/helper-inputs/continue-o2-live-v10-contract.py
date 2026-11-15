"""Finite local input discovery and receipt checks. No credentials or network."""
from pathlib import Path
import ast
import hashlib
import importlib.util
import json
import os
import re
import subprocess
import sys

sys.dont_write_bytecode = True
LOCAL = Path(__file__).resolve().parent
ROOT = LOCAL.parents[1]
PARENT = ROOT / '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'
EV = PARENT / 'o2-current-live-v10'
OWNER = PARENT / 'o2-service-transaction-20261110'
PREFIX = 'continue-o2-live-v10-'
TASK = '2026-11-07-ops-planned-task-closeout'
GO_ROOT = Path('D:/cache/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.windows-amd64')
GO = GO_ROOT / 'bin/go.exe'
GOPLS = Path('D:/Go/bin/gopls.exe')
CASES = ('success', 'failure-retry', 'pause-resume', 'cancel', 'concurrent',
         'revoke-credential', 'revoke-rotation', 'revoke-source', 'revoke-host',
         'start-negative', 'hostkey-negative', 'cold-restart')
HELPERS = ('contract.py', 'prepare.py', 'driver.go', 'driver_test.go', 'transport_test.go',
           'runner.py', 'verify.py', 'seal.py', 'root-execute.py', 'contract_test.py')
FILE_FIELDS = ('GoFiles', 'CgoFiles', 'CFiles', 'CXXFiles', 'MFiles', 'HFiles',
               'FFiles', 'SFiles', 'SwigFiles', 'SwigCXXFiles', 'SysoFiles',
               'EmbedFiles', 'TestGoFiles', 'XTestGoFiles', 'TestEmbedFiles', 'XTestEmbedFiles')


def require(ok, reason):
    if not ok:
        raise RuntimeError(reason)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def relative(path):
    return Path(path).resolve().relative_to(ROOT.resolve()).as_posix()


def read(path):
    return json.loads(Path(path).read_text(encoding='utf-8'))


def write(path, value):
    with Path(path).open('x', encoding='utf-8', newline='\n') as stream:
        json.dump(value, stream, indent=2, sort_keys=True)
        stream.write('\n')


def load(suffix):
    path = LOCAL / (PREFIX + suffix)
    spec = importlib.util.spec_from_file_location('v10_' + suffix.replace('.', '_'), path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def environment(platform='windows'):
    env = os.environ.copy()
    env.update(PATH='D:/msys64/mingw64/bin;' + str(GO_ROOT / 'bin') + ';' + env['PATH'],
               GOROOT=str(GO_ROOT), GOTOOLCHAIN='local', CGO_ENABLED='1' if platform == 'windows' else '0',
               GOOS=platform, GOARCH='amd64', GOMAXPROCS='2',
               GOCACHE=str(LOCAL / (PREFIX + 'gocache')), GOFLAGS='', GOWORK='off',
               GOPROXY='off', GOSUMDB='off')
    # No inherited acceptance/test DSN is used by these named-file offline tests.
    for name in tuple(env):
        if name.endswith('_DSN') or name == 'PANTHEON_DEPLOY_CREDENTIAL_KEY':
            env.pop(name)
    return env


def protected_paths():
    # Two direct directories only; never recurse into .local/.tmp or credentials.
    paths = {p for directory in (LOCAL, PARENT) for p in directory.iterdir()
             if p.is_file() and re.search(r'(?:resume-o2-v[789]|remote-web-v9)', p.name)}
    paths.update(LOCAL / name for name in ('remote.py', 'known_hosts'))
    # Historical inventories are protected documents, not current source pins.
    for name in ('resume-o2-v9-preserved-before.json', 'resume-o2-v9-continuation-preserved-before.json'):
        paths.update(ROOT / key for key in read(PARENT / name)
                     if key.startswith(('.local/', '.harness/')))
    return sorted(paths)


def hashes(paths):
    return {relative(path): digest(path) for path in paths}


def check_hashes(pins):
    require(isinstance(pins, dict) and bool(pins), 'empty/malformed pins')
    for name, expected in pins.items():
        path = (ROOT / name).resolve()
        require(path.is_relative_to(ROOT.resolve()) and path.is_file()
                and re.fullmatch(r'[0-9a-f]{64}', expected) is not None
                and digest(path) == expected, 'protected/source/artifact/evidence drift')


def protected():
    pins = read(EV / 'protected-old-inputs.json')
    require(pins == hashes(protected_paths()), 'protected OLD inputs drift or omission')
    return pins


def artifacts():
    names = [PREFIX + suffix for suffix in HELPERS] + [PREFIX + 'driver.exe', PREFIX + 'driver-linux']
    return {name: digest(LOCAL / name) for name in names}


def json_rows(raw):
    decoder = json.JSONDecoder()
    while raw.strip():
        row, end = decoder.raw_decode(raw.lstrip())
        yield row
        raw = raw.lstrip()[end:]


def inventory(archive=None):
    """Discover actual named-file tests, linked owner package tests and Linux build.

    External module/stdlib bytes are included; there is no old count/map fallback.
    go list logs are retained when requested. Inventory results are local inputs,
    never cases or runtime evidence.
    """
    require(GO.is_file(), 'exact installed toolchain missing')
    paths = {ROOT / 'backend/go.mod', ROOT / 'backend/go.sum'}
    tooling = {GO, GO_ROOT / 'bin/gofmt.exe', GOPLS, Path(sys.executable),
               Path('D:/msys64/mingw64/bin/gcc.exe'), Path('D:/msys64/usr/bin/bash.exe')}
    tooling.update(GO_ROOT / 'pkg/tool/windows_amd64' / (name + '.exe')
                   for name in ('compile', 'link', 'asm', 'cgo', 'vet'))
    plans = [('windows', [str(LOCAL / (PREFIX + suffix))
                           for suffix in ('driver.go', 'driver_test.go', 'transport_test.go')]),
             ('windows', ['./modules/business/deploy', './modules/business/service', './modules/business/capability']),
             ('linux', [str(LOCAL / (PREFIX + 'driver.go'))])]
    discoveries = []
    for index, (platform, inputs) in enumerate(plans):
        command = [str(GO), 'list', '-buildvcs=false', '-mod=readonly', '-deps', '-test', '-json', *inputs]
        result = subprocess.run(command, cwd=ROOT / 'backend', env=environment(platform),
                                capture_output=True, timeout=180)
        if archive is not None:
            with (archive / f'go-list-{index}.json-stream').open('xb') as stream:
                stream.write(result.stdout)
            with (archive / f'go-list-{index}.stderr.txt').open('xb') as stream:
                stream.write(result.stderr)
        require(result.returncode == 0, 'current go list dependency discovery failed (no fallback)')
        discovered = []
        for row in json_rows(result.stdout.decode('utf-8')):
            require(not row.get('Error') and not row.get('DepsErrors'), 'go list dependency errors')
            discovered.append(row['ImportPath'])
            directory = Path(row['Dir']).resolve()
            for field in FILE_FIELDS:
                for name in row.get(field, []):
                    paths.add((directory / name).resolve())
            module = row.get('Module', {})
            for entry in (module, module.get('Replace', {})):
                if entry.get('GoMod'):
                    paths.add(Path(entry['GoMod']).resolve())
        discoveries.append({'platform': platform, 'command': command, 'packages': sorted(set(discovered))})
    env_names = ['GOOS', 'GOARCH', 'CGO_ENABLED', 'GOVERSION', 'GOROOT', 'GOMOD',
                 'GOWORK', 'GOFLAGS', 'GOEXPERIMENT', 'GOAMD64', 'CC', 'CXX']
    result = subprocess.run([str(GO), 'env', '-json', *env_names], cwd=ROOT / 'backend',
                            env=environment(), capture_output=True, timeout=30)
    require(result.returncode == 0, 'Go build environment unavailable')
    # Native compiler auxiliaries actually selected by MinGW.
    for option in ('-print-prog-name=cc1', '-print-prog-name=collect2', '-print-prog-name=as', '-print-prog-name=ld'):
        result_cc = subprocess.run(['D:/msys64/mingw64/bin/gcc.exe', option],
                                   env=environment(), capture_output=True, timeout=30)
        require(result_cc.returncode == 0, 'MinGW tool inventory failed')
        name = result_cc.stdout.decode().strip()
        path = Path(name)
        if not path.is_absolute():
            path = Path('D:/msys64/mingw64/bin') / (name if name.endswith('.exe') else name + '.exe')
        tooling.add(path.resolve())
    def key(path):
        path = path.resolve()
        return 'repo:' + relative(path) if path.is_relative_to(ROOT.resolve()) else 'external:' + path.as_posix()
    return {'schema': 'o2-current-dependencies-v10', 'discovery': discoveries,
            'environment': json.loads(result.stdout),
            'files': {key(path): digest(path) for path in sorted(paths)},
            'toolFiles': {key(path): digest(path) for path in sorted(tooling)}}


def invariant_checks():
    old = (LOCAL / 'resume-o2-v9-driver.go').read_text()
    new = (LOCAL / (PREFIX + 'driver.go')).read_text()
    require(old.replace('resume-o2-v9', PREFIX.rstrip('-')) == new, 'driver changes exceed version filenames')
    assertions = lambda source: re.findall(r'^\s*r\.check\(.*$', source, re.M)
    require(assertions(old) == assertions(new) and len(assertions(new)) == 45, 'original assertions changed')
    for suffix in ('driver_test.go', 'transport_test.go'):
        original = (LOCAL / ('resume-o2-v9-' + suffix)).read_text()
        expected = original.replace('resume-o2-v9', PREFIX.rstrip('-'))
        require((LOCAL / (PREFIX + suffix)).read_text() == expected, 'original tests/negative assertions changed')
    old_tree = ast.parse((LOCAL / 'resume-o2-v9-runner.py').read_text())
    new_tree = ast.parse((LOCAL / (PREFIX + 'runner.py')).read_text())
    for name in ('cleanup', 'remote_command', 'pin', 'root_execute_after_review'):
        find = lambda tree: next(n for n in tree.body if isinstance(n, ast.FunctionDef) and n.name == name)
        original = ast.parse(ast.unparse(find(old_tree)).replace('resume-o2-v9', PREFIX.rstrip('-'))).body[0]
        require(ast.dump(original) == ast.dump(find(new_tree)), 'original matrix/cleanup/transport changed')
    require(load('runner.py').CASES == CASES, 'case order changed')


def review(receipt_path, role, verdict, subject):
    """New hash-bound receipts; old prose/exit0 receipts cannot authorize v10."""
    path = Path(receipt_path).resolve()
    require(path.is_relative_to(PARENT.resolve()), 'review receipt outside parent evidence')
    value = read(path)
    require(value.get('schema') == 'o2-v10-review-v1' and value.get('role') == role
            and value.get('verdict') == verdict and value.get('exitCode') == 0
            and value.get('subjectSHA256') == digest(subject), 'bad prior review/subject receipt')
    report = (ROOT / value['report']).resolve()
    require(report.is_relative_to(PARENT.resolve()) and digest(report) == value['reportSHA256'], 'review report hash mismatch')
    require(re.search(r'(?m)^' + re.escape(verdict) + r'\s*$', report.read_text(encoding='utf-8')), 'review verdict missing')
    return {relative(path): digest(path), relative(report): digest(report)}


def owner_gate(code_path, arch_path):
    handoff = OWNER / 'handoff.md'
    require(handoff.is_file(), 'final O2 Service source-stable handoff missing; UNSEALED')
    pins = {relative(handoff): digest(handoff)}
    for receipt, role, verdict in ((code_path, 'code-reviewer', 'CODE APPROVE'),
                                  (arch_path, 'architect', 'ARCH CLEAR')):
        require(Path(receipt).resolve().parent == OWNER.resolve(), 'final owner review must belong to Service transaction lane')
        pins.update(review(receipt, role, verdict, handoff))
        value = read(receipt)
        require(value.get('sourceStable') is True, 'owner source not final/stable')
        source = value.get('sourceHashes', {})
        check_hashes(source)
        required = ('backend/modules/business/capability/capability.go',
                    'backend/modules/business/service/state.go', 'backend/modules/business/deploy/target_selection.go')
        require(all(name in source for name in required), 'owner review omits transaction inputs')
        pins.update(source)
    return pins


def validate_request(path):
    value = read(path)
    require(value.get('schema') == 'o2-current-request-v10' and value.get('state') == 'SEALED-CANDIDATE'
            and value.get('host') == '115.190.251.143' and value.get('allCases') == list(CASES)
            and value.get('assertionCallCount') == 45 and value.get('existingAssertionsUnchanged') is True,
            'UNSEALED or changed matrix scope')
    require('productSourceHashes' not in value and 'currentInventory' in value, 'old185 map substitution')
    for group in ('candidateHashes', 'evidenceHashes', 'protectedOldHashes', 'ownerPins'):
        check_hashes(value[group])
    require(value['protectedOldHashes'] == protected(), 'OLD protected pins differ')
    current = read(ROOT / value['currentInventory'])
    require(current.get('schema') == 'o2-current-dependencies-v10' and inventory() == current,
            'current source/tool/dependency discovery drift')
    invariant_checks()
    return value


def authorization(path, request_path):
    require(path is not None, 'execution disabled: independent CODE/ARCH + Dispatcher authorization receipt required')
    value = read(path)
    require(value.get('schema') == 'o2-v10-authorization-v1'
            and value.get('requestSHA256') == digest(request_path), 'authorization request hash mismatch')
    receipts = [('codeReceipt', 'code-reviewer', 'CODE APPROVE'),
                ('archReceipt', 'architect', 'ARCH CLEAR'),
                ('dispatcherReceipt', 'dispatcher', 'DISPATCHER AUTHORIZE')]
    pins = {}
    for field, role, verdict in receipts:
        reference = value[field]
        receipt = ROOT / reference['path']
        require(digest(receipt) == reference['sha256'], 'authorization receipt hash mismatch')
        pins.update(review(receipt, role, verdict, request_path))
    require(len({value[field]['path'] for field, _, _ in receipts}) == 3, 'independent review receipts required')
    return pins


def reserve_attempt(request_path):
    # Creation is the durable one-attempt boundary, before any credential import.
    path = EV / ('attempt-' + digest(request_path) + '.json')
    try:
        write(path, {'requestSHA256': digest(request_path), 'state': 'STARTED',
                     'retries': 0, 'reconnects': 0, 'replays': 0})
    except FileExistsError:
        raise RuntimeError('attempt replay rejected; preserve prior failure/attempt') from None
    return path
