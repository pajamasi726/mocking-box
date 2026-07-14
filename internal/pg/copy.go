package pg

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/pajamasi726/mocking-box/internal/config"
)

// CopyStats mirrors seed.Stats for the PostgreSQL path.
type CopyStats struct {
	Schemas []string
	Tables  int
	Rows    int64
}

// Copy replicates schemas+data from a source datastore into a target datastore
// (same database name). Pure pgx — recreates each schema, its tables (columns,
// PK, defaults), and copies rows via COPY. Excluded tables are skipped.
func Copy(src, dst *config.Datastore, schemas []string, exclude map[string]bool) (*CopyStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	sconn, err := connect(ctx, src, databaseName(src))
	if err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}
	defer sconn.Close(ctx)
	dconn, err := connect(ctx, dst, databaseName(dst))
	if err != nil {
		return nil, fmt.Errorf("target: %w", err)
	}
	defer dconn.Close(ctx)

	if len(schemas) == 0 {
		if schemas, err = listSchemas(ctx, sconn); err != nil {
			return nil, err
		}
	}
	stats := &CopyStats{Schemas: schemas}
	for _, schema := range schemas {
		if err := copySchema(ctx, sconn, dconn, schema, exclude, stats); err != nil {
			return nil, fmt.Errorf("schema %s: %w", schema, err)
		}
	}
	if err := writeMarker(ctx, dconn, src.Addr(), schemas, stats); err != nil {
		return nil, fmt.Errorf("copy marker: %w", err)
	}
	return stats, nil
}

func listSchemas(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	rows, err := conn.Query(ctx,
		`SELECT nspname FROM pg_namespace WHERE nspname NOT LIKE 'pg\_%' AND nspname <> 'information_schema' ORDER BY nspname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		if !systemSchemas[s] {
			out = append(out, s)
		}
	}
	return out, rows.Err()
}

func copySchema(ctx context.Context, sconn, dconn *pgx.Conn, schema string, exclude map[string]bool, stats *CopyStats) error {
	if _, err := dconn.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", pgx.Identifier{schema}.Sanitize())); err != nil {
		return err
	}
	if _, err := dconn.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", pgx.Identifier{schema}.Sanitize())); err != nil {
		return err
	}
	// custom enum types must exist before tables that use them
	if err := copyEnums(ctx, sconn, dconn, schema); err != nil {
		return fmt.Errorf("enum types: %w", err)
	}

	tables, err := listTables(ctx, sconn, schema)
	if err != nil {
		return err
	}
	for _, table := range tables {
		if exclude[table] || exclude[schema+"."+table] {
			continue
		}
		log.Printf("[copy] %s.%s …", schema, table)
		n, err := copyTable(ctx, sconn, dconn, schema, table)
		if err != nil {
			return fmt.Errorf("table %s: %w", table, err)
		}
		stats.Tables++
		stats.Rows += n
		log.Printf("[copy]   %s.%s: %d rows", schema, table, n)
	}
	return nil
}

// copyEnums recreates the schema's enum types on the target (labels in order).
func copyEnums(ctx context.Context, sconn, dconn *pgx.Conn, schema string) error {
	rows, err := sconn.Query(ctx, `
		SELECT t.typname, array_agg(e.enumlabel ORDER BY e.enumsortorder)
		FROM pg_type t
		JOIN pg_enum e ON e.enumtypid = t.oid
		JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = $1
		GROUP BY t.typname`, schema)
	if err != nil {
		return err
	}
	defer rows.Close()
	type enumDef struct {
		name   string
		labels []string
	}
	var enums []enumDef
	for rows.Next() {
		var e enumDef
		if err := rows.Scan(&e.name, &e.labels); err != nil {
			return err
		}
		enums = append(enums, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, e := range enums {
		quoted := make([]string, len(e.labels))
		for i, l := range e.labels {
			quoted[i] = "'" + strings.ReplaceAll(l, "'", "''") + "'"
		}
		stmt := fmt.Sprintf("CREATE TYPE %s AS ENUM (%s)",
			pgx.Identifier{schema, e.name}.Sanitize(), strings.Join(quoted, ", "))
		if _, err := dconn.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func listTables(ctx context.Context, conn *pgx.Conn, schema string) ([]string, error) {
	rows, err := conn.Query(ctx,
		`SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		 WHERE n.nspname = $1 AND c.relkind = 'r' ORDER BY c.relname`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type columnDef struct {
	name    string
	typ     string
	notNull bool
	deflt   *string
}

func copyTable(ctx context.Context, sconn, dconn *pgx.Conn, schema, table string) (int64, error) {
	cols, err := tableColumns(ctx, sconn, schema, table)
	if err != nil {
		return 0, err
	}
	if len(cols) == 0 {
		return 0, nil
	}

	// serial/identity columns default to nextval('<seq>') — create those
	// sequences first so the table's DEFAULT resolves.
	for _, c := range cols {
		if c.deflt == nil {
			continue
		}
		if seq := sequenceName(*c.deflt); seq != "" {
			_, _ = dconn.Exec(ctx, "CREATE SEQUENCE IF NOT EXISTS "+seq)
		}
	}

	// CREATE TABLE with columns, defaults, not-null; then add PK separately.
	var defs []string
	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.name
		def := pgx.Identifier{c.name}.Sanitize() + " " + c.typ
		if c.deflt != nil {
			def += " DEFAULT " + *c.deflt
		}
		if c.notNull {
			def += " NOT NULL"
		}
		defs = append(defs, def)
	}
	qtable := pgx.Identifier{schema, table}.Sanitize()
	if _, err := dconn.Exec(ctx, fmt.Sprintf("CREATE TABLE %s (%s)", qtable, strings.Join(defs, ", "))); err != nil {
		return 0, err
	}
	if pk, err := primaryKey(ctx, sconn, schema, table); err == nil && len(pk) > 0 {
		quoted := make([]string, len(pk))
		for i, c := range pk {
			quoted[i] = pgx.Identifier{c}.Sanitize()
		}
		_, _ = dconn.Exec(ctx, fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (%s)", qtable, strings.Join(quoted, ", ")))
	}

	// stream rows source→target via the COPY protocol (binary), piped — O(1)
	// memory regardless of table size (18M-row tables must not buffer in Go).
	copyN, err := streamCopy(ctx, sconn, dconn, qtable)
	if err != nil {
		return 0, err
	}
	// keep serial/identity sequences ahead of copied data
	resetSequences(ctx, dconn, schema, table, cols)
	return copyN, nil
}

// streamCopy pipes `COPY <tbl> TO STDOUT (FORMAT binary)` on the source into
// `COPY <tbl> FROM STDIN (FORMAT binary)` on the target.
func streamCopy(ctx context.Context, sconn, dconn *pgx.Conn, qtable string) (int64, error) {
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		_, err := sconn.PgConn().CopyTo(ctx, pw, "COPY "+qtable+" TO STDOUT (FORMAT binary)")
		pw.CloseWithError(err) // nil err → clean EOF for the reader
		errCh <- err
	}()
	tag, err := dconn.PgConn().CopyFrom(ctx, pr, "COPY "+qtable+" FROM STDIN (FORMAT binary)")
	pr.CloseWithError(err)
	if srcErr := <-errCh; srcErr != nil {
		return 0, fmt.Errorf("read source: %w", srcErr)
	}
	if err != nil {
		return 0, fmt.Errorf("write target: %w", err)
	}
	return tag.RowsAffected(), nil
}

func tableColumns(ctx context.Context, conn *pgx.Conn, schema, table string) ([]columnDef, error) {
	// pg_catalog.format_type renders the exact type text (arrays, enums, custom
	// types, precision/length) — unlike information_schema which reports "ARRAY".
	rows, err := conn.Query(ctx, `
		SELECT a.attname,
		       pg_catalog.format_type(a.atttypid, a.atttypmod),
		       a.attnotnull,
		       pg_catalog.pg_get_expr(d.adbin, d.adrelid)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE n.nspname = $1 AND c.relname = $2 AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []columnDef
	for rows.Next() {
		var c columnDef
		if err := rows.Scan(&c.name, &c.typ, &c.notNull, &c.deflt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func primaryKey(ctx context.Context, conn *pgx.Conn, schema, table string) ([]string, error) {
	rows, err := conn.Query(ctx, `
		SELECT a.attname FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE i.indrelid = ($1 || '.' || $2)::regclass AND i.indisprimary
		ORDER BY array_position(i.indkey, a.attnum)`,
		pgx.Identifier{schema}.Sanitize(), pgx.Identifier{table}.Sanitize())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// sequenceName extracts "schema.seq" from a nextval('schema.seq'::regclass) default.
func sequenceName(deflt string) string {
	const marker = "nextval('"
	i := strings.Index(deflt, marker)
	if i < 0 {
		return ""
	}
	rest := deflt[i+len(marker):]
	j := strings.Index(rest, "'")
	if j < 0 {
		return ""
	}
	return rest[:j] // e.g. booster.review_id_seq
}

// resetSequences advances owned sequences past the max copied value so new
// inserts on the target don't collide (serial/identity columns).
func resetSequences(ctx context.Context, conn *pgx.Conn, schema, table string, cols []columnDef) {
	for _, c := range cols {
		if c.deflt == nil || !strings.Contains(*c.deflt, "nextval(") {
			continue
		}
		qtable := pgx.Identifier{schema, table}.Sanitize()
		qcol := pgx.Identifier{c.name}.Sanitize()
		_, _ = conn.Exec(ctx, fmt.Sprintf(
			"SELECT setval(pg_get_serial_sequence('%s', '%s'), COALESCE((SELECT MAX(%s) FROM %s), 1))",
			schema+"."+table, c.name, qcol, qtable))
	}
}

const markerSchema = "_mockingbox"

func writeMarker(ctx context.Context, conn *pgx.Conn, source string, schemas []string, stats *CopyStats) error {
	if _, err := conn.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+markerSchema); err != nil {
		return err
	}
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+markerSchema+`.copy_info (
		id SERIAL PRIMARY KEY, copied_at TIMESTAMP NOT NULL DEFAULT now(),
		source TEXT NOT NULL, schemas TEXT NOT NULL, tables_copied INT NOT NULL, rows_copied BIGINT NOT NULL)`); err != nil {
		return err
	}
	_, err := conn.Exec(ctx,
		"INSERT INTO "+markerSchema+".copy_info (source, schemas, tables_copied, rows_copied) VALUES ($1,$2,$3,$4)",
		source, strings.Join(schemas, ","), stats.Tables, stats.Rows)
	return err
}
