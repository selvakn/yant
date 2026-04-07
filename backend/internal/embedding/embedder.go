package embedding

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

const Dimensions = 384
const maxInputChars = 8000
const maxTokens = 512

// Embedder generates 384-dimensional sentence embeddings using all-MiniLM-L6-v2.
type Embedder struct {
	tk      *tokenizer.Tokenizer
	session *ort.DynamicAdvancedSession
}

// New creates an Embedder. runtimeLibPath is the path to libonnxruntime.so
// (empty = use ONNXRUNTIME_LIB_PATH env or default search).
// modelPath is the path to the ONNX model file.
// tokenizerPath is the path to tokenizer.json.
func New(runtimeLibPath, modelPath, tokenizerPath string) (*Embedder, error) {
	if runtimeLibPath != "" {
		ort.SetSharedLibraryPath(runtimeLibPath)
	} else if p, ok := os.LookupEnv("ONNXRUNTIME_LIB_PATH"); ok {
		ort.SetSharedLibraryPath(p)
	}

	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("init onnx runtime: %w", err)
	}

	modelData, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("read model: %w", err)
	}

	inputNames := []string{"input_ids", "attention_mask", "token_type_ids"}
	outputNames := []string{"last_hidden_state"}

	session, err := ort.NewDynamicAdvancedSessionWithONNXData(modelData, inputNames, outputNames, nil)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	f, err := os.Open(tokenizerPath)
	if err != nil {
		session.Destroy()
		return nil, fmt.Errorf("open tokenizer: %w", err)
	}
	defer f.Close()

	tk, err := pretrained.FromReader(f)
	if err != nil {
		session.Destroy()
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}

	return &Embedder{tk: tk, session: session}, nil
}

// Close releases resources held by the model.
func (e *Embedder) Close() {
	if e.session != nil {
		e.session.Destroy()
	}
	_ = ort.DestroyEnvironment()
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

	inputIDs := make([]int64, seqLen)
	attMask := make([]int64, seqLen)
	tokenTypes := make([]int64, seqLen)
	for i, id := range ids {
		inputIDs[i] = int64(id)
		attMask[i] = 1
		tokenTypes[i] = 0
	}

	shape := ort.NewShape(1, int64(seqLen))

	inputIDsTensor, err := ort.NewTensor(shape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	attMaskTensor, err := ort.NewTensor(shape, attMask)
	if err != nil {
		return nil, fmt.Errorf("attention_mask tensor: %w", err)
	}
	defer attMaskTensor.Destroy()

	tokenTypesTensor, err := ort.NewTensor(shape, tokenTypes)
	if err != nil {
		return nil, fmt.Errorf("token_type_ids tensor: %w", err)
	}
	defer tokenTypesTensor.Destroy()

	outShape := ort.NewShape(1, int64(seqLen), int64(Dimensions))
	outputTensor, err := ort.NewEmptyTensor[float32](outShape)
	if err != nil {
		return nil, fmt.Errorf("output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	err = e.session.Run(
		[]ort.ArbitraryTensor{inputIDsTensor, attMaskTensor, tokenTypesTensor},
		[]ort.ArbitraryTensor{outputTensor},
	)
	if err != nil {
		return nil, fmt.Errorf("run inference: %w", err)
	}

	// Mean pooling over the sequence dimension, using attention mask
	rawOutput := outputTensor.GetData()
	pooled := make([]float32, Dimensions)
	count := float32(0)
	for i := 0; i < seqLen; i++ {
		if attMask[i] == 1 {
			for j := 0; j < Dimensions; j++ {
				pooled[j] += rawOutput[i*Dimensions+j]
			}
			count++
		}
	}
	if count > 0 {
		for j := range pooled {
			pooled[j] /= count
		}
	}

	normalize(pooled)
	return pooled, nil
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
