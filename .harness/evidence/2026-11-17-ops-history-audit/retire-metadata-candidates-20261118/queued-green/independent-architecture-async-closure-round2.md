# Queued rejection: independent architecture closure

Author: c01_handoff_architecture. Persisted by Root as condensed transcription of the independent final response. Narrow CLEAR; C01-RETIRE-ASYNC-01 CLOSED. Original metadata CLEAR and async BLOCK reports remain. Whole C01 WATCH; wholeTaskComplete=false; archiveApproved=false.

The guard logs rejection before calling existing fenced failure settlement; settlement error logs retain both the rejection and transaction causes. failQueuedTask returns the fenced transaction error, and the existing Start/credential callers also handle it. Running-attempt protection, unfinished-host reason writes, release of this task's lease and recompute remain. No localOnly/skipOwner option, new owner exclusion, transaction or lock order was introduced.

Valid owner state permits failure settlement. Stale owner, missing snapshot and real owner query failure may still roll back, with errors now observable; terminal completion is not invented. Unknown-kind Install retains its existing failed-owner transition. Retire rejection has no owner Apply or CMDB writeback. Tests observe worker exit and preserve these boundaries.

The architect independently matched all 7 manifest/raw/current files to receipt before/after and all 730 entries unchanged. Root's real disposable MySQL CGO race/count1 run selected ^TestRetire: 5 groups PASS including 8 queue subcases, exit0/51.208s; no 1213/ERROR/DATA RACE/FAIL. Evidence: continuation-c01-retire-queued-all-green-20261118.log and receipt in the audit directory. The architect changed no files and reran no DB tests.

This closes only the introduced silent rejection. Full queue recovery, Service PATCH and wider owner policy, whole-repository regression, current-binary roles/provider and runtime/final acceptance remain open.
