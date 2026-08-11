# Evidence Summary

Routine locked sync now validates the verification marker against the lock checksum and re-hashes the manifest and full release tree before reading shared files. Explicit local upgrades pass the verified archive checksum into the consumer, Windows/MSYS extraction avoids absolute `tar -C` paths, and Ops workflows use the same Node 24 and pinned-action baseline as Base.

Regression tests cover missing markers, checksum mismatches, post-verification tree modification, explicit checksum binding, portable archive installation, and workflow baseline drift. Full repository validation is recorded in `commands.json`.
