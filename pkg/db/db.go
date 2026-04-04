// Package db provides a lightweight database abstraction layer.
// Supports SQLite (via database/sql) and includes a built-in JSON file database
// for zero-dependency usage.
package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// --- Interface ---

// DB is the database interface.
type DB interface {
	Collection(name string) Collection
	Close() error
}

// Collection is a table/collection interface.
type Collection interface {
	Insert(doc map[string]interface{}) (string, error)
	FindByID(id string) (map[string]interface{}, error)
	Find(query Query) ([]map[string]interface{}, error)
	Update(id string, doc map[string]interface{}) error
	Delete(id string) error
	Count(query Query) (int, error)
}

// Query defines a database query.
type Query struct {
	Where   map[string]interface{} // exact match conditions
	OrderBy string                 // field name
	Desc    bool                   // descending order
	Limit   int
	Offset  int
}

// --- JSON File Database (zero dependencies) ---

// JSONDB is a file-based JSON database for development and small apps.
type JSONDB struct {
	mu   sync.RWMutex
	dir  string
	data map[string]*jsonCollection
}

type jsonCollection struct {
	mu      sync.RWMutex
	file    string
	records map[string]map[string]interface{}
	nextID  int
}

// NewJSONDB creates a file-based JSON database.
func NewJSONDB(dir string) (*JSONDB, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &JSONDB{
		dir:  dir,
		data: make(map[string]*jsonCollection),
	}, nil
}

func (db *JSONDB) Collection(name string) Collection {
	db.mu.Lock()
	defer db.mu.Unlock()

	if col, ok := db.data[name]; ok {
		return col
	}

	col := &jsonCollection{
		file:    filepath.Join(db.dir, name+".json"),
		records: make(map[string]map[string]interface{}),
		nextID:  1,
	}
	col.load()
	db.data[name] = col
	return col
}

func (db *JSONDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, col := range db.data {
		col.save()
	}
	return nil
}

func (c *jsonCollection) load() {
	data, err := os.ReadFile(c.file)
	if err != nil {
		return
	}
	var records map[string]map[string]interface{}
	if err := json.Unmarshal(data, &records); err != nil {
		return
	}
	c.records = records
	// Find max ID
	for id := range records {
		var n int
		fmt.Sscanf(id, "%d", &n)
		if n >= c.nextID {
			c.nextID = n + 1
		}
	}
}

func (c *jsonCollection) save() error {
	data, err := json.MarshalIndent(c.records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.file, data, 0644)
}

func (c *jsonCollection) Insert(doc map[string]interface{}) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := fmt.Sprintf("%d", c.nextID)
	c.nextID++

	doc["_id"] = id
	doc["_created_at"] = time.Now().Format(time.RFC3339)
	doc["_updated_at"] = time.Now().Format(time.RFC3339)

	c.records[id] = doc
	return id, c.save()
}

func (c *jsonCollection) FindByID(id string) (map[string]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	doc, ok := c.records[id]
	if !ok {
		return nil, errors.New("db: document not found")
	}
	return copyDoc(doc), nil
}

func (c *jsonCollection) Find(q Query) ([]map[string]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []map[string]interface{}
	for _, doc := range c.records {
		if matchesQuery(doc, q.Where) {
			results = append(results, copyDoc(doc))
		}
	}

	// Sort
	if q.OrderBy != "" {
		sort.Slice(results, func(i, j int) bool {
			a := fmt.Sprintf("%v", results[i][q.OrderBy])
			b := fmt.Sprintf("%v", results[j][q.OrderBy])
			if q.Desc {
				return a > b
			}
			return a < b
		})
	}

	// Pagination
	if q.Offset > 0 {
		if q.Offset >= len(results) {
			return nil, nil
		}
		results = results[q.Offset:]
	}
	if q.Limit > 0 && q.Limit < len(results) {
		results = results[:q.Limit]
	}

	return results, nil
}

func (c *jsonCollection) Update(id string, doc map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	existing, ok := c.records[id]
	if !ok {
		return errors.New("db: document not found")
	}

	for k, v := range doc {
		existing[k] = v
	}
	existing["_updated_at"] = time.Now().Format(time.RFC3339)

	return c.save()
}

func (c *jsonCollection) Delete(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.records[id]; !ok {
		return errors.New("db: document not found")
	}
	delete(c.records, id)
	return c.save()
}

func (c *jsonCollection) Count(q Query) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, doc := range c.records {
		if matchesQuery(doc, q.Where) {
			count++
		}
	}
	return count, nil
}

func matchesQuery(doc, where map[string]interface{}) bool {
	if len(where) == 0 {
		return true
	}
	for key, expected := range where {
		actual, ok := doc[key]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
			return false
		}
	}
	return true
}

func copyDoc(doc map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{}, len(doc))
	for k, v := range doc {
		cp[k] = v
	}
	return cp
}

// --- SQL Driver Adapter ---

// SQLConfig holds SQL database configuration.
type SQLConfig struct {
	Driver string // "sqlite3", "postgres", "mysql"
	DSN    string // connection string
}

// SQLAdapter wraps database/sql with the DB interface.
// Usage requires importing the appropriate driver:
//
//	import _ "github.com/mattn/go-sqlite3"
//	db, err := db.NewSQL(db.SQLConfig{Driver: "sqlite3", DSN: "./data.db"})
type SQLAdapter struct {
	config SQLConfig
}

// NewSQL creates a SQL database adapter.
// Note: Actual database/sql usage requires importing a driver.
// This provides the interface and query building.
func NewSQL(cfg SQLConfig) (*SQLAdapter, error) {
	return &SQLAdapter{config: cfg}, nil
}

// --- Query Builder ---

// QueryBuilder helps construct SQL queries safely.
type QueryBuilder struct {
	table  string
	wheres []string
	args   []interface{}
	order  string
	limit  int
	offset int
}

// Table starts a query on a table.
func Table(name string) *QueryBuilder {
	return &QueryBuilder{table: name}
}

// Where adds a condition.
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.wheres = append(qb.wheres, condition)
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy sets the order.
func (qb *QueryBuilder) OrderBy(field string, desc bool) *QueryBuilder {
	dir := "ASC"
	if desc {
		dir = "DESC"
	}
	qb.order = field + " " + dir
	return qb
}

// Limit sets the limit.
func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	qb.limit = n
	return qb
}

// Offset sets the offset.
func (qb *QueryBuilder) Offset(n int) *QueryBuilder {
	qb.offset = n
	return qb
}

// SelectSQL builds a SELECT query string.
func (qb *QueryBuilder) SelectSQL(fields ...string) (string, []interface{}) {
	cols := "*"
	if len(fields) > 0 {
		cols = strings.Join(fields, ", ")
	}
	sql := fmt.Sprintf("SELECT %s FROM %s", cols, qb.table)
	if len(qb.wheres) > 0 {
		sql += " WHERE " + strings.Join(qb.wheres, " AND ")
	}
	if qb.order != "" {
		sql += " ORDER BY " + qb.order
	}
	if qb.limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", qb.limit)
	}
	if qb.offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", qb.offset)
	}
	return sql, qb.args
}

// InsertSQL builds an INSERT query string.
func InsertSQL(table string, data map[string]interface{}) (string, []interface{}) {
	keys := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	i := 1
	for k, v := range data {
		keys = append(keys, k)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, v)
		i++
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table, strings.Join(keys, ", "), strings.Join(placeholders, ", "))
	return sql, values
}

// UpdateSQL builds an UPDATE query string.
func UpdateSQL(table string, id string, data map[string]interface{}) (string, []interface{}) {
	sets := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data)+1)

	i := 1
	for k, v := range data {
		sets = append(sets, fmt.Sprintf("%s = $%d", k, i))
		values = append(values, v)
		i++
	}
	values = append(values, id)

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d",
		table, strings.Join(sets, ", "), i)
	return sql, values
}
