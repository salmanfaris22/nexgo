package migrate

import (
	"path/filepath"
	"testing"
)

func TestMigrateUp(t *testing.T) {
	dir := t.TempDir()
	m := New(filepath.Join(dir, "migrations.json"))

	order := []string{}
	m.Register("001", "first", func() error {
		order = append(order, "001-up")
		return nil
	}, func() error {
		order = append(order, "001-down")
		return nil
	})
	m.Register("002", "second", func() error {
		order = append(order, "002-up")
		return nil
	}, func() error {
		order = append(order, "002-down")
		return nil
	})

	applied, err := m.Up()
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 2 {
		t.Errorf("applied = %d, want 2", len(applied))
	}
	if order[0] != "001-up" || order[1] != "002-up" {
		t.Errorf("order = %v", order)
	}
}

func TestMigrateUpIdempotent(t *testing.T) {
	dir := t.TempDir()
	m := New(filepath.Join(dir, "migrations.json"))

	count := 0
	m.Register("001", "first", func() error {
		count++
		return nil
	}, nil)

	m.Up()
	m.Up() // second call should not re-run

	if count != 1 {
		t.Errorf("migration ran %d times, want 1", count)
	}
}

func TestMigrateDown(t *testing.T) {
	dir := t.TempDir()
	m := New(filepath.Join(dir, "migrations.json"))

	var rolledBack string
	m.Register("001", "first", func() error { return nil }, func() error {
		rolledBack = "001"
		return nil
	})
	m.Register("002", "second", func() error { return nil }, func() error {
		rolledBack = "002"
		return nil
	})

	m.Up()
	rolled, err := m.Down()
	if err != nil {
		t.Fatal(err)
	}
	if rolled != "002" {
		t.Errorf("rolled = %q, want %q", rolled, "002")
	}
	if rolledBack != "002" {
		t.Errorf("rolledBack = %q, want %q", rolledBack, "002")
	}
}

func TestMigrateStatus(t *testing.T) {
	dir := t.TempDir()
	m := New(filepath.Join(dir, "migrations.json"))

	m.Register("001", "Create users", func() error { return nil }, nil)
	m.Register("002", "Add index", func() error { return nil }, nil)

	m.Up()

	statuses := m.Status()
	if len(statuses) != 2 {
		t.Fatalf("statuses = %d, want 2", len(statuses))
	}
	for _, s := range statuses {
		if !s.Applied {
			t.Errorf("migration %s should be applied", s.Version)
		}
	}
}

func TestPending(t *testing.T) {
	dir := t.TempDir()
	m := New(filepath.Join(dir, "migrations.json"))

	m.Register("001", "first", func() error { return nil }, nil)
	m.Register("002", "second", func() error { return nil }, nil)

	pending := m.Pending()
	if len(pending) != 2 {
		t.Errorf("pending = %d, want 2", len(pending))
	}

	m.Up()
	pending = m.Pending()
	if len(pending) != 0 {
		t.Errorf("pending after up = %d, want 0", len(pending))
	}
}

func TestStatePersistence(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "migrations.json")

	m1 := New(stateFile)
	m1.Register("001", "first", func() error { return nil }, nil)
	m1.Up()

	// New migrator reads persisted state
	m2 := New(stateFile)
	m2.Register("001", "first", func() error { return nil }, nil)
	m2.Register("002", "second", func() error { return nil }, nil)

	pending := m2.Pending()
	if len(pending) != 1 {
		t.Errorf("pending = %d, want 1 (001 already applied)", len(pending))
	}
	if pending[0].Version != "002" {
		t.Errorf("pending version = %q, want %q", pending[0].Version, "002")
	}
}
