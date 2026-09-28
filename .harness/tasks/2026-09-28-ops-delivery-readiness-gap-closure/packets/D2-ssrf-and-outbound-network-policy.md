# D2: SSRF And Outbound Network Policy

**Priority:** P0  
**Estimate:** 1-2 engineer-days after policy ownership is decided  
**State:** blocked on maintainer and Base policy decision

## In

- Allowed URL schemes, ports, DNS resolution, IP and private-network rules.
- Redirect, DNS-rebinding, timeout, response-size, and credential-forwarding behavior.
- Audit and error semantics for denied datasource proxy requests.
- Base-first implementation and Ops consumer verification.

## Out

- Enabling arbitrary stored URLs.
- A business-local network policy copy.
- Production endpoint changes before the policy is reviewed.

## Exit Evidence

- Approved policy document and owning Base task/release.
- Tests for allowed, denied, redirect, private-address, rebinding, and timeout cases.
- Overlay/build/smoke results after consumption.

## Stop Conditions

Keep datasource proxy queries disabled if the policy cannot distinguish approved private services from arbitrary internal targets.
