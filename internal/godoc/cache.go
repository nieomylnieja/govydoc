package godoc

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/nieomylnieja/govydoc/internal/modroot"
)

type parserLoader func(root string) (*Parser, error)

type parserCache struct {
	mutex   sync.Mutex
	entries map[string]*parserCacheEntry
	loader  parserLoader
}

type parserCacheEntry struct {
	ready  chan struct{}
	parser *Parser
	err    error
}

var defaultParserCache = parserCache{loader: newParser}

// Parse extracts documentation for goType from the current module's source packages.
func Parse(goType reflect.Type) (Docs, error) {
	root, err := modroot.Find()
	if err != nil {
		return nil, fmt.Errorf("failed to find module root: %w", err)
	}
	parser, err := defaultParserCache.parserFor(root)
	if err != nil {
		return nil, err
	}
	return parser.Parse(goType)
}

func (c *parserCache) parserFor(root string) (*Parser, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve module root %q: %w", root, err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve module root %q: %w", root, err)
	}
	entry, load := c.entry(resolvedRoot)
	if !load {
		return entry.result()
	}
	parser, err := c.loader(resolvedRoot)
	if err != nil {
		err = fmt.Errorf("failed to load Go packages from module root %q: %w", resolvedRoot, err)
	}
	c.complete(resolvedRoot, entry, parser, err)
	return parser, err
}

func (c *parserCache) complete(root string, entry *parserCacheEntry, parser *Parser, err error) {
	entry.parser = parser
	entry.err = err

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err != nil {
		delete(c.entries, root)
	}
	close(entry.ready)
}

func (e *parserCacheEntry) result() (*Parser, error) {
	<-e.ready
	return e.parser, e.err
}

func (c *parserCache) entry(root string) (*parserCacheEntry, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.entries == nil {
		c.entries = make(map[string]*parserCacheEntry)
	}
	if entry := c.entries[root]; entry != nil {
		return entry, false
	}
	entry := &parserCacheEntry{ready: make(chan struct{})}
	c.entries[root] = entry
	return entry, true
}
