package ui

import (
	"encoding/json"
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

// seedStart seeds every new datastore from its configured source. The user
// gives DB connection info only (in Settings) — no CLI, no AWS.
func (s *Server) seedStart(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "config: "+err.Error())
		return
	}
	if len(cfg.New.Datastores) == 0 {
		jsonError(w, http.StatusBadRequest, "검증 대상 datastore가 없습니다 — 설정에서 추가하세요")
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

	go s.runSeed(cfg)
	writeJSON(w, map[string]bool{"started": true})
}

func (s *Server) runSeed(cfg *config.Config) {
	stats, err := seed.RunDatastores(cfg.New.Datastores, "")
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

// seedDiscover connects to a source DB (from the request body) and returns its
// schemas+tables so the UI can show checkboxes with sizes.
func (s *Server) seedDiscover(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Host, User, Password string
		Port                 int
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Host == "" {
		jsonError(w, http.StatusBadRequest, "host is required")
		return
	}
	if req.Port == 0 {
		req.Port = 3306
	}
	src := &config.MySQL{Host: req.Host, Port: req.Port, User: req.User, Password: req.Password}
	schemas, err := seed.Discover(src)
	if err != nil {
		jsonError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, schemas)
}
