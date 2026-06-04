# Pantheon Ops

Chinese version: [README.md](./README.md)

Pantheon Ops is the operations-management repository derived from the Pantheon Platform foundation. It uses CMDB and Deploy as its first business modules and focuses on operations resource inventory, grouping, tag standards, and deployment task management.

The repository keeps platform foundation capabilities as business runtime infrastructure, but its main evolution focus is `business/cmdb`, `business/deploy`, and future operations-domain modules. Ongoing evolution of generic backoffice capabilities should flow back to `pantheon-base`.

The default collaboration model is now: `pantheon-ops` consumes `pantheon-base` foundation releases instead of tracking `pantheon-base/main` directly.

## Positioning

- **Platform layer**: inherits the application shell, route composition, middleware, workbench, and cross-domain aggregate views from Pantheon Platform
- **System domains**: inherits auth/security, users/roles/permissions, menus, organization, configuration, dictionaries, audit
- **Business domains**: currently ships `business/cmdb` and `business/deploy`

## Recommended Reading Order

For Chinese-first onboarding, read:

1. [README.md](./README.md)
2. [docs/README.md](./docs/README.md)
3. [docs/PROJECT_INHERITANCE.md](./docs/PROJECT_INHERITANCE.md)
4. [DESIGN.md](./DESIGN.md)
5. [AGENTS.md](./AGENTS.md)

Then continue through the Chinese document index in [docs/README.md](./docs/README.md). Use [docs/README.en.md](./docs/README.en.md) only when an English companion is needed.

## Quick Start

### 1. Start infrastructure

```bash
docker compose up -d
```

Defaults:

- MySQL: `127.0.0.1:3306`
- Redis: `127.0.0.1:6379`
- default database: `pantheon_ops`

### 2. Start backend

```powershell
$env:PANTHEON_DSN='root:DHCCroot@2025@tcp(127.0.0.1:3306)/pantheon_ops?charset=utf8mb4&parseTime=True&loc=Local'
$env:PANTHEON_REDIS_ADDR='127.0.0.1:6379'
$env:PANTHEON_REDIS_PASSWORD='DHCCdhcc2025'
go run ./backend/cmd/server
```

### 3. Start frontend

```bash
cd frontend
npm install
npm run dev
```

## Common Commands

```bash
npm run upgrade:foundation:plan -- --manifest <bundle-root>\manifest.json --bundle <bundle-root>
npm run upgrade:foundation:apply -- --manifest <bundle-root>\manifest.json --bundle <bundle-root>
```

## Document Entry

- [docs/README.md](./docs/README.md): Chinese primary index
- [docs/PROJECT_INHERITANCE.md](./docs/PROJECT_INHERITANCE.md): inheritance lock, local scope, and override boundaries
- `Base release line + Base version` inside `docs/PROJECT_INHERITANCE.md` is the current consumer-version anchor
- [.agents/skills/README.md](./.agents/skills/README.md): repository-local Codex workflow skills
- [DESIGN.md](./DESIGN.md): repo-level design stance
- `../pantheon-base/docs/designs/QUALITY_AND_SECURITY_STRATEGY.md`: shared code-quality and security governance strategy
- [docs/README.en.md](./docs/README.en.md) and [docs/PROJECT_INHERITANCE.en.md](./docs/PROJECT_INHERITANCE.en.md): English companion entry points
