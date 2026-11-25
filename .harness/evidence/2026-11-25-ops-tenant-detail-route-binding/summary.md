# Summary: bind detail routes now that Base enforces by route template

## What changed

Base v0.19.0 (#525) authorizes business routes against the Gin route template, and `permissionroutes.Covers` lets a `:param` policy segment cover only `:param` route segments. A `GET /hosts/:id` grant therefore no longer reaches `/hosts/export`, which is why these routes were deliberately left unbound.

| Key | Newly bound |
| --- | --- |
| `business:cmdb:host:detail` | `GET /hosts/:id` |
| `business:cmdb:group:detail` | `GET /groups/:id`, `GET /groups/:id/members` |
| `business:deploy:package:view` | `GET /packages/:id` |
| `business:deploy:template:list` | `GET /templates/:id` |
| `business:deploy:task:detail` | `GET /tasks/:id`, `/events`, `/events/history`, `/logs`, `/rollback` |

The cmdb detail keys already existed in the seeds and the frontend already gates the detail pages on them, so the API now matches the UI. Package and template have no detail key, so they ride on view/list. `POST /tasks/:id/rollback` has no seeded permission and stays unbound. Service and k8s were already bound.

## Tests

The route tests still modelled the old `keyMatch2` matcher, so they asserted the opposite of the new contract. They now check route templates through `permissionroutes.Covers`. The first run after binding went red on exactly the export routes, which confirms the old tests were guarding the old gap. New tests pin that detail never reaches export, export never reaches detail, and `task:detail` reaches neither `tasks/export` nor `POST rollback`.

## Gaps (explicit)

- The HTTP-level dual-tenant counterexample was **not run**. It needs the full auth stack and a live multi-tenant database; it moves to T19.
- `WithDataScope` applies no `tenant_id` filter, so row-level tenant isolation on detail lookups was **not independently proven** here. This task proves the authorization binding, not the data isolation.
- `check:harness-permission-contract` could not scan this worktree (directory name), so it is not evidence.
- `business/observability` fails locally without its Redis; unrelated.
- No independent CODE/ARCH review.
