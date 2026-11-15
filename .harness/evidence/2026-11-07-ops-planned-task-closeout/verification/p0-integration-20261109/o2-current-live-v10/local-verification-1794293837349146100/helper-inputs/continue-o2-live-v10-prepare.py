"""Create v10 copies only; do not run/freeze any historical product source."""
from pathlib import Path
import ast
import importlib.util
import json
import sys

sys.dont_write_bytecode = True
LOCAL = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('v10_contract', LOCAL / 'continue-o2-live-v10-contract.py')
C = importlib.util.module_from_spec(spec)
spec.loader.exec_module(C)


def create(path, text):
    with path.open('x', encoding='utf-8', newline='\n') as stream:
        stream.write(text)


def replace_function(source, name, body):
    node = next(n for n in ast.parse(source).body if isinstance(n, ast.FunctionDef) and n.name == name)
    lines = source.splitlines(keepends=True)
    return ''.join(lines[:node.lineno - 1]) + body + '\n\n' + ''.join(lines[node.end_lineno:])


def main():
    C.EV.mkdir(parents=True, exist_ok=True)
    C.write(C.EV / 'protected-old-inputs.json', C.hashes(C.protected_paths()))
    for suffix in ('driver.go', 'driver_test.go', 'transport_test.go'):
        text = (LOCAL / ('resume-o2-v9-' + suffix)).read_text().replace('resume-o2-v9', 'continue-o2-live-v10')
        create(LOCAL / (C.PREFIX + suffix), text)
    text = (LOCAL / 'resume-o2-v9-runner.py').read_text().replace('resume-o2-v9', 'continue-o2-live-v10')
    text = text.replace("EVIDENCE = ROOT / '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'",
                        "EVIDENCE = ROOT / '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v10'")
    text = text.replace('from pathlib import Path', 'from pathlib import Path\nimport importlib.util')
    text = text.replace("REQUEST = LOCAL / 'continue-o2-live-v10-phase-request.json'", """REQUEST = LOCAL / 'continue-o2-live-v10-phase-request.json'
spec = importlib.util.spec_from_file_location('v10_runner_contract', LOCAL / 'continue-o2-live-v10-contract.py')
C = importlib.util.module_from_spec(spec)
spec.loader.exec_module(C)""")
    text = replace_function(text, 'source_hashes', 'def source_hashes():\n    return C.inventory()')
    text = replace_function(text, 'artifacts', 'def artifacts():\n    return C.artifacts()')
    text = replace_function(text, 'request', """def request(phase='execute', dependencies=None):
    # Retain failure evidence without overwriting any sealed proposal/request.
    value = {'phase': phase, 'dependencies': dependencies or [], 'executionDisabled': True}
    write('runner-blocker-' + str(time.time_ns()), value)
    return value""")
    text = replace_function(text, 'build', "def build():\n    return C.load('verify.py').main()")
    create(LOCAL / (C.PREFIX + 'runner.py'), text)
    C.invariant_checks()
    C.write(C.EV / 'proposal.json', {
        'schema': 'o2-current-request-v10', 'state': 'UNSEALED', 'executionDisabled': True,
        'taskId': C.TASK, 'tier': 'L2', 'host': '115.190.251.143', 'allCases': list(C.CASES),
        'assertionCallCount': 45, 'existingAssertionsUnchanged': True,
        'parentTaskPacket': 'docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md',
        'requiredPins': ['actual go list -mod=readonly -deps -test inventory before/after',
                         'all stdlib/module/source/helper/tool/build inputs and native outputs',
                         'final O2 owner source-stable handoff + CODE APPROVE + ARCH CLEAR',
                         'final build receipt, verification inputs/logs and protected OLD hashes',
                         'independent v10 CODE/ARCH reports and Dispatcher request authorization'],
        'ownerHandoff': C.relative(C.OWNER / 'handoff.md'),
        'protectedOldHashes': C.protected(), 'selfVerificationIsApproval': False,
        'publicBaseIdentityAuditAcceptanceImplemented': False,
        'remoteExecuted': False, 'credentialsLoaded': False})
    print(json.dumps({'state': 'UNSEALED', 'existingAssertionsUnchanged': True,
                      'remoteExecuted': False, 'credentialsLoaded': False}))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
