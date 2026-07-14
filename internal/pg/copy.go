package pg

import (
	"context"
	"fmt"
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
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

	tables, err := listTables(ctx, sconn, schema)
	if err != nil {
		return err
	}
	for _, table := range tables {
		if exclude[table] || exclude[schema+"."+table] {
			continue
		}
		n, err := copyTable(ctx, sconn, dconn, schema, table)
		if err != nil {
			return fmt.Errorf("table %s: %w", table, err)
		}
		stats.Tables++
		stats.Rows += n
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

	// copy rows: read all, COPY into target
	src, err := sconn.Query(ctx, fmt.Sprintf("SELECT * FROM %s", qtable))
	if err != nil {
		return 0, err
	}
	defer src.Close()
	var data [][]any
	for src.Next() {
		vals, err := src.Values()
		if err != nil {
			return 0, err
		}
		data = append(data, vals)
	}
	if err := src.Err(); err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, nil
	}
	n, err := dconn.CopyFrom(ctx, pgx.Identifier{schema, table}, colNames, pgx.CopyFromRows(data))
	if err != nil {
		return 0, err
	}
	// keep serial/identity sequences ahead of copied data
	resetSequences(ctx, dconn, schema, table, cols)
	return n, nil
}

func tableColumns(ctx context.Context, conn *pgx.Conn, schema, table string) ([]columnDef, error) {
	rows, err := conn.Query(ctx, `
		SELECT column_name,
		       CASE WHEN data_type = 'USER-DEFINED' THEN udt_name
		            WHEN character_maximum_length IS NOT NULL THEN data_type || '(' || character_maximum_length || ')'
		            WHEN numeric_precision IS NOT NULL AND data_type IN ('numeric','decimal') THEN data_type || '(' || numeric_precision || ',' || COALESCE(numeric_scale,0) || ')'
		            ELSE data_type END,
		       is_nullable = 'NO', column_default
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`, schema, table)
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
