// Package migrate provides a simple database migration system.
// Migrations are Go functions registered in order and tracked in a migrations table.
package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Migration represents a single database migration.
type Migration struct {
	Version     string
	Description string
	Up          func() error
	Down        func() error
}

// Record tracks applied migrations.
type Record struct {
	Version   string    `json:"version"`
	AppliedAt time.Time `json:"applied_at"`
}

// Migrator manages database migrations.
type Migrator struct {
	mu         sync.Mutex
	migrations []Migration
	stateFile  string
	applied    map[string]Record
}

// New creates a migrator that tracks state in a JSON file.
func New(stateFile string) *Migrator {
	m := &Migrator{
		stateFile: stateFile,
		applied:   make(map[string]Record),
	}
	m.loadState()
	return m
}

// Register adds a migration.
func (m *Migrator) Register(version, description string, up, down func() error) {
	m.migrations = append(m.migrations, Migration{
		Version:     version,
		Description: description,
		Up:          up,
		Down:        down,
	})
}

// Up runs all pending migrations.
func (m *Migrator) Up() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sortMigrations()
	var applied []string

	for _, mig := range m.migrations {
		if _, done := m.applied[mig.Version]; done {
			continue
		}
		if err := mig.Up(); err != nil {
			return applied, fmt.Errorf("migration %s failed: %w", mig.Version, err)
		}
		m.applied[mig.Version] = Record{
			Version:   mig.Version,
			AppliedAt: time.Now(),
		}
		applied = append(applied, mig.Version)
	}

	return applied, m.saveState()
}

// Down rolls back the last applied migration.
func (m *Migrator) Down() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sortMigrations()

	// Find the last applied migration
	var last *Migration
	for i := len(m.migrations) - 1; i >= 0; i-- {
		if _, done := m.applied[m.migrations[i].Version]; done {
			mig := m.migrations[i]
			last = &mig
			break
		}
	}

	if last == nil {
		return "", nil
	}

	if last.Down == nil {
		return "", fmt.Errorf("migration %s has no down function", last.Version)
	}

	if err := last.Down(); err != nil {
		return "", fmt.Errorf("rollback %s failed: %w", last.Version, err)
	}

	delete(m.applied, last.Version)
	return last.Version, m.saveState()
}

// DownTo rolls back to a specific version (exclusive — the target version stays applied).
func (m *Migrator) DownTo(version string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sortMigrations()

	var rolledBack []string
	for i := len(m.migrations) - 1; i >= 0; i-- {
		mig := m.migrations[i]
		if mig.Version == version {
			break
		}
		if _, done := m.applied[mig.Version]; !done {
			continue
		}
		if mig.Down == nil {
			return rolledBack, fmt.Errorf("migration %s has no down function", mig.Version)
		}
		if err := mig.Down(); err != nil {
			return rolledBack, fmt.Errorf("rollback %s failed: %w", mig.Version, err)
		}
		delete(m.applied, mig.Version)
		rolledBack = append(rolledBack, mig.Version)
	}

	return rolledBack, m.saveState()
}

// Status returns the state of all migrations.
func (m *Migrator) Status() []MigrationStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sortMigrations()
	var statuses []MigrationStatus
	for _, mig := range m.migrations {
		s := MigrationStatus{
			Version:     mig.Version,
			Description: mig.Description,
			Applied:     false,
		}
		if rec, ok := m.applied[mig.Version]; ok {
			s.Applied = true
			s.AppliedAt = rec.AppliedAt
		}
		statuses = append(statuses, s)
	}
	return statuses
}

// Pending returns migrations that haven't been applied yet.
func (m *Migrator) Pending() []Migration {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sortMigrations()
	var pending []Migration
	for _, mig := range m.migrations {
		if _, done := m.applied[mig.Version]; !done {
			pending = append(pending, mig)
		}
	}
	return pending
}

// MigrationStatus shows the state of a migration.
type MigrationStatus struct {
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Applied     bool      `json:"applied"`
	AppliedAt   time.Time `json:"applied_at,omitempty"`
}

func (m *Migrator) sortMigrations() {
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})
}

func (m *Migrator) loadState() {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		return
	}
	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return
	}
	for _, r := range records {
		m.applied[r.Version] = r
	}
}

func (m *Migrator) saveState() error {
	if err := os.MkdirAll(filepath.Dir(m.stateFile), 0755); err != nil {
		return err
	}
	records := make([]Record, 0, len(m.applied))
	for _, r := range m.applied {
		records = append(records, r)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Version < records[j].Version
	})
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.stateFile, data, 0644)
}

// --- SQL Migration helpers ---

// SQLMigration is a migration defined as SQL strings.
type SQLMigration struct {
	Version     string
	Description string
	UpSQL       string
	DownSQL     string
}

// RegisterSQL registers a migration from SQL strings.
// The execFn should execute SQL against your database.
func (m *Migrator) RegisterSQL(mig SQLMigration, execFn func(sql string) error) {
	m.Register(mig.Version, mig.Description,
		func() error { return execFn(mig.UpSQL) },
		func() error {
			if mig.DownSQL == "" {
				return nil
			}
			return execFn(mig.DownSQL)
		},
	)
}

// CreateTableSQL generates a CREATE TABLE statement.
func CreateTableSQL(table string, columns map[string]string) string {
	var cols []string
	for name, typedef := range columns {
		cols = append(cols, fmt.Sprintf("  %s %s", name, typedef))
	}
	sort.Strings(cols)
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);",
		table, joinLines(cols))
}

func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		result += l
		if i < len(lines)-1 {
			result += ",\n"
		}
	}
	return result
}

// DropTableSQL generates a DROP TABLE statement.
func DropTableSQL(table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", table)
}
