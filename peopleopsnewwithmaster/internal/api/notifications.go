package api

import (
	"net/http"
	"strconv"
	"time"
)

// notificationSettings returns the document-expiry reminder window. Only Master Admin
// can change it; reads are available to signed-in users so the bell can explain its scope.
func (s *Server) notificationSettings(w http.ResponseWriter, r *http.Request) {
	days := 30
	_ = s.Store.DB.QueryRow(r.Context(), `SELECT setting_value::integer FROM dcs_app_settings WHERE setting_key='document_expiry_notice_days'`).Scan(&days)
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"documentExpiryNoticeDays": days})
		return
	}
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "app.configure") {
		writeError(w, http.StatusForbidden, "forbidden", "Master Admin role required to change notification settings")
		return
	}
	var in struct {
		DocumentExpiryNoticeDays int `json:"documentExpiryNoticeDays"`
	}
	if decode(r, &in) != nil || in.DocumentExpiryNoticeDays < 1 || in.DocumentExpiryNoticeDays > 365 {
		writeError(w, http.StatusBadRequest, "validation_error", "Document expiry reminder days must be between 1 and 365")
		return
	}
	_, err := s.Store.DB.Exec(r.Context(), `INSERT INTO dcs_app_settings(setting_key,setting_value,updated_at,updated_by) VALUES('document_expiry_notice_days',$1,NOW(),$2) ON CONFLICT(setting_key) DO UPDATE SET setting_value=EXCLUDED.setting_value,updated_at=NOW(),updated_by=EXCLUDED.updated_by`, strconv.Itoa(in.DocumentExpiryNoticeDays), u.ID)
	if err != nil {
		writeError(w, 500, "database_error", "Could not save notification settings")
		return
	}
	writeJSON(w, 200, map[string]any{"documentExpiryNoticeDays": in.DocumentExpiryNoticeDays})
}

func (s *Server) notifications(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	days := 30
	_ = s.Store.DB.QueryRow(r.Context(), `SELECT setting_value::integer FROM dcs_app_settings WHERE setting_key='document_expiry_notice_days'`).Scan(&days)
	items := make([]map[string]any, 0)
	// Employees only see their own expiring documents. HR/document managers see all.
	canManageDocs := s.hasPermission(r.Context(), u.Role, "documents.manage")
	if canManageDocs || u.EmployeeID != "" {
		rows, err := s.Store.DB.Query(r.Context(), `SELECT d.id,d.document_name,d.document_type,d.expiry_date::text,e.id,e.first_name||' '||e.last_name FROM dcs_doc_store d JOIN employees e ON e.id=d.employee_id WHERE d.expiry_date IS NOT NULL AND d.expiry_date <= CURRENT_DATE + $1::int AND ($2 OR d.employee_id=$3) ORDER BY d.expiry_date,d.document_name`, days, canManageDocs, u.EmployeeID)
		if err != nil {
			writeError(w, 500, "database_error", "Could not load document-expiry notifications")
			return
		}
		for rows.Next() {
			var id, name, typ, expiry, employeeID, employeeName string
			if err := rows.Scan(&id, &name, &typ, &expiry, &employeeID, &employeeName); err != nil {
				rows.Close()
				writeError(w, 500, "database_error", "Could not read document-expiry notifications")
				return
			}
			message := typ + " “" + name + "” expires on " + expiry + "."
			if expiry < time.Now().Format("2006-01-02") {
				message = typ + " “" + name + "” expired on " + expiry + "."
			}
			if canManageDocs {
				message = employeeName + " — " + message
			}
			items = append(items, map[string]any{"id": "document-expiry:" + id, "type": "document_expiry", "title": "Document expiring soon", "message": message, "date": expiry, "page": "Documents", "employeeId": employeeID})
		}
		rows.Close()
	}
	// HR users who can review leave receive pending leave-review notifications.
	if s.hasPermission(r.Context(), u.Role, "leave.review") {
		rows, err := s.Store.DB.Query(r.Context(), `SELECT l.id,e.first_name||' '||e.last_name,l.leave_type,l.start_date::text,l.end_date::text FROM leave_requests l JOIN employees e ON e.id=l.employee_id WHERE lower(l.status)='pending' ORDER BY l.submitted_at DESC LIMIT 50`)
		if err != nil {
			writeError(w, 500, "database_error", "Could not load leave notifications")
			return
		}
		for rows.Next() {
			var id, employee, leaveType, start, end string
			if err := rows.Scan(&id, &employee, &leaveType, &start, &end); err != nil {
				rows.Close()
				writeError(w, 500, "database_error", "Could not read leave notifications")
				return
			}
			items = append(items, map[string]any{"id": "leave-pending:" + id, "type": "leave_pending", "title": "Leave request needs review", "message": employee + " requested " + leaveType + " from " + start + " to " + end + ".", "date": start, "page": "Leave management"})
		}
		rows.Close()
	}
	// Stable ordering: earliest document expiry and leave dates first.
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items), "documentExpiryNoticeDays": days, "generatedAt": time.Now().UTC().Format(time.RFC3339)})
}
