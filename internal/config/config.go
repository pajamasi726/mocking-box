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

type Corpus struct {
	Dir string `yaml:"dir"`
}

// SeedSource: 검증 DB를 채울 원본 (dev DB / read replica / 복원 인스턴스).
// 접속만 하면 되는 이미 존재하는 DB — 대시보드 [시딩] 버튼이 사용.
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
	if cfg.Corpus.Dir == "" {
		cfg.Corpus.Dir = "./corpus"
	}
	for i, h := range cfg.CompareHeaders {
		cfg.CompareHeaders[i] = strings.ToLower(h)
	}
	return cfg, nil
}
