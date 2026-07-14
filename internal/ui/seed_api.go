package ui

import (
	"log"
	"net/http"

	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/seed"
)

func (s *Server) seedStatusHandler(w http.ResponseWriter, r *http.Request) {
	s.seedMu.Lock()
	defer s.seedMu.Unlock()
	writeJSON(w, s.seedST)
}

// seedStart runs the seed engine using config's seed_source (or an override in
// the request body). The user gives DB connection info only — no CLI, no AWS.
func (s *Server) seedStart(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "config: "+err.Error())
		return
	}
	if cfg.New.MySQL == nil {
		jsonError(w, http.StatusBadRequest, "검증 대상 DB(new.mysql)가 설정되어 있어야 합니다")
		return
	}
	src := cfg.SeedSource
	if src == nil || src.Host == "" {
		jsonError(w, http.StatusBadRequest, "시딩 소스 DB 정보가 없습니다 — 설정에서 입력하세요")
		return
	}

	s.seedMu.Lock()
	if s.seedST.Running {
		s.seedMu.Unlock()
		jsonError(w, http.StatusConflict, "이미 시딩 중입니다")
		return
	}
	s.seedST = seedStatus{Running: true, Log: "시딩 시작…"}
	s.seedMu.Unlock()

	go s.runSeed(cfg, src)
	writeJSON(w, map[string]bool{"started": true})
}

func (s *Server) runSeed(cfg *config.Config, src *config.SeedSource) {
	port := src.Port
	if port == 0 {
		port = 3306
	}
	srcDB := &config.MySQL{Host: src.Host, Port: port, User: src.User, Password: src.Password}
	opts := seed.Options{Schemas: src.Schemas, ExcludeTables: map[string]bool{}}
	for _, t := range src.Exclude {
		if t != "" {
			opts.ExcludeTables[t] = true
		}
	}

	stats, err := seed.Run(srcDB, cfg.New.MySQL, opts)
	s.seedMu.Lock()
	defer s.seedMu.Unlock()
	s.seedST.Running = false
	if err != nil {
		s.seedST.Error = err.Error()
		s.seedST.Log = "시딩 실패"
		log.Printf("[seed] %v", err)
		return
	}
	s.seedST.Done = true
	s.seedST.Tables, s.seedST.Rows = stats.Tables, stats.Rows
	s.seedST.Log = "시딩 완료"
	log.Printf("[seed] done: schemas=%v tables=%d rows=%d", stats.Schemas, stats.Tables, stats.Rows)
}
