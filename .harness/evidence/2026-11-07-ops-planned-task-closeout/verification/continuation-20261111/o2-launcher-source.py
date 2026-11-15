"""Check the unchanged v12 candidate while preserving Windows' system root.

The original dictionary used mixed-case SystemRoot lookup on uppercase Windows
environment keys. Its null is not an instruction to remove the OS runtime input.
This entry uses only the existing sealed authorization; it adds no authorization.
"""
from pathlib import Path
import hashlib
import argparse
import importlib.util
import json
import os
import sys

sys.dont_write_bytecode = True
local = Path(__file__).resolve().parent
parser = argparse.ArgumentParser()
parser.add_argument('--phase', choices=('check', 'execute'), default='check')
args = parser.parse_args()
root = local.parents[1]
entry = local / 'continue-o2-v12-dispatcher.py'
original_system_root = os.environ.get('SYSTEMROOT')
if not original_system_root or not Path(original_system_root).is_dir():
    raise RuntimeError('A valid existing Windows SYSTEMROOT is required')
spec = importlib.util.spec_from_file_location('original_v12_dispatcher', entry)
dispatcher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(dispatcher)
os.chdir(root)
dispatcher.bound_environment()
req = dispatcher.read(dispatcher.REQUEST)
inventory = dispatcher.read(root / req['currentInventory'])
native = inventory['nativeClosure']['environment']
if native.get('SystemRoot') is not None or Path(native['WINDIR']).resolve() != Path(original_system_root).resolve():
    raise RuntimeError('The sealed Windows directory must match the existing system root')
os.environ['SYSTEMROOT'] = original_system_root
print(json.dumps({'restoredExistingSystemRoot': True, 'originalDispatcherSHA256': hashlib.sha256(entry.read_bytes()).hexdigest(), 'requestSHA256': dispatcher.digest(dispatcher.REQUEST), 'remoteExecuted': False, 'addsAuthorization': False}), flush=True)
runner = dispatcher.load('root-execute.py')
raise SystemExit(runner.main(['--phase', args.phase, '--request', str(dispatcher.REQUEST), '--authorization', str(dispatcher.OUT / 'authorization.json')]))
