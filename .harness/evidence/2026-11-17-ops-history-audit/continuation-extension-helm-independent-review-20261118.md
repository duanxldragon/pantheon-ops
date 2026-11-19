# Extension startup / error semantics / Helm independent review

Narrow CLEAR, whole packages WATCH. Returned by /root/c04_e01_architecture_gate; root persists. No new scoped blocker, no source edits or independent test rerun by reviewer.

FileDist provider/migration precedes Start; failed Start cancels/Stops then existing void Init panics. Route registration panic defers cleanup. sync.Once cleanup orders cancel->observer.Stop->join Start goroutine->distributor.Stop. Reconciler Stop-before-Start is safe; combination-layer observerDone provides join.

Explicit six FileDist canonical keys preserve typed authorization/error precedence. Argo sync conflict HTTP409, other typed classes keep legacy HTTP200/envelope error codes; raw provider/SQL use safe fallback. HTTP contracts were read. Chinese/English seeds/canonical agree.

Helm new status registered. List captures raw DB status into response-only storedStatus then projects status/effectiveStatus registered; gorm ignored fields do not alter DB defaults/history. Four regressions including HTTP create/legacy-list assert DB legacy deployed unchanged. README discloses no Chart install/upgrade/rollback/uninstall; delete is record removal.

Startup17/17, Helm4/4 current hashes match their frozen handoffs. Central filtered GREEN CGO1/required disposableMySQL/-race exit0,748 before/after stable; twelve lane backend hashes independently match. Package durations opsinit1.115s/filedist5.021s/Argo3.720s/Helm1.461s. Original E03 actual RED create+legacy-list exit1/0.136s retained.

WATCH: Ops signal context independent of Base main. Base shutdown and HTTP bind failure do not await/cancel extension cleanup; helper join cannot establish process shutdown. Public scheduled actor/source authority adapter absent: OwnerLifecycle existence is not a grant, selection path nilfailclosed while existing legacy all/admin path remains. Real Init routepanic/OSsignal/runtime seed persistence, remote fencing/byte rollback/progress join and final fullrace/officialoverlay/provider/frontend/human gates still unproved by this review. Recommend Base-first L2 public lifecycle and job/source authority contracts followed by formal release consumption; no business shortcut.
