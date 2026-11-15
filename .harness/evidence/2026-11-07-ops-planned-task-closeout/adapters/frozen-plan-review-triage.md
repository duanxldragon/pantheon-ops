# Scoped independent review triage

APPROVE applies only to each supplied source excerpt/file. Final source corrections have follow-up APPROVE where listed below. Runtime evidence is separately verified.

| Finding                                                                                | Disposition                                                                                   | Verification                                                      |
| -------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| Secret Find/Count split could disagree; legacy duplicate keys lose secret completeness | Fixed: one row set returns map/completeness; duplicates and decrypt failures hide new strings | Real MySQL TestPreview race; reviewed-plan-secret-read APPROVE    |
| onStarted rejection reported an accepted start as failure                              | Fixed: accepted and refresh warning isolated; execution controls stay disabled                | UI 7/7; reviewed-plan-start-callback APPROVE                      |
| Dashboard retry waits with no feedback                                                 | Fixed: immediate loading, hides retry, retains snapshots; keeps nextAllowedAt budget          | Unit 8/8; rendered 2/2; reviewed-base-retry APPROVE               |
| Port/path display normalization suggestion                                             | Not adopted: render the saved frozen structural value; preview does not execute paths         | Existing grammar/budget/redaction tests and source excerpt review |
| Registry optional diagnostic detail                                                    | Deferred; no proved correctness failure                                                       | Registry source APPROVE; consumer integration review pending      |

Prior timeout and authentication-cancelled runs are retained and confer no approval. Provenance and prompt/output hashes are in frozen-plan-review-provenance.json. Base IAM consumers, editable ConditionBuilder and consumer archive projection are under additional bounded reviews. Full queue remains open.

## Further bounded scopes

**final-consumer-archive-projection**: APPROVE for projection function/config and supplied hostile tests. Helper normalization/call ordering and checker internals omitted. Full 62 overlay tests independently cover integration.

**final-base-iam-consumers**: 360-second timeout; no verdict, approval or waived gap.

**final-base-condition-editor**: 360-second timeout; no verdict, approval or waived gap.

**base-iam-registration-delta**: APPROVE for discovery/API route composition delta. Registry already reviewed; full role sync omitted from this delta. Existing grant/revoke tests remain separate evidence.

**base-condition-change-delta**: 360-second timeout; no verdict. ConditionBuilder final independent source review remains open.

**frozen-rollback-source**: APPROVE for frozen rollback transaction and scoped wrappers. No-rollbacks gorm-not-found suggestion is not applied: GetRollbackInfo explicitly treats it as HTTP success with null. Authentication/authorization middleware and helpers omitted; actual handler scope tests are separate evidence.
