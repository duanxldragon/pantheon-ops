"""Local negative regressions; fake resources never perform VM/DB acceptance."""
from pathlib import Path
import copy
import importlib.util
import io
import json
import sys
import tempfile
import types
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
LOCAL = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('v10_test_contract', LOCAL / 'continue-o2-live-v10-contract.py')
C = importlib.util.module_from_spec(spec)
spec.loader.exec_module(C)


class Contracts(unittest.TestCase):
    def test_original_matrix_assertions_tests_and_cleanup_preserved(self):
        C.invariant_checks()
        C.protected()

    def test_unsealed_and_old185_substitution_rejected(self):
        with tempfile.TemporaryDirectory(dir=LOCAL, prefix=C.PREFIX + 'fixture-') as directory:
            file = Path(directory) / 'request.json'
            C.write(file, {'state': 'UNSEALED'})
            with self.assertRaisesRegex(RuntimeError, 'UNSEALED'):
                C.validate_request(file)
        value = self.request_fixture()
        value.pop('currentInventory')
        value['productSourceHashes'] = {str(index): '0' * 64 for index in range(185)}
        with patch.object(C, 'read', return_value=value), self.assertRaisesRegex(RuntimeError, 'old185'):
            C.validate_request(Path('fixture'))

    def request_fixture(self):
        return {'schema': 'o2-current-request-v10', 'state': 'SEALED-CANDIDATE',
                'host': '115.190.251.143', 'allCases': list(C.CASES),
                'assertionCallCount': 45, 'existingAssertionsUnchanged': True,
                'candidateHashes': {'fixture': '0' * 64}, 'evidenceHashes': {'fixture': '0' * 64},
                'ownerPins': {'fixture': '0' * 64}, 'protectedOldHashes': {'fixture': '0' * 64},
                'currentInventory': 'fixture.json'}

    def test_source_tool_or_discovery_drift_rejected(self):
        value = self.request_fixture()
        before = {'schema': 'o2-current-dependencies-v10', 'files': {'repo:source.go': '0' * 64},
                  'toolFiles': {'external:go.exe': '0' * 64}, 'discovery': [{'command': ['go', 'list', '-deps', '-test']}]}
        for field in ('files', 'toolFiles', 'discovery'):
            changed = copy.deepcopy(before)
            changed[field] = {}
            with self.subTest(field=field), patch.object(C, 'read', side_effect=[value, before]), \
                 patch.object(C, 'check_hashes'), patch.object(C, 'protected', return_value=value['protectedOldHashes']), \
                 patch.object(C, 'inventory', return_value=changed), self.assertRaisesRegex(RuntimeError, 'drift'):
                C.validate_request(Path('fixture'))

    def test_missing_authority_rejects_before_validation_and_runner_import(self):
        entry = C.load('root-execute.py')
        with patch.object(entry.C, 'validate_request') as validation, patch.object(entry.C, 'load') as loading, \
             patch.dict(sys.modules, {'remote': None, 'pymysql': None}), self.assertRaisesRegex(RuntimeError, 'execution disabled'):
            entry.main(['--phase', 'execute', '--request', 'nonexistent.json'])
        validation.assert_not_called()
        loading.assert_not_called()

    def test_attempt_replay_and_failed_start_remain_consumed(self):
        with tempfile.TemporaryDirectory(dir=LOCAL, prefix=C.PREFIX + 'fixture-') as directory:
            path = Path(directory)
            request = path / 'request.json'
            C.write(request, {'localFixture': True})
            with patch.object(C, 'EV', path):
                first = C.reserve_attempt(request)
                prior = first.read_bytes()
                with self.assertRaisesRegex(RuntimeError, 'replay'):
                    C.reserve_attempt(request)
                self.assertEqual(first.read_bytes(), prior)

    def test_old_review_bad_subject_bad_report_and_wrong_role_rejected(self):
        with tempfile.TemporaryDirectory(dir=C.EV, prefix='review-fixture-') as directory:
            path = Path(directory)
            subject = path / 'subject.json'
            report = path / 'report.md'
            C.write(subject, {'localFixture': True})
            report.write_text('CODE APPROVE\n', encoding='utf-8')
            receipt = path / 'receipt.json'
            base = {'schema': 'o2-v10-review-v1', 'role': 'code-reviewer', 'verdict': 'CODE APPROVE',
                    'exitCode': 0, 'subjectSHA256': C.digest(subject),
                    'report': C.relative(report), 'reportSHA256': C.digest(report)}
            for field, changed in (('schema', 'historical-exit0'), ('subjectSHA256', '0' * 64),
                                   ('reportSHA256', '0' * 64), ('role', 'generator'), ('exitCode', 1)):
                with self.subTest(field=field):
                    value = dict(base, **{field: changed})
                    receipt.write_text(json.dumps(value), encoding='utf-8')
                    with self.assertRaises(RuntimeError):
                        C.review(receipt, 'code-reviewer', 'CODE APPROVE', subject)
            receipt.write_text(json.dumps(base), encoding='utf-8')
            pins = C.review(receipt, 'code-reviewer', 'CODE APPROVE', subject)
            self.assertEqual(pins[C.relative(report)], C.digest(report))
            report.write_text('CODE BLOCK\n', encoding='utf-8')
            with self.assertRaisesRegex(RuntimeError, 'hash mismatch'):
                C.review(receipt, 'code-reviewer', 'CODE APPROVE', subject)

    def test_bad_authorization_request_or_receipt_hash_rejected(self):
        with tempfile.TemporaryDirectory(dir=C.EV, prefix='authorization-fixture-') as directory:
            path = Path(directory)
            request = path / 'request.json'
            authorization = path / 'authorization.json'
            receipt = path / 'receipt.json'
            C.write(request, {'localFixture': True})
            C.write(receipt, {'notApproval': True})
            C.write(authorization, {'schema': 'o2-v10-authorization-v1', 'requestSHA256': '0' * 64})
            with self.assertRaisesRegex(RuntimeError, 'request hash'):
                C.authorization(authorization, request)
            authorization.write_text(json.dumps({'schema': 'o2-v10-authorization-v1',
                'requestSHA256': C.digest(request), 'codeReceipt': {'path': C.relative(receipt), 'sha256': '0' * 64}}))
            with self.assertRaisesRegex(RuntimeError, 'receipt hash'):
                C.authorization(authorization, request)

    def test_transient_cannot_seal_and_missing_owner_does_not_build(self):
        verifier = C.load('verify.py')
        with patch.object(verifier.C, 'owner_gate', side_effect=RuntimeError('final handoff missing')), \
             patch.object(verifier.C, 'inventory') as listing, patch.object(verifier.subprocess, 'run') as command:
            with self.assertRaisesRegex(RuntimeError, 'handoff missing'):
                verifier.main(['--final', '--owner-code', 'missing-code', '--owner-arch', 'missing-arch'])
        listing.assert_not_called()
        command.assert_not_called()
        seal = C.load('seal.py')
        with patch.object(seal.C, 'owner_gate', return_value={}), \
             patch.object(seal.C, 'read', return_value={'schema': 'o2-v10-local-verification-v1', 'mode': 'TRANSIENT'}):
            with self.assertRaisesRegex(RuntimeError, 'FINAL verification'):
                seal.main(['--verification', str(C.EV / 'fixture.json'), '--owner-code', 'fixture', '--owner-arch', 'fixture'])

    def test_runner_cli_resource_guards(self):
        runner = C.load('runner.py')
        with patch.dict(sys.modules, {'remote': None, 'pymysql': None}):
            for action in (lambda: runner.execute(runner.CASES), runner.preflight):
                with self.assertRaises(RuntimeError):
                    action()

    def test_original_cold_failure_retains_events_and_exit_one(self):
        runner = C.load('runner.py')
        events = [{'phase': 'independent-ssh-error', 'stage': 'handshake-auth', 'unexpectedEOF': True},
                  {'phase': 'driver-result', 'failed': 1}]
        class FakePath:
            def __truediv__(self, name): return self
            def read_text(self): return '{"sourceStable":true,"artifacts":{}}'
        class FakeProcess:
            returncode = 1
            def __init__(self, *args, **kwargs): pass
            def communicate(self, *args, **kwargs): return '\n'.join(json.dumps(e) for e in events), ''
            def poll(self): return 1
        class FakeClient:
            def get_transport(self): return self
            def set_keepalive(self, seconds): pass
            def close(self): pass
        class FakeDB:
            def cursor(self): return self
            def __enter__(self): return self
            def __exit__(self, *args): pass
            def execute(self, sql): pass
            def close(self): pass
        saved = {}
        remote = types.SimpleNamespace(connect=lambda: FakeClient(), load_credential=lambda: (
            {'username': 'fixture', 'auth_mode': 'password'}, 'ephemeral-local-only'))
        with patch.dict(sys.modules, {'remote': remote, 'pymysql': types.SimpleNamespace(connect=lambda **kw: FakeDB())}), \
             patch.object(runner, 'EVIDENCE', FakePath()), \
             patch.object(runner, 'source_hashes', return_value={'sourceStable': True, 'artifacts': {}}), \
             patch.object(runner, 'artifacts', return_value={}), patch.object(runner, 'pin', return_value='fixture'), \
             patch.object(runner, 'remote_command', return_value=''), patch.object(runner, 'cleanup', return_value={'absent': True}), \
             patch.object(runner, 'write', side_effect=lambda name, value: saved.update({name: value})), \
             patch.object(runner.subprocess, 'Popen', FakeProcess), patch.object(runner.time, 'sleep'), \
             patch('sys.stdout', new_callable=io.StringIO) as output:
            code = runner.root_execute_after_review(['cold-restart'])
        case = next(value for name, value in saved.items() if name.startswith('case-cold-restart-'))
        self.assertEqual(code, 1)
        self.assertEqual(case['status'], 'blocked')
        self.assertEqual(case['events'], events)
        self.assertTrue(case['initialDriverOutputRetained'])
        self.assertNotIn('ephemeral-local-only', output.getvalue())


if __name__ == '__main__':
    unittest.main(verbosity=2)
