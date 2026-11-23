import json, io, glob, os, sys

base = r'D:\workspace\go\pantheon-platform\pantheon-ops\.harness\evidence\2026-11-20-ops-review-live-runtime-acceptance'
run = sys.argv[1]
d = glob.glob(os.path.join(base, 'k8s', run, 'summary-*.json'))
if not d:
    print('no summary for', run)
    sys.exit(1)
o = json.load(io.open(d[0], encoding='utf-8'))
print('run:', o.get('runId'), 'status:', o.get('status'))
for c in o['cases']:
    st = c['status']
    marker = '  ok      ' if st == 'PASS' else '  **' + st + '** '
    print(marker + c['id'], '|', c.get('intent', '')[:100])
    if st != 'PASS':
        print('       actual:', json.dumps(c.get('actual'), ensure_ascii=False)[:900])
        print('       evidence:', c.get('evidence'))