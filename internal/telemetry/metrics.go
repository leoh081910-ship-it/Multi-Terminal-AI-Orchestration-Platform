package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const Namespace = "ai_orchestrator"

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})

	TasksCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "tasks_created_total",
		Help:      "Total number of tasks created",
	}, []string{"project_id", "owner_agent", "type"})

	TasksCompleted = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "tasks_completed_total",
		Help:      "Total number of tasks completed",
	}, []string{"project_id", "owner_agent", "status"})

	TaskExecutionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Name:      "task_execution_duration_seconds",
		Help:      "Task execution duration in seconds",
		Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
	}, []string{"owner_agent"})

	ActiveWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "active_workers",
		Help:      "Number of currently active worker goroutines",
	})

	TasksInQueue = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "tasks_in_queue",
		Help:      "Number of tasks waiting in the execution queue",
	})

	WebSocketConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "websocket_connections",
		Help:      "Number of active WebSocket connections",
	})

	AgentRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "agent_requests_total",
		Help:      "Total number of agent execution requests",
	}, []string{"agent_id", "runner_type", "task_type", "status"})

	AgentDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Name:      "agent_duration_seconds",
		Help:      "Agent execution duration in seconds",
		Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
	}, []string{"agent_id", "runner_type"})
)
