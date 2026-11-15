"""Run FileDist evidence with the existing verified isolated MySQL fixture."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parents[5]
HERE = Path(__file__).resolve().parent
helper_path = ROOT / "scripts/testing/current-p0-runtime.py"
spec = importlib.util.spec_from_file_location("o1_runtime", helper_path)
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
RUN = HERE / ("run-" + str(time.time_ns()))
RUN.mkdir()
env = helper.child_env(RUN)
env["GOCACHE"] = "D:/cache/go/build"
command = ["go", "test", "-race", "-p=1", "-buildvcs=false", "-mod=readonly", "-json", "./modules/business/filedist"]
def pins():
    return {str(p.relative_to(ROOT)).replace("\\", "/"): hashlib.sha256(p.read_bytes()).hexdigest()
            for area in ("filedist", "cmdb", "capability", "secrets")
            for p in (ROOT / "backend/modules/business" / area).rglob("*")
            if p.is_file() and p.suffix in (".go", ".sql")}
before = pins()
started = time.monotonic()
with (RUN / "stdout.log").open("xb") as out, (RUN / "stderr.log").open("xb") as err:
    completed = subprocess.run(command, cwd=ROOT / "backend", env=env, stdin=subprocess.DEVNULL,
                               stdout=out, stderr=err, timeout=300,
                               creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0))
after = pins()
events = []
for line in (RUN / "stdout.log").read_text(encoding="utf-8", errors="replace").splitlines():
    try:
        value = json.loads(line)
    except ValueError:
        continue
    if value.get("Action") in ("pass", "fail", "skip"):
        events.append({k: value[k] for k in ("Action", "Package", "Test") if k in value})
receipt = {"command": command, "exitCode": completed.returncode, "elapsedSeconds": round(time.monotonic()-started, 3),
           "sourceStable": before == after, "sourceCount": len(before), "fixtureSHA256": helper.FIXTURE_SHA,
           "testPasses": sum(e["Action"] == "pass" and "Test" in e for e in events),
           "skipped": [e for e in events if e["Action"] == "skip"], "failed": [e for e in events if e["Action"] == "fail"],
           "credentialsPrinted": False, "privateLogs": str(RUN.relative_to(ROOT)).replace("\\", "/")}
(RUN / "source-before.json").write_text(json.dumps(before, indent=2), encoding="utf-8")
(RUN / "source-after.json").write_text(json.dumps(after, indent=2), encoding="utf-8")
(RUN / "receipt.json").write_text(json.dumps(receipt, indent=2), encoding="utf-8")
print(json.dumps(receipt))
raise SystemExit(completed.returncode)
