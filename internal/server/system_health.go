package server

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// SystemHealth represents the health status of the platform backend.
type SystemHealth struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
	Version   string    `json:"version"`
}

// WorkerStatus represents the running state of a background worker.
type WorkerStatus struct {
	Name     string `json:"name"`
	Running  bool   `json:"running"`
	Interval string `json:"interval,omitempty"`
}

// SystemWorkersResponse is the API response for worker status.
type SystemWorkersResponse struct {
	Workers []WorkerStatus `json:"workers"`
}

// systemHealthState tracks server start time for uptime calculation.
var serverStartTime time.Time

func init() {
	serverStartTime = time.Now().UTC()
}

// handleSystemHealth returns the health status of the backend.
// GET /api/v1/system/health
func (s *Server) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(serverStartTime)
	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: SystemHealth{
			Status:    "ok",
			Timestamp: time.Now().UTC(),
			Uptime:    uptime.Truncate(time.Second).String(),
			Version:   "v2.1",
		},
	})
}

// handleSystemWorkers returns the status of all background workers.
// GET /api/v1/system/workers
func (s *Server) handleSystemWorkers(w http.ResponseWriter, r *http.Request) {
	workers := []WorkerStatus{
		{Name: "auto_dispatcher", Running: s.autoDispatcherRunning()},
		{Name: "failure_orchestrator", Running: s.failureOrchestrator != nil && s.failureOrchestrator.running},
		{Name: "retry_worker", Running: s.retryWorker != nil && s.retryWorker.running},
		{Name: "review_worker", Running: s.reviewWorker != nil && s.reviewWorker.running},
		{Name: "ttl_cleanup", Running: s.ttlCleanupRunning()},
		{Name: "execution_reaper", Running: s.executionReaperRunning()},
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    SystemWorkersResponse{Workers: workers},
	})
}

// Worker status helpers — these check nil safely and report running state.
// The workers are set on Server after initialization, so we guard with nil checks.

func (s *Server) autoDispatcherRunning() bool {
	// autoDispatcher is not stored on Server directly; check via the field
	// if it were. For now, report based on whether the dispatcher was started.
	// We store a flag on the server to track this.
	return s.autoDispatcherActive
}

func (s *Server) ttlCleanupRunning() bool {
	return s.ttlCleanupActive
}

func (s *Server) executionReaperRunning() bool {
	return s.executionReaperActive
}

// SetAutoDispatcherActive marks the auto dispatcher as running.
func (s *Server) SetAutoDispatcherActive(active bool) {
	s.autoDispatcherActive = active
}

// SetTTLCleanupActive marks the TTL cleanup runner as running.
func (s *Server) SetTTLCleanupActive(active bool) {
	s.ttlCleanupActive = active
}

// SetExecutionReaperActive marks the execution reaper as running.
func (s *Server) SetExecutionReaperActive(active bool) {
	s.executionReaperActive = active
}

// LogSystemHealthStartup logs the system health endpoints availability.
func LogSystemHealthStartup(logger zerolog.Logger) {
	logger.Info().Msg("system health endpoints: /api/v1/system/health, /api/v1/system/workers")
}
