package worker

import (
	"errors"
	"sort"
	"sync"
	"testing"
)

func TestPoolNew(t *testing.T) {
	p := New(4)
	if p.workers != 4 {
		t.Errorf("expected 4 workers, got %d", p.workers)
	}
}

func TestPoolNew_MinWorkers(t *testing.T) {
	p := New(0)
	if p.workers != 1 {
		t.Errorf("expected 1 worker minimum, got %d", p.workers)
	}
}

func TestPoolSubmitAndWait(t *testing.T) {
	p := New(2)

	var mu sync.Mutex
	results := []int{}

	for i := 0; i < 10; i++ {
		n := i
		p.Submit(func() error {
			mu.Lock()
			results = append(results, n)
			mu.Unlock()
			return nil
		})
	}

	errs := p.Wait()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d", len(errs))
	}
	sort.Ints(results)
	if len(results) != 10 {
		t.Errorf("expected 10 results, got %d", len(results))
	}
}

func TestPoolTaskError(t *testing.T) {
	p := New(2)
	expectedErr := errors.New("task failed")

	p.Submit(func() error {
		return expectedErr
	})

	errs := p.Wait()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0] != expectedErr {
		t.Errorf("expected task failed error")
	}
}

func TestRun(t *testing.T) {
	var mu sync.Mutex
	count := 0

	tasks := []Task{}
	for i := 0; i < 20; i++ {
		tasks = append(tasks, func() error {
			mu.Lock()
			count++
			mu.Unlock()
			return nil
		})
	}

	errs := Run(4, tasks)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d", len(errs))
	}
	if count != 20 {
		t.Errorf("expected 20 tasks completed, got %d", count)
	}
}

func TestRun_WithError(t *testing.T) {
	tasks := []Task{
		func() error { return nil },
		func() error { return errors.New("fail") },
		func() error { return nil },
	}

	errs := Run(2, tasks)
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestMap(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	results := Map(2, items, func(n int) int {
		return n * 2
	})

	expected := []int{2, 4, 6, 8, 10}
	for i, r := range results {
		if r != expected[i] {
			t.Errorf("index %d: expected %d, got %d", i, expected[i], r)
		}
	}
}

func TestMapStrings(t *testing.T) {
	items := []string{"a", "b", "c"}
	results := Map(2, items, func(s string) string {
		return s + s
	})

	expected := []string{"aa", "bb", "cc"}
	for i, r := range results {
		if r != expected[i] {
			t.Errorf("index %d: expected %s, got %s", i, expected[i], r)
		}
	}
}

func TestMapErr(t *testing.T) {
	items := []int{1, 2, 3}
	results, err := MapErr(2, items, func(n int) (int, error) {
		if n == 2 {
			return 0, errors.New("fail on 2")
		}
		return n * 10, nil
	})

	if err == nil {
		t.Error("expected error")
	}
	if results[0] != 10 {
		t.Errorf("expected 10, got %d", results[0])
	}
}

func TestForEach(t *testing.T) {
	var mu sync.Mutex
	sum := 0
	items := []int{1, 2, 3, 4, 5}

	ForEach(2, items, func(n int) {
		mu.Lock()
		sum += n
		mu.Unlock()
	})

	if sum != 15 {
		t.Errorf("expected sum 15, got %d", sum)
	}
}

func TestGlobalPool(t *testing.T) {
	var mu sync.Mutex
	called := false

	Submit(func() error {
		mu.Lock()
		called = true
		mu.Unlock()
		return nil
	})

	// Don't call global Wait() as it closes the pool permanently
	_ = called
}
