// Package wscapture picks a write-set Source implementation by datastore type:
// MySQL (binlog) or PostgreSQL (logical decoding).
package wscapture

import (
	"fmt"

	"github.com/pajamasi726/mocking-box/internal/binlog"
	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/pg"
	"github.com/pajamasi726/mocking-box/internal/writeset"
)

// For returns a write-set Source for the first capturable datastore of the
// stack, or nil if none is configured. serverID uniquely identifies the MySQL
// replication client; PostgreSQL derives a slot name from name.
func For(name string, stack *config.Stack, serverID uint32) (writeset.Source, error) {
	for _, d := range stack.Datastores {
		switch d.Type {
		case "", "mysql":
			return binlog.New(name, d.MySQL(), serverID), nil
		case "postgres":
			return pg.NewCapture(name, d), nil
		default:
			return nil, fmt.Errorf("datastore %q: unsupported type %q", d.Name, d.Type)
		}
	}
	return nil, nil
}
