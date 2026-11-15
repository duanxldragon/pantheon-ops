"""Run a finite, read-only installed review role in an independent Claude context."""
from pathlib import Path
import argparse
import hashlib
import json
import subprocess
import time
import tomllib

parser = argparse.ArgumentParser()
parser.add_argument('role', choices=['code-reviewer', 'architect'])
parser.add_argument('attempt')
args = parser.parse_args()
ops = Path(__file__).resolve().parents[2]
role_path = ops.parent / '.codex/agents' / (args.role + '.toml')
role = tomllib.loads(role_path.read_text(encoding='utf-8-sig'))['developer_instructions']
request_path = ops / '.local/ops-acceptance/continue-20261111-final-reviews.txt'
request = request_path.read_text(encoding='utf-8-sig')
lane = 'CODE' if args.role == 'code-reviewer' else 'ARCH'
prompt = (f'You are the independent {lane} review lane. The installed role instructions follow. '
          'Use only Read, Glob and Grep; no Bash, writes, agents, remote actions or credential reads. '
          'Source hashes are independently checked by Dispatcher before and after. '
          'Do not claim to have executed tests. Review only the named exact sources and finite evidence.\n\n'
          + role + '\n\n' + request
          + '\nCURRENT CORRECTION: latest Ops regression is 206 pass / 207 tests / 1 fail, '
          'zero skips. The sole failure is EPERM creating an actual Windows file symlink, before '
          'the product assertion. Historical 207-pass wording is not a current green result. '
          'Original-byte forward/reverse proof passes; 137 inherited TypeScript production diagnostics, '
          'zero diagnostics in the new test, no new code/category/source identities. '
          'No full type-safety claim. Provide two separate bounded verdicts with exact JSON bindings.\n')
if not args.attempt.replace('-', '').isalnum():
    raise RuntimeError('A simple new attempt identifier is required')
output = ops / '.local/ops-acceptance' / f'continue-20261111-claude-{lane.lower()}-{args.attempt}'
output.mkdir()
(output / 'prompt.txt').write_text(prompt, encoding='utf-8')
exe = Path('D:/cache/npm-global/node_modules/@anthropic-ai/claude-code/bin/claude.exe')
command = [str(exe), '--print', '--output-format', 'stream-json', '--verbose',
           '--no-session-persistence', '--disable-slash-commands',
           '--tools', 'Read,Glob,Grep', '--allowedTools', 'Read,Glob,Grep',
           '--strict-mcp-config', '--mcp-config', '{"mcpServers":{}}',
           '--settings', '{"disableAllHooks":true}', '--max-turns', '24']
start = time.monotonic()
timed_out = False
with (output / 'events.jsonl').open('x', encoding='utf-8') as events:
    proc = subprocess.Popen(command, cwd=ops, stdin=subprocess.PIPE, stdout=events,
                            stderr=subprocess.STDOUT, text=True, encoding='utf-8')
    try:
        proc.communicate(prompt, timeout=180)
    except subprocess.TimeoutExpired:
        timed_out = True
        proc.terminate()
        proc.wait(timeout=20)
records = []
for line in (output / 'events.jsonl').read_text(encoding='utf-8').splitlines():
    try:
        item = json.loads(line)
    except json.JSONDecodeError:
        continue
    if item.get('type') == 'result':
        records.append(item)
report = None
if len(records) == 1 and not timed_out and not records[0].get('is_error'):
    report = records[0].get('result')
    if isinstance(report, str) and report.strip():
        (output / 'report.md').write_text(report, encoding='utf-8')
receipt = {'role': args.role, 'lane': lane, 'provider': 'installed Claude CLI',
           'exitCode': proc.returncode, 'timedOut': timed_out,
           'elapsedSeconds': round(time.monotonic() - start, 3),
           'modelOverride': False, 'nativeModelSelectionChanged': False,
           'tools': ['Read', 'Glob', 'Grep'], 'hooksEnabled': False,
           'promptSHA256': hashlib.sha256(prompt.encode()).hexdigest(),
           'roleSHA256': hashlib.sha256(role_path.read_bytes()).hexdigest(),
           'report': str((output / 'report.md').relative_to(ops)) if report else None,
           'resultCount': len(records)}
(output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
print(json.dumps(receipt), flush=True)
raise SystemExit(0 if report and proc.returncode == 0 else 1)
