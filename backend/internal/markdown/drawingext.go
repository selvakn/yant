package markdown

import (
	"regexp"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// DrawingMarker is an AST node representing a ![[draw:id]] marker.
var KindDrawingMarker = ast.NewNodeKind("DrawingMarker")

type DrawingMarker struct {
	ast.BaseInline
	DrawingID string
}

func (n *DrawingMarker) Kind() ast.NodeKind { return KindDrawingMarker }
func (n *DrawingMarker) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"DrawingID": n.DrawingID}, nil)
}

// drawingMarkerParser parses ![[draw:<id>]] inline markers.
type drawingMarkerParser struct{}

var drawingMarkerRe = regexp.MustCompile(`^!\[\[draw:([a-z0-9]+)\]\]`)

func (p *drawingMarkerParser) Trigger() []byte {
	return []byte{'!'}
}

func (p *drawingMarkerParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, seg := block.PeekLine()
	if len(line) == 0 || line[0] != '!' {
		return nil
	}

	m := drawingMarkerRe.FindSubmatch(line)
	if m == nil {
		return nil
	}

	matchLen := len(m[0])
	block.Advance(matchLen)
	_ = seg // suppress unused variable

	return &DrawingMarker{DrawingID: string(m[1])}
}

// drawingMarkerRenderer renders DrawingMarker nodes to HTML.
type drawingMarkerRenderer struct {
	html.Config
}

func (r *drawingMarkerRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindDrawingMarker, r.renderDrawingMarker)
}

func (r *drawingMarkerRenderer) renderDrawingMarker(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	dm := n.(*DrawingMarker)
	_, _ = w.WriteString(`<div class="drawing-embed" data-drawing-id="`)
	_, _ = w.WriteString(dm.DrawingID)
	_, _ = w.WriteString(`"></div>`)
	return ast.WalkContinue, nil
}

// DrawingMarkerExtension is a goldmark extension that parses ![[draw:id]] markers.
var DrawingMarkerExtension = &drawingMarkerExtension{}

type drawingMarkerExtension struct{}

func (e *drawingMarkerExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(&drawingMarkerParser{}, 100),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&drawingMarkerRenderer{}, 100),
		),
	)
}
