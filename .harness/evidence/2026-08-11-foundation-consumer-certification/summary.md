# Evidence Summary

Routine locked sync now validates the verification marker against the lock checksum and re-hashes the manifest and full release tree before reading shared files. Explicit local upgrades pass the verified archive checksum into the consumer, allow release-metadata commits after the manifest source commit, Windows/MSYS extraction avoids absolute `tar -C` paths, and Ops workflows use the same Node 24 and pinned-action baseline as Base.

Ops is locked to the published immutable `pantheon-base-v0.10.12` archive at Base commit `16918771e2650f8c045b0e086144eb290e774704` with SHA-256 `42a62f08881abe0a1680f721e1da9aa7ee50f0ffbf7a1ad2ccb65350f960fb21`. Regression tests cover missing markers, checksum mismatches, post-verification tree modification, explicit checksum binding, portable archive installation, metadata-only HEAD advancement, and workflow baseline drift. Full repository validation is recorded in `commands.json`.
