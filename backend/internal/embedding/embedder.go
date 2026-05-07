//go:build ncnn

package embedding

// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/lib -lncnn -lstdc++ -lgomp -lm
// #include "ncnn_bridge.h"
// #include <stdlib.h>
import "C"

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
)

// Embedder generates 384-dimensional sentence embeddings using all-MiniLM-L6-v2
// via the ncnn inference runtime.
type Embedder struct {
	handle *C.NcnnEmbedder
	tk     *tokenizer.Tokenizer
}

// New loads the ncnn model from paramPath and binPath, and the HuggingFace
// tokenizer from tokenizerPath. The caller must call Close() when done.
func New(paramPath, binPath, tokenizerPath string) (*Embedder, error) {
	cParam := C.CString(paramPath)
	cBin := C.CString(binPath)
	defer C.free(unsafe.Pointer(cParam))
	defer C.free(unsafe.Pointer(cBin))

	handle := C.ncnn_embedder_create(cParam, cBin)
	if handle == nil {
		return nil, fmt.Errorf("ncnn: failed to load model from %s / %s", paramPath, binPath)
	}

	f, err := os.Open(tokenizerPath)
	if err != nil {
		C.ncnn_embedder_destroy(handle)
		return nil, fmt.Errorf("open tokenizer: %w", err)
	}
	defer f.Close()

	tk, err := pretrained.FromReader(f)
	if err != nil {
		C.ncnn_embedder_destroy(handle)
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}

	return &Embedder{handle: handle, tk: tk}, nil
}

// Close releases the ncnn model resources.
func (e *Embedder) Close() {
	if e.handle != nil {
		C.ncnn_embedder_destroy(e.handle)
		e.handle = nil
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

	enc, err := e.tk.EncodeSingle(text, true)
	if err != nil {
		return nil, fmt.Errorf("tokenize: %w", err)
	}

	ids := enc.Ids
	if len(ids) > maxTokens {
		ids = ids[:maxTokens]
	}
	seqLen := len(ids)

	// Pad to maxTokens (PNNX export uses fixed shape [1, 128])
	inputIDs := make([]int32, maxTokens)
	attMask := make([]int32, maxTokens)
	tokenTypes := make([]int32, maxTokens)
	for i, id := range ids {
		inputIDs[i] = int32(id)
		attMask[i] = 1
	}

	// Output buffer: [maxTokens * Dimensions] floats for last_hidden_state
	output := make([]float32, maxTokens*Dimensions)

	ret := C.ncnn_embedder_run(
		e.handle,
		(*C.int)(unsafe.Pointer(&inputIDs[0])),
		(*C.int)(unsafe.Pointer(&attMask[0])),
		(*C.int)(unsafe.Pointer(&tokenTypes[0])),
		C.int(maxTokens),
		(*C.float)(unsafe.Pointer(&output[0])),
		C.int(Dimensions),
	)
	if ret != 0 {
		return nil, fmt.Errorf("ncnn inference failed (code %d)", int(ret))
	}

	// Mean pooling over actual sequence positions (attention_mask weighted)
	pooled := make([]float32, Dimensions)
	count := 0
	for i := 0; i < seqLen; i++ {
		if attMask[i] == 1 {
			for j := 0; j < Dimensions; j++ {
				pooled[j] += output[i*Dimensions+j]
			}
			count++
		}
	}
	if count > 0 {
		for j := range pooled {
			pooled[j] /= float32(count)
		}
	}

	normalize(pooled)
	return pooled, nil
}

