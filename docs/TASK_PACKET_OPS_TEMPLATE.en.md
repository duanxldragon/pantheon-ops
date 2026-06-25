# Pantheon Ops Task Packet Template

Chinese version: [TASK_PACKET_OPS_TEMPLATE.md](./TASK_PACKET_OPS_TEMPLATE.md)

This is the smallest reusable task-packet example for `pantheon-ops`.

Use it for:

- `business/cmdb`
- `business/deploy`
- `business/bizscope`
- `base -> ops` inheritance sync

Copy and fill:

```text
Target repo: pantheon-ops
Layer: business/cmdb / business/deploy / business/bizscope / inheritance-sync
Task mode: review / implement / ui / inheritance-sync / smoke / docs
Read first:
- pantheon-ops/AGENTS.md
- pantheon-ops/docs/PROJECT_INHERITANCE.md
- pantheon-base/DESIGN.md
- pantheon-base/AGENTS.md
- pantheon-base/docs/README.md
- matching base contract / design / acceptance docs
- matching BUSINESS_* docs

Implementation scope:
- state whether this turn is a business-feature closure or a shared-foundation sync
- state which business or platform areas are explicitly out of scope
- apply the minimal complexity ladder first: skip / reuse base or ops capability / standard library / native platform / installed dependency / one local expression / minimum new code

Sync expectation:
- business-only
- or `base -> ops` sync
- if the root cause belongs to a shared shell or system domain, route it back to base instead of overriding locally in ops

Verification:
- Backend: `go test ...` / `go test ./...`
- Frontend: `npm run build`
- Smoke: business smoke / end-to-end smoke / `none`
- UI work must attach rendered evidence or record why rendering was not produced

Stop points:
- pause before changing the base version, deleting inherited overrides, changing business schema, or backporting shared logic
```

Additional sync requirements:

- record the base commit
- record which shared paths were synced and which were intentionally skipped
- record whether local `business/*` paths stayed intact
- record the minimum validation result for both base and ops
