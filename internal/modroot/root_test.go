package modroot

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFind(t *testing.T) {
	t.Run("go.mod in current directory", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "test")
		t.Chdir(dir)

		root, err := Find()

		require.NoError(t, err)
		assert.Equal(t, dir, root)
	})

	t.Run("go.mod in parent directory", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "test")
		nestedDir := filepath.Join(dir, "a", "b", "c")
		require.NoError(t, os.MkdirAll(nestedDir, 0o750))
		t.Chdir(nestedDir)

		root, err := Find()

		require.NoError(t, err)
		assert.Equal(t, dir, root)
	})

	t.Run("no go.mod in directory tree", func(t *testing.T) {
		dir := t.TempDir()
		volumeRoot := filepath.VolumeName(dir) + string(filepath.Separator)
		_, err := os.Stat(filepath.Join(volumeRoot, "go.mod"))
		if !errors.Is(err, os.ErrNotExist) {
			t.Skip("filesystem root contains go.mod or cannot be inspected")
		}
		t.Chdir(volumeRoot)

		root, err := Find()

		require.EqualError(t, err, "go.mod not found in directory tree")
		assert.Empty(t, root)
	})

	t.Run("go.mod directory", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "parent")
		nestedDir := filepath.Join(dir, "nested")
		require.NoError(t, os.Mkdir(nestedDir, 0o750))
		require.NoError(t, os.Mkdir(filepath.Join(nestedDir, "go.mod"), 0o750))
		t.Chdir(nestedDir)

		root, err := Find()

		require.NoError(t, err)
		assert.Equal(t, dir, root)
	})

	t.Run("nearest go.mod", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "root")
		nestedDir := filepath.Join(dir, "nested")
		require.NoError(t, os.Mkdir(nestedDir, 0o750))
		writeGoMod(t, nestedDir, "nested")
		t.Chdir(nestedDir)

		root, err := Find()

		require.NoError(t, err)
		assert.Equal(t, nestedDir, root)
	})

	t.Run("filesystem permission error", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		info, err := os.Stat(dir)
		require.NoError(t, err)
		originalMode := info.Mode().Perm()
		require.NoError(t, os.Chmod(dir, 0o000))
		t.Cleanup(func() {
			require.NoError(t, os.Chmod(dir, originalMode))
		})

		_, err = Find()
		if err == nil {
			t.Skip("filesystem permissions do not prevent stat")
		}
		assert.ErrorContains(t, err, "failed")
	})
}

func writeGoMod(t *testing.T, dir, module string) {
	t.Helper()
	path := filepath.Join(dir, "go.mod")
	require.NoError(t, os.WriteFile(path, []byte("module "+module+"\n"), 0o600))
}
