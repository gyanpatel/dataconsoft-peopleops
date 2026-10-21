package api

import (
	"log"
	"net/http"
	"strings"
	"time"
)

func (s *Server) appSettings(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if r.Method != http.MethodGet && !s.hasPermission(r.Context(), u.Role, "app.configure") {
		writeError(w, 403, "forbidden", "Only Master Admin can change global application settings")
		return
	}
	if r.Method == http.MethodGet {
		var currency, zone, symbol string
		err := s.Store.DB.QueryRow(r.Context(), `SELECT COALESCE(MAX(setting_value) FILTER (WHERE setting_key='currency'),'GBP'), COALESCE(MAX(setting_symbol) FILTER (WHERE setting_key='currency'),'Dh.'),COALESCE(MAX(setting_value) FILTER (WHERE setting_key='timezone'),'Asia/Dubai') FROM dcs_app_settings`).Scan(&currency, &symbol, &zone)
		if err != nil {
			writeError(w, 500, "database_error", "Could not load application settings")
			return
		}
		writeJSON(w, 200, map[string]string{"currency": strings.TrimSpace(currency), "timezone": strings.TrimSpace(zone), "cursymbol": strings.TrimSpace(symbol)})
		return
	}
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		Currency string `json:"currency"`
		Timezone string `json:"timezone"`
	}
	if decode(r, &in) != nil {
		writeError(w, 400, "invalid_request", "Invalid application settings")
		return
	}
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.Timezone = strings.TrimSpace(in.Timezone)
	if len(in.Currency) != 3 || in.Currency[0] < 'A' || in.Currency[0] > 'Z' || in.Currency[1] < 'A' || in.Currency[1] > 'Z' || in.Currency[2] < 'A' || in.Currency[2] > 'Z' {
		writeError(w, 400, "validation_error", "Currency must be a valid three-letter ISO code")
		return
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		log.Println("validation_error", "Timezone must be a valid IANA time-zone name, such as Asia/Dubai", err, in.Timezone)
		writeError(w, 400, "validation_error", "Timezone must be a valid IANA time-zone name, such as Asia/Dubai")
		return
	}
	tx, err := s.Store.DB.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "database_error", "Could not save application settings")
		return
	}
	defer tx.Rollback(r.Context())
	for k, v := range map[string]string{"currency": in.Currency, "timezone": in.Timezone} {
		if _, err = tx.Exec(r.Context(), `INSERT INTO dcs_app_settings(setting_key,setting_value,updated_by,updated_at) VALUES($1,$2,$3,now()) ON CONFLICT(setting_key) DO UPDATE SET setting_value=EXCLUDED.setting_value,updated_by=EXCLUDED.updated_by,updated_at=now()`, k, v, u.ID); err != nil {
			writeError(w, 500, "database_error", "Could not save application settings")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "database_error", "Could not save application settings")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "settings.update", "settings", "global", "Updated global currency and time zone", requestID(r))
	writeJSON(w, 200, map[string]string{"currency": in.Currency, "timezone": in.Timezone, "message": "Global application settings saved. Restart the API service for server-side time-zone changes to take effect."})
}
