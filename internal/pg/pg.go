// Package pg is the PostgreSQL connector: schema discovery, old→new copy,
// health checks, and per-request write-set capture via logical decoding.
// It mirrors the MySQL (binlog) connector so replay/verify treat both alike.
package pg

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/pajamasi726/mocking-box/internal/config"
)

var systemSchemas = map[string]bool{
	"pg_catalog": true, "information_schema": true, "pg_toast": true,
}

func databaseName(d *config.Datastore) string {
	if d.Database != "" {
		return d.Database
	}
	return d.User
}

// connString builds a libpq connection string (special chars single-quoted).
func connString(d *config.Datastore, dbname string) string {
	q := func(v string) string { return "'" + strings.ReplaceAll(v, "'", "\\'") + "'" }
	parts := []string{
		"host=" + q(d.Host),
		fmt.Sprintf("port=%d", d.Port),
		"user=" + q(d.User),
		"password=" + q(d.Password),
		"dbname=" + q(dbname),
		"sslmode=prefer",
	}
	return strings.Join(parts, " ")
}

func connect(ctx context.Context, d *config.Datastore, dbname string) (*pgx.Conn, error) {
	return pgx.Connect(ctx, connString(d, dbname))
}

// -- discovery ----------------------------------------------------------------

type TableInfo struct {
	Name   string `json:"name"`
	Rows   int64  `json:"rows"`
	SizeMB int64  `json:"size_mb"`
}

type SchemaInfo struct {
	Name   string      `json:"name"`
	Tables []TableInfo `json:"tables"`
}

// Discover lists non-system schemas with each table's estimated row count and
// size, for the Copy DB picker.
func Discover(d *config.Datastore) ([]SchemaInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := connect(ctx, d, databaseName(d))
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT n.nspname, c.relname,
		       GREATEST(COALESCE(c.reltuples, 0), 0)::bigint,
		       COALESCE(pg_total_relation_size(c.oid) / 1048576, 0)::bigint
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'r' AND n.nspname NOT LIKE 'pg\_%' AND n.nspname <> 'information_schema'
		ORDER BY pg_total_relation_size(c.oid) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bySchema := map[string]*SchemaInfo{}
	var order []string
	for rows.Next() {
		var schema, table string
		var nrows, sizeMB int64
		if err := rows.Scan(&schema, &table, &nrows, &sizeMB); err != nil {
			return nil, err
		}
		if systemSchemas[schema] {
			continue
		}
		si := bySchema[schema]
		if si == nil {
			si = &SchemaInfo{Name: schema}
			bySchema[schema] = si
			order = append(order, schema)
		}
		si.Tables = append(si.Tables, TableInfo{Name: table, Rows: nrows, SizeMB: sizeMB})
	}
	out := make([]SchemaInfo, 0, len(order))
	for _, s := range order {
		out = append(out, *bySchema[s])
	}
	return out, rows.Err()
}

// -- health -------------------------------------------------------------------

// Health checks connectivity and that logical decoding is available
// (wal_level=logical) — required for write-set capture.
func Health(d *config.Datastore) (ok bool, detail string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := connect(ctx, d, databaseName(d))
	if err != nil {
		return false, err.Error()
	}
	defer conn.Close(ctx)

	var walLevel string
	if err := conn.QueryRow(ctx, "SHOW wal_level").Scan(&walLevel); err != nil {
		return false, "wal_level 조회 실패: " + err.Error()
	}
	if walLevel != "logical" {
		return false, "wal_level=" + walLevel + " (logical 필요 — write-set 캡처 불가)"
	}
	return true, "connected · wal_level=logical"
}
