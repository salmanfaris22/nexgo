package watcher

import (
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
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			info, err := d.Info()
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
	callbacks := make([]func(Event), len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.Unlock()

	changed := false
	for path, modTime := range current {
		cached, existed := cache[path]
		if !existed {
			for _, cb := range callbacks {
				cb(Event{Path: path, Type: ChangeCreate})
			}
			changed = true
		} else if modTime.After(cached) {
			for _, cb := range callbacks {
				cb(Event{Path: path, Type: ChangeModify})
			}
			changed = true
		}
	}
	// Detect deletions and remove from cache
	for path := range cache {
		if _, exists := current[path]; !exists {
			for _, cb := range callbacks {
				cb(Event{Path: path, Type: ChangeDelete})
			}
			changed = true
			// ✅ NEW CODE: immediately delete from fileCache to prevent memory leak
			w.mu.Lock()
			delete(w.fileCache, path)
			w.mu.Unlock()
		}
	}
	_ = changed

	w.mu.Lock()
	// Merge current into fileCache (preserve new files)
	for path, modTime := range current {
		w.fileCache[path] = modTime
	}
	w.mu.Unlock()
}
