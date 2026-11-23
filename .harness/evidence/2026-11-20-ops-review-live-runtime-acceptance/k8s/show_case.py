"""Print one case evidence file (another script that actually works in this repo)."""
import json, io, sys

p = sys.argv[1]
d = json.load(io.open(p, encoding='utf-8'))
print(json.dumps(d, ensure_ascii=False, indent=1)[:2500])