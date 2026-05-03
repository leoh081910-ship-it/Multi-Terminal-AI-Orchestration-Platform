package backup

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (*Service, string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	backupDir := filepath.Join(tmpDir, "backups")

	// Create a fake db file with random content
	data := make([]byte, 1024)
	_, _ = rand.Read(data)
	require.NoError(t, os.WriteFile(dbPath, data, 0644))

	svc, err := NewService(Config{
		DBPath:    dbPath,
		BackupDir: backupDir,
		MaxKeep:   3,
	}, zerolog.Nop())
	require.NoError(t, err)

	return svc, dbPath, backupDir
}

func TestCreateBackup(t *testing.T) {
	svc, _, _ := setupTest(t)
	info, err := svc.CreateBackup(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, info.Filename)
	assert.NotEmpty(t, info.Checksum)
	assert.Equal(t, int64(1024), info.Size)
	assert.FileExists(t, info.Path)
}

func TestListBackups(t *testing.T) {
	svc, _, _ := setupTest(t)
	for i := 0; i < 3; i++ {
		_, err := svc.CreateBackup(context.Background())
		require.NoError(t, err)
	}
	backups, err := svc.ListBackups()
	require.NoError(t, err)
	assert.Len(t, backups, 3)
}

func TestCleanupExceedsMax(t *testing.T) {
	svc, _, _ := setupTest(t)
	// Create 5 backups, max_keep = 3
	for i := 0; i < 5; i++ {
		_, err := svc.CreateBackup(context.Background())
		require.NoError(t, err)
	}
	backups, err := svc.ListBackups()
	require.NoError(t, err)
	assert.LessOrEqual(t, len(backups), 3, "cleanup should keep at most max_keep backups")
}

func TestVerifyBackup(t *testing.T) {
	svc, _, _ := setupTest(t)
	info, err := svc.CreateBackup(context.Background())
	require.NoError(t, err)

	checksum, err := svc.VerifyBackup(info.Filename)
	require.NoError(t, err)
	assert.Equal(t, info.Checksum, checksum)
}

func TestRestoreBackup(t *testing.T) {
	svc, dbPath, _ := setupTest(t)
	originalData, _ := os.ReadFile(dbPath)

	info, err := svc.CreateBackup(context.Background())
	require.NoError(t, err)

	// Modify the original db
	require.NoError(t, os.WriteFile(dbPath, []byte("modified"), 0644))

	// Restore
	require.NoError(t, svc.RestoreBackup(info.Filename))

	// Verify restoration
	restored, err := os.ReadFile(dbPath)
	require.NoError(t, err)
	assert.Equal(t, originalData, restored)
}

func TestStats(t *testing.T) {
	svc, _, _ := setupTest(t)
	_, err := svc.CreateBackup(context.Background())
	require.NoError(t, err)

	stats := svc.Stats()
	assert.Equal(t, int64(1), stats["success_count"])
	assert.Equal(t, 1, stats["backup_count"])
}
