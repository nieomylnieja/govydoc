// Package modroot locates the Go module containing the current working directory.
package modroot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Find returns the absolute path of the nearest directory containing a go.mod file.
// It searches from the current working directory toward the filesystem root.
func Find() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	dir = filepath.Clean(dir)
	for {
		goModPath := filepath.Join(dir, "go.mod")
		info, err := os.Stat(goModPath)
		switch {
		case err == nil && !info.IsDir():
			return dir, nil
		case err == nil:
		case errors.Is(err, os.ErrNotExist):
		default:
			return "", fmt.Errorf("failed to stat %s: %w", goModPath, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found in directory tree")
		}
		dir = parent
	}
}
