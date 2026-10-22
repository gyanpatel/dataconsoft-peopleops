package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"peopleops/internal/auth"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func countWorkingDays(start, end time.Time, weekends map[time.Weekday]bool, holidays map[string]bool) float64 {
	days := 0.0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if !weekends[d.Weekday()] && !holidays[d.Format("2006-01-02")] {
			days++
		}
	}
	return days
}
func (s *Server) workingLeaveDays(ctx context.Context, start, end time.Time) (float64, error) {
	if end.Before(start) || end.Sub(start) > 365*24*time.Hour {
		return 0, fmt.Errorf("Invalid date range")
	}
	weekends := map[time.Weekday]bool{}
	holidays := map[string]bool{}
	rows, err := s.Store.DB.Query(ctx, `SELECT weekday FROM dcs_leave_weekend_days`)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var d int
		if err = rows.Scan(&d); err != nil {
			rows.Close()
			return 0, err
		}
		weekends[time.Weekday(d)] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	rows, err = s.Store.DB.Query(ctx, `SELECT holiday_date::text FROM dcs_leave_holidays WHERE holiday_date BETWEEN $1 AND $2`, start, end)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var d string
		if err = rows.Scan(&d); err != nil {
			rows.Close()
			return 0, err
		}
		holidays[d] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	return countWorkingDays(start, end, weekends, holidays), nil
}
func (s *Server) leaveDuration(w http.ResponseWriter, r *http.Request) {
	a, e1 := time.Parse("2006-01-02", r.URL.Query().Get("startDate"))
	b, e2 := time.Parse("2006-01-02", r.URL.Query().Get("endDate"))
	if e1 != nil || e2 != nil {
		writeError(w, 400, "invalid_dates", "Enter valid start and end dates")
		return
	}
	days, err := s.workingLeaveDays(r.Context(), a, b)
	if err != nil {
		writeError(w, 400, "duration_unavailable", "Could not calculate working days; check the dates and calendar settings")
		return
	}
	writeJSON(w, 200, map[string]any{"days": days, "explanation": "Working days, excluding configured weekends and holidays"})
}

// Both individual and batch actions use the request primary key, never employee ID.
func (s *Server) decideLeave(ctx context.Context, u User, id, decision, note, rid string) (int, error) {
	if !s.hasPermission(ctx, u.Role, "leave.review") {
		return 403, fmt.Errorf("You cannot review leave requests")
	}
	if id == "" || (decision != "Approved" && decision != "Rejected") || len(note) > 1000 {
		return 400, fmt.Errorf("Invalid decision or note")
	}
	canReviewAll := s.hasPermission(ctx, u.Role, "employees.manage") || s.hasPermission(ctx, u.Role, "documents.manage")
	tx, err := s.Store.DB.Begin(ctx)
	if err != nil {
		return 500, fmt.Errorf("Could not start decision")
	}
	defer tx.Rollback(ctx)
	var emp, status string
	err = tx.QueryRow(ctx, `SELECT employee_id,status FROM leave_requests WHERE id=$1 FOR UPDATE`, id).Scan(&emp, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return 404, fmt.Errorf("Request not found")
	}
	if err != nil {
		return 500, fmt.Errorf("Could not read request")
	}
	if status != "Pending" {
		return 409, fmt.Errorf("Request already decided")
	}
	if emp == u.EmployeeID {
		return 403, fmt.Errorf("You cannot approve or reject your own request")
	}
	// Managers review their reporting tree; existing organization-wide HR reviewers retain their scope.
	if !canReviewAll {
		var member bool
		err = tx.QueryRow(ctx, `WITH RECURSIVE team AS (SELECT id FROM employees WHERE manager_id=$1 UNION SELECT e.id FROM employees e JOIN team t ON e.manager_id=t.id) SELECT EXISTS(SELECT 1 FROM team WHERE id=$2)`, u.EmployeeID, emp).Scan(&member)
		if err != nil {
			return 500, fmt.Errorf("Could not validate reporting scope")
		}
		if !member {
			return 403, fmt.Errorf("This employee is outside your reporting team")
		}
	}
	tag, err := tx.Exec(ctx, `UPDATE leave_requests SET status=$1,decided_at=now(),decided_by=$2,decision_note=$3 WHERE id=$4 AND status='Pending'`, decision, u.ID, strings.TrimSpace(note), id)
	if err != nil || tag.RowsAffected() != 1 {
		return 409, fmt.Errorf("Request changed; refresh and try again")
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(id,actor_user_id,action,target_type,target_id,summary,request_id,created_at) VALUES($1,$2,$3,'leave_request',$4,$5,$6,now())`, auth.NewID(), u.ID, "leave."+strings.ToLower(decision), id, "Leave request decision: "+decision, rid)
	if err != nil {
		return 500, fmt.Errorf("Could not record decision history")
	}
	if err = tx.Commit(ctx); err != nil {
		return 500, fmt.Errorf("Could not save decision")
	}
	return 200, nil
}
func (s *Server) bulkLeaveDecision(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs      []string `json:"ids"`
		Decision string   `json:"decision"`
		Note     string   `json:"note"`
	}
	if decode(r, &in) != nil || len(in.IDs) == 0 || len(in.IDs) > 200 || (in.Decision != "Approved" && in.Decision != "Rejected") || len(in.Note) > 1000 {
		writeError(w, 400, "invalid_request", "Select 1–200 requests and a valid decision")
		return
	}
	results := []map[string]any{}
	seen := map[string]bool{}
	for _, id := range in.IDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		code, err := s.decideLeave(r.Context(), current(r), id, in.Decision, in.Note, requestID(r))
		message := "Request " + strings.ToLower(in.Decision)
		if err != nil {
			message = err.Error()
		}
		results = append(results, map[string]any{"id": id, "ok": err == nil, "status": code, "message": message})
	}
	writeJSON(w, 200, map[string]any{"results": results})
}

func (s *Server) assignShift(ctx context.Context, u User, shiftID, employeeID, date, note string, preview ...bool) (string, error) {
	if !s.hasPermission(ctx, u.Role, "shift.assign") {
		return "", fmt.Errorf("Shift assignment permission required")
	}
	day, err := time.Parse("2006-01-02", date)
	if err != nil || len(note) > 1000 {
		return "", fmt.Errorf("Invalid date or note")
	}
	tx, err := s.Store.DB.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("Could not start assignment")
	}
	defer tx.Rollback(ctx)
	var employee string
	// Serialize assignments for an employee, including across midnight on adjacent days.
	if err = tx.QueryRow(ctx, `SELECT id FROM employees WHERE id=$1 AND status='Active' FOR UPDATE`, employeeID).Scan(&employee); err != nil {
		return "", fmt.Errorf("Employee is not active or could not be loaded")
	}
	var start, end time.Time
	err = tx.QueryRow(ctx, `SELECT $2::date+start_time,$2::date+end_time+CASE WHEN end_time<=start_time THEN interval '1 day' ELSE interval '0 day' END FROM shifts WHERE id=$1 AND active=true`, shiftID, day).Scan(&start, &end)
	if err != nil {
		return "", fmt.Errorf("Shift is not active or could not be loaded")
	}
	var overlaps bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM shift_assignments a JOIN shifts s ON s.id=a.shift_id WHERE a.employee_id=$1 AND a.status='Assigned' AND a.shift_date+s.start_time < $3::timestamp AND a.shift_date+s.end_time+CASE WHEN s.end_time<=s.start_time THEN interval '1 day' ELSE interval '0 day' END > $2::timestamp)`, employeeID, start, end).Scan(&overlaps)
	if err != nil {
		return "", fmt.Errorf("Could not check assignment conflicts")
	}
	if overlaps {
		return "", fmt.Errorf("Conflicts with an existing shift; nothing was replaced")
	}
	if len(preview) > 0 && preview[0] {
		return "", nil
	}
	id := auth.NewID()
	_, err = tx.Exec(ctx, `INSERT INTO shift_assignments(id,shift_id,employee_id,shift_date,status,note,created_by,created_at) VALUES($1,$2,$3,$4,'Assigned',$5,$6,now())`, id, shiftID, employeeID, day, strings.TrimSpace(note), u.ID)
	if err != nil {
		return "", fmt.Errorf("Could not save assignment; check for duplicates and schema migration 019")
	}
	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("Could not commit assignment")
	}
	return id, nil
}
func (s *Server) bulkShiftAssignments(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ShiftID     string   `json:"shiftId"`
		EmployeeIDs []string `json:"employeeIds"`
		Date        string   `json:"date"`
		EndDate     string   `json:"endDate"`
		Note        string   `json:"note"`
		Preview     bool     `json:"preview"`
	}
	if decode(r, &in) != nil || len(in.EmployeeIDs) == 0 || len(in.EmployeeIDs) > 200 {
		writeError(w, 400, "invalid_request", "Select 1–200 employees")
		return
	}
	start, e1 := time.Parse("2006-01-02", in.Date)
	if in.EndDate == "" {
		in.EndDate = in.Date
	}
	end, e2 := time.Parse("2006-01-02", in.EndDate)
	if e1 != nil || e2 != nil || end.Before(start) || end.Sub(start) > 30*24*time.Hour || int(end.Sub(start).Hours()/24+1)*len(in.EmployeeIDs) > 500 {
		writeError(w, 400, "invalid_dates", "Select up to 31 days and 500 total assignments per batch")
		return
	}
	results := []map[string]any{}
	seen := map[string]bool{}
	for _, emp := range in.EmployeeIDs {
		if seen[emp] {
			continue
		}
		seen[emp] = true
		for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
			date := day.Format("2006-01-02")
			id, err := s.assignShift(r.Context(), current(r), in.ShiftID, emp, date, in.Note, in.Preview)
			message := "Assigned"
			if in.Preview {
				message = "Available"
			}
			if err != nil {
				message = err.Error()
			} else if !in.Preview {
				s.Store.Audit(r.Context(), current(r).ID, "shift.assigned", "shift_assignment", id, "Assigned shift on "+date, requestID(r))
			}
			results = append(results, map[string]any{"employeeId": emp, "date": date, "ok": err == nil, "message": message})
		}
	}
	writeJSON(w, 200, map[string]any{"results": results, "preview": in.Preview})
}

var modules = [][2]string{{"dashboard", "Dashboard"}, {"employees", "Employees"}, {"employee-hierarchy", "Employee hierarchy"}, {"salary-breakup", "Salary breakup"}, {"bank-details", "Bank details"}, {"attendance", "Attendance"}, {"shift-management", "Shift management"}, {"shift-assignments", "Shift assignments"}, {"leave-management", "Leave management"}, {"leave-balance", "Leave balance"}, {"bonus-management", "Bonus management"}, {"payroll", "Payroll"}, {"documents", "Documents"}, {"approvals", "Approvals"}, {"reports-analytics", "Reports & analytics"}, {"audit-log", "Audit log"}, {"settings", "Settings"}, {"my-profile", "My profile"}}

func (s *Server) moduleCatalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"items": modules})
}
func (s *Server) rolePermissions(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.adminRole(r.Context(), u.Role) {
		writeError(w, 403, "forbidden", "Role management permission required")
		return
	}
	if r.Method == http.MethodGet {
		rows, err := s.Store.DB.Query(r.Context(), `SELECT keys.permission_key,COALESCE(p.enabled,false) FROM (SELECT DISTINCT permission_key FROM dcs_role_permissions) keys LEFT JOIN dcs_role_permissions p ON p.permission_key=keys.permission_key AND p.role_id=$1 ORDER BY keys.permission_key`, r.URL.Query().Get("roleId"))
		if err != nil {
			writeError(w, 500, "database_error", "Could not read actions")
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var key string
			var enabled bool
			if rows.Scan(&key, &enabled) != nil {
				writeError(w, 500, "database_error", "Could not read action")
				return
			}
			items = append(items, map[string]any{"key": key, "enabled": enabled})
		}
		writeJSON(w, 200, map[string]any{"items": items})
		return
	}
	var in struct {
		RoleID string `json:"roleId"`
		Items  []struct {
			Key     string `json:"key"`
			Enabled bool   `json:"enabled"`
		} `json:"items"`
	}
	if decode(r, &in) != nil || in.RoleID == "" {
		writeError(w, 400, "invalid_request", "Select a role")
		return
	}
	tx, err := s.Store.DB.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "database_error", "Could not save actions")
		return
	}
	defer tx.Rollback(r.Context())
	// Do not modify the existing administrator privilege model in this task.
	var name string
	if err = tx.QueryRow(r.Context(), `SELECT role_name FROM dcs_app_roles WHERE id=$1`, in.RoleID).Scan(&name); err != nil {
		writeError(w, 404, "not_found", "Role not found")
		return
	}
	if name == "Master Admin" || name == "Super Admin" {
		writeError(w, 403, "protected_role", "Administrator action permissions are protected")
		return
	}
	for _, p := range in.Items {
		var known bool
		err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM dcs_role_permissions WHERE permission_key=$1)`, p.Key).Scan(&known)
		if err != nil || !known {
			writeError(w, 400, "unknown_permission", "Unknown action")
			return
		}
		// Privileged management capabilities must not be delegated through this editor.
		if p.Key == "roles.manage" || p.Key == "app.configure" {
			continue
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO dcs_role_permissions(role_id,permission_key,enabled) VALUES($1,$2,$3) ON CONFLICT(role_id,permission_key) DO UPDATE SET enabled=EXCLUDED.enabled,updated_at=now()`, in.RoleID, p.Key, p.Enabled)
		if err != nil {
			writeError(w, 500, "database_error", "Could not save actions")
			return
		}
	}
	if tx.Commit(r.Context()) != nil {
		writeError(w, 500, "database_error", "Could not commit actions")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "role.actions_updated", "role", in.RoleID, "Updated allowed actions", requestID(r))
	writeJSON(w, 200, map[string]bool{"saved": true})
}
