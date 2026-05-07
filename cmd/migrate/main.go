// Package main provides the v3 migration tool that moves Agent definitions
// from config.yaml into the SQLite database (agent table).
//
// Usage:
//
//	go run cmd/migrate/main.go --config config.yaml --db ai-orchestration.db
//
// What it does:
//
//  1. Reads all projects from config.yaml.
//  2. For each project, reads the claude/gemini/codex runtime blocks.
//  3. Creates (or upserts) an Agent DB record with:
//     - runner_type = "cli" (all current agents are CLI-based)
//     - runner_config JSON = {"base_path": "...", "main_repo": "..."}
//     - config JSON updated with specialties and original type
//  4. Prints a summary of migrated agents.
//  5. Backs up config.yaml to config.yaml.v3-migration-bak.
//
// The migration is idempotent: re-running it updates existing records but
// does not duplicate them. It is safe to run multiple times.
//
// Rollback: restore from config.yaml.v3-migration-bak and run the migration
// again to revert DB changes.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	entdialect "entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/agent"
)

type config struct {
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
	Projects struct {
		Items []projectConfig `yaml:"items"`
	} `yaml:"projects"`
}

type projectConfig struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	RepoRoot     string `yaml:"repo_root"`
	WorktreeBase string `yaml:"worktree_base"`
	WorkspaceBase string `yaml:"workspace_base"`
	ArtifactBase string `yaml:"artifact_base"`
	Claude       runtimeSpec `yaml:"claude"`
	Gemini       runtimeSpec `yaml:"gemini"`
	Codex        runtimeSpec `yaml:"codex"`
}

type runtimeSpec struct {
	Command string `yaml:"command"`
	Shell   string `yaml:"shell"`
}

type agentRecord struct {
	ID          string `json:"id"`
	OrgID       string `json:"org_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	RunnerType  string `json:"runner_type"`
	RunnerConfig string `json:"runner_config"`
	Config      string `json:"config"`
	Specialties string `json:"specialties"`
	Status      string `json:"status"`
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config.yaml")
	dbPath := flag.String("db", "ai-orchestration.db", "path to SQLite database")
	dryRun := flag.Bool("dry-run", false, "print what would be migrated without writing to DB")
	flag.Parse()

	// Load config
	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config.yaml: %v\n", err)
		os.Exit(1)
	}

	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse config.yaml: %v\n", err)
		os.Exit(1)
	}

	// Backup config
	backupPath := *configPath + ".v3-migration-bak"
	if !*dryRun {
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to create backup at %s: %v\n", backupPath, err)
		} else {
			fmt.Printf("Backed up config.yaml → %s\n", backupPath)
		}
	}

	// Open DB
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", *dbPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	drv := entsql.OpenDB(entdialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))

	ctx := context.Background()

	migrated := 0
	skipped := 0

	for _, proj := range cfg.Projects.Items {
		agents := []struct {
			agentName string
			agentType string
			spec      runtimeSpec
		}{
			{"claude", "claude", proj.Claude},
			{"gemini", "gemini", proj.Gemini},
			{"codex", "codex", proj.Codex},
		}

		for _, a := range agents {
			// Skip agents without a configured command
			if strings.TrimSpace(a.spec.Command) == "" {
				skipped++
				continue
			}

			agentID := fmt.Sprintf("%s-%s", proj.ID, a.agentName)

			runnerConfigJSON, _ := json.Marshal(map[string]interface{}{
				"base_path":  proj.WorktreeBase,
				"main_repo":  proj.RepoRoot,
				"command":    a.spec.Command,
				"shell":      a.spec.Shell,
			})

			configJSON, _ := json.Marshal(map[string]interface{}{
				"project_id": proj.ID,
				"project_name": proj.Name,
			})

			specialtiesJSON, _ := json.Marshal([]string{a.agentType, "feature", "bugfix", "code-review"})

			record := agentRecord{
				ID:           agentID,
				OrgID:        "default-org",
				Name:         fmt.Sprintf("%s-%s", proj.Name, a.agentName),
				Type:         a.agentType,
				RunnerType:   "cli",
				RunnerConfig: string(runnerConfigJSON),
				Config:       string(configJSON),
				Specialties:  string(specialtiesJSON),
				Status:       "idle",
			}

			if *dryRun {
				b, _ := json.MarshalIndent(record, "", "  ")
				fmt.Printf("[DRY-RUN] Would migrate:\n%s\n\n", b)
				migrated++
				continue
			}

			// Upsert: create if not exists, update if exists
			existing, err := client.Agent.Query().
				Where(agent.ID(record.ID)).
				Only(ctx)

			if err != nil && !ent.IsNotFound(err) {
				fmt.Fprintf(os.Stderr, "  ERROR querying agent %s: %v\n", record.ID, err)
				continue
			}

			if existing != nil {
				// Update existing: use dedicated runner_type/runner_config columns
				runnerConfig := map[string]interface{}{
					"base_path":   proj.WorktreeBase,
					"main_repo":   proj.RepoRoot,
					"command":     a.spec.Command,
					"shell":       a.spec.Shell,
					"project_id":  proj.ID,
					"project_name": proj.Name,
				}
				cfgJSON, _ := json.Marshal(runnerConfig)
				_, err = client.Agent.UpdateOne(existing).
					Where(agent.ID(record.ID)).
					SetRunnerType("cli").
					SetRunnerConfig(string(cfgJSON)).
					SetConfig(string(cfgJSON)).
					Save(ctx)
			} else {
				// Create new: use dedicated runner_type/runner_config columns
				runnerConfig := map[string]interface{}{
					"base_path":   proj.WorktreeBase,
					"main_repo":   proj.RepoRoot,
					"command":     a.spec.Command,
					"shell":       a.spec.Shell,
					"project_id":  proj.ID,
					"project_name": proj.Name,
				}
				cfgJSON, _ := json.Marshal(runnerConfig)
				_, err = client.Agent.Create().
					SetID(record.ID).
					SetOrgID(record.OrgID).
					SetName(record.Name).
					SetType(record.Type).
					SetRunnerType("cli").
					SetRunnerConfig(string(cfgJSON)).
					SetConfig(string(cfgJSON)).
					SetSpecialties(string(specialtiesJSON)).
					SetStatus(record.Status).
					Save(ctx)
			}

			if err != nil {
				fmt.Fprintf(os.Stderr, "  ERROR upserting agent %s: %v\n", record.ID, err)
				continue
			}

			fmt.Printf("  ✓ Migrated: %s (%s/%s)\n", record.ID, proj.Name, a.agentName)
			migrated++
		}
	}

	fmt.Printf("\nMigration complete: %d migrated, %d skipped (no command configured)\n", migrated, skipped)

	if *dryRun {
		fmt.Println("\n(Dry run — no changes written to DB)")
	} else {
		fmt.Printf("Config backed up to: %s\n", backupPath)
		fmt.Println("\nNext step: regenerate ent code and restart the server.")
		fmt.Println("  go generate ./ent")
	}
}