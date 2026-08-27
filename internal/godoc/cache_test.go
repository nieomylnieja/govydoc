package godoc

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserCache_parserFor(t *testing.T) {
	t.Run("same module root", func(t *testing.T) {
		root := t.TempDir()
		var calls int
		expected := &Parser{}
		resolvedRoot, err := filepath.EvalSymlinks(root)
		require.NoError(t, err)
		cache := parserCache{loader: func(actualRoot string) (*Parser, error) {
			calls++
			assert.Equal(t, resolvedRoot, actualRoot)
			return expected, nil
		}}

		first, err := cache.parserFor(root)
		require.NoError(t, err)
		second, err := cache.parserFor(root)
		require.NoError(t, err)

		assert.Same(t, expected, first)
		assert.Same(t, expected, second)
		assert.Equal(t, 1, calls)
	})

	t.Run("different module roots", func(t *testing.T) {
		firstRoot := t.TempDir()
		secondRoot := t.TempDir()
		var calls int
		cache := parserCache{loader: func(string) (*Parser, error) {
			calls++
			return &Parser{}, nil
		}}

		first, err := cache.parserFor(firstRoot)
		require.NoError(t, err)
		second, err := cache.parserFor(secondRoot)
		require.NoError(t, err)

		assert.NotSame(t, first, second)
		assert.Equal(t, 2, calls)
	})

	t.Run("failed load", func(t *testing.T) {
		root := t.TempDir()
		resolvedRoot, err := filepath.EvalSymlinks(root)
		require.NoError(t, err)
		loadErr := errors.New("load failed")
		var calls int
		expected := &Parser{}
		cache := parserCache{loader: func(string) (*Parser, error) {
			calls++
			if calls == 1 {
				return nil, loadErr
			}
			return expected, nil
		}}

		_, err = cache.parserFor(root)
		require.ErrorIs(t, err, loadErr)
		require.ErrorContains(t, err, resolvedRoot)
		actual, err := cache.parserFor(root)
		require.NoError(t, err)

		assert.Same(t, expected, actual)
		assert.Equal(t, 2, calls)
	})

	t.Run("concurrent load", func(t *testing.T) {
		const goroutines = 16
		root := t.TempDir()
		var calls atomic.Int64
		started := make(chan struct{})
		release := make(chan struct{})
		expected := &Parser{}
		cache := parserCache{loader: func(string) (*Parser, error) {
			if calls.Add(1) == 1 {
				close(started)
			}
			<-release
			return expected, nil
		}}

		parsers := make([]*Parser, goroutines)
		errs := make([]error, goroutines)
		var waitGroup sync.WaitGroup
		for i := range goroutines {
			waitGroup.Go(func() {
				parsers[i], errs[i] = cache.parserFor(root)
			})
		}
		<-started
		close(release)
		waitGroup.Wait()

		for i := range goroutines {
			require.NoError(t, errs[i])
			assert.Same(t, expected, parsers[i])
		}
		assert.Equal(t, int64(1), calls.Load())
	})

	t.Run("concurrent failed load", func(t *testing.T) {
		const waiters = 16
		root, err := filepath.EvalSymlinks(t.TempDir())
		require.NoError(t, err)
		loadErr := errors.New("load failed")
		var calls int
		var startedOnce sync.Once
		started := make(chan struct{})
		release := make(chan struct{})
		cache := parserCache{loader: func(string) (*Parser, error) {
			calls++
			startedOnce.Do(func() { close(started) })
			<-release
			return nil, loadErr
		}}
		firstDone := make(chan struct{})
		var firstErr error
		go func() {
			_, firstErr = cache.parserFor(root)
			close(firstDone)
		}()
		<-started
		cache.mutex.Lock()
		entry := cache.entries[root]
		cache.mutex.Unlock()
		require.NotNil(t, entry)

		errs := make([]error, waiters)
		var waitGroup sync.WaitGroup
		for i := range waiters {
			waitGroup.Go(func() {
				_, errs[i] = entry.result()
			})
		}
		close(release)
		<-firstDone
		waitGroup.Wait()

		require.ErrorIs(t, firstErr, loadErr)
		for _, err := range errs {
			require.ErrorIs(t, err, loadErr)
			require.ErrorContains(t, err, root)
		}
		assert.Equal(t, 1, calls)

		_, err = cache.parserFor(root)
		require.ErrorIs(t, err, loadErr)
		assert.Equal(t, 2, calls)
	})

	t.Run("symlinked module root", func(t *testing.T) {
		root := t.TempDir()
		link := filepath.Join(t.TempDir(), "module")
		require.NoError(t, os.Symlink(root, link))
		var calls int
		expected := &Parser{}
		cache := parserCache{loader: func(string) (*Parser, error) {
			calls++
			return expected, nil
		}}

		first, err := cache.parserFor(root)
		require.NoError(t, err)
		second, err := cache.parserFor(link)
		require.NoError(t, err)

		assert.Same(t, first, second)
		assert.Equal(t, 1, calls)
	})
}
