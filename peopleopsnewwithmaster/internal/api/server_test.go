package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"peopleops/internal/auth"
	"peopleops/internal/store"
)

func TestHealth(t *testing.T) {
	s, e := store.Open()
	if e != nil {
		t.Fatal(e)
	}
	defer s.DB.Close()
	h := (&Server{Store: s, Secret: "test-secret-that-is-long-enough-for-tests", Origin: ""}).Handler()
	r := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}
func TestRoleChecks(t *testing.T) {
	if !hasRole("Super Admin", "Super Admin", "HR Admin") {
		t.Fatal("expected role match")
	}
	if hasRole("Employee", "Super Admin", "HR Admin") {
		t.Fatal("employee should not match HR roles")
	}
}

func TestShiftAssignmentAndDuplicateProtection(t *testing.T) {
	st, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer st.DB.Close()
	const secret = "test-secret-that-is-long-enough-for-tests"
	var userID string
	if err := st.DB.QueryRow(context.Background(), `SELECT id FROM users WHERE email='admin@peopleops.local'`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	token, err := auth.Issue(secret, auth.Claims{UserID: userID, Email: "admin@peopleops.local", Role: "Super Admin"})
	if err != nil {
		t.Fatal(err)
	}
	var shiftID, employeeID string
	if err := st.DB.QueryRow(context.Background(), `SELECT id FROM shifts LIMIT 1`).Scan(&shiftID); err != nil {
		t.Fatal(err)
	}
	if err := st.DB.QueryRow(context.Background(), `SELECT id FROM employees WHERE status='Active' LIMIT 1`).Scan(&employeeID); err != nil {
		t.Fatal(err)
	}
	h := (&Server{Store: st, Secret: secret}).Handler()
	call := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.AddCookie(&http.Cookie{Name: "peopleops_session", Value: token})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	body := `{"shiftId":"` + shiftID + `","employeeId":"` + employeeID + `","date":"2026-11-02","note":"test assignment"}`
	if w := call(http.MethodPost, "/api/v1/shift-assignments", body); w.Code != http.StatusCreated {
		t.Fatalf("first assignment status=%d body=%s", w.Code, w.Body.String())
	}
	if w := call(http.MethodPost, "/api/v1/shift-assignments", body); w.Code != http.StatusConflict {
		t.Fatalf("duplicate assignment status=%d body=%s", w.Code, w.Body.String())
	}
	if w := call(http.MethodGet, "/api/v1/shift-assignments", ""); w.Code != http.StatusOK {
		t.Fatalf("list assignments status=%d body=%s", w.Code, w.Body.String())
	}
}
