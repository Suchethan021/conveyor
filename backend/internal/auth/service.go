package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Suchethan021/conveyor/backend/internal/config"
	"github.com/Suchethan021/conveyor/backend/internal/db/sqlc"
	"github.com/Suchethan021/conveyor/backend/internal/httpx"
)

const stateCookie = "conveyor_oauth_state"

// Service wires session management, the GitHub OAuth flow, and user lookups.
// The OAuth client is optional: if GitHub credentials are unset, login routes
// return 503 while session-based routes (/me, logout) still function.
type Service struct {
	sessions    *SessionManager
	oauth       *GitHubOAuth
	queries     *sqlc.Queries
	frontendURL string
	secure      bool
	devLogin    bool
}

// NewService constructs the auth service. SESSION_SECRET is required; GitHub
// OAuth is enabled only when both client id and secret are present.
func NewService(cfg *config.Config, queries *sqlc.Queries) (*Service, error) {
	if cfg.SessionSecret == "" {
		return nil, errMissing("SESSION_SECRET")
	}
	secure := strings.HasPrefix(cfg.GitHubCallbackURL, "https://")

	var oauth *GitHubOAuth
	if cfg.GitHubClientID != "" && cfg.GitHubClientSecret != "" {
		oauth = NewGitHubOAuth(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.GitHubCallbackURL)
	}

	return &Service{
		sessions:    NewSessionManager(cfg.SessionSecret, secure),
		oauth:       oauth,
		queries:     queries,
		frontendURL: cfg.FrontendURL,
		secure:      secure,
		devLogin:    cfg.AllowDevLogin,
	}, nil
}

// OAuthEnabled reports whether GitHub login is configured.
func (s *Service) OAuthEnabled() bool { return s.oauth != nil }

// Mount registers the auth routes on the given router.
func (s *Service) Mount(r chi.Router) {
	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/github/login", s.handleGitHubLogin)
		r.Get("/github/callback", s.handleGitHubCallback)
		r.Post("/logout", s.handleLogout)
		if s.devLogin {
			r.Post("/dev-login", s.handleDevLogin)
		}
	})
	r.With(RequireAuth).Get("/api/me", s.handleMe)
}

// handleDevLogin issues a session for a fixed local test user. Only mounted
// when ALLOW_DEV_LOGIN=true; never enable in a real deployment.
func (s *Service) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	user, err := s.queries.UpsertUserByGithubID(r.Context(), sqlc.UpsertUserByGithubIDParams{
		GithubID:  -1,
		Username:  "devuser",
		Email:     text("dev@example.com"),
		AvatarUrl: pgtype.Text{},
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "db_error", "could not create dev user")
		return
	}
	s.sessions.Issue(w, user.ID)
	httpx.JSON(w, http.StatusOK, meResponse{
		ID:       user.ID.String(),
		GithubID: user.GithubID,
		Username: user.Username,
		Email:    user.Email.String,
	})
}

func (s *Service) handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	if s.oauth == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "oauth_unconfigured", "GitHub OAuth is not configured")
		return
	}
	state, err := randomState()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal", "could not start login")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	http.Redirect(w, r, s.oauth.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (s *Service) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if s.oauth == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "oauth_unconfigured", "GitHub OAuth is not configured")
		return
	}

	q := r.URL.Query()
	cookie, err := r.Cookie(stateCookie)
	if err != nil || q.Get("state") == "" || q.Get("state") != cookie.Value {
		httpx.Error(w, http.StatusBadRequest, "bad_state", "invalid oauth state")
		return
	}
	// State consumed — clear it.
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Path: "/", MaxAge: -1})

	code := q.Get("code")
	if code == "" {
		httpx.Error(w, http.StatusBadRequest, "missing_code", "missing authorization code")
		return
	}

	gu, err := s.oauth.Exchange(r.Context(), code)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "oauth_failed", "GitHub exchange failed")
		return
	}

	user, err := s.queries.UpsertUserByGithubID(r.Context(), sqlc.UpsertUserByGithubIDParams{
		GithubID:  gu.ID,
		Username:  gu.Login,
		Email:     text(gu.Email),
		AvatarUrl: text(gu.AvatarURL),
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "db_error", "could not persist user")
		return
	}

	s.sessions.Issue(w, user.ID)
	http.Redirect(w, r, s.frontendURL, http.StatusTemporaryRedirect)
}

func (s *Service) handleLogout(w http.ResponseWriter, _ *http.Request) {
	s.sessions.Clear(w)
	w.WriteHeader(http.StatusNoContent)
}

type meResponse struct {
	ID        string `json:"id"`
	GithubID  int64  `json:"github_id"`
	Username  string `json:"username"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

func (s *Service) handleMe(w http.ResponseWriter, r *http.Request) {
	id, _ := UserIDFromContext(r.Context()) // guaranteed by RequireAuth
	user, err := s.queries.GetUserByID(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	httpx.JSON(w, http.StatusOK, meResponse{
		ID:        user.ID.String(),
		GithubID:  user.GithubID,
		Username:  user.Username,
		Email:     user.Email.String,
		AvatarURL: user.AvatarUrl.String,
	})
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func text(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

type missingEnvError string

func (e missingEnvError) Error() string { return "auth: missing required env " + string(e) }

func errMissing(name string) error { return missingEnvError(name) }
