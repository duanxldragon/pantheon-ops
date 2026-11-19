# F-03 Release and Rollback Rehearsal

## Release Build

- Backend: `go build -trimpath -ldflags '-s -w' ./cmd/server`
- Frontend: `npm run build`
- Backend artifact: 78,667,776 bytes, SHA-256
  `C5B5AE18C8ED454D8DC28619A96415A8CBAD7AD9A18FDB635BEC3EAB5BB84B3F`
- Frontend `dist`: 3,608,659 bytes; `index.html` SHA-256
  `503B686309DFE708E36B70F0BC407DE306D7AF8EE1D2AA6D7270F3CAD6CDEECA`

## Database Rehearsal

1. Performed a read-only inspection of source `pantheon_ops` (MySQL `8.0.36`,
   migration version `21`, `dirty=0`, active generated-key unique indexes).
2. Restored a full `mysqldump --single-transaction --routines --triggers --events`
   snapshot into isolated schema `pantheon_ops_f_final_rehearsal`.
3. Verified version `21`/clean state, business row counts, and active-key
   unique indexes match the source snapshot.
4. Ran `go test ./pkg/database/...` against isolated temporary schemas; the
   migration runner and repeat-run/legacy-key repair assertions passed.
5. Reviewed every `000013`-`000021` down migration. Destructive drops are
   limited to newly owned tables/columns; credential material is retained by
   design and migration `000021` is explicitly non-destructive.
6. Dropped the isolated rehearsal schema and removed the temporary dump.

## Production-Style Snapshot Rollback

1. Restored a new full snapshot copy into
   `pantheon_ops_prod_snapshot_rollback`; the source `pantheon_ops` schema was
   queried/dumped only and never migrated down.
2. The first `20 -> 19` rollback exposed real drift: the snapshot recorded
   version 21 while Ops objects from migrations 16-20 were absent. The legacy
   runtime bootstrap had advanced already-versioned schemas to latest without
   applying later Ops migrations.
3. Fixed bootstrap so clean recorded versions are never rewritten, added the
   durable `000022_ops_skipped_migrations_repair` compatibility marker, repaired
   missing version-owned objects before migration execution, made 19/20 down
   migrations conditional, and made credential backfill repeatable.
4. Rebuilt the snapshot copy and ran the actual `golang-migrate` engine through
   `21 -> 22 -> 16 -> 22`. Every transition finished clean.
5. Exact row counts remained unchanged after rollback/reapply:
   `biz_cmdb_group=4`, `biz_cmdb_host=8`, `biz_cmdb_label_schema=6`,
   `biz_deploy_package=3`, and `biz_deploy_template=3`.
6. Verified all four Ops tables, Deploy credential snapshot columns, and their
   indexes exist after reapply. Dropped the isolated snapshot database.

Direct rollback of the source database remains intentionally out of scope;
production rollout should use the verified snapshot/restore procedure and a
separate operator change window.
