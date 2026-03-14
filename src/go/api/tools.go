//go:build tools

package tools

// Pin dependencies used by the project before application code imports them.
// goose: database migration CLI (invoked via Makefile, not imported in app code).
// pgx/v5/stdlib: database/sql driver for PostgreSQL (used by goose and sqlc-generated code).

import (
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/pressly/goose/v3"
)
