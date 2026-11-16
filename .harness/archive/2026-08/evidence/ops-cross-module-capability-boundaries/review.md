# Findings-First Review

1. **Medium residual risk:** the current CMDB capability still stores legacy
   installed-component JSON on Host. This task routes the write through the
   owner command but does not redesign that model; ServiceInstance owns that
   follow-up.
2. **Low:** constructors retain a typed provider fallback for existing unit
   tests that instantiate Deploy/K8s services directly. Production overlay
   wiring always injects the shared provider instances.

No cross-module GORM model, raw table join, loopback HTTP call, base/system
change, or REST response change was introduced. The checker is intentionally
strict for runtime module files and allowlists owner models, seeds, and tests.
