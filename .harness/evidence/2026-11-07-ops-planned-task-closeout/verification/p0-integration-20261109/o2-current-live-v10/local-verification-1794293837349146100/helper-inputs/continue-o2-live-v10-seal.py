"""Final source/build candidate seal. Does not issue any review/authorization."""
from pathlib import Path
import argparse
import importlib.util
import json
import sys

sys.dont_write_bytecode = True
LOCAL = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('v10_seal_contract', LOCAL / 'continue-o2-live-v10-contract.py')
C = importlib.util.module_from_spec(spec)
spec.loader.exec_module(C)


def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument('--verification', type=Path, required=True)
    parser.add_argument('--owner-code', type=Path, required=True)
    parser.add_argument('--owner-arch', type=Path, required=True)
    args = parser.parse_args(argv)
    owner = C.owner_gate(args.owner_code, args.owner_arch)
    path = args.verification.resolve()
    C.require(path.is_relative_to(C.EV.resolve()), 'verification outside v10 evidence')
    receipt = C.read(path)
    C.require(receipt.get('schema') == 'o2-v10-local-verification-v1' and receipt.get('mode') == 'FINAL'
              and receipt.get('sourceStable') is True and receipt.get('failed') == 0
              and len(receipt['commands']) == 7 and all(row['exitCode'] == 0 for row in receipt['commands']),
              'fresh complete FINAL verification required; TRANSIENT never seals')
    C.check_hashes(receipt['inputHashes'])
    C.check_hashes(receipt['outputs'])
    C.require(receipt['ownerPins'] == owner and receipt['protectedOldHashes'] == C.protected(), 'owner/protected drift')
    before = C.read(C.ROOT / receipt['sourceBefore'])
    after = C.read(C.ROOT / receipt['sourceAfter'])
    C.require(before == after and after == C.inventory(), 'actual current inventory drift; do not repin old185 map')
    C.invariant_checks()
    evidence = {C.relative(path): C.digest(path), receipt['sourceBefore']: C.digest(C.ROOT / receipt['sourceBefore']),
                receipt['sourceAfter']: C.digest(C.ROOT / receipt['sourceAfter']),
                C.relative(C.EV / 'protected-old-inputs.json'): C.digest(C.EV / 'protected-old-inputs.json')}
    for row in receipt['commands']:
        C.require(C.digest(C.ROOT / row['log']) == row['logSHA256'], 'native verification log drift')
        evidence[row['log']] = row['logSHA256']
    output = C.EV / 'continue-o2-live-v10-build-ready-result.json'
    build = {'sourceStable': True, 'artifacts': C.artifacts(), 'verificationReceipt': C.relative(path),
             'remoteExecuted': False, 'credentialsLoaded': False}
    C.write(output, build)
    C.write(C.EV / 'continue-o2-live-v10-build-ready-source-after.json', after)
    for file in (output, C.EV / 'continue-o2-live-v10-build-ready-source-after.json'):
        evidence[C.relative(file)] = C.digest(file)
    request = {'schema': 'o2-current-request-v10', 'state': 'SEALED-CANDIDATE', 'executionDisabled': True,
               'taskId': C.TASK, 'tier': 'L2', 'host': '115.190.251.143', 'allCases': list(C.CASES),
               'assertionCallCount': 45, 'existingAssertionsUnchanged': True,
               'currentInventory': receipt['sourceAfter'], 'ownerPins': owner,
               'candidateHashes': C.hashes(LOCAL / name for name in build['artifacts']),
               'evidenceHashes': evidence, 'protectedOldHashes': C.protected(),
               'parentTaskPacket': 'docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md',
               'publicBaseIdentityAuditAcceptanceImplemented': False, 'selfVerificationIsApproval': False,
               'authorizationRequired': 'independent CODE APPROVE + ARCH CLEAR + DISPATCHER AUTHORIZE receipts',
               'preservedExitCodes': {'failure': 1, 'deliberateCrashReady': 75, 'remoteCommand': 'exact SSH status'}}
    request_path = C.EV / 'sealed-candidate-request.json'
    C.write(request_path, request)
    C.validate_request(request_path)
    print(json.dumps({'request': C.relative(request_path), 'sha256': C.digest(request_path),
                      'state': 'SEALED-CANDIDATE', 'executionDisabled': True}))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
