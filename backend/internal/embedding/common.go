package embedding

import (
	"errors"
	"math"
)

// Dimensions is the output embedding size for all-MiniLM-L6-v2.
const Dimensions = 384

const maxInputChars = 8000

// maxTokens matches the fixed input shape used in the PNNX export ([1, 128]).
const maxTokens = 128

// ErrNotAvailable is returned when the ncnn runtime is not compiled in.
var ErrNotAvailable = errors.New("embedding: ncnn runtime not compiled (build with -tags ncnn)")

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
