// Package binlog captures per-request write-sets from the MySQL ROW binlog.
//
// A Capture connects as a replication client (the same mechanism Debezium
// uses) and streams row events on a background goroutine. The replayer opens
// an attribution window per request: every row change committed between
// window start and quiesce belongs to that request (Level 0 attribution).
//
// With binlog_rows_query_log_events=ON the originating SQL text (including
// any /* rid=... */ comment injected by SQLCommenter/datasource-proxy) is
// attached to each change, ready for Level 1 attribution.
package binlog

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	_ "github.com/go-sql-driver/mysql"

	"github.com/pajamasi726/mocking-box/internal/config"
)

// RowChange is one committed row change, normalized from a binlog row event.
type RowChange struct {
	Table  string         `json:"table"` // schema.table
	Op     string         `json:"op"`    // INSERT | UPDATE | DELETE
	Before map[string]any `json:"before,omitempty"`
	After  map[string]any `json:"after,omitempty"`
	Query  string         `json:"query,omitempty"` // Rows_query text when available
}

var systemSchemas = map[string]bool{
	"mysql": true, "sys": true, "information_schema": true, "performance_schema": true,
}

type Capture struct {
	name     string
	cfg      *config.MySQL
	serverID uint32

	db     *sql.DB
	syncer *replication.BinlogSyncer
	cancel context.CancelFunc

	mu        sync.Mutex
	changes   []RowChange
	lastEvent time.Time
	err       error

	colCache map[string][]string
}

func New(name string, cfg *config.MySQL, serverID uint32) *Capture {
	return &Capture{name: name, cfg: cfg, serverID: serverID, colCache: map[string][]string{}}
}

func (c *Capture) Start() error {
	db, err := sql.Open("mysql", c.cfg.DSN())
	if err != nil {
		return err
	}
	c.db = db

	file, pos, err := c.currentPosition()
	if err != nil {
		return fmt.Errorf("[%s] %w", c.name, err)
	}

	c.syncer = replication.NewBinlogSyncer(replication.BinlogSyncerConfig{
		ServerID:   c.serverID,
		Flavor:     "mysql",
		Host:       c.cfg.Host,
		Port:       uint16(c.cfg.Port),
		User:       c.cfg.User,
		Password:   c.cfg.Password,
		UseDecimal: true, // exact DECIMAL values instead of float64
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	streamer, err := c.syncer.StartSync(mysql.Position{Name: file, Pos: pos})
	if err != nil {
		return fmt.Errorf("[%s] start binlog sync: %w", c.name, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.pump(ctx, streamer)
	log.Printf("[%s] binlog capture started at %s:%d", c.name, file, pos)
	return nil
}

func (c *Capture) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.syncer != nil {
		c.syncer.Close()
	}
	if c.db != nil {
		c.db.Close()
	}
}

func (c *Capture) currentPosition() (string, uint32, error) {
	// MySQL <= 8.0 uses SHOW MASTER STATUS; >= 8.4 renamed it.
	for _, stmt := range []string{"SHOW MASTER STATUS", "SHOW BINARY LOG STATUS"} {
		file, pos, err := c.tryPosition(stmt)
		if err == nil {
			return file, pos, nil
		}
	}
	return "", 0, fmt.Errorf("read binlog position on %s (is binary logging enabled?)", c.cfg.Addr())
}

func (c *Capture) tryPosition(stmt string) (string, uint32, error) {
	rows, err := c.db.Query(stmt)
	if err != nil {
		return "", 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", 0, fmt.Errorf("no binlog status row")
	}
	cols, err := rows.Columns()
	if err != nil || len(cols) < 2 {
		return "", 0, fmt.Errorf("unexpected binlog status shape")
	}
	var file string
	var pos uint32
	vals := make([]any, len(cols))
	vals[0], vals[1] = &file, &pos
	for i := 2; i < len(cols); i++ {
		vals[i] = &sql.RawBytes{}
	}
	if err := rows.Scan(vals...); err != nil {
		return "", 0, err
	}
	return file, pos, nil
}

// -- pump goroutine ---------------------------------------------------------

func (c *Capture) pump(ctx context.Context, streamer *replication.BinlogStreamer) {
	var lastQuery string
	for {
		ev, err := streamer.GetEvent(ctx)
		if err != nil {
			if ctx.Err() == nil {
				c.mu.Lock()
				c.err = err
				c.mu.Unlock()
				log.Printf("[%s] binlog stream died: %v", c.name, err)
			}
			return
		}
		switch e := ev.Event.(type) {
		case *replication.RowsQueryEvent:
			lastQuery = string(e.Query)
		case *replication.RowsEvent:
			schema := string(e.Table.Schema)
			if systemSchemas[schema] {
				continue
			}
			changes := c.normalize(ev.Header.EventType, e, schema, lastQuery)
			if len(changes) == 0 {
				continue
			}
			c.mu.Lock()
			c.changes = append(c.changes, changes...)
			c.lastEvent = time.Now()
			c.mu.Unlock()
		}
	}
}

func opOf(t replication.EventType) string {
	switch t {
	case replication.WRITE_ROWS_EVENTv0, replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
		return "INSERT"
	case replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
		return "UPDATE"
	case replication.DELETE_ROWS_EVENTv0, replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
		return "DELETE"
	}
	return ""
}

func (c *Capture) normalize(t replication.EventType, e *replication.RowsEvent, schema, lastQuery string) []RowChange {
	op := opOf(t)
	if op == "" {
		return nil
	}
	table := schema + "." + string(e.Table.Table)
	cols := c.columnNames(e, schema, string(e.Table.Table))

	rowMap := func(row []any) map[string]any {
		m := make(map[string]any, len(row))
		for i, v := range row {
			name := fmt.Sprintf("col_%d", i)
			if i < len(cols) && cols[i] != "" {
				name = cols[i]
			}
			m[name] = v
		}
		return m
	}

	var out []RowChange
	if op == "UPDATE" {
		// rows arrive as before/after pairs
		for i := 0; i+1 < len(e.Rows); i += 2 {
			out = append(out, RowChange{
				Table: table, Op: op,
				Before: rowMap(e.Rows[i]), After: rowMap(e.Rows[i+1]),
				Query: lastQuery,
			})
		}
		return out
	}
	for _, row := range e.Rows {
		ch := RowChange{Table: table, Op: op, Query: lastQuery}
		if op == "INSERT" {
			ch.After = rowMap(row)
		} else {
			ch.Before = rowMap(row)
		}
		out = append(out, ch)
	}
	return out
}

// columnNames prefers binlog_row_metadata=FULL table-map metadata and falls
// back to information_schema (cached).
func (c *Capture) columnNames(e *replication.RowsEvent, schema, table string) []string {
	if len(e.Table.ColumnName) > 0 {
		names := make([]string, len(e.Table.ColumnName))
		for i, b := range e.Table.ColumnName {
			names[i] = string(b)
		}
		return names
	}
	key := schema + "." + table
	if names, ok := c.colCache[key]; ok {
		return names
	}
	rows, err := c.db.Query(
		`SELECT COLUMN_NAME FROM information_schema.COLUMNS
		 WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION`,
		schema, table,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			names = append(names, n)
		}
	}
	c.colCache[key] = names
	return names
}

// -- attribution window (Level 0) --------------------------------------------

func (c *Capture) BeginWindow() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n := len(c.changes); n > 0 {
		log.Printf("[%s] warning: %d row change(s) arrived between windows (async leak?) — attributed to the next request", c.name, n)
	}
	c.lastEvent = time.Time{}
}

// TakeWindow waits for quiesce, then returns (and clears) this window's changes.
func (c *Capture) TakeWindow(attr config.Attribution) ([]RowChange, error) {
	quiet := time.Duration(attr.QuietMs) * time.Millisecond
	deadline := time.Now().Add(time.Duration(attr.TimeoutMs) * time.Millisecond)
	opened := time.Now()

	for {
		c.mu.Lock()
		err := c.err
		last := c.lastEvent
		c.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("[%s] binlog capture failed: %w", c.name, err)
		}
		if last.IsZero() {
			last = opened
		}
		if time.Since(last) >= quiet && (!attr.TrxCheck() || !c.hasActiveTrx()) {
			break
		}
		if time.Now().After(deadline) {
			log.Printf("[%s] quiesce timeout after %dms — window may be truncated", c.name, attr.TimeoutMs)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	changes := c.changes
	c.changes = nil
	return changes, nil
}

func (c *Capture) hasActiveTrx() bool {
	var count int
	err := c.db.QueryRow("SELECT COUNT(*) FROM information_schema.innodb_trx").Scan(&count)
	if err != nil {
		if !strings.Contains(err.Error(), "denied") {
			log.Printf("[%s] innodb_trx check failed: %v", c.name, err)
		}
		return false
	}
	return count > 0
}
