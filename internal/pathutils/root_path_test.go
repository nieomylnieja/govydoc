package pathutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindModuleRoot(t *testing.T) {
	t.Run("finds go.mod in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n"), 0o644)
		require.NoError(t, err)

		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(origDir)
		}()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		root, err := FindModuleRoot()
		require.NoError(t, err)
		assert.Equal(t, tmpDir, root)
	})

	t.Run("finds go.mod in parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n"), 0o644)
		require.NoError(t, err)

		subDir := filepath.Join(tmpDir, "subdir", "nested")
		err = os.MkdirAll(subDir, 0o755)
		require.NoError(t, err)

		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(origDir)
		}()

		err = os.Chdir(subDir)
		require.NoError(t, err)

		root, err := FindModuleRoot()
		require.NoError(t, err)
		assert.Equal(t, tmpDir, root)
	})

	t.Run("finds go.mod multiple levels up", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n"), 0o644)
		require.NoError(t, err)

		deepDir := filepath.Join(tmpDir, "a", "b", "c", "d")
		err = os.MkdirAll(deepDir, 0o755)
		require.NoError(t, err)

		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(origDir)
		}()

		err = os.Chdir(deepDir)
		require.NoError(t, err)

		root, err := FindModuleRoot()
		require.NoError(t, err)
		assert.Equal(t, tmpDir, root)
	})

	t.Run("returns error when go.mod not found", func(t *testing.T) {
		// Note: This test is challenging because t.TempDir() might be within
		// a directory tree that has a go.mod somewhere. We'll verify the
		// error message format is correct if an error occurs.
		tmpDir := t.TempDir()

		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(origDir)
		}()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		result, err := FindModuleRoot()
		// Either we find a go.mod (in parent dirs) or we get the expected error
		if err != nil {
			assert.Contains(t, err.Error(), "go.mod not found")
		} else {
			// If we found one, verify it's actually valid
			goModPath := filepath.Join(result, "go.mod")
			_, statErr := os.Stat(goModPath)
			require.NoError(t, statErr, "if no error, go.mod must exist at returned path")
		}
	})

	t.Run("ignores go.mod directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create a directory named go.mod (not a file)
		goModDir := filepath.Join(tmpDir, "go.mod")
		err := os.Mkdir(goModDir, 0o755)
		require.NoError(t, err)

		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(origDir)
		}()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		result, err := FindModuleRoot()
		// Should skip the go.mod directory and either find a real go.mod file
		// in parent directories or return an error
		if err == nil {
			// If we found a go.mod, it must not be the directory we created
			assert.NotEqual(t, tmpDir, result, "should not return directory with go.mod directory")
			// Verify the returned path has a real go.mod file
			goModPath := filepath.Join(result, "go.mod")
			fi, statErr := os.Stat(goModPath)
			require.NoError(t, statErr)
			assert.False(t, fi.IsDir(), "found go.mod must be a file, not directory")
		} else {
			assert.Contains(t, err.Error(), "go.mod not found")
		}
	})

	t.Run("stops at nearest go.mod", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create go.mod in root
		rootGoMod := filepath.Join(tmpDir, "go.mod")
		err := os.WriteFile(rootGoMod, []byte("module root\n"), 0o644)
		require.NoError(t, err)

		// Create nested directory with its own go.mod
		nestedDir := filepath.Join(tmpDir, "nested")
		err = os.Mkdir(nestedDir, 0o755)
		require.NoError(t, err)
		nestedGoMod := filepath.Join(nestedDir, "go.mod")
		err = os.WriteFile(nestedGoMod, []byte("module nested\n"), 0o644)
		require.NoError(t, err)

		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(origDir)
		}()

		err = os.Chdir(nestedDir)
		require.NoError(t, err)

		root, err := FindModuleRoot()
		require.NoError(t, err)
		// Should find the nearest go.mod, not the parent one
		assert.Equal(t, nestedDir, root)
	})

	t.Run("handles permission errors gracefully", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Test not applicable when running as root")
		}

		tmpDir := t.TempDir()

		// Create a go.mod file that we'll make inaccessible
		goModPath := filepath.Join(tmpDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n"), 0o644)
		require.NoError(t, err)

		// Remove read permissions from the go.mod file
		err = os.Chmod(goModPath, 0o000)
		require.NoError(t, err)
		defer func() {
			_ = os.Chmod(goModPath, 0o644)
		}()

		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(origDir)
		}()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		_, err = FindModuleRoot()
		// Should get a permission error, not a panic
		// The important thing is we don't panic - we return an error
		if err != nil {
			// If we get an error, it should be about permissions or stat failure
			assert.Contains(t, err.Error(), "failed to stat")
		}
		// Note: on some systems, os.Stat might still succeed even with 0000 perms
		// The key is that we don't panic
	})

	t.Run("works from actual project directory", func(t *testing.T) {
		// This test validates that the function works in the real project
		root, err := FindModuleRoot()
		require.NoError(t, err)
		assert.NotEmpty(t, root)

		// Verify go.mod actually exists at the returned path
		goModPath := filepath.Join(root, "go.mod")
		_, err = os.Stat(goModPath)
		require.NoError(t, err, "go.mod should exist at returned root path")
	})
}
