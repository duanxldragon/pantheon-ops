# Business Migration Evidence

Implemented migration `000012_business_schema` for all currently modeled Ops
business tables. Production uses the base-owned versioned runner by default;
module AutoMigrate remains an explicit development-only path controlled by
`PANTHEON_AUTO_MIGRATE=true`.

Validation passed for empty-schema creation, generated-column effective
uniqueness, duplicate task-host rejection, rollback, repeat-up, business Go
tests, database package tests, and `go vet`. MySQL evidence uses local MySQL
8.0.36 and is recorded in `schema-empty.txt`, `schema-upgrade.txt`, and
`schema-repeat.txt`.

Maintainer-deferred scope: the additive migration does not rename or replace
legacy indexes on tables created by prior AutoMigrate runs. Production deployment
will generate that model-derived upgrade script in a separate task.
