package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"peopleops/internal/auth"
)

func (s *Server) bonuses(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !(s.hasPermission(r.Context(), u.Role, "payroll.manage") || s.hasPermission(r.Context(), u.Role, "payroll.calculate") || strings.EqualFold(u.Role, "Master Admin") || strings.EqualFold(u.Role, "HR Admin") || strings.EqualFold(u.Role, "Accounts")) {
		writeError(w, http.StatusForbidden, "forbidden", "You do not have permission to manage bonus payments")
		return
	}
	if r.Method == http.MethodGet {
		rows, err := s.Store.DB.Query(r.Context(), `SELECT h.id,h.employee_id,e.first_name||' '||e.last_name,h.bonus_pay_minor,h.bonus_month,h.updated_by,h.created_at FROM dcs_bonus_history h JOIN employees e ON e.id=h.employee_id ORDER BY h.created_at DESC LIMIT 500`)
		if err != nil {
			log.Println("bonus_history_query_failed:", err)
			writeError(w, 500, "database_error", "Could not load bonus history")
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, eid, name, by string
			var amount int64
			var month int
			var created any
			if err := rows.Scan(&id, &eid, &name, &amount, &month, &by, &created); err != nil {
				log.Println("bonus_history_scan_failed:", err)
				writeError(w, 500, "database_error", "Could not read bonus history")
				return
			}
			items = append(items, map[string]any{"id": id, "employeeId": eid, "employeeName": name, "bonusPayMinor": amount, "bonusMonth": month, "updatedBy": by, "createdAt": created})
		}
		if err := rows.Err(); err != nil {
			log.Println("bonus_history_iteration_failed:", err)
			writeError(w, 500, "database_error", "Could not read bonus history")
			return
		}
		writeJSON(w, 200, map[string]any{"history": items})
		return
	}
	var req struct {
		EmployeeIDs   []string `json:"employeeIds"`
		BonusPayMinor int64    `json:"bonusPayMinor"`
		BonusMonth    int      `json:"bonusMonth"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		log.Println("bonus_request_decode_failed:", err)
		writeError(w, 400, "invalid_request", "Invalid bonus request")
		return
	}
	if len(req.EmployeeIDs) == 0 || req.BonusPayMinor < 0 || req.BonusMonth < 1 || req.BonusMonth > 12 {
		writeError(w, 400, "validation_error", "Select at least one employee and provide a valid amount and month")
		return
	}
	tx, err := s.Store.DB.Begin(r.Context())
	if err != nil {
		log.Println("bonus_transaction_begin_failed:", err)
		writeError(w, 500, "database_error", "Could not start bonus update")
		return
	}
	defer tx.Rollback(r.Context())
	seen := map[string]bool{}
	for _, eid := range req.EmployeeIDs {
		eid = strings.TrimSpace(eid)
		if eid == "" || seen[eid] {
			continue
		}
		seen[eid] = true
		var exists bool
		if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1)`, eid).Scan(&exists); err != nil {
			log.Println("bonus_employee_check_failed:", err)
			writeError(w, 500, "database_error", "Could not validate selected employees")
			return
		}
		if !exists {
			writeError(w, 400, "validation_error", "One or more selected employees do not exist")
			return
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO dsc_employee_salary_breakup(id,employee_id,bonus_pay_minor,bonus_month) VALUES($1,$2,$3,$4) ON CONFLICT(employee_id) DO UPDATE SET bonus_pay_minor=EXCLUDED.bonus_pay_minor,bonus_month=EXCLUDED.bonus_month,updated_at=now()`, "bonus-"+auth.NewID(), eid, req.BonusPayMinor, req.BonusMonth)
		if err != nil {
			log.Println("bonus_salary_breakup_update_failed:", err)
			writeError(w, 500, "database_error", "Could not update employee bonus")
			return
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO dcs_bonus_history(id,employee_id,bonus_pay_minor,bonus_month,updated_by) VALUES($1,$2,$3,$4,$5)`, "bonus-history-"+auth.NewID(), eid, req.BonusPayMinor, req.BonusMonth, u.Name)
		if err != nil {
			log.Println("bonus_history_insert_failed:", err)
			writeError(w, 500, "database_error", "Could not record bonus history")
			return
		}
	}
	if len(seen) == 0 {
		writeError(w, 400, "validation_error", "No valid employees were selected")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		log.Println("bonus_transaction_commit_failed:", err)
		writeError(w, 500, "database_error", "Could not save bonuses")
		return
	}
	writeJSON(w, 200, map[string]any{"message": "Bonus saved", "employeesUpdated": len(seen)})
}
