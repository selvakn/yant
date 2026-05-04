package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func renderWithExtension(src string) string {
	md := goldmark.New(goldmark.WithExtensions(DrawingMarkerExtension))
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return "ERROR: " + err.Error()
	}
	return buf.String()
}

func TestDrawingMarker_single(t *testing.T) {
	out := renderWithExtension("![[draw:abc12345]]")
	expected := `<div class="drawing-embed" data-drawing-id="abc12345"></div>`
	if !strings.Contains(out, expected) {
		t.Errorf("expected %q in output, got: %s", expected, out)
	}
}

func TestDrawingMarker_in_paragraph(t *testing.T) {
	out := renderWithExtension("Some text ![[draw:xyz99]] more text")
	if !strings.Contains(out, `data-drawing-id="xyz99"`) {
		t.Errorf("marker not found in output: %s", out)
	}
	if !strings.Contains(out, "Some text") {
		t.Errorf("surrounding text missing: %s", out)
	}
}

func TestDrawingMarker_multiple(t *testing.T) {
	out := renderWithExtension("![[draw:aaa]] and ![[draw:bbb]]")
	if !strings.Contains(out, `data-drawing-id="aaa"`) {
		t.Errorf("first marker not found: %s", out)
	}
	if !strings.Contains(out, `data-drawing-id="bbb"`) {
		t.Errorf("second marker not found: %s", out)
	}
}

func TestDrawingMarker_malformed_ignored(t *testing.T) {
	cases := []string{
		"![[draw:]]",       // empty id
		"![[draw:ABC]]",    // uppercase
		"![[draw:abc def]]", // space
		"![[notdraw:abc]]",  // wrong prefix
		"[[draw:abc]]",      // missing !
	}
	for _, input := range cases {
		out := renderWithExtension(input)
		if strings.Contains(out, "drawing-embed") {
			t.Errorf("malformed input %q should not produce drawing-embed, got: %s", input, out)
		}
	}
}

func TestDrawingMarker_on_own_line(t *testing.T) {
	out := renderWithExtension("# Heading\n\n![[draw:solo1]]\n\nMore text")
	if !strings.Contains(out, `data-drawing-id="solo1"`) {
		t.Errorf("marker on own line not parsed: %s", out)
	}
}
