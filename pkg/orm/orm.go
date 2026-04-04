// Package orm provides a lightweight ORM with model definitions, relations,
// auto-migrations, and a fluent query DSL. Zero external dependencies.
package orm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// FieldType represents a database column type.
type FieldType string

const (
	TypeString   FieldType = "TEXT"
	TypeInt      FieldType = "INTEGER"
	TypeFloat    FieldType = "REAL"
	TypeBool     FieldType = "BOOLEAN"
	TypeDateTime FieldType = "DATETIME"
	TypeJSON     FieldType = "JSON"
)

// Field describes a model field / database column.
type Field struct {
	Name       string
	Type       FieldType
	PrimaryKey bool
	Required   bool
	Unique     bool
	Default    interface{}
	Index      bool
	ForeignKey string // "table.column"
}

// Model defines a database model.
type Model struct {
	Name       string
	TableName  string
	Fields     []Field
	Timestamps bool // auto-add created_at, updated_at
}

// Schema holds all registered models.
type Schema struct {
	mu     sync.RWMutex
	models map[string]*Model
}

// NewSchema creates a new schema registry.
func NewSchema() *Schema {
	return &Schema{models: make(map[string]*Model)}
}

// Define registers a model.
func (s *Schema) Define(name string, fields []Field) *Model {
	tableName := strings.ToLower(name) + "s"
	m := &Model{
		Name:       name,
		TableName:  tableName,
		Fields:     fields,
		Timestamps: true,
	}

	// Ensure ID field exists
	hasID := false
	for _, f := range fields {
		if f.PrimaryKey {
			hasID = true
			break
		}
	}
	if !hasID {
		m.Fields = append([]Field{{
			Name:       "id",
			Type:       TypeInt,
			PrimaryKey: true,
		}}, m.Fields...)
	}

	s.mu.Lock()
	s.models[name] = m
	s.mu.Unlock()

	return m
}

// Get returns a model by name.
func (s *Schema) Get(name string) *Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.models[name]
}

// Models returns all registered models.
func (s *Schema) Models() []*Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var models []*Model
	for _, m := range s.models {
		models = append(models, m)
	}
	return models
}

// CreateTableSQL generates SQL to create a table.
func (m *Model) CreateTableSQL() string {
	var cols []string
	for _, f := range m.Fields {
		col := f.Name + " " + string(f.Type)
		if f.PrimaryKey {
			col += " PRIMARY KEY"
			if f.Type == TypeInt {
				col += " AUTOINCREMENT"
			}
		}
		if f.Required && !f.PrimaryKey {
			col += " NOT NULL"
		}
		if f.Unique {
			col += " UNIQUE"
		}
		if f.Default != nil {
			col += fmt.Sprintf(" DEFAULT %v", f.Default)
		}
		cols = append(cols, "  "+col)
	}
	if m.Timestamps {
		cols = append(cols, "  created_at DATETIME DEFAULT CURRENT_TIMESTAMP")
		cols = append(cols, "  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP")
	}

	var indexes []string
	for _, f := range m.Fields {
		if f.Index {
			indexes = append(indexes, fmt.Sprintf(
				"CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s (%s);",
				m.TableName, f.Name, m.TableName, f.Name,
			))
		}
		if f.ForeignKey != "" {
			parts := strings.SplitN(f.ForeignKey, ".", 2)
			if len(parts) == 2 {
				cols = append(cols, fmt.Sprintf(
					"  FOREIGN KEY (%s) REFERENCES %s(%s)",
					f.Name, parts[0], parts[1],
				))
			}
		}
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);",
		m.TableName, strings.Join(cols, ",\n"))

	if len(indexes) > 0 {
		sql += "\n" + strings.Join(indexes, "\n")
	}

	return sql
}

// DropTableSQL generates SQL to drop a table.
func (m *Model) DropTableSQL() string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", m.TableName)
}

// --- Query DSL ---

// QueryDSL is a fluent query builder for models.
type QueryDSL struct {
	model      *Model
	conditions []condition
	orderBy    string
	orderDesc  bool
	limitVal   int
	offsetVal  int
	selectCols []string
	joins      []string
}

type condition struct {
	field string
	op    string
	value interface{}
}

// Query starts a query on a model.
func (m *Model) Query() *QueryDSL {
	return &QueryDSL{model: m}
}

// Select specifies which columns to return.
func (q *QueryDSL) Select(cols ...string) *QueryDSL {
	q.selectCols = cols
	return q
}

// Where adds a condition.
func (q *QueryDSL) Where(field, op string, value interface{}) *QueryDSL {
	q.conditions = append(q.conditions, condition{field, op, value})
	return q
}

// Eq is shorthand for Where(field, "=", value).
func (q *QueryDSL) Eq(field string, value interface{}) *QueryDSL {
	return q.Where(field, "=", value)
}

// Gt is shorthand for Where(field, ">", value).
func (q *QueryDSL) Gt(field string, value interface{}) *QueryDSL {
	return q.Where(field, ">", value)
}

// Lt is shorthand for Where(field, "<", value).
func (q *QueryDSL) Lt(field string, value interface{}) *QueryDSL {
	return q.Where(field, "<", value)
}

// Like adds a LIKE condition.
func (q *QueryDSL) Like(field, pattern string) *QueryDSL {
	return q.Where(field, "LIKE", pattern)
}

// In adds an IN condition.
func (q *QueryDSL) In(field string, values ...interface{}) *QueryDSL {
	return q.Where(field, "IN", values)
}

// OrderBy sets the order.
func (q *QueryDSL) OrderBy(field string) *QueryDSL {
	q.orderBy = field
	q.orderDesc = false
	return q
}

// OrderByDesc sets descending order.
func (q *QueryDSL) OrderByDesc(field string) *QueryDSL {
	q.orderBy = field
	q.orderDesc = true
	return q
}

// Limit sets the result limit.
func (q *QueryDSL) Limit(n int) *QueryDSL {
	q.limitVal = n
	return q
}

// Offset sets the result offset.
func (q *QueryDSL) Offset(n int) *QueryDSL {
	q.offsetVal = n
	return q
}

// Join adds a JOIN clause.
func (q *QueryDSL) Join(table, on string) *QueryDSL {
	q.joins = append(q.joins, fmt.Sprintf("JOIN %s ON %s", table, on))
	return q
}

// LeftJoin adds a LEFT JOIN clause.
func (q *QueryDSL) LeftJoin(table, on string) *QueryDSL {
	q.joins = append(q.joins, fmt.Sprintf("LEFT JOIN %s ON %s", table, on))
	return q
}

// ToSQL builds the SQL query string and arguments.
func (q *QueryDSL) ToSQL() (string, []interface{}) {
	cols := "*"
	if len(q.selectCols) > 0 {
		cols = strings.Join(q.selectCols, ", ")
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", cols, q.model.TableName)

	if len(q.joins) > 0 {
		sql += " " + strings.Join(q.joins, " ")
	}

	var args []interface{}
	if len(q.conditions) > 0 {
		var wheres []string
		for _, c := range q.conditions {
			idx := len(args) + 1
			if c.op == "IN" {
				if vals, ok := c.value.([]interface{}); ok {
					placeholders := make([]string, len(vals))
					for i, v := range vals {
						placeholders[i] = fmt.Sprintf("$%d", idx+i)
						args = append(args, v)
					}
					wheres = append(wheres, fmt.Sprintf("%s IN (%s)", c.field, strings.Join(placeholders, ", ")))
					continue
				}
			}
			wheres = append(wheres, fmt.Sprintf("%s %s $%d", c.field, c.op, idx))
			args = append(args, c.value)
		}
		sql += " WHERE " + strings.Join(wheres, " AND ")
	}

	if q.orderBy != "" {
		dir := "ASC"
		if q.orderDesc {
			dir = "DESC"
		}
		sql += fmt.Sprintf(" ORDER BY %s %s", q.orderBy, dir)
	}

	if q.limitVal > 0 {
		sql += fmt.Sprintf(" LIMIT %d", q.limitVal)
	}
	if q.offsetVal > 0 {
		sql += fmt.Sprintf(" OFFSET %d", q.offsetVal)
	}

	return sql, args
}

// CountSQL builds a COUNT query.
func (q *QueryDSL) CountSQL() (string, []interface{}) {
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s", q.model.TableName)
	var args []interface{}
	if len(q.conditions) > 0 {
		var wheres []string
		for _, c := range q.conditions {
			idx := len(args) + 1
			wheres = append(wheres, fmt.Sprintf("%s %s $%d", c.field, c.op, idx))
			args = append(args, c.value)
		}
		sql += " WHERE " + strings.Join(wheres, " AND ")
	}
	return sql, args
}

// InsertSQL builds an INSERT statement for the model.
func (m *Model) InsertSQL(data map[string]interface{}) (string, []interface{}) {
	var cols []string
	var placeholders []string
	var args []interface{}
	i := 1
	for _, f := range m.Fields {
		if f.PrimaryKey && f.Type == TypeInt {
			continue // auto-increment
		}
		if v, ok := data[f.Name]; ok {
			cols = append(cols, f.Name)
			placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			args = append(args, v)
			i++
		}
	}
	if m.Timestamps {
		now := time.Now().Format(time.RFC3339)
		cols = append(cols, "created_at", "updated_at")
		placeholders = append(placeholders, fmt.Sprintf("$%d", i), fmt.Sprintf("$%d", i+1))
		args = append(args, now, now)
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		m.TableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	return sql, args
}

// UpdateSQL builds an UPDATE statement for the model.
func (m *Model) UpdateSQL(id interface{}, data map[string]interface{}) (string, []interface{}) {
	var sets []string
	var args []interface{}
	i := 1
	for _, f := range m.Fields {
		if f.PrimaryKey {
			continue
		}
		if v, ok := data[f.Name]; ok {
			sets = append(sets, fmt.Sprintf("%s = $%d", f.Name, i))
			args = append(args, v)
			i++
		}
	}
	if m.Timestamps {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, time.Now().Format(time.RFC3339))
		i++
	}
	args = append(args, id)

	pk := "id"
	for _, f := range m.Fields {
		if f.PrimaryKey {
			pk = f.Name
			break
		}
	}

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		m.TableName, strings.Join(sets, ", "), pk, i)
	return sql, args
}

// DeleteSQL builds a DELETE statement.
func (m *Model) DeleteSQL(id interface{}) (string, []interface{}) {
	pk := "id"
	for _, f := range m.Fields {
		if f.PrimaryKey {
			pk = f.Name
			break
		}
	}
	return fmt.Sprintf("DELETE FROM %s WHERE %s = $1", m.TableName, pk), []interface{}{id}
}

// --- JSON-based ORM (zero deps, for dev/small apps) ---

// JSONORM is a file-based ORM using JSON storage.
type JSONORM struct {
	mu      sync.RWMutex
	dir     string
	schema  *Schema
	tables  map[string]*jsonTable
}

type jsonTable struct {
	mu      sync.RWMutex
	file    string
	records []map[string]interface{}
	nextID  int
}

// NewJSONORM creates a file-based ORM.
func NewJSONORM(dir string, schema *Schema) (*JSONORM, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	orm := &JSONORM{
		dir:    dir,
		schema: schema,
		tables: make(map[string]*jsonTable),
	}
	// Auto-create tables for all models
	for _, m := range schema.Models() {
		orm.ensureTable(m)
	}
	return orm, nil
}

func (o *JSONORM) ensureTable(m *Model) *jsonTable {
	o.mu.Lock()
	defer o.mu.Unlock()

	if t, ok := o.tables[m.TableName]; ok {
		return t
	}

	t := &jsonTable{
		file:   filepath.Join(o.dir, m.TableName+".json"),
		nextID: 1,
	}
	t.load()
	o.tables[m.TableName] = t
	return t
}

func (t *jsonTable) load() {
	data, err := os.ReadFile(t.file)
	if err != nil {
		return
	}
	json.Unmarshal(data, &t.records)
	for _, r := range t.records {
		if id, ok := r["id"]; ok {
			if n, ok := id.(float64); ok && int(n) >= t.nextID {
				t.nextID = int(n) + 1
			}
		}
	}
}

func (t *jsonTable) save() error {
	data, err := json.MarshalIndent(t.records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.file, data, 0644)
}

// Create inserts a record.
func (o *JSONORM) Create(modelName string, data map[string]interface{}) (int, error) {
	m := o.schema.Get(modelName)
	if m == nil {
		return 0, fmt.Errorf("orm: model %q not found", modelName)
	}

	// Validate required fields
	for _, f := range m.Fields {
		if f.Required && !f.PrimaryKey {
			if _, ok := data[f.Name]; !ok {
				if f.Default != nil {
					data[f.Name] = f.Default
				} else {
					return 0, fmt.Errorf("orm: field %q is required", f.Name)
				}
			}
		}
	}

	t := o.ensureTable(m)
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check unique constraints
	for _, f := range m.Fields {
		if f.Unique {
			if v, ok := data[f.Name]; ok {
				for _, r := range t.records {
					if fmt.Sprintf("%v", r[f.Name]) == fmt.Sprintf("%v", v) {
						return 0, fmt.Errorf("orm: duplicate value for unique field %q", f.Name)
					}
				}
			}
		}
	}

	id := t.nextID
	t.nextID++
	data["id"] = id
	now := time.Now().Format(time.RFC3339)
	if m.Timestamps {
		data["created_at"] = now
		data["updated_at"] = now
	}

	t.records = append(t.records, data)
	return id, t.save()
}

// FindByID finds a record by primary key.
func (o *JSONORM) FindByID(modelName string, id int) (map[string]interface{}, error) {
	m := o.schema.Get(modelName)
	if m == nil {
		return nil, fmt.Errorf("orm: model %q not found", modelName)
	}
	t := o.ensureTable(m)
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, r := range t.records {
		if rid, ok := r["id"]; ok {
			if toInt(rid) == id {
				return copyRecord(r), nil
			}
		}
	}
	return nil, errors.New("orm: record not found")
}

// FindAll returns all records matching conditions.
func (o *JSONORM) FindAll(modelName string, where map[string]interface{}, orderBy string, desc bool, limit, offset int) ([]map[string]interface{}, error) {
	m := o.schema.Get(modelName)
	if m == nil {
		return nil, fmt.Errorf("orm: model %q not found", modelName)
	}
	t := o.ensureTable(m)
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []map[string]interface{}
	for _, r := range t.records {
		if matchWhere(r, where) {
			results = append(results, copyRecord(r))
		}
	}

	if orderBy != "" {
		sort.Slice(results, func(i, j int) bool {
			a := fmt.Sprintf("%v", results[i][orderBy])
			b := fmt.Sprintf("%v", results[j][orderBy])
			if desc {
				return a > b
			}
			return a < b
		})
	}

	if offset > 0 && offset < len(results) {
		results = results[offset:]
	}
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results, nil
}

// Update updates a record.
func (o *JSONORM) Update(modelName string, id int, data map[string]interface{}) error {
	m := o.schema.Get(modelName)
	if m == nil {
		return fmt.Errorf("orm: model %q not found", modelName)
	}
	t := o.ensureTable(m)
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, r := range t.records {
		if toInt(r["id"]) == id {
			for k, v := range data {
				if k != "id" && k != "created_at" {
					r[k] = v
				}
			}
			if m.Timestamps {
				r["updated_at"] = time.Now().Format(time.RFC3339)
			}
			return t.save()
		}
	}
	return errors.New("orm: record not found")
}

// Delete removes a record.
func (o *JSONORM) Delete(modelName string, id int) error {
	m := o.schema.Get(modelName)
	if m == nil {
		return fmt.Errorf("orm: model %q not found", modelName)
	}
	t := o.ensureTable(m)
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, r := range t.records {
		if toInt(r["id"]) == id {
			t.records = append(t.records[:i], t.records[i+1:]...)
			return t.save()
		}
	}
	return errors.New("orm: record not found")
}

// Count returns the number of matching records.
func (o *JSONORM) Count(modelName string, where map[string]interface{}) (int, error) {
	m := o.schema.Get(modelName)
	if m == nil {
		return 0, fmt.Errorf("orm: model %q not found", modelName)
	}
	t := o.ensureTable(m)
	t.mu.RLock()
	defer t.mu.RUnlock()

	count := 0
	for _, r := range t.records {
		if matchWhere(r, where) {
			count++
		}
	}
	return count, nil
}

// Close persists all tables.
func (o *JSONORM) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, t := range o.tables {
		t.save()
	}
	return nil
}

// --- Helpers ---

func matchWhere(record, where map[string]interface{}) bool {
	if len(where) == 0 {
		return true
	}
	for k, v := range where {
		rv, ok := record[k]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", rv) != fmt.Sprintf("%v", v) {
			return false
		}
	}
	return true
}

func copyRecord(r map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{}, len(r))
	for k, v := range r {
		cp[k] = v
	}
	return cp
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}

// StructToMap converts a struct to map[string]interface{} using reflect.
func StructToMap(v interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return result
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			tag = strings.ToLower(field.Name)
		}
		tag = strings.SplitN(tag, ",", 2)[0]
		result[tag] = val.Field(i).Interface()
	}
	return result
}
