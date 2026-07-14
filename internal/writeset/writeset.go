// Package writeset defines the engine-neutral write-set model and the Source
// interface. MySQL (binlog) and PostgreSQL (logical decoding) both implement
// Source, so replay/verify capture per-request row changes the same way
// regardless of database.
package writeset

import "github.com/pajamasi726/mocking-box/internal/config"

// RowChange is one committed row change, normalized from a DB change stream.
type RowChange struct {
	Table  string         `json:"table"` // schema.table
	Op     string         `json:"op"`    // INSERT | UPDATE | DELETE
	Before map[string]any `json:"before,omitempty"`
	After  map[string]any `json:"after,omitempty"`
	Query  string         `json:"query,omitempty"` // originating SQL when the stream carries it
}

// Source streams committed row changes and attributes them to requests via
// an open/quiesce window (Level 0 attribution).
type Source interface {
	Start() error
	Stop()
	BeginWindow()
	TakeWindow(attr config.Attribution) ([]RowChange, error)
}
