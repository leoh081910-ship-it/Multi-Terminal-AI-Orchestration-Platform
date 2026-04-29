package transport

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeArtifactsToPath(artifacts []Artifact, artifactPath string) error {
	if err := os.MkdirAll(artifactPath, 0755); err != nil {
		return fmt.Errorf("failed to create artifact directory %s: %w", artifactPath, err)
	}

	for _, artifact := range artifacts {
		dir := filepath.Join(artifactPath, filepath.Dir(artifact.Path))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		fullPath := filepath.Join(artifactPath, artifact.Path)
		if artifact.IsDir {
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
			}
			continue
		}

		if err := os.WriteFile(fullPath, artifact.Content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}
