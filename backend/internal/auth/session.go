package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	cookieName    = "conveyor_session"
	sessionMaxAge = 7 * 24 * time.Hour
)

// SessionManager issues and verifies HMAC-signed, httpOnly session cookies.
// The cookie carries only the user id and an expiry; both are signed so a
// client cannot forge or tamper with them. No server-side session store needed.
type SessionManager struct {
	secret []byte
	secure bool
}

func NewSessionManager(secret string, secure bool) *SessionManager {
	return &SessionManager{secret: []byte(secret), secure: secure}
}

func (m *SessionManager) sign(msg string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// SameSite returns the cookie policy: None for HTTPS (so the cookie works
// cross-site when the frontend is on another domain), Lax otherwise.
func (m *SessionManager) SameSite() http.SameSite {
	if m.secure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

// Issue sets a signed session cookie for the given user.
func (m *SessionManager) Issue(w http.ResponseWriter, userID uuid.UUID) {
	exp := time.Now().Add(sessionMaxAge).Unix()
	payload := fmt.Sprintf("%s|%d", userID.String(), exp)
	token := payload + "|" + m.sign(payload)
	value := base64.RawURLEncoding.EncodeToString([]byte(token))

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: m.SameSite(),
		Expires:  time.Unix(exp, 0),
		MaxAge:   int(sessionMaxAge.Seconds()),
	})
}

// Clear removes the session cookie.
func (m *SessionManager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: m.SameSite(),
		MaxAge:   -1,
	})
}

// UserID returns the authenticated user id if the cookie is present,
// well-formed, correctly signed, and unexpired.
func (m *SessionManager) UserID(r *http.Request) (uuid.UUID, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return uuid.Nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return uuid.Nil, false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return uuid.Nil, false
	}
	payload := parts[0] + "|" + parts[1]
	if !hmac.Equal([]byte(m.sign(payload)), []byte(parts[2])) {
		return uuid.Nil, false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
