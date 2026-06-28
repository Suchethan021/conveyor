// Package worker runs build jobs asynchronously. It is a pool of goroutines
// that claim queued jobs from Postgres (FOR UPDATE SKIP LOCKED), move them
// through simulated build stages, write logs, and handle retries, cancellation,
// and graceful shutdown.
package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Suchethan021/conveyor/backend/internal/db/sqlc"
	"github.com/Suchethan021/conveyor/backend/internal/logsec"
)

// Worker is a pool of build-processing goroutines.
type Worker struct {
	q            *sqlc.Queries
	concurrency  int
	pollInterval time.Duration
}

func New(q *sqlc.Queries, concurrency int) *Worker {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Worker{q: q, concurrency: concurrency, pollInterval: time.Second}
}

// stage is one simulated step of a build.
type stage struct {
	status   string // job status while this stage runs
	label    string // human label for logs
	duration time.Duration
}

var stages = []stage{
	{status: "building", label: "build", duration: 2 * time.Second},
	{status: "scanning", label: "scan", duration: 1500 * time.Millisecond},
	{status: "deploying", label: "deploy", duration: 2 * time.Second},
}

// Run starts the worker pool and blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	log.Printf("worker pool starting with concurrency=%d", w.concurrency)
	done := make(chan struct{})
	for i := 0; i < w.concurrency; i++ {
		go func(n int) {
			w.loop(ctx, fmt.Sprintf("worker-%d", n))
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < w.concurrency; i++ {
		<-done
	}
	log.Println("worker pool stopped")
}

func (w *Worker) loop(ctx context.Context, workerID string) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			job, err := w.q.ClaimNextBuildJob(ctx, pgtype.Text{String: workerID, Valid: true})
			if errors.Is(err, pgx.ErrNoRows) {
				continue // nothing to do
			}
			if err != nil {
				if ctx.Err() == nil {
					log.Printf("%s: claim error: %v", workerID, err)
				}
				continue
			}
			w.process(ctx, workerID, job)
		}
	}
}

// process runs one claimed job through its stages.
func (w *Worker) process(ctx context.Context, workerID string, job sqlc.BuildJob) {
	w.appendLog(ctx, job.ID, "building", "info", fmt.Sprintf("%s picked up job (attempt %d)", workerID, job.RetryCount+1))

	// Demo log line proving secret masking is applied before storage.
	w.appendLog(ctx, job.ID, "building", "info", "authenticating to github.com with token=ghp_exampleSECRETvalue1234567890")

	shouldFail := w.simulatedFailure(ctx, job)

	for _, st := range stages {
		if w.isCancelled(ctx, job.ID) {
			w.appendLog(ctx, job.ID, st.status, "warn", "cancellation requested; stopping build")
			_ = w.q.MarkBuildJobCancelled(ctx, job.ID)
			return
		}

		if err := w.q.SetBuildJobStatus(ctx, sqlc.SetBuildJobStatusParams{ID: job.ID, Status: st.status}); err != nil {
			return // db gone or shutting down
		}
		w.appendLog(ctx, job.ID, st.status, "info", fmt.Sprintf("stage %q started", st.label))

		if !sleepCtx(ctx, st.duration) {
			return // shutdown mid-stage; job stays claimed for a future run
		}

		if shouldFail && st.status == "scanning" {
			w.fail(ctx, job, "vulnerability scan failed (simulated)")
			return
		}
		w.appendLog(ctx, job.ID, st.status, "info", fmt.Sprintf("stage %q completed", st.label))
	}

	_ = w.q.MarkBuildJobSuccess(ctx, job.ID)
	w.appendLog(ctx, job.ID, "deploying", "info", "build succeeded")
}

// fail either requeues for another attempt or marks the job failed.
func (w *Worker) fail(ctx context.Context, job sqlc.BuildJob, reason string) {
	if job.RetryCount < job.MaxRetries {
		w.appendLog(ctx, job.ID, "scanning", "warn", fmt.Sprintf("%s; requeueing for retry", reason))
		_ = w.q.RequeueBuildJob(ctx, job.ID)
		return
	}
	w.appendLog(ctx, job.ID, "scanning", "error", fmt.Sprintf("%s; no retries left", reason))
	_ = w.q.MarkBuildJobFailed(ctx, sqlc.MarkBuildJobFailedParams{
		ID:            job.ID,
		FailureReason: pgtype.Text{String: reason, Valid: true},
	})
}

// simulatedFailure makes failures deterministic for demos: a build fails if its
// project's branch name contains "fail". Everything else succeeds.
func (w *Worker) simulatedFailure(ctx context.Context, job sqlc.BuildJob) bool {
	project, err := w.q.GetProjectByID(ctx, job.ProjectID)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(project.Branch), "fail")
}

func (w *Worker) isCancelled(ctx context.Context, jobID uuid.UUID) bool {
	cancelled, err := w.q.IsCancelRequested(ctx, jobID)
	return err == nil && cancelled
}

func (w *Worker) appendLog(ctx context.Context, jobID uuid.UUID, stage, level, msg string) {
	err := w.q.AppendBuildLog(ctx, sqlc.AppendBuildLogParams{
		JobID:   jobID,
		Stage:   pgtype.Text{String: stage, Valid: true},
		Level:   level,
		Message: logsec.Mask(msg),
	})
	if err != nil && ctx.Err() == nil {
		log.Printf("append log failed for job %s: %v", jobID, err)
	}
}

// sleepCtx sleeps for d, returning false if ctx is cancelled first.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
