package db

import (
	"testing"
)

func TestJSONDBInsertAndFind(t *testing.T) {
	dir := t.TempDir()
	database, err := NewJSONDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	users := database.Collection("users")

	id, err := users.Insert(map[string]interface{}{
		"name":  "John",
		"email": "john@example.com",
		"role":  "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("ID should not be empty")
	}

	user, err := users.FindByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if user["name"] != "John" {
		t.Errorf("name = %v, want John", user["name"])
	}
	if user["email"] != "john@example.com" {
		t.Errorf("email = %v, want john@example.com", user["email"])
	}
}

func TestJSONDBQuery(t *testing.T) {
	dir := t.TempDir()
	database, _ := NewJSONDB(dir)
	defer database.Close()

	users := database.Collection("users")
	users.Insert(map[string]interface{}{"name": "Alice", "role": "admin"})
	users.Insert(map[string]interface{}{"name": "Bob", "role": "user"})
	users.Insert(map[string]interface{}{"name": "Charlie", "role": "admin"})

	admins, err := users.Find(Query{
		Where: map[string]interface{}{"role": "admin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(admins) != 2 {
		t.Errorf("found %d admins, want 2", len(admins))
	}
}

func TestJSONDBUpdate(t *testing.T) {
	dir := t.TempDir()
	database, _ := NewJSONDB(dir)
	defer database.Close()

	users := database.Collection("users")
	id, _ := users.Insert(map[string]interface{}{"name": "John"})

	users.Update(id, map[string]interface{}{"name": "John Doe"})

	user, _ := users.FindByID(id)
	if user["name"] != "John Doe" {
		t.Errorf("name = %v, want John Doe", user["name"])
	}
}

func TestJSONDBDelete(t *testing.T) {
	dir := t.TempDir()
	database, _ := NewJSONDB(dir)
	defer database.Close()

	users := database.Collection("users")
	id, _ := users.Insert(map[string]interface{}{"name": "John"})

	if err := users.Delete(id); err != nil {
		t.Fatal(err)
	}

	_, err := users.FindByID(id)
	if err == nil {
		t.Error("should return error for deleted doc")
	}
}

func TestJSONDBCount(t *testing.T) {
	dir := t.TempDir()
	database, _ := NewJSONDB(dir)
	defer database.Close()

	users := database.Collection("users")
	users.Insert(map[string]interface{}{"role": "admin"})
	users.Insert(map[string]interface{}{"role": "user"})
	users.Insert(map[string]interface{}{"role": "admin"})

	count, _ := users.Count(Query{Where: map[string]interface{}{"role": "admin"}})
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	total, _ := users.Count(Query{})
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}

func TestJSONDBPagination(t *testing.T) {
	dir := t.TempDir()
	database, _ := NewJSONDB(dir)
	defer database.Close()

	items := database.Collection("items")
	for i := 0; i < 10; i++ {
		items.Insert(map[string]interface{}{"idx": i})
	}

	page, _ := items.Find(Query{Limit: 3})
	if len(page) != 3 {
		t.Errorf("page len = %d, want 3", len(page))
	}

	page2, _ := items.Find(Query{Offset: 8, Limit: 5})
	if len(page2) != 2 {
		t.Errorf("page2 len = %d, want 2", len(page2))
	}
}

func TestQueryBuilder(t *testing.T) {
	sql, args := Table("users").
		Where("role = $1", "admin").
		Where("active = $2", true).
		OrderBy("name", false).
		Limit(10).
		Offset(20).
		SelectSQL("id", "name")

	if sql != "SELECT id, name FROM users WHERE role = $1 AND active = $2 ORDER BY name ASC LIMIT 10 OFFSET 20" {
		t.Errorf("SQL = %q", sql)
	}
	if len(args) != 2 {
		t.Errorf("args = %d, want 2", len(args))
	}
}

func TestInsertSQL(t *testing.T) {
	sql, args := InsertSQL("users", map[string]interface{}{
		"name": "John",
	})
	if sql == "" {
		t.Error("SQL should not be empty")
	}
	if len(args) != 1 {
		t.Errorf("args = %d, want 1", len(args))
	}
}

func TestFindByIDNotFound(t *testing.T) {
	dir := t.TempDir()
	database, _ := NewJSONDB(dir)
	defer database.Close()

	users := database.Collection("users")
	_, err := users.FindByID("nonexistent")
	if err == nil {
		t.Error("should return error for missing doc")
	}
}
