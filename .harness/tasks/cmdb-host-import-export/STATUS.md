# Status: CMDB Host Import/Export

- Status: `implementation-complete; authorization gap open`
- Updated: `2026-11-07`
- Original evidence: `.harness/evidence/cmdb-host-import-export/summary.md`
- Current evidence: `.harness/evidence/2026-11-07-ops-planned-task-closeout/summary.md`

CSV template, transactional import and filtered export are implemented. Current full CGO race, frontend gates, CMDB runtime smoke (9/9), admin CSV API (7/7), and desktop/mobile rendered evidence now supplement the original session.

Restricted view/export/import/self roles with exact API grants pass 4/4. Key-only import/export roles still return 403 because the consumed Base permission binding table lacks a business extension. This is a Base-first gap, not a reason to broaden policies or patch shared Ops sources. The self-owner DataScope CSV scenario now verifies own-only export and rejection of cross-owner overwrite; final visual/function acceptance remains open. Historical evidence is retained unchanged.
