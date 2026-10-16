package auth

import (
	"testing"
	"time"
)

func TestPasswordHash(t *testing.T) {
	h, e := HashPassword("ExamplePass123!")
	if e != nil {
		t.Fatal(e)
	}
	if !CheckPassword(h, "ExamplePass123!") {
		t.Fatal("valid password rejected")
	}
	if CheckPassword(h, "wrong") {
		t.Fatal("invalid password accepted")
	}
}
func TestSignedSession(t *testing.T) {
	secret := "a-test-secret-with-more-than-thirty-two-characters"
	token, e := Issue(secret, Claims{UserID: "user-1", Email: "test@example.com", Role: "Employee"})
	if e != nil {
		t.Fatal(e)
	}
	c, e := Verify(secret, token)
	if e != nil || c.UserID != "user-1" {
		t.Fatalf("verify failed: %+v %v", c, e)
	}
	if _, e = Verify("wrong-secret", token); e == nil {
		t.Fatal("expected signature validation failure")
	}
	c.Expires = time.Now().Add(-time.Hour).Unix()
	_ = c
}
