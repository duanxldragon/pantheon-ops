---
title: Velero S3 v4 generator CLI summary
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

Candidate ready; local-only Generator, no remote execution or P0 approval.

- `python -B .local/ops-acceptance/resume-velero-s3-v4-test.py`: 25 tests pass;
  six new ignored Python scripts compile, 13 other remote functions and all six
  fixture functions remain AST-identical, original validate assertions retained.
- `python -B .local/ops-acceptance/resume-velero-s3-v4-root.py check-inputs`:
  eight original scripts, 17 product sources, binary and all v4 hashes verified.
- `npm run check:docs-frontmatter` and `npm run check:task-packet-template`: pass.
- Request SHA256: `ea4d2d5e406f95cb3e838067f66ed32c40e0e7b02152e117d25e59ad1da4f812`.
- Remote v4 SHA256: `63b2e9a90fb7b54fc522e6190ff6ad0cb353f199752dd7fea76b7d93cb47183a`.
- Root v4 SHA256: `20209264b26af232a1b8c68e93943b5496e37ff93098f100928b81df5b186898`.
- Exact commands and review conditions: `resume-velero-s3-v4-handoff.md`.
- Real server acceptance and independent root evaluation remain pending.
