"""Package existing exact regression bytes for a capable, already-installed Linux runtime."""
from pathlib import Path
import hashlib
import json
import shutil
import tarfile

ops = Path(__file__).resolve().parents[2]
workspace = ops.parent
local = ops / '.local/ops-acceptance'
target = local / 'continue-20261111-symlink-replay-a1'
target.mkdir()
sources = ['pantheon-base/scripts/foundation-release/' + name for name in
           ('build-candidate-artifact.mjs', 'candidate-descriptor.mjs')]
sources += ['pantheon-ops/scripts/business-overlay/' + name for name in
            ('foundation-candidate.mjs', 'foundation-candidate.test.mjs',
             'rebuild-from-candidate.mjs', 'rebuild-from-base.mjs',
             'foundation-release.mjs', 'check-shared-drift.mjs', 'candidate-copy-binding.test.mjs')]
sha = lambda file: hashlib.sha256(file.read_bytes()).hexdigest()
files = []
for name in sources:
    source = workspace / name
    assert source.is_file() and not source.is_symlink()
    before = sha(source)
    copy = target / name
    copy.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, copy)
    assert sha(copy) == before == sha(source)
    files.append({'path': name, 'sha256': before, 'bytes': source.stat().st_size})
request = ops / '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-o4-copy-binding-20261110-r2-request.json'
manifest = {'schemaVersion': 1, 'test': 'selected inputs: selected symlink still refuses before stage creation',
            'o4RequestSHA256': sha(request), 'officialLockSHA256': sha(ops / 'foundation-release.lock.json'),
            'runtimeRequirement': 'Existing Linux Node 24 and Git; no npm install or image pull',
            'actualCompleteCandidateAssembly': False, 'independentApproval': False, 'files': files}
(target / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n', encoding='utf-8')
shutil.copyfile(local / 'continue-20261111-symlink-replay.mjs', target / 'verify.mjs')
(target / 'README.txt').write_text(
    'Source-bound isolated fixture regression, not a real O4 candidate receipt or approval.\n'
    'On an existing Linux Node 24 + Git runtime: node verify.mjs\n'
    'Integrity-only check on any installed Node runtime: node verify.mjs --verify-only\n'
    'Keep manifest.json, symlink-test.tap and result.json together. A successful receipt requires\n'
    'exactly 1 test / 1 pass / 0 failures / 0 skips and unchanged source hashes.\n'
    'The historical Windows EPERM remains preserved. Do not suppress or mock symlink creation.\n', encoding='utf-8')
archive = local / 'continue-20261111-symlink-replay-a1.tar.gz'
with tarfile.open(archive, 'x:gz') as tar:
    for file in sorted(target.rglob('*')):
        if file.is_file():
            tar.add(file, arcname=file.relative_to(target).as_posix(), recursive=False)
receipt = {'files': len(files), 'manifestSHA256': sha(target / 'manifest.json'),
           'archive': archive.name, 'archiveSHA256': sha(archive),
           'bytes': archive.stat().st_size, 'sourcesStillMatch': all(sha(workspace / item['path']) == item['sha256'] for item in files),
           'runtimeTestExecuted': False, 'independentApproval': False}
(local / 'continue-20261111-symlink-replay-build-result.json').open('x', encoding='utf-8').write(json.dumps(receipt, indent=2) + '\n')
print(json.dumps(receipt))
