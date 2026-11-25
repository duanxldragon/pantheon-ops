# Review: credential visibility capability

Independent CODE/ARCH review: **not performed**. Recorded gap, not an approval.

Check: the fail-closed paths in `credentialVisibleQuery`, that `VisibleIDs` uses exactly the `WithDataScope` the old subquery did, and that adding a method to `BizScopeReader` has no implementer left without it.
