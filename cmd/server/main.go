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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	_ "modernc.org/sqlite"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/migrate"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/server"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
)

type projectRuntimeCommandConfig struct {
	Command string `mapstructure:"command"`
	Shell   string `mapstructure:"shell"`
}

const defaultProjectID = "default"

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

	if err := viper.ReadInConfig(); err != nil {
		log.Warn().Err(err).Msg("config file not found, using defaults")
	}

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
	if err := client.Schema.Create(
		ctx,
		migrate.WithGlobalUniqueID(true),
	); err != nil {
		log.Fatal().Err(err).Msg("failed to create database schema")
	}
	log.Info().Msg("database schema initialized")

	// Initialize repository and server
	repo := store.NewRepository(client, &log.Logger)
	srv := server.New(repo, log.Logger)
	srv.SetWebDistDir(viper.GetString("web.dist_dir"))
	projectsConfig := loadCompatProjectsConfig()
	if err := srv.ConfigureCompatProjects(projectsConfig); err != nil {
		log.Fatal().Err(err).Msg("failed to configure projects")
	}
	srv.SetProjectConfigStore(server.NewProjectConfigStore(*configPath))
	queueCtx, queueCancel := context.WithCancel(context.Background())
	defer queueCancel()

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
