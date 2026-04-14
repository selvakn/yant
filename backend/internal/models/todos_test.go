package models_test

import (
	"testing"

	"github.com/selvakn/yant/internal/models"
)

func TestParseTodos(t *testing.T) {
	body := `# My Note

Some text here.

- [ ] Buy groceries @due(2026-04-20)
- [x] Send email @due(2026-04-10)
- [ ] Task without date
- Not a todo
- [ ] Another task @due(2026-05-01) with extra text
`
	todos := models.ParseTodos(body)

	if len(todos) != 4 {
		t.Fatalf("expected 4 todos, got %d", len(todos))
	}

	// First todo
	if todos[0].Line != 5 {
		t.Errorf("todo 0: expected line 5, got %d", todos[0].Line)
	}
	if todos[0].Text != "Buy groceries" {
		t.Errorf("todo 0: expected 'Buy groceries', got '%s'", todos[0].Text)
	}
	if todos[0].DueDate != "2026-04-20" {
		t.Errorf("todo 0: expected due 2026-04-20, got '%s'", todos[0].DueDate)
	}
	if todos[0].Completed {
		t.Error("todo 0: should not be completed")
	}

	// Completed todo
	if !todos[1].Completed {
		t.Error("todo 1: should be completed")
	}
	if todos[1].DueDate != "2026-04-10" {
		t.Errorf("todo 1: expected due 2026-04-10, got '%s'", todos[1].DueDate)
	}

	// No date
	if todos[2].DueDate != "" {
		t.Errorf("todo 2: expected no due date, got '%s'", todos[2].DueDate)
	}
	if todos[2].Text != "Task without date" {
		t.Errorf("todo 2: expected 'Task without date', got '%s'", todos[2].Text)
	}

	// Due date with extra text after
	if todos[3].Text != "Another task with extra text" {
		t.Errorf("todo 3: expected 'Another task with extra text', got '%s'", todos[3].Text)
	}
	if todos[3].DueDate != "2026-05-01" {
		t.Errorf("todo 3: expected due 2026-05-01, got '%s'", todos[3].DueDate)
	}
}

func TestParseTodosEmpty(t *testing.T) {
	todos := models.ParseTodos("No todos here\nJust text")
	if len(todos) != 0 {
		t.Fatalf("expected 0 todos, got %d", len(todos))
	}
}

func TestParseTodosMalformedDate(t *testing.T) {
	body := "- [ ] Task @due(not-a-date)\n"
	todos := models.ParseTodos(body)
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	// Malformed date doesn't match regex, so it stays in the text
	if todos[0].DueDate != "" {
		t.Errorf("expected no due date for malformed, got '%s'", todos[0].DueDate)
	}
	if todos[0].Text != "Task @due(not-a-date)" {
		t.Errorf("unexpected text: '%s'", todos[0].Text)
	}
}

func TestSyncTodos(t *testing.T) {
	db := openTestDB(t)

	// Create user and note
	_, err := db.Exec("INSERT INTO users (username, created_at) VALUES ('testuser', datetime('now'))")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO notes (user_id, slug, title, created_at, updated_at) VALUES (1, 'test', 'Test', datetime('now'), datetime('now'))")
	if err != nil {
		t.Fatal(err)
	}

	todos := []models.TodoItem{
		{Line: 3, Text: "Buy groceries", DueDate: "2026-04-20", Completed: false},
		{Line: 5, Text: "Send email", DueDate: "", Completed: true},
	}

	if err := models.SyncTodos(db, 1, todos); err != nil {
		t.Fatal(err)
	}

	// Verify
	rows, err := db.Query("SELECT line, text, due_date, completed FROM note_todos WHERE note_id = 1 ORDER BY line")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var line int
		var text string
		var dueDate *string
		var completed bool
		if err := rows.Scan(&line, &text, &dueDate, &completed); err != nil {
			t.Fatal(err)
		}
		count++
		if count == 1 {
			if line != 3 || text != "Buy groceries" {
				t.Errorf("unexpected first todo: line=%d text=%s", line, text)
			}
			if dueDate == nil || *dueDate != "2026-04-20" {
				t.Error("expected due date 2026-04-20")
			}
		}
	}
	if count != 2 {
		t.Errorf("expected 2 todos in DB, got %d", count)
	}

	// Re-sync with different data (should replace)
	todos2 := []models.TodoItem{
		{Line: 1, Text: "New task", DueDate: "", Completed: false},
	}
	if err := models.SyncTodos(db, 1, todos2); err != nil {
		t.Fatal(err)
	}

	var newCount int
	db.QueryRow("SELECT COUNT(*) FROM note_todos WHERE note_id = 1").Scan(&newCount)
	if newCount != 1 {
		t.Errorf("expected 1 todo after re-sync, got %d", newCount)
	}
}
