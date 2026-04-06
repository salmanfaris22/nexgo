package orm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSchema(t *testing.T) {
	s := NewSchema()
	if s == nil {
		t.Fatal("expected schema")
	}
}

func TestDefine(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{
		{Name: "name", Type: TypeString, Required: true},
		{Name: "email", Type: TypeString, Unique: true},
	})

	if m.Name != "User" {
		t.Errorf("expected User, got %s", m.Name)
	}
	if m.TableName != "users" {
		t.Errorf("expected users, got %s", m.TableName)
	}
	if len(m.Fields) < 3 {
		t.Errorf("expected at least 3 fields (id + 2), got %d", len(m.Fields))
	}
	if !m.Fields[0].PrimaryKey {
		t.Error("expected first field to be primary key (auto-added id)")
	}
}

func TestDefine_WithPrimaryKey(t *testing.T) {
	s := NewSchema()
	m := s.Define("Post", []Field{
		{Name: "uuid", Type: TypeString, PrimaryKey: true},
		{Name: "title", Type: TypeString},
	})

	if len(m.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(m.Fields))
	}
}

func TestGet(t *testing.T) {
	s := NewSchema()
	s.Define("User", []Field{{Name: "name", Type: TypeString}})

	m := s.Get("User")
	if m == nil {
		t.Error("expected model")
	}
	if m.Name != "User" {
		t.Errorf("expected User, got %s", m.Name)
	}
}

func TestGet_NotFound(t *testing.T) {
	s := NewSchema()
	m := s.Get("Missing")
	if m != nil {
		t.Error("expected nil for missing model")
	}
}

func TestModels(t *testing.T) {
	s := NewSchema()
	s.Define("User", nil)
	s.Define("Post", nil)

	models := s.Models()
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
}

func TestCreateTableSQL(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{
		{Name: "name", Type: TypeString, Required: true},
		{Name: "age", Type: TypeInt},
	})

	sql := m.CreateTableSQL()

	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS users") {
		t.Error("expected CREATE TABLE statement")
	}
	if !strings.Contains(sql, "name TEXT NOT NULL") {
		t.Error("expected name column with NOT NULL")
	}
	if !strings.Contains(sql, "age INTEGER") {
		t.Error("expected age column")
	}
	if !strings.Contains(sql, "created_at DATETIME") {
		t.Error("expected created_at timestamp")
	}
	if !strings.Contains(sql, "updated_at DATETIME") {
		t.Error("expected updated_at timestamp")
	}
}

func TestCreateTableSQL_Unique(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{
		{Name: "email", Type: TypeString, Unique: true},
	})

	sql := m.CreateTableSQL()
	if !strings.Contains(sql, "UNIQUE") {
		t.Error("expected UNIQUE constraint")
	}
}

func TestCreateTableSQL_Index(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{
		{Name: "email", Type: TypeString, Index: true},
	})

	sql := m.CreateTableSQL()
	if !strings.Contains(sql, "CREATE INDEX") {
		t.Error("expected CREATE INDEX statement")
	}
}

func TestCreateTableSQL_ForeignKey(t *testing.T) {
	s := NewSchema()
	m := s.Define("Post", []Field{
		{Name: "user_id", Type: TypeInt, ForeignKey: "users.id"},
	})

	sql := m.CreateTableSQL()
	if !strings.Contains(sql, "FOREIGN KEY") {
		t.Error("expected FOREIGN KEY constraint")
	}
}

func TestDropTableSQL(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", nil)

	sql := m.DropTableSQL()
	if sql != "DROP TABLE IF EXISTS users;" {
		t.Errorf("unexpected SQL: %s", sql)
	}
}

func TestQueryDSL(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{
		{Name: "name", Type: TypeString},
		{Name: "age", Type: TypeInt},
	})

	sql, args := m.Query().
		Select("name", "age").
		Where("age", ">", 18).
		OrderBy("name").
		Limit(10).
		Offset(5).
		ToSQL()

	if !strings.Contains(sql, "SELECT name, age FROM users") {
		t.Errorf("unexpected SQL: %s", sql)
	}
	if !strings.Contains(sql, "WHERE age > $1") {
		t.Errorf("expected WHERE clause: %s", sql)
	}
	if !strings.Contains(sql, "ORDER BY name ASC") {
		t.Errorf("expected ORDER BY: %s", sql)
	}
	if !strings.Contains(sql, "LIMIT 10") {
		t.Errorf("expected LIMIT: %s", sql)
	}
	if !strings.Contains(sql, "OFFSET 5") {
		t.Errorf("expected OFFSET: %s", sql)
	}
	if len(args) != 1 || args[0] != 18 {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestQueryDSL_EqGtLt(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{
		{Name: "name", Type: TypeString},
	})

	sql, _ := m.Query().Eq("name", "John").ToSQL()
	if !strings.Contains(sql, "name = $1") {
		t.Errorf("expected eq: %s", sql)
	}

	sql, _ = m.Query().Gt("age", 18).ToSQL()
	if !strings.Contains(sql, "age > $1") {
		t.Errorf("expected gt: %s", sql)
	}

	sql, _ = m.Query().Lt("age", 65).ToSQL()
	if !strings.Contains(sql, "age < $1") {
		t.Errorf("expected lt: %s", sql)
	}
}

func TestQueryDSL_Like(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{{Name: "name", Type: TypeString}})

	sql, _ := m.Query().Like("name", "%John%").ToSQL()
	if !strings.Contains(sql, "name LIKE $1") {
		t.Errorf("expected LIKE: %s", sql)
	}
}

func TestQueryDSL_In(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{{Name: "id", Type: TypeInt}})

	sql, args := m.Query().In("id", 1, 2, 3).ToSQL()
	if !strings.Contains(sql, "IN ($1, $2, $3)") {
		t.Errorf("expected IN clause: %s", sql)
	}
	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d", len(args))
	}
}

func TestQueryDSL_OrderByDesc(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", nil)

	sql, _ := m.Query().OrderByDesc("created_at").ToSQL()
	if !strings.Contains(sql, "ORDER BY created_at DESC") {
		t.Errorf("expected DESC: %s", sql)
	}
}

func TestQueryDSL_Join(t *testing.T) {
	s := NewSchema()
	m := s.Define("Post", []Field{{Name: "title", Type: TypeString}})

	sql, _ := m.Query().Join("users", "posts.user_id = users.id").ToSQL()
	if !strings.Contains(sql, "JOIN users ON posts.user_id = users.id") {
		t.Errorf("expected JOIN: %s", sql)
	}
}

func TestQueryDSL_LeftJoin(t *testing.T) {
	s := NewSchema()
	m := s.Define("Post", nil)

	sql, _ := m.Query().LeftJoin("users", "posts.user_id = users.id").ToSQL()
	if !strings.Contains(sql, "LEFT JOIN users") {
		t.Errorf("expected LEFT JOIN: %s", sql)
	}
}

func TestCountSQL(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{{Name: "name", Type: TypeString}})

	sql, args := m.Query().Where("name", "=", "John").CountSQL()
	if !strings.Contains(sql, "SELECT COUNT(*) FROM users") {
		t.Errorf("expected COUNT: %s", sql)
	}
	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
}

func TestInsertSQL(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{
		{Name: "name", Type: TypeString},
	})

	sql, args := m.InsertSQL(map[string]interface{}{"name": "John"})
	if !strings.Contains(sql, "INSERT INTO users") {
		t.Errorf("expected INSERT: %s", sql)
	}
	if len(args) < 1 {
		t.Errorf("expected args")
	}
}

func TestUpdateSQL(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", []Field{
		{Name: "name", Type: TypeString},
	})

	sql, args := m.UpdateSQL(1, map[string]interface{}{"name": "Jane"})
	if !strings.Contains(sql, "UPDATE users SET") {
		t.Errorf("expected UPDATE: %s", sql)
	}
	if len(args) < 2 {
		t.Errorf("expected at least 2 args, got %d", len(args))
	}
}

func TestDeleteSQL(t *testing.T) {
	s := NewSchema()
	m := s.Define("User", nil)

	sql, args := m.DeleteSQL(1)
	if sql != "DELETE FROM users WHERE id = $1" {
		t.Errorf("unexpected SQL: %s", sql)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestJSONORM_CRUD(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", []Field{
		{Name: "name", Type: TypeString, Required: true},
	})

	orm, err := NewJSONORM(dir, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create
	id, err := orm.Create("User", map[string]interface{}{"name": "John"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if id != 1 {
		t.Errorf("expected id 1, got %d", id)
	}

	// FindByID
	record, err := orm.FindByID("User", id)
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if record["name"] != "John" {
		t.Errorf("expected name John, got %v", record["name"])
	}

	// Update
	err = orm.Update("User", id, map[string]interface{}{"name": "Jane"})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	record, _ = orm.FindByID("User", id)
	if record["name"] != "Jane" {
		t.Errorf("expected name Jane, got %v", record["name"])
	}

	// Delete
	err = orm.Delete("User", id)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	_, err = orm.FindByID("User", id)
	if err == nil {
		t.Error("expected not found after delete")
	}
}

func TestJSONORM_FindAll(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", []Field{{Name: "name", Type: TypeString}})

	orm, _ := NewJSONORM(dir, s)
	orm.Create("User", map[string]interface{}{"name": "Alice"})
	orm.Create("User", map[string]interface{}{"name": "Bob"})
	orm.Create("User", map[string]interface{}{"name": "Charlie"})

	results, err := orm.FindAll("User", nil, "name", false, 0, 0)
	if err != nil {
		t.Fatalf("find all failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestJSONORM_FindAll_WithWhere(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", []Field{{Name: "name", Type: TypeString}})

	orm, _ := NewJSONORM(dir, s)
	orm.Create("User", map[string]interface{}{"name": "Alice"})
	orm.Create("User", map[string]interface{}{"name": "Bob"})

	results, err := orm.FindAll("User", map[string]interface{}{"name": "Alice"}, "", false, 0, 0)
	if err != nil {
		t.Fatalf("find all failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestJSONORM_FindAll_LimitOffset(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", []Field{{Name: "name", Type: TypeString}})

	orm, _ := NewJSONORM(dir, s)
	for i := 0; i < 5; i++ {
		orm.Create("User", map[string]interface{}{"name": "User"})
	}

	results, err := orm.FindAll("User", nil, "", false, 2, 1)
	if err != nil {
		t.Fatalf("find all failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestJSONORM_Count(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", []Field{{Name: "name", Type: TypeString}})

	orm, _ := NewJSONORM(dir, s)
	orm.Create("User", map[string]interface{}{"name": "Alice"})
	orm.Create("User", map[string]interface{}{"name": "Bob"})

	count, err := orm.Count("User", nil)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestJSONORM_RequiredField(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", []Field{{Name: "name", Type: TypeString, Required: true}})

	orm, _ := NewJSONORM(dir, s)
	_, err := orm.Create("User", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing required field")
	}
}

func TestJSONORM_UniqueConstraint(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", []Field{{Name: "email", Type: TypeString, Unique: true}})

	orm, _ := NewJSONORM(dir, s)
	orm.Create("User", map[string]interface{}{"email": "test@example.com"})
	_, err := orm.Create("User", map[string]interface{}{"email": "test@example.com"})
	if err == nil {
		t.Error("expected error for duplicate unique field")
	}
}

func TestJSONORM_ModelNotFound(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	orm, _ := NewJSONORM(dir, s)

	_, err := orm.Create("Missing", nil)
	if err == nil {
		t.Error("expected error for missing model")
	}
}

func TestJSONORM_FindByIDNotFound(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", nil)
	orm, _ := NewJSONORM(dir, s)

	_, err := orm.FindByID("User", 999)
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestJSONORM_UpdateNotFound(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", nil)
	orm, _ := NewJSONORM(dir, s)

	err := orm.Update("User", 999, map[string]interface{}{"name": "test"})
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestJSONORM_DeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", nil)
	orm, _ := NewJSONORM(dir, s)

	err := orm.Delete("User", 999)
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestJSONORM_Close(t *testing.T) {
	dir := t.TempDir()
	s := NewSchema()
	s.Define("User", []Field{{Name: "name", Type: TypeString}})

	orm, _ := NewJSONORM(dir, s)
	orm.Create("User", map[string]interface{}{"name": "John"})

	err := orm.Close()
	if err != nil {
		t.Errorf("close failed: %v", err)
	}

	// Verify data was persisted
	data, err := os.ReadFile(filepath.Join(dir, "users.json"))
	if err != nil {
		t.Fatalf("failed to read persisted data: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty persisted data")
	}
}

func TestStructToMap(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	m := StructToMap(Person{Name: "John", Age: 30})
	if m["name"] != "John" {
		t.Errorf("expected name John, got %v", m["name"])
	}
	if m["age"] != 30 {
		t.Errorf("expected age 30, got %v", m["age"])
	}
}

func TestStructToMap_Pointer(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
	}

	p := &Person{Name: "Jane"}
	m := StructToMap(p)
	if m["name"] != "Jane" {
		t.Errorf("expected name Jane, got %v", m["name"])
	}
}

func TestStructToMap_NonStruct(t *testing.T) {
	m := StructToMap("not a struct")
	if len(m) != 0 {
		t.Errorf("expected empty map for non-struct, got %v", m)
	}
}
