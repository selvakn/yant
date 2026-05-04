package markdown_test

import (
	"strings"
	"testing"

	"github.com/selvakn/yant/internal/markdown"
)

func TestGenerateExcerpt_plain_text(t *testing.T) {
	// Arrange
	body := "Hello world"
	maxLen := 100

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "Hello world"
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_strips_headings(t *testing.T) {
	// Arrange
	body := "# Title\n\nBody text"
	maxLen := 100

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "Body text"
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_strips_drawing_markers(t *testing.T) {
	// Arrange
	body := "Intro ![[draw:abc12345]] outro"
	maxLen := 100

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "Intro outro"
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_strips_bold_italic(t *testing.T) {
	// Arrange
	body := "**bold** and *italic*"
	maxLen := 100

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "bold and italic"
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_strips_links(t *testing.T) {
	// Arrange
	body := "[click here](http://example.com)"
	maxLen := 100

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "click here"
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_strips_images(t *testing.T) {
	// Arrange
	body := "Before ![alt](img.png) after"
	maxLen := 100

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "Before after"
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_strips_blockquotes(t *testing.T) {
	// Arrange
	body := "> quoted text"
	maxLen := 100

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "quoted text"
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_strips_code_blocks(t *testing.T) {
	// Arrange
	body := "Intro\n\n```go\nfunc main() {}\n```\n\nOutro"
	maxLen := 100

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "Intro Outro"
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_strips_inline_code(t *testing.T) {
	// Arrange
	body := "Use `code` here"
	maxLen := 100

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "Use code here"
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_truncates_with_ellipsis(t *testing.T) {
	// Arrange
	body := "one two three four five six seven eight nine ten"
	maxLen := 20

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "one two three..."
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
	if len(got) > maxLen {
		t.Errorf("len(%q) = %d, want <= %d", got, len(got), maxLen)
	}
}

func TestGenerateExcerpt_empty_body(t *testing.T) {
	// Arrange
	body := ""
	maxLen := 50

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := ""
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_complex_markdown(t *testing.T) {
	// Arrange
	body := strings.TrimSpace(`
# Post title

![[draw:abc12345]]

First paragraph with **bold** and a [link label](https://example.com/path).

![hero](hero.png)

> Pull quote line

---

More plain text at the end.
`)
	maxLen := 300

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "First paragraph with bold and a link label. Pull quote line More plain text at the end."
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}

func TestGenerateExcerpt_collapses_whitespace(t *testing.T) {
	// Arrange
	body := "Hello\n\n\nworld   wide"
	maxLen := 100

	// Act
	got := markdown.GenerateExcerpt(body, maxLen)

	// Assert
	want := "Hello world wide"
	if got != want {
		t.Errorf("GenerateExcerpt() = %q, want %q", got, want)
	}
}
