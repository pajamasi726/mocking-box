package ui

import (
	"encoding/json"
	"net/http"

	"github.com/pajamasi726/mocking-box/internal/config"
	"github.com/pajamasi726/mocking-box/internal/pg"
)

// testConnection checks a datastore connection (read-only) without saving it to
// config — used by the MCP/AI flow to validate credentials before use.
func (s *Server) testConnection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type, Host, User, Password, Database string
		Port                                 int
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Host == "" {
		jsonError(w, http.StatusBadRequest, "host required")
		return
	}
	if req.Type == "postgres" {
		if req.Port == 0 {
			req.Port = 5432
		}
		d := &config.Datastore{Type: "postgres", Host: req.Host, Port: req.Port,
			User: req.User, Password: req.Password, Database: req.Database}
		ok, detail := pg.Health(d)
		writeJSON(w, healthItem{ok, detail})
		return
	}
	if req.Port == 0 {
		req.Port = 3306
	}
	writeJSON(w, checkMySQL(&config.MySQL{Host: req.Host, Port: req.Port, User: req.User, Password: req.Password}))
}
