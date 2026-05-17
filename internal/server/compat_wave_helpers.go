package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
)

func (s *Server) mapCompatEvents(events []*ent.Event, dispatchByTask map[string]string) []compatEventRecord {
	records := make([]compatEventRecord, 0, len(events))
	for _, e := range events {
		records = append(records, compatEventRecord{
			EventID:     e.EventID,
			ProjectID:   e.ProjectID,
			TaskID:      e.TaskID,
			DispatchRef: dispatchByTask[e.TaskID],
			EventType:   e.EventType,
			FromState:   e.FromState,
			ToState:     e.ToState,
			Timestamp:   e.Timestamp,
			Reason:      e.Reason,
			Attempt:     e.Attempt,
			Transport:   e.Transport,
			RunnerID:    e.RunnerID,
			Details:     e.Details,
		})
	}
	return records
}

func (s *Server) buildCompatWaveSummary(waveRow *ent.Wave, tasks []*ent.Task) compatWaveSummary {
	counts := make(map[string]int)
	for _, t := range tasks {
		counts[t.State]++
	}
	status := "open"
	if !waveRow.SealedAt.IsZero() {
		status = "sealed"
	}
	return compatWaveSummary{
		ProjectID:      waveRow.ProjectID,
		DispatchRef:    waveRow.DispatchRef,
		Wave:           waveRow.Wave,
		Status:         status,
		TaskCount:      len(tasks),
		CountsByStatus: counts,
		CreatedAt:      waveRow.CreatedAt,
	}
}

func parseCompatWaveParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "waveNum")
	if raw == "" {
		raw = r.URL.Query().Get("wave")
	}
	if raw == "" {
		raw = "1"
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}
