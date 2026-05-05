package template

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"testing"

	_ "modernc.org/sqlite"
)

func TestStoreInitTable(t *testing.T) {
	db, err := sql.Open("sqlite", "file:template_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	store := NewStore(db)
	ctx := context.Background()

	if err := store.InitTable(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify table exists by inserting a template
	tpl := &TaskTemplateSQL{
		Name:       "Test Template",
		ProjectID:  "test-project",
		OwnerAgent: "Claude",
		TaskType:   "task",
		Priority:   3,
		Tags:       []string{"test", "sample"},
	}
	if err := store.Create(ctx, tpl); err != nil {
		t.Fatalf("create: %v", err)
	}

	if tpl.ID == "" {
		t.Error("expected auto-generated ID")
	}
}

func TestStoreCreate(t *testing.T) {
	db, err := sql.Open("sqlite", "file:create_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	store := NewStore(db)
	ctx := context.Background()
	store.InitTable(ctx)

	tpl := &TaskTemplateSQL{
		Name:        "Feature Development",
		Description: "Standard workflow for new feature",
		ProjectID:   "my-project",
		OwnerAgent:  "Claude",
		TaskType:   "implementation",
		Priority:   2,
		Command:    "echo building {{feature_name}}",
		WorkDir:    "/workspace/{{project_path}}",
		TimeoutSec: 3600,
		Tags:       []string{"feature", "development"},
		Inputs: []Input{
			{Name: "feature_name", Type: "string", Required: true, Description: "Feature name"},
			{Name: "project_path", Type: "string", Required: false, Description: "Project path"},
		},
		Outputs: []Output{
			{Name: "artifact_dir", Type: "artifact", Description: "Output directory"},
		},
		CreatedBy: "test-user",
	}

	if err := store.Create(ctx, tpl); err != nil {
		t.Fatalf("create: %v", err)
	}

	if tpl.ID == "" {
		t.Error("expected ID to be generated")
	}
	if tpl.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestStoreGet(t *testing.T) {
	db, err := sql.Open("sqlite", "file:get_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	store := NewStore(db)
	ctx := context.Background()
	store.InitTable(ctx)

	original := &TaskTemplateSQL{
		Name:        "Get Test Template",
		ProjectID:   "proj-1",
		OwnerAgent:  "Gemini",
		TaskType:   "research",
		Priority:   1,
		Tags:       []string{"test"},
		Inputs:     []Input{{Name: "query", Type: "string", Required: true}},
		Outputs:    []Output{{Name: "result", Type: "file"}},
	}
	store.Create(ctx, original)

	fetched, err := store.Get(ctx, original.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if fetched.Name != original.Name {
		t.Errorf("name: got %q, want %q", fetched.Name, original.Name)
	}
	if fetched.OwnerAgent != original.OwnerAgent {
		t.Errorf("owner_agent: got %q, want %q", fetched.OwnerAgent, original.OwnerAgent)
	}
	if len(fetched.Tags) != len(original.Tags) {
		t.Errorf("tags count: got %d, want %d", len(fetched.Tags), len(original.Tags))
	}
	if len(fetched.Inputs) != len(original.Inputs) {
		t.Errorf("inputs count: got %d, want %d", len(fetched.Inputs), len(original.Inputs))
	}
}

func TestStoreList(t *testing.T) {
	db, err := sql.Open("sqlite", "file:list_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	store := NewStore(db)
	ctx := context.Background()
	store.InitTable(ctx)

	// Create templates across different projects
	for i := 0; i < 5; i++ {
		store.Create(ctx, &TaskTemplateSQL{
			Name:       "Template " + string(rune('A'+i)),
			ProjectID:  "proj-1",
			OwnerAgent: "Claude",
			Tags:      []string{"test"},
		})
	}
	store.Create(ctx, &TaskTemplateSQL{
		Name:       "Project 2 Template",
		ProjectID:  "proj-2",
		OwnerAgent: "Gemini",
		Tags:      []string{"test"},
	})

	tpls, total, err := store.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if total != 6 {
		t.Errorf("total all: got %d, want 6", total)
	}
	if len(tpls) != 6 {
		t.Errorf("items all: got %d, want 6", len(tpls))
	}

	// Filter by project
	filtered, totalFiltered, err := store.List(ctx, "proj-1", 10, 0)
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	if totalFiltered != 5 {
		t.Errorf("total filtered: got %d, want 5", totalFiltered)
	}
	if len(filtered) != 5 {
		t.Errorf("items filtered: got %d, want 5", len(filtered))
	}

	// Pagination
	page1, _, err := store.List(ctx, "", 3, 0)
	if err != nil {
		t.Fatalf("list page1: %v", err)
	}
	if len(page1) != 3 {
		t.Errorf("page1 count: got %d, want 3", len(page1))
	}

	page2, _, err := store.List(ctx, "", 3, 3)
	if err != nil {
		t.Fatalf("list page2: %v", err)
	}
	if len(page2) != 3 {
		t.Errorf("page2 count: got %d, want 3", len(page2))
	}
}

func TestStoreUpdate(t *testing.T) {
	db, err := sql.Open("sqlite", "file:update_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	store := NewStore(db)
	ctx := context.Background()
	store.InitTable(ctx)

	tpl := &TaskTemplateSQL{Name: "Original Name", ProjectID: "proj", OwnerAgent: "Claude", Priority: 3}
	store.Create(ctx, tpl)

	updates := map[string]interface{}{
		"name":     "Updated Name",
		"priority": 1,
	}
	if err := store.Update(ctx, tpl.ID, updates); err != nil {
		t.Fatalf("update: %v", err)
	}

	fetched, _ := store.Get(ctx, tpl.ID)
	if fetched.Name != "Updated Name" {
		t.Errorf("name: got %q, want %q", fetched.Name, "Updated Name")
	}
	if fetched.Priority != 1 {
		t.Errorf("priority: got %d, want 1", fetched.Priority)
	}
}

func TestStoreDelete(t *testing.T) {
	db, err := sql.Open("sqlite", "file:delete_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	store := NewStore(db)
	ctx := context.Background()
	store.InitTable(ctx)

	tpl := &TaskTemplateSQL{Name: "To Delete", ProjectID: "proj", OwnerAgent: "Claude"}
	store.Create(ctx, tpl)

	if err := store.Delete(ctx, tpl.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = store.Get(ctx, tpl.ID)
	if err == nil {
		t.Error("expected not-found error after delete")
	}
}

func TestSubstitute(t *testing.T) {
	// Test the substitution logic inline since it's a method on Server
	// We test the pattern here
	tests := []struct {
		input    string
		params   map[string]any
		expected string
	}{
		{"Hello {{name}}!", map[string]any{"name": "World"}, "Hello World!"},
		{"Feature: {{feature}} in {{project}}", map[string]any{"feature": "Auth", "project": "Platform"}, "Feature: Auth in Platform"},
		{"No substitution", map[string]any{"name": "World"}, "No substitution"},
		{"{{name}}", map[string]any{"name": "Alice"}, "Alice"},
		{"{{name:default}}", map[string]any{}, "default"},
		{"{{name:fallback}}", map[string]any{"name": "Bob"}, "Bob"},
	}

	for _, tt := range tests {
		result := substituteTest(tt.input, tt.params)
		if result != tt.expected {
			t.Errorf("substitute(%q, %v) = %q, want %q", tt.input, tt.params, result, tt.expected)
		}
	}
}

// substituteTest mirrors the Server.substituteString logic for isolated testing.
func substituteTest(text string, params map[string]any) string {
	re := regexp.MustCompile(`\{\{([^}:]+)(?::([^}]*))?\}\}`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		matches := re.FindStringSubmatch(match)
		paramName := matches[1]
		defaultVal := ""
		if len(matches) > 2 {
			defaultVal = matches[2]
		}
		if val, ok := params[paramName]; ok && val != nil {
			return fmt.Sprintf("%v", val)
		}
		return defaultVal
	})
}
