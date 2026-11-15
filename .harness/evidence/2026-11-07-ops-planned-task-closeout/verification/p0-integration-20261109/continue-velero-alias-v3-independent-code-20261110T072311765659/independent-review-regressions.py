"""Independent offline regressions: expected to fail on sealed alias v3.
No connection, source edit, generator, remote phase, or actual approval is performed.
Synthetic approval fixtures stay in this own evidence directory.
"""
from pathlib import Path
import hashlib, io, json, runpy, types, unittest
from unittest.mock import patch
OWNED = Path(__file__).absolute().parent
ROOT = OWNED.parents[5]
NAMESPACE = runpy.run_path(str(ROOT/'.local/ops-acceptance/continue-velero-alias-v3-test.py'), run_name='independent_gate_cases')
W, C = NAMESPACE['W'], NAMESPACE['C']
REQUEST_BYTES = (C.EV/C.REQUEST).read_bytes()
REQUEST = json.loads(REQUEST_BYTES)
REQUEST['_capturedRequestSHA256'] = hashlib.sha256(REQUEST_BYTES).hexdigest()

class IndependentRegressions(unittest.TestCase):
    def test_arch_watch_block_unknown_or_missing_cannot_approve(self):
        for status in ['WATCH', 'BLOCK', 'UNKNOWN', None]:
            with self.subTest(status=status), patch.object(W.C, 'EV', OWNED/('gate-'+str(status))):
                with self.assertRaises(ValueError):
                    W.approved_reviews(REQUEST)

    def test_missing_exact_remote_repair_receipt_is_incomplete_inspection(self):
        name = 'continue-velero-alias-v3-root-repair-20261110T000000000000.json'
        args = types.SimpleNamespace(request_sha256=REQUEST['_capturedRequestSHA256'], receipt=name,
                                     sha256=hashlib.sha256((OWNED/name).read_bytes()).hexdigest())
        class OfflineClient:
            def __enter__(self): return self
            def __exit__(self, *args): return False
        with patch.object(W.C, 'EV', OWNED), patch.object(W, 'reviewed_inputs', return_value=(REQUEST,None,None,None)), \
             patch.object(W, 'approved_reviews', return_value={}), patch.object(W.W, 'connect', return_value=OfflineClient()), \
             patch.object(W, 'capture_partial', return_value=REQUEST['expectedRemoteFiles']), patch.object(W, 'invoke') as invoke:
            result = W.inspect_failed(args)
        invoke.assert_not_called()
        self.assertEqual(result['exitCode'], 1)
        self.assertTrue(result['partialInspectionRequired'])
        self.assertFalse(result['replayPermitted'])

if __name__ == '__main__':
    unittest.main(verbosity=2)
