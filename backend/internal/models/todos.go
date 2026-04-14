package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var multiSpaceRe = regexp.MustCompile(`\s{2,}`)

// TodoItem represents a parsed todo from markdown.
type TodoItem struct {
	Line      int
	Text      string
	DueDate   string // "YYYY-MM-DD" or empty
	Completed bool
}

var todoLineRe = regexp.MustCompile(`^(\s*)- \[([ xX])\] (.+)$`)
var dueDateRe = regexp.MustCompile(`@due\((\d{4}-\d{2}-\d{2})\)`)

// ParseTodos extracts todo items from markdown body.
// Returns a slice of TodoItem with 1-based line numbers.
func ParseTodos(body string) []TodoItem {
	var todos []TodoItem
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		m := todoLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		completed := m[2] == "x" || m[2] == "X"
		text := m[3]

		var dueDate string
		if dm := dueDateRe.FindStringSubmatch(text); dm != nil {
			dueDate = dm[1]
			// Remove @due(...) from display text and collapse extra spaces
			text = strings.TrimSpace(dueDateRe.ReplaceAllString(text, ""))
			text = multiSpaceRe.ReplaceAllString(text, " ")
		}

		todos = append(todos, TodoItem{
			Line:      i + 1, // 1-based
			Text:      text,
			DueDate:   dueDate,
			Completed: completed,
		})
	}
	return todos
}

// ToggleTodoInMarkdown reads a markdown body, toggles the checkbox on the given
// 1-based line number, and returns the updated body. Returns an error if the
// line is not a valid todo line.
func ToggleTodoInMarkdown(body string, line int, checked bool) (string, error) {
	lines := strings.Split(body, "\n")
	if line < 1 || line > len(lines) {
		return "", fmt.Errorf("line %d out of range (1-%d)", line, len(lines))
	}
	idx := line - 1
	l := lines[idx]
	m := todoLineRe.FindStringSubmatch(l)
	if m == nil {
		return "", fmt.Errorf("line %d is not a todo line", line)
	}

	if checked {
		lines[idx] = strings.Replace(l, "- [ ]", "- [x]", 1)
	} else {
		// Handle both [x] and [X]
		lines[idx] = strings.Replace(l, "- [x]", "- [ ]", 1)
		lines[idx] = strings.Replace(lines[idx], "- [X]", "- [ ]", 1)
	}

	return strings.Join(lines, "\n"), nil
}

// PendingTodo represents a todo item with its parent note context.
type PendingTodo struct {
	Line             int
	Text             string
	DueDate          string
	DueDateFormatted string
	IsOverdue        bool
	NoteSlug         string
	NoteTitle        string
	NoteTags         []string
}

// ListPendingTodos returns all pending (uncompleted) todos for a user,
// excluding archived notes. Optionally filtered by tag.
func ListPendingTodos(db *DB, userID int64, tag string) ([]PendingTodo, error) {
	query := `
		SELECT t.line, t.text, COALESCE(t.due_date, '') as due_date,
		       n.slug, n.title
		FROM note_todos t
		JOIN notes n ON n.id = t.note_id
		WHERE n.user_id = ? AND n.archived = 0 AND t.completed = 0`
	args := []any{userID}

	if tag != "" {
		query += ` AND EXISTS (SELECT 1 FROM note_tags nt WHERE nt.note_id = n.id AND nt.tag_name = ?)`
		args = append(args, tag)
	}

	query += `
		ORDER BY
			CASE WHEN t.due_date IS NULL OR t.due_date = '' THEN 1 ELSE 0 END,
			t.due_date ASC,
			n.title ASC,
			t.line ASC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now().Truncate(24 * time.Hour)
	var todos []PendingTodo
	for rows.Next() {
		var todo PendingTodo
		if err := rows.Scan(&todo.Line, &todo.Text, &todo.DueDate, &todo.NoteSlug, &todo.NoteTitle); err != nil {
			return nil, err
		}
		if todo.DueDate != "" {
			if t, err := time.Parse("2006-01-02", todo.DueDate); err == nil {
				todo.DueDateFormatted = t.Format("Jan 2, 2006")
				todo.IsOverdue = t.Before(now)
			}
		}
		todos = append(todos, todo)
	}

	// Load tags for each unique note
	slugTags := make(map[string][]string)
	for i := range todos {
		slug := todos[i].NoteSlug
		if _, ok := slugTags[slug]; !ok {
			tagRows, err := db.Query(`
				SELECT nt.tag_name FROM note_tags nt
				JOIN notes n ON n.id = nt.note_id
				WHERE n.slug = ? AND n.user_id = ?
				ORDER BY nt.tag_name`, slug, userID)
			if err != nil {
				continue
			}
			var tags []string
			for tagRows.Next() {
				var t string
				tagRows.Scan(&t)
				tags = append(tags, t)
			}
			tagRows.Close()
			slugTags[slug] = tags
		}
		todos[i].NoteTags = slugTags[slug]
	}

	return todos, nil
}

// CountPendingTodos returns the number of pending todos for a user.
func CountPendingTodos(db *DB, userID int64) int {
	var count int
	db.QueryRow(`
		SELECT COUNT(*) FROM note_todos t
		JOIN notes n ON n.id = t.note_id
		WHERE n.user_id = ? AND n.archived = 0 AND t.completed = 0`, userID).Scan(&count)
	return count
}

// SyncTodos replaces all todos for a note in the database.
// Follows the same delete-then-insert pattern as SyncTags.
func SyncTodos(db *DB, noteID int64, todos []TodoItem) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec("DELETE FROM note_todos WHERE note_id = ?", noteID); err != nil {
		return err
	}

	for _, t := range todos {
		var due *string
		if t.DueDate != "" {
			due = &t.DueDate
		}
		if _, err := tx.Exec(
			"INSERT INTO note_todos (note_id, line, text, due_date, completed) VALUES (?, ?, ?, ?, ?)",
			noteID, t.Line, t.Text, due, t.Completed,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
