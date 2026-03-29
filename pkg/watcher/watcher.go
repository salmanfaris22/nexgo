package watcher

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ChangeType indicates what kind of file change occurred
type ChangeType int

const (
	ChangeCreate ChangeType = iota
	ChangeModify
	ChangeDelete
	ChangeRename
)

// Event represents a file system change
type Event struct {
	Path string
	Type ChangeType
}

// Watcher watches files for changes (simplified, no external deps)
type Watcher struct {
	mu        sync.RWMutex
	dirs      []string
	callbacks []func(Event)
	fileCache map[string]time.Time
	interval  time.Duration
	done      chan struct{}
}

// New creates a new file watcher
func New(interval time.Duration) *Watcher {
	return &Watcher{
		fileCache: make(map[string]time.Time),
		interval:  interval,
		done:      make(chan struct{}),
	}
}

// Watch adds a directory to watch
func (w *Watcher) Watch(dir string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.dirs = append(w.dirs, dir)
}

// OnChange registers a callback for file changes
func (w *Watcher) OnChange(fn func(Event)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, fn)
}

// Start begins watching
func (w *Watcher) Start() {
	// Initial scan to populate cache
	w.scan(false)

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.scan(true)
			case <-w.done:
				return
			}
		}
	}()

	log.Printf("[NexGo] 👁  Watching for changes...")
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.done)
}

func (w *Watcher) scan(notify bool) {
	w.mu.Lock()
	dirs := make([]string, len(w.dirs))
	copy(dirs, w.dirs)
	w.mu.Unlock()

	current := make(map[string]time.Time)

	for _, dir := range dirs {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if d.IsDir() {
				return nil
			}

			info, err := os.Stat(path)
			if err != nil {
				return nil
			}

			current[path] = info.ModTime()
			return nil
		})
	}

	if !notify {
		w.mu.Lock()
		w.fileCache = current
		w.mu.Unlock()
		return
	}

	w.mu.Lock()
	cache := w.fileCache
	callbacks := append([]func(Event){}, w.callbacks...)
	w.mu.Unlock()

	// CREATE + MODIFY
	for path, modTime := range current {
		if oldTime, existed := cache[path]; !existed {
			event := Event{Path: path, Type: ChangeCreate}
			for _, cb := range callbacks {
				cb(event)
			}
		} else if modTime.After(oldTime) {
			event := Event{Path: path, Type: ChangeModify}
			for _, cb := range callbacks {
				cb(event)
			}
		}
	}

	// DELETE
	for path := range cache {
		if _, exists := current[path]; !exists {
			event := Event{Path: path, Type: ChangeDelete}
			for _, cb := range callbacks {
				cb(event)
			}
		}
	}

	w.mu.Lock()
	w.fileCache = current
	w.mu.Unlock()
}
