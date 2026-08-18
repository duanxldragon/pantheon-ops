import json
import os

TRANSCRIPT = r"C:\Users\xiaolong\.claude\projects\D--workspace-go-pantheon-platform\012be091-6157-4f0d-ac18-4907db0a94f3.jsonl"

# ops[key] -> list of (name, input_dict) in chronological order
ops = {}
order = []


def norm_key(fp):
    return fp.replace("\\", "/")


def is_k8s(fp):
    return ("business/k8s" in fp) or ("business\\k8s" in fp)


with open(TRANSCRIPT, encoding="utf-8") as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        try:
            obj = json.loads(line)
        except Exception:
            continue
        if obj.get("type") != "assistant":
            continue
        msg = obj.get("message")
        if not isinstance(msg, dict):
            continue
        content = msg.get("content")
        if not isinstance(content, list):
            continue
        for c in content:
            if not isinstance(c, dict):
                continue
            if c.get("type") != "tool_use":
                continue
            name = c.get("name")
            if name not in ("Write", "Edit"):
                continue
            inp = c.get("input") or {}
            fp = inp.get("file_path", "")
            if not fp or not is_k8s(fp):
                continue
            key = norm_key(fp)
            if key not in ops:
                ops[key] = []
                order.append(key)
            ops[key].append((name, inp))

print("TOTAL k8s file targets:", len(ops))
print()

final = {}
errors = []
for key in order:
    seq = ops[key]
    first_name = seq[0][0]
    if first_name != "Write":
        errors.append((key, "first op is %s (no base)" % first_name))
        print("SKIP (no base):", key)
        continue
    content = seq[0][1].get("content", "")
    for name, inp in seq[1:]:
        if name == "Write":
            content = inp.get("content", "")
        elif name == "Edit":
            old = inp.get("old_string", "")
            new = inp.get("new_string", "")
            if inp.get("replace_all"):
                content = content.replace(old, new)
            else:
                if old in content:
                    content = content.replace(old, new, 1)
                else:
                    errors.append((key, "Edit old_string not found"))
    final[key] = content
    print("%2d ops -> %-90s %6d chars" % (len(seq), key, len(content)))

print()
if errors:
    print("ERRORS:")
    for e in errors:
        print("  ", e)
else:
    print("No reconstruction errors.")

# Write reconstructed files to disk (canonical forward-slash absolute paths)
written = 0
for key, content in final.items():
    # key is already a normalized forward-slash absolute path like
    # D:/workspace/go/pantheon-platform/pantheon-ops/backend/...
    d = os.path.dirname(key)
    os.makedirs(d, exist_ok=True)
    with open(key, "w", encoding="utf-8", newline="\n") as out:
        out.write(content)
    written += 1

print()
print("WROTE %d files to disk." % written)
