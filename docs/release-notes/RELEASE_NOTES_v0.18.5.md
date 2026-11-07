# Pantheon Base v0.18.5 — corrective foundation candidate

Candidate only, not published. Latest verified official foundation remainsv0.18.4.

## Changes

Fix the shared purge-module test scanner root so a consumer repository external i18n reference cannot leak into a temporary four-row fixture. Preserve all original assertions and add a regression proving genuinely external references remain active after purge. Production behavior, interfaces, routes, permissions, UI and migrations are unchanged.

## Upgrade notes

After this candidate satisfies independent review, merge and existing hosted release gates, consume the official verified four assets through the standard Ops acquisition/rebuild path. Do not copy test source from Base main or treat test-source overlay as official consumption.

## Consumer impact

The unmodified official consumer should include the corrected shared test fixture. Ops dependency-graph normalization is a distinct consumer repair, not a Base production override. Raw full CGO race and applicable gates remain required before Ops replacement/closure.

## Gaps

S3 provider integration remains unverified/unwaived and maintenance freeze unsigned. Original12 tasks are not closed by this candidate source patch.
