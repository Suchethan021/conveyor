package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// reqWithIssuedCookie issues a session for id via m and returns a request
// carrying the resulting cookie.
func reqWithIssuedCookie(t *testing.T, m *SessionManager, id uuid.UUID) *http.Request {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Issue(rec, id)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

func TestSessionRoundTrip(t *testing.T) {
	m := NewSessionManager("test-secret", false)
	id := uuid.New()

	got, ok := m.UserID(reqWithIssuedCookie(t, m, id))
	if !ok || got != id {
		t.Fatalf("round trip failed: got=%v ok=%v want=%v", got, ok, id)
	}
}

func TestSessionRejectsTamperedCookie(t *testing.T) {
	m := NewSessionManager("test-secret", false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "garbage-not-a-token"})

	if _, ok := m.UserID(req); ok {
		t.Fatal("expected a tampered/invalid cookie to be rejected")
	}
}

func TestSessionRejectsWrongSecret(t *testing.T) {
	issuer := NewSessionManager("secret-a", false)
	verifier := NewSessionManager("secret-b", false)

	req := reqWithIssuedCookie(t, issuer, uuid.New())
	if _, ok := verifier.UserID(req); ok {
		t.Fatal("cookie signed with a different secret must be rejected")
	}
}

func TestSessionNoCookie(t *testing.T) {
	m := NewSessionManager("test-secret", false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := m.UserID(req); ok {
		t.Fatal("expected no cookie to be unauthenticated")
	}
}
