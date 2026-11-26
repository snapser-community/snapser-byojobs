// A basic BYOJob worker for the Snapser Jobs snap. It logs every batch and
// acks it: a 200 with no results means every user in the batch is ok.
// Replace the body of the /batch handler with your own per-user work.
//
// Failure mode, for trying out retries and failures: set FAILURE_MODE=true in
// the job's worker environment. Each user then fails with probability
// FAILURE_RATE (default 0.1): half of those ask to be retried ("retry"),
// the rest fail for good ("failed").
//
// Worker contract (https://snapser.com/docs/infra/snaps/jobs#the-worker-contract):
//
//	POST /batch  {"job_id", "run_id", "batch_id", "attempt", "items": [...]}
//	GET  /health readiness and liveness probes
//
// Snapser sets PORT, SNAPSER_JOB_ID, SNAPSER_RUN_ID, SNAPEND_ID, and the
// SNAPEND_<SNAP>_<PORT>_URL snap URLs on every worker.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

type itemResult struct {
	ID     string `json:"id"`
	Status string `json:"status"` // ok, retry or failed
	Error  string `json:"error,omitempty"`
}

// failureRate is 0 unless FAILURE_MODE is true. FAILURE_RATE (0 to 1)
// overrides the default of 0.1.
func failureRate() (float64, error) {
	on, _ := strconv.ParseBool(os.Getenv("FAILURE_MODE"))
	if !on {
		return 0, nil
	}
	raw := os.Getenv("FAILURE_RATE")
	if raw == "" {
		return 0.1, nil
	}
	rate, err := strconv.ParseFloat(raw, 64)
	if err != nil || rate < 0 || rate > 1 {
		return 0, fmt.Errorf("FAILURE_RATE must be a number from 0 to 1, got %q", raw)
	}
	return rate, nil
}

// failures picks the items that fail in this batch. Items left out are ok.
func failures(items []string, rate float64, attempt int32) []itemResult {
	var out []itemResult
	for _, id := range items {
		if rand.Float64() >= rate {
			continue
		}
		if rand.IntN(2) == 0 {
			out = append(out, itemResult{ID: id, Status: "retry", Error: fmt.Sprintf("simulated transient failure (attempt %d)", attempt)})
		} else {
			out = append(out, itemResult{ID: id, Status: "failed", Error: "simulated permanent failure"})
		}
	}
	return out
}

type batchRequest struct {
	JobID   string   `json:"job_id"`
	RunID   string   `json:"run_id"`
	BatchID string   `json:"batch_id"`
	Attempt int32    `json:"attempt"`
	Items   []string `json:"items"`
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	rate, err := failureRate()
	if err != nil {
		log.Error("bad failure mode settings", "err", err)
		os.Exit(1)
	}

	var batches, items, failed atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /batch", func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
			log.Error("bad batch request", "err", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		results := failures(req.Items, rate, req.Attempt)
		b, n, f := batches.Add(1), items.Add(int64(len(req.Items))), failed.Add(int64(len(results)))
		log.Info("batch",
			"run_id", req.RunID,
			"batch_id", req.BatchID,
			"attempt", req.Attempt,
			"items", len(req.Items),
			"failed", len(results),
			"first", first(req.Items),
			"total_batches", b,
			"total_items", n,
			"total_failed", f,
		)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"results": results})
	})

	srv := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()

	log.Info("worker starting",
		"port", port,
		"job_id", os.Getenv("SNAPSER_JOB_ID"),
		"run_id", os.Getenv("SNAPSER_RUN_ID"),
		"snapend_id", os.Getenv("SNAPEND_ID"),
		"failure_rate", rate,
	)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server", "err", err)
		os.Exit(1)
	}
	log.Info("worker stopped", "total_batches", batches.Load(), "total_items", items.Load(), "total_failed", failed.Load())
}

func first(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}
