# Verification Evidence — 2026-09-19 RustFS upload target docs

## Scope
docs + i18n copy: list RustFS among verified S3-compatible upload targets, document the http:// endpoint prefix semantics. No runtime behavior change.

## Commands & results

| Check | Command | Result |
|---|---|---|
| JSON integrity | python json.load on builtin_locale_resources.json | OK, 5 locales, 10 keys contain "RustFS" |
| Backend (touched scope) | `go test ./modules/system/i18n/... ./modules/system/config/setting/...` | ok (both) |
| Docs frontmatter | `node scripts/frontmatter-check.mjs` | passed, 266 docs |
| Task packet template | `node scripts/check-task-packet-template.mjs` | OK |
| ESLint | `node node_modules/eslint/bin/eslint.js .` (frontend) | exit 0 |
| Type check | `node node_modules/typescript/bin/tsc -b` (frontend) | exit 0 |
| Build | `node node_modules/vite/bin/vite.js build` (frontend) | built in 698ms |
| i18n hardcode gate | `node scripts/check-i18n-hardcode.mjs` | passed (214 files) |
| i18n generated scope | `node scripts/check-i18n-generated-scope.mjs` | passed |
| i18n locale parity | `node scripts/audit-i18n-locales.mjs` | 2811 keys, missing=0 extra=0 empty=0 |

## Real-world S3 round-trip (outside CI, local environment)
RustFS v1.0.0 (windows-x86_64) against minio-go v7.2.1 (same client as pkg/upload):
MakeBucket / PutObject / StatObject / GetObject / ListObjects / RemoveObject all passed;
pantheon upload API stored an object into RustFS and a byte-level read-back comparison matched.
This is the practical basis for listing RustFS as a verified S3-compatible target.
