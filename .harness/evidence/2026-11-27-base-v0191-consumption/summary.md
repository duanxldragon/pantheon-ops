# Ops consumes Pantheon Base v0.19.1

The consumer locks `pantheon-base-v0.19.1` at `7f5cc804b86394724d42a60dbbb559b610fa2279`. The official
bundle, repository snapshot and both SHA-256 sidecars match the GitHub
release digests. The annotated tag, release manifest, archive header, lock
and rebuild report all name the same commit. [Base release](https://github.com/duanxldragon/pantheon-base/releases/tag/pantheon-base-v0.19.1)

## Rebuild

Overlay and shared drift passed on the staged tree and the consumer. The
checks covered 54 declared business paths, 878 shared files and 20 derived
migration files. Comparing two official rebuilds isolated the v0.19.1 delta.
The shared authorization sources, tests, contracts and product version come
from that output. Eight pre-existing generated-file differences and the
two legacy release indexes were retained from Ops main.

## Verification

- The Windows MinGW CGO race passed 77 test packages
  with zero failures and zero race reports. 4 optional test(s) skipped,
  as recorded in the [race receipt](./race.json). The server rebuilt to the
  same binary SHA-256. The test schemas and owned Redis listeners were
  removed. The development schema ended at version 41 with
  `dirty=false`, the same as before testing.
- Frontend lint, type checking, build and 430 unit tests passed on the
  preserved frontend inputs. The existing AlertCenter lint warning remains.
- Overlay tests passed 59/59. Root script tests passed 255/255.
  Method consumption, docs, encoding and GORM SQL literal checks passed.
  The supplemental security pattern scan reported the same 28 matches on
  the v0.19.0 rebuild and the v0.19.1 candidate.
- The [independent code and inheritance reviews](./review.md) approved the
  candidate. Local duplication was 1.08%. The Ops baseline SonarCloud
  check was already ERROR on `new_security_rating`.

Hosted checks, smoke and the squash commit are available in the consuming
pull request. Inheritance acceptance is reached when Ops main contains this
lock, the matching product VERSION and the rebuild report.
