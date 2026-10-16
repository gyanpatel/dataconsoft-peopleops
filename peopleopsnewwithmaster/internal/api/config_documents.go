package api

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"peopleops/internal/auth"
)


func (s *Server) appMaster(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if r.Method != http.MethodGet && !s.hasPermission(r.Context(), u.Role, "app.configure") {
		writeError(w, 403, "forbidden", "Master Admin role required to change application configuration")
		return
	}
	switch r.Method {
	case http.MethodGet:
		page := strings.TrimSpace(r.URL.Query().Get("page"))
		rows, err := s.Store.DB.Query(r.Context(), `SELECT id,page_id,page_name,page_item,field_key,label,data_type,required,visible,display_order,options,config,help_text FROM dcs_app_master WHERE ($1='' OR page_id=$1) ORDER BY page_id,display_order,id`, page)
		if err != nil {
			writeError(w, 500, "database_error", "Could not load application configuration")
			return
		}
		defer rows.Close()
		items := make([]map[string]any, 0)
		for rows.Next() {
			var id int64
			var pageID, pageName, item, key, label, typ, help string
			var required, visible bool
			var order int
			var opts, cfg []byte
			if err := rows.Scan(&id, &pageID, &pageName, &item, &key, &label, &typ, &required, &visible, &order, &opts, &cfg, &help); err != nil {
				writeError(w, 500, "database_error", "Could not read application configuration")
				return
			}
			var options any = []any{}
			var config any = map[string]any{}
			_ = json.Unmarshal(opts, &options)
			_ = json.Unmarshal(cfg, &config)
			items = append(items, map[string]any{"id": id, "pageId": pageID, "pageName": pageName, "pageItem": item, "fieldKey": key, "label": label, "dataType": typ, "required": required, "visible": visible, "displayOrder": order, "options": options, "config": config, "helpText": help})
		}
		writeJSON(w, 200, map[string]any{"items": items, "count": len(items)})
	case http.MethodPost:
		var in struct {
			PageID       string         `json:"pageId"`
			PageName     string         `json:"pageName"`
			PageItem     string         `json:"pageItem"`
			FieldKey     string         `json:"fieldKey"`
			Label        string         `json:"label"`
			DataType     string         `json:"dataType"`
			Required     bool           `json:"required"`
			Visible      bool           `json:"visible"`
			DisplayOrder int            `json:"displayOrder"`
			Options      []string       `json:"options"`
			Config       map[string]any `json:"config"`
			HelpText     string         `json:"helpText"`
		}
		if decode(r, &in) != nil || !validConfig(in.PageID, in.PageName, in.FieldKey, in.Label, in.DataType) {
			writeError(w, 400, "validation_error", "Provide valid page, field key, label and supported data type")
			return
		}
		opts, _ := json.Marshal(in.Options)
		cfg, _ := json.Marshal(in.Config)
		id := int64(0)
		err := s.Store.DB.QueryRow(r.Context(), `INSERT INTO dcs_app_master(page_id,page_name,page_item,field_key,label,data_type,required,visible,display_order,options,config,help_text) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11::jsonb,$12) RETURNING id`, in.PageID, in.PageName, in.PageItem, in.FieldKey, in.Label, in.DataType, in.Required, in.Visible, in.DisplayOrder, opts, cfg, in.HelpText).Scan(&id)
		if err != nil {
			writeError(w, 409, "conflict", "A field with this key already exists for the page")
			return
		}
		s.Store.Audit(r.Context(), u.ID, "app_config.created", "dcs_app_master", fmt.Sprint(id), "Added configurable field "+in.Label, requestID(r))
		writeJSON(w, 201, map[string]any{"id": id})
	case http.MethodPut:
		var in struct {
			ID           int64          `json:"id"`
			PageID       string         `json:"pageId"`
			PageName     string         `json:"pageName"`
			PageItem     string         `json:"pageItem"`
			FieldKey     string         `json:"fieldKey"`
			Label        string         `json:"label"`
			DataType     string         `json:"dataType"`
			Required     bool           `json:"required"`
			Visible      bool           `json:"visible"`
			DisplayOrder int            `json:"displayOrder"`
			Options      []string       `json:"options"`
			Config       map[string]any `json:"config"`
			HelpText     string         `json:"helpText"`
		}
		if decode(r, &in) != nil || in.ID < 1 || !validConfig(in.PageID, in.PageName, in.FieldKey, in.Label, in.DataType) {
			writeError(w, 400, "validation_error", "Invalid application configuration")
			return
		}
		opts, _ := json.Marshal(in.Options)
		cfg, _ := json.Marshal(in.Config)
		tag, err := s.Store.DB.Exec(r.Context(), `UPDATE dcs_app_master SET page_id=$2,page_name=$3,page_item=$4,field_key=$5,label=$6,data_type=$7,required=$8,visible=$9,display_order=$10,options=$11::jsonb,config=$12::jsonb,help_text=$13,updated_at=NOW() WHERE id=$1`, in.ID, in.PageID, in.PageName, in.PageItem, in.FieldKey, in.Label, in.DataType, in.Required, in.Visible, in.DisplayOrder, opts, cfg, in.HelpText)
		if err != nil {
			writeError(w, 409, "conflict", "Could not update configuration; check for duplicate field keys")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, 404, "not_found", "Configuration item not found")
			return
		}
		s.Store.Audit(r.Context(), u.ID, "app_config.updated", "dcs_app_master", fmt.Sprint(in.ID), "Updated configurable field "+in.Label, requestID(r))
		writeJSON(w, 200, map[string]any{"message": "Configuration updated"})
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, 400, "validation_error", "id is required")
			return
		}
		tag, err := s.Store.DB.Exec(r.Context(), `DELETE FROM dcs_app_master WHERE id=$1`, id)
		if err != nil {
			writeError(w, 500, "database_error", "Could not delete configuration item")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, 404, "not_found", "Configuration item not found")
			return
		}
		s.Store.Audit(r.Context(), u.ID, "app_config.deleted", "dcs_app_master", id, "Deleted configurable field", requestID(r))
		w.WriteHeader(204)
	default:
		writeError(w, 405, "method_not_allowed", "Method not allowed")
	}
}
func validConfig(pageID, pageName, key, label, typ string) bool {
	if len(pageID) < 2 || len(pageID) > 80 || len(pageName) < 2 || len(pageName) > 100 || len(key) < 2 || len(key) > 80 || len(label) < 1 || len(label) > 120 {
		return false
	}
	for _, c := range key {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	switch typ {
	case "text", "textarea", "number", "date", "email", "select", "checkbox", "url":
		return true
	}
	return false
}

const maxDocumentBytes = 15 << 20

func (s *Server) documents(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if r.Method == http.MethodPost && !s.hasPermission(r.Context(), u.Role, "documents.manage") && u.EmployeeID == "" {
		writeError(w, 403, "forbidden", "Employee account or HR document management permission required")
		return
	}
	if r.Method == http.MethodGet {
		rows, err := s.Store.DB.Query(r.Context(), `SELECT d.id,d.employee_id,e.first_name||' '||e.last_name,d.document_name,d.document_type,d.mime_type,d.file_size,COALESCE(d.issue_date::text,''),COALESCE(d.expiry_date::text,''),d.verification_status,d.notes,d.uploaded_at FROM dcs_doc_store d JOIN employees e ON e.id=d.employee_id WHERE ($1 OR d.employee_id=$2) ORDER BY d.uploaded_at DESC LIMIT 500`, s.hasPermission(r.Context(), u.Role, "documents.manage"), u.EmployeeID)
		if err != nil {
			writeError(w, 500, "database_error", "Could not load documents")
			return
		}
		defer rows.Close()
		items := make([]map[string]any, 0)
		for rows.Next() {
			var id, eid, name, docName, typ, mimeType, issue, expiry, status, notes string
			var size int64
			var uploaded time.Time
			if err := rows.Scan(&id, &eid, &name, &docName, &typ, &mimeType, &size, &issue, &expiry, &status, &notes, &uploaded); err != nil {
				writeError(w, 500, "database_error", "Could not read documents")
				return
			}
			items = append(items, map[string]any{"id": id, "employeeId": eid, "employeeName": name, "documentName": docName, "documentType": typ, "mimeType": mimeType, "fileSize": size, "issueDate": issue, "expiryDate": expiry, "verificationStatus": status, "notes": notes, "uploadedAt": uploaded})
		}
		writeJSON(w, 200, map[string]any{"items": items, "count": len(items)})
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, 405, "method_not_allowed", "Method not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentBytes+(1<<20))
	if err := r.ParseMultipartForm(maxDocumentBytes); err != nil {
		writeError(w, 413, "file_too_large", "Document must be smaller than 15 MB")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "validation_error", "Choose a document file")
		return
	}
	defer file.Close()
	if header.Size <= 0 || header.Size > maxDocumentBytes {
		writeError(w, 413, "file_too_large", "Document must be between 1 byte and 15 MB")
		return
	}
	employeeID := strings.TrimSpace(r.FormValue("employeeId"))
	if !s.hasPermission(r.Context(), u.Role, "documents.manage") {
		employeeID = u.EmployeeID
	}
	if employeeID == "" {
		employeeID = u.EmployeeID
	}
	if employeeID == "" {
		writeError(w, 400, "validation_error", "Select an employee")
		return
	}
	var exists int
	if err := s.Store.DB.QueryRow(r.Context(), `SELECT COUNT(*) FROM employees WHERE id=$1`, employeeID).Scan(&exists); err != nil || exists == 0 {
		writeError(w, 400, "invalid_employee", "Select a valid employee")
		return
	}
	docType := strings.TrimSpace(r.FormValue("documentType"))
	if docType == "" {
		docType = "Other"
	}
	docName := filepath.Base(header.Filename)
	if len(docName) > 180 {
		docName = docName[len(docName)-180:]
	}
	ext := strings.ToLower(filepath.Ext(docName))
	allowed := map[string]bool{".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".docx": true, ".xlsx": true, ".txt": true}
	if !allowed[ext] {
		writeError(w, 400, "unsupported_file", "Allowed formats: PDF, PNG, JPG, DOCX, XLSX and TXT")
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxDocumentBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxDocumentBytes {
		writeError(w, 413, "file_too_large", "Document must be smaller than 15 MB")
		return
	}
	mimeType := http.DetectContentType(data)
	if ext == ".docx" || ext == ".xlsx" {
		mimeType = "application/octet-stream"
	}
	if ext == ".txt" {
		mimeType = "text/plain; charset=utf-8"
	}
	issue := nullableDate(r.FormValue("issueDate"))
	expiry := nullableDate(r.FormValue("expiryDate"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	if len(notes) > 1000 {
		writeError(w, 400, "validation_error", "Notes are limited to 1000 characters")
		return
	}
	id := auth.NewID()
	_, err = s.Store.DB.Exec(r.Context(), `INSERT INTO dcs_doc_store(id,employee_id,document_name,document_type,mime_type,file_size,doc_file,issue_date,expiry_date,notes,uploaded_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, employeeID, docName, docType, mimeType, len(data), data, issue, expiry, notes, u.ID)
	if err != nil {
		writeError(w, 500, "database_error", "Could not store document")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "document.uploaded", "dcs_doc_store", id, "Uploaded document: "+docName, requestID(r))
	writeJSON(w, 201, map[string]any{"id": id, "message": "Document uploaded"})
}
func nullableDate(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return parsed
}
func (s *Server) documentDownload(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	id := r.PathValue("id")
	var eid, name, mimeType string
	var data []byte
	err := s.Store.DB.QueryRow(r.Context(), `SELECT employee_id,document_name,mime_type,doc_file FROM dcs_doc_store WHERE id=$1`, id).Scan(&eid, &name, &mimeType, &data)
	if err != nil {
		writeError(w, 404, "not_found", "Document not found")
		return
	}
	if !s.hasPermission(r.Context(), u.Role, "documents.manage") && eid != u.EmployeeID {
		writeError(w, 403, "forbidden", "You cannot access this document")
		return
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	w.WriteHeader(200)
	_, _ = w.Write(data)
}
func (s *Server) documentDecision(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !s.hasPermission(r.Context(), u.Role, "documents.manage") {
		writeError(w, 403, "forbidden", "HR document management permission required")
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if decode(r, &in) != nil || (in.Status != "Verified" && in.Status != "Rejected" && in.Status != "Pending") {
		writeError(w, 400, "validation_error", "Status must be Verified, Rejected or Pending")
		return
	}
	id := r.PathValue("id")
	tag, err := s.Store.DB.Exec(r.Context(), `UPDATE dcs_doc_store SET verification_status=$2,updated_at=NOW() WHERE id=$1`, id, in.Status)
	if err != nil {
		writeError(w, 500, "database_error", "Could not update document status")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, 404, "not_found", "Document not found")
		return
	}
	s.Store.Audit(r.Context(), u.ID, "document.status_changed", "dcs_doc_store", id, "Set document status to "+in.Status, requestID(r))
	writeJSON(w, 200, map[string]any{"message": "Document status updated"})
}

func parseShiftClock(value string) time.Time { parsed, _ := time.Parse("15:04", value); return parsed }
