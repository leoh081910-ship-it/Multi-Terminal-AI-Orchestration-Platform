// Package engine provides wave management for the AI orchestration platform.
package engine

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// WaveState represents the state of a wave.
type WaveState int

const (
	// WaveOpen means the wave can still accept new tasks.
	WaveOpen WaveState = iota
	// WaveSealed means the wave is sealed and no new tasks can be added.
	WaveSealed
)

// WaveManager handles wave lifecycle and consistency rules.
type WaveManager struct {
	// waveResolver provides wave lookup capabilities
	resolver WaveResolver
}

// WaveResolver provides wave lookup capabilities.
type WaveResolver interface {
	// GetWave retrieves a wave by dispatch reference and wave number.
	GetWave(ctx context.Context, dispatchRef string, waveNum int) (WaveInfo, error)
	// SealWave marks a wave as sealed.
	SealWave(ctx context.Context, dispatchRef string, waveNum int) error
	// CreateWave creates a new wave if it doesn't exist.
	CreateWave(ctx context.Context, dispatchRef string, waveNum int) error
}

// WaveInfo provides information about a wave.
type WaveInfo interface {
	GetDispatchRef() string
	GetWave() int
	GetSealedAt() *time.Time
	GetCreatedAt() time.Time
}

// NewWaveManager creates a new wave manager.
func NewWaveManager(resolver WaveResolver) *WaveManager {
	return &WaveManager{
		resolver: resolver,
	}
}

// CanRouteToWave checks if a task can be routed to a wave.
// REQUIRES (WAVE-02): Tasks cannot enter 'routed' before wave is sealed.
func (wm *WaveManager) CanRouteToWave(ctx context.Context, dispatchRef string, waveNum int) (bool, error) {
	wave, err := wm.resolver.GetWave(ctx, dispatchRef, waveNum)
	if err != nil {
		return false, fmt.Errorf("failed to get wave: %w", err)
	}

	// Wave must be sealed to allow routing
	if wave.GetSealedAt() == nil {
		return false, nil
	}

	return true, nil
}

// SealWave seals a wave, preventing new tasks from being added.
// REQUIRES (WAVE-04): Once sealed, wave cannot accept new tasks.
func (wm *WaveManager) SealWave(ctx context.Context, dispatchRef string, waveNum int) error {
	wave, err := wm.resolver.GetWave(ctx, dispatchRef, waveNum)
	if err != nil {
		return fmt.Errorf("failed to get wave: %w", err)
	}

	if wave.GetSealedAt() != nil {
		return ErrWaveAlreadySealed
	}

	return wm.resolver.SealWave(ctx, dispatchRef, waveNum)
}

// EnsureWave ensures a wave exists for a task.
// REQUIRES (WAVE-01): Wave record is automatically created/updated when task is enqueued.
func (wm *WaveManager) EnsureWave(ctx context.Context, dispatchRef string, waveNum int) error {
	wave, err := wm.resolver.GetWave(ctx, dispatchRef, waveNum)
	if err != nil && !errors.Is(err, ErrWaveNotFound) {
		return fmt.Errorf("failed to check wave existence: %w", err)
	}

	if wave != nil {
		// Wave exists, check if sealed
		if wave.GetSealedAt() != nil {
			return ErrWaveAlreadySealed
		}
		return nil
	}

	// Create new wave
	return wm.resolver.CreateWave(ctx, dispatchRef, waveNum)
}

// GetWaveState returns the current state of a wave.
func (wm *WaveManager) GetWaveState(ctx context.Context, dispatchRef string, waveNum int) (WaveState, error) {
	wave, err := wm.resolver.GetWave(ctx, dispatchRef, waveNum)
	if err != nil {
		return WaveOpen, err
	}

	if wave.GetSealedAt() != nil {
		return WaveSealed, nil
	}

	return WaveOpen, nil
}

// Errors.
var (
	// ErrWaveAlreadySealed is returned when trying to add tasks to a sealed wave.
	ErrWaveAlreadySealed = errors.New("wave already sealed")
	// ErrWaveNotFound is returned when a wave doesn't exist.
	ErrWaveNotFound = errors.New("wave not found")
	// ErrWaveNotSealed is returned when trying to route to an unsealed wave.
	ErrWaveNotSealed = errors.New("wave not sealed")
)
