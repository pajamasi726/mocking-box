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

// Stack is one side of the comparison: a running app + the MySQL it writes to.
// MySQL may be nil for response-diff-only mode.
type Stack struct {
	Name    string `yaml:"-"`
	BaseURL string `yaml:"base_url"`
	MySQL   *MySQL `yaml:"mysql"`
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

type Config struct {
	Old            Stack       `yaml:"old"`
	New            Stack       `yaml:"new"`
	Attribution    Attribution `yaml:"attribution"`
	Noise          Noise       `yaml:"noise"`
	HTTPTimeoutS   float64     `yaml:"http_timeout_s"`
	CompareHeaders []string    `yaml:"compare_headers"`
	Report         Report      `yaml:"report"`
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
	if cfg.Old.BaseURL == "" || cfg.New.BaseURL == "" {
		return nil, fmt.Errorf("both old.base_url and new.base_url are required")
	}
	for _, m := range []*MySQL{cfg.Old.MySQL, cfg.New.MySQL} {
		if m != nil && m.Port == 0 {
			m.Port = 3306
		}
	}
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
	for i, h := range cfg.CompareHeaders {
		cfg.CompareHeaders[i] = strings.ToLower(h)
	}
	return cfg, nil
}
