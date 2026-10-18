package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func (s *Server) leaveBalances(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, e := time.Parse("2006", y); e == nil {
			year = parsed.Year()
		}
	}
	if r.Method == http.MethodPut || r.Method == http.MethodPost {
		s.saveLeaveSettings(w, r)
		return
	}
	if r.URL.Query().Get("settings") == "true" {
		s.getLeaveSettings(w, r)
		return
	}
	if !s.hasPermission(r.Context(), u.Role, "leave.review") && u.EmployeeID == "" {
		writeError(w, 403, "forbidden", "Employee account is not linked to an employee profile")
		return
	}
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)
	rows, err := s.Store.DB.Query(r.Context(), `SELECT e.id,e.employee_number,e.first_name||' '||e.last_name,COALESCE(u.role,''),e.start_date FROM employees e LEFT JOIN LATERAL (SELECT role FROM users WHERE employee_id=e.id AND active=true ORDER BY id LIMIT 1) u ON true WHERE ($1 OR e.id=$2) AND e.start_date < $3 ORDER BY e.last_name,e.first_name`, s.hasPermission(r.Context(), u.Role, "leave.review"), u.EmployeeID, end)
	if err != nil {
		log.Println("database_error", "Could not load leave balances", err)
		writeError(w, 500, "database_error", "Could not load leave balances")
		return
	}
	defer rows.Close()
	weekends := map[time.Weekday]bool{time.Saturday: true, time.Sunday: true}
	wr, we := s.Store.DB.Query(r.Context(), `SELECT weekday FROM dcs_leave_weekend_days`)
	if we == nil {
		weekends = map[time.Weekday]bool{}
		for wr.Next() {
			var d int
			if wr.Scan(&d) == nil && d >= 0 && d <= 6 {
				weekends[time.Weekday(d)] = true
			}
		}
		wr.Close()
	}
	holidays := map[string]bool{}
	hr, he := s.Store.DB.Query(r.Context(), `SELECT holiday_date::text FROM dcs_leave_holidays WHERE holiday_date >= $1 AND holiday_date < $2`, start, end)
	if he == nil {
		for hr.Next() {
			var d string
			if hr.Scan(&d) == nil {
				holidays[d] = true
			}
		}
		hr.Close()
	}
	items := []map[string]any{}
	for rows.Next() {
		var id, num, name, role string
		var joined time.Time
		if rows.Scan(&id, &num, &name, &role, &joined) != nil {
			continue
		}
		var total float64
		_ = s.Store.DB.QueryRow(r.Context(), `SELECT annual_days FROM dcs_leave_role_entitlements WHERE lower(role_name)=lower($1)`, role).Scan(&total)
		eligibleStart := joined
		if eligibleStart.Before(start) {
			eligibleStart = start
		}
		daysYear := float64(end.Sub(start).Hours() / 24)
		eligibleDays := float64(end.Sub(eligibleStart).Hours() / 24)
		if eligibleDays < 0 {
			eligibleDays = 0
		}
		entitlement := 0.0
		if daysYear > 0 {
			entitlement = roundLeave(total * eligibleDays / daysYear)
		}
		var taken float64
		lr, le := s.Store.DB.Query(r.Context(), `SELECT start_date,end_date FROM leave_requests WHERE employee_id=$1 AND status='Approved' AND lower(leave_type)='annual leave' AND start_date <= $2 AND end_date >= $3`, id, end.AddDate(0, 0, -1), start)
		if le == nil {
			for lr.Next() {
				var a, b time.Time
				if lr.Scan(&a, &b) == nil {
					if a.Before(start) {
						a = start
					}
					if b.After(end.AddDate(0, 0, -1)) {
						b = end.AddDate(0, 0, -1)
					}
					for d := a; !d.After(b); d = d.AddDate(0, 0, 1) {
						wd := d.Weekday()
						key := d.Format("2006-01-02")
						if !weekends[wd] && !holidays[key] {
							taken++
						}
					}
				}
			}
			lr.Close()
		}
		items = append(items, map[string]any{"employeeId": id, "employeeNumber": num, "employeeName": name, "role": role, "joinDate": joined.Format("2006-01-02"), "totalLeaves": entitlement, "leavesTaken": taken, "remainingLeaves": roundLeave(entitlement - taken)})
	}
	writeJSON(w, 200, map[string]any{"year": year, "items": items})
}
func roundLeave(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
func (s *Server) getLeaveSettings(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "roles.manage") {
		writeError(w, 403, "forbidden", "Master Admin permission required")
		return
	}
	rows, e := s.Store.DB.Query(r.Context(), `SELECT role_name,annual_days FROM dcs_leave_role_entitlements ORDER BY role_name`)
	if e != nil {
		writeError(w, 500, "database_error", "Could not load leave entitlements")
		return
	}
	roles := []map[string]any{}
	for rows.Next() {
		var role string
		var days float64
		if rows.Scan(&role, &days) == nil {
			roles = append(roles, map[string]any{"role": role, "annualDays": days})
		}
	}
	rows.Close()
	hr, e := s.Store.DB.Query(r.Context(), `SELECT holiday_date::text,name FROM dcs_leave_holidays ORDER BY holiday_date`)
	if e != nil {
		writeError(w, 500, "database_error", "Could not load holidays")
		return
	}
	holidays := []map[string]string{}
	for hr.Next() {
		var d, n string
		if hr.Scan(&d, &n) == nil {
			holidays = append(holidays, map[string]string{"date": d, "name": n})
		}
	}
	hr.Close()
	weekendDays := []int{}
	wr, we := s.Store.DB.Query(r.Context(), `SELECT weekday FROM dcs_leave_weekend_days ORDER BY weekday`)
	if we == nil {
		for wr.Next() {
			var d int
			if wr.Scan(&d) == nil {
				weekendDays = append(weekendDays, d)
			}
		}
		wr.Close()
	}
	writeJSON(w, 200, map[string]any{"roles": roles, "holidays": holidays, "weekendDays": weekendDays})
}
func (s *Server) saveLeaveSettings(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "roles.manage") {
		writeError(w, 403, "forbidden", "Master Admin permission required")
		return
	}
	var in struct {
		Role          string  `json:"role"`
		AnnualDays    float64 `json:"annualDays"`
		Date          string  `json:"date"`
		Name          string  `json:"name"`
		DeleteHoliday string  `json:"deleteHoliday"`
		WeekendDays   []int   `json:"weekendDays"`
	}
	if decode(r, &in) != nil {
		writeError(w, 400, "validation_error", "Invalid leave settings")
		return
	}
	if in.WeekendDays != nil {
		tx, e := s.Store.DB.Begin(r.Context())
		if e != nil {
			writeError(w, 500, "database_error", "Could not save weekend days")
			return
		}
		if _, e = tx.Exec(r.Context(), `DELETE FROM dcs_leave_weekend_days`); e == nil {
			for _, d := range in.WeekendDays {
				if d < 0 || d > 6 {
					e = fmt.Errorf("invalid weekday")
					break
				}
				if _, e = tx.Exec(r.Context(), `INSERT INTO dcs_leave_weekend_days(weekday) VALUES($1) ON CONFLICT DO NOTHING`, d); e != nil {
					break
				}
			}
		}
		if e != nil {
			tx.Rollback(r.Context())
			writeError(w, 400, "validation_error", "Could not save weekend days")
			return
		}
		if e = tx.Commit(r.Context()); e != nil {
			writeError(w, 500, "database_error", "Could not save weekend days")
			return
		}
		writeJSON(w, 200, map[string]any{"saved": true})
		return
	}
	if in.Role != "" {
		if in.AnnualDays < 0 || in.AnnualDays > 366 {
			writeError(w, 400, "validation_error", "Annual leave must be between 0 and 366 days")
			return
		}
		_, e := s.Store.DB.Exec(r.Context(), `INSERT INTO dcs_leave_role_entitlements(role_name,annual_days,updated_by) VALUES($1,$2,$3) ON CONFLICT(role_name) DO UPDATE SET annual_days=EXCLUDED.annual_days,updated_by=EXCLUDED.updated_by,updated_at=now()`, strings.TrimSpace(in.Role), in.AnnualDays, u.ID)
		if e != nil {
			writeError(w, 500, "database_error", "Could not save role entitlement")
			return
		}
		s.Store.Audit(r.Context(), u.ID, "leave.entitlement.updated", "leave_entitlement", in.Role, "Updated annual leave entitlement", requestID(r))
		writeJSON(w, 200, map[string]any{"saved": true})
		return
	}
	if in.DeleteHoliday != "" {
		_, e := s.Store.DB.Exec(r.Context(), `DELETE FROM dcs_leave_holidays WHERE holiday_date=$1`, in.DeleteHoliday)
		if e != nil {
			writeError(w, 500, "database_error", "Could not delete holiday")
			return
		}
		writeJSON(w, 200, map[string]any{"saved": true})
		return
	}
	if _, e := time.Parse("2006-01-02", in.Date); e != nil || strings.TrimSpace(in.Name) == "" {
		writeError(w, 400, "validation_error", "Enter a valid holiday date and name")
		return
	}
	_, e := s.Store.DB.Exec(r.Context(), `INSERT INTO dcs_leave_holidays(holiday_date,name,created_by) VALUES($1,$2,$3) ON CONFLICT(holiday_date) DO UPDATE SET name=EXCLUDED.name`, in.Date, strings.TrimSpace(in.Name), u.ID)
	if e != nil {
		writeError(w, 500, "database_error", "Could not save holiday")
		return
	}
	writeJSON(w, 200, map[string]any{"saved": true})
}
