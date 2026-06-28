package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Suchethan021/conveyor/backend/internal/auth"
	"github.com/Suchethan021/conveyor/backend/internal/db/sqlc"
	"github.com/Suchethan021/conveyor/backend/internal/httpx"
)

func isTerminal(status string) bool {
	switch status {
	case "success", "failed", "cancelled":
		return true
	}
	return false
}

// stream sends build logs over Server-Sent Events: all existing lines first,
// then new lines as the worker writes them, closing with a "done" event once
// the job reaches a terminal state.
func (h *buildHandlers) stream(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "invalid job id")
		return
	}

	// Ownership check before opening the stream.
	if _, err := h.q.GetBuildJobForOwner(r.Context(), sqlc.GetBuildJobForOwnerParams{ID: jobID, OwnerID: userID}); err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "build job not found")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	rc := http.NewResponseController(w)

	ctx := r.Context()
	var lastID int64

	// emit flushes any new log lines and reports whether the job has finished.
	emit := func() (done bool, err error) {
		logs, err := h.q.GetBuildLogsAfterForOwner(ctx, sqlc.GetBuildLogsAfterForOwnerParams{
			JobID: jobID, OwnerID: userID, ID: lastID,
		})
		if err != nil {
			return false, err
		}
		for _, l := range logs {
			payload, _ := json.Marshal(toBuildLogResponse(l))
			fmt.Fprintf(w, "data: %s\n\n", payload)
			lastID = l.ID
		}
		job, err := h.q.GetBuildJobForOwner(ctx, sqlc.GetBuildJobForOwnerParams{ID: jobID, OwnerID: userID})
		if err != nil {
			return false, err
		}
		return isTerminal(job.Status), nil
	}

	finish := func() {
		fmt.Fprint(w, "event: done\ndata: {}\n\n")
		_ = rc.Flush()
	}

	done, err := emit()
	if err != nil {
		return
	}
	_ = rc.Flush()
	if done {
		finish()
		return
	}

	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return // client disconnected
		case <-ticker.C:
			done, err := emit()
			if err != nil {
				return
			}
			_ = rc.Flush()
			if done {
				finish()
				return
			}
		}
	}
}
