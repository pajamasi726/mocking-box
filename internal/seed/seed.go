// Package seed prepares the verification DB: it copies schemas+data from a
// source MySQL (a PITR temp instance, a dev DB, …) into the target datastore,
// pure-Go (no mysqldump dependency). The user provides connection info only —
// tables are created automatically, AUTO_INCREMENT counters carried over, and
// a seed marker is written so verify can preflight-check T0 alignment.
package seed

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/pajamasi726/mocking-box/internal/config"
)

var systemSchemas = map[string]bool{
	"mysql": true, "sys": true, "information_schema": true, "performance_schema": true,
	markerSchema: true,
}

const (
	markerSchema = "_mockingbox"
	batchRows    = 500
)

type Stats struct {
	Schemas []string
	Tables  int
	Rows    int64
}

type Options struct {
	Schemas       []string // empty = discover all non-system schemas on source
	ExcludeTables map[string]bool
	GoldenName    string // recorded in the marker for preflight matching
}

// RunPairs seeds each new datastore from the old datastore of the same name:
// the reference (old) DB IS the seed source. Scope (schemas/exclude) is read
// from the old datastore, since that's where discovery happens.
func RunPairs(oldStack, newStack *config.Stack, goldenName string) (*Stats, error) {
	total := &Stats{}
	seeded := 0
	for _, nd := range newStack.Datastores {
		dst := nd.MySQL()
		if dst == nil {
			continue
		}
		od := oldStack.DatastoreByName(nd.Name)
		if od == nil {
			return total, fmt.Errorf("datastore %q: no matching reference (old) DB — 설정에서 짝을 맞추세요", nd.Name)
		}
		src := od.MySQL()
		if src == nil || src.Host == "" {
			return total, fmt.Errorf("datastore %q: reference (old) connection is empty", nd.Name)
		}
		opts := Options{Schemas: od.Schemas, GoldenName: goldenName, ExcludeTables: map[string]bool{}}
		for _, t := range od.Exclude {
			if t != "" {
				opts.ExcludeTables[t] = true
			}
		}
		st, err := Run(src, dst, opts)
		if err != nil {
			return total, fmt.Errorf("datastore %q: %w", nd.Name, err)
		}
		total.Schemas = append(total.Schemas, st.Schemas...)
		total.Tables += st.Tables
		total.Rows += st.Rows
		seeded++
	}
	if seeded == 0 {
		return total, fmt.Errorf("검증 대상(신) DB가 없습니다 — 설정에서 추가하세요")
	}
	return total, nil
}

func Run(src, dst *config.MySQL, opts Options) (*Stats, error) {
	srcDB, err := open(src)
	if err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}
	defer srcDB.Close()
	dstDB, err := open(dst)
	if err != nil {
		return nil, fmt.Errorf("target: %w", err)
	}
	defer dstDB.Close()

	schemas := opts.Schemas
	if len(schemas) == 0 {
		if schemas, err = discoverSchemas(srcDB); err != nil {
			return nil, err
		}
	}
	if len(schemas) == 0 {
		return nil, fmt.Errorf("no non-system schemas found on source")
	}

	stats := &Stats{Schemas: schemas}
	for _, schema := range schemas {
		if err := copySchema(srcDB, dstDB, schema, opts.ExcludeTables, stats); err != nil {
			return nil, fmt.Errorf("schema %s: %w", schema, err)
		}
	}
	if err := writeMarker(dstDB, src, schemas, opts.GoldenName, stats); err != nil {
		return nil, fmt.Errorf("seed marker: %w", err)
	}
	return stats, nil
}

func open(m *config.MySQL) (*sql.DB, error) {
	db, err := sql.Open("mysql", m.DSN())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connect %s: %w", m.Addr(), err)
	}
	return db, nil
}

func discoverSchemas(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if rows.Scan(&s) == nil && !systemSchemas[s] {
			out = append(out, s)
		}
	}
	return out, rows.Err()
}

func copySchema(src, dst *sql.DB, schema string, exclude map[string]bool, stats *Stats) error {
	log.Printf("[seed] schema %s: recreating on target", schema)
	if _, err := dst.Exec("DROP DATABASE IF EXISTS `" + schema + "`"); err != nil {
		return err
	}
	var createDB string
	var name string
	if err := src.QueryRow("SHOW CREATE DATABASE `"+schema+"`").Scan(&name, &createDB); err != nil {
		return err
	}
	if _, err := dst.Exec(createDB); err != nil {
		return err
	}

	tables, err := listTables(src, schema)
	if err != nil {
		return err
	}
	for _, table := range tables {
		if exclude[table] || exclude[schema+"."+table] {
			log.Printf("[seed]   %s.%s: excluded", schema, table)
			continue
		}
		rowsCopied, err := copyTable(src, dst, schema, table)
		if err != nil {
			return fmt.Errorf("table %s: %w", table, err)
		}
		stats.Tables++
		stats.Rows += rowsCopied
		log.Printf("[seed]   %s.%s: %d row(s)", schema, table, rowsCopied)
	}
	return nil
}

func listTables(db *sql.DB, schema string) ([]string, error) {
	rows, err := db.Query(
		"SELECT TABLE_NAME FROM information_schema.TABLES"+
			" WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME", schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if rows.Scan(&t) == nil {
			out = append(out, t)
		}
	}
	return out, rows.Err()
}

func copyTable(src, dst *sql.DB, schema, table string) (int64, error) {
	// schema (CREATE TABLE includes indexes, charset, AUTO_INCREMENT counter)
	var name, createSQL string
	if err := src.QueryRow("SHOW CREATE TABLE `"+schema+"`.`"+table+"`").Scan(&name, &createSQL); err != nil {
		return 0, err
	}
	if _, err := dst.Exec("USE `" + schema + "`"); err != nil {
		return 0, err
	}
	if _, err := dst.Exec(createSQL); err != nil {
		return 0, err
	}

	rows, err := src.Query("SELECT * FROM `" + schema + "`.`" + table + "`")
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	if len(cols) == 0 {
		return 0, nil
	}

	colList := "`" + strings.Join(cols, "`,`") + "`"
	placeholderRow := "(" + strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",") + ")"

	var total int64
	batch := make([]any, 0, batchRows*len(cols))
	rowsInBatch := 0
	flush := func() error {
		if rowsInBatch == 0 {
			return nil
		}
		stmt := "INSERT INTO `" + schema + "`.`" + table + "` (" + colList + ") VALUES " +
			strings.TrimSuffix(strings.Repeat(placeholderRow+",", rowsInBatch), ",")
		if _, err := dst.Exec(stmt, batch...); err != nil {
			return err
		}
		total += int64(rowsInBatch)
		batch = batch[:0]
		rowsInBatch = 0
		return nil
	}

	scan := make([]sql.RawBytes, len(cols))
	scanPtrs := make([]any, len(cols))
	for i := range scan {
		scanPtrs[i] = &scan[i]
	}
	for rows.Next() {
		if err := rows.Scan(scanPtrs...); err != nil {
			return total, err
		}
		for _, raw := range scan {
			if raw == nil {
				batch = append(batch, nil)
			} else {
				val := make([]byte, len(raw))
				copy(val, raw)
				batch = append(batch, val)
			}
		}
		rowsInBatch++
		if rowsInBatch >= batchRows {
			if err := flush(); err != nil {
				return total, err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return total, err
	}
	return total, flush()
}

// -- seed marker: lets verify preflight-check that the target was seeded ------

type Marker struct {
	SeededAt string
	Source   string
	Schemas  string
	Golden   string
	Tables   int
	Rows     int64
}

func writeMarker(dst *sql.DB, src *config.MySQL, schemas []string, goldenName string, stats *Stats) error {
	if _, err := dst.Exec("CREATE DATABASE IF NOT EXISTS `" + markerSchema + "`"); err != nil {
		return err
	}
	if _, err := dst.Exec("CREATE TABLE IF NOT EXISTS " + markerSchema + ".seed_info (" +
		"`id` BIGINT AUTO_INCREMENT PRIMARY KEY," +
		"`seeded_at` DATETIME NOT NULL," +
		"`source` VARCHAR(255) NOT NULL," +
		"`schemas` VARCHAR(500) NOT NULL," +
		"`golden` VARCHAR(255) NOT NULL," +
		"`tables_copied` INT NOT NULL," +
		"`rows_copied` BIGINT NOT NULL)"); err != nil {
		return err
	}
	_, err := dst.Exec(
		"INSERT INTO "+markerSchema+".seed_info (`seeded_at`,`source`,`schemas`,`golden`,`tables_copied`,`rows_copied`)"+
			" VALUES (?,?,?,?,?,?)",
		time.Now().Format("2006-01-02 15:04:05"), src.Addr(),
		strings.Join(schemas, ","), goldenName, stats.Tables, stats.Rows,
	)
	return err
}

// ReadMarker returns the latest seed marker on the datastore (nil if never seeded).
func ReadMarker(m *config.MySQL) (*Marker, error) {
	db, err := open(m)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	row := db.QueryRow("SELECT `seeded_at`,`source`,`schemas`,`golden`,`tables_copied`,`rows_copied`" +
		" FROM " + markerSchema + ".seed_info ORDER BY `id` DESC LIMIT 1")
	mk := &Marker{}
	if err := row.Scan(&mk.SeededAt, &mk.Source, &mk.Schemas, &mk.Golden, &mk.Tables, &mk.Rows); err != nil {
		return nil, nil // no marker (or table missing) — treated as "never seeded"
	}
	return mk, nil
}

// -- discovery: list schemas and tables (with size) for the UI checkbox picker

type TableInfo struct {
	Name   string `json:"name"`
	Rows   int64  `json:"rows"`
	SizeMB int64  `json:"size_mb"`
}

type SchemaInfo struct {
	Name   string      `json:"name"`
	Tables []TableInfo `json:"tables"`
}

// Discover connects to a source MySQL and returns its non-system schemas with
// each table's approximate row count and size — powers the seed picker.
func Discover(src *config.MySQL) ([]SchemaInfo, error) {
	db, err := open(src)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	schemas, err := discoverSchemas(db)
	if err != nil {
		return nil, err
	}
	out := make([]SchemaInfo, 0, len(schemas))
	for _, schema := range schemas {
		rows, err := db.Query(
			"SELECT TABLE_NAME, COALESCE(TABLE_ROWS,0), COALESCE(ROUND((DATA_LENGTH+INDEX_LENGTH)/1048576),0)"+
				" FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE='BASE TABLE'"+
				" ORDER BY (DATA_LENGTH+INDEX_LENGTH) DESC", schema)
		if err != nil {
			return nil, err
		}
		si := SchemaInfo{Name: schema}
		for rows.Next() {
			var ti TableInfo
			if rows.Scan(&ti.Name, &ti.Rows, &ti.SizeMB) == nil {
				si.Tables = append(si.Tables, ti)
			}
		}
		rows.Close()
		out = append(out, si)
	}
	return out, nil
}
