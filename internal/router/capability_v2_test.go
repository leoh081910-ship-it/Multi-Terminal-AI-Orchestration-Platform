package router

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/registry"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/runner"
	"github.com/rs/zerolog"
)

type manifestRunner struct {
	manifest *runner.CapabilityManifest
	err      error
}

func (m manifestRunner) Execute(ctx context.Context, task runner.RunnerTask) (*runner.RunnerResult, error) {
	return &runner.RunnerResult{Success: true, ExitCode: 0}, nil
}

func (m manifestRunner) HealthCheck(ctx context.Context) error { return nil }
func (m manifestRunner) Cancel(ctx context.Context, taskID string) error { return nil }

func (m manifestRunner) GetCapabilities(ctx context.Context) (*runner.CapabilityManifest, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.manifest, nil
}

func (m manifestRunner) Type() runner.RunnerType { return runner.RunnerTypeCLI }
func (m manifestRunner) String() string { return "manifest-runner" }

func newManifestMatcher(manifests map[string]*runner.CapabilityManifest) *EnhancedCapabilityMatcher {
	reg := registry.New(zerolog.New(nil), nil)
	for id, manifest := range manifests {
		reg.Register(id, manifestRunner{manifest: manifest})
	}
	return NewEnhancedCapabilityMatcher(reg)
}

func TestEnhancedCapabilityMatcherScoreHelpers(t *testing.T) {
	matcher := NewEnhancedCapabilityMatcher(nil)

	manifest := &runner.CapabilityManifest{
		TaskTypes:          []string{"feature", "code-review", "reverse"},
		ContextWindow:      200000,
		SupportsThinking:   true,
		SupportsMultimodal: true,
		SupportsStreaming:  true,
		LatencyP50:         300,
		CostPer1KInput:     0.1,
		CostPer1KOutput:    0.2,
		Runtime:            "cli",
	}

	assertClose(t, matcher.scoreTaskType(manifest, "feature"), 1.0)
	assertClose(t, matcher.scoreTaskType(&runner.CapabilityManifest{TaskTypes: []string{"bugfix"}}, "debug"), 0.8)
	assertClose(t, matcher.scoreTaskType(&runner.CapabilityManifest{TaskTypes: []string{"documentation"}}, "docs"), 0.8)
	assertClose(t, matcher.scoreTaskType(&runner.CapabilityManifest{TaskTypes: []string{"static-analysis"}}, "analysis"), 0.6)
	assertClose(t, matcher.scoreTaskType(&runner.CapabilityManifest{}, "unknown"), 0.4)
	assertClose(t, matcher.scoreTaskType(&runner.CapabilityManifest{TaskTypes: []string{"translation"}}, "reverse"), 0.4)

	assertClose(t, matcher.scoreContextWindow(manifest, nil), 1.0)
	assertClose(t, matcher.scoreContextWindow(&runner.CapabilityManifest{ContextWindow: 100000}, nil), 0.8)
	assertClose(t, matcher.scoreContextWindow(&runner.CapabilityManifest{ContextWindow: 32000}, nil), 0.6)
	assertClose(t, matcher.scoreContextWindow(&runner.CapabilityManifest{ContextWindow: 8000}, nil), 0.4)
	assertClose(t, matcher.scoreContextWindow(&runner.CapabilityManifest{ContextWindow: 4000}, nil), 0.2)
	assertClose(t, matcher.scoreContextWindow(&runner.CapabilityManifest{}, nil), 0.5)

	assertClose(t, matcher.scoreFeatures(manifest, []string{"reasoning", "image", "streaming"}), 1.0/3.0)
	assertClose(t, matcher.scoreFeatures(manifest, nil), 0.7)
	assertClose(t, matcher.scoreFeatures(&runner.CapabilityManifest{}, nil), 0.3)

	assertClose(t, matcher.scoreEfficiency(manifest), 1.0)
	assertClose(t, matcher.scoreEfficiency(&runner.CapabilityManifest{LatencyP50: 3000, CostPer1KInput: 30}), 0.4)
	assertClose(t, matcher.scoreEfficiency(nil), 0.5)
}

func TestEnhancedCapabilityMatcherScoreV2(t *testing.T) {
	ctx := context.Background()
	matcher := newManifestMatcher(map[string]*runner.CapabilityManifest{
		"strong": {
			TaskTypes:          []string{"feature"},
			ContextWindow:      200000,
			SupportsThinking:   true,
			SupportsMultimodal: true,
			SupportsStreaming:  true,
			LatencyP50:         250,
			CostPer1KInput:     0.1,
			CostPer1KOutput:    0.1,
			Runtime:            "cli",
		},
		"weak": {
			TaskTypes:       []string{"translation"},
			ContextWindow:   8000,
			LatencyP50:      6000,
			CostPer1KInput:  30,
			CostPer1KOutput: 30,
			Runtime:         "http",
		},
	})

	strong := matcher.ScoreV2(ctx, "strong", "feature", []string{"reasoning", "image", "streaming"})
	weak := matcher.ScoreV2(ctx, "weak", "feature", []string{"reasoning", "image", "streaming"})

	if strong <= weak {
		t.Fatalf("expected strong manifest score %v to beat weak score %v", strong, weak)
	}
	assertClose(t, matcher.ScoreV2(ctx, "strong", "", nil), 0.5)
	assertClose(t, matcher.ScoreV2(ctx, "missing", "feature", nil), 0.1)
}

func TestEnhancedCapabilityMatcherBatchScore(t *testing.T) {
	ctx := context.Background()
	matcher := newManifestMatcher(map[string]*runner.CapabilityManifest{
		"feature-agent": {TaskTypes: []string{"feature"}, ContextWindow: 200000, Runtime: "cli"},
		"docs-agent":    {TaskTypes: []string{"documentation"}, ContextWindow: 8000},
	})

	scores := matcher.BatchScore(ctx, []string{"feature-agent", "docs-agent", "missing"}, "feature", nil)

	if len(scores) != 3 {
		t.Fatalf("BatchScore returned %d scores, want 3", len(scores))
	}
	assertClose(t, scores["feature-agent"], matcher.ScoreV2(ctx, "feature-agent", "feature", nil))
	assertClose(t, scores["docs-agent"], matcher.ScoreV2(ctx, "docs-agent", "feature", nil))
	assertClose(t, scores["missing"], 0.1)
	if scores["feature-agent"] <= scores["docs-agent"] {
		t.Fatalf("expected feature-agent score %v to beat docs-agent score %v", scores["feature-agent"], scores["docs-agent"])
	}
}

func TestEnhancedCapabilityMatcherManifestError(t *testing.T) {
	reg := registry.New(zerolog.New(nil), nil)
	reg.Register("broken", manifestRunner{err: errors.New("capabilities unavailable")})
	matcher := NewEnhancedCapabilityMatcher(reg)

	assertClose(t, matcher.ScoreV2(context.Background(), "broken", "feature", nil), 0.1)
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("got %v, want %v", got, want)
	}
}
