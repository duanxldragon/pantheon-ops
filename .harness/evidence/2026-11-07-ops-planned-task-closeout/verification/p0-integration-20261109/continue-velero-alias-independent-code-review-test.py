"""Read-only local counterexamples; synthetic inputs, no remote or author edits."""
from pathlib import Path
import ast
import base64
import copy
import hashlib
import io
import json
import runpy
import tempfile
import types
import unittest
from unittest.mock import patch

ROOT = Path(__file__).absolute().parents[5]
LOCAL = ROOT / '.local/ops-acceptance'
T = runpy.run_path(str(LOCAL / 'continue-velero-alias-test.py'), run_name='independent_review_import')
C, R = T['C'], T['R']


class IndependentCounterexamples(unittest.TestCase):
    def test_unreviewed_service_spec_returns_identical_native_five_proof(self):
        entries = C.parse(C.capture(C.EV / C.FAILURE, C.FAILURE_SHA))['records'][0]['ownedObjects'][:5]
        auth = {'accessKey': 'synthetic', 'secretKey': 'synthetic'}
        cloud = b'[default]\naws_access_key_id=synthetic\naws_secret_access_key=synthetic\n'
        objects = []
        for entry in entries:
            component = 'velero' if entry['kind'] == 'Namespace' and entry['name'] == 'velero' else 'seaweedfs-s3'
            metadata = {'name': entry['name'], 'uid': entry['uid'], 'labels': {
                'pantheon-p0-owner': C.OWNER, 'app.kubernetes.io/component': component}}
            if entry['namespace']:
                metadata['namespace'] = entry['namespace']
            objects.append({'kind': entry['kind'], 'metadata': metadata})
        objects[2]['spec'] = {'type': 'ClusterIP', 'ports': [{'name': 's3', 'port': 18333, 'targetPort': 18333,
                                                          'protocol': 'TCP'}], 'sessionAffinity': 'None'}
        objects[3]['subsets'] = [{'addresses': [{'ip': '192.168.94.127'}],
                                 'ports': [{'name': 's3', 'port': 18333, 'protocol': 'TCP'}]}]
        objects[4].update(type='Opaque', data={'cloud': base64.b64encode(cloud).decode()})
        def get(*args):
            return copy.deepcopy(next(o for o in objects if o['kind'] == args[1] and o['metadata']['name'] == args[2]))
        with tempfile.TemporaryDirectory(prefix='alias-review-five-') as directory:
            home = Path(directory)
            (home / 's3-auth.json').write_text(json.dumps({'identities': [{'credentials': [auth]}]}))
            with patch.object(R.N, 'HOME_DIR', home), patch.object(R.N.P, 'k', side_effect=get), \
                    patch.object(R.N.V4, 'credential_summary', return_value={'synthetic': True}), \
                    patch.object(R.N, 'guarded_namespace', return_value={'namespaceDefaults': [], 'controllerMirror': None}), \
                    patch.object(R, 'current_velero_namespace', return_value={'namespaceDefaults': [], 'controllerMirror': None}):
                original_adapter = R.original_five_adapter()
                approved = original_adapter(entries, '192.168.94.127')
                for field, value in [('sessionAffinity', 'ClientIP'), ('internalTrafficPolicy', 'Local')]:
                    with self.subTest(field=field):
                        objects[2]['spec'][field] = value
                        current = original_adapter(entries, '192.168.94.127')
                        self.assertEqual(current, approved)

    def test_recompiled_inspector_executes_reopened_unsealed_source_expression(self):
        source = C.function_source(R.N, 'inspect')
        tree = ast.parse(source).body[0]
        self.assertIsInstance(tree, ast.FunctionDef)
        start = next(i for i, n in enumerate(tree.body) if isinstance(n, ast.Assign) and
                     any(isinstance(t, ast.Name) and t.id == 'source_text' for t in n.targets))
        end = next(i for i, n in enumerate(tree.body) if isinstance(n, ast.Expr) and isinstance(n.value, ast.Call) and
                   any(isinstance(a, ast.Constant) and a.value == 'native-process-argv-drift' for a in n.value.args))
        block = ast.Module(body=tree.body[start:end + 1], type_ignores=[])
        context = dict(R.N.__dict__)
        exec(compile(source, '<actual-alias-inspector-adapter>', 'exec'), context)
        self.assertIn('_sealed_read', R.N.inspect.__code__.co_names)
        self.assertNotIn('_sealed_read', context['inspect'].__code__.co_names)
        with tempfile.TemporaryDirectory(prefix='alias-review-source-') as directory:
            home = Path(directory)
            file = home / 'synthetic-original-provider.py'
            approved = b"def install():\n    command(['systemd-run', str(HOME_DIR / 'weed')])\n"
            file.write_bytes(approved)
            approved_hash = hashlib.sha256(approved).hexdigest()
            file.write_text("def install():\n    command(['systemd-run', (HOME_DIR / 'unexpected-effect').touch(), str(HOME_DIR / 'weed')])\n")
            values = dict(context, ast=ast, P=types.SimpleNamespace(__file__=str(file)), HOME_DIR=home,
                          UNIT='synthetic', ip='192.168.94.127', command_line=[str(home / 'weed')])
            # Exact reopened-source block from the current adapter executes the expression.
            exec(compile(block, '<actual-current-reopened-source-block>', 'exec'), values)
            self.assertTrue((home / 'unexpected-effect').exists())
            (home / 'unexpected-effect').unlink()
            # Retaining the existing captured syntax transformation refuses before the effect.
            fixed_block = ast.fix_missing_locations(C.S.CapturedSyntax().visit(copy.deepcopy(block)))
            values['_sealed_read'] = C.S.read
            with patch.dict(C.S.SEALS, {str(file): approved_hash}):
                with self.assertRaisesRegex(ValueError, 'captured-sealed-bytes-drift'):
                    exec(compile(fixed_block, '<original-captured-source-block>', 'exec'), values)
            self.assertFalse((home / 'unexpected-effect').exists())


if __name__ == '__main__':
    unittest.main(verbosity=2)
