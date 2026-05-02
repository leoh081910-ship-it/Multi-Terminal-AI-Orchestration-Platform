package server

import (
	"context"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/org"
	"github.com/rs/zerolog"
)

const (
	DefaultOrgName   = "Default Organization"
	DefaultOrgDesc   = "Auto-created default organization for backward compatibility"
	DefaultOrgID     = "default"
)

// Bootstrap ensures a default organization exists and registers the legacy
// agents (Claude, Gemini, Codex) if no organizations exist yet.
// This provides backward compatibility with the pre-Phase-1 system.
func Bootstrap(ctx context.Context, orgSvc *org.Service, logger zerolog.Logger) error {
	orgs, err := orgSvc.ListOrgs(ctx)
	if err != nil {
		return err
	}

	if len(orgs) > 0 {
		logger.Info().Int("count", len(orgs)).Msg("organizations exist, skipping bootstrap")
		return nil
	}

	logger.Info().Msg("no organizations found, bootstrapping default org")

	// Create default organization
	orgView, err := orgSvc.CreateOrg(ctx, org.CreateOrgInput{
		Name:        DefaultOrgName,
		Description: DefaultOrgDesc,
	})
	if err != nil {
		return err
	}
	logger.Info().Str("org_id", orgView.ID).Msg("created default organization")

	// Register legacy agents
	legacyAgents := []struct {
		name        string
		agentType   string
		specialties []string
	}{
		{"Claude", "claude", []string{"code", "analysis", "review", "general"}},
		{"Gemini", "gemini", []string{"code", "analysis", "research", "general"}},
		{"Codex", "codex", []string{"code", "refactoring", "general"}},
	}

	for _, la := range legacyAgents {
		agentView, err := orgSvc.CreateAgent(ctx, orgView.ID, org.CreateAgentInput{
			Name:        la.name,
			Type:        la.agentType,
			Specialties: la.specialties,
		})
		if err != nil {
			logger.Error().Err(err).Str("name", la.name).Msg("failed to register legacy agent")
			continue
		}
		logger.Info().
			Str("agent_id", agentView.ID).
			Str("name", agentView.Name).
			Str("type", agentView.Type).
			Msg("registered legacy agent")
	}

	return nil
}

// DefaultOrgID returns the ID of the default organization.
// Used to auto-assign tasks that don't specify an org_id.
func GetDefaultOrgID(ctx context.Context, orgSvc *org.Service) string {
	orgs, err := orgSvc.ListOrgs(ctx)
	if err != nil || len(orgs) == 0 {
		return ""
	}
	return orgs[0].ID
}
