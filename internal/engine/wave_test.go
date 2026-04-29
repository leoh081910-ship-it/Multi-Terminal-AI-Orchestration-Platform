package engine

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Mock wave resolver for testing
type mockWaveResolver struct {
	waves map[string]*mockWaveInfo
}

type mockWaveInfo struct {
	dispatchRef string
	waveNum     int
	sealedAt    *time.Time
	createdAt   time.Time
}

func (m *mockWaveInfo) GetDispatchRef() string  { return m.dispatchRef }
func (m *mockWaveInfo) GetWave() int            { return m.waveNum }
func (m *mockWaveInfo) GetSealedAt() *time.Time { return m.sealedAt }
func (m *mockWaveInfo) GetCreatedAt() time.Time { return m.createdAt }

func newMockWaveResolver() *mockWaveResolver {
	return &mockWaveResolver{
		waves: make(map[string]*mockWaveInfo),
	}
}

func (m *mockWaveResolver) waveKey(dispatchRef string, waveNum int) string {
	return fmt.Sprintf("%s:%d", dispatchRef, waveNum)
}

func (m *mockWaveResolver) GetWave(ctx context.Context, dispatchRef string, waveNum int) (WaveInfo, error) {
	key := m.waveKey(dispatchRef, waveNum)
	wave, exists := m.waves[key]
	if !exists {
		return nil, ErrWaveNotFound
	}
	return wave, nil
}

func (m *mockWaveResolver) SealWave(ctx context.Context, dispatchRef string, waveNum int) error {
	key := m.waveKey(dispatchRef, waveNum)
	wave, exists := m.waves[key]
	if !exists {
		return ErrWaveNotFound
	}
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	wave.sealedAt = &now
	return nil
}

func (m *mockWaveResolver) CreateWave(ctx context.Context, dispatchRef string, waveNum int) error {
	key := m.waveKey(dispatchRef, waveNum)
	m.waves[key] = &mockWaveInfo{
		dispatchRef: dispatchRef,
		waveNum:     waveNum,
		sealedAt:    nil,
		createdAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	return nil
}

func TestWaveManagerCanRouteToWave(t *testing.T) {
	resolver := newMockWaveResolver()
	wm := NewWaveManager(resolver)

	ctx := context.Background()

	// Create an unsealed wave
	resolver.CreateWave(ctx, "dispatch1", 1)

	// Should not be able to route to unsealed wave
	canRoute, err := wm.CanRouteToWave(ctx, "dispatch1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canRoute {
		t.Error("Should not be able to route to unsealed wave")
	}

	// Seal the wave
	if err := wm.SealWave(ctx, "dispatch1", 1); err != nil {
		t.Fatalf("failed to seal wave: %v", err)
	}

	// Now should be able to route
	canRoute, err = wm.CanRouteToWave(ctx, "dispatch1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !canRoute {
		t.Error("Should be able to route to sealed wave")
	}
}

func TestWaveManagerEnsureWave(t *testing.T) {
	resolver := newMockWaveResolver()
	wm := NewWaveManager(resolver)

	ctx := context.Background()

	// Create a new wave
	if err := wm.EnsureWave(ctx, "dispatch1", 1); err != nil {
		t.Fatalf("failed to ensure wave: %v", err)
	}

	// Should be able to get the created wave
	wave, err := resolver.GetWave(ctx, "dispatch1", 1)
	if err != nil {
		t.Fatalf("failed to get wave: %v", err)
	}

	if wave.GetDispatchRef() != "dispatch1" {
		t.Errorf("expected dispatchRef 'dispatch1', got %q", wave.GetDispatchRef())
	}

	if wave.GetWave() != 1 {
		t.Errorf("expected wave 1, got %d", wave.GetWave())
	}

	// Try to create again (should succeed since it's not sealed)
	if err := wm.EnsureWave(ctx, "dispatch1", 1); err != nil {
		t.Fatalf("should be able to ensure existing unsealed wave: %v", err)
	}
}

func TestWaveManagerSealAlreadySealedWave(t *testing.T) {
	resolver := newMockWaveResolver()
	wm := NewWaveManager(resolver)

	ctx := context.Background()

	// Create and seal wave
	resolver.CreateWave(ctx, "dispatch1", 1)
	if err := wm.SealWave(ctx, "dispatch1", 1); err != nil {
		t.Fatalf("failed to seal wave: %v", err)
	}

	// Try to seal again (should return ErrWaveAlreadySealed)
	err := wm.SealWave(ctx, "dispatch1", 1)
	if err != ErrWaveAlreadySealed {
		t.Errorf("expected ErrWaveAlreadySealed, got %v", err)
	}
}
