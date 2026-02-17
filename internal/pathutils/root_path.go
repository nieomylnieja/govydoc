package pathutils

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// FindModuleRoot returns the absolute path to the module's root directory by
// searching for a go.mod file in the current directory and parent directories.
// Returns an error if the current working directory cannot be determined,
// if filesystem operations fail, or if no go.mod file is found.
func FindModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "failed to get current working directory")
	}
	dir = filepath.Clean(dir)
	for {
		goModPath := filepath.Join(dir, "go.mod")
		fi, err := os.Stat(goModPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", errors.Wrapf(err, "failed to stat %s", goModPath)
			}
			// File doesn't exist, continue searching parent directories
		} else if !fi.IsDir() {
			return dir, nil
		}

		d := filepath.Dir(dir)
		if d == dir {
			break
		}
		dir = d
	}
	return "", errors.New("go.mod not found in directory tree")
}
