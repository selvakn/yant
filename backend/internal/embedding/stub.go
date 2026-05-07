//go:build !ncnn

package embedding

// Embedder is a placeholder when the ncnn build tag is absent.
// All methods return ErrNotAvailable so the caller can degrade gracefully.
type Embedder struct{}

// New returns ErrNotAvailable. Semantic search is disabled in this build.
func New(paramPath, binPath, tokenizerPath string) (*Embedder, error) {
	return nil, ErrNotAvailable
}

// Embed returns ErrNotAvailable.
func (e *Embedder) Embed(text string) ([]float32, error) {
	return nil, ErrNotAvailable
}

// Close is a no-op.
func (e *Embedder) Close() {}
