"""New immutable request/receipt boundary; only Dispatcher performs execution."""
from pathlib import Path
import argparse
import importlib.util
import json
import sys

sys.dont_write_bytecode = True
LOCAL = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('v10_root_contract', LOCAL / 'continue-o2-live-v10-contract.py')
C = importlib.util.module_from_spec(spec)
spec.loader.exec_module(C)


def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument('--phase', choices=('check', 'execute'), default='check')
    parser.add_argument('--request', type=Path, required=True)
    parser.add_argument('--authorization', type=Path)
    args = parser.parse_args(argv)
    # No mutable reviewed flag. Missing authorization rejects even UNSEALED input
    # before runner/resource imports, discovery, or private material access.
    if args.phase == 'execute' or args.authorization is not None:
        C.authorization(args.authorization, args.request)
    C.validate_request(args.request)
    if args.phase == 'check':
        print(json.dumps({'exitCode': 0, 'state': 'SEALED-CANDIDATE', 'executionDisabled': args.authorization is None,
                          'remoteExecuted': False, 'credentialsLoaded': False}))
        return 0
    attempt = C.reserve_attempt(args.request)
    # Recheck the request after durable reservation. Failed starts remain consumed.
    C.validate_request(args.request)
    C.authorization(args.authorization, args.request)
    runner = C.load('runner.py')
    code = runner.root_execute_after_review(runner.CASES)
    C.write(attempt.with_name(attempt.stem + '-result.json'), {'exitCode': code, 'requestSHA256': C.digest(args.request)})
    return code


if __name__ == '__main__':
    raise SystemExit(main())
