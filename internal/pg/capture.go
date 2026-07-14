package pg

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/writeset"
)

// Capture streams per-request write-sets from PostgreSQL via logical decoding
// (pgoutput). It creates a temporary publication + slot on Start and drops
// them on Stop. Implements writeset.Source, mirroring the MySQL binlog Capture.
type Capture struct {
	name string
	d    *config.Datastore

	slot string
	pub  string

	conn   *pgconn.PgConn
	admin  *pgconn.PgConn
	cancel context.CancelFunc

	relations map[uint32]*pglogrepl.RelationMessageV2
	typeMap   *pgtype.Map

	mu            sync.Mutex
	changes       []writeset.RowChange
	lastEvent     time.Time
	lastCommitLSN pglogrepl.LSN
	windowStart   pglogrepl.LSN // lastCommitLSN when the current window opened
	err           error
}

func NewCapture(name string, d *config.Datastore) *Capture {
	safe := ""
	for _, r := range name {
		if r == '-' || r == ' ' {
			safe += "_"
		} else {
			safe += string(r)
		}
	}
	return &Capture{
		name:      name,
		d:         d,
		slot:      "mb_slot_" + safe,
		pub:       "mb_pub_" + safe,
		relations: map[uint32]*pglogrepl.RelationMessageV2{},
		typeMap:   pgtype.NewMap(),
	}
}

func (c *Capture) Start() error {
	ctx := context.Background()
	dbname := databaseName(c.d)

	// admin conn (plain) for publication + slot lifecycle
	admin, err := pgconn.Connect(ctx, connString(c.d, dbname))
	if err != nil {
		return fmt.Errorf("[%s] pg admin connect: %w", c.name, err)
	}
	c.admin = admin
	_ = admin.Exec(ctx, fmt.Sprintf("DROP PUBLICATION IF EXISTS %s", c.pub)).Close()
	if err := admin.Exec(ctx, fmt.Sprintf("CREATE PUBLICATION %s FOR ALL TABLES", c.pub)).Close(); err != nil {
		return fmt.Errorf("[%s] create publication: %w", c.name, err)
	}
	// REPLICA IDENTITY FULL on captured tables → UPDATE/DELETE carry full
	// before-images (else only the PK). Metadata-only change; scoped to the
	// datastore's schemas so we don't touch the whole DB.
	c.setReplicaIdentityFull(ctx, admin)

	// replication conn
	repl, err := pgconn.Connect(ctx, connString(c.d, dbname)+" replication=database")
	if err != nil {
		return fmt.Errorf("[%s] pg replication connect: %w", c.name, err)
	}
	c.conn = repl

	// drop a stale slot, then create fresh (temporary slots vanish on disconnect)
	_ = pglogrepl.DropReplicationSlot(ctx, repl, c.slot, pglogrepl.DropReplicationSlotOptions{})
	if _, err := pglogrepl.CreateReplicationSlot(ctx, repl, c.slot, "pgoutput",
		pglogrepl.CreateReplicationSlotOptions{Temporary: true}); err != nil {
		return fmt.Errorf("[%s] create slot: %w", c.name, err)
	}

	sysident, err := pglogrepl.IdentifySystem(ctx, repl)
	if err != nil {
		return fmt.Errorf("[%s] identify system: %w", c.name, err)
	}
	pluginArgs := []string{
		"proto_version '2'",
		fmt.Sprintf("publication_names '%s'", c.pub),
		"messages 'true'",
	}
	if err := pglogrepl.StartReplication(ctx, repl, c.slot, sysident.XLogPos,
		pglogrepl.StartReplicationOptions{PluginArgs: pluginArgs}); err != nil {
		return fmt.Errorf("[%s] start replication: %w", c.name, err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.pump(runCtx, sysident.XLogPos)
	log.Printf("[%s] pg logical decoding started (slot=%s)", c.name, c.slot)
	return nil
}

func (c *Capture) setReplicaIdentityFull(ctx context.Context, admin *pgconn.PgConn) {
	if len(c.d.Schemas) == 0 {
		return // no schema scope — leave replica identity as-is
	}
	for _, schema := range c.d.Schemas {
		res := admin.Exec(ctx, fmt.Sprintf(
			"SELECT quote_ident(schemaname)||'.'||quote_ident(tablename) FROM pg_tables WHERE schemaname = %s",
			quoteLiteral(schema)))
		results, err := res.ReadAll()
		if err != nil || len(results) == 0 {
			continue
		}
		for _, row := range results[0].Rows {
			qtable := string(row[0])
			_ = admin.Exec(ctx, "ALTER TABLE "+qtable+" REPLICA IDENTITY FULL").Close()
		}
	}
}

func quoteLiteral(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func (c *Capture) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		c.conn.Close(context.Background())
	}
	if c.admin != nil {
		ctx := context.Background()
		_ = c.admin.Exec(ctx, fmt.Sprintf("DROP PUBLICATION IF EXISTS %s", c.pub)).Close()
		c.admin.Close(ctx)
	}
}

func (c *Capture) pump(ctx context.Context, clientXLogPos pglogrepl.LSN) {
	nextDeadline := time.Now().Add(10 * time.Second)
	for {
		if ctx.Err() != nil {
			return
		}
		if time.Now().After(nextDeadline) {
			_ = pglogrepl.SendStandbyStatusUpdate(ctx, c.conn,
				pglogrepl.StandbyStatusUpdate{WALWritePosition: clientXLogPos})
			nextDeadline = time.Now().Add(10 * time.Second)
		}
		recvCtx, cancel := context.WithDeadline(ctx, nextDeadline)
		raw, err := c.conn.ReceiveMessage(recvCtx)
		cancel()
		if err != nil {
			if pgconn.Timeout(err) || ctx.Err() != nil {
				continue
			}
			c.setErr(err)
			return
		}
		cd, ok := raw.(*pgproto3.CopyData)
		if !ok {
			continue
		}
		switch cd.Data[0] {
		case pglogrepl.PrimaryKeepaliveMessageByteID:
			pk, err := pglogrepl.ParsePrimaryKeepaliveMessage(cd.Data[1:])
			if err == nil && pk.ReplyRequested {
				nextDeadline = time.Now()
			}
		case pglogrepl.XLogDataByteID:
			xld, err := pglogrepl.ParseXLogData(cd.Data[1:])
			if err != nil {
				continue
			}
			clientXLogPos = xld.WALStart + pglogrepl.LSN(len(xld.WALData))
			c.handleWAL(xld.WALData)
		}
	}
}

func (c *Capture) handleWAL(data []byte) {
	msg, err := pglogrepl.ParseV2(data, false)
	if err != nil {
		return
	}
	switch m := msg.(type) {
	case *pglogrepl.RelationMessageV2:
		c.relations[m.RelationID] = m
	case *pglogrepl.InsertMessageV2:
		c.record(m.RelationID, "INSERT", nil, m.Tuple)
	case *pglogrepl.UpdateMessageV2:
		c.record(m.RelationID, "UPDATE", m.OldTuple, m.NewTuple)
	case *pglogrepl.DeleteMessageV2:
		c.record(m.RelationID, "DELETE", m.OldTuple, nil)
	case *pglogrepl.CommitMessage:
		c.mu.Lock()
		c.lastCommitLSN = m.CommitLSN
		c.lastEvent = time.Now()
		c.mu.Unlock()
	}
}

func (c *Capture) record(relID uint32, op string, oldTup, newTup *pglogrepl.TupleData) {
	rel := c.relations[relID]
	if rel == nil {
		return
	}
	table := rel.Namespace + "." + rel.RelationName
	ch := writeset.RowChange{Table: table, Op: op}
	if oldTup != nil {
		ch.Before = c.decodeTuple(rel, oldTup)
	}
	if newTup != nil {
		ch.After = c.decodeTuple(rel, newTup)
	}
	c.mu.Lock()
	c.changes = append(c.changes, ch)
	c.lastEvent = time.Now()
	c.mu.Unlock()
}

func (c *Capture) decodeTuple(rel *pglogrepl.RelationMessageV2, tup *pglogrepl.TupleData) map[string]any {
	out := map[string]any{}
	for i, col := range tup.Columns {
		if i >= len(rel.Columns) {
			break
		}
		name := rel.Columns[i].Name
		switch col.DataType {
		case 'n': // null
			out[name] = nil
		case 't': // text
			out[name] = decodeText(c.typeMap, rel.Columns[i].DataType, col.Data)
		default: // 'u' unchanged toast — omit
		}
	}
	return out
}

func decodeText(m *pgtype.Map, oid uint32, data []byte) any {
	if dt, ok := m.TypeForOID(oid); ok {
		if v, err := dt.Codec.DecodeValue(m, oid, pgtype.TextFormatCode, data); err == nil {
			return v
		}
	}
	return string(data)
}

func (c *Capture) setErr(err error) {
	c.mu.Lock()
	c.err = err
	c.mu.Unlock()
	log.Printf("[%s] pg logical stream died: %v", c.name, err)
}

// -- attribution window (Level 0) --------------------------------------------

func (c *Capture) BeginWindow() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changes = nil
	c.windowStart = c.lastCommitLSN
	c.lastEvent = time.Time{}
}

// TakeWindow returns the request's write-set. Because the proxy serializes
// requests, each request maps to one transaction (one commit). We wait until
// the logical stream decodes a commit past the window's start LSN — at which
// point all of that transaction's row changes are already buffered. Read-only
// requests emit no commit, so they fall through the quiesce timeout quickly.
func (c *Capture) TakeWindow(attr config.Attribution) ([]writeset.RowChange, error) {
	deadline := time.Now().Add(time.Duration(attr.TimeoutMs) * time.Millisecond)
	quiet := time.Duration(attr.QuietMs) * time.Millisecond
	opened := time.Now()

	for {
		c.mu.Lock()
		err, committed, start, nChanges := c.err, c.lastCommitLSN, c.windowStart, len(c.changes)
		c.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("[%s] pg capture failed: %w", c.name, err)
		}
		if committed > start {
			break // this request's transaction has been fully decoded
		}
		if nChanges == 0 && time.Since(opened) >= quiet {
			break // read-only request: no transaction to wait for
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	changes := c.changes
	c.changes = nil
	return changes, nil
}
