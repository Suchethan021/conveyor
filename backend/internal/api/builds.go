package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Suchethan021/conveyor/backend/internal/auth"
	"github.com/Suchethan021/conveyor/backend/internal/db/sqlc"
	"github.com/Suchethan021/conveyor/backend/internal/httpx"
)

// defaultMaxRetries is how many times a failed job is retried before giving up.
const defaultMaxRetries = 1

type buildHandlers struct {
	q *sqlc.Queries
}

type buildJobResponse struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"project_id"`
	Status          string     `json:"status"`
	RetryCount      int32      `json:"retry_count"`
	MaxRetries      int32      `json:"max_retries"`
	FailureReason   string     `json:"failure_reason,omitempty"`
	CancelRequested bool       `json:"cancel_requested"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func toBuildJobResponse(j sqlc.BuildJob) buildJobResponse {
	return buildJobResponse{
		ID:              j.ID.String(),
		ProjectID:       j.ProjectID.String(),
		Status:          j.Status,
		RetryCount:      j.RetryCount,
		MaxRetries:      j.MaxRetries,
		FailureReason:   j.FailureReason.String,
		CancelRequested: j.CancelRequested,
		StartedAt:       timePtr(j.StartedAt),
		FinishedAt:      timePtr(j.FinishedAt),
		CreatedAt:       j.CreatedAt.Time,
		UpdatedAt:       j.UpdatedAt.Time,
	}
}

type buildLogResponse struct {
	ID        int64     `json:"id"`
	Stage     string    `json:"stage,omitempty"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

func toBuildLogResponse(l sqlc.BuildLog) buildLogResponse {
	return buildLogResponse{
		ID:        l.ID,
		Stage:     l.Stage.String,
		Level:     l.Level,
		Message:   l.Message,
		CreatedAt: l.CreatedAt.Time,
	}
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}

// trigger enqueues a build job for a project the caller owns.
func (h *buildHandlers) trigger(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "invalid project id")
		return
	}

	// Ownership: only the project owner may build it.
	if _, err := h.q.GetProjectForOwner(r.Context(), sqlc.GetProjectForOwnerParams{ID: projectID, OwnerID: userID}); err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "project not found")
		return
	}

	job, err := h.q.CreateBuildJob(r.Context(), sqlc.CreateBuildJobParams{
		ProjectID:  projectID,
		CreatedBy:  &userID,
		MaxRetries: defaultMaxRetries,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "db_error", "could not create build job")
		return
	}
	httpx.JSON(w, http.StatusCreated, toBuildJobResponse(job))
}

func (h *buildHandlers) listForProject(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "invalid project id")
		return
	}

	jobs, err := h.q.ListBuildJobsForProject(r.Context(), sqlc.ListBuildJobsForProjectParams{ProjectID: projectID, OwnerID: userID})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "db_error", "could not list build jobs")
		return
	}

	out := make([]buildJobResponse, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, toBuildJobResponse(j))
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *buildHandlers) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "invalid job id")
		return
	}

	job, err := h.q.GetBuildJobForOwner(r.Context(), sqlc.GetBuildJobForOwnerParams{ID: jobID, OwnerID: userID})
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "build job not found")
		return
	}
	httpx.JSON(w, http.StatusOK, toBuildJobResponse(job))
}

func (h *buildHandlers) logs(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "invalid job id")
		return
	}

	logs, err := h.q.GetBuildLogsForOwner(r.Context(), sqlc.GetBuildLogsForOwnerParams{JobID: jobID, OwnerID: userID})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "db_error", "could not fetch logs")
		return
	}

	out := make([]buildLogResponse, 0, len(logs))
	for _, l := range logs {
		out = append(out, toBuildLogResponse(l))
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *buildHandlers) cancel(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "invalid job id")
		return
	}

	rows, err := h.q.RequestCancelForOwner(r.Context(), sqlc.RequestCancelForOwnerParams{ID: jobID, OwnerID: userID})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "db_error", "could not cancel job")
		return
	}
	if rows == 0 {
		httpx.Error(w, http.StatusConflict, "not_cancellable", "job not found or not in a cancellable state")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
