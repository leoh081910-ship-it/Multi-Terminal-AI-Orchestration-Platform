// Package main provides the server entry point for the AI orchestration platform.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	entdialect "entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	_ "modernc.org/sqlite"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/migrate"
	_ "github.com/mCP-DevOS/ai-orchestration-platform/docs"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/auth"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/backup"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/org"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/router"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/server"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/telemetry"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
)

type projectRuntimeCommandConfig struct {
	Command string `mapstructure:"command"`
	Shell   string `mapstructure:"shell"`
}

const defaultProjectID = "default"

// @title AI Orchestration Platform API
// @version 2.1
// @description Multi-terminal AI orchestration platform for task scheduling, agent management, and intelligent routing.
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

type projectConfigFile struct {
	ID            string                      `mapstructure:"id"`
	Name          string                      `mapstructure:"name"`
	RepoRoot      string                      `mapstructure:"repo_root"`
	WorktreeBase  string                      `mapstructure:"worktree_base"`
	WorkspaceBase string                      `mapstructure:"workspace_base"`
	ArtifactBase  string                      `mapstructure:"artifact_base"`
	Claude        projectRuntimeCommandConfig `mapstructure:"claude"`
	Gemini        projectRuntimeCommandConfig `mapstructure:"gemini"`
	Codex         projectRuntimeCommandConfig `mapstructure:"codex"`
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load config
	viper.SetConfigFile(*configPath)
	viper.AutomaticEnv()
	viper.SetEnvPrefix("AIOP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Defaults
	viper.SetDefault("database.path", "ai-orchestration.db")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("runtime.main_repo", ".")
	viper.SetDefault("runtime.worktree_base", ".orchestrator/worktrees")
	viper.SetDefault("runtime.workspace_base", ".orchestrator/workspaces")
	viper.SetDefault("runtime.artifact_base", ".orchestrator/artifacts")
	viper.SetDefault("projects.default", defaultProjectID)
	viper.SetDefault("auto_dispatch.enabled", true)
	viper.SetDefault("auto_dispatch.interval_ms", 2000)
	viper.SetDefault("ttl_cleanup.enabled", true)
	viper.SetDefault("ttl_cleanup.interval_ms", 60000)
	viper.SetDefault("web.dist_dir", "web/dist")
	viper.SetDefault("routing.strategy", "weighted")
	viper.SetDefault("routing.fallback", "manual")
	viper.SetDefault("routing.weights.capability", 0.5)
	viper.SetDefault("routing.weights.load", 0.3)
	viper.SetDefault("routing.weights.affinity", 0.2)
	viper.SetDefault("backup.enabled", true)
	viper.SetDefault("backup.dir", "backups")
	viper.SetDefault("backup.max_keep", 24)
	viper.SetDefault("backup.interval_minutes", 60)
	viper.SetDefault("sentry.dsn", "")
	viper.SetDefault("sentry.environment", "development")
	viper.SetDefault("sentry.traces_sample_rate", 0.1)

	if err := viper.ReadInConfig(); err != nil {
		log.Warn().Err(err).Msg("config file not found, using defaults")
	}

	// Initialize Sentry (no-op if DSN is empty)
	if err := telemetry.InitSentry(telemetry.SentryConfig{
		DSN:              viper.GetString("sentry.dsn"),
		Environment:      viper.GetString("sentry.environment"),
		Release:          "ai-orchestration-platform@v2.1",
		TracesSampleRate: viper.GetFloat64("sentry.traces_sample_rate"),
	}, log.Logger); err != nil {
		log.Warn().Err(err).Msg("sentry initialization failed (non-fatal)")
	}
	defer telemetry.FlushSentry()

	dbPath := viper.GetString("database.path")
	host := viper.GetString("server.host")
	port := viper.GetInt("server.port")

	// Open SQLite with WAL mode
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath))
	if err != nil {
		log.Fatal().Err(err).Str("path", dbPath).Msg("failed to open database")
	}

	drv := entsql.OpenDB(entdialect.SQLite, db)

	// Configure connection pool for SQLite
	db.SetMaxOpenConns(1) // SQLite single-writer constraint
	db.SetMaxIdleConns(1)

	client := ent.NewClient(ent.Driver(drv))
	defer func() {
		if err := client.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close database client")
		}
	}()

	// Run auto-migration
	ctx := context.Background()
	db.Exec("CREATE TABLE IF NOT EXISTS sqlite_sequence (name TEXT, seq INTEGER)")
	if err := client.Schema.Create(
		ctx,
		migrate.WithGlobalUniqueID(true),
	); err != nil {
		log.Fatal().Err(err).Msg("failed to create database schema")
	}
	log.Info().Msg("database schema initialized")

	// Initialize repository and server
	repo := store.NewRepository(client, db, &log.Logger)
	srv := server.New(repo, log.Logger)
	srv.SetWebDistDir(viper.GetString("web.dist_dir"))

	// Enable Sentry middleware if DSN is configured
	if viper.GetString("sentry.dsn") != "" {
		srv.EnableSentryMiddleware()
		log.Info().Msg("Sentry middleware enabled")
	}

	// Initialize authentication
	tokenRepo := auth.NewEntTokenRepository(client)
	tokenService := auth.NewTokenService(tokenRepo)
	authMiddleware := auth.NewMiddleware(tokenService)
	tokenHandler := server.NewTokenHandler(tokenService)
	log.Info().Msg("Authentication system initialized")

	// Wire WebSocket authentication
	srv.SetWSAuth(func(token string) (string, bool) {
		t, err := tokenService.ValidateToken(context.Background(), token)
		if err != nil {
			return "", false
		}
		return t.UserID, true
	})

	// Initialize RBAC permission system
	permService := auth.NewPermissionService(client)
	rbacMiddleware := auth.NewRBACMiddleware(client, authMiddleware)
	if err := permService.SeedDefaultRoles(ctx); err != nil {
		log.Error().Err(err).Msg("failed to seed default roles (non-fatal)")
	} else {
		log.Info().Msg("RBAC system initialized with default roles and permissions")
	}
	_ = rbacMiddleware // Available for future endpoint protection

	// Phase 5: Initialize intelligent router
	routingCfg := router.Config{
		Strategy: router.Strategy(viper.GetString("routing.strategy")),
		Fallback: router.Strategy(viper.GetString("routing.fallback")),
		Weights: router.Weights{
			Capability: viper.GetFloat64("routing.weights.capability"),
			Load:       viper.GetFloat64("routing.weights.load"),
			Affinity:   viper.GetFloat64("routing.weights.affinity"),
		},
	}
	orgSvc := org.NewService(client)
	taskRouter := router.NewRouter(routingCfg, orgSvc, client, log.Logger)
	srv.SetTaskRouter(taskRouter)

	// Phase 6: Bootstrap default org and legacy agents for backward compatibility
	if err := server.Bootstrap(context.Background(), orgSvc, log.Logger); err != nil {
		log.Error().Err(err).Msg("bootstrap failed (non-fatal)")
	}
	projectsConfig := loadCompatProjectsConfig()
	if err := srv.ConfigureCompatProjects(projectsConfig); err != nil {
		log.Fatal().Err(err).Msg("failed to configure projects")
	}

	// Register authentication routes
	srv.RegisterAuthRoutes(srv.Handler().(*chi.Mux), tokenHandler, authMiddleware)
	log.Info().Msg("Authentication routes registered")

	// Initialize backup service and scheduler
	var backupScheduler *backup.Scheduler
	if viper.GetBool("backup.enabled") {
		backupSvc, err := backup.NewService(backup.Config{
			DBPath:    dbPath,
			BackupDir: viper.GetString("backup.dir"),
			MaxKeep:   viper.GetInt("backup.max_keep"),
		}, log.Logger)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to initialize backup service")
		}
		backupHandler := server.NewBackupHandler(backupSvc)
		srv.RegisterBackupRoutes(srv.Handler().(*chi.Mux), backupHandler, authMiddleware)
		backupScheduler = backup.NewScheduler(
			backupSvc,
			time.Duration(viper.GetInt("backup.interval_minutes"))*time.Minute,
			log.Logger,
		)
		log.Info().
			Str("backup_dir", viper.GetString("backup.dir")).
			Int("max_keep", viper.GetInt("backup.max_keep")).
			Int("interval_minutes", viper.GetInt("backup.interval_minutes")).
			Msg("backup service initialized")
	}

	srv.SetProjectConfigStore(server.NewProjectConfigStore(*configPath))
	queueCtx, queueCancel := context.WithCancel(context.Background())
	defer queueCancel()

	if backupScheduler != nil {
		if err := backupScheduler.Start(queueCtx); err != nil {
			log.Error().Err(err).Msg("failed to start backup scheduler (non-fatal)")
		}
		defer func() {
			if err := backupScheduler.Stop(); err != nil {
				log.Error().Err(err).Msg("failed to stop backup scheduler")
			}
		}()
	}

	autoDispatcher := server.NewAutoDispatcher(srv, log.Logger, server.AutoDispatchConfig{
		Interval: time.Duration(viper.GetInt("auto_dispatch.interval_ms")) * time.Millisecond,
	})
	if viper.GetBool("auto_dispatch.enabled") {
		if err := autoDispatcher.Start(queueCtx); err != nil {
			log.Fatal().Err(err).Msg("failed to start auto dispatcher")
		}
		srv.SetAutoDispatcherActive(true)
		defer func() {
			if err := autoDispatcher.Stop(); err != nil {
				log.Error().Err(err).Msg("failed to stop auto dispatcher")
			}
		}()
	}

	ttlCleanup := server.NewTTLCleanupRunner(repo, log.Logger, server.TTLCleanupConfig{
		Interval: time.Duration(viper.GetInt("ttl_cleanup.interval_ms")) * time.Millisecond,
	})
	if viper.GetBool("ttl_cleanup.enabled") {
		if err := ttlCleanup.Start(queueCtx); err != nil {
			log.Fatal().Err(err).Msg("failed to start ttl cleanup runner")
		}
		srv.SetTTLCleanupActive(true)
		defer func() {
			if err := ttlCleanup.Stop(); err != nil {
				log.Error().Err(err).Msg("failed to stop ttl cleanup runner")
			}
		}()
	}

	queueManager := server.NewProjectQueueManager(repo, log.Logger, projectsConfig.DefaultProjectID)
	if err := queueManager.Start(queueCtx, projectsConfig.Projects); err != nil {
		log.Fatal().Err(err).Msg("failed to start project queue manager")
	}
	srv.SetProjectQueueManager(queueManager)
	defer func() {
		if err := queueManager.Stop(); err != nil {
			log.Error().Err(err).Msg("failed to stop project queue manager")
		}
	}()

	// PRD-DA-001: Start coordination workers
	failureOrchestrator := server.NewFailureOrchestrator(srv, repo, log.Logger, 5*time.Second)
	retryWorker := server.NewRetryWorker(srv, repo, log.Logger, 5*time.Second)
	reviewWorker := server.NewReviewWorker(srv, repo, log.Logger, 5*time.Second)
	srv.SetCoordinationWorkers(failureOrchestrator, retryWorker, reviewWorker)

	if err := failureOrchestrator.Start(queueCtx); err != nil {
		log.Fatal().Err(err).Msg("failed to start failure orchestrator")
	}
	defer func() {
		if err := failureOrchestrator.Stop(); err != nil {
			log.Error().Err(err).Msg("failed to stop failure orchestrator")
		}
	}()

	if err := retryWorker.Start(queueCtx); err != nil {
		log.Fatal().Err(err).Msg("failed to start retry worker")
	}
	defer func() {
		if err := retryWorker.Stop(); err != nil {
			log.Error().Err(err).Msg("failed to stop retry worker")
		}
	}()

	if err := reviewWorker.Start(queueCtx); err != nil {
		log.Fatal().Err(err).Msg("failed to start review worker")
	}
	defer func() {
		if err := reviewWorker.Stop(); err != nil {
			log.Error().Err(err).Msg("failed to stop review worker")
		}
	}()

	// PR-OPS-002: Startup recovery — reclaim zombie tasks left from previous run
	startupRecovery := server.NewStartupRecovery(srv, repo, log.Logger)
	if err := startupRecovery.Run(queueCtx); err != nil {
		log.Error().Err(err).Msg("startup recovery failed (non-fatal, workers will continue)")
	}

	// PR-OPS-002: Execution reaper — periodic scan for zombie running tasks
	executionReaper := server.NewExecutionReaper(srv, repo, log.Logger, server.ExecutionReaperConfig{
		Interval: 15 * time.Second,
	})
	if err := executionReaper.Start(queueCtx); err != nil {
		log.Fatal().Err(err).Msg("failed to start execution reaper")
	}
	srv.SetExecutionReaperActive(true)
	defer func() {
		if err := executionReaper.Stop(); err != nil {
			log.Error().Err(err).Msg("failed to stop execution reaper")
		}
	}()

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", host, port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Info().Str("signal", sig.String()).Msg("shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		queueCancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("server shutdown error")
		}
	}()

	log.Info().Str("addr", addr).Msg("server starting")
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server failed")
	}

	log.Info().Msg("server stopped")
}

func loadCompatProjectsConfig() server.CompatProjectsConfig {
	var items []projectConfigFile
	_ = viper.UnmarshalKey("projects.items", &items)

	if len(items) == 0 {
		return server.CompatProjectsConfig{
			DefaultProjectID: defaultProjectID,
			Projects: []server.CompatProjectConfig{
				{
					ID:                defaultProjectID,
					Name:              defaultProjectID,
					MainRepoPath:      viper.GetString("runtime.main_repo"),
					WorktreeBasePath:  viper.GetString("runtime.worktree_base"),
					WorkspaceBasePath: viper.GetString("runtime.workspace_base"),
					ArtifactBasePath:  viper.GetString("runtime.artifact_base"),
					ClaudeCommand:     viper.GetString("runtime.claude.command"),
					ClaudeShell:       viper.GetString("runtime.claude.shell"),
					GeminiCommand:     viper.GetString("runtime.gemini.command"),
					GeminiShell:       viper.GetString("runtime.gemini.shell"),
					CodexCommand:      viper.GetString("runtime.codex.command"),
					CodexShell:        viper.GetString("runtime.codex.shell"),
				},
			},
		}
	}

	projects := make([]server.CompatProjectConfig, 0, len(items))
	for _, item := range items {
		projects = append(projects, server.CompatProjectConfig{
			ID:                item.ID,
			Name:              item.Name,
			MainRepoPath:      item.RepoRoot,
			WorktreeBasePath:  item.WorktreeBase,
			WorkspaceBasePath: item.WorkspaceBase,
			ArtifactBasePath:  item.ArtifactBase,
			ClaudeCommand:     item.Claude.Command,
			ClaudeShell:       item.Claude.Shell,
			GeminiCommand:     item.Gemini.Command,
			GeminiShell:       item.Gemini.Shell,
			CodexCommand:      item.Codex.Command,
			CodexShell:        item.Codex.Shell,
		})
	}

	return server.CompatProjectsConfig{
		DefaultProjectID: viper.GetString("projects.default"),
		Projects:         projects,
	}
}
