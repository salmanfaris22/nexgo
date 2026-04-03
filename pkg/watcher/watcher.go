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
)

// Event represents a file system change
type Event struct {
	Path string
	Type ChangeType
}

// Watcher watches files for changes (polling-based, no external deps)
type Watcher struct {
	mu        sync.RWMutex
	dirs      []string
	callbacks []func(Event)
	fileCache map[string]time.Time
	interval  time.Duration
	done      chan struct{}
	stopped   bool
	wg        sync.WaitGroup
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
	w.scan(false)

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
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

	log.Printf("[NexGo] Watching for changes...")
}

// Stop stops the watcher. Safe to call multiple times.
func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	w.mu.Unlock()
	close(w.done)
	w.wg.Wait()
}

func (w *Watcher) scan(notify bool) {
	w.mu.Lock()
	dirs := make([]string, len(w.dirs))
	copy(dirs, w.dirs)
	w.mu.Unlock()

	current := make(map[string]time.Time)

	for _, dir := range dirs {
		if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			current[path] = info.ModTime()
			return nil
		}); err != nil {
			log.Printf("[NexGo] Watcher scan error for %s: %v", dir, err)
		}
	}

	if !notify {
		w.mu.Lock()
		w.fileCache = current
		w.mu.Unlock()
		return
	}

	// Copy cache and callbacks under lock
	w.mu.Lock()
	cache := make(map[string]time.Time, len(w.fileCache))
	for k, v := range w.fileCache {
		cache[k] = v
	}
	callbacks := make([]func(Event), len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.Unlock()

	for path, modTime := range current {
		cached, existed := cache[path]
		if !existed {
			for _, cb := range callbacks {
				cb(Event{Path: path, Type: ChangeCreate})
			}
		} else if modTime.After(cached) {
			for _, cb := range callbacks {
				cb(Event{Path: path, Type: ChangeModify})
			}
		}
	}

	for path := range cache {
		if _, exists := current[path]; !exists {
			for _, cb := range callbacks {
				cb(Event{Path: path, Type: ChangeDelete})
			}
		}
	}

	w.mu.Lock()
	w.fileCache = current
	w.mu.Unlock()
}
