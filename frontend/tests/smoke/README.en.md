# Smoke Test Layout

Chinese version: [README.md](./README.md)

Pantheon smoke tests are grouped by ownership boundary.

## Directories

- `platform/`: app shell, visual contracts, navigation chrome, and cross-domain full-page checks
- `system/`: auth, IAM, org, config, permissions, and governance flows
- `system/api/`: system-domain API smoke checks without browser pages
- `system/governance/`: module governance and permission-governance flows
- `business/<module>/`: business-domain smoke tests such as CMDB
- `helpers/`: shared helpers only

## Script Rules

- `npm run test:smoke:system` covers system and platform checks
- `npm run test:smoke:business` covers business checks
- each new business module should expose `test:smoke:business:<module>` and roll up into the business aggregate command

## Current Focus

The current matrix emphasizes:

- platform shell and visual stability
- system-domain page reachability and permission behavior
- governance workflows
- system import/export flows
- CMDB business smoke coverage

Known gaps still include SSH-collection browser coverage, business permission negative cases, and automated audit assertions for CMDB actions.
