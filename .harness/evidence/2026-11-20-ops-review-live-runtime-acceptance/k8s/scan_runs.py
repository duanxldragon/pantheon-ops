import json, io, glob, os, sys
from collections import Counter

base = sys.argv[1]
for d in sorted(glob.glob(os.path.join(base, 'omx-accept-20261121-k8s*/'))):
    for s in sorted(glob.glob(os.path.join(d, 'summary-*.json'))):
        o = json.load(io.open(s, encoding='utf-8'))
        cases = o.get('cases', [])
        c = Counter(x.get('status') for x in cases)
        fails = [x.get('id') for x in cases if x.get('status') != 'PASS']
        name = os.path.basename(d.rstrip('/\\'))
        tail = ','.join(fails) if fails else 'ALL-PASS'
        print('%-32s | cases=%2d %s | %s' % (name, len(cases), dict(c), tail))