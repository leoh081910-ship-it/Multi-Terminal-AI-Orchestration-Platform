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
		if t.DispatchRef != waveRow.DispatchRef || t.Wave != waveRow.Wave {
			continue
		}
		counts[t.State]++
	}
	status := "open"
	if !waveRow.SealedAt.IsZero() {
		status = "sealed"
	}
	summary := compatWaveSummary{
		ProjectID:      waveRow.ProjectID,
		DispatchRef:    waveRow.DispatchRef,
		Wave:           waveRow.Wave,
		Status:         status,
		TaskCount:      sumCompatWaveCounts(counts),
		CountsByStatus: counts,
		CreatedAt:      waveRow.CreatedAt,
	}
	if !waveRow.SealedAt.IsZero() {
		summary.SealedAt = &waveRow.SealedAt
	}
	return summary
}

func sumCompatWaveCounts(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func parseCompatWaveParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "wave")
	if raw == "" {
		raw = chi.URLParam(r, "waveNum")
	}
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
