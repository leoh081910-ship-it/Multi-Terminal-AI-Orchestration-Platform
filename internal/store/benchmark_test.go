package store

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	entdialect "entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/rs/zerolog"
	_ "modernc.org/sqlite"
)

var benchStates = []string{"pending", "running", "done", "failed", "blocked", "review"}

func seedTasks(b *testing.B, repo *Repository, n int) {
	b.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("BENCH-%06d", i)
		ref := fmt.Sprintf("bench-ref-%06d", i)
		card := &TaskCard{
			ID:          id,
			DispatchRef: ref,
			State:       benchStates[i%len(benchStates)],
			Wave:        i % 10,
			Transport:   "claude",
			ProjectID:   "bench-project",
			CardJSON:    fmt.Sprintf(`{"id":"%s","project_id":"bench-project","dispatch_ref":"%s","transport":"claude","title":"bench task %d"}`, id, ref, i),
		}
		if _, err := repo.CreateTask(ctx, card); err != nil {
			b.Fatalf("seed task %d: %v", i, err)
		}
	}
}

func setupBenchDB(b *testing.B) (*Repository, func()) {
	b.Helper()
	db, err := sql.Open("sqlite", "file:ent?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		b.Fatal(err)
	}
	drv := entsql.OpenDB(entdialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		b.Fatal(err)
	}
	logger := zerolog.New(nil)
	repo := NewRepository(client, &logger)
	return repo, func() { client.Close() }
}

func BenchmarkListAllTasks100(b *testing.B)  { benchListAll(b, 100) }
func BenchmarkListAllTasks1000(b *testing.B) { benchListAll(b, 1000) }

func benchListAll(b *testing.B, n int) {
	repo, cleanup := setupBenchDB(b)
	defer cleanup()
	seedTasks(b, repo, n)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.ListAllTasks(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListTasksByState(b *testing.B) {
	repo, cleanup := setupBenchDB(b)
	defer cleanup()
	seedTasks(b, repo, 1000)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.ListTasksByState(ctx, "running"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCreateTask(b *testing.B) {
	repo, cleanup := setupBenchDB(b)
	defer cleanup()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("BENCH-CR-%06d", i)
		ref := fmt.Sprintf("bench-cr-%06d", i)
		card := &TaskCard{
			ID:          id,
			DispatchRef: ref,
			State:       "pending",
			Wave:        0,
			Transport:   "claude",
			ProjectID:   "bench-project",
			CardJSON:    fmt.Sprintf(`{"id":"%s","project_id":"bench-project","dispatch_ref":"%s","transport":"claude","title":"bench create"}`, id, ref),
		}
		if _, err := repo.CreateTask(ctx, card); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTaskByID(b *testing.B) {
	repo, cleanup := setupBenchDB(b)
	defer cleanup()
	seedTasks(b, repo, 1000)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.GetTaskByID(ctx, fmt.Sprintf("BENCH-%06d", i%1000)); err != nil {
			b.Fatal(err)
		}
	}
}
