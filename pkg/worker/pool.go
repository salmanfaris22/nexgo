// Package worker provides a lightweight worker pool for concurrent task execution.
package worker

import (
	"sync"
)

// Task is a unit of work.
type Task func() error

// Pool manages a set of worker goroutines.
type Pool struct {
	workers int
	tasks   chan Task
	wg      sync.WaitGroup
	errs    []error
	mu      sync.Mutex
	started bool
}

// New creates a worker pool with the given number of workers.
func New(workers int) *Pool {
	if workers < 1 {
		workers = 1
	}
	p := &Pool{
		workers: workers,
		tasks:   make(chan Task, workers*10),
	}
	p.Start()
	return p
}

// Start launches worker goroutines. Safe to call only once.
func (p *Pool) Start() {
	if p.started {
		return
	}
	p.started = true
	for i := 0; i < p.workers; i++ {
		go func() {
			for task := range p.tasks {
				if err := task(); err != nil {
					p.mu.Lock()
					p.errs = append(p.errs, err)
					p.mu.Unlock()
				}
				p.wg.Done()
			}
		}()
	}
}

// Submit adds a task to the pool.
func (p *Pool) Submit(task Task) {
	p.wg.Add(1)
	p.tasks <- task
}

// Wait blocks until all submitted tasks are done, then closes the task channel.
// After Wait, the pool cannot be reused.
func (p *Pool) Wait() []error {
	p.wg.Wait()
	close(p.tasks)
	p.mu.Lock()
	errs := p.errs
	p.mu.Unlock()
	return errs
}

// Run executes tasks concurrently and waits for completion.
// This is a convenience function for one-shot use.
func Run(workers int, tasks []Task) []error {
	p := New(workers)
	for _, t := range tasks {
		p.Submit(t)
	}
	return p.Wait()
}

// Map executes a function on each item concurrently and returns results in order.
func Map[T any, R any](workers int, items []T, fn func(T) R) []R {
	results := make([]R, len(items))
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, item T) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = fn(item)
		}(i, item)
	}
	wg.Wait()
	return results
}

// MapErr executes a function on each item concurrently, returns results and first error.
func MapErr[T any, R any](workers int, items []T, fn func(T) (R, error)) ([]R, error) {
	results := make([]R, len(items))
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	var firstErr error
	var errMu sync.Mutex

	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, item T) {
			defer wg.Done()
			defer func() { <-sem }()
			r, err := fn(item)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}
			results[i] = r
		}(i, item)
	}
	wg.Wait()
	return results, firstErr
}

// ForEach executes a function on each item concurrently.
func ForEach[T any](workers int, items []T, fn func(T)) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for _, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(item T) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(item)
		}(item)
	}
	wg.Wait()
}

// Global pool for convenience — auto-started.
var globalPool = New(4)

// Submit adds a task to the global pool.
func Submit(task Task) {
	globalPool.Submit(task)
}

// Wait waits for the global pool to finish and closes it.
func Wait() []error {
	return globalPool.Wait()
}
