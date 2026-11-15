# Rendered P0 checkpoint — 2026-11-09

Inspected remote v13 screenshots for deployment target selection (dark,390x844) and recorded host/attempt workbench (dark,390x844). Selection labels wrap, explicit resolution/deduplication counts are visible, and navigation is collapsed. Historical attempt, UTC observation, failure navigation and log text are present. Smoke layout assertions passed. The shared ExecutionStepRail still renders a white surface and weak status contrast in dark mode; this remains a Base-first release dependency. No full visual approval.

Remote v13 passed15/18 cases. Both deploy selection/preview and both workbench cases pass after waiting for actual initial UI readiness and closing the browser page before API fixture teardown. Complex scenario budgets remain90s, interaction expectations10s. The proxy uses HTTP/1.1 and static gzip; isolated SSE headers/events remain incremental and error status/body are preserved.

Tempo dark failed before any datasource request. Loki light/dark failed in auth login after90s, so neither failure proves a Loki product regression. Fresh SSH checks also failed (session establishment/authentication timeout) during this period. The Argo pinned upstream test installer was launched as PID1231440 in the owned namespace, but its final state is not verified. No new provider installation or browser acceptance is claimed. Existing v13 owned backend/web and forwarding were stopped successfully.

The initial Nginx-only v12 attempt failed because the executable is unavailable in the host environment; its owned backend was cleaned. The existing port80 service was untouched. v11 isolated editor diagnostic passed with zero recorded page errors. Failed diagnostics and v6/v10/v12/v13 receipts are retained.
