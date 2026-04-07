package embedding

import (
	"fmt"
	"math"
	"strings"

	allminilm "github.com/clems4ever/all-minilm-l6-v2-go/all_minilm_l6_v2"
)

const Dimensions = 384
const maxInputChars = 8000

// Embedder generates 384-dimensional sentence embeddings using all-MiniLM-L6-v2.
type Embedder struct {
	model *allminilm.Model
}

// New creates an Embedder. runtimeLibPath is the path to libonnxruntime.so;
// pass empty string to use the default search path (or ONNXRUNTIME_LIB_PATH env var).
func New(runtimeLibPath string) (*Embedder, error) {
	var opts []allminilm.ModelOption
	if runtimeLibPath != "" {
		opts = append(opts, allminilm.WithRuntimePath(runtimeLibPath))
	}
	model, err := allminilm.NewModel(opts...)
	if err != nil {
		return nil, fmt.Errorf("embedding: init model: %w", err)
	}
	return &Embedder{model: model}, nil
}

// Close releases resources held by the model.
func (e *Embedder) Close() {
	if e.model != nil {
		_ = e.model.Close()
	}
}

// Embed generates a normalized 384-dimensional embedding for the given text.
func (e *Embedder) Embed(text string) ([]float32, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return make([]float32, Dimensions), nil
	}
	if len(text) > maxInputChars {
		text = text[:maxInputChars]
	}
	emb, err := e.model.Compute(text, true)
	if err != nil {
		return nil, fmt.Errorf("embedding: compute: %w", err)
	}
	normalize(emb)
	return emb, nil
}

func normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	norm := float32(math.Sqrt(sum))
	if norm == 0 {
		return
	}
	for i := range v {
		v[i] /= norm
	}
}

