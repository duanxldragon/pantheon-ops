"""Offline diagnostics/race/build; final builds require the owner final receipts."""
from pathlib import Path
import argparse
import ast
import importlib.util
import json
import subprocess
import sys
import time

sys.dont_write_bytecode = True
LOCAL = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('v10_verify_contract', LOCAL / 'continue-o2-live-v10-contract.py')
C = importlib.util.module_from_spec(spec)
spec.loader.exec_module(C)


def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument('--final', action='store_true')
    parser.add_argument('--owner-code', type=Path)
    parser.add_argument('--owner-arch', type=Path)
    args = parser.parse_args(argv)
    owner_pins = C.owner_gate(args.owner_code, args.owner_arch) if args.final else {}
    attempt = time.time_ns()
    archive = C.EV / ('local-verification-' + str(attempt))
    archive.mkdir()
    inputs = C.hashes(LOCAL / (C.PREFIX + suffix) for suffix in C.HELPERS)
    old = C.protected()
    C.invariant_checks()
    for suffix in C.HELPERS:
        if suffix.endswith('.py'):
            ast.parse((LOCAL / (C.PREFIX + suffix)).read_text())
    before_dir = archive / 'before'
    before_dir.mkdir()
    before = C.inventory(before_dir)
    C.write(archive / 'source-before.json', before)
    files = [str(LOCAL / (C.PREFIX + suffix)) for suffix in ('driver.go', 'driver_test.go', 'transport_test.go')]
    suffix = 'driver' if args.final else 'driver-transient-' + str(attempt)
    win = LOCAL / (C.PREFIX + suffix + '.exe')
    linux = LOCAL / (C.PREFIX + suffix + '-linux')
    C.require(not win.exists() and not linux.exists(), 'build outputs already exist; do not overwrite prior builds')
    commands = [([sys.executable, str(LOCAL / (C.PREFIX + 'contract_test.py'))], 'windows'),
                ([str(C.GO_ROOT / 'bin/gofmt.exe'), '-l', *files], 'windows'),
                ([str(C.GOPLS), 'check', *files], 'windows'),
                ([str(C.GO), 'vet', '-mod=readonly', *files], 'windows'),
                ([str(C.GO), 'test', '-mod=readonly', '-race', '-count=1', '-v', *files], 'windows'),
                ([str(C.GO), 'build', '-mod=readonly', '-race', '-o', str(win), files[0]], 'windows'),
                ([str(C.GO), 'build', '-mod=readonly', '-o', str(linux), files[0]], 'linux')]
    rows = []
    for index, (command, platform) in enumerate(commands):
        started = time.monotonic()
        try:
            result = subprocess.run(command, cwd=C.ROOT / 'backend', env=C.environment(platform),
                                    capture_output=True, timeout=360)
            raw, code = result.stdout + result.stderr, result.returncode
            if index == 1 and raw.strip():
                code = 1  # gofmt -l itself exits 0 for unformatted source.
        except subprocess.TimeoutExpired as error:
            raw, code = (error.stdout or b'') + (error.stderr or b''), 124
        log = archive / f'command-{index}.txt'
        with log.open('xb') as stream:
            stream.write(raw)
        row = {'command': command, 'cwd': C.relative(C.ROOT / 'backend'), 'exitCode': code,
               'log': C.relative(log), 'logSHA256': C.digest(log), 'seconds': round(time.monotonic() - started, 3),
               'GOOS': platform, 'CGO_ENABLED': C.environment(platform)['CGO_ENABLED'],
               'PATHPrefix': 'D:/msys64/mingw64/bin', 'GOCACHE': C.environment(platform)['GOCACHE']}
        rows.append(row)
        print(json.dumps({'index': index, 'exitCode': code, 'seconds': row['seconds']}), flush=True)
        if code:
            break
    after_dir = archive / 'after'
    after_dir.mkdir()
    after = C.inventory(after_dir)
    C.write(archive / 'source-after.json', after)
    stable = before == after and inputs == C.hashes(C.ROOT / name for name in inputs) and old == C.protected()
    if args.final:
        stable = stable and owner_pins == C.owner_gate(args.owner_code, args.owner_arch)
    failed = sum(row['exitCode'] != 0 for row in rows) + int(len(rows) != len(commands)) + int(not stable)
    receipt = {'schema': 'o2-v10-local-verification-v1', 'taskId': C.TASK, 'mode': 'FINAL' if args.final else 'TRANSIENT',
               'commands': rows, 'inputHashes': inputs, 'protectedOldHashes': old, 'ownerPins': owner_pins,
               'sourceBefore': C.relative(archive / 'source-before.json'), 'sourceAfter': C.relative(archive / 'source-after.json'),
               'sourceStable': stable, 'failed': failed, 'exitCode': 1 if failed else 0,
               'outputs': {C.relative(path): C.digest(path) for path in (win, linux) if path.is_file()},
               'existingAssertionsUnchanged': True, 'remoteExecuted': False, 'credentialsLoaded': False,
               'runtimeCasesExecuted': 0, 'selfVerificationIsApproval': False}
    # Archive exact Go/Python helper input bytes as well as output hashes/logs.
    input_dir = archive / 'helper-inputs'
    input_dir.mkdir()
    for name in inputs:
        with (input_dir / Path(name).name).open('xb') as stream:
            stream.write((C.ROOT / name).read_bytes())
    C.write(archive / 'receipt.json', receipt)
    print(json.dumps({'receipt': C.relative(archive / 'receipt.json'), 'failed': failed,
                      'sourceStable': stable, 'state': 'UNSEALED', 'runtimeCasesExecuted': 0}), flush=True)
    return receipt['exitCode']


if __name__ == '__main__':
    raise SystemExit(main())
