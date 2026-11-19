# C04/E01 architecture — first review

Original status BLOCK: E01 claim-error queue loss; C04 WATCH. Preserved from /root/c04_e01_architecture_gate's read-only first final. No repaired hashes or later GREEN substituted.

C04 protocol is Cluster row -> same-transaction credential read -> Binding row -> bounded provider. Legacy Cluster update takes Cluster row before advisory lock; new cancellable mutation uses no session advisory lock. No lock-order reversal found (cluster_service.go326/366/371; namespace_service.go113/115/128).

GREEN01 cancellation cases timed out waiting for provider exit (admission-lock-green01.log29; receipt exit1/sourceUnchanged true). At review the fixture consumed the body before its new gate and had changed hash; production/admission-test hashes still matched the old receipt. Preserve failed candidate and rerun cancellation, lock release, deadline, single connection, rejection and provider-error against new hashes.

E01 claimTarget boolean collapses transaction error and benign lost claim (transfer_engine.go127/159/241/257). Parent is already Running (distribution_service.go386), dispatcher only Pending (distribution_lifecycle.go217), so consumed work can be stranded. Fix typed claim error, forbid IO, guarded terminal persistence and injected regressions.

Cancellation first transactionally terminates parent/unfinished targets and then cancels active transfer; late updates are conditional. Stop cancels/joins dispatcher, lease monitor and workers before releasing dedicated MySQL connection. Progress monitor is cancelled with no independent join. Lease health checks every1s with5s DB timeout are bounded detection, not remote fencing; SFTP closes transport, cannot roll back accepted bytes.

Whole package UI/provider/human gaps remain. Root persists this historical BLOCK independently of round2.
