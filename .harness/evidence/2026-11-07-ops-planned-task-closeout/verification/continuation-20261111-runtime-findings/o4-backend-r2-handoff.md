# O4 C-F current backend handoff (interim, 2026-11-11 r2)

Owned implementation is ready for exact current-source independent review. This report is not a final fresh assembly receipt, official consumption, runtime acceptance or P0 approval. Root still holds the full-source assembly freeze while other phase receipts are written under `.harness`.

## Current implementation

- Tenant-owned datasource/revision and tenant0 legacy/default/alert fences, private-source refusal, explicit protected POST test capped30s. Tenant-bound CRUD never starts detached outbound probes; legacy0 behavior stays established.
- Fresh actor/session/claim/provenance/action/DataScope/source owner/type/revision reads at entry, immediately before provider credentials leave, after provider, after status/history storage, and before DTO release. Source-only permissions are also refreshed for history/view reads and restore. No lock crosses provider work.
- Strict enum/catalog recipes and bounded decoder, metadata-only raw-editor history, catalog-only saved restoration with no auto-run; protected POST query routes and all39 exact privacy/audit/idempotency declarations including old expression GET deprecations. Near-match/default idempotency remains intact.
- Private tenant+actor history/views use owner guard -> sorted sources -> stable metadata locks, retained guards, atomic history retention500/2MiB/30days and views50/256KiB/revisions. Coalesced history updates one actor/tenant-owned session summary; relative duration recompiles to current UTC per poll. History storage failure has a safe independent flag.
- NEW hourly maintenance through Base `pkg/maintenance`,15s timeout, at most100 candidates per table/pass. Sweeps expired history and missing/soft-deleted/foreign datasource metadata under the same lock protocol. The actual MySQL guard barrier test proves expiry is rechecked after waiting, preserving a concurrently renewed summary. Append retention now locks all affected historical sources and retries changed enumerations at most3 times.
- Reserved migration37 exact shape/capability, retained-down and explicit fixture-only dirty recovery; supported startup helper integrates reviewed artifact/binary/schema and excludes incompatible old processes. Actual fresh-tree startup is still due after Root permits final assembly.

## Frontend HTTP contract

Canonical prefix `/api/v1/business/observability`. Source query routes: POST `/datasources/:id/query` (Prometheus/VM), `/logs` (Loki), `/traces` (Tempo). `GET /datasources/:id/query-fields` supplies current source catalog. Raw branch sends `{rawQuery,time,limit,resultMode}`; recipe branch sends `{recipe}` only. Tempo accepts catalog recipe only. Unknown fields, mixed branches, untrusted owner/config/name/provider strings and incompatible source intent reject.

Follow polling adds `historyMode:"coalesced"`; first response returns `data.historyId`, then reuse `historyId` with the same safe source/time/limit/result mode/recipe. Omitted/`record` uses ordinary history; nonzero historyId requires coalesced. Every poll reauthorizes; only enum intent/time/summary persist.

History endpoints: GET collection/item, DELETE collection/item, POST item `/restore`; saved views: GET collection/item, POST create, PUT item with `expectedRevision`, DELETE item, POST item `/restore`. Save body `{datasourceId,recipe,expectedRevision?}`. Restore is an explicit metadata/catalog operation and never executes provider queries. Raw history sets `expressionUnavailable:true`. Source catalog v1: metrics availability/instant; Loki stream/exists and severity/equals with info|warning|error; Tempo service/exists. Relative durations5m|15m|1h|6h|24h; resultMode table.

Permission keys: `business:observability:metric_source:{view,create,update,delete}`, logs:view, traces:view, query-history:{view,delete,restore}, saved-view:{view,create,update,delete,restore}. Current canonical route bindings only; aliases require Root's explicitly exact public IAM policy setup, never automatic grant synthesis.

## Fresh evidence

See adjacent JSON for exact hashes of every owned Go/SQL file, commands/results and evidence.

- CGO race `go test -mod=readonly -race -count=1 -p 2 -timeout 10m -v ./modules/business/observability ./modules/business/opsinit`: exit0,41.781s, sourceStable. One explicit skip: TestDatasourceLiveAcceptance; actual baseline migration executes.
- `go vet -mod=readonly ./modules/business/observability/... ./modules/business/opsinit/...`: exit0,2.343s. Matching build exit0,9.484s. Interim verification uses a business-only Go overlay into the historical verified paired stage; stage/shared sources are untouched.
- Actual assembled embedded `database.RunMigrations` helper, official-migration-v4:6/6 fresh/replay/retained-down/clean36-data-upgrade/three actual interrupted dirty37 cases. Ordinary restart refuses dirty; bounded explicit recovery and resume clean37 preserve original ciphertext/data SHA. Each recovery uses its own real no-connections quiescence file SHA, not a placeholder. v1/v2 failures and v3 placeholder fixture history remain preserved and are not promoted.
- Maintenance RED compile evidence and GREEN CGO race plus actual owner-lock/expiry interleaving are retained separately.

## Outstanding boundaries

1. Root must supply final full-source quiescence before a NEW reviewed --attempt; original sealed snapshot/stage/failures stay immutable. Re-run direct checks against that new current tree. Existing intermediate receipt does not certify later business edits.
2. User/tenant deletion has no observed public callback/narrow lifecycle read port. Current maintenance only sweeps O4 expiry/source orphans; deleted actor/tenant saved-view orphan handling needs Root's public-seam decision. No credential repository/model or direct system Service/Repository was introduced.
3. Actual public-product HTTP acceptance requires Root-supplied full and distinct authenticated403-denied sessions and4 source IDs; no remote was called in this lane. Full product race, independent review, official release/consumption and final visual/function acceptance remain Root gates.
