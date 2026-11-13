# Official Base v0.18.7 consumer

1. Acquire the four official v0.18.7 assets published after the exact 3203624e Release Gate; verify sidecar and hosted digests before locking.
2. Standard isolated staging rebuild from the locked snapshot; bilingual version anchors and inheritance docs updated only from the validated tree.
3. Full readonly CGO race, frontend, overlay/drift, fresh migration and applicable standard smoke suites on the exact staged tree.
4. Materialize the exact validated staged source into the consumer PR; independent CODE/ARCH review of the assembled delta.
5. Normal hooks, required CI, squash merge, main-only consolidation; retain v0.18.6 lock history for rollback.
