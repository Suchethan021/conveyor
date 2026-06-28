package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Suchethan021/conveyor/backend/internal/auth"
	"github.com/Suchethan021/conveyor/backend/internal/db/sqlc"
	"github.com/Suchethan021/conveyor/backend/internal/httpx"
)

type projectHandlers struct {
	q *sqlc.Queries
}

var (
	validProviders = map[string]bool{"github": true, "gitlab": true}
	validRuntimes  = map[string]bool{"go": true, "node": true, "python": true, "static": true}
	validEnvs      = map[string]bool{"dev": true, "staging": true, "prod": true}
	providerHosts  = map[string]string{"github": "github.com", "gitlab": "gitlab.com"}
)

type createProjectRequest struct {
	Name        string `json:"name"`
	GitProvider string `json:"git_provider"`
	RepoURL     string `json:"repo_url"`
	Branch      string `json:"branch"`
	Runtime     string `json:"runtime"`
	Environment string `json:"environment"`
}

type projectResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	GitProvider string    `json:"git_provider"`
	RepoURL     string    `json:"repo_url"`
	Branch      string    `json:"branch"`
	Runtime     string    `json:"runtime"`
	Environment string    `json:"environment"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toProjectResponse(p sqlc.Project) projectResponse {
	return projectResponse{
		ID:          p.ID.String(),
		Name:        p.Name,
		GitProvider: p.GitProvider,
		RepoURL:     p.RepoUrl,
		Branch:      p.Branch,
		Runtime:     p.Runtime,
		Environment: p.Environment,
		CreatedAt:   p.CreatedAt.Time,
		UpdatedAt:   p.UpdatedAt.Time,
	}
}

func (h *projectHandlers) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	var req createProjectRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Branch = strings.TrimSpace(req.Branch)
	req.RepoURL = strings.TrimSpace(req.RepoURL)

	if msg := validateCreateProject(req); msg != "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", msg)
		return
	}

	project, err := h.q.CreateProject(r.Context(), sqlc.CreateProjectParams{
		OwnerID:     userID,
		Name:        req.Name,
		GitProvider: req.GitProvider,
		RepoUrl:     req.RepoURL,
		Branch:      req.Branch,
		Runtime:     req.Runtime,
		Environment: req.Environment,
		CreatedBy:   &userID,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "db_error", "could not create project")
		return
	}
	httpx.JSON(w, http.StatusCreated, toProjectResponse(project))
}

func (h *projectHandlers) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	projects, err := h.q.ListProjectsByOwner(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "db_error", "could not list projects")
		return
	}

	out := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		out = append(out, toProjectResponse(p))
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *projectHandlers) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "invalid project id")
		return
	}

	project, err := h.q.GetProjectForOwner(r.Context(), sqlc.GetProjectForOwnerParams{ID: id, OwnerID: userID})
	if err != nil {
		// Not found or owned by someone else — same 404 either way, so we
		// never reveal that another user's project exists.
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}
	httpx.JSON(w, http.StatusOK, toProjectResponse(project))
}

func validateCreateProject(req createProjectRequest) string {
	if req.Name == "" || len(req.Name) > 100 {
		return "name is required and must be at most 100 characters"
	}
	if !validProviders[req.GitProvider] {
		return "git_provider must be one of: github, gitlab"
	}
	if !validRuntimes[req.Runtime] {
		return "runtime must be one of: go, node, python, static"
	}
	if !validEnvs[req.Environment] {
		return "environment must be one of: dev, staging, prod"
	}
	if req.Branch == "" || len(req.Branch) > 255 {
		return "branch is required and must be at most 255 characters"
	}
	if err := validateRepoURL(req.RepoURL, req.GitProvider); err != nil {
		return err.Error()
	}
	return ""
}

// validateRepoURL enforces an https URL on the provider's host with at least
// an owner/repository path.
func validateRepoURL(raw, provider string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errors.New("repo_url must be a valid https URL")
	}
	expected := providerHosts[provider]
	if strings.ToLower(u.Host) != expected {
		return errors.New("repo_url host must be " + expected + " for provider " + provider)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return errors.New("repo_url must include owner/repository")
	}
	return nil
}
