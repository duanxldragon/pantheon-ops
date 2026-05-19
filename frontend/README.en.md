# Pantheon Ops Frontend Workspace

Chinese version: [README.md](./README.md)

This frontend is no longer just the default Vite template surface.

Use this directory for the Pantheon Ops backoffice application built with React, TypeScript, Vite, Arco Design, Zustand, and i18next.

## Expected Responsibilities

- app shell and route registration
- platform and system-domain pages inherited from Pantheon Base
- business-domain pages for `business/cmdb`, `business/deploy`, and future ops modules
- shared component registry, menu contract handling, and i18n integration

## Baseline Commands

```bash
npm install
npm run dev
npm run build
```

Common smoke commands:

```bash
npm run test:smoke:system
npm run test:smoke:business
```

## Development Rules

- preserve Pantheon Base layout, token, and state conventions
- do not treat the generated Vite README as the project guide
- keep user-facing copy localized
- maintain permission separation across menu, page, action, and API layers

For project-specific architecture and design rules, use the repo root README and `docs/` as the primary documentation surface.
