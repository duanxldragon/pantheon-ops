Resume your previous partial 0d6c4daa-3992-4517-ab84-3fe5a6fd386e. Only Edit/Write, no Read. File currently has these exact remaining bugs (Codex read final state). Finish in 4-5 edits then END, final <=50 words.

File `.local/acceptance-20261121/ssh-probe.py`:
exec_capture now returns FIVE values `(rc, out, err, out_trunc, err_trunc)` but all three callers still unpack THREE. Update probe/expect_file/query_absent and fail ok on any truncation. Deadline check must occur inside EACH recv/recv_stderr inner loop too; continuous stdout can currently starve deadline.
_confine_path signature now (path, run_id); _guarded still calls it with ONE. Add run_id to _guarded/expect_file/query_absent and propagate.
_guarded still allows realpath failure/absence fallback: require realpath, `d` canonical exactly /tmp/<run_id>, `r=$(realpath -m -- "$p") || exit 3`, require r==d or d/*, reject symlink root alias. No fallback.
main still returns0 regardless failed assertions. Build result variable then JSON-print once; return0 only result.ok is True, and fingerprint result.knownHostsVerified is True. query_absent.ok must require actual ABSENT (exact stripped output), expect_file.ok requires found even without contains.
CLI add --run-id required on expect-file/assert-absent; validate path/run-id before Probe() so no password/auth/network happens for invalid path. probe already has run-id, validate it before Probe too.

Do not write tests/docs; no remote execution; no new functionality. These are fixes to your incomplete 11 fragment, not another refactor. End immediately after all edits.
