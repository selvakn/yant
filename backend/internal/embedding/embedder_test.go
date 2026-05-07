//go:build !ncnn

package embedding

import (
	"errors"
	"math"
	"testing"
)

func TestStub_NewReturnsError(t *testing.T) {
	emb, err := New("", "", "")
	if emb != nil {
		t.Error("expected nil embedder from stub")
	}
	if !errors.Is(err, ErrNotAvailable) {
		t.Errorf("expected ErrNotAvailable, got %v", err)
	}
}

func TestStub_EmbedReturnsError(t *testing.T) {
	e := &Embedder{}
	_, err := e.Embed("hello")
	if !errors.Is(err, ErrNotAvailable) {
		t.Errorf("expected ErrNotAvailable, got %v", err)
	}
}

func TestStub_CloseIsNoop(t *testing.T) {
	e := &Embedder{}
	e.Close() // must not panic
}

func TestNormalize_UnitVector(t *testing.T) {
	v := []float32{3, 4}
	normalize(v)
	got := math.Sqrt(float64(v[0])*float64(v[0]) + float64(v[1])*float64(v[1]))
	if math.Abs(got-1.0) > 1e-6 {
		t.Errorf("normalize: want norm=1, got %f", got)
	}
}

func TestNormalize_ZeroVector(t *testing.T) {
	v := []float32{0, 0, 0}
	normalize(v) // must not divide by zero
	for _, x := range v {
		if x != 0 {
			t.Errorf("zero vector should remain zero after normalize")
		}
	}
}
