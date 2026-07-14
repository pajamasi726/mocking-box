// Package config loads the mocking-box YAML configuration.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type MySQL struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func (m *MySQL) Addr() string { return fmt.Sprintf("%s:%d", m.Host, m.Port) }

// DSN for database/sql (information_schema checks, position lookup).
func (m *MySQL) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/", m.User, m.Password, m.Host, m.Port)
}

// Datastore is one database a stack reads/writes. A stack may own several
// (legalcare: a lawkit MySQL + a medi PostgreSQL). Old and new are paired by
// Name: the datastore called "lawkit-db" on old is compared with "lawkit-db"
// on new. SeedFrom is the source (dev/replica/restore) this datastore is
// seeded from; Schemas/Exclude scope what gets copied and captured.
type Datastore struct {
	Name     string   `yaml:"name" json:"name"`
	Type     string   `yaml:"type" json:"type"` // "mysql" (postgres: roadmap)
	Host     string   `yaml:"host" json:"host"`
	Port     int      `yaml:"port" json:"port"`
	User     string   `yaml:"user" json:"user"`
	Password string   `yaml:"password" json:"password"`
	Schemas  []string `yaml:"schemas" json:"schemas"`
	Exclude  []string `yaml:"exclude_tables" json:"exclude_tables"`
	SeedFrom *MySQL   `yaml:"seed_from" json:"seed_from,omitempty"`
}

func (d *Datastore) Addr() string { return fmt.Sprintf("%s:%d", d.Host, d.Port) }

// MySQL returns a MySQL handle for datastores of type mysql (nil otherwise).
func (d *Datastore) MySQL() *MySQL {
	if d == nil || (d.Type != "" && d.Type != "mysql") {
		return nil
	}
	return &MySQL{Host: d.Host, Port: d.Port, User: d.User, Password: d.Password}
}

// Stack is one side of the comparison: a running app + the datastores it writes.
type Stack struct {
	Name       string       `yaml:"-"`
	BaseURL    string       `yaml:"base_url"`
	MySQL      *MySQL       `yaml:"mysql"` // legacy single-DB shorthand (promoted to Datastores on load)
	Datastores []*Datastore `yaml:"datastores"`
}

// PrimaryMySQL returns the first mysql datastore (single-DB write-set path).
func (s *Stack) PrimaryMySQL() *MySQL {
	for _, d := range s.Datastores {
		if m := d.MySQL(); m != nil {
			return m
		}
	}
	return nil
}

func (s *Stack) DatastoreByName(name string) *Datastore {
	for _, d := range s.Datastores {
		if d.Name == name {
			return d
		}
	}
	return nil
}

type Attribution struct {
	QuietMs        int   `yaml:"quiet_ms"`
	TimeoutMs      int   `yaml:"timeout_ms"`
	CheckInnodbTrx *bool `yaml:"check_innodb_trx"`
}

func (a Attribution) TrxCheck() bool { return a.CheckInnodbTrx == nil || *a.CheckInnodbTrx }

type Noise struct {
	ResponsePaths []string `yaml:"response_paths"`
	Columns       []string `yaml:"columns"`
	TablesIgnore  []string `yaml:"tables_ignore"`
}

type Report struct {
	Dir string `yaml:"dir"`
}

type Corpus struct {
	Dir string `yaml:"dir"`
}

// SeedSource is the legacy top-level `seed_source:` shorthand — folded into the
// first `new` datastore's SeedFrom on load. New configs use per-datastore seed_from.
type SeedSource struct {
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	User     string   `yaml:"user"`
	Password string   `yaml:"password"`
	Schemas  []string `yaml:"schemas"`
	Exclude  []string `yaml:"exclude_tables"`
}

// SortRule: sort matching arrays before response comparison (mapper-dependent
// ordering). By is an element key, or "$canonical" for whole-element sort.
type SortRule struct {
	Path string `yaml:"path" json:"path"`
	By   string `yaml:"by" json:"by"`
}

type Compare struct {
	SortArrays []SortRule `yaml:"sort_arrays" json:"sort_arrays"`
}

type Config struct {
	Old            Stack       `yaml:"old"`
	New            Stack       `yaml:"new"`
	Attribution    Attribution `yaml:"attribution"`
	Noise          Noise       `yaml:"noise"`
	Compare        Compare     `yaml:"compare"`
	HTTPTimeoutS   float64     `yaml:"http_timeout_s"`
	CompareHeaders []string    `yaml:"compare_headers"`
	Report         Report      `yaml:"report"`
	Corpus         Corpus      `yaml:"corpus"`
	SeedSource     *SeedSource `yaml:"seed_source"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	cfg.Old.Name, cfg.New.Name = "old", "new"
	cfg.Old.BaseURL = strings.TrimRight(cfg.Old.BaseURL, "/")
	cfg.New.BaseURL = strings.TrimRight(cfg.New.BaseURL, "/")
	// new = 검증 대상(필수). old = 선택 — 라이브 비교(run)와 기준선 셀프체크에만 필요.
	// 사이드카 수집 세계에서 대시보드는 구서버에 접속할 일이 없다.
	if cfg.New.BaseURL == "" {
		return nil, fmt.Errorf("new.base_url is required (the stack under test)")
	}
	// legacy shorthand: `mysql:` (+ top-level seed_source) → a single datastore
	promoteLegacy(&cfg.Old, nil)
	promoteLegacy(&cfg.New, cfg.SeedSource)
	normalizeDatastores(cfg.Old.Datastores)
	normalizeDatastores(cfg.New.Datastores)
	if cfg.Attribution.QuietMs == 0 {
		cfg.Attribution.QuietMs = 300
	}
	if cfg.Attribution.TimeoutMs == 0 {
		cfg.Attribution.TimeoutMs = 5000
	}
	if cfg.HTTPTimeoutS == 0 {
		cfg.HTTPTimeoutS = 10
	}
	if cfg.Report.Dir == "" {
		cfg.Report.Dir = "./report"
	}
	if cfg.Corpus.Dir == "" {
		cfg.Corpus.Dir = "./corpus"
	}
	for i, h := range cfg.CompareHeaders {
		cfg.CompareHeaders[i] = strings.ToLower(h)
	}
	return cfg, nil
}

// promoteLegacy folds the old single-`mysql:` shorthand (and a top-level
// seed_source, for the `new` stack) into one datastore named "default", so
// existing configs keep working under the datastore model.
func promoteLegacy(stack *Stack, seedSource *SeedSource) {
	if stack.MySQL == nil || len(stack.Datastores) > 0 {
		return
	}
	d := &Datastore{
		Name: "default", Type: "mysql",
		Host: stack.MySQL.Host, Port: stack.MySQL.Port,
		User: stack.MySQL.User, Password: stack.MySQL.Password,
	}
	if seedSource != nil && seedSource.Host != "" {
		d.Schemas = seedSource.Schemas
		d.Exclude = seedSource.Exclude
		d.SeedFrom = &MySQL{
			Host: seedSource.Host, Port: seedSource.Port,
			User: seedSource.User, Password: seedSource.Password,
		}
	}
	stack.Datastores = []*Datastore{d}
	stack.MySQL = nil
}

func normalizeDatastores(datastores []*Datastore) {
	for i, d := range datastores {
		if d.Type == "" {
			d.Type = "mysql"
		}
		if d.Port == 0 {
			d.Port = 3306
		}
		if d.Name == "" {
			d.Name = fmt.Sprintf("db%d", i+1)
		}
		if d.SeedFrom != nil && d.SeedFrom.Port == 0 {
			d.SeedFrom.Port = 3306
		}
	}
}
