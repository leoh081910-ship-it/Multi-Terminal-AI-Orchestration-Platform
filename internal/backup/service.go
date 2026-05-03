package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// BackupInfo describes a backup file
type BackupInfo struct {
	Filename  string    `json:"filename"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Checksum  string    `json:"checksum"`
	CreatedAt time.Time `json:"created_at"`
}

// Service manages database backups
type Service struct {
	dbPath    string
	backupDir string
	maxKeep   int
	logger    zerolog.Logger
	mu        sync.Mutex

	// Stats
	lastSuccess time.Time
	lastError   error
	successCount int64
	failureCount int64
}

// Config configures the backup service
type Config struct {
	DBPath    string
	BackupDir string
	MaxKeep   int
}

// NewService creates a new backup service
func NewService(cfg Config, logger zerolog.Logger) (*Service, error) {
	if cfg.MaxKeep <= 0 {
		cfg.MaxKeep = 24
	}
	if err := os.MkdirAll(cfg.BackupDir, 0755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}
	return &Service{
		dbPath:    cfg.DBPath,
		backupDir: cfg.BackupDir,
		maxKeep:   cfg.MaxKeep,
		logger:    logger,
	}, nil
}

// CreateBackup performs a single backup
func (s *Service) CreateBackup(ctx context.Context) (*BackupInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	timestamp := now.Format("20060102-150405.000000000")
	filename := fmt.Sprintf("backup-%s.db", timestamp)
	dest := filepath.Join(s.backupDir, filename)

	src, err := os.Open(s.dbPath)
	if err != nil {
		s.recordFailure(err)
		return nil, fmt.Errorf("open source db: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dest)
	if err != nil {
		s.recordFailure(err)
		return nil, fmt.Errorf("create backup file: %w", err)
	}
	defer dst.Close()

	hash := sha256.New()
	mw := io.MultiWriter(dst, hash)
	size, err := io.Copy(mw, src)
	if err != nil {
		s.recordFailure(err)
		os.Remove(dest)
		return nil, fmt.Errorf("copy data: %w", err)
	}

	info := &BackupInfo{
		Filename:  filename,
		Path:      dest,
		Size:      size,
		Checksum:  hex.EncodeToString(hash.Sum(nil)),
		CreatedAt: time.Now().UTC(),
	}

	s.lastSuccess = info.CreatedAt
	s.lastError = nil
	s.successCount++

	s.logger.Info().
		Str("filename", filename).
		Int64("size", size).
		Str("checksum", info.Checksum[:16]).
		Msg("backup created")

	if err := s.cleanup(); err != nil {
		s.logger.Warn().Err(err).Msg("backup cleanup failed")
	}

	return info, nil
}

// ListBackups returns all available backups, newest first
func (s *Service) ListBackups() ([]*BackupInfo, error) {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		return nil, fmt.Errorf("read backup dir: %w", err)
	}

	var backups []*BackupInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".db" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, &BackupInfo{
			Filename:  name,
			Path:      filepath.Join(s.backupDir, name),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// VerifyBackup checks file integrity by recomputing checksum
func (s *Service) VerifyBackup(filename string) (string, error) {
	path := filepath.Join(s.backupDir, filename)
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open backup: %w", err)
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", fmt.Errorf("hash backup: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// RestoreBackup replaces the live database with the backup
// IMPORTANT: server must be stopped before calling this
func (s *Service) RestoreBackup(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	src := filepath.Join(s.backupDir, filename)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	// Save current db as pre-restore snapshot
	preRestore := s.dbPath + ".pre-restore-" + time.Now().UTC().Format("20060102-150405")
	if _, err := os.Stat(s.dbPath); err == nil {
		if err := copyFile(s.dbPath, preRestore); err != nil {
			return fmt.Errorf("save pre-restore snapshot: %w", err)
		}
	}

	// Copy backup over current db
	if err := copyFile(src, s.dbPath); err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}

	s.logger.Info().
		Str("backup", filename).
		Str("pre_restore_snapshot", preRestore).
		Msg("backup restored")

	return nil
}

// Stats returns backup service statistics
func (s *Service) Stats() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	backups, _ := s.ListBackups()
	var totalSize int64
	for _, b := range backups {
		totalSize += b.Size
	}

	stats := map[string]any{
		"backup_count":  len(backups),
		"total_size":    totalSize,
		"success_count": s.successCount,
		"failure_count": s.failureCount,
		"max_keep":      s.maxKeep,
		"backup_dir":    s.backupDir,
	}
	if !s.lastSuccess.IsZero() {
		stats["last_success"] = s.lastSuccess
	}
	if s.lastError != nil {
		stats["last_error"] = s.lastError.Error()
	}
	return stats
}

func (s *Service) cleanup() error {
	backups, err := s.ListBackups()
	if err != nil {
		return err
	}
	if len(backups) <= s.maxKeep {
		return nil
	}
	for _, b := range backups[s.maxKeep:] {
		if err := os.Remove(b.Path); err != nil {
			s.logger.Warn().Err(err).Str("file", b.Filename).Msg("failed to delete old backup")
			continue
		}
		s.logger.Info().Str("file", b.Filename).Msg("old backup deleted")
	}
	return nil
}

func (s *Service) recordFailure(err error) {
	s.lastError = err
	s.failureCount++
	s.logger.Error().Err(err).Msg("backup failed")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
